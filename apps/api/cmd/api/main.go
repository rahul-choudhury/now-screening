package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-scraping/internal/bookmyshow"
	"go-scraping/internal/config"
	"go-scraping/internal/movies"
	"go-scraping/internal/postgres"
	"go-scraping/internal/web"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	if err := run(logger); err != nil {
		logger.Fatal(err)
	}
}

func run(logger *log.Logger) error {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	logger.Println("Connected to database...")

	repo := postgres.NewMovieRepository(pool)
	scraper := bookmyshow.NewScraper(cfg.ScrapeTimeout)
	service := movies.NewMovieService(repo, scraper, cfg.CacheTTL, logger)

	mux := http.NewServeMux()
	web.RegisterMovieRoutes(mux, service, cfg.DefaultCity, logger)

	server := &http.Server{
		Addr: cfg.ServerAddr,
		Handler: web.Chain(
			mux,
			web.CORSMiddleware(),
			web.RecoverMiddleware(logger),
			web.LoggingMiddleware(logger),
		),
	}

	listener, err := net.Listen("tcp", cfg.ServerAddr)
	if err != nil {
		return err
	}

	go func() {
		if err := service.Preload(ctx, cfg.PreloadCities); err != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("Initial movie preload completed with errors: %v", err)
		}
	}()

	serverErr := make(chan error, 1)
	go func() {
		logger.Printf("Server starting on %s...", cfg.ServerAddr)

		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}

		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}

		return <-serverErr
	}
}
