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
