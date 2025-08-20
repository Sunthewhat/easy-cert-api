package certificatemodel

import (
	"errors"
	"log/slog"

	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
	"gorm.io/gorm"
)

func GetAll() ([]*model.Certificate, error) {
	certs, queryErr := common.Gorm.Certificate.Find()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.Error("Certificate GetAll", "error", queryErr)
		return nil, queryErr
	}

	return certs, nil
}

func GetById(certId string) (*model.Certificate, error) {
	cert, queryErr := common.Gorm.Certificate.Where(common.Gorm.Certificate.ID.Eq(certId)).First()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.Error("Certificate GetById", "error", queryErr)
		return nil, queryErr
	}

	return cert, nil
}

func Delete(id string) (*model.Certificate, error) {
	cert, queryErr := common.Gorm.Certificate.Where(common.Gorm.Certificate.ID.Eq(id)).First()
	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, errors.New("certificate not found")
		}
		slog.Error("Certificate Delete find", "error", queryErr)
		return nil, queryErr
	}

	_, deleteErr := common.Gorm.Certificate.Delete(cert)
	if deleteErr != nil {
		slog.Error("Certificate Delete", "error", deleteErr)
		return nil, deleteErr
	}

	return cert, nil
}

func Update(id string, name string, design string) (*model.Certificate, error) {
	cert, queryErr := common.Gorm.Certificate.Where(common.Gorm.Certificate.ID.Eq(id)).First()
	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, errors.New("certificate not found")
		}
		slog.Error("Certificate Update find", "error", queryErr)
		return nil, queryErr
	}

	updates := make(map[string]any)
	if name != "" {
		updates["name"] = name
	}
	if design != "" {
		updates["design"] = design
	}

	if len(updates) == 0 {
		return cert, nil
	}

	_, updateErr := common.Gorm.Certificate.Where(common.Gorm.Certificate.ID.Eq(id)).Updates(updates)
	if updateErr != nil {
		slog.Error("Certificate Update", "error", updateErr)
		return nil, updateErr
	}

	// Fetch updated certificate
	updatedCert, fetchErr := common.Gorm.Certificate.Where(common.Gorm.Certificate.ID.Eq(id)).First()
	if fetchErr != nil {
		slog.Error("Certificate Update fetch", "error", fetchErr)
		return nil, fetchErr
	}

	return updatedCert, nil
}
