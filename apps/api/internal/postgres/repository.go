package postgres

import (
	"context"
	"time"

	"go-scraping/internal/movies"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MovieRepository struct {
	pool *pgxpool.Pool
}

var _ movies.Repository = (*MovieRepository)(nil)

func NewMovieRepository(pool *pgxpool.Pool) *MovieRepository {
	return &MovieRepository{pool: pool}
}

func (r *MovieRepository) ListFresh(ctx context.Context, city string, since time.Time) ([]movies.Movie, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT title, href FROM movies
		WHERE city = $1 AND scraped_at > $2
		ORDER BY scraped_at DESC
	`, city, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []movies.Movie
	for rows.Next() {
		var movie movies.Movie
		if err := rows.Scan(&movie.Title, &movie.Href); err != nil {
			return nil, err
		}

		result = append(result, movie)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *MovieRepository) HasFreshScrape(ctx context.Context, city string, since time.Time) (bool, error) {
	var exists bool

	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM city_scrapes
			WHERE city = $1 AND scraped_at > $2
		)
	`, city, since).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (r *MovieRepository) ReplaceCity(ctx context.Context, city string, list []movies.Movie, scrapedAt time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `DELETE FROM movies WHERE city = $1`, city); err != nil {
		return err
	}

	for _, movie := range list {
		if _, err := tx.Exec(ctx, `
			INSERT INTO movies (city, title, href, scraped_at)
			VALUES ($1, $2, $3, $4)
		`, city, movie.Title, movie.Href, scrapedAt); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO city_scrapes (city, scraped_at)
		VALUES ($1, $2)
		ON CONFLICT (city) DO UPDATE SET scraped_at = EXCLUDED.scraped_at
	`, city, scrapedAt); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
