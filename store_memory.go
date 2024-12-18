package tinyflags

import (
	"context"
	crand "crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	mrand "math/rand"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type memoryStoreValue struct {
	value   []byte
	hash    string
	expires time.Time
}

type MemoryStore struct {
	id        string
	client    *redis.Client
	pubsub    *redis.PubSub
	mu        sync.RWMutex
	ttl       time.Duration
	values    map[string]memoryStoreValue
	isActive  bool
	isClosed  bool
	closeOnce sync.Once
	done      chan struct{}
}

type memoryStoreOption func(*MemoryStore)

func WithMemoryStoreTTL(ttl time.Duration) memoryStoreOption {
	return func(s *MemoryStore) {
		s.ttl = ttl
	}
}

func NewMemoryStore(client *redis.Client, opts ...memoryStoreOption) *MemoryStore {
	b := make([]byte, 16)
	if _, err := crand.Read(b); err != nil {
		panic(err)
	}
	s := &MemoryStore{
		id:        hex.EncodeToString(b),
		client:    client,
		pubsub:    nil,
		mu:        sync.RWMutex{},
		ttl:       1 * time.Minute,
		values:    make(map[string]memoryStoreValue),
		isActive:  false,
		isClosed:  false,
		closeOnce: sync.Once{},
		done:      make(chan struct{}),
	}
	for _, apply := range opts {
		apply(s)
	}
	s.listen()
	s.cleanup()
	return s
}

func (s *MemoryStore) Read(ctx context.Context, k string) ([]byte, error) {
	s.mu.RLock()
	if s.isClosed || !s.isActive {
		s.mu.RUnlock()
		return nil, nil
	}
	k = s.getKey(k)
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
	if s.isClosed || !s.isActive {
		return nil
	}
	k = s.getKey(k)
	if v == nil {
		delete(s.values, k)
		s.triggerInvalidation("", k)
		return nil
	}
	h := sha1.New()
	if _, err := h.Write(v); err != nil {
		return err
	}
	hash := hex.EncodeToString(h.Sum(nil))
	s.values[k] = memoryStoreValue{v, hash, time.Now().Add(s.ttl)}
	s.triggerInvalidation(hash, k)
	return nil
}

func (s *MemoryStore) Close() error {
	var err error
	s.closeOnce.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if !s.isClosed {
			logger.Debugf("closing memory store")
			close(s.done)
			s.isClosed = true
		}
		if s.pubsub != nil {
			logger.Debugf("closing an active pubsub connection")
			err = s.pubsub.Close()
		}
	})
	return err
}

func (s *MemoryStore) getKey(k string) string {
	return strings.Join([]string{"global", k}, "::")
}
func (s *MemoryStore) getInvalidationsChannel() string {
	return strings.Join([]string{"tinyflags", "memoryStore", "invalidations"}, "::")
}

func (s *MemoryStore) triggerInvalidation(hash, key string) {
	ctx := context.Background()
	err := s.client.Publish(ctx, s.getInvalidationsChannel(), fmt.Sprintf("%s:%s:%s", s.id, hash, key)).Err()
	if err != nil {
		logger.Errorf(ctx, "failed to invalidate '%s': %v", key, err)
	}
}

func (s *MemoryStore) listen() {
	ctx := context.Background()
	r := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	go func() {
		for {
			err := s.subscribe()
			s.mu.Lock()
			s.isActive = false
			s.values = make(map[string]memoryStoreValue)
			s.mu.Unlock()
			if err == nil {
				logger.Debugf("subscribe returned without an error")
				return
			}
			if err := s.client.Ping(ctx).Err(); err != nil {
				logger.Errorf(ctx, "redis client ping returned an error, retrying in ~10s: %v", err)
				delay, jitter := 10000, 2000
				time.Sleep(time.Duration(delay-(jitter/2)+r.Intn(jitter)) * time.Millisecond)
			} else {
				logger.Errorf(ctx, "listening for invalidations returned an error, retrying in ~1s: %v", err)
				delay, jitter := 1000, 500
				time.Sleep(time.Duration(delay-(jitter/2)+r.Intn(jitter)) * time.Millisecond)
			}
		}
	}()
}

func (s *MemoryStore) subscribe() error {
	select {
	case <-s.done:
		return nil
	default:
	}
	logger.Debugf("subscribing to key invalidations")
	ctx := context.Background()
	s.mu.Lock()
	if s.pubsub != nil {
		logger.Debugf("closing an existing pubsub connection")
		s.pubsub.Close() //nolint:errcheck
	}
	s.pubsub = s.client.Subscribe(ctx, s.getInvalidationsChannel())
	s.mu.Unlock()
	if _, err := s.pubsub.Receive(ctx); err != nil {
		return err
	}
	c := s.pubsub.Channel()
	s.mu.Lock()
	s.isActive = true
	s.mu.Unlock()
	for {
		select {
		case <-s.done:
			s.pubsub.Close() //nolint:errcheck
			return nil
		case msg, ok := <-c:
			if !ok {
				return errors.New("subscription closed")
			}
			parts := strings.SplitN(msg.Payload, ":", 3)
			if len(parts) != 3 || parts[0] == s.id {
				logger.Debugf("skipping invalidation for '%s'", msg.Payload)
				continue
			}
			s.mu.Lock()
			hash, key := parts[1], parts[2]
			if v, ok := s.values[key]; ok && v.hash != hash {
				logger.Debugf("invalidating '%s'", key)
				delete(s.values, key)
			}
			s.mu.Unlock()
		}
	}
}

func (s *MemoryStore) cleanup() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				s.mu.Lock()
				if s.isClosed {
					s.mu.Unlock()
					continue
				}
				now := time.Now()
				for k, v := range s.values {
					if v.expires.Before(now) {
						delete(s.values, k)
					}
				}
				s.mu.Unlock()
			}
		}
	}()
}
