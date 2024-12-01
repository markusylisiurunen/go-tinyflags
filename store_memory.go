package tinyflags

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type memoryStoreValue struct {
	value   []byte
	expires time.Time
}

type MemoryStore struct {
	active bool
	client *redis.Client
	mu     sync.RWMutex
	pubsub *redis.PubSub
	ttl    time.Duration
	values map[string]memoryStoreValue
}

type memoryStoreOption func(*MemoryStore)

func WithMemoryStoreTTL(ttl time.Duration) memoryStoreOption {
	return func(s *MemoryStore) {
		s.ttl = ttl
	}
}

func NewMemoryStore(client *redis.Client, opts ...memoryStoreOption) *MemoryStore {
	s := &MemoryStore{client: client, ttl: 1 * time.Minute, values: make(map[string]memoryStoreValue)}
	for _, apply := range opts {
		apply(s)
	}
	s.listen()
	return s
}

func (s *MemoryStore) Read(ctx context.Context, k string) ([]byte, error) {
	s.mu.RLock()
	if !s.active {
		s.mu.RUnlock()
		return nil, nil
	}
	k = s.key(ctx, k)
	if v, ok := s.values[k]; ok {
		if v.expires.Before(time.Now()) {
			s.mu.RUnlock()
			s.mu.Lock()
			delete(s.values, k)
			s.mu.Unlock()
			return nil, nil
		}
		s.mu.RUnlock()
		return v.value, nil
	}
	s.mu.RUnlock()
	return nil, nil
}

func (s *MemoryStore) Write(ctx context.Context, k string, v []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		return nil
	}
	k = s.key(ctx, k)
	if v == nil {
		delete(s.values, k)
		s.invalidate(k) // nolint:errcheck
		return nil
	}
	s.values[k] = memoryStoreValue{v, time.Now().Add(s.ttl)}
	s.invalidate(k) // nolint:errcheck
	return nil
}

func (s *MemoryStore) scope(_ context.Context, _ string) string {
	return "global"
}

func (s *MemoryStore) key(ctx context.Context, k string) string {
	return s.scope(ctx, k) + "::" + k
}

func (s *MemoryStore) activate() {
	s.mu.Lock()
	s.active = true
	s.mu.Unlock()
}

func (s *MemoryStore) deactivate() {
	s.mu.Lock()
	s.active = false
	s.values = make(map[string]memoryStoreValue)
	s.mu.Unlock()
}

func (s *MemoryStore) invalidationsChannelName() string {
	key := strings.Join([]string{"tinyflags", "memoryStore", "invalidations"}, "::")
	return key
}

func (s *MemoryStore) invalidate(k string) error {
	ctx := context.Background()
	err := s.client.Publish(ctx, s.invalidationsChannelName(), k).Err()
	if err != nil {
		logger.Errorf(ctx, "failed to invalidate '%s': %v", k, err)
	}
	return err
}

func (s *MemoryStore) listen() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	go func() {
		ctx := context.Background()
		for {
			if err := s.subscribe(); err != nil {
				s.deactivate()
				if err := s.client.Ping(ctx).Err(); err != nil {
					logger.Errorf(ctx, "stopped listening for invalidations because ping failed: %v", err)
					return
				}
				logger.Errorf(ctx, "listening for invalidations returned an error, retrying in ~1s: %v", err)
				delay := 1000
				time.Sleep(time.Duration(delay/2+r.Intn(delay)) * time.Millisecond)
				continue
			}
		}
	}()
}

func (s *MemoryStore) subscribe() error {
	if s.pubsub != nil {
		s.pubsub.Close() // nolint:errcheck
	}
	ctx := context.Background()
	s.mu.Lock()
	s.pubsub = s.client.Subscribe(ctx, s.invalidationsChannelName())
	s.mu.Unlock()
	if _, err := s.pubsub.Receive(ctx); err != nil {
		return err
	}
	c := s.pubsub.Channel()
	s.activate()
	for msg := range c {
		s.mu.Lock()
		delete(s.values, msg.Payload)
		s.mu.Unlock()
	}
	return errors.New("subscription closed")
}
