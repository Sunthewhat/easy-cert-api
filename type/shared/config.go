package shared

type Config struct {
	Environment   *bool     `yaml:"environment" validate:"required"`
	Port          *string   `yaml:"port" validate:"required"`
	Cors          []*string `yaml:"cors" validate:"required"`
	JWTSecret     *string   `yaml:"jwt_secret" validate:"required"`
	Postgres      *string   `yaml:"postgres" validate:"required"`
	Mongo         *string   `yaml:"mongo" validate:"required"`
	MongoDatabase *string   `yaml:"mongo_database" validate:"required"`
	RendererUrl   *string   `yaml:"renderer_url" validate:"required"`
}
