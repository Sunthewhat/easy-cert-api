package common

import (
	"github.com/minio/minio-go/v7"
	"github.com/sunthewhat/easy-cert-api/type/shared"
	"github.com/sunthewhat/easy-cert-api/type/shared/query"
	"go.mongodb.org/mongo-driver/mongo"
	"gopkg.in/gomail.v2"
)

var Config *shared.Config
var Gorm *query.Query
var Mongo *mongo.Database
var Dialer *gomail.Dialer
var MinIOClient *minio.Client
