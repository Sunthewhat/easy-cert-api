package gorm

import (
	"log"
	"os"
	"time"

	"github.com/bsthun/gut"
	"github.com/sunthewhat/easy-cert-api/common"
	"gorm.io/driver/postgres"
	"gorm.io/gen"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Pull_db() {
	lg := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             100 * time.Millisecond,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
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
		gut.Fatal("Failed to connect to database", connectionErr)
	}

	g := gen.NewGenerator(
		gen.Config{
			OutPath: "./type/shared/query",
			Mode:    gen.WithoutContext,
		},
	)

	g.UseDB(db)

	g.ApplyBasic(g.GenerateAllTable()...)

	g.Execute()
}
