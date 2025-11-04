package util

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEncryptDecrypt tests basic encryption and decryption flow
func TestEncryptDecrypt(t *testing.T) {
	// Generate a valid 32-byte (64 hex chars) key
	validKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	testData := []byte("Hello, World! This is sensitive data.")

	// Encrypt
	encrypted, err := EncryptData(testData, validKey)
	require.NoError(t, err, "Encryption should not fail with valid key")
	assert.NotEmpty(t, encrypted, "Encrypted data should not be empty")

	// Decrypt
	decrypted, err := DecryptData(encrypted, validKey)
	require.NoError(t, err, "Decryption should not fail with valid key")
	assert.Equal(t, testData, decrypted, "Decrypted data should match original")
}

// TestEncryptData_EmptyData tests encryption of empty data
func TestEncryptData_EmptyData(t *testing.T) {
	validKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	emptyData := []byte("")

	encrypted, err := EncryptData(emptyData, validKey)
	require.NoError(t, err, "Should encrypt empty data")
	assert.NotEmpty(t, encrypted, "Encrypted empty data should still produce output")

	decrypted, err := DecryptData(encrypted, validKey)
	require.NoError(t, err, "Should decrypt empty data")
	assert.Empty(t, decrypted, "Decrypted empty data should be empty")
}

// TestEncryptData_LargeData tests encryption of large data
func TestEncryptData_LargeData(t *testing.T) {
	validKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	// Create 1MB of test data
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	encrypted, err := EncryptData(largeData, validKey)
	require.NoError(t, err, "Should encrypt large data")
	assert.NotEmpty(t, encrypted, "Encrypted large data should not be empty")

	decrypted, err := DecryptData(encrypted, validKey)
	require.NoError(t, err, "Should decrypt large data")
	assert.Equal(t, largeData, decrypted, "Decrypted large data should match")
}

// TestEncryptData_SpecialCharacters tests encryption with special characters
func TestEncryptData_SpecialCharacters(t *testing.T) {
	validKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	testCases := []string{
		"Special chars: !@#$%^&*()",
		"Unicode: 你好世界 🌍🚀",
		"JSON: {\"key\":\"value\",\"number\":123}",
		"Newlines:\nLine1\nLine2\nLine3",
		"Tabs:\tTab1\tTab2",
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Case%d", i), func(t *testing.T) {
			data := []byte(tc)

			encrypted, err := EncryptData(data, validKey)
			require.NoError(t, err, "Should encrypt special characters")

			decrypted, err := DecryptData(encrypted, validKey)
			require.NoError(t, err, "Should decrypt special characters")
			assert.Equal(t, data, decrypted, "Decrypted data should match original")
		})
	}
}

// TestEncryptData_InvalidKey tests encryption with invalid keys
func TestEncryptData_InvalidKey(t *testing.T) {
	testData := []byte("test data")

	testCases := []struct {
		name        string
		key         string
		expectedErr string
	}{
		{
			name:        "Short key (16 bytes)",
			key:         "0123456789abcdef0123456789abcdef",
			expectedErr: "encryption key must be 32 bytes",
		},
		{
			name:        "Long key (48 bytes)",
			key:         "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expectedErr: "encryption key must be 32 bytes",
		},
		{
			name:        "Non-hex key",
			key:         "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			expectedErr: "invalid encryption key format",
		},
		{
			name:        "Empty key",
			key:         "",
			expectedErr: "encryption key must be 32 bytes",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := EncryptData(testData, tc.key)
			assert.Error(t, err, "Should fail with invalid key")
			assert.Empty(t, encrypted, "Should not return encrypted data on error")
			assert.Contains(t, err.Error(), tc.expectedErr, "Error message should be correct")
		})
	}
}

// TestDecryptData_InvalidKey tests decryption with invalid keys
func TestDecryptData_InvalidKey(t *testing.T) {
	validKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	testData := []byte("test data")

	encrypted, err := EncryptData(testData, validKey)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		key         string
		expectedErr string
	}{
		{
			name:        "Wrong key (different valid key)",
			key:         "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedErr: "decryption failed: data may be tampered",
		},
		{
			name:        "Invalid key format",
			key:         "invalid-key",
			expectedErr: "invalid encryption key format",
		},
		{
			name:        "Short key",
			key:         "0123456789abcdef",
			expectedErr: "encryption key must be 32 bytes",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			decrypted, err := DecryptData(encrypted, tc.key)
			assert.Error(t, err, "Should fail with invalid key")
			assert.Nil(t, decrypted, "Should not return decrypted data on error")
			assert.Contains(t, err.Error(), tc.expectedErr, "Error message should be correct")
		})
	}
}

