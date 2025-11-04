package certificatemodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunthewhat/easy-cert-api/test/helpers"
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
	"github.com/sunthewhat/easy-cert-api/type/shared/query"
)

// TestCertificateRepository_Create tests certificate creation
func TestCertificateRepository_Create(t *testing.T) {
	// Setup test database
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)
	q := query.Use(db)
	repo := NewCertificateRepository(q)

	// Test data
	certData := payload.CreateCertificatePayload{
		Name:   "Test Certificate",
		Design: "template-1",
	}

	// Execute
	cert, err := repo.Create(certData, "user-123")

	// Assert
	require.NoError(t, err, "Create should succeed")
	assert.NotEmpty(t, cert.ID, "Certificate ID should be generated")
	assert.Equal(t, "user-123", cert.UserID)
	assert.Equal(t, "Test Certificate", cert.Name)
	assert.Equal(t, "template-1", cert.Design)

	// Verify in database
	var found model.Certificate
	err = db.Where("id = ?", cert.ID).First(&found).Error
	require.NoError(t, err)
	assert.Equal(t, cert.ID, found.ID)
}

// TestCertificateRepository_GetById tests retrieving certificate by ID
func TestCertificateRepository_GetById(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)
	q := query.Use(db)
	repo := NewCertificateRepository(q)

	// Create test certificate
	cert := &model.Certificate{
		ID:     "cert-123",
		UserID: "user-1",
		Name:   "Test Cert",
		Design: "design-1",
	}
	err := db.Create(cert).Error
	require.NoError(t, err)

	// Test: Get by ID
	found, err := repo.GetById("cert-123")

	// Assert
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "cert-123", found.ID)
	assert.Equal(t, "Test Cert", found.Name)
}

// TestCertificateRepository_GetById_NotFound tests getting non-existent certificate
func TestCertificateRepository_GetById_NotFound(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)
	q := query.Use(db)
	repo := NewCertificateRepository(q)

	// Test: Get non-existent
	found, err := repo.GetById("nonexistent")

	// Assert
	assert.NoError(t, err, "Should not error for not found")
	assert.Nil(t, found, "Should return nil for not found")
}

// TestCertificateRepository_GetByUser tests retrieving certificates by user
func TestCertificateRepository_GetByUser(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)
	q := query.Use(db)
	repo := NewCertificateRepository(q)

	// Create test certificates
	certs := []model.Certificate{
		{ID: "cert-1", UserID: "user-1", Name: "Cert 1", Design: "design-1"},
		{ID: "cert-2", UserID: "user-1", Name: "Cert 2", Design: "design-1"},
		{ID: "cert-3", UserID: "user-2", Name: "Cert 3", Design: "design-1"},
	}
	for _, c := range certs {
		err := db.Create(&c).Error
		require.NoError(t, err)
	}

	// Test: Get by user
	found, err := repo.GetByUser("user-1")

	// Assert
	require.NoError(t, err)
	assert.Len(t, found, 2, "Should find 2 certificates for user-1")
	assert.Equal(t, "cert-1", found[0].ID)
	assert.Equal(t, "cert-2", found[1].ID)
}

// TestCertificateRepository_GetAll tests retrieving all certificates
func TestCertificateRepository_GetAll(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)
	q := query.Use(db)
	repo := NewCertificateRepository(q)

	// Create test certificates
	for i := 1; i <= 3; i++ {
		cert := &model.Certificate{
			ID:     string(rune('a' + i)),
			UserID: "user-1",
			Name:   "Cert",
			Design: "design-1",
		}
		err := db.Create(cert).Error
		require.NoError(t, err)
	}

	// Test: Get all
	found, err := repo.GetAll()

	// Assert
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(found), 3, "Should find at least 3 certificates")
}

