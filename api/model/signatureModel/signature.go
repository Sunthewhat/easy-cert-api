package signaturemodel

import (
	"errors"
	"log/slog"
	"time"

	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
	"github.com/sunthewhat/easy-cert-api/type/shared/query"
	"gorm.io/gorm"
)

// SignatureRepository handles all signature database operations
type SignatureRepository struct {
	q *query.Query
}

// NewSignatureRepository creates a new signature repository with dependency injection
func NewSignatureRepository(q *query.Query) *SignatureRepository {
	return &SignatureRepository{q: q}
}

// ============================================================================
// Repository Methods (Instance methods for dependency injection)
// ============================================================================

// Create creates a new signature
func (r *SignatureRepository) Create(signatureData payload.CreateSignaturePayload, userId string) (*model.Signature, error) {
	signature := &model.Signature{
		SignerID:      signatureData.SignerId,
		CertificateID: signatureData.CertificateId,
		CreatedBy:     userId,
	}

	createErr := r.q.Signature.Create(signature)

	if createErr != nil {
		slog.Error("Create Signature Error", "error", createErr, "data", signatureData, "userId", userId)
		return nil, createErr
	}

	return signature, nil
}

// GetById retrieves a signature by ID
func (r *SignatureRepository) GetById(signatureId string) (*model.Signature, error) {
	signature, queryErr := r.q.Signature.Where(r.q.Signature.ID.Eq(signatureId)).First()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.Error("Get Signature By Id Error", "error", queryErr, "signatureId", signatureId)
		return nil, queryErr
	}

	return signature, nil
}

// GetByCertificateAndSignerId retrieves a signature by certificate ID and signer ID
func (r *SignatureRepository) GetByCertificateAndSignerId(certificateId string, signerId string) (*model.Signature, error) {
	slog.Info("Requesting signature", "certId", certificateId, "signerId", signerId)
	signature, queryErr := r.q.Signature.Where(r.q.Signature.CertificateID.Eq(certificateId)).Where(r.q.Signature.SignerID.Eq(signerId)).First()

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
func (r *SignatureRepository) UpdateSignature(signatureId string, encryptedSignature string) (*model.Signature, error) {
	// Update the signature with encrypted data and mark as signed
	_, err := r.q.Signature.Where(
		r.q.Signature.ID.Eq(signatureId),
	).Updates(map[string]interface{}{
		"signature": encryptedSignature,
		"is_signed": true,
	})

	if err != nil {
		slog.Error("Update Signature Error", "error", err, "signatureId", signatureId)
		return nil, err
	}

	// Fetch and return the updated signature
	updatedSignature, err := r.q.Signature.Where(
		r.q.Signature.ID.Eq(signatureId),
	).First()

	if err != nil {
		slog.Error("Fetch Updated Signature Error", "error", err, "signatureId", signatureId)
		return nil, err
	}

	return updatedSignature, nil
}

// MarkAsRequested marks a signature as requested and updates the last request timestamp
func (r *SignatureRepository) MarkAsRequested(certificateId, signerId string) error {
	_, err := r.q.Signature.Where(
		r.q.Signature.CertificateID.Eq(certificateId),
	).Where(
		r.q.Signature.SignerID.Eq(signerId),
	).Updates(map[string]interface{}{
		"is_requested": true,
		"last_request": time.Now(),
	})

	if err != nil {
		slog.Error("MarkAsRequested Error", "error", err, "certificateId", certificateId, "signerId", signerId)
		return err
	}

	return nil
}

// UpdateAfterRequestResign updates signature after requesting a re-signature
func (r *SignatureRepository) UpdateAfterRequestResign(signatureId string) error {
	_, err := r.q.Signature.Where(
		r.q.Signature.ID.Eq(signatureId),
	).Updates(map[string]any{
		"is_requested": true,
		"is_signed":    false,
		"signature":    "",
		"last_request": time.Now(),
	})

	if err != nil {
		slog.Error("UpdateAfterRequestResign Error", "error", err, "signatureId", signatureId)
		return err
	}
	return nil
}

// BulkCreateSignatures creates signature records for multiple signers for a certificate
// Skips signers that already have signatures for this certificate
func (r *SignatureRepository) BulkCreateSignatures(certificateId string, signerIds []string, userId string) error {
	if len(signerIds) == 0 {
		return nil
	}

	// Check for existing signatures to avoid duplicates
	existingSignatures, queryErr := r.q.Signature.Where(
		r.q.Signature.CertificateID.Eq(certificateId),
	).Where(
		r.q.Signature.SignerID.In(signerIds...),
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
		createErr := r.q.Signature.Create(newSignatures...)
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

// GetPendingSignaturesForReminder returns signatures that need reminder emails
// (requested but not signed, and last request was more than 24 hours ago)
func (r *SignatureRepository) GetPendingSignaturesForReminder() ([]*model.Signature, error) {
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)

	signatures, queryErr := r.q.Signature.Where(
		r.q.Signature.IsRequested.Is(true),
	).Where(
		r.q.Signature.IsSigned.Is(false),
	).Where(
		r.q.Signature.LastRequest.Lt(twentyFourHoursAgo),
	).Find()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return []*model.Signature{}, nil
		}
		slog.Error("GetPendingSignaturesForReminder Error", "error", queryErr)
		return nil, queryErr
	}

	return signatures, nil
}

