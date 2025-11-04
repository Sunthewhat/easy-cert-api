package helpers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/sunthewhat/easy-cert-api/type/shared/model"
)

// PostgresContainer holds the test database container
type PostgresContainer struct {
	Container testcontainers.Container
	DB        *gorm.DB
	ConnStr   string
}

// SetupTestDatabase creates a PostgreSQL container and returns a GORM DB connection
func SetupTestDatabase(t *testing.T) *PostgresContainer {
	ctx := context.Background()

	// Create PostgreSQL container
	postgresContainer, err := postgrescontainer.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgrescontainer.WithDatabase("test_easycert"),
		postgrescontainer.WithUsername("test"),
		postgrescontainer.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err, "Failed to start PostgreSQL container")

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "Failed to get connection string")

	// Connect with GORM
	db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Silent mode for tests
	})
	require.NoError(t, err, "Failed to connect to test database")

	// Run migrations
	err = db.AutoMigrate(
		&model.Certificate{},
		&model.Signer{},
		&model.Signature{},
		&model.Participant{},
	)
	require.NoError(t, err, "Failed to run migrations")

	// Register cleanup
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	})

	return &PostgresContainer{
		Container: postgresContainer,
		DB:        db,
		ConnStr:   connStr,
	}
}

// GetTestDB returns a DB transaction that auto-rollbacks for test isolation
func GetTestDB(t *testing.T, container *PostgresContainer) *gorm.DB {
	tx := container.DB.Begin()
	require.NoError(t, tx.Error, "Failed to begin transaction")

	t.Cleanup(func() {
		tx.Rollback()
	})

	return tx
}

// SeedTestData inserts common test data
func SeedTestData(t *testing.T, db *gorm.DB) {
	// Example: Create test signers
	testSigners := []model.Signer{
		{
			ID:          "signer-1",
			Email:       "test1@example.com",
			DisplayName: "John Doe",
			CreatedBy:   "system",
		},
		{
			ID:          "signer-2",
			Email:       "test2@example.com",
			DisplayName: "Jane Smith",
			CreatedBy:   "system",
		},
	}

	for _, signer := range testSigners {
		err := db.Create(&signer).Error
		require.NoError(t, err, "Failed to seed test signer")
	}
}

// CleanupTestData removes all data from tables (for tests not using transactions)
func CleanupTestData(t *testing.T, db *gorm.DB) {
	tables := []string{
		"participants",
		"signatures",
		"certificates",
		"signers",
	}

	for _, table := range tables {
		err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)).Error
		require.NoError(t, err, "Failed to truncate table %s", table)
	}
}

// AssertRecordExists checks if a record exists in the database
func AssertRecordExists(t *testing.T, db *gorm.DB, model interface{}, condition string, args ...interface{}) {
	var count int64
	err := db.Model(model).Where(condition, args...).Count(&count).Error
	require.NoError(t, err, "Failed to count records")
	require.Greater(t, count, int64(0), "Expected record to exist but found none")
}

// AssertRecordNotExists checks that a record does not exist
func AssertRecordNotExists(t *testing.T, db *gorm.DB, model interface{}, condition string, args ...interface{}) {
	var count int64
	err := db.Model(model).Where(condition, args...).Count(&count).Error
	require.NoError(t, err, "Failed to count records")
	require.Equal(t, int64(0), count, "Expected no records but found %d", count)
}
