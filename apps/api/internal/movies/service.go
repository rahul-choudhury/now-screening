package movies

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

type movieService struct {
	repo     Repository
	scraper  Scraper
	cacheTTL time.Duration
	logger   *log.Logger
}

var errEmptyScrape = errors.New("scrape returned no movies")

func NewMovieService(repo Repository, scraper Scraper, cacheTTL time.Duration, logger *log.Logger) Service {
	return &movieService{
		repo:     repo,
		scraper:  scraper,
		cacheTTL: cacheTTL,
		logger:   logger,
	}
}

func (s *movieService) Load(ctx context.Context, city string) ([]Movie, bool, error) {
	since := time.Now().Add(-s.cacheTTL)

	cachedMovies, err := s.repo.ListFresh(ctx, city, since)
	if err != nil {
		return nil, false, fmt.Errorf("query cached movies: %w", err)
	}

	cacheValid, err := s.repo.HasFreshScrape(ctx, city, since)
	if err != nil {
		return nil, false, fmt.Errorf("query cached movies: %w", err)
	}

	if cacheValid {
		return cachedMovies, true, nil
	}

	s.logger.Printf("No cached data for %s, scraping...", city)

	scrapedMovies, err := s.scraper.Scrape(ctx, city)
	if err != nil {
		return nil, false, fmt.Errorf("scrape movies: %w", err)
	}

	if len(scrapedMovies) == 0 {
		return nil, false, fmt.Errorf("scrape movies: %w", errEmptyScrape)
	}

	if err := s.repo.ReplaceCity(ctx, city, scrapedMovies, time.Now()); err != nil {
		s.logger.Printf("Failed to save movies for %s: %v", city, err)
	} else {
		s.logger.Printf("Saved %d movies to database for city: %s", len(scrapedMovies), city)
	}

	return scrapedMovies, false, nil
}

func (s *movieService) Preload(ctx context.Context, cities []string) error {
	s.logger.Println("Starting initial movie scraping for cities:", cities)

	var preloadErrs []error

	for _, city := range cities {
		if err := ctx.Err(); err != nil {
			return err
		}

		loadedMovies, fromCache, err := s.Load(ctx, city)
		if err != nil {
			s.logger.Printf("Failed to load movies for %s: %v", city, err)
			preloadErrs = append(preloadErrs, fmt.Errorf("%s: %w", city, err))
			continue
		}

		if fromCache {
			s.logger.Printf("Found %d cached movies for %s (within 24 hours), skipping scrape", len(loadedMovies), city)
			continue
		}

		s.logger.Printf("Successfully preloaded %d movies for %s", len(loadedMovies), city)
	}

	s.logger.Println("Initial movie scraping completed")

	return errors.Join(preloadErrs...)
}
