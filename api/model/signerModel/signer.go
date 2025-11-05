package signermodel

import (
	"errors"
	"log/slog"

	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
	"github.com/sunthewhat/easy-cert-api/type/shared/query"
	"gorm.io/gorm"
)

// SignerRepository handles all signer database operations
type SignerRepository struct {
	q *query.Query
}

// NewSignerRepository creates a new signer repository with dependency injection
func NewSignerRepository(q *query.Query) *SignerRepository {
	return &SignerRepository{q: q}
}

// ============================================================================
// Repository Methods (Instance methods for dependency injection)
// ============================================================================

// Create creates a new signer
func (r *SignerRepository) Create(signerData payload.CreateSignerPayload, userId string) (*model.Signer, error) {
	signer := &model.Signer{
		Email:       signerData.Email,
		DisplayName: signerData.DisplayName,
		CreatedBy:   userId,
	}

	createErr := r.q.Signer.Create(signer)

	if createErr != nil {
		slog.Error("Signer Create", "error", createErr, "data", signerData, "userId", userId)
		return nil, createErr
	}

	return signer, nil
}

// GetByUser retrieves all signers created by a specific user
func (r *SignerRepository) GetByUser(userId string) ([]*model.Signer, error) {
	signers, queryErr := r.q.Signer.Where(r.q.Signer.CreatedBy.Eq(userId)).Find()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.Error("Signer Get by User", "error", queryErr, "userId", userId)
		return nil, queryErr
	}

	return signers, nil
}

// GetById retrieves a signer by ID
func (r *SignerRepository) GetById(signerId string) (*model.Signer, error) {
	signer, queryErr := r.q.Signer.Where(r.q.Signer.ID.Eq(signerId)).First()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.Error("Get Signer by Id Error", "error", queryErr, "signerId", signerId)
		return nil, queryErr
	}

	return signer, nil
}

// GetByEmail retrieves a signer by email and creator ID
func (r *SignerRepository) GetByEmail(email string, creatorId string) (*model.Signer, error) {
	signer, queryErr := r.q.Signer.Where(r.q.Signer.Email.Eq(email)).Where(r.q.Signer.CreatedBy.Eq(creatorId)).First()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.Error("Get Signer by Email Error", "error", queryErr, "email", email)
		return nil, queryErr
	}

	return signer, nil
}

// IsEmailExisted checks if an email already exists in the signers table
func (r *SignerRepository) IsEmailExisted(email string) (bool, error) {
	_, queryErr := r.q.Signer.Where(r.q.Signer.Email.Eq(email)).First()
	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return false, nil
		}
		slog.Error("Signer model IsEmailExisted Error", "error", queryErr, "email", email)
		return false, nil
	}
	return true, nil
}
