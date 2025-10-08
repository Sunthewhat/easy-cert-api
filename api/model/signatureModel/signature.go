package signaturemodel

import (
	"errors"
	"log/slog"

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

func GetByCertificateAndSignerId(certificateId string, signerId string) (*model.Signature, error) {
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
