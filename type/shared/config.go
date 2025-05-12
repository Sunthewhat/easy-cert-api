package shared

type Config struct {
	Environment *bool     `yaml:"environment" validate:"required"`
	Port        *string   `yaml:"port" validate:"required"`
	Cors        []*string `yaml:"cors" validate:"required"`
	JWTSecret   *string   `yaml:"jwt_secret" validate:"required"`
	Db          *string   `yaml:"db" validate:"required"`
}
