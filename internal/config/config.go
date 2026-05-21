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

	R2AccountID       string `env:"R2_ACCOUNT_ID,required"`
	R2AccessKeyID     string `env:"R2_ACCESS_KEY_ID,required"`
	R2SecretAccessKey string `env:"R2_SECRET_ACCESS_KEY,required"`
	R2BucketName      string `env:"R2_BUCKET_NAME,required"`
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

func (c Config) R2Endpoint() string {
	return "https://" + c.R2AccountID + ".r2.cloudflarestorage.com"
}
