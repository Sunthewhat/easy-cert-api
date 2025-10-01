package participantmodel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	"github.com/sunthewhat/easy-cert-api/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// extractAnchorNames extracts anchor names from certificate design JSON
func extractAnchorNames(designJSON string) ([]string, error) {
	var design map[string]any
	if err := json.Unmarshal([]byte(designJSON), &design); err != nil {
		return nil, fmt.Errorf("failed to parse certificate design: %w", err)
	}

	objects, ok := design["objects"].([]any)
	if !ok {
		return nil, fmt.Errorf("invalid design format - objects array not found")
	}

	var anchorNames []string
	for _, obj := range objects {
		objMap, ok := obj.(map[string]any)
		if !ok {
			continue
		}

		id, exists := objMap["id"].(string)
		if exists && strings.HasPrefix(id, "PLACEHOLDER-") {
			anchorName := strings.TrimPrefix(id, "PLACEHOLDER-")
			anchorNames = append(anchorNames, anchorName)
		}
	}

	sort.Strings(anchorNames) // Sort for consistent ordering
	return anchorNames, nil
}

// GetParticipantCollectionCount returns the count of participants in the MongoDB collection
func GetParticipantCollectionCount(certId string) (int64, error) {
	collectionName := "participant-" + certId
	collection := common.Mongo.Collection(collectionName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		slog.Error("ParticipantModel GetCollectionCount failed", "error", err, "cert_id", certId)
		return 0, err
	}

	return count, nil
}

// GetParticipantsByMongo returns participants from MongoDB by certificate ID
func GetParticipantsByMongo(certId string) ([]map[string]any, error) {
	collectionName := "participant-" + certId
	collection := common.Mongo.Collection(collectionName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{"certificate_id": certId})
	if err != nil {
		slog.Error("ParticipantModel GetParticipantsByMongo find failed", "error", err, "cert_id", certId)
		return nil, err
	}
	defer cursor.Close(ctx)

	var participants []map[string]any
	if err = cursor.All(ctx, &participants); err != nil {
		slog.Error("ParticipantModel GetParticipantsByMongo cursor failed", "error", err, "cert_id", certId)
		return nil, err
	}

	slog.Info("ParticipantModel GetParticipantsByMongo", "cert_id", certId, "count", len(participants))
	return participants, nil
}

// GetParticipantByIdFromMongo returns a specific participant from MongoDB by participant ID
func GetParticipantByIdFromMongo(certId string, participantID string) (map[string]any, error) {
	collectionName := "participant-" + certId
	collection := common.Mongo.Collection(collectionName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var participant map[string]any
	err := collection.FindOne(ctx, bson.M{"_id": participantID}).Decode(&participant)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			slog.Warn("ParticipantModel GetParticipantByIdFromMongo: participant not found", "cert_id", certId, "participant_id", participantID)
			return nil, fmt.Errorf("participant not found")
		}
		slog.Error("ParticipantModel GetParticipantByIdFromMongo failed", "error", err, "cert_id", certId, "participant_id", participantID)
		return nil, err
	}

	slog.Info("ParticipantModel GetParticipantByIdFromMongo", "cert_id", certId, "participant_id", participantID)
	return participant, nil
}

