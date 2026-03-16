package movies

import (
	"context"
	"time"
)

type Repository interface {
	ListFresh(ctx context.Context, city string, since time.Time) ([]Movie, error)
	HasFreshScrape(ctx context.Context, city string, since time.Time) (bool, error)
	ReplaceCity(ctx context.Context, city string, movies []Movie, scrapedAt time.Time) error
}

type Scraper interface {
	Scrape(ctx context.Context, city string) ([]Movie, error)
}

type Service interface {
	Load(ctx context.Context, city string) ([]Movie, bool, error)
	Preload(ctx context.Context, cities []string) error
}
