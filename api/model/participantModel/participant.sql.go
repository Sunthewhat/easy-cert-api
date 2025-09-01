package participantmodel

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
)

// Revoke updates the participant's revoke status in PostgreSQL
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

// GetParticipantsByPostgres returns participants from PostgreSQL by certificate ID
func GetParticipantsByPostgres(certId string) ([]*model.Participant, error) {
	participants, err := common.Gorm.Participant.Where(common.Gorm.Participant.CertificateID.Eq(certId)).Find()
	if err != nil {
		slog.Error("ParticipantModel GetParticipantsByPostgres failed", "error", err, "cert_id", certId)
		return nil, err
	}

	slog.Info("ParticipantModel GetParticipantsByPostgres", "cert_id", certId, "count", len(participants))
	return participants, nil
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

func DeleteByCertIdFromPostgres(certId string) ([]*model.Participant, error) {
	// First get all participants for the certificate to return them
	participants, err := common.Gorm.Participant.Where(common.Gorm.Participant.CertificateID.Eq(certId)).Find()
	if err != nil {
		slog.Error("ParticipantModel DeleteByCertId get participants failed", "error", err, "cert_id", certId)
		return nil, err
	}

	// Delete all participants for the certificate
	result, err := common.Gorm.Participant.Where(common.Gorm.Participant.CertificateID.Eq(certId)).Delete()
	if err != nil {
		slog.Error("ParticipantModel DeleteByCertId delete failed", "error", err, "cert_id", certId)
		return nil, err
	}

	slog.Info("ParticipantModel DeleteByCertId successful", "cert_id", certId, "deleted_count", result.RowsAffected)
	return participants, nil
}

// GetParticipantByIdFromPostgres returns a single participant by ID from PostgreSQL
func GetParticipantByIdFromPostgres(participantID string) (*model.Participant, error) {
	participant, err := common.Gorm.Participant.Where(common.Gorm.Participant.ID.Eq(participantID)).First()
	if err != nil {
		slog.Error("ParticipantModel GetParticipantByIdFromPostgres failed", "error", err, "participant_id", participantID)
		return nil, err
	}

	slog.Info("ParticipantModel GetParticipantByIdFromPostgres success", "participant_id", participantID)
	return participant, nil
}

// deleteParticipantByIdFromPostgres deletes a single participant from PostgreSQL by participant ID
func deleteParticipantByIdFromPostgres(participantID string) error {
	result, err := common.Gorm.Participant.Where(common.Gorm.Participant.ID.Eq(participantID)).Delete()
	if err != nil {
		slog.Error("ParticipantModel deleteParticipantByIdFromPostgres failed", "error", err, "participant_id", participantID)
		return err
	}

	if result.RowsAffected == 0 {
		slog.Warn("ParticipantModel deleteParticipantByIdFromPostgres: no rows deleted", "participant_id", participantID)
		return fmt.Errorf("participant not found")
	}

	slog.Info("ParticipantModel deleteParticipantByIdFromPostgres success", "participant_id", participantID, "rows_affected", result.RowsAffected)
	return nil
}

// updateParticipantTimestampInPostgres updates the updated_at timestamp for a participant
func updateParticipantTimestampInPostgres(participantID string) error {
	_, err := common.Gorm.Participant.Where(common.Gorm.Participant.ID.Eq(participantID)).Update(common.Gorm.Participant.UpdatedAt, time.Now())
	if err != nil {
		slog.Error("ParticipantModel updateParticipantTimestampInPostgres failed", "error", err, "participant_id", participantID)
		return err
	}

	slog.Info("ParticipantModel updateParticipantTimestampInPostgres success", "participant_id", participantID)
	return nil
}
