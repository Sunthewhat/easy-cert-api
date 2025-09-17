package gorm

import (
	"log/slog"
	"os"
	"time"

	slogGorm "github.com/orandin/slog-gorm"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/shared/query"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitGorm() {
	// Configure slog-gorm logger
	lg := slogGorm.New(
		slogGorm.WithHandler(slog.Default().Handler()),
		slogGorm.WithSlowThreshold(100*time.Millisecond),
	)

	// Config GORM Connector
	connector := postgres.New(
		postgres.Config{
			DSN:                  *common.Config.Postgres,
			PreferSimpleProtocol: true,
		},
	)

	// Open connection
	db, connectionErr := gorm.Open(connector, &gorm.Config{
		Logger: lg,
	})

	if connectionErr != nil {
		slog.Error("Failed to connect to database", "error", connectionErr)
		os.Exit(1)
	}

	slog.Info("GORM Connected!")

	common.Gorm = query.Use(db)
}
