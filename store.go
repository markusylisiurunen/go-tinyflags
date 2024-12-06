package tinyflags

import "context"

type Store interface {
	Read(ctx context.Context, k string) ([]byte, error)
	Write(ctx context.Context, k string, v []byte) error
	Close() error
}
