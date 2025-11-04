package util

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDecodeJWTToken_ValidToken tests decoding a valid JWT token
func TestDecodeJWTToken_ValidToken(t *testing.T) {
	// Create a valid JWT token manually
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	payload := map[string]interface{}{
		"sub":                "1234567890",
		"name":               "John Doe",
		"email":              "john@example.com",
		"preferred_username": "johndoe",
	}
	payloadJSON, _ := json.Marshal(payload)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// JWT without signature (we're only testing payload decoding)
	signature := base64.RawURLEncoding.EncodeToString([]byte("fake-signature"))
	token := header + "." + payloadEncoded + "." + signature

	// Test: Decode the token
	jwtPayload, err := DecodeJWTToken(token)
	require.NoError(t, err, "Should decode valid JWT token")
	require.NotNil(t, jwtPayload, "Payload should not be nil")

	// Verify: Check decoded fields
	assert.Equal(t, "john@example.com", jwtPayload.Email)
	assert.Equal(t, "johndoe", jwtPayload.PreferredUsername)
}

// TestDecodeJWTToken_InvalidFormat tests decoding with invalid format
func TestDecodeJWTToken_InvalidFormat(t *testing.T) {
	testCases := []struct {
		name  string
		token string
	}{
		{"Empty token", ""},
		{"Single part", "onlyonepart"},
		{"Two parts", "header.payload"},
		{"Four parts", "header.payload.signature.extra"},
		{"No dots", "headerpayloadsignature"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jwtPayload, err := DecodeJWTToken(tc.token)
			assert.Error(t, err, "Should fail with invalid format")
			assert.Nil(t, jwtPayload, "Payload should be nil")
			assert.Contains(t, err.Error(), "invalid JWT token format")
		})
	}
}

// TestDecodeJWTToken_InvalidBase64 tests decoding with invalid base64 payload
func TestDecodeJWTToken_InvalidBase64(t *testing.T) {
	// Create token with invalid base64 in payload
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
	invalidPayload := "!!!invalid-base64!!!"
	signature := base64.RawURLEncoding.EncodeToString([]byte("sig"))
	token := header + "." + invalidPayload + "." + signature

	jwtPayload, err := DecodeJWTToken(token)
	assert.Error(t, err, "Should fail with invalid base64")
	assert.Nil(t, jwtPayload)
	assert.Contains(t, err.Error(), "failed to decode JWT payload")
}

// TestDecodeJWTToken_InvalidJSON tests decoding with invalid JSON payload
func TestDecodeJWTToken_InvalidJSON(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))

	// Valid base64 but invalid JSON
	invalidJSON := base64.RawURLEncoding.EncodeToString([]byte(`{invalid json}`))
	signature := base64.RawURLEncoding.EncodeToString([]byte("sig"))
	token := header + "." + invalidJSON + "." + signature

	jwtPayload, err := DecodeJWTToken(token)
	assert.Error(t, err, "Should fail with invalid JSON")
	assert.Nil(t, jwtPayload)
	assert.Contains(t, err.Error(), "failed to unmarshal JWT payload")
}

// TestDecodeJWTToken_EmptyPayload tests decoding with empty payload
func TestDecodeJWTToken_EmptyPayload(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
	emptyPayload := base64.RawURLEncoding.EncodeToString([]byte(`{}`))
	signature := base64.RawURLEncoding.EncodeToString([]byte("sig"))
	token := header + "." + emptyPayload + "." + signature

	jwtPayload, err := DecodeJWTToken(token)
	require.NoError(t, err, "Should decode empty payload")
	require.NotNil(t, jwtPayload)

	// Empty payload should have empty/zero values
	assert.Empty(t, jwtPayload.Email)
	assert.Empty(t, jwtPayload.PreferredUsername)
}

// TestDecodeJWTToken_AllFields tests decoding with all possible fields
func TestDecodeJWTToken_AllFields(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))

	payload := map[string]interface{}{
		"sub":                "user-123",
		"email":              "test@example.com",
		"preferred_username": "testuser",
		"name":               "Test User",
		"given_name":         "Test",
		"family_name":        "User",
		"exp":                1234567890,
		"iat":                1234567800,
	}
	payloadJSON, _ := json.Marshal(payload)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signature := base64.RawURLEncoding.EncodeToString([]byte("sig"))
	token := header + "." + payloadEncoded + "." + signature

	jwtPayload, err := DecodeJWTToken(token)
	require.NoError(t, err)
	require.NotNil(t, jwtPayload)

	assert.Equal(t, "test@example.com", jwtPayload.Email)
	assert.Equal(t, "testuser", jwtPayload.PreferredUsername)
}

// TestDecodeJWTToken_Base64Padding tests that padding is correctly added
func TestDecodeJWTToken_Base64Padding(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))

	// Create payload that needs padding
	payload := map[string]interface{}{
		"email": "a@b.c", // Short payload that may need padding
	}
	payloadJSON, _ := json.Marshal(payload)

	// Encode without padding (URL encoding doesn't use padding by default)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signature := base64.RawURLEncoding.EncodeToString([]byte("sig"))
	token := header + "." + payloadEncoded + "." + signature

	jwtPayload, err := DecodeJWTToken(token)
	require.NoError(t, err, "Should handle base64 padding correctly")
	require.NotNil(t, jwtPayload)
	assert.Equal(t, "a@b.c", jwtPayload.Email)
}

