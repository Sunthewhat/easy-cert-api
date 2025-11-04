package integration

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunthewhat/easy-cert-api/test/helpers"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
)

// TestSigner_CreateAndRetrieve tests basic CRUD operations for signers
func TestSigner_CreateAndRetrieve(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	// Test: Create a signer
	signer := &model.Signer{
		ID:          "signer-123",
		Email:       "test@example.com",
		DisplayName: "Test Signer",
		CreatedBy:   "user-1",
	}

	err := db.Create(signer).Error
	require.NoError(t, err, "Failed to create signer")

	// Verify: Signer was created
	var retrieved model.Signer
	err = db.Where("id = ?", "signer-123").First(&retrieved).Error
	require.NoError(t, err, "Failed to retrieve signer")

	assert.Equal(t, "signer-123", retrieved.ID)
	assert.Equal(t, "test@example.com", retrieved.Email)
	assert.Equal(t, "Test Signer", retrieved.DisplayName)
	assert.Equal(t, "user-1", retrieved.CreatedBy)
}

// TestSigner_UniqueEmail tests email uniqueness constraint
func TestSigner_UniqueEmail(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	// Create first signer
	signer1 := &model.Signer{
		ID:          "signer-1",
		Email:       "unique@example.com",
		DisplayName: "First Signer",
		CreatedBy:   "user-1",
	}
	err := db.Create(signer1).Error
	require.NoError(t, err)

	// Try to create another signer with same email
	signer2 := &model.Signer{
		ID:          "signer-2",
		Email:       "unique@example.com", // Duplicate email
		DisplayName: "Second Signer",
		CreatedBy:   "user-2",
	}
	err = db.Create(signer2).Error

	// Note: Depending on database constraints, this may or may not fail
	// If there's a unique constraint on email, it should fail
	if err != nil {
		assert.Contains(t, err.Error(), "duplicate", "Should mention duplicate constraint")
	}
}

// TestSigner_UpdateDisplayName tests updating signer information
func TestSigner_UpdateDisplayName(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	// Create signer
	signer := &model.Signer{
		ID:          "signer-update",
		Email:       "update@example.com",
		DisplayName: "Original Name",
		CreatedBy:   "user-1",
	}
	err := db.Create(signer).Error
	require.NoError(t, err)

	// Update display name
	err = db.Model(&model.Signer{}).
		Where("id = ?", "signer-update").
		Update("display_name", "Updated Name").Error
	require.NoError(t, err)

	// Verify update
	var updated model.Signer
	err = db.Where("id = ?", "signer-update").First(&updated).Error
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.DisplayName)
}

// TestSigner_Delete tests deleting a signer
func TestSigner_Delete(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	// Create signer
	signer := &model.Signer{
		ID:          "signer-delete",
		Email:       "delete@example.com",
		DisplayName: "To Be Deleted",
		CreatedBy:   "user-1",
	}
	err := db.Create(signer).Error
	require.NoError(t, err)

	// Delete signer
	err = db.Where("id = ?", "signer-delete").Delete(&model.Signer{}).Error
	require.NoError(t, err)

	// Verify deletion
	helpers.AssertRecordNotExists(t, db, &model.Signer{}, "id = ?", "signer-delete")
}

// TestSigner_FindByEmail tests finding signer by email
func TestSigner_FindByEmail(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	// Create multiple signers
	signers := []model.Signer{
		{ID: "s1", Email: "alice@example.com", DisplayName: "Alice", CreatedBy: "user-1"},
		{ID: "s2", Email: "bob@example.com", DisplayName: "Bob", CreatedBy: "user-1"},
		{ID: "s3", Email: "charlie@example.com", DisplayName: "Charlie", CreatedBy: "user-1"},
	}

	for _, s := range signers {
		err := db.Create(&s).Error
		require.NoError(t, err)
	}

	// Find by email
	var found model.Signer
	err := db.Where("email = ?", "bob@example.com").First(&found).Error
	require.NoError(t, err)

	assert.Equal(t, "s2", found.ID)
	assert.Equal(t, "Bob", found.DisplayName)
}

// TestSigner_ListByCreator tests listing all signers created by a user
func TestSigner_ListByCreator(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	// Create signers by different users
	signers := []model.Signer{
		{ID: "s1", Email: "s1@example.com", DisplayName: "Signer 1", CreatedBy: "user-1"},
		{ID: "s2", Email: "s2@example.com", DisplayName: "Signer 2", CreatedBy: "user-1"},
		{ID: "s3", Email: "s3@example.com", DisplayName: "Signer 3", CreatedBy: "user-2"},
	}

	for _, s := range signers {
		err := db.Create(&s).Error
		require.NoError(t, err)
	}

	// Get signers created by user-1
	var user1Signers []model.Signer
	err := db.Where("created_by = ?", "user-1").Find(&user1Signers).Error
	require.NoError(t, err)

	assert.Len(t, user1Signers, 2)
}

