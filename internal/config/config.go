package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv      string `env:"APP_ENV" envDefault:"development"`
	AppHost     string `env:"APP_HOST" envDefault:"localhost"`
	AppPort     string `env:"APP_PORT" envDefault:"8080"`
	AppBaseURL  string `env:"APP_BASE_URL" envDefault:"http://localhost:8080"`
	DatabaseURL string `env:"DATABASE_URL,required"`

	SessionSecret string `env:"SESSION_SECRET,required"`
}

func Load() (Config, error) {
	_ = godotenv.Load()

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

func (c Config) Addr() string {
	return c.AppHost + ":" + c.AppPort
}

func (c Config) IsProduction() bool {
	return c.AppEnv == "production"
}
