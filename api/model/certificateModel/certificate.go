package certificatemodel

import (
	"errors"
	"log/slog"

	"github.com/sunthewhat/secure-docs-api/common"
	"github.com/sunthewhat/secure-docs-api/type/shared/model"
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