package signer_controller_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
	signermodel "github.com/sunthewhat/easy-cert-api/api/model/signerModel"
	signer_controller "github.com/sunthewhat/easy-cert-api/api/controllers/signer"
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
)

func TestSignerController_GetByUser(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(c *fiber.Ctx)
		setupMock      func() *signermodel.MockSignerRepository
		wantStatusCode int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful get by user - with signers",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupMock: func() *signermodel.MockSignerRepository {
				mock := signermodel.NewMockSignerRepository()
				mock.GetByUserFunc = func(userId string) ([]*model.Signer, error) {
					return []*model.Signer{
						{
							ID:          "signer1",
							Email:       "signer1@example.com",
							DisplayName: "Signer One",
							CreatedBy:   userId,
							CreatedAt:   time.Now(),
						},
						{
							ID:          "signer2",
							Email:       "signer2@example.com",
							DisplayName: "Signer Two",
							CreatedBy:   userId,
							CreatedAt:   time.Now(),
						},
					}, nil
				}
				return mock
			},
			wantStatusCode: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["success"] != true {
					t.Errorf("Expected success=true, got %v", response["success"])
				}
				if response["msg"] != "Signer fetched" {
					t.Errorf("Expected msg='Signer fetched', got %v", response["msg"])
				}
				data, ok := response["data"].([]any)
				if !ok {
					t.Fatal("Expected data to be an array")
				}
				if len(data) != 2 {
					t.Errorf("Expected 2 signers, got %d", len(data))
				}
			},
		},
		{
			name: "successful get by user - empty list",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupMock: func() *signermodel.MockSignerRepository {
				mock := signermodel.NewMockSignerRepository()
				mock.GetByUserFunc = func(userId string) ([]*model.Signer, error) {
					return []*model.Signer{}, nil
				}
				return mock
			},
			wantStatusCode: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["success"] != true {
					t.Errorf("Expected success=true, got %v", response["success"])
				}
				data, ok := response["data"].([]any)
				if !ok {
					t.Fatal("Expected data to be an array")
				}
				if len(data) != 0 {
					t.Errorf("Expected 0 signers, got %d", len(data))
				}
			},
		},
		{
			name: "failed - no user in context",
			setupContext: func(c *fiber.Ctx) {
				// Don't set user_id
			},
			setupMock: func() *signermodel.MockSignerRepository {
				return signermodel.NewMockSignerRepository()
			},
			wantStatusCode: fiber.StatusUnauthorized,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["success"] != false {
					t.Errorf("Expected success=false, got %v", response["success"])
				}
				if response["msg"] != "User context failed" {
					t.Errorf("Expected msg='User context failed', got %v", response["msg"])
				}
			},
		},
		{
			name: "failed - database error",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupMock: func() *signermodel.MockSignerRepository {
				mock := signermodel.NewMockSignerRepository()
				mock.GetByUserFunc = func(userId string) ([]*model.Signer, error) {
					return nil, errors.New("database connection error")
				}
				return mock
			},
			wantStatusCode: fiber.StatusInternalServerError,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["success"] != false {
					t.Errorf("Expected success=false, got %v", response["success"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			mockSignerRepo := tt.setupMock()
			mockSignatureRepo := signaturemodel.NewMockSignatureRepository()
			mockCertRepo := certificatemodel.NewMockCertificateRepository()

			ctrl := signer_controller.NewSignerController(mockSignerRepo, mockSignatureRepo, mockCertRepo)

			app.Get("/signer", func(c *fiber.Ctx) error {
				if tt.setupContext != nil {
					tt.setupContext(c)
				}
				return ctrl.GetByUser(c)
			})

			req := httptest.NewRequest("GET", "/signer", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.wantStatusCode, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, body)
			}
		})
	}
}

