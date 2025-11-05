package participantmodel

import (
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
)

// IParticipantRepository defines the interface for participant repository operations
type IParticipantRepository interface {
	DeleteByCertId(certId string) ([]*model.Participant, error)
	GetParticipantsByCertId(certId string) ([]*CombinedParticipant, error)
	MarkAsDownloaded(participantId string) error
	ResetParticipantStatuses(participantIds []string) error
	UpdateParticipantCertificateUrl(participantId string, certificateUrl string) error
	UpdateEmailStatus(participantId string, status string) error
	GetParticipantsById(participantId string) (*CombinedParticipant, error)
	CleanupDeletedAnchors(certId string, designJSON string) error
}

// Ensure ParticipantRepository implements IParticipantRepository
var _ IParticipantRepository = (*ParticipantRepository)(nil)

// MockParticipantRepository is a mock implementation for testing
type MockParticipantRepository struct {
	DeleteByCertIdFunc                  func(certId string) ([]*model.Participant, error)
	GetParticipantsByCertIdFunc         func(certId string) ([]*CombinedParticipant, error)
	MarkAsDownloadedFunc                func(participantId string) error
	ResetParticipantStatusesFunc        func(participantIds []string) error
	UpdateParticipantCertificateUrlFunc func(participantId string, certificateUrl string) error
	UpdateEmailStatusFunc               func(participantId string, status string) error
	GetParticipantsByIdFunc             func(participantId string) (*CombinedParticipant, error)
	CleanupDeletedAnchorsFunc           func(certId string, designJSON string) error
}

// Ensure MockParticipantRepository implements IParticipantRepository
var _ IParticipantRepository = (*MockParticipantRepository)(nil)

// NewMockParticipantRepository creates a new mock repository
func NewMockParticipantRepository() *MockParticipantRepository {
	return &MockParticipantRepository{}
}

func (m *MockParticipantRepository) DeleteByCertId(certId string) ([]*model.Participant, error) {
	if m.DeleteByCertIdFunc != nil {
		return m.DeleteByCertIdFunc(certId)
	}
	return nil, nil
}

func (m *MockParticipantRepository) GetParticipantsByCertId(certId string) ([]*CombinedParticipant, error) {
	if m.GetParticipantsByCertIdFunc != nil {
		return m.GetParticipantsByCertIdFunc(certId)
	}
	return nil, nil
}

func (m *MockParticipantRepository) MarkAsDownloaded(participantId string) error {
	if m.MarkAsDownloadedFunc != nil {
		return m.MarkAsDownloadedFunc(participantId)
	}
	return nil
}

func (m *MockParticipantRepository) ResetParticipantStatuses(participantIds []string) error {
	if m.ResetParticipantStatusesFunc != nil {
		return m.ResetParticipantStatusesFunc(participantIds)
	}
	return nil
}

func (m *MockParticipantRepository) UpdateParticipantCertificateUrl(participantId string, certificateUrl string) error {
	if m.UpdateParticipantCertificateUrlFunc != nil {
		return m.UpdateParticipantCertificateUrlFunc(participantId, certificateUrl)
	}
	return nil
}

func (m *MockParticipantRepository) UpdateEmailStatus(participantId string, status string) error {
	if m.UpdateEmailStatusFunc != nil {
		return m.UpdateEmailStatusFunc(participantId, status)
	}
	return nil
}

func (m *MockParticipantRepository) GetParticipantsById(participantId string) (*CombinedParticipant, error) {
	if m.GetParticipantsByIdFunc != nil {
		return m.GetParticipantsByIdFunc(participantId)
	}
	return nil, nil
}

func (m *MockParticipantRepository) CleanupDeletedAnchors(certId string, designJSON string) error {
	if m.CleanupDeletedAnchorsFunc != nil {
		return m.CleanupDeletedAnchorsFunc(certId, designJSON)
	}
	return nil
}
