package certificatemodel

import (
	"errors"
	"log/slog"

	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
	"gorm.io/gorm"
)

func Create(certData payload.CreateCertificatePayload, userId string) (*model.Certificate, error) {
	cert := &model.Certificate{
		UserID: userId,
		Name:   certData.Name,
		Design: certData.Design,
	}

	createErr := common.Gorm.Certificate.Create(cert)

	if createErr != nil {
		slog.Error("Certificate Create", "error", createErr, "data", certData, "userId", userId)
		return nil, createErr
	}

	return cert, nil
}

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

func GetByUser(userId string) ([]*model.Certificate, error) {
	certs, queryErr := common.Gorm.Certificate.Where(common.Gorm.Certificate.UserID.Eq(userId)).Find()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.Error("Certificate GetByUser", "error", queryErr)
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

func AddThumbnailUrl(certificateId string, thumbnailUrl string) error {
	_, queryErr := common.Gorm.Certificate.Where(common.Gorm.Certificate.ID.Eq(certificateId)).Update(common.Gorm.Certificate.ThumbnailURL, thumbnailUrl)
	if queryErr != nil {
		slog.Error("Add ThumbnailUrl to certificate failed", "error", queryErr)
		return queryErr
	}
	return nil
}

func EditArchiveUrl(certificateId string, archiveUrl string) error {
	_, queryErr := common.Gorm.Certificate.Where(common.Gorm.Certificate.ID.Eq(certificateId)).Update(common.Gorm.Certificate.ArchiveURL, archiveUrl)
	if queryErr != nil {
		slog.Error("Edit Archive Url Error", "error", queryErr)
		return queryErr
	}
	return nil
}

func MarkAsDistributed(certificateId string) error {
	_, queryErr := common.Gorm.Certificate.Where(common.Gorm.Certificate.ID.Eq(certificateId)).Update(common.Gorm.Certificate.IsDistributed, true)
	if queryErr != nil {
		slog.Error("Mark certificate as distributed Error", "error", queryErr)
		return queryErr
	}
	return nil
}