func TestSignerController_Create(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    any
		setupContext   func(c *fiber.Ctx)
		setupMock      func() *signermodel.MockSignerRepository
		wantStatusCode int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful create",
			requestBody: payload.CreateSignerPayload{
				Email:       "newsigner@example.com",
				DisplayName: "New Signer",
			},
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupMock: func() *signermodel.MockSignerRepository {
				mock := signermodel.NewMockSignerRepository()
				mock.IsEmailExistedFunc = func(email string) (bool, error) {
					return false, nil
				}
				mock.CreateFunc = func(signerData payload.CreateSignerPayload, userId string) (*model.Signer, error) {
					return &model.Signer{
						ID:          "new-signer-id",
						Email:       signerData.Email,
						DisplayName: signerData.DisplayName,
						CreatedBy:   userId,
						CreatedAt:   time.Now(),
					}, nil
				}
				return mock
			},
			wantStatusCode: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["success"] != true {
					t.Errorf("Expected success=true, got %v", response["success"])
				}
				if response["msg"] != "Signer Created" {
					t.Errorf("Expected msg='Signer Created', got %v", response["msg"])
				}
				data, ok := response["data"].(map[string]any)
				if !ok {
					t.Fatal("Expected data to be a map")
				}
				if data["email"] != "newsigner@example.com" {
					t.Errorf("Expected email='newsigner@example.com', got %v", data["email"])
				}
			},
		},
		{
			name:        "failed - invalid request body",
			requestBody: "invalid json",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupMock: func() *signermodel.MockSignerRepository {
				return signermodel.NewMockSignerRepository()
			},
			wantStatusCode: fiber.StatusInternalServerError,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["success"] != false {
					t.Errorf("Expected success=false, got %v", response["success"])
				}
			},
		},
		{
			name: "failed - validation error missing email",
			requestBody: payload.CreateSignerPayload{
				Email:       "",
				DisplayName: "New Signer",
			},
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupMock: func() *signermodel.MockSignerRepository {
				return signermodel.NewMockSignerRepository()
			},
			wantStatusCode: fiber.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["success"] != false {
					t.Errorf("Expected success=false, got %v", response["success"])
				}
			},
		},
		{
			name: "failed - validation error missing display name",
			requestBody: payload.CreateSignerPayload{
				Email:       "test@example.com",
				DisplayName: "",
			},
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupMock: func() *signermodel.MockSignerRepository {
				return signermodel.NewMockSignerRepository()
			},
			wantStatusCode: fiber.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["success"] != false {
					t.Errorf("Expected success=false, got %v", response["success"])
				}
			},
		},
		{
			name: "failed - no user in context",
			requestBody: payload.CreateSignerPayload{
				Email:       "newsigner@example.com",
				DisplayName: "New Signer",
			},
			setupContext: func(c *fiber.Ctx) {
				// Don't set user_id
			},
			setupMock: func() *signermodel.MockSignerRepository {
				return signermodel.NewMockSignerRepository()
			},
			wantStatusCode: fiber.StatusUnauthorized,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["success"] != false {
					t.Errorf("Expected success=false, got %v", response["success"])
				}
				if response["msg"] != "Invalid token context" {
					t.Errorf("Expected msg='Invalid token context', got %v", response["msg"])
				}
			},
		},
		{
			name: "failed - database error checking email existence",
			requestBody: payload.CreateSignerPayload{
				Email:       "test@example.com",
				DisplayName: "New Signer",
			},
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupMock: func() *signermodel.MockSignerRepository {
				mock := signermodel.NewMockSignerRepository()
				mock.IsEmailExistedFunc = func(email string) (bool, error) {
					return false, errors.New("database connection error")
				}
				return mock
			},
			wantStatusCode: fiber.StatusInternalServerError,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["success"] != false {
					t.Errorf("Expected success=false, got %v", response["success"])
				}
			},
		},
		{
			name: "failed - email already exists",
			requestBody: payload.CreateSignerPayload{
				Email:       "existing@example.com",
				DisplayName: "New Signer",
			},
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupMock: func() *signermodel.MockSignerRepository {
				mock := signermodel.NewMockSignerRepository()
				mock.IsEmailExistedFunc = func(email string) (bool, error) {
					return true, nil
				}
				return mock
			},
			wantStatusCode: fiber.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["success"] != false {
					t.Errorf("Expected success=false, got %v", response["success"])
				}
				if response["msg"] != "Signer with this email already existed" {
					t.Errorf("Expected msg='Signer with this email already existed', got %v", response["msg"])
				}
			},
		},
		{
			name: "failed - database error on create",
			requestBody: payload.CreateSignerPayload{
				Email:       "newsigner@example.com",
				DisplayName: "New Signer",
			},
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupMock: func() *signermodel.MockSignerRepository {
				mock := signermodel.NewMockSignerRepository()
				mock.IsEmailExistedFunc = func(email string) (bool, error) {
					return false, nil
				}
				mock.CreateFunc = func(signerData payload.CreateSignerPayload, userId string) (*model.Signer, error) {
					return nil, errors.New("database error")
				}
				return mock
			},
			wantStatusCode: fiber.StatusInternalServerError,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["success"] != false {
					t.Errorf("Expected success=false, got %v", response["success"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			mockSignerRepo := tt.setupMock()
			mockSignatureRepo := signaturemodel.NewMockSignatureRepository()
			mockCertRepo := certificatemodel.NewMockCertificateRepository()

			ctrl := signer_controller.NewSignerController(mockSignerRepo, mockSignatureRepo, mockCertRepo)

			app.Post("/signer", func(c *fiber.Ctx) error {
				if tt.setupContext != nil {
					tt.setupContext(c)
				}
				return ctrl.Create(c)
			})

			var bodyReader io.Reader
			if str, ok := tt.requestBody.(string); ok {
				bodyReader = bytes.NewBufferString(str)
			} else {
				bodyBytes, err := json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
				bodyReader = bytes.NewBuffer(bodyBytes)
			}

			req := httptest.NewRequest("POST", "/signer", bodyReader)
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.wantStatusCode, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, body)
			}
		})
	}
}

