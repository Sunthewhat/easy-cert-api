package certificatemodel

import (
	"errors"
	"log/slog"

	"github.com/sunthewhat/secure-docs-api/common"
	"github.com/sunthewhat/secure-docs-api/type/shared/model"
	"gorm.io/gorm"
)

func GetAll() ([]*model.Certificate, error) {
	cert, queryErr := common.Gorm.Certificate.Find()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.Error("Certificate GetAll", "error", queryErr)
		return nil, queryErr
	}

	return cert, nil
}
