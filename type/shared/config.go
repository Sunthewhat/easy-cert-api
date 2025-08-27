package shared

type Config struct {
	Environment       *bool     `yaml:"environment" validate:"required"`
	Port              *string   `yaml:"port" validate:"required"`
	Cors              []*string `yaml:"cors" validate:"required"`
	JWTSecret         *string   `yaml:"jwt_secret" validate:"required"`
	Postgres          *string   `yaml:"postgres" validate:"required"`
	Mongo             *string   `yaml:"mongo" validate:"required"`
	MongoDatabase     *string   `yaml:"mongo_database" validate:"required"`
	RendererUrl       *string   `yaml:"renderer_url" validate:"required"`
	MinIoEndpoint     *string   `yaml:"minio_endpoint" validate:"required"`
	MinIoAccessKey    *string   `yaml:"minio_access_key" validate:"required"`
	MinIoSecretKey    *string   `yaml:"minio_secret_key" validate:"required"`
	BucketResource    *string   `yaml:"bucket_resource" validate:"required"`
	BucketCertificate *string   `yaml:"bucket_certificate" validate:"required"`
}