// TestDecodeJWTToken_SpecialCharacters tests decoding with special characters
func TestDecodeJWTToken_SpecialCharacters(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))

	payload := map[string]interface{}{
		"email":              "user+tag@example.com",
		"preferred_username": "user-name_123",
		"name":               "José García-Martínez",
	}
	payloadJSON, _ := json.Marshal(payload)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signature := base64.RawURLEncoding.EncodeToString([]byte("sig"))
	token := header + "." + payloadEncoded + "." + signature

	jwtPayload, err := DecodeJWTToken(token)
	require.NoError(t, err)
	require.NotNil(t, jwtPayload)

	assert.Equal(t, "user+tag@example.com", jwtPayload.Email)
	assert.Equal(t, "user-name_123", jwtPayload.PreferredUsername)
}

// TestDecodeJWTToken_UnicodeContent tests decoding with Unicode content
func TestDecodeJWTToken_UnicodeContent(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))

	payload := map[string]interface{}{
		"email": "test@例え.jp",
		"name":  "山田太郎",
	}
	payloadJSON, _ := json.Marshal(payload)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signature := base64.RawURLEncoding.EncodeToString([]byte("sig"))
	token := header + "." + payloadEncoded + "." + signature

	jwtPayload, err := DecodeJWTToken(token)
	require.NoError(t, err)
	require.NotNil(t, jwtPayload)

	assert.Equal(t, "test@例え.jp", jwtPayload.Email)
}

// TestDecodeJWTToken_LargePayload tests decoding with large payload
func TestDecodeJWTToken_LargePayload(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))

	// Create large payload with many claims
	payload := make(map[string]interface{})
	payload["email"] = "test@example.com"
	payload["preferred_username"] = "testuser"

	// Add many custom claims
	for i := 0; i < 100; i++ {
		payload[string(rune('a'+i%26))+string(rune('0'+i/26))] = "value" + string(rune(i))
	}

	payloadJSON, _ := json.Marshal(payload)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signature := base64.RawURLEncoding.EncodeToString([]byte("sig"))
	token := header + "." + payloadEncoded + "." + signature

	jwtPayload, err := DecodeJWTToken(token)
	require.NoError(t, err, "Should handle large payload")
	require.NotNil(t, jwtPayload)
	assert.Equal(t, "test@example.com", jwtPayload.Email)
}

// TestDecodeJWTToken_RealWorldKeycloakToken tests with a realistic Keycloak-like token structure
func TestDecodeJWTToken_RealWorldKeycloakToken(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT","kid":"key-id"}`))

	// Realistic Keycloak payload
	payload := map[string]interface{}{
		"exp":                1234567890,
		"iat":                1234567800,
		"jti":                "unique-token-id",
		"iss":                "http://localhost:8080/realms/myrealm",
		"sub":                "f:unique-id:username",
		"typ":                "Bearer",
		"azp":                "my-client",
		"session_state":      "session-uuid",
		"acr":                "1",
		"scope":              "openid email profile",
		"email_verified":     true,
		"name":               "John Doe",
		"preferred_username": "johndoe",
		"given_name":         "John",
		"family_name":        "Doe",
		"email":              "john.doe@example.com",
	}

	payloadJSON, _ := json.Marshal(payload)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signature := base64.RawURLEncoding.EncodeToString([]byte("signature-bytes"))
	token := header + "." + payloadEncoded + "." + signature

	jwtPayload, err := DecodeJWTToken(token)
	require.NoError(t, err, "Should decode realistic Keycloak token")
	require.NotNil(t, jwtPayload)

	assert.Equal(t, "john.doe@example.com", jwtPayload.Email)
	assert.Equal(t, "johndoe", jwtPayload.PreferredUsername)
}

// Benchmark for DecodeJWTToken
func BenchmarkDecodeJWTToken(b *testing.B) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
	payload := map[string]interface{}{
		"email":              "bench@example.com",
		"preferred_username": "benchuser",
	}
	payloadJSON, _ := json.Marshal(payload)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signature := base64.RawURLEncoding.EncodeToString([]byte("sig"))
	token := header + "." + payloadEncoded + "." + signature

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeJWTToken(token)
	}
}

// TestDecodeJWTToken_Concurrency tests thread safety
func TestDecodeJWTToken_Concurrency(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
	payload := map[string]interface{}{
		"email":              "concurrent@example.com",
		"preferred_username": "concurrentuser",
	}
	payloadJSON, _ := json.Marshal(payload)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signature := base64.RawURLEncoding.EncodeToString([]byte("sig"))
	token := header + "." + payloadEncoded + "." + signature

	iterations := 100
	done := make(chan bool, iterations)

	for i := 0; i < iterations; i++ {
		go func() {
			jwtPayload, err := DecodeJWTToken(token)
			assert.NoError(t, err)
			assert.NotNil(t, jwtPayload)
			assert.Equal(t, "concurrent@example.com", jwtPayload.Email)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < iterations; i++ {
		<-done
	}
}
