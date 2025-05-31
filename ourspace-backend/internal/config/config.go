package config

import "github.com/caarlos0/env/v11"

type Config struct {
	HTTPPort int `env:"OURSPACE_BACKEND_HTTP_PORT" envDefault:"8080"`
	GRPCPort int `env:"OURSPACE_BACKEND_GRPC_PORT" envDefault:"50051"`
	Database Database
}

type Database struct {
	URL          string `env:"OURSPACE_BACKEND_DATABASE_URL" envDefault:"postgresql://postgres:postgres@localhost:5433/postgres"`
	MaxOpenConns int    `env:"OURSPACE_BACKEND_DATABASE_MAX_OPEN_CONNS"`
}

func Get() (*Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, err
	}

	return &cfg, err
}
