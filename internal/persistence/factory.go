package persistence

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	envDriver        = "DMUD_PERSISTENCE"
	envRedisURL      = "DMUD_REDIS_URL"
	envRedisURL2     = "REDIS_URL"
	envKeyPrefix     = "DMUD_PERSIST_PREFIX"
	envTTL           = "DMUD_PERSIST_TTL"
	defaultRedisURL  = "redis://dmud:password@127.0.0.1:6379/0"
	defaultKeyPrefix = "dmud"
)

func NewStoreFromEnv() (Store, error) {
	driver := strings.ToLower(strings.TrimSpace(os.Getenv(envDriver)))
	if driver == "" || driver == "none" || driver == "off" {
		return &NoopStore{}, nil
	}

	config := StoreConfig{
		Driver:    driver,
		RedisURL:  firstNonEmpty(os.Getenv(envRedisURL), os.Getenv(envRedisURL2)),
		KeyPrefix: os.Getenv(envKeyPrefix),
		TTL:       0,
	}

	if config.RedisURL == "" {
		config.RedisURL = defaultRedisURL
	}
	if strings.TrimSpace(config.KeyPrefix) == "" {
		config.KeyPrefix = defaultKeyPrefix
	}

	if ttlRaw := strings.TrimSpace(os.Getenv(envTTL)); ttlRaw != "" {
		ttl, err := time.ParseDuration(ttlRaw)
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", envTTL, err)
		}
		config.TTL = ttl
	}

	return NewStore(config)
}

func NewStore(config StoreConfig) (Store, error) {
	switch strings.ToLower(strings.TrimSpace(config.Driver)) {
	case "redis":
		return NewRedisStore(RedisOptions{
			URL:       config.RedisURL,
			KeyPrefix: config.KeyPrefix,
			TTL:       config.TTL,
		})
	case "memory":
		return NewMemoryStore(), nil
	case "none", "off", "":
		return &NoopStore{}, nil
	default:
		return nil, fmt.Errorf("unknown persistence driver: %s", config.Driver)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