func TestSignerController_GetStatus(t *testing.T) {
	tests := []struct {
		name               string
		certId             string
		setupContext       func(c *fiber.Ctx)
		setupSignerMock    func() *signermodel.MockSignerRepository
		setupSignatureMock func() *signaturemodel.MockSignatureRepository
		setupCertMock      func() *certificatemodel.MockCertificateRepository
		wantStatusCode     int
		checkResponse      func(t *testing.T, body []byte)
	}{
		{
			name:   "successful get status",
			certId: "cert123",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupSignerMock: func() *signermodel.MockSignerRepository {
				mock := signermodel.NewMockSignerRepository()
				mock.GetByIdFunc = func(signerId string) (*model.Signer, error) {
					if signerId == "signer1" {
						return &model.Signer{
							ID:          "signer1",
							Email:       "signer1@example.com",
							DisplayName: "Signer One",
						}, nil
					}
					if signerId == "signer2" {
						return &model.Signer{
							ID:          "signer2",
							Email:       "signer2@example.com",
							DisplayName: "Signer Two",
						}, nil
					}
					return nil, errors.New("signer not found")
				}
				return mock
			},
			setupSignatureMock: func() *signaturemodel.MockSignatureRepository {
				mock := signaturemodel.NewMockSignatureRepository()
				mock.GetSignaturesByCertificateFunc = func(certId string) ([]*model.Signature, error) {
					return []*model.Signature{
						{
							ID:          "sig1",
							SignerID:    "signer1",
							IsSigned:    true,
							IsRequested: true,
						},
						{
							ID:          "sig2",
							SignerID:    "signer2",
							IsSigned:    false,
							IsRequested: false,
						},
					}, nil
				}
				return mock
			},
			setupCertMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:     certId,
						UserID: "user123@example.com",
						Name:   "Test Certificate",
					}, nil
				}
				return mock
			},
			wantStatusCode: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["success"] != true {
					t.Errorf("Expected success=true, got %v", response["success"])
				}
				if response["msg"] != "Get signer status successfully" {
					t.Errorf("Expected msg='Get signer status successfully', got %v", response["msg"])
				}
				data, ok := response["data"].([]any)
				if !ok {
					t.Fatal("Expected data to be an array")
				}
				if len(data) != 2 {
					t.Errorf("Expected 2 signature records, got %d", len(data))
				}
			},
		},
		{
			name:   "failed - no user in context",
			certId: "cert123",
			setupContext: func(c *fiber.Ctx) {
				// Don't set user_id
			},
			setupSignerMock: func() *signermodel.MockSignerRepository {
				return signermodel.NewMockSignerRepository()
			},
			setupSignatureMock: func() *signaturemodel.MockSignatureRepository {
				return signaturemodel.NewMockSignatureRepository()
			},
			setupCertMock: func() *certificatemodel.MockCertificateRepository {
				return certificatemodel.NewMockCertificateRepository()
			},
			wantStatusCode: fiber.StatusUnauthorized,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["success"] != false {
					t.Errorf("Expected success=false, got %v", response["success"])
				}
				if response["msg"] != "User context failed" {
					t.Errorf("Expected msg='User context failed', got %v", response["msg"])
				}
			},
		},
		{
			name:   "failed - certificate not found",
			certId: "cert123",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupSignerMock: func() *signermodel.MockSignerRepository {
				return signermodel.NewMockSignerRepository()
			},
			setupSignatureMock: func() *signaturemodel.MockSignatureRepository {
				return signaturemodel.NewMockSignatureRepository()
			},
			setupCertMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return nil, errors.New("certificate not found")
				}
				return mock
			},
			wantStatusCode: fiber.StatusInternalServerError,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["success"] != false {
					t.Errorf("Expected success=false, got %v", response["success"])
				}
			},
		},
		{
			name:   "failed - user does not own certificate",
			certId: "cert123",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupSignerMock: func() *signermodel.MockSignerRepository {
				return signermodel.NewMockSignerRepository()
			},
			setupSignatureMock: func() *signaturemodel.MockSignatureRepository {
				return signaturemodel.NewMockSignatureRepository()
			},
			setupCertMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:     certId,
						UserID: "different-user@example.com",
						Name:   "Test Certificate",
					}, nil
				}
				return mock
			},
			wantStatusCode: fiber.StatusUnauthorized,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["success"] != false {
					t.Errorf("Expected success=false, got %v", response["success"])
				}
				if response["msg"] != "You did not own this certificate" {
					t.Errorf("Expected msg='You did not own this certificate', got %v", response["msg"])
				}
			},
		},
		{
			name:   "failed - error getting signatures",
			certId: "cert123",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupSignerMock: func() *signermodel.MockSignerRepository {
				return signermodel.NewMockSignerRepository()
			},
			setupSignatureMock: func() *signaturemodel.MockSignatureRepository {
				mock := signaturemodel.NewMockSignatureRepository()
				mock.GetSignaturesByCertificateFunc = func(certId string) ([]*model.Signature, error) {
					return nil, errors.New("database error")
				}
				return mock
			},
			setupCertMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:     certId,
						UserID: "user123@example.com",
						Name:   "Test Certificate",
					}, nil
				}
				return mock
			},
			wantStatusCode: fiber.StatusInternalServerError,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["success"] != false {
					t.Errorf("Expected success=false, got %v", response["success"])
				}
			},
		},
		{
			name:   "successful get status - with signer error (partial data)",
			certId: "cert123",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupSignerMock: func() *signermodel.MockSignerRepository {
				mock := signermodel.NewMockSignerRepository()
				mock.GetByIdFunc = func(signerId string) (*model.Signer, error) {
					if signerId == "signer1" {
						return &model.Signer{
							ID:          "signer1",
							Email:       "signer1@example.com",
							DisplayName: "Signer One",
						}, nil
					}
					// Simulate error for signer2
					if signerId == "signer2" {
						return nil, errors.New("signer not found")
					}
					return nil, errors.New("signer not found")
				}
				return mock
			},
			setupSignatureMock: func() *signaturemodel.MockSignatureRepository {
				mock := signaturemodel.NewMockSignatureRepository()
				mock.GetSignaturesByCertificateFunc = func(certId string) ([]*model.Signature, error) {
					return []*model.Signature{
						{
							ID:          "sig1",
							SignerID:    "signer1",
							IsSigned:    true,
							IsRequested: true,
						},
						{
							ID:          "sig2",
							SignerID:    "signer2",
							IsSigned:    false,
							IsRequested: false,
						},
					}, nil
				}
				return mock
			},
			setupCertMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:     certId,
						UserID: "user123@example.com",
						Name:   "Test Certificate",
					}, nil
				}
				return mock
			},
			wantStatusCode: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["success"] != true {
					t.Errorf("Expected success=true, got %v", response["success"])
				}
				data, ok := response["data"].([]any)
				if !ok {
					t.Fatal("Expected data to be an array")
				}
				// Should only have 1 signature (signer1 succeeded, signer2 failed and was skipped)
				if len(data) != 1 {
					t.Errorf("Expected 1 signature record (error on signer2 should be skipped), got %d", len(data))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			mockSignerRepo := tt.setupSignerMock()
			mockSignatureRepo := tt.setupSignatureMock()
			mockCertRepo := tt.setupCertMock()

			ctrl := signer_controller.NewSignerController(mockSignerRepo, mockSignatureRepo, mockCertRepo)

			app.Get("/signer/status/:certId", func(c *fiber.Ctx) error {
				if tt.setupContext != nil {
					tt.setupContext(c)
				}
				return ctrl.GetStatus(c)
			})

			req := httptest.NewRequest("GET", "/signer/status/"+tt.certId, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.wantStatusCode, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, body)
			}
		})
	}
}
