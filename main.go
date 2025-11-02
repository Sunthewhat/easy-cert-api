package main

import (
	"flag"
	"log/slog"

	"github.com/sunthewhat/easy-cert-api/api"
	"github.com/sunthewhat/easy-cert-api/common/config"
	"github.com/sunthewhat/easy-cert-api/common/gorm"
	"github.com/sunthewhat/easy-cert-api/common/mongo"
	"github.com/sunthewhat/easy-cert-api/common/util"
)

func main() {
	isPushDB := flag.Bool("PushDB", false, "Run database migration")
	isPullDB := flag.Bool("PullDB", false, "Run database pulling")
	isRunAfter := flag.Bool("Run", false, "Run after db process")
	isProd := flag.Bool("Prod", false, "Run a production")
	flag.Parse()
	config.LoadConfig()
	if *isPushDB || *isPullDB {
		if *isPullDB {
			gorm.Pull_db()
		}
		if *isPushDB {
			gorm.Push_db()
		}
		if !*isRunAfter {
			return
		}
	}

	if *isProd {
		slog.Info("Pusing database to PostgreSQL")
		gorm.Push_db()
	}

	gorm.InitGorm()
	mongo.InitMongo()
	util.InitDialer()

	if err := util.InitMinIO(); err != nil {
		slog.Error("Failed to initialize MinIO", "error", err)
	} else {
		slog.Info("MinIO initialized successfully")
	}

	// Start signature reminder job for daily email reminders
	util.StartSignatureReminderJob()

	// Start preview cleanup job for removing old preview images (30 days)
	util.StartPreviewCleanupJob()

	api.InitFiber()
}
