package redis

import (
	"context"
	"fmt"
	infrastructure "github.com/qpubio/qpub-server/internal/config/infrastructure"
	"time"

	"github.com/go-redis/redis"
)

type Service interface {
	Ping(ctx context.Context) error
	Get(key string) *redis.StringCmd
	Set(key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	SetNX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Del(keys ...string) *redis.IntCmd
	TTL(key string) *redis.DurationCmd
	Keys(pattern string) *redis.StringSliceCmd
	Close() error
	Client() *redis.Client
}

type service struct {
	client *redis.Client
	config *infrastructure.Redis
}

func New(config *infrastructure.Redis) (Service, error) {
	if config == nil {
		return nil, fmt.Errorf("redis config cannot be nil")
	}

	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.Host, config.Port),
		DB:       config.Options.DB,
		Password: config.Options.Password,

		// Pool configuration
		PoolSize:     config.Pool.MaxActive,
		MinIdleConns: config.Pool.MinIdle,
		IdleTimeout:  config.Pool.IdleTimeout,

		// Timeouts
		DialTimeout:  config.Options.Timeout,
		ReadTimeout:  config.Options.Timeout,
		WriteTimeout: config.Options.Timeout,
	}

	client := redis.NewClient(opts)

	return &service{
		client: client,
		config: config,
	}, nil
}

func (s *service) Ping(ctx context.Context) error {
	return s.client.Ping().Err()
}

func (s *service) Get(key string) *redis.StringCmd {
	return s.client.Get(key)
}

func (s *service) Set(key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return s.client.Set(key, value, expiration)
}

func (s *service) SetNX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	return s.client.SetNX(key, value, expiration)
}

func (s *service) Del(keys ...string) *redis.IntCmd {
	return s.client.Del(keys...)
}

func (s *service) TTL(key string) *redis.DurationCmd {
	return s.client.TTL(key)
}

func (s *service) Keys(pattern string) *redis.StringSliceCmd {
	return s.client.Keys(pattern)
}

func (s *service) Close() error {
	return s.client.Close()
}

func (s *service) Client() *redis.Client {
	return s.client
}
