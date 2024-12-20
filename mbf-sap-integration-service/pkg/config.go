package pkg

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type (
	// Config -.
	Config struct {
		App   `yaml:"app"`
		Redis `yaml:"redis"`
		Name  string `env-required:"false" yaml:"name" env:"NAME"`
	}

	// App -.
	App struct {
		Name string `env-required:"false" yaml:"name"    env:"APP_NAME"`
	}

	// Redis -.
	Redis struct {
		RedisHost string `env-required:"false" yaml:"host" env:"REDIS_HOST"`
		RedisPort int    `env-required:"false" yaml:"port" env:"REDIS_PORT"`
		RedisUser string `env-required:"false" yaml:"user" env:"REDIS_USER"`
		RedisPass string `env-required:"false" yaml:"pass" env:"REDIS_PASS"`
		Enabled   bool   `env-required:"false" yaml:"enabled" env:"REDIS_ENABLED"`
	}
)

// NewConfig returns app config.
func NewConfig() (*Config, error) {
	cfg := &Config{}

	err := cleanenv.ReadConfig("func.yaml", cfg)
	if err != nil {
		return cfg, fmt.Errorf("config error: %w", err)
	}

	err = godotenv.Load("/app/.env")
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	err = cleanenv.ReadEnv(cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}
