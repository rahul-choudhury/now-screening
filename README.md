# Now Screening

A Chrome extension that bridges Letterboxd movie pages with BookMyShow ticket booking. Why? Because, I'm lazy.

## Overview

<!-- Add screenshot of the extension in action on a Letterboxd page showing the BookMyShow link -->
<!-- ![Extension Demo](docs/images/extension-demo.png) -->

The system consists of two main components:

- **Backend API**: Go web scraper that fetches BookMyShow movie listings and provides fuzzy search
- **Chrome Extension**: Content script that injects booking links into Letterboxd movie pages

## Features

- Automatic movie detection on Letterboxd pages
- Real-time BookMyShow availability checking
- 24-hour caching of scraped movie links
- Support for city-specific movie listings (supports Cuttack & Bhubaneswar for now)

## Installation

### Prerequisites

- Node.js and npm
- Go 1.19+
- Docker (for PostgreSQL)
- Chrome browser

### Setup

1. Clone the repository.

2. Install dependencies:
```bash
npm install
```

3. Start the database:
```bash
npm run db:up
```

4. Start the API server:
```bash
npm run dev
```

The API server will start on `http://localhost:8080`.

### Chrome Extension Installation

1. Open Chrome and navigate to `chrome://extensions/`
2. Enable "Developer mode" in the top right
3. Click "Load unpacked" and select the `apps/extension/` directory
4. The extension is now installed and will activate on Letterboxd movie pages

<!-- Add screenshot of Chrome extension management page showing the loaded extension -->
<!-- ![Extension Installation](docs/images/extension-install.png) -->

### Extension Configuration

The extension includes a popup interface for city selection:

1. Click the extension icon in the toolbar to open the settings popup
2. Select your preferred city (currently supports Cuttack and Bhubaneswar)
3. Your selection is automatically saved using Chrome's sync storage
4. All open Letterboxd tabs will immediately reflect the new city selection

<!-- Add screenshot showing the extension popup with city selection options -->
<!-- ![Extension Popup](docs/images/extension-popup.png) -->

## Usage

1. Visit any movie page on Letterboxd (e.g., `https://letterboxd.com/film/28-years-later/`)
2. Look for the "BookMyShow" link just under the watch section
3. Click the link to open the bookmyshow link for that movie in a new tab

<!-- Add screenshot showing the BookMyShow link integrated into Letterboxd's watch section -->
<!-- ![Letterboxd Integration](docs/images/letterboxd-integration.png) -->

## API Endpoints

### Get Movies
```
GET /movies?city={city}&query={movie_title}
```

**Parameters:**
- `city` (optional): City name for location-specific results (default: "cuttack")
- `query` (optional): Movie title for fuzzy search

**Examples:**
```bash
# Get all movies in Bhubaneswar
curl "http://localhost:8080/movies?city=bhubaneswar"

# Search for specific movie
curl "http://localhost:8080/movies?query=Ballerina"

# Search in specific city
curl "http://localhost:8080/movies?city=bhubaneswar&query=Ballerina"
```

## Development

### Project Structure

```
now-screening/
├── apps/
│   ├── api/           # Go backend server
│   │   ├── main.go    # Main server file
│   │   └── init.sql   # Database schema
│   └── extension/     # Chrome extension
│       ├── manifest.json
│       ├── content.js
│       ├── popup.html
│       ├── popup.js
│       └── popup.css
└── package.json       # Workspace configuration
```

### Available Commands

```bash
# Development
npm run dev          # Start API server
npm run db:up        # Start database
npm run db:down      # Stop database

# Manual commands
cd apps/api && go run main.go    # Run API directly
cd apps/api && go mod tidy       # Clean Go dependencies
```

### Database

The application uses PostgreSQL with Docker. The database schema is automatically initialized from `apps/api/init.sql`. Movie data is cached for 24 hours to reduce scraping frequency.

**Connection details:**
- Host: `localhost:5432`
- Username: `postgres`
- Password: `password`

## Architecture

<!-- Add system architecture diagram showing flow between components -->
<!-- ![System Architecture](docs/images/architecture-diagram.png) -->

### Data Flow

1. User visits Letterboxd movie page
2. Chrome extension extracts movie title from page DOM
3. Extension retrieves selected city from Chrome sync storage
4. Extension queries backend API with movie title and city
5. Backend checks PostgreSQL cache (24-hour TTL)
6. If cache miss: Backend scrapes BookMyShow using headless Chrome
7. Backend applies fuzzy search to find best match
8. Extension injects BookMyShow link into Letterboxd's watch section

### Extension Architecture

**Message Passing & Storage:**
- Uses Chrome's `chrome.storage.sync` API for persistent city preferences across devices
- Implements runtime message passing between popup and content scripts
- When city changes in popup, all active Letterboxd tabs receive `CITY_CHANGED` messages
- Content scripts automatically refresh movie links when receiving city change notifications

**Content Script Features:**
- Waits for Letterboxd's scripts to fully load before injecting BookMyShow links
- Uses MutationObserver to monitor the watch div for DOM changes (childList and subtree)
- Implements 100ms debounced re-injection to restore BookMyShow links if they're removed

### Web Scraping

The backend uses `chromedp` with headless Chrome to scrape BookMyShow. It targets explore pages like `https://in.bookmyshow.com/explore/home/{city}` and extracts movie links.
