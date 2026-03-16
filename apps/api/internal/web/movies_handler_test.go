package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-scraping/internal/movies"
)

type fakeMoviesService struct {
	loadMovies []movies.Movie
	fromCache  bool
	err        error
	loadCity   string
	loadCalls  int
}

func (f *fakeMoviesService) Load(_ context.Context, city string) ([]movies.Movie, bool, error) {
	f.loadCalls++
	f.loadCity = city

	if f.err != nil {
		return nil, false, f.err
	}

	return append([]movies.Movie(nil), f.loadMovies...), f.fromCache, nil
}

func testHandler(t *testing.T, service movieLoader) http.Handler {
	t.Helper()

	mux := http.NewServeMux()
	logger := log.New(io.Discard, "", 0)
	RegisterMovieRoutes(mux, service, "cuttack", logger)

	return Chain(mux, CORSMiddleware())
}

func decodeResponse(t *testing.T, recorder *httptest.ResponseRecorder) movies.Response {
	t.Helper()

	var payload movies.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	return payload
}

func TestGetMoviesUsesDefaultCity(t *testing.T) {
	t.Parallel()

	service := &fakeMoviesService{
		loadMovies: []movies.Movie{{Title: "Cached", Href: "/cached"}},
	}

	req := httptest.NewRequest(http.MethodGet, "/movies", nil)
	recorder := httptest.NewRecorder()

	testHandler(t, service).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	if service.loadCity != "cuttack" {
		t.Fatalf("Load() city = %q, want %q", service.loadCity, "cuttack")
	}
}

func TestGetMoviesFiltersResultsForQuery(t *testing.T) {
	t.Parallel()

	service := &fakeMoviesService{
		loadMovies: []movies.Movie{
			{Title: "Interstellar", Href: "/interstellar"},
			{Title: "Ballerina", Href: "/ballerina"},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/movies?city=bhubaneswar&query=Interstellar", nil)
	recorder := httptest.NewRecorder()

	testHandler(t, service).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	payload := decodeResponse(t, recorder)
	if payload.Count != 1 {
		t.Fatalf("count = %d, want %d", payload.Count, 1)
	}

	if len(payload.Movies) != 1 || payload.Movies[0].Title != "Interstellar" {
		t.Fatalf("movies = %+v, want only Interstellar", payload.Movies)
	}
}

func TestGetMoviesReturnsSuccessPayloadShape(t *testing.T) {
	t.Parallel()

	service := &fakeMoviesService{
		loadMovies: []movies.Movie{{Title: "Ballerina", Href: "/ballerina"}},
	}

	req := httptest.NewRequest(http.MethodGet, "/movies?city=bhubaneswar", nil)
	recorder := httptest.NewRecorder()

	testHandler(t, service).ServeHTTP(recorder, req)

	payload := decodeResponse(t, recorder)
	if payload.City != "bhubaneswar" {
		t.Fatalf("city = %q, want %q", payload.City, "bhubaneswar")
	}

	if payload.Count != 1 {
		t.Fatalf("count = %d, want %d", payload.Count, 1)
	}

	if len(payload.Movies) != 1 || payload.Movies[0].Href != "/ballerina" {
		t.Fatalf("movies = %+v, want Ballerina payload", payload.Movies)
	}
}

func TestGetMoviesReturnsErrorPayload(t *testing.T) {
	t.Parallel()

	service := &fakeMoviesService{err: errors.New("boom")}
	req := httptest.NewRequest(http.MethodGet, "/movies?city=cuttack", nil)
	recorder := httptest.NewRecorder()

	testHandler(t, service).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}

	var payload map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if payload["error"] != "Failed to load movies: boom" {
		t.Fatalf("error payload = %q, want %q", payload["error"], "Failed to load movies: boom")
	}
}

func TestOptionsMoviesReturnsCORSHeaders(t *testing.T) {
	t.Parallel()

	service := &fakeMoviesService{}
	req := httptest.NewRequest(http.MethodOptions, "/movies", nil)
	recorder := httptest.NewRecorder()

	testHandler(t, service).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}

	if got := recorder.Header().Get("Access-Control-Allow-Methods"); got != "GET, OPTIONS" {
		t.Fatalf("Access-Control-Allow-Methods = %q, want %q", got, "GET, OPTIONS")
	}

	if got := recorder.Header().Get("Access-Control-Allow-Headers"); got != "Origin, Content-Type" {
		t.Fatalf("Access-Control-Allow-Headers = %q, want %q", got, "Origin, Content-Type")
	}
}
