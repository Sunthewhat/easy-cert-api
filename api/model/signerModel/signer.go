package signermodel

import (
	"errors"
	"log/slog"

	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
	"gorm.io/gorm"
)

func Create(signerData payload.CreateSignerPayload, userId string) (*model.Signer, error) {
	signer := &model.Signer{
		Email:       signerData.Email,
		DisplayName: signerData.DisplayName,
		CreatedBy:   userId,
	}

	createErr := common.Gorm.Signer.Create(signer)

	if createErr != nil {
		slog.Error("Signer Create", "error", createErr, "data", signerData, "userId", userId)
		return nil, createErr
	}

	return signer, nil
}

func GetByUser(userId string) ([]*model.Signer, error) {
	signers, queryErr := common.Gorm.Signer.Where(common.Gorm.Signer.CreatedBy.Eq(userId)).Find()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.Error("Signer Get by User", "error", queryErr, "userId", userId)
		return nil, queryErr
	}

	return signers, nil
}

func GetById(signerId string) (*model.Signer, error) {
	signer, queryErr := common.Gorm.Signer.Where(common.Gorm.Signer.ID.Eq(signerId)).First()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.Error("Get Signer by Id Error", "error", queryErr, "signerId", signerId)
		return nil, queryErr
	}

	return signer, nil
}

func GetByEmail(email string) (*model.Signer, error) {
	signer, queryErr := common.Gorm.Signer.Where(common.Gorm.Signer.Email.Eq(email)).First()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.Error("Get Signer by Email Error", "error", queryErr, "email", email)
		return nil, queryErr
	}

	return signer, nil
}
