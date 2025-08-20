package participantmodel

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ParticipantCreateResult represents the result of creating participants in both databases
type ParticipantCreateResult struct {
	MongoResult       *mongo.InsertManyResult
	PostgresRecords   []*model.Participant
	CreatedIDs        []string
	FailedPostgresIDs []string
}

func Revoke(id string) (*model.Participant, error) {
	// Get the participant by ID
	participant, err := common.Gorm.Participant.Where(common.Gorm.Participant.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}

	// Update the isrevoke field to true
	_, err = common.Gorm.Participant.Where(common.Gorm.Participant.ID.Eq(id)).Update(common.Gorm.Participant.Isrevoke, true)
	if err != nil {
		return nil, err
	}

	// Return the updated participant
	participant.Isrevoke = true
	return participant, nil
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

// AddParticipants adds participants to both MongoDB (data) and PostgreSQL (index/status) with same IDs
func AddParticipants(certId string, participants []map[string]any) (*ParticipantCreateResult, error) {
	// Validate field consistency before adding
	if err := ValidateFieldConsistency(certId, participants); err != nil {
		slog.Warn("ParticipantModel AddParticipants field validation failed", "error", err, "cert_id", certId)
		return nil, fmt.Errorf("field validation failed: %w", err)
	}

	// Generate UUIDs for consistent IDs across both databases
	participantIDs := make([]string, len(participants))
	for i := range participants {
		participantIDs[i] = uuid.New().String()
	}

	result := &ParticipantCreateResult{
		CreatedIDs:        []string{},
		FailedPostgresIDs: []string{},
		PostgresRecords:   []*model.Participant{},
	}

	// Step 1: Create records in MongoDB first
	mongoResult, mongoErr := addParticipantsToMongo(certId, participants, participantIDs)
	if mongoErr != nil {
		slog.Error("ParticipantModel AddParticipants MongoDB failed", "error", mongoErr, "cert_id", certId)
		return nil, fmt.Errorf("MongoDB insertion failed: %w", mongoErr)
	}
	result.MongoResult = mongoResult

	// Step 2: Create corresponding records in PostgreSQL
	postgresRecords, failedIDs := addParticipantsToPostgres(certId, participantIDs)
	result.PostgresRecords = postgresRecords
	result.FailedPostgresIDs = failedIDs

	// Determine successfully created IDs (those that succeeded in both databases)
	for _, id := range participantIDs {
		failed := false
		for _, failedID := range failedIDs {
			if id == failedID {
				failed = true
				break
			}
		}
		if !failed {
			result.CreatedIDs = append(result.CreatedIDs, id)
		}
	}

	slog.Info("ParticipantModel AddParticipants completed",
		"cert_id", certId,
		"total_requested", len(participants),
		"mongo_created", len(mongoResult.InsertedIDs),
		"postgres_created", len(postgresRecords),
		"postgres_failed", len(failedIDs),
		"fully_created", len(result.CreatedIDs))

	// If some PostgreSQL records failed, log warning but don't fail the entire operation
	if len(failedIDs) > 0 {
		slog.Warn("ParticipantModel AddParticipants partial PostgreSQL failure",
			"cert_id", certId,
			"failed_ids", failedIDs,
			"failed_count", len(failedIDs))
	}

	return result, nil
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

// addParticipantsToPostgres creates index/status records in PostgreSQL
func addParticipantsToPostgres(certId string, participantIDs []string) ([]*model.Participant, []string) {
	var successfulRecords []*model.Participant
	var failedIDs []string

	for _, id := range participantIDs {
		participant := &model.Participant{
			ID:            id,
			CertificateID: certId,
			Isrevoke:      false, // Default to not revoked
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		// Create record in PostgreSQL
		createErr := common.Gorm.Participant.Create(participant)
		if createErr != nil {
			slog.Error("ParticipantModel PostgreSQL creation failed",
				"error", createErr,
				"participant_id", id,
				"cert_id", certId)
			failedIDs = append(failedIDs, id)
			continue
		}

		successfulRecords = append(successfulRecords, participant)
		slog.Debug("ParticipantModel PostgreSQL record created",
			"participant_id", id,
			"cert_id", certId)
	}

	slog.Info("ParticipantModel PostgreSQL creation summary",
		"cert_id", certId,
		"successful", len(successfulRecords),
		"failed", len(failedIDs))

	return successfulRecords, failedIDs
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
