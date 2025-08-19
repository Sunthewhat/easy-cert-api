package main

import (
	"flag"

	"github.com/sunthewhat/easy-cert-api/api"
	"github.com/sunthewhat/easy-cert-api/common/config"
	"github.com/sunthewhat/easy-cert-api/common/gorm"
	"github.com/sunthewhat/easy-cert-api/common/mongo"
)

func main() {
	isPushDB := flag.Bool("PushDB", false, "Run database migration")
	isPullDB := flag.Bool("PullDB", false, "Run database pulling")
	isRunAfter := flag.Bool("Run", false, "Run after db process")
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

	gorm.InitGorm()
	mongo.InitMongo()
	api.InitFiber()
}
