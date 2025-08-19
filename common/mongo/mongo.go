package mongo

import (
	"context"
	"log/slog"
	"os"
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
		slog.Error("Failed to connect to MongoDB", "error", err)
		os.Exit(1)
	}

	err = client.Ping(ctx, nil)

	if err != nil {
		slog.Error("Failed to ping MongoDB", "error", err)
		os.Exit(1)
	}

	slog.Info("Mongo Connected!")

	common.Mongo = client.Database(*common.Config.MongoDatabase)

}
