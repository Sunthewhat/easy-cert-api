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

	"github.com/google/uuid"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
	"github.com/sunthewhat/easy-cert-api/type/shared/query"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ParticipantRepository handles all participant database operations
// It manages both PostgreSQL (for indexes/status) and MongoDB (for dynamic data)
type ParticipantRepository struct {
	q  *query.Query    // PostgreSQL query builder
	db *mongo.Database // MongoDB database
}

// ParticipantCreateResult represents the result of creating participants in both databases
type ParticipantCreateResult struct {
	MongoResult       *mongo.InsertManyResult
	PostgresRecords   []*model.Participant
	CreatedIDs        []string
	FailedPostgresIDs []string
}

// CombinedParticipant represents participant data from both databases
type CombinedParticipant struct {
	ID             string         `json:"id"`
	CertificateID  string         `json:"certificate_id"`
	IsRevoke       bool           `json:"is_revoked"`
	CertificateURL string         `json:"certificate_url"`
	EmailStatus    string         `json:"email_status"`
	IsDownloaded   bool           `json:"is_downloaded"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DynamicData    map[string]any `json:"data"`
}

// NewParticipantRepository creates a new participant repository with dependency injection
func NewParticipantRepository(q *query.Query, db *mongo.Database) *ParticipantRepository {
	return &ParticipantRepository{
		q:  q,
		db: db,
	}
}

// AddParticipants adds participants to both MongoDB (data) and PostgreSQL (index/status) with same IDs
func (r *ParticipantRepository) AddParticipants(certId string, participants []map[string]any) (*ParticipantCreateResult, error) {
	// Validate field consistency before adding
	if err := r.ValidateFieldConsistency(certId, participants); err != nil {
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
	mongoResult, mongoErr := r.addParticipantsToMongo(certId, participants, participantIDs)
	if mongoErr != nil {
		slog.Error("ParticipantModel AddParticipants MongoDB failed", "error", mongoErr, "cert_id", certId)
		return nil, fmt.Errorf("MongoDB insertion failed: %w", mongoErr)
	}
	result.MongoResult = mongoResult

	// Step 2: Create corresponding records in PostgreSQL
	postgresRecords, failedIDs := r.addParticipantsToPostgres(certId, participantIDs)
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
func (r *ParticipantRepository) GetParticipantsByCertId(certId string) ([]*CombinedParticipant, error) {
	// Get PostgreSQL data
	postgresParticipants, pgErr := r.getParticipantsByPostgres(certId)
	if pgErr != nil {
		return nil, fmt.Errorf("failed to get PostgreSQL participants: %w", pgErr)
	}

	// Get MongoDB data
	mongoParticipants, mongoErr := r.getParticipantsByMongo(certId)
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
			ID:             pgParticipant.ID,
			CertificateID:  pgParticipant.CertificateID,
			IsRevoke:       pgParticipant.Isrevoke,
			CertificateURL: pgParticipant.CertificateURL,
			EmailStatus:    pgParticipant.EmailStatus,
			IsDownloaded:   pgParticipant.IsDownloaded,
			CreatedAt:      pgParticipant.CreatedAt,
			UpdatedAt:      pgParticipant.UpdatedAt,
			DynamicData:    make(map[string]any),
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

// GetParticipantsById returns a participant by participant ID
func (r *ParticipantRepository) GetParticipantsById(participantId string) (*CombinedParticipant, error) {
	participant, err := r.getParticipantByIdFromPostgres(participantId)
	if err != nil {
		return nil, err
	}

	participantData, err := r.getParticipantByIdFromMongo(participant.CertificateID, participantId)
	if err != nil {
		return nil, err
	}

	combinedParticipant := &CombinedParticipant{
		ID:             participant.ID,
		CertificateID:  participant.CertificateID,
		IsRevoke:       participant.Isrevoke,
		CertificateURL: participant.CertificateURL,
		EmailStatus:    participant.EmailStatus,
		IsDownloaded:   participant.IsDownloaded,
		CreatedAt:      participant.CreatedAt,
		UpdatedAt:      participant.UpdatedAt,
		DynamicData:    make(map[string]any),
	}

	for key, value := range participantData {
		if key != "_id" && key != "certificate_id" {
			combinedParticipant.DynamicData[key] = value
		}
	}

	return combinedParticipant, nil
}

// DeleteByCertId deletes participants from both PostgreSQL and MongoDB for a certificate
func (r *ParticipantRepository) DeleteByCertId(certId string) ([]*model.Participant, error) {
	// Delete from PostgreSQL first
	participants, err := r.deleteByCertIdFromPostgres(certId)
	if err != nil {
		slog.Error("ParticipantModel DeleteByCertId PostgreSQL deletion failed", "error", err, "cert_id", certId)
		return nil, err
	}

	// Delete MongoDB collection
	err = r.deleteCollectionByCertIdFromMongo(certId)
	if err != nil {
		slog.Error("ParticipantModel DeleteByCertId MongoDB deletion failed", "error", err, "cert_id", certId)
		return participants, err
	}

	slog.Info("ParticipantModel DeleteByCertId completed", "cert_id", certId, "postgres_count", len(participants))
	return participants, nil
}

// EditParticipantByID updates a participant's data with structure validation
func (r *ParticipantRepository) EditParticipantByID(participantID string, newData map[string]any) (*CombinedParticipant, error) {
	// First, get the participant from PostgreSQL to get certificate ID
	participant, err := r.getParticipantByIdFromPostgres(participantID)
	if err != nil {
		slog.Error("ParticipantModel EditParticipantByID: Failed to get participant from PostgreSQL", "error", err, "participant_id", participantID)
		return nil, fmt.Errorf("participant not found: %w", err)
	}

	certId := participant.CertificateID

	// Validate that new data structure matches existing structure
	if err := r.validateEditDataStructure(certId, newData); err != nil {
		slog.Warn("ParticipantModel EditParticipantByID: Data structure validation failed", "error", err, "participant_id", participantID, "cert_id", certId)
		return nil, fmt.Errorf("data structure validation failed: %w", err)
	}

	// Update in MongoDB
	err = r.updateParticipantInMongo(certId, participantID, newData)
	if err != nil {
		slog.Error("ParticipantModel EditParticipantByID: Failed to update MongoDB", "error", err, "participant_id", participantID, "cert_id", certId)
		return nil, fmt.Errorf("failed to update participant data: %w", err)
	}

	// Update timestamp in PostgreSQL
	err = r.updateParticipantTimestampInPostgres(participantID)
	if err != nil {
		slog.Warn("ParticipantModel EditParticipantByID: Failed to update PostgreSQL timestamp", "error", err, "participant_id", participantID)
		// Don't fail the operation for timestamp update failure
	}

	// Return the updated combined participant data
	combinedData := &CombinedParticipant{
		ID:             participant.ID,
		CertificateID:  participant.CertificateID,
		IsRevoke:       participant.Isrevoke,
		CertificateURL: participant.CertificateURL,
		EmailStatus:    participant.EmailStatus,
		IsDownloaded:   participant.IsDownloaded,
		CreatedAt:      participant.CreatedAt,
		UpdatedAt:      time.Now(), // Use current time for updated_at
		DynamicData:    newData,
	}

	slog.Info("ParticipantModel EditParticipantByID completed successfully", "participant_id", participantID, "cert_id", certId)
	return combinedData, nil
}

// DeleteParticipantByID deletes a single participant from both PostgreSQL and MongoDB by participant ID
func (r *ParticipantRepository) DeleteParticipantByID(participantID string) (*model.Participant, error) {
	// First, get the participant from PostgreSQL to get certificate ID and return data
	participant, err := r.getParticipantByIdFromPostgres(participantID)
	if err != nil {
		slog.Error("ParticipantModel DeleteParticipantByID: Failed to get participant from PostgreSQL", "error", err, "participant_id", participantID)
		return nil, fmt.Errorf("participant not found: %w", err)
	}

	certId := participant.CertificateID

	// Delete from PostgreSQL
	err = r.deleteParticipantByIdFromPostgres(participantID)
	if err != nil {
		slog.Error("ParticipantModel DeleteParticipantByID: Failed to delete from PostgreSQL", "error", err, "participant_id", participantID)
		return nil, fmt.Errorf("failed to delete participant from PostgreSQL: %w", err)
	}

	// Delete from MongoDB
	err = r.deleteParticipantByIdFromMongo(certId, participantID)
	if err != nil {
		slog.Error("ParticipantModel DeleteParticipantByID: Failed to delete from MongoDB", "error", err, "participant_id", participantID, "cert_id", certId)
		// Note: PostgreSQL record is already deleted, so we log the error but don't fail the operation completely
		slog.Warn("ParticipantModel DeleteParticipantByID: MongoDB deletion failed but PostgreSQL succeeded", "participant_id", participantID, "cert_id", certId)
	}

	slog.Info("ParticipantModel DeleteParticipantByID completed", "participant_id", participantID, "cert_id", certId)
	return participant, nil
}

// Revoke updates the participant's revoke status in PostgreSQL
func (r *ParticipantRepository) Revoke(id string) (*model.Participant, error) {
	// Get the participant by ID
	participant, err := r.q.Participant.Where(r.q.Participant.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}

	// Update the isrevoke field to true
	_, err = r.q.Participant.Where(r.q.Participant.ID.Eq(id)).Update(r.q.Participant.Isrevoke, true)
	if err != nil {
		return nil, err
	}

	// Return the updated participant
	participant.Isrevoke = true
	return participant, nil
}

// UpdateParticipantCertificateUrl updates the certificate URL for a participant
func (r *ParticipantRepository) UpdateParticipantCertificateUrl(participantId string, certificateUrl string) error {
	_, err := r.q.Participant.Where(r.q.Participant.ID.Eq(participantId)).Update(r.q.Participant.CertificateURL, certificateUrl)
	if err != nil {
		slog.Error("ParticipantModel updateParticipantCertificateUrlInPostgres failed", "error", err, "participantId", participantId, "certificateUrl", certificateUrl)
		return err
	}
	slog.Info("ParticipantModel updateParticipantCertificateUrlInPostgres success", "participantId", participantId)
	return nil
}

// UpdateEmailStatus updates the email status for a participant
func (r *ParticipantRepository) UpdateEmailStatus(participantId string, status string) error {
	_, err := r.q.Participant.Where(r.q.Participant.ID.Eq(participantId)).Update(r.q.Participant.EmailStatus, status)
	if err != nil {
		slog.Error("ParticipantModel UpdateEmailStatus failed", "error", err, "participantId", participantId, "status", status)
		return err
	}
	slog.Info("ParticipantModel UpdateEmailStatus success", "participantId", participantId, "status", status)
	return nil
}

// UpdateDownloadStatus updates the download status for a participant
func (r *ParticipantRepository) UpdateDownloadStatus(participantId string, status bool) error {
	_, err := r.q.Participant.Where(r.q.Participant.ID.Eq(participantId)).Update(r.q.Participant.IsDownloaded, status)
	if err != nil {
		slog.Error("ParticipantModel UpdateDownloadStatus failed", "error", err, "participantId", participantId)
		return err
	}
	slog.Info("ParticipantModel UpdateDownloadStatus success", "participantId", participantId, "status", status)
	return nil
}

// ResetParticipantStatuses resets email_status to "pending" and is_downloaded to false for multiple participants
func (r *ParticipantRepository) ResetParticipantStatuses(participantIds []string) error {
	if len(participantIds) == 0 {
		return nil
	}

	_, err := r.q.Participant.Where(
		r.q.Participant.ID.In(participantIds...),
	).Updates(map[string]any{
		"email_status":  "pending",
		"is_downloaded": false,
	})

	if err != nil {
		slog.Error("ParticipantModel ResetParticipantStatuses failed", "error", err, "count", len(participantIds))
		return err
	}

	slog.Info("ParticipantModel ResetParticipantStatuses success", "count", len(participantIds))
	return nil
}

// MarkAsDownloaded marks a participant as downloaded
func (r *ParticipantRepository) MarkAsDownloaded(participantId string) error {
	return r.UpdateDownloadStatus(participantId, true)
}

// GetParticipantCollectionCount returns the count of participants in the MongoDB collection
func (r *ParticipantRepository) GetParticipantCollectionCount(certId string) (int64, error) {
	collectionName := "participant-" + certId
	collection := r.db.Collection(collectionName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		slog.Error("ParticipantModel GetCollectionCount failed", "error", err, "cert_id", certId)
		return 0, err
	}

	return count, nil
}

// CleanupDeletedAnchors removes fields from all participant documents that are no longer anchors in the certificate design
// ========== Internal helper methods ==========

// addParticipantsToPostgres creates index/status records in PostgreSQL
func (r *ParticipantRepository) addParticipantsToPostgres(certId string, participantIDs []string) ([]*model.Participant, []string) {
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

		// Create record in PostgreSQL using injected query
		createErr := r.q.Participant.Create(participant)
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

// addParticipantsToMongo handles MongoDB insertion with generated IDs
func (r *ParticipantRepository) addParticipantsToMongo(certId string, participants []map[string]any, participantIDs []string) (*mongo.InsertManyResult, error) {
	collectionName := "participant-" + certId
	collection := r.db.Collection(collectionName)

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

// getParticipantsByPostgres returns participants from PostgreSQL by certificate ID
func (r *ParticipantRepository) getParticipantsByPostgres(certId string) ([]*model.Participant, error) {
	participants, err := r.q.Participant.Where(r.q.Participant.CertificateID.Eq(certId)).Find()
	if err != nil {
		slog.Error("ParticipantModel GetParticipantsByPostgres failed", "error", err, "cert_id", certId)
		return nil, err
	}

	slog.Info("ParticipantModel GetParticipantsByPostgres", "cert_id", certId, "count", len(participants))
	return participants, nil
}

// getParticipantsByMongo returns participants from MongoDB by certificate ID
func (r *ParticipantRepository) getParticipantsByMongo(certId string) ([]map[string]any, error) {
	collectionName := "participant-" + certId
	collection := r.db.Collection(collectionName)

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

// getParticipantByIdFromPostgres returns a single participant by ID from PostgreSQL
func (r *ParticipantRepository) getParticipantByIdFromPostgres(participantId string) (*model.Participant, error) {
	participant, err := r.q.Participant.Where(r.q.Participant.ID.Eq(participantId)).First()
	if err != nil {
		slog.Error("ParticipantModel GetParticipantByIdFromPostgres failed", "error", err, "participant_id", participantId)
		return nil, err
	}

	slog.Info("ParticipantModel GetParticipantByIdFromPostgres success", "participant_id", participantId)
	return participant, nil
}

// getParticipantByIdFromMongo returns a specific participant from MongoDB by participant ID
func (r *ParticipantRepository) getParticipantByIdFromMongo(certId string, participantID string) (map[string]any, error) {
	collectionName := "participant-" + certId
	collection := r.db.Collection(collectionName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var participant map[string]any
	err := collection.FindOne(ctx, bson.M{"_id": participantID}).Decode(&participant)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			slog.Warn("ParticipantModel GetParticipantByIdFromMongo: participant not found", "cert_id", certId, "participant_id", participantID)
			return nil, fmt.Errorf("participant not found")
		}
		slog.Error("ParticipantModel GetParticipantByIdFromMongo failed", "error", err, "cert_id", certId, "participant_id", participantID)
		return nil, err
	}

	slog.Info("ParticipantModel GetParticipantByIdFromMongo", "cert_id", certId, "participant_id", participantID)
	return participant, nil
}

// deleteByCertIdFromPostgres deletes participants from PostgreSQL by certificate ID
func (r *ParticipantRepository) deleteByCertIdFromPostgres(certId string) ([]*model.Participant, error) {
	// First get all participants for the certificate to return them
	participants, err := r.q.Participant.Where(r.q.Participant.CertificateID.Eq(certId)).Find()
	if err != nil {
		slog.Error("ParticipantModel DeleteByCertId get participants failed", "error", err, "cert_id", certId)
		return nil, err
	}

	// Delete all participants for the certificate
	result, err := r.q.Participant.Where(r.q.Participant.CertificateID.Eq(certId)).Delete()
	if err != nil {
		slog.Error("ParticipantModel DeleteByCertId delete failed", "error", err, "cert_id", certId)
		return nil, err
	}

	slog.Info("ParticipantModel DeleteByCertId successful", "cert_id", certId, "deleted_count", result.RowsAffected)
	return participants, nil
}

// deleteCollectionByCertIdFromMongo deletes the entire MongoDB collection for a certificate
func (r *ParticipantRepository) deleteCollectionByCertIdFromMongo(certId string) error {
	collectionName := fmt.Sprintf("participant-%s", certId)

	err := r.db.Collection(collectionName).Drop(context.Background())
	if err != nil {
		slog.Error("ParticipantModel DeleteCollectionByCertId failed", "error", err, "cert_id", certId, "collection", collectionName)
		return err
	}

	slog.Info("ParticipantModel DeleteCollectionByCertId successful", "cert_id", certId, "collection", collectionName)
	return nil
}

// deleteParticipantByIdFromPostgres deletes a single participant from PostgreSQL by participant ID
func (r *ParticipantRepository) deleteParticipantByIdFromPostgres(participantId string) error {
	result, err := r.q.Participant.Where(r.q.Participant.ID.Eq(participantId)).Delete()
	if err != nil {
		slog.Error("ParticipantModel deleteParticipantByIdFromPostgres failed", "error", err, "participant_id", participantId)
		return err
	}

	if result.RowsAffected == 0 {
		slog.Warn("ParticipantModel deleteParticipantByIdFromPostgres: no rows deleted", "participant_id", participantId)
		return fmt.Errorf("participant not found")
	}

	slog.Info("ParticipantModel deleteParticipantByIdFromPostgres success", "participant_id", participantId, "rows_affected", result.RowsAffected)
	return nil
}

// deleteParticipantByIdFromMongo deletes a single participant from MongoDB by participant ID
func (r *ParticipantRepository) deleteParticipantByIdFromMongo(certId, participantID string) error {
	collectionName := "participant-" + certId
	collection := r.db.Collection(collectionName)

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

// updateParticipantTimestampInPostgres updates the updated_at timestamp for a participant
func (r *ParticipantRepository) updateParticipantTimestampInPostgres(participantId string) error {
	_, err := r.q.Participant.Where(r.q.Participant.ID.Eq(participantId)).Update(r.q.Participant.UpdatedAt, time.Now())
	if err != nil {
		slog.Error("ParticipantModel updateParticipantTimestampInPostgres failed", "error", err, "participant_id", participantId)
		return err
	}

	slog.Info("ParticipantModel updateParticipantTimestampInPostgres success", "participant_id", participantId)
	return nil
}

// updateParticipantInMongo updates a participant's data in MongoDB
func (r *ParticipantRepository) updateParticipantInMongo(certId, participantID string, newData map[string]any) error {
	collectionName := "participant-" + certId
	collection := r.db.Collection(collectionName)

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
		return fmt.Errorf("participant not found in MongoDB")
	}

	slog.Info("ParticipantModel updateParticipantInMongo successful",
		"cert_id", certId,
		"participant_id", participantID,
		"matched_count", result.MatchedCount,
		"modified_count", result.ModifiedCount)

	return nil
}

// validateEditDataStructure validates that new data matches the certificate design anchors
func (r *ParticipantRepository) validateEditDataStructure(certId string, newData map[string]any) error {
	// Get certificate design to validate against current anchors
	certRepo := certificatemodel.NewCertificateRepository(r.q)
	cert, err := certRepo.GetById(certId)
	if err != nil {
		return fmt.Errorf("failed to get certificate: %w", err)
	}
	if cert == nil {
		return fmt.Errorf("certificate not found")
	}

	// Extract required anchor fields from certificate design
	requiredAnchors, err := r.extractAnchorNames(cert.Design)
	if err != nil {
		return fmt.Errorf("failed to extract anchor names from certificate design: %w", err)
	}

	// Protected fields that should not be validated
	protectedFields := map[string]bool{
		"_id":            true,
		"certificate_id": true,
		"email":          true,
	}

	// Get fields from new data (excluding protected fields)
	var newFields []string
	for key := range newData {
		if !protectedFields[key] {
			newFields = append(newFields, key)
		}
	}
	sort.Strings(newFields)

	// Check if all required anchors are present in the new data
	missingAnchors := []string{}
	for _, anchor := range requiredAnchors {
		if _, exists := newData[anchor]; !exists {
			missingAnchors = append(missingAnchors, anchor)
		}
	}

	// Check if there are any unexpected fields (not in anchors and not protected)
	validAnchors := make(map[string]bool)
	for _, anchor := range requiredAnchors {
		validAnchors[anchor] = true
	}

	invalidFields := []string{}
	for key := range newData {
		if !protectedFields[key] && !validAnchors[key] {
			invalidFields = append(invalidFields, key)
		}
	}

	// Build error message if validation failed
	if len(missingAnchors) > 0 || len(invalidFields) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString("data structure validation failed")

		if len(missingAnchors) > 0 {
			errorMsg.WriteString(fmt.Sprintf(", missing required anchor fields: %s", strings.Join(missingAnchors, ", ")))
		}
		if len(invalidFields) > 0 {
			errorMsg.WriteString(fmt.Sprintf(", invalid fields (not in certificate anchors): %s", strings.Join(invalidFields, ", ")))
		}

		errorMsg.WriteString(fmt.Sprintf(". Required anchors: %s", strings.Join(requiredAnchors, ", ")))

		slog.Warn("ParticipantModel edit validation failed",
			"cert_id", certId,
			"required_anchors", requiredAnchors,
			"provided_fields", newFields,
			"missing_anchors", missingAnchors,
			"invalid_fields", invalidFields)

		return errors.New(errorMsg.String())
	}

	slog.Info("ParticipantModel edit validation passed",
		"cert_id", certId,
		"required_anchors", requiredAnchors,
		"provided_fields", newFields)
	return nil
}


// ========== Private Helper Methods for Validation ==========

// extractAnchorNames extracts anchor names from certificate design JSON
func (r *ParticipantRepository) extractAnchorNames(designJSON string) ([]string, error) {
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

// ValidateFieldConsistency validates that new participants match the certificate design anchors
func (r *ParticipantRepository) ValidateFieldConsistency(certId string, newParticipants []map[string]any) error {
	// Get certificate design to extract required anchor fields
	certRepo := certificatemodel.NewCertificateRepository(r.q)
	cert, err := certRepo.GetById(certId)
	if err != nil {
		return fmt.Errorf("failed to get certificate: %w", err)
	}
	if cert == nil {
		return fmt.Errorf("certificate not found")
	}

	// Extract anchor names from certificate design
	requiredFields, err := r.extractAnchorNames(cert.Design)
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

// CleanupDeletedAnchors removes fields from all participant documents that are no longer anchors in the certificate design
func (r *ParticipantRepository) CleanupDeletedAnchors(certId string, designJSON string) error {
	// Extract current anchor names from certificate design
	currentAnchors, err := r.extractAnchorNames(designJSON)
	if err != nil {
		return fmt.Errorf("failed to extract anchor names: %w", err)
	}

	// Create a set of valid anchor names for quick lookup
	validAnchors := make(map[string]bool)
	for _, anchor := range currentAnchors {
		validAnchors[anchor] = true
	}

	// Always keep these fields
	protectedFields := map[string]bool{
		"_id":            true,
		"certificate_id": true,
		"email":          true,
	}

	collectionName := "participant-" + certId
	collection := r.db.Collection(collectionName)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get all participants
	cursor, err := collection.Find(ctx, bson.M{"certificate_id": certId})
	if err != nil {
		slog.Error("ParticipantModel CleanupDeletedAnchors: failed to find participants", "error", err, "cert_id", certId)
		return fmt.Errorf("failed to find participants: %w", err)
	}
	defer cursor.Close(ctx)

	var participants []map[string]any
	if err = cursor.All(ctx, &participants); err != nil {
		slog.Error("ParticipantModel CleanupDeletedAnchors: failed to decode participants", "error", err, "cert_id", certId)
		return fmt.Errorf("failed to decode participants: %w", err)
	}

	// Process each participant and find fields to remove
	updatedCount := 0
	for _, participant := range participants {
		participantID, ok := participant["_id"].(string)
		if !ok {
			slog.Warn("ParticipantModel CleanupDeletedAnchors: participant missing _id, skipping", "cert_id", certId)
			continue
		}

		// Find fields that should be removed (not in anchors and not protected)
		fieldsToRemove := []string{}
		for key := range participant {
			if !protectedFields[key] && !validAnchors[key] {
				fieldsToRemove = append(fieldsToRemove, key)
			}
		}

		// Update document if there are fields to remove
		if len(fieldsToRemove) > 0 {
			unsetFields := bson.M{}
			for _, field := range fieldsToRemove {
				unsetFields[field] = ""
			}

			filter := bson.M{"_id": participantID, "certificate_id": certId}
			update := bson.M{"$unset": unsetFields}

			result, err := collection.UpdateOne(ctx, filter, update)
			if err != nil {
				slog.Error("ParticipantModel CleanupDeletedAnchors: failed to update participant",
					"error", err,
					"cert_id", certId,
					"participant_id", participantID,
					"fields_to_remove", fieldsToRemove)
				continue
			}

			if result.ModifiedCount > 0 {
				updatedCount++
				slog.Info("ParticipantModel CleanupDeletedAnchors: removed fields from participant",
					"cert_id", certId,
					"participant_id", participantID,
					"removed_fields", fieldsToRemove)
			}
		}
	}

	slog.Info("ParticipantModel CleanupDeletedAnchors completed",
		"cert_id", certId,
		"total_participants", len(participants),
		"updated_count", updatedCount,
		"current_anchors", currentAnchors)

	return nil
}
