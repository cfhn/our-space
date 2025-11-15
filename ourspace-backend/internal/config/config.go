package config

import "github.com/caarlos0/env/v11"

type Config struct {
	HTTPPort int `env:"OURSPACE_BACKEND_HTTP_PORT" envDefault:"8080"`
	GRPCPort int `env:"OURSPACE_BACKEND_GRPC_PORT" envDefault:"50051"`
	Database Database
	Auth     Auth
}

type Database struct {
	URL          string `env:"OURSPACE_BACKEND_DATABASE_URL" envDefault:"postgresql://postgres:postgres@localhost:5432/postgres"`
	MaxOpenConns int    `env:"OURSPACE_BACKEND_DATABASE_MAX_OPEN_CONNS"`
}

type Auth struct {
	SigningKeyPath       string `env:"OURSPACE_BACKEND_SIGNING_KEY_PATH" envDefault:"./signing_key.pem"`
	VerificationKeysPath string `env:"OURSPACE_BACKEND_VERIFICATION_KEY_PATH" envDefault:"verification_key.pem"`
}

func Get() (*Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, err
	}

	return &cfg, err
}