// TestDecryptData_InvalidCiphertext tests decryption with invalid ciphertext
func TestDecryptData_InvalidCiphertext(t *testing.T) {
	validKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	testCases := []struct {
		name        string
		ciphertext  string
		expectedErr string
	}{
		{
			name:        "Invalid base64",
			ciphertext:  "not-valid-base64!!!",
			expectedErr: "illegal base64 data",
		},
		{
			name:        "Empty ciphertext",
			ciphertext:  "",
			expectedErr: "ciphertext too short",
		},
		{
			name:        "Too short ciphertext (valid base64 but too short)",
			ciphertext:  "YWJj", // "abc" in base64 (3 bytes)
			expectedErr: "ciphertext too short",
		},
		{
			name:        "Tampered ciphertext",
			ciphertext:  "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo=", // Valid base64 but tampered data
			expectedErr: "decryption failed: data may be tampered",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			decrypted, err := DecryptData(tc.ciphertext, validKey)
			assert.Error(t, err, "Should fail with invalid ciphertext")
			assert.Nil(t, decrypted, "Should not return data on error")
			if tc.expectedErr != "" {
				assert.Contains(t, err.Error(), tc.expectedErr, "Error message should be descriptive")
			}
		})
	}
}

// TestEncryptData_Randomness tests that encryption produces different output each time
func TestEncryptData_Randomness(t *testing.T) {
	validKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	testData := []byte("test data")

	encrypted1, err1 := EncryptData(testData, validKey)
	require.NoError(t, err1)

	encrypted2, err2 := EncryptData(testData, validKey)
	require.NoError(t, err2)

	// Same data encrypted twice should produce different ciphertext (due to random nonce)
	assert.NotEqual(t, encrypted1, encrypted2, "Encryption should use random nonce")

	// But both should decrypt to the same original data
	decrypted1, err := DecryptData(encrypted1, validKey)
	require.NoError(t, err)
	assert.Equal(t, testData, decrypted1)

	decrypted2, err := DecryptData(encrypted2, validKey)
	require.NoError(t, err)
	assert.Equal(t, testData, decrypted2)
}

// TestEncryptData_KeyFormat tests that only hex format is accepted
func TestEncryptData_KeyFormat(t *testing.T) {
	testData := []byte("test")

	// 64 hex chars (32 bytes when decoded) but not in hex format should fail
	invalidKey := "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
	assert.Equal(t, 64, len(invalidKey), "Key should be 64 hex chars")

	encrypted, err := EncryptData(testData, invalidKey)
	assert.Error(t, err, "Non-hex key should fail")
	assert.Empty(t, encrypted)
	assert.Contains(t, err.Error(), "invalid encryption key format")
}

// TestDecryptData_DataIntegrity tests GCM authentication tag verification
func TestDecryptData_DataIntegrity(t *testing.T) {
	validKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	testData := []byte("important data")

	encrypted, err := EncryptData(testData, validKey)
	require.NoError(t, err)

	// Tamper with the encrypted data by modifying one character
	tamperedEncrypted := encrypted[:len(encrypted)-5] + "X" + encrypted[len(encrypted)-4:]

	// Decryption should fail due to authentication tag mismatch
	decrypted, err := DecryptData(tamperedEncrypted, validKey)
	assert.Error(t, err, "Tampered data should fail authentication")
	assert.Nil(t, decrypted)
	assert.Contains(t, err.Error(), "decryption failed: data may be tampered")
}

// Benchmark tests
func BenchmarkEncryptData(b *testing.B) {
	validKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	testData := []byte("This is some test data for benchmarking encryption performance.")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncryptData(testData, validKey)
	}
}

func BenchmarkDecryptData(b *testing.B) {
	validKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	testData := []byte("This is some test data for benchmarking decryption performance.")
	encrypted, _ := EncryptData(testData, validKey)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecryptData(encrypted, validKey)
	}
}

// TestEncryptDecrypt_RealWorldScenario simulates real-world usage
func TestEncryptDecrypt_RealWorldScenario(t *testing.T) {
	// Simulate real encryption key (generated from crypto/rand)
	keyBytes := make([]byte, 32)
	for i := range keyBytes {
		keyBytes[i] = byte(i * 7 % 256)
	}
	key := hex.EncodeToString(keyBytes)

	// Simulate sensitive user data
	sensitiveData := []byte(`{
		"userId": "12345",
		"email": "user@example.com",
		"ssn": "123-45-6789",
		"creditCard": "4111-1111-1111-1111"
	}`)

	// Encrypt
	encrypted, err := EncryptData(sensitiveData, key)
	require.NoError(t, err, "Real-world encryption should succeed")

	// Store encrypted data (simulated)
	storedData := encrypted

	// Retrieve and decrypt
	decrypted, err := DecryptData(storedData, key)
	require.NoError(t, err, "Real-world decryption should succeed")
	assert.Equal(t, sensitiveData, decrypted, "Real-world round-trip should preserve data")
}
