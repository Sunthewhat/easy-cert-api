package gorm

import (
	"log"
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
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(
		new(model.User),
		new(model.Certificate),
		new(model.Participant),
		new(model.Graphic),
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
}
