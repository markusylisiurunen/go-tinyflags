package tinyflags

import (
	"encoding/json"
	"errors"
)

type (
	BoolFlag    = Flag[bool]
	Float32Flag = Flag[float32]
	Float64Flag = Flag[float64]
	Int32Flag   = Flag[int32]
	Int64Flag   = Flag[int64]
	IntFlag     = Flag[int]
	StringFlag  = Flag[string]
)

func NewBoolFlag(k string) BoolFlag       { return NewFlag[bool](k) }
func NewFloat32Flag(k string) Float32Flag { return NewFlag[float32](k) }
func NewFloat64Flag(k string) Float64Flag { return NewFlag[float64](k) }
func NewInt32Flag(k string) Int32Flag     { return NewFlag[int32](k) }
func NewInt64Flag(k string) Int64Flag     { return NewFlag[int64](k) }
func NewIntFlag(k string) IntFlag         { return NewFlag[int](k) }
func NewStringFlag(k string) StringFlag   { return NewFlag[string](k) }

type flagger interface {
	key() string
	emit() ([]byte, error)
	absorb([]byte) error
}

type _flag struct {
	i bool
	k string
}

func (f _flag) key() string {
	return f.k
}

type Flag[V any] struct {
	_flag
	v V
}

func NewFlag[V any](k string) Flag[V] {
	return Flag[V]{_flag: _flag{k: k}}
}

func (f Flag[V]) With(v V) Flag[V] {
	f.i = true
	f.v = v
	return f
}

func (f *Flag[V]) Get() V {
	return f.v
}

func (f *Flag[V]) Set(v V) {
	f.i = true
	f.v = v
}

func (f *Flag[V]) emit() ([]byte, error) {
	if !f.i {
		return nil, errors.New("tried to write an unset flag; use With() or Set() to set a value first")
	}
	return json.Marshal(f.v)
}

func (f *Flag[V]) absorb(b []byte) error {
	if err := json.Unmarshal(b, &f.v); err != nil {
		return err
	}
	f.i = true
	return nil
}