// GetSignaturesByCertificate returns all signatures for a specific certificate
func (r *SignatureRepository) GetSignaturesByCertificate(certificateId string) ([]*model.Signature, error) {
	signatures, queryErr := r.q.Signature.Where(
		r.q.Signature.CertificateID.Eq(certificateId),
	).Find()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return []*model.Signature{}, nil
		}
		slog.Error("GetSignaturesByCertificate Error", "error", queryErr, "certificateId", certificateId)
		return nil, queryErr
	}

	return signatures, nil
}

// DeleteSignature deletes a specific signature by certificate ID and signer ID
func (r *SignatureRepository) DeleteSignature(certificateId, signerId string) error {
	result, err := r.q.Signature.Where(
		r.q.Signature.CertificateID.Eq(certificateId),
	).Where(
		r.q.Signature.SignerID.Eq(signerId),
	).Delete()

	if err != nil {
		slog.Error("DeleteSignature Error", "error", err, "certificateId", certificateId, "signerId", signerId)
		return err
	}

	slog.Info("DeleteSignature successful", "certificateId", certificateId, "signerId", signerId, "rowsAffected", result.RowsAffected)
	return nil
}

// DeleteSignaturesByCertificate deletes all signatures for a specific certificate
func (r *SignatureRepository) DeleteSignaturesByCertificate(certificateId string) ([]*model.Signature, error) {
	// First, get all signatures for this certificate so we can return them
	signatures, queryErr := r.q.Signature.Where(
		r.q.Signature.CertificateID.Eq(certificateId),
	).Find()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return []*model.Signature{}, nil
		}
		slog.Error("DeleteSignaturesByCertificate: Error fetching signatures", "error", queryErr, "certificateId", certificateId)
		return nil, queryErr
	}

	// Delete all signatures for this certificate
	result, err := r.q.Signature.Where(
		r.q.Signature.CertificateID.Eq(certificateId),
	).Delete()

	if err != nil {
		slog.Error("DeleteSignaturesByCertificate Error", "error", err, "certificateId", certificateId)
		return nil, err
	}

	slog.Info("DeleteSignaturesByCertificate successful", "certificateId", certificateId, "deletedCount", result.RowsAffected)
	return signatures, nil
}

// AreAllSignaturesComplete checks if all signatures for a certificate are signed
func (r *SignatureRepository) AreAllSignaturesComplete(certificateId string) (bool, error) {
	// Get all signatures for the certificate
	signatures, err := r.GetSignaturesByCertificate(certificateId)
	if err != nil {
		slog.Error("AreAllSignaturesComplete: Error fetching signatures", "error", err, "certificateId", certificateId)
		return false, err
	}

	// If there are no signatures, return false (nothing to complete)
	if len(signatures) == 0 {
		slog.Info("No signatures found for certificate", "certificateId", certificateId)
		return false, nil
	}

	// Check if all signatures are signed
	for _, sig := range signatures {
		if !sig.IsSigned {
			return false, nil
		}
	}

	slog.Info("All signatures complete for certificate", "certificateId", certificateId, "totalSignatures", len(signatures))
	return true, nil
}