// addParticipantsToMongo handles MongoDB insertion with generated IDs
func addParticipantsToMongo(certId string, participants []map[string]any, participantIDs []string) (*mongo.InsertManyResult, error) {
	collectionName := "participant-" + certId
	collection := common.Mongo.Collection(collectionName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Prepare documents with metadata and custom IDs
	var documents []any
	for i, participant := range participants {
		// Create a copy to avoid modifying original
		doc := make(map[string]any)
		for k, v := range participant {
			doc[k] = v
		}

		// Add metadata with custom ID
		doc["_id"] = participantIDs[i] // Use our generated UUID as MongoDB _id
		doc["certificate_id"] = certId
		documents = append(documents, doc)
	}

	result, err := collection.InsertMany(ctx, documents)
	if err != nil {
		slog.Error("ParticipantModel MongoDB insertion failed", "error", err, "cert_id", certId)
		return nil, err
	}

	slog.Info("ParticipantModel MongoDB insertion successful",
		"cert_id", certId,
		"collection", collectionName,
		"inserted_count", len(result.InsertedIDs))

	return result, nil
}

// GetExistingParticipantFields gets the field names from the first document in the collection
func GetExistingParticipantFields(certId string) ([]string, error) {
	collectionName := "participant-" + certId
	collection := common.Mongo.Collection(collectionName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Find the first document to get field structure
	var existingDoc bson.M
	err := collection.FindOne(ctx, bson.M{}).Decode(&existingDoc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return []string{}, nil // Empty collection, no existing fields
		}
		slog.Error("ParticipantModel GetExistingFields failed", "error", err, "cert_id", certId)
		return nil, err
	}

	// Extract field names (excluding MongoDB internal fields)
	var fields []string
	for key := range existingDoc {
		if key != "_id" { // Exclude MongoDB internal ID
			fields = append(fields, key)
		}
	}

	sort.Strings(fields) // Sort for consistent comparison
	slog.Info("ParticipantModel GetExistingFields", "cert_id", certId, "fields", fields)
	return fields, nil
}

// ValidateFieldConsistency checks if new participant fields match certificate design anchors
func ValidateFieldConsistency(certId string, newParticipants []map[string]any) error {
	// Get certificate design to extract required anchor fields
	cert, err := certificatemodel.GetById(certId)
	if err != nil {
		return fmt.Errorf("failed to get certificate: %w", err)
	}
	if cert == nil {
		return fmt.Errorf("certificate not found")
	}

	// Extract anchor names from certificate design
	requiredFields, err := extractAnchorNames(cert.Design)
	if err != nil {
		return fmt.Errorf("failed to extract anchor names from certificate design: %w", err)
	}

	// If no anchors in design, allow any structure (for backward compatibility)
	if len(requiredFields) == 0 {
		slog.Info("ParticipantModel ValidateFieldConsistency: no anchors in certificate design, allowing any fields", "cert_id", certId)
		return nil
	}

	// Check each new participant against required anchor fields
	for i, participant := range newParticipants {
		// Check if all required anchor fields are present
		var missingFields []string
		for _, requiredField := range requiredFields {
			value, exists := participant[requiredField]
			if !exists {
				missingFields = append(missingFields, requiredField)
			} else if value == nil {
				missingFields = append(missingFields, requiredField+" (empty)")
			} else if strValue, isString := value.(string); isString && strings.TrimSpace(strValue) == "" {
				missingFields = append(missingFields, requiredField+" (empty)")
			}
		}

		if len(missingFields) > 0 {
			var participantFields []string
			for key := range participant {
				participantFields = append(participantFields, key)
			}
			sort.Strings(participantFields)

			errorMsg := fmt.Sprintf("participant %d is missing required anchor fields: %s. Required: %s, Provided: %s",
				i+1,
				strings.Join(missingFields, ", "),
				strings.Join(requiredFields, ", "),
				strings.Join(participantFields, ", "))

			slog.Warn("ParticipantModel anchor field validation failed",
				"cert_id", certId,
				"participant_index", i,
				"required_anchor_fields", requiredFields,
				"provided_fields", participantFields,
				"missing_fields", missingFields)

			return errors.New(errorMsg)
		}
	}

	slog.Info("ParticipantModel ValidateFieldConsistency passed",
		"cert_id", certId,
		"participant_count", len(newParticipants),
		"required_anchor_fields", requiredFields)
	return nil
}

// areFieldsConsistent checks if two field slices contain the same elements (ignoring auto-added fields)
func areFieldsConsistent(existing, new []string) bool {
	// Filter out auto-added fields from comparison
	autoFields := []string{"certificate_id", "created_at", "updated_at"}

	existingFiltered := filterOutFields(existing, autoFields)
	newFiltered := filterOutFields(new, autoFields)

	if len(existingFiltered) != len(newFiltered) {
		return false
	}

	for i, field := range existingFiltered {
		if field != newFiltered[i] {
			return false
		}
	}
	return true
}

// filterOutFields removes specified fields from a slice
func filterOutFields(fields, toRemove []string) []string {
	var result []string
	removeMap := make(map[string]bool)
	for _, field := range toRemove {
		removeMap[field] = true
	}

	for _, field := range fields {
		if !removeMap[field] {
			result = append(result, field)
		}
	}
	return result
}

// findMissingFields returns fields that are in 'required' but not in 'actual'
func findMissingFields(required, actual []string) []string {
	actualMap := make(map[string]bool)
	for _, field := range actual {
		actualMap[field] = true
	}

	var missing []string
	for _, field := range required {
		if !actualMap[field] {
			missing = append(missing, field)
		}
	}
	return missing
}

// DeleteCollectionByCertId deletes the entire MongoDB collection for a certificate
func DeleteCollectionByCertIdFromMongo(certId string) error {
	collectionName := fmt.Sprintf("participant-%s", certId)

	err := common.Mongo.Collection(collectionName).Drop(context.Background())
	if err != nil {
		slog.Error("ParticipantModel DeleteCollectionByCertId failed", "error", err, "cert_id", certId, "collection", collectionName)
		return err
	}

	slog.Info("ParticipantModel DeleteCollectionByCertId successful", "cert_id", certId, "collection", collectionName)
	return nil
}

// validateEditDataStructure validates that new data has the same structure as existing data
func validateEditDataStructure(certId string, newData map[string]any) error {
	existingFields, err := GetExistingParticipantFields(certId)
	if err != nil {
		return fmt.Errorf("failed to get existing fields: %w", err)
	}

	// If no existing fields (empty collection), cannot edit non-existent data
	if len(existingFields) == 0 {
		return errors.New("cannot edit participant data: no existing participants found for this certificate")
	}

	// Get fields from new data (excluding auto-added fields)
	var newFields []string
	for key := range newData {
		newFields = append(newFields, key)
	}
	sort.Strings(newFields)

	// Compare field sets using existing validation logic
	if !areFieldsConsistent(existingFields, newFields) {
		missingFields := findMissingFields(existingFields, newFields)
		extraFields := findMissingFields(newFields, existingFields)

		var errorMsg strings.Builder
		errorMsg.WriteString("data structure mismatch")

		if len(missingFields) > 0 {
			errorMsg.WriteString(fmt.Sprintf(", missing required fields: %s", strings.Join(missingFields, ", ")))
		}
		if len(extraFields) > 0 {
			errorMsg.WriteString(fmt.Sprintf(", unexpected fields: %s", strings.Join(extraFields, ", ")))
		}

		errorMsg.WriteString(fmt.Sprintf(". Expected fields: %s", strings.Join(existingFields, ", ")))

		slog.Warn("ParticipantModel edit validation failed",
			"cert_id", certId,
			"expected_fields", existingFields,
			"actual_fields", newFields,
			"missing_fields", missingFields,
			"extra_fields", extraFields)

		return errors.New(errorMsg.String())
	}

	slog.Info("ParticipantModel edit validation passed", "cert_id", certId, "fields", newFields)
	return nil
}

// updateParticipantInMongo updates a participant's data in MongoDB
func updateParticipantInMongo(certId, participantID string, newData map[string]any) error {
	collectionName := "participant-" + certId
	collection := common.Mongo.Collection(collectionName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create update document - only update the provided fields
	updateDoc := bson.M{"$set": newData}

	// Update the document
	result, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": participantID},
		updateDoc,
	)

	if err != nil {
		slog.Error("ParticipantModel updateParticipantInMongo failed", "error", err, "cert_id", certId, "participant_id", participantID)
		return err
	}

	if result.MatchedCount == 0 {
		slog.Warn("ParticipantModel updateParticipantInMongo: no document matched", "cert_id", certId, "participant_id", participantID)
		return errors.New("participant not found in MongoDB")
	}

	slog.Info("ParticipantModel updateParticipantInMongo successful",
		"cert_id", certId,
		"participant_id", participantID,
		"matched_count", result.MatchedCount,
		"modified_count", result.ModifiedCount)

	return nil
}

// deleteParticipantByIdFromMongo deletes a single participant from MongoDB by participant ID
func deleteParticipantByIdFromMongo(certId, participantID string) error {
	collectionName := "participant-" + certId
	collection := common.Mongo.Collection(collectionName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Delete the document with the specified ID
	result, err := collection.DeleteOne(ctx, bson.M{"_id": participantID})
	if err != nil {
		slog.Error("ParticipantModel deleteParticipantByIdFromMongo failed", "error", err, "cert_id", certId, "participant_id", participantID)
		return err
	}

	if result.DeletedCount == 0 {
		slog.Warn("ParticipantModel deleteParticipantByIdFromMongo: no document deleted", "cert_id", certId, "participant_id", participantID)
		return fmt.Errorf("participant not found in MongoDB")
	}

	slog.Info("ParticipantModel deleteParticipantByIdFromMongo successful",
		"cert_id", certId,
		"participant_id", participantID,
		"deleted_count", result.DeletedCount)

	return nil
}
