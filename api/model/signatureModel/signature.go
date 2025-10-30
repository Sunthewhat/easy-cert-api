package signaturemodel

import (
	"errors"
	"log/slog"
	"time"

	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
	"gorm.io/gorm"
)

func Create(signatureData payload.CreateSignaturePayload, userId string) (*model.Signature, error) {
	signature := &model.Signature{
		SignerID:      signatureData.SignerId,
		CertificateID: signatureData.CertificateId,
		CreatedBy:     userId,
	}

	createErr := common.Gorm.Signature.Create(signature)

	if createErr != nil {
		slog.Error("Create Signature Error", "error", createErr, "data", signatureData, "userId", userId)
		return nil, createErr
	}

	return signature, nil
}

func GetById(signatureId string) (*model.Signature, error) {
	signature, queryErr := common.Gorm.Signature.Where(common.Gorm.Signature.ID.Eq(signatureId)).First()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.Error("Get Signature By Id Error", "error", queryErr, "signatureId", signatureId)
		return nil, queryErr
	}

	return signature, nil
}

func GetByCertificateAndSignerId(certificateId string, signerId string) (*model.Signature, error) {
	slog.Info("Requesting signature", "certId", certificateId, "signerId", signerId)
	signature, queryErr := common.Gorm.Signature.Where(common.Gorm.Signature.CertificateID.Eq(certificateId)).Where(common.Gorm.Signature.SignerID.Eq(signerId)).First()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.Error("Get Signature By Id Error", "error", queryErr)
		return nil, queryErr
	}

	return signature, nil
}

// UpdateSignature updates an existing signature with encrypted signature image
func UpdateSignature(signatureId string, encryptedSignature string) (*model.Signature, error) {
	// Update the signature with encrypted data and mark as signed
	_, err := common.Gorm.Signature.Where(
		common.Gorm.Signature.ID.Eq(signatureId),
	).Updates(map[string]interface{}{
		"signature": encryptedSignature,
		"is_signed": true,
	})

	if err != nil {
		slog.Error("Update Signature Error", "error", err, "signatureId", signatureId)
		return nil, err
	}

	// Fetch and return the updated signature
	updatedSignature, err := common.Gorm.Signature.Where(
		common.Gorm.Signature.ID.Eq(signatureId),
	).First()

	if err != nil {
		slog.Error("Fetch Updated Signature Error", "error", err, "signatureId", signatureId)
		return nil, err
	}

	return updatedSignature, nil
}

// MarkAsRequested marks a signature as requested and updates the last request timestamp
func MarkAsRequested(certificateId, signerId string) error {
	_, err := common.Gorm.Signature.Where(
		common.Gorm.Signature.CertificateID.Eq(certificateId),
	).Where(
		common.Gorm.Signature.SignerID.Eq(signerId),
	).Updates(map[string]interface{}{
		"is_requested": true,
		"last_request": time.Now(),
	})

	if err != nil {
		slog.Error("MarkAsRequested Error", "error", err, "certificateId", certificateId, "signerId", signerId)
		return err
	}

	return nil
}

func UpdateAfterRequestResign(signatureId string) error {
	_, err := common.Gorm.Signature.Where(
		common.Gorm.Signature.ID.Eq(signatureId),
	).Updates(map[string]any{
		"is_requested": true,
		"is_signed":    false,
		"signature":    "",
		"last_request": time.Now(),
	})

	if err != nil {
		slog.Error("UpdateAfterRequestResign Error", "error", err, "signatureId", signatureId)
		return err
	}
	return nil
}

// BulkCreateSignatures creates signature records for multiple signers for a certificate
// Skips signers that already have signatures for this certificate
func BulkCreateSignatures(certificateId string, signerIds []string, userId string) error {
	if len(signerIds) == 0 {
		return nil
	}

	// Check for existing signatures to avoid duplicates
	existingSignatures, queryErr := common.Gorm.Signature.Where(
		common.Gorm.Signature.CertificateID.Eq(certificateId),
	).Where(
		common.Gorm.Signature.SignerID.In(signerIds...),
	).Find()

	if queryErr != nil && !errors.Is(queryErr, gorm.ErrRecordNotFound) {
		slog.Error("BulkCreateSignatures: Error checking existing signatures", "error", queryErr, "certificateId", certificateId)
		return queryErr
	}

	// Create a map of existing signer IDs
	existingSignerIds := make(map[string]bool)
	for _, sig := range existingSignatures {
		existingSignerIds[sig.SignerID] = true
	}

	// Prepare new signatures to create
	var newSignatures []*model.Signature
	for _, signerId := range signerIds {
		if !existingSignerIds[signerId] {
			newSignatures = append(newSignatures, &model.Signature{
				SignerID:      signerId,
				CertificateID: certificateId,
				CreatedBy:     userId,
			})
		}
	}

	// Create all new signatures in bulk
	if len(newSignatures) > 0 {
		createErr := common.Gorm.Signature.Create(newSignatures...)
		if createErr != nil {
			slog.Error("BulkCreateSignatures: Error creating signatures", "error", createErr, "certificateId", certificateId, "count", len(newSignatures))
			return createErr
		}
		slog.Info("BulkCreateSignatures: Created signatures", "certificateId", certificateId, "count", len(newSignatures), "skipped", len(signerIds)-len(newSignatures))
	} else {
		slog.Info("BulkCreateSignatures: All signatures already exist", "certificateId", certificateId)
	}

	return nil
}

