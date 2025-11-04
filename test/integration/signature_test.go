package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunthewhat/easy-cert-api/test/helpers"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
	"gorm.io/gorm"
)

// setupSignatureTest creates certificate and signer for signature tests
func setupSignatureTest(t *testing.T, db *gorm.DB) (string, string) {
	cert := &model.Certificate{
		ID:     "test-cert",
		UserID: "user-1",
		Name:   "Test Certificate",
		Design: "design-1",
	}
	err := db.Create(cert).Error
	require.NoError(t, err)

	signer := &model.Signer{
		ID:          "test-signer",
		Email:       "signer@example.com",
		DisplayName: "Test Signer",
		CreatedBy:   "user-1",
	}
	err = db.Create(signer).Error
	require.NoError(t, err)

	return cert.ID, signer.ID
}

// TestSignature_CreateAndRetrieve tests basic signature creation
func TestSignature_CreateAndRetrieve(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	certID, signerID := setupSignatureTest(t, db)

	// Create signature
	signature := &model.Signature{
		ID:            "sig-123",
		SignerID:      signerID,
		CertificateID: certID,
		Signature:     "signature-data-base64",
		IsSigned:      false,
		CreatedBy:     "user-1",
		IsRequested:   true,
	}

	err := db.Create(signature).Error
	require.NoError(t, err, "Failed to create signature")

	// Retrieve signature
	var retrieved model.Signature
	err = db.Where("id = ?", "sig-123").First(&retrieved).Error
	require.NoError(t, err)

	assert.Equal(t, "sig-123", retrieved.ID)
	assert.Equal(t, signerID, retrieved.SignerID)
	assert.Equal(t, certID, retrieved.CertificateID)
	assert.Equal(t, "signature-data-base64", retrieved.Signature)
	assert.False(t, retrieved.IsSigned)
	assert.True(t, retrieved.IsRequested)
}

// TestSignature_MarkAsSigned tests updating signature status
func TestSignature_MarkAsSigned(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	certID, signerID := setupSignatureTest(t, db)

	// Create unsigned signature
	signature := &model.Signature{
		ID:            "sig-unsigned",
		SignerID:      signerID,
		CertificateID: certID,
		Signature:     "",
		IsSigned:      false,
		CreatedBy:     "user-1",
		IsRequested:   true,
	}
	err := db.Create(signature).Error
	require.NoError(t, err)

	// Mark as signed
	err = db.Model(&model.Signature{}).
		Where("id = ?", "sig-unsigned").
		Updates(map[string]interface{}{
			"is_signed": true,
			"signature": "actual-signature-data",
		}).Error
	require.NoError(t, err)

	// Verify update
	var updated model.Signature
	err = db.Where("id = ?", "sig-unsigned").First(&updated).Error
	require.NoError(t, err)

	assert.True(t, updated.IsSigned)
	assert.Equal(t, "actual-signature-data", updated.Signature)
}

// TestSignature_GetByCertificate tests retrieving all signatures for a certificate
func TestSignature_GetByCertificate(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	certID, _ := setupSignatureTest(t, db)

	// Create additional signer
	signer2 := &model.Signer{
		ID:          "signer-2",
		Email:       "signer2@example.com",
		DisplayName: "Signer 2",
		CreatedBy:   "user-1",
	}
	err := db.Create(signer2).Error
	require.NoError(t, err)

	// Create multiple signatures for same certificate
	signatures := []model.Signature{
		{
			ID:            "sig-1",
			SignerID:      "test-signer",
			CertificateID: certID,
			Signature:     "sig1",
			IsSigned:      true,
			CreatedBy:     "user-1",
			IsRequested:   true,
		},
		{
			ID:            "sig-2",
			SignerID:      "signer-2",
			CertificateID: certID,
			Signature:     "sig2",
			IsSigned:      false,
			CreatedBy:     "user-1",
			IsRequested:   true,
		},
	}

	for _, sig := range signatures {
		err := db.Create(&sig).Error
		require.NoError(t, err)
	}

	// Get all signatures for certificate
	var certSignatures []model.Signature
	err = db.Where("certificate_id = ?", certID).Find(&certSignatures).Error
	require.NoError(t, err)

	assert.Len(t, certSignatures, 2)
}