// TestCertificateRepository_Update tests updating a certificate
func TestCertificateRepository_Update(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)
	q := query.Use(db)
	repo := NewCertificateRepository(q)

	// Create test certificate
	cert := &model.Certificate{
		ID:     "cert-update",
		UserID: "user-1",
		Name:   "Original Name",
		Design: "original-design",
	}
	err := db.Create(cert).Error
	require.NoError(t, err)

	// Test: Update name and design
	updated, err := repo.Update("cert-update", "Updated Name", "new-design")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, "new-design", updated.Design)

	// Verify in database
	var found model.Certificate
	err = db.Where("id = ?", "cert-update").First(&found).Error
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", found.Name)
	assert.Equal(t, "new-design", found.Design)
}

// TestCertificateRepository_Update_PartialUpdate tests partial updates
func TestCertificateRepository_Update_PartialUpdate(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)
	q := query.Use(db)
	repo := NewCertificateRepository(q)

	// Create test certificate
	cert := &model.Certificate{
		ID:     "cert-partial",
		UserID: "user-1",
		Name:   "Original",
		Design: "original",
	}
	err := db.Create(cert).Error
	require.NoError(t, err)

	// Test: Update only name
	updated, err := repo.Update("cert-partial", "New Name", "")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, "original", updated.Design, "Design should remain unchanged")
}

// TestCertificateRepository_Update_NoChanges tests update with no changes
func TestCertificateRepository_Update_NoChanges(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)
	q := query.Use(db)
	repo := NewCertificateRepository(q)

	// Create test certificate
	cert := &model.Certificate{
		ID:     "cert-nochange",
		UserID: "user-1",
		Name:   "Original",
		Design: "original",
	}
	err := db.Create(cert).Error
	require.NoError(t, err)

	// Test: Update with empty values (no changes)
	updated, err := repo.Update("cert-nochange", "", "")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "Original", updated.Name)
	assert.Equal(t, "original", updated.Design)
}

// TestCertificateRepository_Delete tests deleting a certificate
func TestCertificateRepository_Delete(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)
	q := query.Use(db)
	repo := NewCertificateRepository(q)

	// Create test certificate
	cert := &model.Certificate{
		ID:     "cert-delete",
		UserID: "user-1",
		Name:   "To Delete",
		Design: "design-1",
	}
	err := db.Create(cert).Error
	require.NoError(t, err)

	// Test: Delete
	deleted, err := repo.Delete("cert-delete")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "cert-delete", deleted.ID)

	// Verify deleted from database
	var found model.Certificate
	err = db.Where("id = ?", "cert-delete").First(&found).Error
	assert.Error(t, err, "Should not find deleted certificate")
}

// TestCertificateRepository_Delete_NotFound tests deleting non-existent certificate
func TestCertificateRepository_Delete_NotFound(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)
	q := query.Use(db)
	repo := NewCertificateRepository(q)

	// Test: Delete non-existent
	deleted, err := repo.Delete("nonexistent")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, deleted)
	assert.Contains(t, err.Error(), "not found")
}

// TestCertificateRepository_AddThumbnailUrl tests adding thumbnail URL
func TestCertificateRepository_AddThumbnailUrl(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)
	q := query.Use(db)
	repo := NewCertificateRepository(q)

	// Create test certificate
	cert := &model.Certificate{
		ID:     "cert-thumb",
		UserID: "user-1",
		Name:   "Test",
		Design: "design-1",
	}
	err := db.Create(cert).Error
	require.NoError(t, err)

	// Test: Add thumbnail URL
	err = repo.AddThumbnailUrl("cert-thumb", "https://example.com/thumb.jpg")

	// Assert
	require.NoError(t, err)

	// Verify in database
	var found model.Certificate
	err = db.Where("id = ?", "cert-thumb").First(&found).Error
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/thumb.jpg", found.ThumbnailURL)
}

