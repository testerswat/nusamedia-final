package store

import (
	"context"
	"database/sql"
	"github.com/jackc/pgx/v5/stdlib"
)

type Store struct{ DB *sql.DB }

func Open(ctx context.Context, url string) (*Store, error) {
	cfg, err := stdlib.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	db := stdlib.OpenDB(*cfg)
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	return &Store{DB: db}, nil
}
