package integration

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunthewhat/easy-cert-api/test/helpers"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
)

// TestCertificate_CreateAndRetrieve tests basic CRUD operations
func TestCertificate_CreateAndRetrieve(t *testing.T) {
	// Setup: Start PostgreSQL container
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	// Test: Create a certificate
	cert := &model.Certificate{
		ID:     "cert-123",
		UserID: "user-123",
		Name:   "Test Certificate",
		Design: "template-1",
	}

	err := db.Create(cert).Error
	require.NoError(t, err, "Failed to create certificate")

	// Verify: Certificate was created
	var retrieved model.Certificate
	err = db.Where("id = ?", "cert-123").First(&retrieved).Error
	require.NoError(t, err, "Failed to retrieve certificate")

	assert.Equal(t, "cert-123", retrieved.ID)
	assert.Equal(t, "user-123", retrieved.UserID)
	assert.Equal(t, "Test Certificate", retrieved.Name)
	assert.Equal(t, "template-1", retrieved.Design)
}

// TestCertificate_GetByUser tests filtering certificates by user
func TestCertificate_GetByUser(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	// Create multiple certificates for different users
	certificates := []model.Certificate{
		{ID: "cert-1", UserID: "user-1", Name: "Cert 1", Design: "design-1"},
		{ID: "cert-2", UserID: "user-1", Name: "Cert 2", Design: "design-1"},
		{ID: "cert-3", UserID: "user-2", Name: "Cert 3", Design: "design-1"},
	}

	for _, cert := range certificates {
		err := db.Create(&cert).Error
		require.NoError(t, err)
	}

	// Test: Get certificates for user-1
	var user1Certs []model.Certificate
	err := db.Where("user_id = ?", "user-1").Find(&user1Certs).Error
	require.NoError(t, err)

	// Verify: Should get 2 certificates
	assert.Len(t, user1Certs, 2)
	assert.Equal(t, "cert-1", user1Certs[0].ID)
	assert.Equal(t, "cert-2", user1Certs[1].ID)
}

// TestCertificate_Update tests updating a certificate
func TestCertificate_Update(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	// Create initial certificate
	cert := &model.Certificate{
		ID:     "cert-update",
		UserID: "user-1",
		Name:   "Original Name",
		Design: "design-1",
	}
	err := db.Create(cert).Error
	require.NoError(t, err)

	// Test: Update the certificate
	err = db.Model(&model.Certificate{}).
		Where("id = ?", "cert-update").
		Updates(map[string]interface{}{
			"name":   "Updated Name",
			"design": "design-2",
		}).Error
	require.NoError(t, err)

	// Verify: Changes were saved
	var updated model.Certificate
	err = db.Where("id = ?", "cert-update").First(&updated).Error
	require.NoError(t, err)

	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, "design-2", updated.Design)
}

// TestCertificate_Delete tests deleting a certificate
func TestCertificate_Delete(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	// Create certificate
	cert := &model.Certificate{
		ID:     "cert-delete",
		UserID: "user-1",
		Name:   "To Be Deleted",
		Design: "design-1",
	}
	err := db.Create(cert).Error
	require.NoError(t, err)

	// Verify it exists
	helpers.AssertRecordExists(t, db, &model.Certificate{}, "id = ?", "cert-delete")

	// Test: Delete the certificate
	err = db.Where("id = ?", "cert-delete").Delete(&model.Certificate{}).Error
	require.NoError(t, err)

	// Verify: Certificate was deleted
	helpers.AssertRecordNotExists(t, db, &model.Certificate{}, "id = ?", "cert-delete")
}

// TestCertificate_WithSignatures tests relationship with signatures
func TestCertificate_WithSignatures(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	// Create certificate
	cert := &model.Certificate{
		ID:     "cert-with-sigs",
		UserID: "user-1",
		Name:   "Certificate with Signatures",
		Design: "design-1",
	}
	err := db.Create(cert).Error
	require.NoError(t, err)

	// Create signatures for this certificate
	signatures := []model.Signature{
		{
			ID:            "sig-1",
			CertificateID: "cert-with-sigs",
			SignerID:      "signer-1",
		},
		{
			ID:            "sig-2",
			CertificateID: "cert-with-sigs",
			SignerID:      "signer-2",
		},
	}

	for _, sig := range signatures {
		err := db.Create(&sig).Error
		require.NoError(t, err)
	}

	// Test: Retrieve signatures for this certificate
	var relatedSignatures []model.Signature
	err = db.Where("certificate_id = ?", "cert-with-sigs").Find(&relatedSignatures).Error
	require.NoError(t, err)

	// Verify: Should have 2 signatures
	assert.Len(t, relatedSignatures, 2)
	assert.Equal(t, "cert-with-sigs", relatedSignatures[0].CertificateID)
	assert.Equal(t, "cert-with-sigs", relatedSignatures[1].CertificateID)
}

// TestCertificate_ConcurrentCreation tests concurrent certificate creation
func TestCertificate_ConcurrentCreation(t *testing.T) {
	container := helpers.SetupTestDatabase(t)

	// Don't use transaction for this test - we want to test concurrent DB access
	db := container.DB

	done := make(chan bool, 10)

	// Create 10 certificates concurrently
	for i := 0; i < 10; i++ {
		go func(index int) {
			cert := &model.Certificate{
				ID:     fmt.Sprintf("cert-concurrent-%d", index),
				UserID: "user-concurrent",
				Name:   fmt.Sprintf("Concurrent Cert %d", index),
				Design: "design-1",
			}
			err := db.Create(cert).Error
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify: All 10 certificates were created
	var count int64
	err := db.Model(&model.Certificate{}).Where("user_id = ?", "user-concurrent").Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(10), count)

	// Cleanup
	db.Where("user_id = ?", "user-concurrent").Delete(&model.Certificate{})
}

// TestCertificate_ValidationConstraints tests database constraints
func TestCertificate_ValidationConstraints(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	// Test: Try to create certificate with duplicate ID
	cert1 := &model.Certificate{
		ID:     "duplicate-id",
		UserID: "user-1",
		Name:   "First",
		Design: "design-1",
	}
	err := db.Create(cert1).Error
	require.NoError(t, err)

	cert2 := &model.Certificate{
		ID:     "duplicate-id", // Same ID
		UserID: "user-2",
		Name:   "Second",
		Design: "design-2",
	}
	err = db.Create(cert2).Error
	assert.Error(t, err, "Should fail with duplicate ID")
	assert.Contains(t, err.Error(), "duplicate", "Error should mention duplicate key")
}

// TestCertificate_TransactionRollback tests transaction rollback on error
func TestCertificate_TransactionRollback(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := container.DB

	// Start a transaction
	tx := db.Begin()

	// Create a certificate in transaction
	cert := &model.Certificate{
		ID:     "cert-rollback",
		UserID: "user-1",
		Name:   "Will Be Rolled Back",
		Design: "design-1",
	}
	err := tx.Create(cert).Error
	require.NoError(t, err)

	// Rollback the transaction
	tx.Rollback()

	// Verify: Certificate should NOT exist in database
	helpers.AssertRecordNotExists(t, db, &model.Certificate{}, "id = ?", "cert-rollback")
}
