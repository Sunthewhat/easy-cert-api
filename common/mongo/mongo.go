package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/bsthun/gut"
	"github.com/sunthewhat/secure-docs-api/common"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func InitMongo() {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	clientOptions := options.Client().ApplyURI(*common.Config.Mongo)
	client, err := mongo.Connect(ctx, clientOptions)

	if err != nil {
		gut.Fatal("Failed to connect to MongoDB", err)
	}

	err = client.Ping(ctx, nil)

	if err != nil {
		gut.Fatal("Failed to ping MongoDB", err)
	}

	fmt.Println("MongoDB Connected!")

	common.Mongo = client.Database(*common.Config.MongoDatabase)

}
