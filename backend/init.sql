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