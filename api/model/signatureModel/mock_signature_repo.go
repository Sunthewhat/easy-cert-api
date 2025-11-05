package signaturemodel

import (
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
)

// ISignatureRepository defines the interface for signature repository operations
type ISignatureRepository interface {
	GetSignaturesByCertificate(certId string) ([]*model.Signature, error)
	GetById(signatureId string) (*model.Signature, error)
	DeleteSignaturesByCertificate(certificateId string) ([]*model.Signature, error)
	AreAllSignaturesComplete(certificateId string) (bool, error)
	BulkCreateSignatures(certificateId string, signerIds []string, userId string) error
	DeleteSignature(certificateId, signerId string) error
}

// Ensure SignatureRepository implements ISignatureRepository
var _ ISignatureRepository = (*SignatureRepository)(nil)

// MockSignatureRepository is a mock implementation for testing
type MockSignatureRepository struct {
	GetSignaturesByCertificateFunc    func(certId string) ([]*model.Signature, error)
	GetByIdFunc                       func(signatureId string) (*model.Signature, error)
	DeleteSignaturesByCertificateFunc func(certificateId string) ([]*model.Signature, error)
	AreAllSignaturesCompleteFunc      func(certificateId string) (bool, error)
	BulkCreateSignaturesFunc          func(certificateId string, signerIds []string, userId string) error
	DeleteSignatureFunc               func(certificateId, signerId string) error
}

// Ensure MockSignatureRepository implements ISignatureRepository
var _ ISignatureRepository = (*MockSignatureRepository)(nil)

// NewMockSignatureRepository creates a new mock repository
func NewMockSignatureRepository() *MockSignatureRepository {
	return &MockSignatureRepository{}
}

func (m *MockSignatureRepository) GetSignaturesByCertificate(certId string) ([]*model.Signature, error) {
	if m.GetSignaturesByCertificateFunc != nil {
		return m.GetSignaturesByCertificateFunc(certId)
	}
	return nil, nil
}

func (m *MockSignatureRepository) GetById(signatureId string) (*model.Signature, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(signatureId)
	}
	return nil, nil
}

func (m *MockSignatureRepository) DeleteSignaturesByCertificate(certificateId string) ([]*model.Signature, error) {
	if m.DeleteSignaturesByCertificateFunc != nil {
		return m.DeleteSignaturesByCertificateFunc(certificateId)
	}
	return nil, nil
}

func (m *MockSignatureRepository) AreAllSignaturesComplete(certificateId string) (bool, error) {
	if m.AreAllSignaturesCompleteFunc != nil {
		return m.AreAllSignaturesCompleteFunc(certificateId)
	}
	return false, nil
}

func (m *MockSignatureRepository) BulkCreateSignatures(certificateId string, signerIds []string, userId string) error {
	if m.BulkCreateSignaturesFunc != nil {
		return m.BulkCreateSignaturesFunc(certificateId, signerIds, userId)
	}
	return nil
}

func (m *MockSignatureRepository) DeleteSignature(certificateId, signerId string) error {
	if m.DeleteSignatureFunc != nil {
		return m.DeleteSignatureFunc(certificateId, signerId)
	}
	return nil
}
