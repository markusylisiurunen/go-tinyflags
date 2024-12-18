package tinyflags

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
	ns     string
	ttl    time.Duration
}

type redisStoreOption func(*RedisStore)

func WithRedisStoreTTL(ttl time.Duration) redisStoreOption {
	return func(s *RedisStore) {
		s.ttl = ttl
	}
}

func NewRedisStore(client *redis.Client, ns string, opts ...redisStoreOption) *RedisStore {
	s := &RedisStore{client, ns, 5 * time.Minute}
	for _, apply := range opts {
		apply(s)
	}
	return s
}

func (s *RedisStore) Read(ctx context.Context, k string) ([]byte, error) {
	v, err := s.client.Get(ctx, s.key(ctx, k)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return v, err
}

func (s *RedisStore) Write(ctx context.Context, k string, v []byte) error {
	if v == nil {
		return s.client.Del(ctx, s.key(ctx, k)).Err()
	}
	return s.client.Set(ctx, s.key(ctx, k), v, s.ttl).Err()
}

func (s *RedisStore) Close() error {
	return nil
}

func (s *RedisStore) scope(_ context.Context, _ string) string {
	return "global"
}

func (s *RedisStore) key(ctx context.Context, k string) string {
	return strings.Join([]string{"tinyflags", "redisStore", s.ns, s.scope(ctx, k), k}, "::")
}