// GetPendingSignaturesForReminder returns signatures that need reminder emails
// (requested but not signed, and last request was more than 24 hours ago)
func GetPendingSignaturesForReminder() ([]*model.Signature, error) {
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)

	signatures, queryErr := common.Gorm.Signature.Where(
		common.Gorm.Signature.IsRequested.Is(true),
	).Where(
		common.Gorm.Signature.IsSigned.Is(false),
	).Where(
		common.Gorm.Signature.LastRequest.Lt(twentyFourHoursAgo),
	).Find()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return []*model.Signature{}, nil
		}
		slog.Error("GetPendingSignaturesForReminder Error", "error", queryErr)
		return nil, queryErr
	}

	return signatures, nil
}

// GetSignaturesByCertificate returns all signatures for a specific certificate
func GetSignaturesByCertificate(certificateId string) ([]*model.Signature, error) {
	signatures, queryErr := common.Gorm.Signature.Where(
		common.Gorm.Signature.CertificateID.Eq(certificateId),
	).Find()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return []*model.Signature{}, nil
		}
		slog.Error("GetSignaturesByCertificate Error", "error", queryErr, "certificateId", certificateId)
		return nil, queryErr
	}

	return signatures, nil
}

// DeleteSignature deletes a specific signature by certificate ID and signer ID
func DeleteSignature(certificateId, signerId string) error {
	result, err := common.Gorm.Signature.Where(
		common.Gorm.Signature.CertificateID.Eq(certificateId),
	).Where(
		common.Gorm.Signature.SignerID.Eq(signerId),
	).Delete()

	if err != nil {
		slog.Error("DeleteSignature Error", "error", err, "certificateId", certificateId, "signerId", signerId)
		return err
	}

	slog.Info("DeleteSignature successful", "certificateId", certificateId, "signerId", signerId, "rowsAffected", result.RowsAffected)
	return nil
}

// DeleteSignaturesByCertificate deletes all signatures for a specific certificate
func DeleteSignaturesByCertificate(certificateId string) ([]*model.Signature, error) {
	// First, get all signatures for this certificate so we can return them
	signatures, queryErr := common.Gorm.Signature.Where(
		common.Gorm.Signature.CertificateID.Eq(certificateId),
	).Find()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return []*model.Signature{}, nil
		}
		slog.Error("DeleteSignaturesByCertificate: Error fetching signatures", "error", queryErr, "certificateId", certificateId)
		return nil, queryErr
	}

	// Delete all signatures for this certificate
	result, err := common.Gorm.Signature.Where(
		common.Gorm.Signature.CertificateID.Eq(certificateId),
	).Delete()

	if err != nil {
		slog.Error("DeleteSignaturesByCertificate Error", "error", err, "certificateId", certificateId)
		return nil, err
	}

	slog.Info("DeleteSignaturesByCertificate successful", "certificateId", certificateId, "deletedCount", result.RowsAffected)
	return signatures, nil
}

// AreAllSignaturesComplete checks if all signatures for a certificate are signed
func AreAllSignaturesComplete(certificateId string) (bool, error) {
	// Get all signatures for the certificate
	signatures, err := GetSignaturesByCertificate(certificateId)
	if err != nil {
		slog.Error("AreAllSignaturesComplete: Error fetching signatures", "error", err, "certificateId", certificateId)
		return false, err
	}

	// Check if all signatures are signed
	for _, sig := range signatures {
		if !sig.IsSigned {
			return false, nil
		}
	}

	slog.Info("All signatures complete for certificate", "certificateId", certificateId, "totalSignatures", len(signatures))
	return true, nil
}
