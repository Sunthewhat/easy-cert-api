package auth_controller_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	auth_controller "github.com/sunthewhat/easy-cert-api/api/controllers/auth"
	"github.com/sunthewhat/easy-cert-api/common/util"
)

func TestAuthController_Verify(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(c *fiber.Ctx) // Function to set up context (e.g., add user_id to Locals)
		wantStatusCode int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful verify - user in context",
			setupContext: func(c *fiber.Ctx) {
				// Simulate middleware setting user_id in context
				c.Locals("user_id", "test@example.com")
			},
			wantStatusCode: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				// Check success flag
				if response["success"] != true {
					t.Errorf("Expected success=true, got %v", response["success"])
				}

				// Check message
				if response["msg"] != "Token is valid" {
					t.Errorf("Expected msg='Token is valid', got %v", response["msg"])
				}

				// Check data contains userId
				data, ok := response["data"].(map[string]any)
				if !ok {
					t.Fatal("Expected data to be a map")
				}
				if data["userId"] != "test@example.com" {
					t.Errorf("Expected userId='test@example.com', got %v", data["userId"])
				}
			},
		},
		{
			name: "failed verify - no user in context",
			setupContext: func(c *fiber.Ctx) {
				// Don't set user_id in context to simulate middleware failure
			},
			wantStatusCode: fiber.StatusUnauthorized,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				// Check failure flag
				if response["success"] != false {
					t.Errorf("Expected success=false, got %v", response["success"])
				}

				// Check unauthorized message
				if response["msg"] != "Failed to read user from context" {
					t.Errorf("Expected msg='Failed to read user from context', got %v", response["msg"])
				}
			},
		},
		{
			name: "failed verify - invalid user type in context",
			setupContext: func(c *fiber.Ctx) {
				// Set user_id with wrong type (not string)
				c.Locals("user_id", 12345)
			},
			wantStatusCode: fiber.StatusUnauthorized,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				// Check failure flag
				if response["success"] != false {
					t.Errorf("Expected success=false, got %v", response["success"])
				}

				// Check unauthorized message
				if response["msg"] != "Failed to read user from context" {
					t.Errorf("Expected msg='Failed to read user from context', got %v", response["msg"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			app := fiber.New()
			mockSSO := util.NewMockSSOService()
			ac := auth_controller.NewAuthController(mockSSO)

			// Setup route with context setup middleware
			app.Get("/verify", func(c *fiber.Ctx) error {
				if tt.setupContext != nil {
					tt.setupContext(c)
				}
				return ac.Verify(c)
			})

			// Create request
			req := httptest.NewRequest("GET", "/verify", nil)

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
