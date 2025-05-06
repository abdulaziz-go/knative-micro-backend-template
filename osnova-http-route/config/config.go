package config

type Config struct {
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	return cfg, nil
}