// TestCertificateRepository_EditArchiveUrl tests updating archive URL
func TestCertificateRepository_EditArchiveUrl(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)
	q := query.Use(db)
	repo := NewCertificateRepository(q)

	// Create test certificate
	cert := &model.Certificate{
		ID:     "cert-archive",
		UserID: "user-1",
		Name:   "Test",
		Design: "design-1",
	}
	err := db.Create(cert).Error
	require.NoError(t, err)

	// Test: Edit archive URL
	err = repo.EditArchiveUrl("cert-archive", "https://example.com/archive.zip")

	// Assert
	require.NoError(t, err)

	// Verify in database
	var found model.Certificate
	err = db.Where("id = ?", "cert-archive").First(&found).Error
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/archive.zip", found.ArchiveURL)
}

// TestCertificateRepository_MarkAsSigned tests marking certificate as signed
func TestCertificateRepository_MarkAsSigned(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)
	q := query.Use(db)
	repo := NewCertificateRepository(q)

	// Create test certificate
	cert := &model.Certificate{
		ID:       "cert-sign",
		UserID:   "user-1",
		Name:     "Test",
		Design:   "design-1",
		IsSigned: false,
	}
	err := db.Create(cert).Error
	require.NoError(t, err)

	// Test: Mark as signed
	err = repo.MarkAsSigned("cert-sign")

	// Assert
	require.NoError(t, err)

	// Verify in database
	var found model.Certificate
	err = db.Where("id = ?", "cert-sign").First(&found).Error
	require.NoError(t, err)
	assert.True(t, found.IsSigned)
}

// TestCertificateRepository_MarkAsUnsigned tests marking certificate as unsigned
func TestCertificateRepository_MarkAsUnsigned(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)
	q := query.Use(db)
	repo := NewCertificateRepository(q)

	// Create test certificate
	cert := &model.Certificate{
		ID:       "cert-unsign",
		UserID:   "user-1",
		Name:     "Test",
		Design:   "design-1",
		IsSigned: true,
	}
	err := db.Create(cert).Error
	require.NoError(t, err)

	// Test: Mark as unsigned
	err = repo.MarkAsUnsigned("cert-unsign")

	// Assert
	require.NoError(t, err)

	// Verify in database
	var found model.Certificate
	err = db.Where("id = ?", "cert-unsign").First(&found).Error
	require.NoError(t, err)
	assert.False(t, found.IsSigned)
}

// TestCertificateRepository_MarkAsDistributed tests marking certificate as distributed
func TestCertificateRepository_MarkAsDistributed(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := helpers.GetTestDB(t, container)
	q := query.Use(db)
	repo := NewCertificateRepository(q)

	// Create test certificate
	cert := &model.Certificate{
		ID:            "cert-dist",
		UserID:        "user-1",
		Name:          "Test",
		Design:        "design-1",
		IsDistributed: false,
	}
	err := db.Create(cert).Error
	require.NoError(t, err)

	// Test: Mark as distributed
	err = repo.MarkAsDistributed("cert-dist")

	// Assert
	require.NoError(t, err)

	// Verify in database
	var found model.Certificate
	err = db.Where("id = ?", "cert-dist").First(&found).Error
	require.NoError(t, err)
	assert.True(t, found.IsDistributed)
}

// TestCertificateRepository_Concurrency tests concurrent operations
func TestCertificateRepository_Concurrency(t *testing.T) {
	container := helpers.SetupTestDatabase(t)
	db := container.DB // Use main DB for concurrency test
	q := query.Use(db)
	repo := NewCertificateRepository(q)

	done := make(chan bool, 10)

	// Create 10 certificates concurrently
	for i := 0; i < 10; i++ {
		go func(idx int) {
			certData := payload.CreateCertificatePayload{
				Name:   "Concurrent Cert",
				Design: "design-1",
			}
			cert, err := repo.Create(certData, "user-concurrent")
			assert.NoError(t, err)
			assert.NotEmpty(t, cert.ID)
			done <- true
		}(i)
	}

	// Wait for all
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify count
	certs, err := repo.GetByUser("user-concurrent")
	require.NoError(t, err)
	assert.Equal(t, 10, len(certs))

	// Cleanup
	for _, cert := range certs {
		db.Delete(cert)
	}
}
