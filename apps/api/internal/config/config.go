package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	ServerAddr    string
	CacheTTL      time.Duration
	ScrapeTimeout time.Duration
	DefaultCity   string
	PreloadCities []string
}

func Load() Config {
	return Config{
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "password"),
		ServerAddr:    ":8080",
		CacheTTL:      24 * time.Hour,
		ScrapeTimeout: 60 * time.Second,
		DefaultCity:   "cuttack",
		PreloadCities: []string{"cuttack", "bhubaneswar"},
	}
}

func (c Config) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s", c.DBUser, c.DBPassword, c.DBHost, c.DBPort)
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultValue
}
