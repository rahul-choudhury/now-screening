package movies

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"
	"time"
)

type fakeRepository struct {
	listFreshMovies []Movie
	listFreshErr    error
	hasFresh        bool
	hasFreshErr     error
	replaceErr      error

	replaceCalls int
	replacedCity string
	replacedAt   time.Time
	replacedWith []Movie
}

func (f *fakeRepository) ListFresh(_ context.Context, _ string, _ time.Time) ([]Movie, error) {
	if f.listFreshErr != nil {
		return nil, f.listFreshErr
	}

	return append([]Movie(nil), f.listFreshMovies...), nil
}

func (f *fakeRepository) HasFreshScrape(_ context.Context, _ string, _ time.Time) (bool, error) {
	if f.hasFreshErr != nil {
		return false, f.hasFreshErr
	}

	return f.hasFresh, nil
}

func (f *fakeRepository) ReplaceCity(_ context.Context, city string, list []Movie, scrapedAt time.Time) error {
	f.replaceCalls++
	f.replacedCity = city
	f.replacedAt = scrapedAt
	f.replacedWith = append([]Movie(nil), list...)

	return f.replaceErr
}

type fakeScraper struct {
	movies []Movie
	err    error
	calls  int
	city   string
}

func (f *fakeScraper) Scrape(_ context.Context, city string) ([]Movie, error) {
	f.calls++
	f.city = city

	if f.err != nil {
		return nil, f.err
	}

	return append([]Movie(nil), f.movies...), nil
}

func testLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

func TestMovieServiceLoadReturnsFreshCache(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		listFreshMovies: []Movie{{Title: "Cached", Href: "/cached"}},
		hasFresh:        true,
	}
	scraper := &fakeScraper{}
	service := NewMovieService(repo, scraper, 24*time.Hour, testLogger())

	got, fromCache, err := service.Load(context.Background(), "cuttack")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !fromCache {
		t.Fatal("Load() fromCache = false, want true")
	}

	if scraper.calls != 0 {
		t.Fatalf("Scrape() calls = %d, want 0", scraper.calls)
	}

	if repo.replaceCalls != 0 {
		t.Fatalf("ReplaceCity() calls = %d, want 0", repo.replaceCalls)
	}

	if len(got) != 1 || got[0].Title != "Cached" {
		t.Fatalf("Load() returned %+v, want cached movie", got)
	}
}

func TestMovieServiceLoadScrapesAndSavesOnCacheMiss(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	scraper := &fakeScraper{
		movies: []Movie{{Title: "Fresh", Href: "/fresh"}},
	}
	service := NewMovieService(repo, scraper, 24*time.Hour, testLogger())

	got, fromCache, err := service.Load(context.Background(), "bhubaneswar")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if fromCache {
		t.Fatal("Load() fromCache = true, want false")
	}

	if scraper.calls != 1 {
		t.Fatalf("Scrape() calls = %d, want 1", scraper.calls)
	}

	if scraper.city != "bhubaneswar" {
		t.Fatalf("Scrape() city = %q, want %q", scraper.city, "bhubaneswar")
	}

	if repo.replaceCalls != 1 {
		t.Fatalf("ReplaceCity() calls = %d, want 1", repo.replaceCalls)
	}

	if repo.replacedCity != "bhubaneswar" {
		t.Fatalf("ReplaceCity() city = %q, want %q", repo.replacedCity, "bhubaneswar")
	}

	if len(got) != 1 || got[0].Title != "Fresh" {
		t.Fatalf("Load() returned %+v, want scraped movies", got)
	}
}

func TestMovieServiceLoadReturnsScrapeError(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	scraper := &fakeScraper{err: errors.New("network down")}
	service := NewMovieService(repo, scraper, 24*time.Hour, testLogger())

	_, _, err := service.Load(context.Background(), "cuttack")
	if err == nil {
		t.Fatal("Load() error = nil, want scrape error")
	}

	if want := "scrape movies: network down"; err.Error() != want {
		t.Fatalf("Load() error = %q, want %q", err.Error(), want)
	}

	if repo.replaceCalls != 0 {
		t.Fatalf("ReplaceCity() calls = %d, want 0", repo.replaceCalls)
	}
}

func TestMovieServiceLoadRejectsEmptyScrape(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	scraper := &fakeScraper{}
	service := NewMovieService(repo, scraper, 24*time.Hour, testLogger())

	_, _, err := service.Load(context.Background(), "cuttack")
	if !errors.Is(err, errEmptyScrape) {
		t.Fatalf("Load() error = %v, want errEmptyScrape", err)
	}

	if repo.replaceCalls != 0 {
		t.Fatalf("ReplaceCity() calls = %d, want 0", repo.replaceCalls)
	}
}

func TestMovieServiceLoadReturnsMoviesWhenSaveFails(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{replaceErr: errors.New("write failed")}
	scraper := &fakeScraper{
		movies: []Movie{{Title: "Fresh", Href: "/fresh"}},
	}
	service := NewMovieService(repo, scraper, 24*time.Hour, testLogger())

	got, fromCache, err := service.Load(context.Background(), "cuttack")
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if fromCache {
		t.Fatal("Load() fromCache = true, want false")
	}

	if repo.replaceCalls != 1 {
		t.Fatalf("ReplaceCity() calls = %d, want 1", repo.replaceCalls)
	}

	if len(got) != 1 || got[0].Title != "Fresh" {
		t.Fatalf("Load() returned %+v, want scraped movies", got)
	}
}
