package participantmodel

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
	"go.mongodb.org/mongo-driver/mongo"
)

// ParticipantCreateResult represents the result of creating participants in both databases
type ParticipantCreateResult struct {
	MongoResult       *mongo.InsertManyResult
	PostgresRecords   []*model.Participant
	CreatedIDs        []string
	FailedPostgresIDs []string
}

// CombinedParticipant represents participant data from both databases
type CombinedParticipant struct {
	ID            string                 `json:"id"`
	CertificateID string                 `json:"certificate_id"`
	Isrevoke      bool                   `json:"is_revoked"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	DynamicData   map[string]any         `json:"data"`
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

// GetParticipantsByCertId returns combined participant data from both PostgreSQL and MongoDB
func GetParticipantsByCertId(certId string) ([]*CombinedParticipant, error) {
	// Get PostgreSQL data
	postgresParticipants, pgErr := GetParticipantsByPostgres(certId)
	if pgErr != nil {
		return nil, fmt.Errorf("failed to get PostgreSQL participants: %w", pgErr)
	}

	// Get MongoDB data
	mongoParticipants, mongoErr := GetParticipantsByMongo(certId)
	if mongoErr != nil {
		return nil, fmt.Errorf("failed to get MongoDB participants: %w", mongoErr)
	}

	// Create a map of MongoDB data by ID for fast lookup
	mongoDataMap := make(map[string]map[string]any)
	for _, participant := range mongoParticipants {
		if id, ok := participant["_id"].(string); ok {
			mongoDataMap[id] = participant
		}
	}

	// Combine data
	var combinedParticipants []*CombinedParticipant
	for _, pgParticipant := range postgresParticipants {
		combined := &CombinedParticipant{
			ID:            pgParticipant.ID,
			CertificateID: pgParticipant.CertificateID,
			Isrevoke:      pgParticipant.Isrevoke,
			CreatedAt:     pgParticipant.CreatedAt,
			UpdatedAt:     pgParticipant.UpdatedAt,
			DynamicData:   make(map[string]any),
		}

		// Add MongoDB data if exists
		if mongoData, exists := mongoDataMap[pgParticipant.ID]; exists {
			// Copy all fields except internal ones
			for key, value := range mongoData {
				if key != "_id" && key != "certificate_id" {
					combined.DynamicData[key] = value
				}
			}
		}

		combinedParticipants = append(combinedParticipants, combined)
	}

	slog.Info("ParticipantModel GetParticipantsByCertId", 
		"cert_id", certId, 
		"postgres_count", len(postgresParticipants),
		"mongo_count", len(mongoParticipants),
		"combined_count", len(combinedParticipants))

	return combinedParticipants, nil
}

// DeleteByCertId deletes participants from both PostgreSQL and MongoDB for a certificate
func DeleteByCertId(certId string) ([]*model.Participant, error) {
	// Delete from PostgreSQL first
	participants, err := DeleteByCertIdFromPostgres(certId)
	if err != nil {
		slog.Error("ParticipantModel DeleteByCertId PostgreSQL deletion failed", "error", err, "cert_id", certId)
		return nil, err
	}

	// Delete MongoDB collection
	err = DeleteCollectionByCertIdFromMongo(certId)
	if err != nil {
		slog.Error("ParticipantModel DeleteByCertId MongoDB deletion failed", "error", err, "cert_id", certId)
		return participants, err
	}

	slog.Info("ParticipantModel DeleteByCertId completed", "cert_id", certId, "postgres_count", len(participants))
	return participants, nil
}

// EditParticipantByID updates a participant's data with structure validation
func EditParticipantByID(participantID string, newData map[string]any) (*CombinedParticipant, error) {
	// First, get the participant from PostgreSQL to get certificate ID
	participant, err := GetParticipantByIdFromPostgres(participantID)
	if err != nil {
		slog.Error("ParticipantModel EditParticipantByID: Failed to get participant from PostgreSQL", "error", err, "participant_id", participantID)
		return nil, fmt.Errorf("participant not found: %w", err)
	}

	certId := participant.CertificateID

	// Validate that new data structure matches existing structure
	if err := validateEditDataStructure(certId, newData); err != nil {
		slog.Warn("ParticipantModel EditParticipantByID: Data structure validation failed", "error", err, "participant_id", participantID, "cert_id", certId)
		return nil, fmt.Errorf("data structure validation failed: %w", err)
	}

	// Update in MongoDB
	err = updateParticipantInMongo(certId, participantID, newData)
	if err != nil {
		slog.Error("ParticipantModel EditParticipantByID: Failed to update MongoDB", "error", err, "participant_id", participantID, "cert_id", certId)
		return nil, fmt.Errorf("failed to update participant data: %w", err)
	}

	// Update timestamp in PostgreSQL
	err = updateParticipantTimestampInPostgres(participantID)
	if err != nil {
		slog.Warn("ParticipantModel EditParticipantByID: Failed to update PostgreSQL timestamp", "error", err, "participant_id", participantID)
		// Don't fail the operation for timestamp update failure
	}

	// Return the updated combined participant data
	combinedData := &CombinedParticipant{
		ID:            participant.ID,
		CertificateID: participant.CertificateID,
		Isrevoke:      participant.Isrevoke,
		CreatedAt:     participant.CreatedAt,
		UpdatedAt:     time.Now(), // Use current time for updated_at
		DynamicData:   newData,
	}

	slog.Info("ParticipantModel EditParticipantByID completed successfully", "participant_id", participantID, "cert_id", certId)
	return combinedData, nil
}

// DeleteParticipantByID deletes a single participant from both PostgreSQL and MongoDB by participant ID
func DeleteParticipantByID(participantID string) (*model.Participant, error) {
	// First, get the participant from PostgreSQL to get certificate ID and return data
	participant, err := GetParticipantByIdFromPostgres(participantID)
	if err != nil {
		slog.Error("ParticipantModel DeleteParticipantByID: Failed to get participant from PostgreSQL", "error", err, "participant_id", participantID)
		return nil, fmt.Errorf("participant not found: %w", err)
	}

	certId := participant.CertificateID

	// Delete from PostgreSQL
	err = deleteParticipantByIdFromPostgres(participantID)
	if err != nil {
		slog.Error("ParticipantModel DeleteParticipantByID: Failed to delete from PostgreSQL", "error", err, "participant_id", participantID)
		return nil, fmt.Errorf("failed to delete participant from PostgreSQL: %w", err)
	}

	// Delete from MongoDB
	err = deleteParticipantByIdFromMongo(certId, participantID)
	if err != nil {
		slog.Error("ParticipantModel DeleteParticipantByID: Failed to delete from MongoDB", "error", err, "participant_id", participantID, "cert_id", certId)
		// Note: PostgreSQL record is already deleted, so we log the error but don't fail the operation completely
		slog.Warn("ParticipantModel DeleteParticipantByID: MongoDB deletion failed but PostgreSQL succeeded", "participant_id", participantID, "cert_id", certId)
	}

	slog.Info("ParticipantModel DeleteParticipantByID completed", "participant_id", participantID, "cert_id", certId)
	return participant, nil
}

