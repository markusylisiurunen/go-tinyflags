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
	scope text not null,
	key text not null,
	value jsonb not null
)
`

var queryCreateScopeKeyIndex = `
create unique index if not exists :SCHEMA_flags_scope_key_idx
on :SCHEMA.flags (scope, key)
`

var queryCreateScopeKeyValueIndex = `
create index if not exists :SCHEMA_flags_scope_key_value_idx
on :SCHEMA.flags (scope, key, value)
`

var queryReadFlag = `
select value
from :SCHEMA.flags
where scope = $1 and key = $2
`

var queryUpsertFlag = `
insert into :SCHEMA.flags (scope, key, value)
values ($1, $2, $3)
on conflict (scope, key) do update set
	value = $3,
	updated_at = now()
`

var queryDeleteFlag = `
delete from :SCHEMA.flags
where scope = $1 and key = $2
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
	row := s.client.QueryRowContext(ctx, strings.ReplaceAll(queryReadFlag, ":SCHEMA", s.schema), s.scope(ctx, k), k)
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
	scope := s.scope(ctx, k)
	if v == nil {
		_, err := s.client.ExecContext(ctx, strings.ReplaceAll(queryDeleteFlag, ":SCHEMA", s.schema), scope, k)
		return err
	}
	_, err := s.client.ExecContext(ctx, strings.ReplaceAll(queryUpsertFlag, ":SCHEMA", s.schema), scope, k, v)
	return err
}

func (s *PostgresStore) Close() error {
	return nil
}

func (s *PostgresStore) scope(_ context.Context, _ string) string {
	return "global"
}

func (s *PostgresStore) migrate(ctx context.Context) error {
	s.migrateOnce.Do(func() {
		tx, err := s.client.BeginTx(ctx, nil)
		if err != nil {
			s.migrateErr = err
			return
		}
		defer tx.Rollback() //nolint:errcheck
		queries := []string{
			queryCreateSchema,
			queryCreateTable,
			queryCreateScopeKeyIndex,
			queryCreateScopeKeyValueIndex,
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
