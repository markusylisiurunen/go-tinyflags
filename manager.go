package tinyflags

import (
	"context"
	"fmt"
	"reflect"
)

type Manager struct {
	stores []Store
}

func New(stores ...Store) *Manager {
	m := &Manager{make([]Store, 0, len(stores))}
	m.stores = append(m.stores, stores...)
	return m
}

func (m *Manager) Read(ctx context.Context, flags ...any) error {
	if len(flags) == 0 {
		return nil
	}
	flaggers := make([]flagger, 0, len(flags))
	for _, flag := range flags {
		v := reflect.ValueOf(flag)
		if v.Kind() != reflect.Ptr {
			return fmt.Errorf("flag must be a pointer to struct, got %T", flag)
		}
		if v.Elem().Kind() != reflect.Struct {
			return fmt.Errorf("flag must be a pointer to struct, got pointer to %T", v.Elem().Interface())
		}
		if _, ok := flag.(flagger); !ok {
			return fmt.Errorf("flag must implement flagger interface: %T", flag)
		}
		flaggers = append(flaggers, flag.(flagger))
	}
	remaining := make(map[int]bool)
	for idx := range flaggers {
		remaining[idx] = true
	}
	for idx, store := range m.stores {
		if len(remaining) == 0 {
			break
		}
		type indexed[T any] struct {
			index int
			value T
		}
		var current []indexed[flagger]
		for idx := range remaining {
			current = append(current, indexed[flagger]{idx, flaggers[idx]})
		}
		for _, flag := range current {
			b, err := store.Read(ctx, flag.value.key())
			if err != nil {
				return err
			}
			if b != nil {
				if err := flag.value.absorb(b); err != nil {
					return err
				}
				delete(remaining, flag.index)
				for i := idx - 1; i >= 0; i-- {
					if err := m.stores[i].Write(ctx, flag.value.key(), b); err != nil {
						logger.Errorf(ctx, "failed to write flag %s to store %T at index %d: %v", flag.value.key(), m.stores[i], i, err)
					}
				}
			}
		}
	}
	return nil
}

func (m *Manager) Write(ctx context.Context, flags ...any) error {
	if len(flags) == 0 {
		return nil
	}
	flaggers := make([]flagger, 0, len(flags))
	for _, flag := range flags {
		v := reflect.ValueOf(flag)
		if v.Kind() != reflect.Ptr {
			return fmt.Errorf("flag must be a pointer to struct, got %T", flag)
		}
		if v.Elem().Kind() != reflect.Struct {
			return fmt.Errorf("flag must be a pointer to struct, got pointer to %T", v.Elem().Interface())
		}
		if _, ok := flag.(flagger); !ok {
			return fmt.Errorf("flag must implement flagger interface: %T", flag)
		}
		flaggers = append(flaggers, flag.(flagger))
	}
	values := make([][]byte, 0, len(flaggers))
	for _, flag := range flaggers {
		b, err := flag.emit()
		if err != nil {
			return err
		}
		values = append(values, b)
	}
	var lastErr error
	for i := len(m.stores) - 1; i >= 0; i-- {
		store := m.stores[i]
		for idx, flag := range flaggers {
			b := values[idx]
			if err := store.Write(ctx, flag.key(), b); err != nil {
				logger.Errorf(ctx, "failed to write flag %s to store %T at index %d: %v", flag.key(), store, i, err)
				lastErr = err
			}
		}
	}
	return lastErr
}

func (m *Manager) Close() error {
	var lastErr error
	for idx := len(m.stores) - 1; idx >= 0; idx-- {
		if err := m.stores[idx].Close(); err != nil {
			logger.Errorf(context.Background(), "failed to close store %T at index %d: %v", m.stores[idx], idx, err)
			lastErr = err
		}
	}
	return lastErr
}
