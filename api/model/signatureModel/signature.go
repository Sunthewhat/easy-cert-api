package signaturemodel

import (
	"log/slog"

	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
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
