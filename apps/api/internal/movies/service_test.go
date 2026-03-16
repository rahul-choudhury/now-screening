package movies

import (
	"context"
	"errors"
	"io"
	"log"
	"sync"
	"testing"
	"time"
)

type fakeRepository struct {
	mu sync.Mutex

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
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.listFreshErr != nil {
		return nil, f.listFreshErr
	}

	return append([]Movie(nil), f.listFreshMovies...), nil
}

func (f *fakeRepository) HasFreshScrape(_ context.Context, _ string, _ time.Time) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.hasFreshErr != nil {
		return false, f.hasFreshErr
	}

	return f.hasFresh, nil
}

func (f *fakeRepository) ReplaceCity(_ context.Context, city string, list []Movie, scrapedAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.replaceCalls++
	f.replacedCity = city
	f.replacedAt = scrapedAt
	f.replacedWith = append([]Movie(nil), list...)

	if f.replaceErr != nil {
		return f.replaceErr
	}

	f.listFreshMovies = append([]Movie(nil), list...)
	f.hasFresh = true

	return f.replaceErr
}

type fakeScraper struct {
	mu sync.Mutex

	movies []Movie
	err    error
	calls  int
	city   string

	started chan struct{}
	release <-chan struct{}
}

func (f *fakeScraper) Scrape(_ context.Context, city string) ([]Movie, error) {
	f.mu.Lock()
	f.calls++
	f.city = city
	movies := append([]Movie(nil), f.movies...)
	err := f.err
	started := f.started
	release := f.release
	f.mu.Unlock()

	if started != nil {
		select {
		case started <- struct{}{}:
		default:
		}
	}

	if release != nil {
		<-release
	}

	if err != nil {
		return nil, err
	}

	return movies, nil
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

func TestMovieServiceLoadCoalescesConcurrentScrapes(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	release := make(chan struct{})
	scraper := &fakeScraper{
		movies:  []Movie{{Title: "Fresh", Href: "/fresh"}},
		started: make(chan struct{}, 1),
		release: release,
	}
	service := NewMovieService(repo, scraper, 24*time.Hour, testLogger())

	type result struct {
		movies    []Movie
		fromCache bool
		err       error
	}

	results := make(chan result, 2)

	go func() {
		movies, fromCache, err := service.Load(context.Background(), "cuttack")
		results <- result{movies: movies, fromCache: fromCache, err: err}
	}()

	<-scraper.started

	go func() {
		movies, fromCache, err := service.Load(context.Background(), "cuttack")
		results <- result{movies: movies, fromCache: fromCache, err: err}
	}()

	close(release)

	first := <-results
	second := <-results

	if first.err != nil {
		t.Fatalf("first Load() error = %v", first.err)
	}

	if second.err != nil {
		t.Fatalf("second Load() error = %v", second.err)
	}

	if scraper.calls != 1 {
		t.Fatalf("Scrape() calls = %d, want 1", scraper.calls)
	}

	if repo.replaceCalls != 1 {
		t.Fatalf("ReplaceCity() calls = %d, want 1", repo.replaceCalls)
	}

	if len(first.movies) != 1 || first.movies[0].Title != "Fresh" {
		t.Fatalf("first Load() movies = %+v, want scraped movie", first.movies)
	}

	if len(second.movies) != 1 || second.movies[0].Title != "Fresh" {
		t.Fatalf("second Load() movies = %+v, want scraped movie", second.movies)
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
