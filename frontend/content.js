const CONFIG = {
  API_URL: "http://localhost:8080/movies",
  SELECTORS: {
    MOVIE_TITLE: "h1.primaryname span.name",
    WATCH_DIV: "watch",
    JUSTWATCH_BRANDING: ".jw-branding",
    BMS_LINK: ".bms-link",
  },
  TIMING: {
    INITIAL_DELAY: 500,
    RETRY_DELAY: 100,
    READINESS_CHECK_DELAY: 500,
    OBSERVER_DEBOUNCE: 100,
  },
  BMS: {
    ICON_URL:
      "https://cdn.brandfetch.io/id4J58sqa_/theme/dark/symbol.svg?c=1dxbfHSJFAPEGdCLU4o5B",
    TEXT: "BookMyShow",
  },
};

let mutationObserver = null;

/**
 * Main entry point - extracts movie title and initiates the flow
 */
async function main() {
  const movieTitle =
    document.querySelector(CONFIG.SELECTORS.MOVIE_TITLE)?.innerHTML?.trim() ||
    null;

  if (!movieTitle) {
    console.log("Movie title not found");
    return;
  }

  try {
    const movieData = await fetchMovieData(movieTitle);
    const bookingLink = validateAndExtractLink(movieData);
    if (bookingLink) {
      waitForWatchDiv(bookingLink);
    }
  } catch (error) {
    console.error("Error in main flow:", error);
  }
}

/**
 * Fetches movie data from the API
 * @param {string} movieTitle - The movie title to search for
 * @returns {Promise<Object>} API response data
 */
async function fetchMovieData(movieTitle) {
  const response = await fetch(
    `${CONFIG.API_URL}?query=${encodeURIComponent(movieTitle)}`,
  );

  if (!response.ok) {
    throw new Error(
      `API request failed: ${response.status} ${response.statusText}`,
    );
  }

  return await response.json();
}

/**
 * Validates API response and extracts booking link
 * @param {Object} data - API response data
 * @returns {string|null} Booking link or null if invalid
 */
function validateAndExtractLink(data) {
  if (!data || !Array.isArray(data.movies) || data.movies.length === 0) {
    console.warn("No movies found in API response");
    return null;
  }

  const firstMovie = data.movies[0];
  if (!firstMovie.href) {
    console.warn("Movie found but no booking link available");
    return null;
  }

  return firstMovie.href;
}

/**
 * Waits for the watch div to be ready and injects the booking link
 * @param {string} bookingLink - The booking URL to inject
 */
function waitForWatchDiv(bookingLink) {
  setTimeout(() => {
    const watchDiv = document.getElementById(CONFIG.SELECTORS.WATCH_DIV);
    if (!watchDiv) {
      setTimeout(() => waitForWatchDiv(bookingLink), CONFIG.TIMING.RETRY_DELAY);
      return;
    }

    const justWatchElement = watchDiv.querySelector(
      CONFIG.SELECTORS.JUSTWATCH_BRANDING,
    );
    if (!justWatchElement) {
      setTimeout(
        () => waitForWatchDiv(bookingLink),
        CONFIG.TIMING.READINESS_CHECK_DELAY,
      );
      return;
    }

    injectBookingLink(watchDiv, bookingLink);
    setupContentObserver(watchDiv, bookingLink);
  }, CONFIG.TIMING.INITIAL_DELAY);
}

/**
 * Creates and injects the BookMyShow link into the watch div
 * @param {Element} watchDiv - The watch container element
 * @param {string} bookingLink - The booking URL
 */
function injectBookingLink(watchDiv, bookingLink) {
  const existingLink = watchDiv.querySelector(CONFIG.SELECTORS.BMS_LINK);
  if (existingLink) {
    existingLink.remove();
  }

  const bmsLink = document.createElement("a");
  bmsLink.href = bookingLink;
  bmsLink.target = "_blank";
  bmsLink.className = "bms-link";

  applyLinkStyles(bmsLink);
  setLinkContent(bmsLink);

  watchDiv.appendChild(bmsLink);
}

/**
 * Applies styling to the BookMyShow link
 * @param {Element} linkElement - The link element to style
 */
function applyLinkStyles(linkElement) {
  linkElement.style.cssText = `
    font-size: 12px;
    padding: 12px 0px;
    margin-left: 10px;
    border-top: 1px solid #202830;
    display: flex;
    align-items: center;
  `;
}

/**
 * Sets the content (icon + text) for the BookMyShow link
 * @param {Element} linkElement - The link element
 */
function setLinkContent(linkElement) {
  linkElement.innerHTML = `
    <img src="${CONFIG.BMS.ICON_URL}" 
         style="width: 23px; height: 23px; margin-right: 7px;" 
         alt="BookMyShow" 
         onerror="this.style.display='none'">
    ${CONFIG.BMS.TEXT}
  `;
}

/**
 * Sets up observer to re-inject link when content changes
 * @param {Element} watchDiv - The watch container element
 * @param {string} bookingLink - The booking URL
 */
function setupContentObserver(watchDiv, bookingLink) {
  if (mutationObserver) {
    mutationObserver.disconnect();
  }

  mutationObserver = new MutationObserver(() => {
    setTimeout(() => {
      if (!watchDiv.querySelector(CONFIG.SELECTORS.BMS_LINK)) {
        injectBookingLink(watchDiv, bookingLink);
      }
    }, CONFIG.TIMING.OBSERVER_DEBOUNCE);
  });

  mutationObserver.observe(watchDiv, {
    childList: true,
    subtree: true,
  });
}

window.addEventListener("beforeunload", () => {
  if (mutationObserver) {
    mutationObserver.disconnect();
  }
});

main();
