package gorm

import (
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Push_db() {
	lg := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             100 * time.Millisecond,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	dialector := postgres.New(
		postgres.Config{
			DSN: *common.Config.Postgres,
		},
	)

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: lg,
	})

	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	if err := db.AutoMigrate(
		new(model.Certificate),
		new(model.Participant),
		new(model.Graphic),
	); err != nil {
		slog.Error("Failed to migrate database", "error", err)
		os.Exit(1)
	}

	slog.Info("Database migration completed successfully")
}
