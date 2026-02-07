package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisOptions struct {
	URL       string
	KeyPrefix string
	TTL       time.Duration
}

type RedisStore struct {
	client    *redis.Client
	keyPrefix string
	ttl       time.Duration
}

func NewRedisStore(opts RedisOptions) (*RedisStore, error) {
	if opts.URL == "" {
		return nil, fmt.Errorf("redis url is required")
	}

	options, err := redis.ParseURL(opts.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis url: %w", err)
	}

	client := redis.NewClient(options)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &RedisStore{
		client:    client,
		keyPrefix: opts.KeyPrefix,
		ttl:       opts.TTL,
	}, nil
}

func (r *RedisStore) LoadPlayer(ctx context.Context, key string) (*PlayerState, error) {
	if key == "" {
		return nil, nil
	}
	payload, err := r.client.Get(ctx, r.prefixed(key)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var state PlayerState
	if err := json.Unmarshal([]byte(payload), &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *RedisStore) SavePlayer(ctx context.Context, key string, state *PlayerState) error {
	if key == "" || state == nil {
		return nil
	}

	state.UpdatedAt = time.Now().UTC()
	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, r.prefixed(key), payload, r.ttl).Err()
}

func (r *RedisStore) DeletePlayer(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	return r.client.Del(ctx, r.prefixed(key)).Err()
}

func (r *RedisStore) LoadWorld(ctx context.Context) (*WorldState, error) {
	payload, err := r.client.Get(ctx, r.worldKey()).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var state WorldState
	if err := json.Unmarshal([]byte(payload), &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *RedisStore) SaveWorld(ctx context.Context, state *WorldState) error {
	if state == nil {
		return nil
	}
	state.UpdatedAt = time.Now().UTC()
	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, r.worldKey(), payload, r.ttl).Err()
}

func (r *RedisStore) IsAdmin(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, nil
	}
	return r.client.SIsMember(ctx, r.adminsKey(), key).Result()
}

func (r *RedisStore) SetAdmin(ctx context.Context, key string, isAdmin bool) error {
	if key == "" {
		return nil
	}
	if isAdmin {
		return r.client.SAdd(ctx, r.adminsKey(), key).Err()
	}
	return r.client.SRem(ctx, r.adminsKey(), key).Err()
}

func (r *RedisStore) prefixed(key string) string {
	if r.keyPrefix == "" {
		return key
	}
	return r.keyPrefix + ":" + key
}

func (r *RedisStore) worldKey() string {
	return r.prefixed("world")
}

func (r *RedisStore) adminsKey() string {
	return r.prefixed("admins")
}