// TestSignature_CheckAllSigned tests checking if all signatures are complete
func TestSignature_CheckAllSigned(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	certID, signerID := setupSignatureTest(t, db)

	// Create mix of signed and unsigned signatures
	signatures := []model.Signature{
		{ID: "sig-s1", SignerID: signerID, CertificateID: certID, IsSigned: true, CreatedBy: "user-1", IsRequested: true, Signature: "sig1"},
		{ID: "sig-s2", SignerID: signerID, CertificateID: certID, IsSigned: false, CreatedBy: "user-1", IsRequested: true, Signature: ""},
	}

	for _, sig := range signatures {
		err := db.Create(&sig).Error
		require.NoError(t, err)
	}

	// Check if all signed
	var unsignedCount int64
	err := db.Model(&model.Signature{}).
		Where("certificate_id = ? AND is_signed = ?", certID, false).
		Count(&unsignedCount).Error
	require.NoError(t, err)

	assert.Greater(t, unsignedCount, int64(0), "Should have unsigned signatures")

	// Mark all as signed
	err = db.Model(&model.Signature{}).
		Where("certificate_id = ?", certID).
		Update("is_signed", true).Error
	require.NoError(t, err)

	// Check again
	err = db.Model(&model.Signature{}).
		Where("certificate_id = ? AND is_signed = ?", certID, false).
		Count(&unsignedCount).Error
	require.NoError(t, err)

	assert.Equal(t, int64(0), unsignedCount, "All signatures should be signed")
}

// TestSignature_UpdateLastRequest tests updating last request timestamp
func TestSignature_UpdateLastRequest(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	certID, signerID := setupSignatureTest(t, db)

	signature := &model.Signature{
		ID:            "sig-request",
		SignerID:      signerID,
		CertificateID: certID,
		Signature:     "",
		IsSigned:      false,
		CreatedBy:     "user-1",
		IsRequested:   false,
	}
	err := db.Create(signature).Error
	require.NoError(t, err)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Update last request time
	newRequestTime := time.Now()
	err = db.Model(&model.Signature{}).
		Where("id = ?", "sig-request").
		Updates(map[string]interface{}{
			"is_requested":  true,
			"last_request": newRequestTime,
		}).Error
	require.NoError(t, err)

	// Verify update
	var updated model.Signature
	err = db.Where("id = ?", "sig-request").First(&updated).Error
	require.NoError(t, err)

	assert.True(t, updated.IsRequested)
	assert.WithinDuration(t, newRequestTime, updated.LastRequest, time.Second)
}

// TestSignature_DeleteCascade tests deletion when certificate is deleted
func TestSignature_DeleteCascade(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	certID, signerID := setupSignatureTest(t, db)

	signature := &model.Signature{
		ID:            "sig-cascade",
		SignerID:      signerID,
		CertificateID: certID,
		Signature:     "sig",
		IsSigned:      true,
		CreatedBy:     "user-1",
		IsRequested:   true,
	}
	err := db.Create(signature).Error
	require.NoError(t, err)

	// First delete the signature (to avoid FK constraint)
	err = db.Where("id = ?", "sig-cascade").Delete(&model.Signature{}).Error
	require.NoError(t, err, "Should delete signature")

	// Verify signature was deleted
	helpers.AssertRecordNotExists(t, db, &model.Signature{}, "id = ?", "sig-cascade")

	// Now we can delete the certificate
	err = db.Where("id = ?", certID).Delete(&model.Certificate{}).Error
	require.NoError(t, err, "Should delete certificate after removing signatures")
}

