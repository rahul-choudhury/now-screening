package postgres

import (
	"context"

	"go-scraping/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, cfg.ConnectionString())
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	if err := ensureSchema(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func ensureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	queries := []string{
		`
			CREATE TABLE IF NOT EXISTS city_scrapes (
				city VARCHAR(100) PRIMARY KEY,
				scraped_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			)
		`,
		`CREATE INDEX IF NOT EXISTS idx_city_scrapes_scraped_at ON city_scrapes(scraped_at)`,
	}

	for _, query := range queries {
		if _, err := pool.Exec(ctx, query); err != nil {
			return err
		}
	}

	return nil
}
