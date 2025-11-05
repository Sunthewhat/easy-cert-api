package certificate_controller_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	certificate_controller "github.com/sunthewhat/easy-cert-api/api/controllers/certificate"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
)

func TestCertificateController_GetByUser(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(c *fiber.Ctx)
		setupMock      func() *certificatemodel.MockCertificateRepository
		wantStatusCode int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful get by user - with certificates",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.GetByUserFunc = func(userId string) ([]*model.Certificate, error) {
					return []*model.Certificate{
						{
							ID:     "cert1",
							UserID: userId,
							Name:   "Certificate One",
							Design: "design1.html",
							CreatedAt: time.Now(),
						},
						{
							ID:     "cert2",
							UserID: userId,
							Name:   "Certificate Two",
							Design: "design2.html",
							CreatedAt: time.Now(),
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
				if response["msg"] != "Certificate fetched" {
					t.Errorf("Expected msg='Certificate fetched', got %v", response["msg"])
				}
				data, ok := response["data"].([]any)
				if !ok {
					t.Fatal("Expected data to be an array")
				}
				if len(data) != 2 {
					t.Errorf("Expected 2 certificates, got %d", len(data))
				}
			},
		},
		{
			name: "successful get by user - empty list",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.GetByUserFunc = func(userId string) ([]*model.Certificate, error) {
					return []*model.Certificate{}, nil
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
					t.Errorf("Expected 0 certificates, got %d", len(data))
				}
			},
		},
		{
			name: "failed - no user in context",
			setupContext: func(c *fiber.Ctx) {
				// Don't set user_id
			},
			setupMock: func() *certificatemodel.MockCertificateRepository {
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
				if response["msg"] != "User token not found" {
					t.Errorf("Expected msg='User token not found', got %v", response["msg"])
				}
			},
		},
		{
			name: "failed - database error",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.GetByUserFunc = func(userId string) ([]*model.Certificate, error) {
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
			mockCertRepo := tt.setupMock()
			mockSignatureRepo := signaturemodel.NewMockSignatureRepository()
			mockParticipantRepo := participantmodel.NewMockParticipantRepository()

			ctrl := certificate_controller.NewCertificateController(mockCertRepo, mockSignatureRepo, mockParticipantRepo)

			app.Get("/certificate", func(c *fiber.Ctx) error {
				if tt.setupContext != nil {
					tt.setupContext(c)
				}
				return ctrl.GetByUser(c)
			})

			req := httptest.NewRequest("GET", "/certificate", nil)
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

func TestCertificateController_Create(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    any
		setupContext   func(c *fiber.Ctx)
		setupMock      func() *certificatemodel.MockCertificateRepository
		wantStatusCode int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful create",
			requestBody: payload.CreateCertificatePayload{
				Name:   "New Certificate",
				Design: "design.html",
			},
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.CreateFunc = func(certData payload.CreateCertificatePayload, userId string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:        "new-cert-id",
						UserID:    userId,
						Name:      certData.Name,
						Design:    certData.Design,
						CreatedAt: time.Now(),
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
				if response["msg"] != "Certificate Created" {
					t.Errorf("Expected msg='Certificate Created', got %v", response["msg"])
				}
				data, ok := response["data"].(map[string]any)
				if !ok {
					t.Fatal("Expected data to be a map")
				}
				if data["name"] != "New Certificate" {
					t.Errorf("Expected name='New Certificate', got %v", data["name"])
				}
			},
		},
		{
			name:        "failed - invalid request body",
			requestBody: "invalid json",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupMock: func() *certificatemodel.MockCertificateRepository {
				return certificatemodel.NewMockCertificateRepository()
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
			name: "failed - validation error missing name",
			requestBody: payload.CreateCertificatePayload{
				Name:   "",
				Design: "design.html",
			},
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupMock: func() *certificatemodel.MockCertificateRepository {
				return certificatemodel.NewMockCertificateRepository()
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
			name: "failed - validation error missing design",
			requestBody: payload.CreateCertificatePayload{
				Name:   "Test Certificate",
				Design: "",
			},
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupMock: func() *certificatemodel.MockCertificateRepository {
				return certificatemodel.NewMockCertificateRepository()
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
			requestBody: payload.CreateCertificatePayload{
				Name:   "New Certificate",
				Design: "design.html",
			},
			setupContext: func(c *fiber.Ctx) {
				// Don't set user_id
			},
			setupMock: func() *certificatemodel.MockCertificateRepository {
				return certificatemodel.NewMockCertificateRepository()
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
			name: "failed - database error on create",
			requestBody: payload.CreateCertificatePayload{
				Name:   "New Certificate",
				Design: "design.html",
			},
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123@example.com")
			},
			setupMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.CreateFunc = func(certData payload.CreateCertificatePayload, userId string) (*model.Certificate, error) {
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
			mockCertRepo := tt.setupMock()
			mockSignatureRepo := signaturemodel.NewMockSignatureRepository()
			mockParticipantRepo := participantmodel.NewMockParticipantRepository()

			ctrl := certificate_controller.NewCertificateController(mockCertRepo, mockSignatureRepo, mockParticipantRepo)

			app.Post("/certificate", func(c *fiber.Ctx) error {
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

			req := httptest.NewRequest("POST", "/certificate", bodyReader)
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

func TestCertificateController_Delete(t *testing.T) {
	tests := []struct {
		name                    string
		certId                  string
		setupMock               func() (*certificatemodel.MockCertificateRepository, *participantmodel.MockParticipantRepository, *signaturemodel.MockSignatureRepository)
		wantStatusCode          int
		checkResponse           func(t *testing.T, body []byte)
	}{
		{
			name:   "successful delete",
			certId: "cert123",
			setupMock: func() (*certificatemodel.MockCertificateRepository, *participantmodel.MockParticipantRepository, *signaturemodel.MockSignatureRepository) {
				mockCert := certificatemodel.NewMockCertificateRepository()
				mockCert.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:     certId,
						UserID: "user123",
						Name:   "Test Certificate",
					}, nil
				}
				mockCert.DeleteFunc = func(id string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:     id,
						UserID: "user123",
						Name:   "Test Certificate",
					}, nil
				}

				mockParticipant := participantmodel.NewMockParticipantRepository()
				mockParticipant.DeleteByCertIdFunc = func(certId string) ([]*model.Participant, error) {
					return []*model.Participant{}, nil
				}

				mockSignature := signaturemodel.NewMockSignatureRepository()
				mockSignature.DeleteSignaturesByCertificateFunc = func(certId string) ([]*model.Signature, error) {
					return []*model.Signature{}, nil
				}

				return mockCert, mockParticipant, mockSignature
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
				if response["msg"] != "Certificate Deleted" {
					t.Errorf("Expected msg='Certificate Deleted', got %v", response["msg"])
				}
			},
		},
		{
			name:   "failed - empty certificate ID",
			certId: "",
			setupMock: func() (*certificatemodel.MockCertificateRepository, *participantmodel.MockParticipantRepository, *signaturemodel.MockSignatureRepository) {
				return certificatemodel.NewMockCertificateRepository(),
					   participantmodel.NewMockParticipantRepository(),
					   signaturemodel.NewMockSignatureRepository()
			},
			wantStatusCode: fiber.StatusNotFound, // Fiber returns 404 when path param is missing
			checkResponse: func(t *testing.T, body []byte) {
				// When certId is empty, Fiber returns 404 before reaching the handler
				// This is expected behavior and doesn't need to check the response format
			},
		},
		{
			name:   "failed - certificate not found in database",
			certId: "nonexistent",
			setupMock: func() (*certificatemodel.MockCertificateRepository, *participantmodel.MockParticipantRepository, *signaturemodel.MockSignatureRepository) {
				mockCert := certificatemodel.NewMockCertificateRepository()
				mockCert.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return nil, nil
				}

				return mockCert,
					   participantmodel.NewMockParticipantRepository(),
					   signaturemodel.NewMockSignatureRepository()
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
				if response["msg"] != "Certificate not found" {
					t.Errorf("Expected msg='Certificate not found', got %v", response["msg"])
				}
			},
		},
		{
			name:   "failed - error getting certificate",
			certId: "cert123",
			setupMock: func() (*certificatemodel.MockCertificateRepository, *participantmodel.MockParticipantRepository, *signaturemodel.MockSignatureRepository) {
				mockCert := certificatemodel.NewMockCertificateRepository()
				mockCert.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return nil, errors.New("database error")
				}

				return mockCert,
					   participantmodel.NewMockParticipantRepository(),
					   signaturemodel.NewMockSignatureRepository()
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
			name:   "failed - error deleting participants",
			certId: "cert123",
			setupMock: func() (*certificatemodel.MockCertificateRepository, *participantmodel.MockParticipantRepository, *signaturemodel.MockSignatureRepository) {
				mockCert := certificatemodel.NewMockCertificateRepository()
				mockCert.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:     certId,
						UserID: "user123",
						Name:   "Test Certificate",
					}, nil
				}

				mockParticipant := participantmodel.NewMockParticipantRepository()
				mockParticipant.DeleteByCertIdFunc = func(certId string) ([]*model.Participant, error) {
					return nil, errors.New("error deleting participants")
				}

				return mockCert, mockParticipant, signaturemodel.NewMockSignatureRepository()
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
			name:   "failed - error deleting signatures",
			certId: "cert123",
			setupMock: func() (*certificatemodel.MockCertificateRepository, *participantmodel.MockParticipantRepository, *signaturemodel.MockSignatureRepository) {
				mockCert := certificatemodel.NewMockCertificateRepository()
				mockCert.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:     certId,
						UserID: "user123",
						Name:   "Test Certificate",
					}, nil
				}

				mockParticipant := participantmodel.NewMockParticipantRepository()
				mockParticipant.DeleteByCertIdFunc = func(certId string) ([]*model.Participant, error) {
					return []*model.Participant{}, nil
				}

				mockSignature := signaturemodel.NewMockSignatureRepository()
				mockSignature.DeleteSignaturesByCertificateFunc = func(certId string) ([]*model.Signature, error) {
					return nil, errors.New("error deleting signatures")
				}

				return mockCert, mockParticipant, mockSignature
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
			name:   "failed - error deleting certificate",
			certId: "cert123",
			setupMock: func() (*certificatemodel.MockCertificateRepository, *participantmodel.MockParticipantRepository, *signaturemodel.MockSignatureRepository) {
				mockCert := certificatemodel.NewMockCertificateRepository()
				mockCert.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:     certId,
						UserID: "user123",
						Name:   "Test Certificate",
					}, nil
				}
				mockCert.DeleteFunc = func(id string) (*model.Certificate, error) {
					return nil, errors.New("certificate not found")
				}

				mockParticipant := participantmodel.NewMockParticipantRepository()
				mockParticipant.DeleteByCertIdFunc = func(certId string) ([]*model.Participant, error) {
					return []*model.Participant{}, nil
				}

				mockSignature := signaturemodel.NewMockSignatureRepository()
				mockSignature.DeleteSignaturesByCertificateFunc = func(certId string) ([]*model.Signature, error) {
					return []*model.Signature{}, nil
				}

				return mockCert, mockParticipant, mockSignature
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
				if response["msg"] != "Certificate not found" {
					t.Errorf("Expected msg='Certificate not found', got %v", response["msg"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			mockCertRepo, mockParticipantRepo, mockSignatureRepo := tt.setupMock()

			ctrl := certificate_controller.NewCertificateController(mockCertRepo, mockSignatureRepo, mockParticipantRepo)

			app.Delete("/certificate/:certId", ctrl.Delete)

			req := httptest.NewRequest("DELETE", "/certificate/"+tt.certId, nil)
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

func TestCertificateController_GetById(t *testing.T) {
	tests := []struct {
		name           string
		certId         string
		setupMock      func() *certificatemodel.MockCertificateRepository
		wantStatusCode int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:   "successful get by id",
			certId: "cert123",
			setupMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:     certId,
						UserID: "user123",
						Name:   "Test Certificate",
						Design: "design.html",
						CreatedAt: time.Now(),
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
				if response["msg"] != "Certificate found" {
					t.Errorf("Expected msg='Certificate found', got %v", response["msg"])
				}
				data, ok := response["data"].(map[string]any)
				if !ok {
					t.Fatal("Expected data to be a map")
				}
				if data["id"] != "cert123" {
					t.Errorf("Expected id='cert123', got %v", data["id"])
				}
			},
		},
		{
			name:   "failed - empty certificate ID",
			certId: "",
			setupMock: func() *certificatemodel.MockCertificateRepository {
				return certificatemodel.NewMockCertificateRepository()
			},
			wantStatusCode: fiber.StatusNotFound,
			checkResponse: func(t *testing.T, body []byte) {
				// Fiber returns 404 when path param is missing
			},
		},
		{
			name:   "failed - certificate not found",
			certId: "nonexistent",
			setupMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return nil, nil
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
				if response["msg"] != "Certificate not found" {
					t.Errorf("Expected msg='Certificate not found', got %v", response["msg"])
				}
			},
		},
		{
			name:   "failed - database error",
			certId: "cert123",
			setupMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.GetByIdFunc = func(certId string) (*model.Certificate, error) {
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
			mockCertRepo := tt.setupMock()
			mockSignatureRepo := signaturemodel.NewMockSignatureRepository()
			mockParticipantRepo := participantmodel.NewMockParticipantRepository()

			ctrl := certificate_controller.NewCertificateController(mockCertRepo, mockSignatureRepo, mockParticipantRepo)

			app.Get("/certificate/:certId", ctrl.GetById)

			req := httptest.NewRequest("GET", "/certificate/"+tt.certId, nil)
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

func TestCertificateController_GetAnchorList(t *testing.T) {
	validDesign := `{
		"objects": [
			{"id": "PLACEHOLDER-name", "type": "textbox"},
			{"id": "PLACEHOLDER-email", "type": "textbox"},
			{"id": "SIGNATURE-signer1", "type": "image"},
			{"id": "other-object", "type": "rect"}
		]
	}`

	tests := []struct {
		name           string
		certId         string
		setupMock      func() *certificatemodel.MockCertificateRepository
		wantStatusCode int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:   "successful get anchor list",
			certId: "cert123",
			setupMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:     certId,
						UserID: "user123",
						Name:   "Test Certificate",
						Design: validDesign,
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
				if len(data) != 2 {
					t.Errorf("Expected 2 anchors, got %d", len(data))
				}
			},
		},
		{
			name:   "failed - certificate not found",
			certId: "nonexistent",
			setupMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return nil, nil
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
			},
		},
		{
			name:   "failed - invalid design JSON",
			certId: "cert123",
			setupMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:     certId,
						Design: "invalid json",
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
			name:   "failed - invalid design format (no objects array)",
			certId: "cert123",
			setupMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:     certId,
						Design: `{"other": "data"}`,
					}, nil
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			mockCertRepo := tt.setupMock()
			mockSignatureRepo := signaturemodel.NewMockSignatureRepository()
			mockParticipantRepo := participantmodel.NewMockParticipantRepository()

			ctrl := certificate_controller.NewCertificateController(mockCertRepo, mockSignatureRepo, mockParticipantRepo)

			app.Get("/certificate/anchor/:certId", ctrl.GetAnchorList)

			req := httptest.NewRequest("GET", "/certificate/anchor/"+tt.certId, nil)
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

func TestCertificateController_CheckGenerateStatus(t *testing.T) {
	tests := []struct {
		name                string
		certId              string
		setupMock           func() (*certificatemodel.MockCertificateRepository, *signaturemodel.MockSignatureRepository, *participantmodel.MockParticipantRepository)
		wantStatusCode      int
		checkResponse       func(t *testing.T, body []byte)
	}{
		{
			name:   "certificate not signed",
			certId: "cert123",
			setupMock: func() (*certificatemodel.MockCertificateRepository, *signaturemodel.MockSignatureRepository, *participantmodel.MockParticipantRepository) {
				mockCert := certificatemodel.NewMockCertificateRepository()
				mockCert.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:            certId,
						IsSigned:      false,
						IsDistributed: false,
					}, nil
				}

				mockSig := signaturemodel.NewMockSignatureRepository()
				mockSig.AreAllSignaturesCompleteFunc = func(certificateId string) (bool, error) {
					return false, nil
				}

				return mockCert, mockSig, participantmodel.NewMockParticipantRepository()
			},
			wantStatusCode: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				data, ok := response["data"].(map[string]any)
				if !ok {
					t.Fatal("Expected data to be a map")
				}
				if data["is_signed"] != false {
					t.Errorf("Expected is_signed=false, got %v", data["is_signed"])
				}
			},
		},
		{
			name:   "certificate signed but not distributed",
			certId: "cert123",
			setupMock: func() (*certificatemodel.MockCertificateRepository, *signaturemodel.MockSignatureRepository, *participantmodel.MockParticipantRepository) {
				mockCert := certificatemodel.NewMockCertificateRepository()
				mockCert.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:            certId,
						IsSigned:      true,
						IsDistributed: false,
					}, nil
				}

				return mockCert, signaturemodel.NewMockSignatureRepository(), participantmodel.NewMockParticipantRepository()
			},
			wantStatusCode: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				data, ok := response["data"].(map[string]any)
				if !ok {
					t.Fatal("Expected data to be a map")
				}
				if data["is_signed"] != true {
					t.Errorf("Expected is_signed=true, got %v", data["is_signed"])
				}
				if data["is_generated"] != false {
					t.Errorf("Expected is_generated=false, got %v", data["is_generated"])
				}
			},
		},
		{
			name:   "certificate distributed - fully generated",
			certId: "cert123",
			setupMock: func() (*certificatemodel.MockCertificateRepository, *signaturemodel.MockSignatureRepository, *participantmodel.MockParticipantRepository) {
				mockCert := certificatemodel.NewMockCertificateRepository()
				mockCert.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:            certId,
						IsSigned:      true,
						IsDistributed: true,
					}, nil
				}

				mockParticipant := participantmodel.NewMockParticipantRepository()
				mockParticipant.GetParticipantsByCertIdFunc = func(certId string) ([]*participantmodel.CombinedParticipant, error) {
					return []*participantmodel.CombinedParticipant{
						{ID: "p1", CertificateURL: "http://example.com/cert1.pdf"},
						{ID: "p2", CertificateURL: "http://example.com/cert2.pdf"},
					}, nil
				}

				return mockCert, signaturemodel.NewMockSignatureRepository(), mockParticipant
			},
			wantStatusCode: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				data, ok := response["data"].(map[string]any)
				if !ok {
					t.Fatal("Expected data to be a map")
				}
				if data["is_partial_generated"] != false {
					t.Errorf("Expected is_partial_generated=false, got %v", data["is_partial_generated"])
				}
			},
		},
		{
			name:   "certificate distributed - partially generated",
			certId: "cert123",
			setupMock: func() (*certificatemodel.MockCertificateRepository, *signaturemodel.MockSignatureRepository, *participantmodel.MockParticipantRepository) {
				mockCert := certificatemodel.NewMockCertificateRepository()
				mockCert.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:            certId,
						IsSigned:      true,
						IsDistributed: true,
					}, nil
				}

				mockParticipant := participantmodel.NewMockParticipantRepository()
				mockParticipant.GetParticipantsByCertIdFunc = func(certId string) ([]*participantmodel.CombinedParticipant, error) {
					return []*participantmodel.CombinedParticipant{
						{ID: "p1", CertificateURL: "http://example.com/cert1.pdf"},
						{ID: "p2", CertificateURL: ""},
					}, nil
				}

				return mockCert, signaturemodel.NewMockSignatureRepository(), mockParticipant
			},
			wantStatusCode: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				data, ok := response["data"].(map[string]any)
				if !ok {
					t.Fatal("Expected data to be a map")
				}
				if data["is_partial_generated"] != true {
					t.Errorf("Expected is_partial_generated=true, got %v", data["is_partial_generated"])
				}
			},
		},
		{
			name:   "failed - certificate not found",
			certId: "nonexistent",
			setupMock: func() (*certificatemodel.MockCertificateRepository, *signaturemodel.MockSignatureRepository, *participantmodel.MockParticipantRepository) {
				mockCert := certificatemodel.NewMockCertificateRepository()
				mockCert.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return nil, nil
				}

				return mockCert, signaturemodel.NewMockSignatureRepository(), participantmodel.NewMockParticipantRepository()
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			mockCertRepo, mockSignatureRepo, mockParticipantRepo := tt.setupMock()

			ctrl := certificate_controller.NewCertificateController(mockCertRepo, mockSignatureRepo, mockParticipantRepo)

			app.Get("/certificate/generate/status/:certificateId", ctrl.CheckGenerateStatus)

			req := httptest.NewRequest("GET", "/certificate/generate/status/"+tt.certId, nil)
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

func TestCertificateController_Update(t *testing.T) {
	tests := []struct {
		name           string
		certId         string
		requestBody    any
		setupContext   func(c *fiber.Ctx)
		setupMock      func() (*certificatemodel.MockCertificateRepository, *signaturemodel.MockSignatureRepository, *participantmodel.MockParticipantRepository)
		wantStatusCode int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:   "successful update - name only",
			certId: "cert123",
			requestBody: payload.UpdateCertificatePayload{
				Name: "Updated Name",
			},
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123")
			},
			setupMock: func() (*certificatemodel.MockCertificateRepository, *signaturemodel.MockSignatureRepository, *participantmodel.MockParticipantRepository) {
				mockCert := certificatemodel.NewMockCertificateRepository()
				mockCert.UpdateFunc = func(id string, name string, design string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:     id,
						Name:   name,
						Design: "existing design",
					}, nil
				}

				return mockCert, signaturemodel.NewMockSignatureRepository(), participantmodel.NewMockParticipantRepository()
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
			},
		},
		{
			name:   "failed - empty certificate ID",
			certId: "",
			requestBody: payload.UpdateCertificatePayload{
				Name: "Test",
			},
			setupContext: func(c *fiber.Ctx) {},
			setupMock: func() (*certificatemodel.MockCertificateRepository, *signaturemodel.MockSignatureRepository, *participantmodel.MockParticipantRepository) {
				return certificatemodel.NewMockCertificateRepository(), signaturemodel.NewMockSignatureRepository(), participantmodel.NewMockParticipantRepository()
			},
			wantStatusCode: fiber.StatusNotFound,
			checkResponse: func(t *testing.T, body []byte) {},
		},
		{
			name:   "failed - no fields provided",
			certId: "cert123",
			requestBody: payload.UpdateCertificatePayload{
				Name:   "",
				Design: "",
			},
			setupContext: func(c *fiber.Ctx) {},
			setupMock: func() (*certificatemodel.MockCertificateRepository, *signaturemodel.MockSignatureRepository, *participantmodel.MockParticipantRepository) {
				return certificatemodel.NewMockCertificateRepository(), signaturemodel.NewMockSignatureRepository(), participantmodel.NewMockParticipantRepository()
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
			name:   "failed - certificate not found",
			certId: "nonexistent",
			requestBody: payload.UpdateCertificatePayload{
				Name: "Updated Name",
			},
			setupContext: func(c *fiber.Ctx) {},
			setupMock: func() (*certificatemodel.MockCertificateRepository, *signaturemodel.MockSignatureRepository, *participantmodel.MockParticipantRepository) {
				mockCert := certificatemodel.NewMockCertificateRepository()
				mockCert.UpdateFunc = func(id string, name string, design string) (*model.Certificate, error) {
					return nil, errors.New("certificate not found")
				}

				return mockCert, signaturemodel.NewMockSignatureRepository(), participantmodel.NewMockParticipantRepository()
			},
			wantStatusCode: fiber.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["msg"] != "Certificate not found" {
					t.Errorf("Expected msg='Certificate not found', got %v", response["msg"])
				}
			},
		},
		{
			name:   "failed - invalid request body",
			certId: "cert123",
			requestBody: "invalid json",
			setupContext: func(c *fiber.Ctx) {},
			setupMock: func() (*certificatemodel.MockCertificateRepository, *signaturemodel.MockSignatureRepository, *participantmodel.MockParticipantRepository) {
				return certificatemodel.NewMockCertificateRepository(), signaturemodel.NewMockSignatureRepository(), participantmodel.NewMockParticipantRepository()
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
			mockCertRepo, mockSignatureRepo, mockParticipantRepo := tt.setupMock()

			ctrl := certificate_controller.NewCertificateController(mockCertRepo, mockSignatureRepo, mockParticipantRepo)

			app.Put("/certificate/:id", func(c *fiber.Ctx) error {
				if tt.setupContext != nil {
					tt.setupContext(c)
				}
				return ctrl.Update(c)
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

			req := httptest.NewRequest("PUT", "/certificate/"+tt.certId, bodyReader)
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req, -1)
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

func TestCertificateController_DistributeByMail(t *testing.T) {
	tests := []struct {
		name           string
		certId         string
		emailField     string
		setupMock      func() (*certificatemodel.MockCertificateRepository, *participantmodel.MockParticipantRepository)
		wantStatusCode int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:       "failed - missing email field parameter",
			certId:     "cert123",
			emailField: "",
			setupMock: func() (*certificatemodel.MockCertificateRepository, *participantmodel.MockParticipantRepository) {
				return certificatemodel.NewMockCertificateRepository(), participantmodel.NewMockParticipantRepository()
			},
			wantStatusCode: fiber.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["msg"] != "Missing email field" {
					t.Errorf("Expected msg='Missing email field', got %v", response["msg"])
				}
			},
		},
		{
			name:       "failed - certificate not found",
			certId:     "nonexistent",
			emailField: "email",
			setupMock: func() (*certificatemodel.MockCertificateRepository, *participantmodel.MockParticipantRepository) {
				mockCert := certificatemodel.NewMockCertificateRepository()
				mockCert.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return nil, nil
				}
				return mockCert, participantmodel.NewMockParticipantRepository()
			},
			wantStatusCode: fiber.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["msg"] != "Certificate not exist" {
					t.Errorf("Expected msg='Certificate not exist', got %v", response["msg"])
				}
			},
		},
		{
			name:       "failed - database error getting certificate",
			certId:     "cert123",
			emailField: "email",
			setupMock: func() (*certificatemodel.MockCertificateRepository, *participantmodel.MockParticipantRepository) {
				mockCert := certificatemodel.NewMockCertificateRepository()
				mockCert.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return nil, errors.New("database error")
				}
				return mockCert, participantmodel.NewMockParticipantRepository()
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
			mockCertRepo, mockParticipantRepo := tt.setupMock()
			mockSignatureRepo := signaturemodel.NewMockSignatureRepository()

			ctrl := certificate_controller.NewCertificateController(mockCertRepo, mockSignatureRepo, mockParticipantRepo)

			app.Get("/certificate/mail/:certId", ctrl.DistributeByMail)

			url := "/certificate/mail/" + tt.certId
			if tt.emailField != "" {
				url += "?email=" + tt.emailField
			}

			req := httptest.NewRequest("GET", url, nil)
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

func TestCertificateController_DownloadArchive(t *testing.T) {
	tests := []struct {
		name           string
		certId         string
		setupMock      func() *certificatemodel.MockCertificateRepository
		wantStatusCode int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:   "failed - empty certificate ID",
			certId: "",
			setupMock: func() *certificatemodel.MockCertificateRepository {
				return certificatemodel.NewMockCertificateRepository()
			},
			wantStatusCode: fiber.StatusNotFound,
			checkResponse:  func(t *testing.T, body []byte) {},
		},
		{
			name:   "failed - certificate not found",
			certId: "nonexistent",
			setupMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return nil, nil
				}
				return mock
			},
			wantStatusCode: fiber.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["msg"] != "Certificate not found" {
					t.Errorf("Expected msg='Certificate not found', got %v", response["msg"])
				}
			},
		},
		{
			name:   "failed - no archive URL",
			certId: "cert123",
			setupMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.GetByIdFunc = func(certId string) (*model.Certificate, error) {
					return &model.Certificate{
						ID:         certId,
						ArchiveURL: "",
					}, nil
				}
				return mock
			},
			wantStatusCode: fiber.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["msg"] != "Certificate archive not available" {
					t.Errorf("Expected msg='Certificate archive not available', got %v", response["msg"])
				}
			},
		},
		{
			name:   "failed - database error",
			certId: "cert123",
			setupMock: func() *certificatemodel.MockCertificateRepository {
				mock := certificatemodel.NewMockCertificateRepository()
				mock.GetByIdFunc = func(certId string) (*model.Certificate, error) {
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
			mockCertRepo := tt.setupMock()
			mockSignatureRepo := signaturemodel.NewMockSignatureRepository()
			mockParticipantRepo := participantmodel.NewMockParticipantRepository()

			ctrl := certificate_controller.NewCertificateController(mockCertRepo, mockSignatureRepo, mockParticipantRepo)

			app.Get("/certificate/archive/:certId", ctrl.DownloadArchive)

			req := httptest.NewRequest("GET", "/certificate/archive/"+tt.certId, nil)
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