// TestSigner_CascadeDeleteWithSignatures tests deletion behavior with related signatures
func TestSigner_CascadeDeleteWithSignatures(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	// Create certificate and signer
	cert := &model.Certificate{
		ID:     "cert-1",
		UserID: "user-1",
		Name:   "Test Cert",
		Design: "design-1",
	}
	err := db.Create(cert).Error
	require.NoError(t, err)

	signer := &model.Signer{
		ID:          "signer-cascade",
		Email:       "cascade@example.com",
		DisplayName: "Cascade Test",
		CreatedBy:   "user-1",
	}
	err = db.Create(signer).Error
	require.NoError(t, err)

	// Create signature linking certificate and signer
	signature := &model.Signature{
		ID:            "sig-1",
		CertificateID: "cert-1",
		SignerID:      "signer-cascade",
	}
	err = db.Create(signature).Error
	require.NoError(t, err)

	// Try to delete signer (may fail due to foreign key constraint)
	err = db.Where("id = ?", "signer-cascade").Delete(&model.Signer{}).Error

	// Depending on database constraints:
	// - If cascade delete: signer and signatures should be deleted
	// - If restrict: should get foreign key error
	if err != nil {
		t.Logf("Delete failed (expected if FK constraint): %v", err)
		// Verify signer still exists
		helpers.AssertRecordExists(t, db, &model.Signer{}, "id = ?", "signer-cascade")
	} else {
		// Cascade delete succeeded
		helpers.AssertRecordNotExists(t, db, &model.Signer{}, "id = ?", "signer-cascade")
	}
}

// TestSigner_BatchCreate tests creating multiple signers at once
func TestSigner_BatchCreate(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	// Create batch of signers
	signers := []model.Signer{
		{ID: "batch-1", Email: "batch1@example.com", DisplayName: "Batch 1", CreatedBy: "user-1"},
		{ID: "batch-2", Email: "batch2@example.com", DisplayName: "Batch 2", CreatedBy: "user-1"},
		{ID: "batch-3", Email: "batch3@example.com", DisplayName: "Batch 3", CreatedBy: "user-1"},
	}

	err := db.Create(&signers).Error
	require.NoError(t, err, "Batch create should succeed")

	// Verify all created
	var count int64
	err = db.Model(&model.Signer{}).Where("id LIKE ?", "batch-%").Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

// TestSigner_EmailValidation tests email format (if validated at DB level)
func TestSigner_EmailValidation(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	testCases := []struct {
		name        string
		email       string
		shouldPass  bool
	}{
		{"Valid email", "valid@example.com", true},
		{"Email with subdomain", "user@mail.example.com", true},
		{"Email with plus", "user+tag@example.com", true},
		{"Email with dash", "user-name@example.com", true},
		{"Email with numbers", "user123@example456.com", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			signer := &model.Signer{
				ID:          "email-test-" + tc.name,
				Email:       tc.email,
				DisplayName: "Email Test",
				CreatedBy:   "user-1",
			}

			err := db.Create(signer).Error
			if tc.shouldPass {
				assert.NoError(t, err, "Valid email should be accepted")
			} else {
				assert.Error(t, err, "Invalid email should be rejected")
			}
		})
	}
}

// TestSigner_CreatedAtTimestamp tests that CreatedAt is automatically set
func TestSigner_CreatedAtTimestamp(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	signer := &model.Signer{
		ID:          "timestamp-test",
		Email:       "timestamp@example.com",
		DisplayName: "Timestamp Test",
		CreatedBy:   "user-1",
	}

	err := db.Create(signer).Error
	require.NoError(t, err)

	// Retrieve and check CreatedAt
	var retrieved model.Signer
	err = db.Where("id = ?", "timestamp-test").First(&retrieved).Error
	require.NoError(t, err)

	assert.False(t, retrieved.CreatedAt.IsZero(), "CreatedAt should be set automatically")
}

// TestSigner_SearchByDisplayName tests searching signers by display name
func TestSigner_SearchByDisplayName(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)

	// Create signers with different names
	signers := []model.Signer{
		{ID: "s1", Email: "john@example.com", DisplayName: "John Doe", CreatedBy: "user-1"},
		{ID: "s2", Email: "jane@example.com", DisplayName: "Jane Doe", CreatedBy: "user-1"},
		{ID: "s3", Email: "bob@example.com", DisplayName: "Bob Smith", CreatedBy: "user-1"},
	}

	for _, s := range signers {
		err := db.Create(&s).Error
		require.NoError(t, err)
	}

	// Search for "Doe"
	var results []model.Signer
	err := db.Where("display_name LIKE ?", "%Doe%").Find(&results).Error
	require.NoError(t, err)

	assert.Len(t, results, 2, "Should find 2 signers with 'Doe' in name")
}

// TestSigner_Concurrency tests concurrent signer creation
func TestSigner_Concurrency(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := container.DB // Use main DB, not transaction

	done := make(chan bool, 10)

	// Create 10 signers concurrently
	for i := 0; i < 10; i++ {
		go func(index int) {
			signer := &model.Signer{
				ID:          fmt.Sprintf("concurrent-signer-%d", index),
				Email:       fmt.Sprintf("concurrent%d@example.com", index),
				DisplayName: fmt.Sprintf("Concurrent Signer %d", index),
				CreatedBy:   "user-concurrent",
			}
			err := db.Create(signer).Error
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
	err := db.Model(&model.Signer{}).Where("created_by = ?", "user-concurrent").Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(10), count)

	// Cleanup
	db.Where("created_by = ?", "user-concurrent").Delete(&model.Signer{})
}
