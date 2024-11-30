package tinyflags

import (
	"context"
	"encoding/json"
	"sync"
)

type ConstantStore struct {
	mu     sync.RWMutex
	values map[string][]byte
}

func NewConstantStore() *ConstantStore {
	return &ConstantStore{values: make(map[string][]byte)}
}

func (s *ConstantStore) With(k string, v any) *ConstantStore {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	s.values[k] = b
	return s
}

func (s *ConstantStore) Read(_ context.Context, k string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.values[k], nil
}

func (s *ConstantStore) Write(_ context.Context, k string, v []byte) error {
	return nil
}
