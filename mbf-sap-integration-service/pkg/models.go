package pkg

import (
	"fmt"
	"os"

	cache "github.com/golanguzb70/redis-cache"
	"github.com/rs/zerolog"
)

type Params struct {
	CacheClient    cache.RedisCache
	CacheAvailable bool
	Log            zerolog.Logger
	Config         *Config
}

func NewParams(cfg *Config) *Params {
	response := &Params{
		Config: cfg,
	}

	fmt.Println("Config: ", cfg)

	response.Log = zerolog.New(os.Stdout).With().Any("function", cfg.Name).Logger()

	if cfg.Redis.Enabled {
		cacheConfig := &cache.Config{
			RedisHost:     cfg.Redis.RedisHost,
			RedisPort:     cfg.Redis.RedisPort,
			RedisUsername: cfg.Redis.RedisUser,
			RedisPassword: cfg.Redis.RedisPass,
		}

		cacheClient, err := cache.New(cacheConfig)
		if err != nil {
			response.Log.Error().Msgf("Error creating cache client: %v", err)
			response.CacheAvailable = false
		} else {
			response.CacheClient = cacheClient
			response.CacheAvailable = true
		}
	}

	return response
}
