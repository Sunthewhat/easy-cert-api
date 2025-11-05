package auth_controller_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	auth_controller "github.com/sunthewhat/easy-cert-api/api/controllers/auth"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/shared"
)

func TestAuthController_Login(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    any
		setupMock      func() *util.MockSSOService
		wantStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful login",
			requestBody: payload.LoginPayload{
				Username: "testuser",
				Password: "testpass123",
			},
			setupMock: func() *util.MockSSOService {
				mock := &util.MockSSOService{
					LoginFunc: func(username, password string) (*shared.SsoTokenType, error) {
						return &shared.SsoTokenType{
							AccessToken:  "mock-access-token",
							RefreshToken: "mock-refresh-token",
							TokenType:    "Bearer",
							ExpiresIn:    3600,
						}, nil
					},
					DecodeFunc: func(token string) (*shared.SsoJwtPayload, error) {
						return &shared.SsoJwtPayload{
							PreferredUsername: "testuser",
							GivenName:         "Test",
							FamilyName:        "User",
							Email:             "test@example.com",
						}, nil
					},
				}
				return mock
			},
			wantStatusCode: fiber.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response["success"] != true {
					t.Errorf("Expected success=true, got %v", response["success"])
				}
				if response["msg"] != "Login Successfull" {
					t.Errorf("Expected msg='Login Successfull', got %v", response["msg"])
				}
				data, ok := response["data"].(map[string]any)
				if !ok {
					t.Fatal("Expected data to be a map")
				}
				if data["token"] != "mock-refresh-token" {
					t.Errorf("Expected token='mock-refresh-token', got %v", data["token"])
				}
				if data["username"] != "testuser" {
					t.Errorf("Expected username='testuser', got %v", data["username"])
				}
				if data["firstname"] != "Test" {
					t.Errorf("Expected firstname='Test', got %v", data["firstname"])
				}
				if data["lastname"] != "User" {
					t.Errorf("Expected lastname='User', got %v", data["lastname"])
				}
			},
		},
		{
			name:        "invalid request body - malformed JSON",
			requestBody: "invalid json",
			setupMock: func() *util.MockSSOService {
				return &util.MockSSOService{}
			},
			wantStatusCode: fiber.StatusInternalServerError,
			wantErr:        false,
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
			name: "validation error - missing username",
			requestBody: payload.LoginPayload{
				Username: "",
				Password: "testpass123",
			},
			setupMock: func() *util.MockSSOService {
				return &util.MockSSOService{}
			},
			wantStatusCode: fiber.StatusBadRequest,
			wantErr:        false,
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
			name: "validation error - missing password",
			requestBody: payload.LoginPayload{
				Username: "testuser",
				Password: "",
			},
			setupMock: func() *util.MockSSOService {
				return &util.MockSSOService{}
			},
			wantStatusCode: fiber.StatusBadRequest,
			wantErr:        false,
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
			name: "SSO login fails - invalid credentials",
			requestBody: payload.LoginPayload{
				Username: "testuser",
				Password: "wrongpassword",
			},
			setupMock: func() *util.MockSSOService {
				mock := &util.MockSSOService{
					LoginFunc: func(username, password string) (*shared.SsoTokenType, error) {
						return nil, errors.New("invalid credentials")
					},
				}
				return mock
			},
			wantStatusCode: fiber.StatusInternalServerError,
			wantErr:        false,
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
			name: "JWT decode fails",
			requestBody: payload.LoginPayload{
				Username: "testuser",
				Password: "testpass123",
			},
			setupMock: func() *util.MockSSOService {
				mock := &util.MockSSOService{
					LoginFunc: func(username, password string) (*shared.SsoTokenType, error) {
						return &shared.SsoTokenType{
							AccessToken:  "mock-access-token",
							RefreshToken: "mock-refresh-token",
							TokenType:    "Bearer",
							ExpiresIn:    3600,
						}, nil
					},
					DecodeFunc: func(token string) (*shared.SsoJwtPayload, error) {
						return nil, errors.New("failed to decode JWT")
					},
				}
				return mock
			},
			wantStatusCode: fiber.StatusInternalServerError,
			wantErr:        false,
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
			// Setup
			app := fiber.New()
			mockSSO := tt.setupMock()
			ac := auth_controller.NewAuthController(mockSSO)

			// Create request body
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

			// Create request
			req := httptest.NewRequest("POST", "/login", bodyReader)
			req.Header.Set("Content-Type", "application/json")

			// Setup route
			app.Post("/login", ac.Login)

			// Execute request
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			// Check status code
			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.wantStatusCode, resp.StatusCode)
			}

			// Read response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			// Run custom response checks
			if tt.checkResponse != nil {
				tt.checkResponse(t, body)
			}
		})
	}
}