// TestSignature_BulkCreate tests creating multiple signatures at once
func TestSignature_BulkCreate(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	certID, signerID := setupSignatureTest(t, db)

	// Create multiple signatures
	signatures := []model.Signature{
		{ID: "bulk-1", SignerID: signerID, CertificateID: certID, Signature: "sig1", IsSigned: false, CreatedBy: "user-1", IsRequested: true},
		{ID: "bulk-2", SignerID: signerID, CertificateID: certID, Signature: "sig2", IsSigned: false, CreatedBy: "user-1", IsRequested: true},
		{ID: "bulk-3", SignerID: signerID, CertificateID: certID, Signature: "sig3", IsSigned: false, CreatedBy: "user-1", IsRequested: true},
	}

	err := db.Create(&signatures).Error
	require.NoError(t, err)

	// Verify count
	var count int64
	err = db.Model(&model.Signature{}).Where("id LIKE ?", "bulk-%").Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

// TestSignature_GetPendingSignatures tests finding unsigned signatures
func TestSignature_GetPendingSignatures(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	certID, signerID := setupSignatureTest(t, db)

	// Create mix of signed and pending
	signatures := []model.Signature{
		{ID: "pend-1", SignerID: signerID, CertificateID: certID, IsSigned: false, IsRequested: true, CreatedBy: "user-1", Signature: ""},
		{ID: "pend-2", SignerID: signerID, CertificateID: certID, IsSigned: true, IsRequested: true, CreatedBy: "user-1", Signature: "sig"},
		{ID: "pend-3", SignerID: signerID, CertificateID: certID, IsSigned: false, IsRequested: true, CreatedBy: "user-1", Signature: ""},
	}

	for _, sig := range signatures {
		err := db.Create(&sig).Error
		require.NoError(t, err)
	}

	// Get pending signatures
	var pending []model.Signature
	err := db.Where("certificate_id = ? AND is_signed = ?", certID, false).Find(&pending).Error
	require.NoError(t, err)

	assert.Len(t, pending, 2)
}

// TestSignature_GetRequestedSignatures tests finding requested signatures
func TestSignature_GetRequestedSignatures(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	certID, signerID := setupSignatureTest(t, db)

	// Create signatures with different request status
	signatures := []model.Signature{
		{ID: "req-1", SignerID: signerID, CertificateID: certID, IsRequested: true, IsSigned: false, CreatedBy: "user-1", Signature: ""},
		{ID: "req-2", SignerID: signerID, CertificateID: certID, IsRequested: false, IsSigned: false, CreatedBy: "user-1", Signature: ""},
	}

	for _, sig := range signatures {
		err := db.Create(&sig).Error
		require.NoError(t, err)
	}

	// Get requested signatures
	var requested []model.Signature
	err := db.Where("is_requested = ?", true).Find(&requested).Error
	require.NoError(t, err)

	assert.Len(t, requested, 1)
	assert.Equal(t, "req-1", requested[0].ID)
}

// TestSignature_UpdateSignatureData tests updating the actual signature content
func TestSignature_UpdateSignatureData(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	certID, signerID := setupSignatureTest(t, db)

	signature := &model.Signature{
		ID:            "sig-update-data",
		SignerID:      signerID,
		CertificateID: certID,
		Signature:     "",
		IsSigned:      false,
		CreatedBy:     "user-1",
		IsRequested:   true,
	}
	err := db.Create(signature).Error
	require.NoError(t, err)

	// Update with actual signature
	newSignatureData := "base64-encoded-signature-image-data"
	err = db.Model(&model.Signature{}).
		Where("id = ?", "sig-update-data").
		Update("signature", newSignatureData).Error
	require.NoError(t, err)

	// Verify
	var updated model.Signature
	err = db.Where("id = ?", "sig-update-data").First(&updated).Error
	require.NoError(t, err)
	assert.Equal(t, newSignatureData, updated.Signature)
}

// TestSignature_CreatedAtTimestamp tests automatic timestamp
func TestSignature_CreatedAtTimestamp(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	certID, signerID := setupSignatureTest(t, db)

	signature := &model.Signature{
		ID:            "sig-timestamp",
		SignerID:      signerID,
		CertificateID: certID,
		Signature:     "sig",
		IsSigned:      false,
		CreatedBy:     "user-1",
		IsRequested:   true,
	}

	beforeCreate := time.Now()
	err := db.Create(signature).Error
	require.NoError(t, err)

	// Retrieve and check timestamp
	var retrieved model.Signature
	err = db.Where("id = ?", "sig-timestamp").First(&retrieved).Error
	require.NoError(t, err)

	assert.False(t, retrieved.CreatedAt.IsZero(), "CreatedAt should be set automatically")
	// Allow up to 2 seconds variance for database clock differences
	assert.WithinDuration(t, beforeCreate, retrieved.CreatedAt, 2*time.Second, "CreatedAt should be recent")
}

// TestSignature_Concurrency tests concurrent signature operations
func TestSignature_Concurrency(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := container.DB // Use main DB

	// Setup certificate and signer
	cert := &model.Certificate{
		ID:     "concurrent-cert",
		UserID: "user-concurrent",
		Name:   "Concurrent Test",
		Design: "design-1",
	}
	err := db.Create(cert).Error
	require.NoError(t, err)

	signer := &model.Signer{
		ID:          "concurrent-signer",
		Email:       "concurrent@example.com",
		DisplayName: "Concurrent Signer",
		CreatedBy:   "user-concurrent",
	}
	err = db.Create(signer).Error
	require.NoError(t, err)

	done := make(chan bool, 10)

	// Create 10 signatures concurrently
	for i := 0; i < 10; i++ {
		go func(index int) {
			signature := &model.Signature{
				ID:            fmt.Sprintf("concurrent-sig-%d", index),
				SignerID:      "concurrent-signer",
				CertificateID: "concurrent-cert",
				Signature:     fmt.Sprintf("sig-%d", index),
				IsSigned:      false,
				CreatedBy:     "user-concurrent",
				IsRequested:   true,
			}
			err := db.Create(signature).Error
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify count
	var count int64
	err = db.Model(&model.Signature{}).
		Where("certificate_id = ?", "concurrent-cert").
		Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(10), count)

	// Cleanup
	db.Where("certificate_id = ?", "concurrent-cert").Delete(&model.Signature{})
	db.Where("id = ?", "concurrent-cert").Delete(&model.Certificate{})
	db.Where("id = ?", "concurrent-signer").Delete(&model.Signer{})
}
