package certificatemodel

import (
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
)

// ICertificateRepository defines the interface for certificate repository operations
type ICertificateRepository interface {
	Create(certData payload.CreateCertificatePayload, userId string) (*model.Certificate, error)
	GetAll() ([]*model.Certificate, error)
	GetByUser(userId string) ([]*model.Certificate, error)
	GetById(certId string) (*model.Certificate, error)
	Delete(id string) (*model.Certificate, error)
	Update(id string, name string, design string) (*model.Certificate, error)
	AddThumbnailUrl(certificateId string, thumbnailUrl string) error
	EditArchiveUrl(certificateId string, archiveUrl string) error
	MarkAsDistributed(certificateId string) error
	MarkAsSigned(certificateId string) error
	MarkAsUnsigned(certificateId string) error
}

// Ensure CertificateRepository implements ICertificateRepository
var _ ICertificateRepository = (*CertificateRepository)(nil)

// MockCertificateRepository is a mock implementation for testing
type MockCertificateRepository struct {
	CreateFunc              func(certData payload.CreateCertificatePayload, userId string) (*model.Certificate, error)
	GetAllFunc              func() ([]*model.Certificate, error)
	GetByUserFunc           func(userId string) ([]*model.Certificate, error)
	GetByIdFunc             func(certId string) (*model.Certificate, error)
	DeleteFunc              func(id string) (*model.Certificate, error)
	UpdateFunc              func(id string, name string, design string) (*model.Certificate, error)
	AddThumbnailUrlFunc     func(certificateId string, thumbnailUrl string) error
	EditArchiveUrlFunc      func(certificateId string, archiveUrl string) error
	MarkAsDistributedFunc   func(certificateId string) error
	MarkAsSignedFunc        func(certificateId string) error
	MarkAsUnsignedFunc      func(certificateId string) error
}

// Ensure MockCertificateRepository implements ICertificateRepository
var _ ICertificateRepository = (*MockCertificateRepository)(nil)

// NewMockCertificateRepository creates a new mock repository
func NewMockCertificateRepository() *MockCertificateRepository {
	return &MockCertificateRepository{}
}

func (m *MockCertificateRepository) Create(certData payload.CreateCertificatePayload, userId string) (*model.Certificate, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(certData, userId)
	}
	return nil, nil
}

func (m *MockCertificateRepository) GetAll() ([]*model.Certificate, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return nil, nil
}

func (m *MockCertificateRepository) GetByUser(userId string) ([]*model.Certificate, error) {
	if m.GetByUserFunc != nil {
		return m.GetByUserFunc(userId)
	}
	return nil, nil
}

func (m *MockCertificateRepository) GetById(certId string) (*model.Certificate, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(certId)
	}
	return nil, nil
}

func (m *MockCertificateRepository) Delete(id string) (*model.Certificate, error) {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil, nil
}

func (m *MockCertificateRepository) Update(id string, name string, design string) (*model.Certificate, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(id, name, design)
	}
	return nil, nil
}

func (m *MockCertificateRepository) AddThumbnailUrl(certificateId string, thumbnailUrl string) error {
	if m.AddThumbnailUrlFunc != nil {
		return m.AddThumbnailUrlFunc(certificateId, thumbnailUrl)
	}
	return nil
}

func (m *MockCertificateRepository) EditArchiveUrl(certificateId string, archiveUrl string) error {
	if m.EditArchiveUrlFunc != nil {
		return m.EditArchiveUrlFunc(certificateId, archiveUrl)
	}
	return nil
}

func (m *MockCertificateRepository) MarkAsDistributed(certificateId string) error {
	if m.MarkAsDistributedFunc != nil {
		return m.MarkAsDistributedFunc(certificateId)
	}
	return nil
}

func (m *MockCertificateRepository) MarkAsSigned(certificateId string) error {
	if m.MarkAsSignedFunc != nil {
		return m.MarkAsSignedFunc(certificateId)
	}
	return nil
}

func (m *MockCertificateRepository) MarkAsUnsigned(certificateId string) error {
	if m.MarkAsUnsignedFunc != nil {
		return m.MarkAsUnsignedFunc(certificateId)
	}
	return nil
}
