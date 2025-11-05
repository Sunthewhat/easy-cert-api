package util

import (
	"errors"

	"github.com/sunthewhat/easy-cert-api/type/shared"
)

// MockSSOService is a mock implementation of ISSOService for testing
type MockSSOService struct {
	// You can add function fields to control behaviour from tests
	LoginFunc   func(username, password string) (*shared.SsoTokenType, error)
	RefreshFunc func(token string) (*shared.SsoTokenType, error)
	VerifyFunc  func(token string) (*shared.SsoVerifyType, error)
	DecodeFunc  func(token string) (*shared.SsoJwtPayload, error)
}

// NewMockSSOService returns a new mock implementation
func NewMockSSOService() *MockSSOService {
	return &MockSSOService{}
}

// Login calls the configured mock function or returns an error
func (m *MockSSOService) Login(username, password string) (*shared.SsoTokenType, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(username, password)
	}
	return nil, errors.New("Login not implemented in mock")
}

// Refresh calls the configured mock function or returns an error
func (m *MockSSOService) Refresh(token string) (*shared.SsoTokenType, error) {
	if m.RefreshFunc != nil {
		return m.RefreshFunc(token)
	}
	return nil, errors.New("Refresh not implemented in mock")
}

// Verify calls the configured mock function or returns an error
func (m *MockSSOService) Verify(token string) (*shared.SsoVerifyType, error) {
	if m.VerifyFunc != nil {
		return m.VerifyFunc(token)
	}
	return nil, errors.New("Verify not implemented in mock")
}

// Decode calls the configured mock function or returns an error
func (m *MockSSOService) Decode(token string) (*shared.SsoJwtPayload, error) {
	if m.DecodeFunc != nil {
		return m.DecodeFunc(token)
	}
	return nil, errors.New("Decode not implemented in mock")
}
