package signermodel

import (
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
)

// ISignerRepository defines the interface for signer repository operations
type ISignerRepository interface {
	Create(signerData payload.CreateSignerPayload, userId string) (*model.Signer, error)
	GetByUser(userId string) ([]*model.Signer, error)
	GetById(signerId string) (*model.Signer, error)
	GetByEmail(email string, creatorId string) (*model.Signer, error)
	IsEmailExisted(email string) (bool, error)
}

// Ensure SignerRepository implements ISignerRepository
var _ ISignerRepository = (*SignerRepository)(nil)

// MockSignerRepository is a mock implementation for testing
type MockSignerRepository struct {
	CreateFunc         func(signerData payload.CreateSignerPayload, userId string) (*model.Signer, error)
	GetByUserFunc      func(userId string) ([]*model.Signer, error)
	GetByIdFunc        func(signerId string) (*model.Signer, error)
	GetByEmailFunc     func(email string, creatorId string) (*model.Signer, error)
	IsEmailExistedFunc func(email string) (bool, error)
}

// Ensure MockSignerRepository implements ISignerRepository
var _ ISignerRepository = (*MockSignerRepository)(nil)

// NewMockSignerRepository creates a new mock repository
func NewMockSignerRepository() *MockSignerRepository {
	return &MockSignerRepository{}
}

func (m *MockSignerRepository) Create(signerData payload.CreateSignerPayload, userId string) (*model.Signer, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(signerData, userId)
	}
	return nil, nil
}

func (m *MockSignerRepository) GetByUser(userId string) ([]*model.Signer, error) {
	if m.GetByUserFunc != nil {
		return m.GetByUserFunc(userId)
	}
	return nil, nil
}

func (m *MockSignerRepository) GetById(signerId string) (*model.Signer, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(signerId)
	}
	return nil, nil
}

func (m *MockSignerRepository) GetByEmail(email string, creatorId string) (*model.Signer, error) {
	if m.GetByEmailFunc != nil {
		return m.GetByEmailFunc(email, creatorId)
	}
	return nil, nil
}

func (m *MockSignerRepository) IsEmailExisted(email string) (bool, error) {
	if m.IsEmailExistedFunc != nil {
		return m.IsEmailExistedFunc(email)
	}
	return false, nil
}
