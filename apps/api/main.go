package main

import (
	"context"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sahilm/fuzzy"
)

var db *pgxpool.Pool

func main() {
	if err := connectDB(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(cors.Default())

	r.GET("/movies", getMovies)

	go preloadMovies()

	fmt.Println("Server starting on :8080...")

	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func connectDB() error {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "password")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s", dbUser, dbPassword, dbHost, dbPort)

	var err error
	db, err = pgxpool.New(context.Background(), connStr)
	if err != nil {
		return err
	}

	if err = db.Ping(context.Background()); err != nil {
		return err
	}

	if err = ensureSchema(context.Background()); err != nil {
		return err
	}

	log.Println("Connected to database...")
	return nil
}

func ensureSchema(ctx context.Context) error {
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
		if _, err := db.Exec(ctx, query); err != nil {
			return err
		}
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func cleanQuery(query string) string {
	htmlDecoded := html.UnescapeString(query)
	// NOTE: Converts non-breaking spaces (UTF-8 \u00a0) to regular spaces for fuzzy matching.
	// Query URLs like "How%20to%20Train%20Your%26nbsp%3BDragon" become "How to Train Your Dragon"
	// with UTF-8 non-breaking space after HTML decoding, but DB titles use regular ASCII spaces.
	cleaned := strings.ReplaceAll(htmlDecoded, "\u00a0", " ")
	return strings.TrimSpace(cleaned)
}

type Movie struct {
	Title string `json:"title"`
	Href  string `json:"href"`
}

func getMoviesFromDB(city string) ([]Movie, bool, error) {
	query := `
		SELECT title, href FROM movies
		WHERE city = $1 AND scraped_at > NOW() - INTERVAL '24 hours'
		ORDER BY scraped_at DESC
	`

	rows, err := db.Query(context.Background(), query, city)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var movies []Movie
	for rows.Next() {
		var movie Movie
		if err := rows.Scan(&movie.Title, &movie.Href); err != nil {
			return nil, false, err
		}
		movies = append(movies, movie)
	}

	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	cacheValid, err := hasFreshScrape(city)
	if err != nil {
		return nil, false, err
	}

	return movies, cacheValid, nil
}

func hasFreshScrape(city string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM city_scrapes
			WHERE city = $1 AND scraped_at > NOW() - INTERVAL '24 hours'
		)
	`

	var exists bool
	if err := db.QueryRow(context.Background(), query, city).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func saveMoviesToDB(city string, movies []Movie) error {
	tx, err := db.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	deleteQuery := `DELETE FROM movies WHERE city = $1`
	if _, err := tx.Exec(context.Background(), deleteQuery, city); err != nil {
		return err
	}

	insertQuery := `INSERT INTO movies (city, title, href) VALUES ($1, $2, $3)`
	for _, movie := range movies {
		if _, err := tx.Exec(context.Background(), insertQuery, city, movie.Title, movie.Href); err != nil {
			return err
		}
	}

	cacheQuery := `
		INSERT INTO city_scrapes (city, scraped_at)
		VALUES ($1, NOW())
		ON CONFLICT (city) DO UPDATE SET scraped_at = EXCLUDED.scraped_at
	`
	if _, err := tx.Exec(context.Background(), cacheQuery, city); err != nil {
		return err
	}

	return tx.Commit(context.Background())
}

func loadMovies(city string) ([]Movie, bool, error) {
	movies, fromCache, err := getMoviesFromDB(city)
	if err != nil {
		return nil, false, fmt.Errorf("query cached movies: %w", err)
	}

	if fromCache {
		return movies, true, nil
	}

	log.Printf("No cached data for %s, scraping...", city)

	movies, err = scrapeMovies(city)
	if err != nil {
		return nil, false, fmt.Errorf("scrape movies: %w", err)
	}

	if err := saveMoviesToDB(city, movies); err != nil {
		log.Printf("Failed to save movies for %s: %v", city, err)
	} else {
		log.Printf("Saved %d movies to database for city: %s", len(movies), city)
	}

	return movies, false, nil
}

func scrapeMovies(city string) ([]Movie, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	url := fmt.Sprintf("https://in.bookmyshow.com/explore/movies-%s", city)
	selector := fmt.Sprintf("a[href*=\"/movies/%s/\"]", city)

	var links []map[string]string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(5*time.Second),
		chromedp.Evaluate(fmt.Sprintf(`
			Array.from(document.querySelectorAll('%s')).map(link => {
				const h3Element = link.querySelector('h3');

				let title = '';
				if (h3Element) {
					title = h3Element.textContent.trim();
				} else {
					title = link.textContent.trim();
				}

				return {
					text: title,
					href: link.href
				};
			});
		`, selector), &links),
	)

	if err != nil {
		return nil, err
	}

	var movies []Movie
	for _, link := range links {
		href := link["href"]
		if href != "" {
			movies = append(movies, Movie{
				Title: cleanQuery(link["text"]),
				Href:  href,
			})
		}
	}

	return movies, nil
}

func getMovies(c *gin.Context) {
	city := c.DefaultQuery("city", "cuttack")
	query := c.Query("query")

	movies, fromCache, err := loadMovies(city)
	if err != nil {
		log.Printf("Error loading movies for %s: %v", city, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to load movies: %v", err),
		})
		return
	}

	if fromCache {
		log.Printf("Returning %d cached movies for city: %s", len(movies), city)
	}

	if query != "" {
		movies = fuzzySearchMovies(movies, cleanQuery(query))
	}

	c.JSON(http.StatusOK, gin.H{
		"city":   city,
		"movies": movies,
		"count":  len(movies),
	})
}

func fuzzySearchMovies(movies []Movie, query string) []Movie {
	if len(movies) == 0 {
		return movies
	}

	titles := make([]string, len(movies))
	for i, movie := range movies {
		titles[i] = movie.Title
	}

	matches := fuzzy.Find(query, titles)

	var result []Movie
	for _, match := range matches {
		result = append(result, movies[match.Index])
	}

	return result
}

func preloadMovies() {
	cities := []string{"cuttack", "bhubaneswar"}

	log.Println("Starting initial movie scraping for cities:", cities)

	for _, city := range cities {
		movies, fromCache, err := loadMovies(city)
		if err != nil {
			log.Printf("Failed to load movies for %s: %v", city, err)
			continue
		}

		if fromCache {
			log.Printf("Found %d cached movies for %s (within 24 hours), skipping scrape", len(movies), city)
			continue
		}

		log.Printf("Successfully preloaded %d movies for %s", len(movies), city)
	}

	log.Println("Initial movie scraping completed")
}
