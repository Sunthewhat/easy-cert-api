package mongo

import (
	"context"
	"log"
	"log/slog"
	"time"

	"github.com/sunthewhat/easy-cert-api/common"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func InitMongo() {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	clientOptions := options.Client().ApplyURI(*common.Config.Mongo)
	client, err := mongo.Connect(ctx, clientOptions)

	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	err = client.Ping(ctx, nil)

	if err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	slog.Info("Mongo Connected!")

	common.Mongo = client.Database(*common.Config.MongoDatabase)

}
