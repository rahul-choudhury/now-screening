package web

import (
	"fmt"
	"log"
	"net/http"

	"go-scraping/internal/movies"
)

type MoviesHandler struct {
	service     movies.Service
	defaultCity string
	logger      *log.Logger
}

func RegisterMovieRoutes(mux *http.ServeMux, service movies.Service, defaultCity string, logger *log.Logger) {
	handler := &MoviesHandler{
		service:     service,
		defaultCity: defaultCity,
		logger:      logger,
	}

	mux.Handle("GET /movies", http.HandlerFunc(handler.GetMovies))
	mux.Handle("OPTIONS /movies", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		WriteNoContent(w, http.StatusNoContent)
	}))
}

func (h *MoviesHandler) GetMovies(w http.ResponseWriter, r *http.Request) {
	city := r.URL.Query().Get("city")
	if city == "" {
		city = h.defaultCity
	}

	query := r.URL.Query().Get("query")

	loadedMovies, fromCache, err := h.service.Load(r.Context(), city)
	if err != nil {
		h.logger.Printf("Error loading movies for %s: %v", city, err)
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to load movies: %v", err))
		return
	}

	if fromCache {
		h.logger.Printf("Returning %d cached movies for city: %s", len(loadedMovies), city)
	}

	if query != "" {
		loadedMovies = movies.FuzzySearch(loadedMovies, movies.NormalizeQuery(query))
	}

	WriteJSON(w, http.StatusOK, movies.Response{
		City:   city,
		Movies: loadedMovies,
		Count:  len(loadedMovies),
	})
}
