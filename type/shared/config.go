package shared

type Config struct {
	Environment       *bool     `yaml:"environment" validate:"required"`
	Port              *string   `yaml:"port" validate:"required"`
	BackendURL        *string   `yaml:"backend_url" validate:"required"`
	Cors              []*string `yaml:"cors" validate:"required"`
	JWTSecret         *string   `yaml:"jwt_secret" validate:"required"`
	Postgres          *string   `yaml:"postgres" validate:"required"`
	Mongo             *string   `yaml:"mongo" validate:"required"`
	MongoDatabase     *string   `yaml:"mongo_database" validate:"required"`
	VerifyHost        *string   `yaml:"verify_host" validate:"required"`
	MinIoEndpoint     *string   `yaml:"minio_endpoint" validate:"required"`
	MinIoAccessKey    *string   `yaml:"minio_access_key" validate:"required"`
	MinIoSecretKey    *string   `yaml:"minio_secret_key" validate:"required"`
	BucketResource    *string   `yaml:"bucket_resource" validate:"required"`
	BucketCertificate *string   `yaml:"bucket_certificate" validate:"required"`
	SsoIssuerUrl      *string   `yaml:"sso_issuer_url" validate:"required"`
	SsoClient         *string   `yaml:"sso_client" validate:"required"`
	SsoSecret         *string   `yaml:"sso_secret" validate:"required"`
	MailHost          *string   `yaml:"mail_host" validate:"required"`
	MailUser          *string   `yaml:"mail_user" validate:"required"`
	MailPass          *string   `yaml:"mail_pass" validate:"required"`
	SigningEnabled    *bool     `yaml:"signing_enabled"`
	SigningCertPath   *string   `yaml:"signing_cert_path"`
	SigningKeyPath    *string   `yaml:"signing_key_path"`
	EncryptionKey     *string   `yaml:"encryption_key" validate:"required"`
}
