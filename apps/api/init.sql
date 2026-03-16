CREATE TABLE IF NOT EXISTS movies (
    id SERIAL PRIMARY KEY,
    city VARCHAR(100) NOT NULL,
    title VARCHAR(500) NOT NULL,
    href VARCHAR(1000) NOT NULL,
    scraped_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(city, href)
);

CREATE INDEX IF NOT EXISTS idx_movies_city ON movies(city);
CREATE INDEX IF NOT EXISTS idx_movies_scraped_at ON movies(scraped_at);

CREATE TABLE IF NOT EXISTS city_scrapes (
    city VARCHAR(100) PRIMARY KEY,
    scraped_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_city_scrapes_scraped_at ON city_scrapes(scraped_at);
