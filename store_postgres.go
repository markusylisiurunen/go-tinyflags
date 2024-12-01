package tinyflags

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"

	_ "github.com/lib/pq"
)

var queryCreateSchema = `
create schema if not exists :SCHEMA
`

var queryCreateTable = `
create table if not exists :SCHEMA.flags (
	id bigserial primary key,
	created_at timestamptz not null default now(),
	updated_at timestamptz,
	key text not null,
	value jsonb not null
)
`

var queryCreateKeyIndex = `
create unique index if not exists :SCHEMA_flags_key_idx
on :SCHEMA.flags (key)
`

var queryCreateKeyValueIndex = `
create index if not exists :SCHEMA_flags_key_value_idx
on :SCHEMA.flags (key, value)
`

var queryReadFlag = `
select value
from :SCHEMA.flags
where key = $1
`

var queryUpsertFlag = `
insert into :SCHEMA.flags (key, value)
values ($1, $2)
on conflict (key) do update set
	value = $2,
	updated_at = now()
returning id
`

var queryDeleteFlag = `
delete from :SCHEMA.flags
where key = $1
`

type PostgresStore struct {
	client *sql.DB
	schema string

	migrateOnce sync.Once
	migrateErr  error
}

type postgresStoreOption func(*PostgresStore)

func WithPostgresStoreSchema(schema string) postgresStoreOption {
	return func(s *PostgresStore) {
		s.schema = schema
	}
}

func NewPostgresStore(client *sql.DB, opts ...postgresStoreOption) *PostgresStore {
	s := &PostgresStore{client: client, schema: "public"}
	for _, apply := range opts {
		apply(s)
	}
	return s
}

func (s *PostgresStore) Read(ctx context.Context, k string) ([]byte, error) {
	if err := s.migrate(ctx); err != nil {
		return nil, err
	}
	var v []byte
	row := s.client.QueryRowContext(ctx, strings.ReplaceAll(queryReadFlag, ":SCHEMA", s.schema), k)
	if err := row.Scan(&v); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return v, nil
}

func (s *PostgresStore) Write(ctx context.Context, k string, v []byte) error {
	if err := s.migrate(ctx); err != nil {
		return err
	}
	if v == nil {
		_, err := s.client.ExecContext(ctx, strings.ReplaceAll(queryDeleteFlag, ":SCHEMA", s.schema), k)
		return err
	}
	_, err := s.client.ExecContext(ctx, strings.ReplaceAll(queryUpsertFlag, ":SCHEMA", s.schema), k, v)
	return err
}

func (s *PostgresStore) migrate(ctx context.Context) error {
	s.migrateOnce.Do(func() {
		tx, err := s.client.BeginTx(ctx, nil)
		if err != nil {
			s.migrateErr = err
			return
		}
		defer tx.Rollback() // nolint:errcheck
		queries := []string{
			queryCreateSchema,
			queryCreateTable,
			queryCreateKeyIndex,
			queryCreateKeyValueIndex,
		}
		for _, query := range queries {
			_, err = tx.Exec(strings.ReplaceAll(query, ":SCHEMA", s.schema))
			if err != nil {
				s.migrateErr = err
				return
			}
		}
		err = tx.Commit()
		if err != nil {
			s.migrateErr = err
			return
		}
	})
	return s.migrateErr
}
