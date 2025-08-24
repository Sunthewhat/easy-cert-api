package participantmodel

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/sunthewhat/easy-cert-api/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

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

// ValidateFieldConsistency checks if new participant fields match existing ones
func ValidateFieldConsistency(certId string, newParticipants []map[string]any) error {
	existingFields, err := GetExistingParticipantFields(certId)
	if err != nil {
		return fmt.Errorf("failed to get existing fields: %w", err)
	}

	// If no existing fields (empty collection), allow any structure
	if len(existingFields) == 0 {
		slog.Info("ParticipantModel ValidateFieldConsistency: empty collection, allowing any fields", "cert_id", certId)
		return nil
	}

	// Check each new participant
	for i, participant := range newParticipants {
		var newFields []string
		for key := range participant {
			newFields = append(newFields, key)
		}
		sort.Strings(newFields)

		// Compare field sets
		if !areFieldsConsistent(existingFields, newFields) {
			missingFields := findMissingFields(existingFields, newFields)
			extraFields := findMissingFields(newFields, existingFields)

			var errorMsg strings.Builder
			errorMsg.WriteString(fmt.Sprintf("participant %d has inconsistent fields", i+1))

			if len(missingFields) > 0 {
				errorMsg.WriteString(fmt.Sprintf(", missing required fields: %s", strings.Join(missingFields, ", ")))
			}
			if len(extraFields) > 0 {
				errorMsg.WriteString(fmt.Sprintf(", unexpected fields: %s", strings.Join(extraFields, ", ")))
			}

			errorMsg.WriteString(fmt.Sprintf(". Expected fields: %s", strings.Join(existingFields, ", ")))

			slog.Warn("ParticipantModel field validation failed",
				"cert_id", certId,
				"participant_index", i,
				"expected_fields", existingFields,
				"actual_fields", newFields,
				"missing_fields", missingFields,
				"extra_fields", extraFields)

			return errors.New(errorMsg.String())
		}
	}

	slog.Info("ParticipantModel ValidateFieldConsistency passed",
		"cert_id", certId,
		"participant_count", len(newParticipants),
		"expected_fields", existingFields)
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
