package certificatemodel

import (
	"errors"
	"log/slog"

	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
	"github.com/sunthewhat/easy-cert-api/type/shared/query"
	"gorm.io/gorm"
)

// CertificateRepository handles all certificate database operations
type CertificateRepository struct {
	q *query.Query
}

// NewCertificateRepository creates a new certificate repository with dependency injection
func NewCertificateRepository(q *query.Query) *CertificateRepository {
	return &CertificateRepository{q: q}
}

// GetDefaultRepository returns a repository instance using the global query
func GetDefaultRepository() *CertificateRepository {
	return NewCertificateRepository(common.Gorm)
}

// ============================================================================
// Repository Methods (Instance methods for dependency injection)
// ============================================================================

// Create creates a new certificate
func (r *CertificateRepository) Create(certData payload.CreateCertificatePayload, userId string) (*model.Certificate, error) {
	cert := &model.Certificate{
		UserID: userId,
		Name:   certData.Name,
		Design: certData.Design,
	}

	createErr := r.q.Certificate.Create(cert)

	if createErr != nil {
		slog.Error("Certificate Create", "error", createErr, "data", certData, "userId", userId)
		return nil, createErr
	}

	return cert, nil
}

// GetAll retrieves all certificates
func (r *CertificateRepository) GetAll() ([]*model.Certificate, error) {
	certs, queryErr := r.q.Certificate.Find()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.Error("Certificate GetAll", "error", queryErr)
		return nil, queryErr
	}

	return certs, nil
}

// GetByUser retrieves all certificates for a specific user
func (r *CertificateRepository) GetByUser(userId string) ([]*model.Certificate, error) {
	certs, queryErr := r.q.Certificate.Where(r.q.Certificate.UserID.Eq(userId)).Find()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.Error("Certificate GetByUser", "error", queryErr)
		return nil, queryErr
	}

	return certs, nil
}

// GetById retrieves a certificate by ID
func (r *CertificateRepository) GetById(certId string) (*model.Certificate, error) {
	cert, queryErr := r.q.Certificate.Where(r.q.Certificate.ID.Eq(certId)).First()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.Error("Certificate GetById", "error", queryErr)
		return nil, queryErr
	}

	return cert, nil
}

// Delete deletes a certificate by ID
func (r *CertificateRepository) Delete(id string) (*model.Certificate, error) {
	cert, queryErr := r.q.Certificate.Where(r.q.Certificate.ID.Eq(id)).First()
	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, errors.New("certificate not found")
		}
		slog.Error("Certificate Delete find", "error", queryErr)
		return nil, queryErr
	}

	_, deleteErr := r.q.Certificate.Delete(cert)
	if deleteErr != nil {
		slog.Error("Certificate Delete", "error", deleteErr)
		return nil, deleteErr
	}

	return cert, nil
}

// Update updates a certificate's name and/or design
func (r *CertificateRepository) Update(id string, name string, design string) (*model.Certificate, error) {
	cert, queryErr := r.q.Certificate.Where(r.q.Certificate.ID.Eq(id)).First()
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

	_, updateErr := r.q.Certificate.Where(r.q.Certificate.ID.Eq(id)).Updates(updates)
	if updateErr != nil {
		slog.Error("Certificate Update", "error", updateErr)
		return nil, updateErr
	}

	// Fetch updated certificate
	updatedCert, fetchErr := r.q.Certificate.Where(r.q.Certificate.ID.Eq(id)).First()
	if fetchErr != nil {
		slog.Error("Certificate Update fetch", "error", fetchErr)
		return nil, fetchErr
	}

	return updatedCert, nil
}

// AddThumbnailUrl adds or updates the thumbnail URL for a certificate
func (r *CertificateRepository) AddThumbnailUrl(certificateId string, thumbnailUrl string) error {
	_, queryErr := r.q.Certificate.Where(r.q.Certificate.ID.Eq(certificateId)).Update(r.q.Certificate.ThumbnailURL, thumbnailUrl)
	if queryErr != nil {
		slog.Error("Add ThumbnailUrl to certificate failed", "error", queryErr)
		return queryErr
	}
	return nil
}

// EditArchiveUrl updates the archive URL for a certificate
func (r *CertificateRepository) EditArchiveUrl(certificateId string, archiveUrl string) error {
	_, queryErr := r.q.Certificate.Where(r.q.Certificate.ID.Eq(certificateId)).Update(r.q.Certificate.ArchiveURL, archiveUrl)
	if queryErr != nil {
		slog.Error("Edit Archive Url Error", "error", queryErr)
		return queryErr
	}
	return nil
}

// MarkAsDistributed marks a certificate as distributed
func (r *CertificateRepository) MarkAsDistributed(certificateId string) error {
	_, queryErr := r.q.Certificate.Where(r.q.Certificate.ID.Eq(certificateId)).Update(r.q.Certificate.IsDistributed, true)
	if queryErr != nil {
		slog.Error("Mark certificate as distributed Error", "error", queryErr)
		return queryErr
	}
	return nil
}

// MarkAsSigned marks a certificate as fully signed (all signatures complete)
func (r *CertificateRepository) MarkAsSigned(certificateId string) error {
	_, queryErr := r.q.Certificate.Where(r.q.Certificate.ID.Eq(certificateId)).Update(r.q.Certificate.IsSigned, true)
	if queryErr != nil {
		slog.Error("Mark certificate as signed Error", "error", queryErr, "certificate_id", certificateId)
		return queryErr
	}
	slog.Info("Certificate marked as signed", "certificate_id", certificateId)
	return nil
}

// MarkAsUnsigned marks a certificate as not fully signed (has incomplete signatures)
func (r *CertificateRepository) MarkAsUnsigned(certificateId string) error {
	_, queryErr := r.q.Certificate.Where(r.q.Certificate.ID.Eq(certificateId)).Update(r.q.Certificate.IsSigned, false)
	if queryErr != nil {
		slog.Error("Mark certificate as unsigned Error", "error", queryErr, "certificate_id", certificateId)
		return queryErr
	}
	slog.Info("Certificate marked as unsigned", "certificate_id", certificateId)
	return nil
}
