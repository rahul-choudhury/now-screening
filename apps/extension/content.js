const CONFIG = {
  API_URL: "http://localhost:8080/movies",
  STORAGE_KEY: "selectedCity",
  DEFAULT_CITY: "cuttack",
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
  ANIMATIONS: {
    FADE_DURATION: 200,
    SKELETON_PULSE_DURATION: 1500,
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
    if (bookingLink && bookingLink !== "NO_MOVIES") {
      waitForWatchDiv(bookingLink);
    } else if (bookingLink === "NO_MOVIES") {
      waitForWatchDiv("NO_MOVIES");
    }
  } catch (error) {
    console.error("Error in main flow:", error);
  }
}

/**
 * Get selected city from storage
 * @returns {Promise<string>} The selected city
 */
async function getSelectedCity() {
  try {
    const result = await chrome.storage.sync.get([CONFIG.STORAGE_KEY]);
    return result[CONFIG.STORAGE_KEY] || CONFIG.DEFAULT_CITY;
  } catch (error) {
    console.error("Error getting selected city:", error);
    return CONFIG.DEFAULT_CITY;
  }
}

/**
 * Fetches movie data from the API
 * @param {string} movieTitle - The movie title to search for
 * @returns {Promise<Object>} API response data
 */
async function fetchMovieData(movieTitle) {
  const selectedCity = await getSelectedCity();
  const params = new URLSearchParams({
    city: selectedCity,
    query: movieTitle,
  });

  const response = await fetch(`${CONFIG.API_URL}?${params}`);

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
 * @returns {string|null|'NO_MOVIES'} Booking link, null if invalid, or 'NO_MOVIES' if no movies found
 */
function validateAndExtractLink(data) {
  if (
    !data ||
    !Array.isArray(data.movies) ||
    data.movies.length === 0 ||
    data.movies === null
  ) {
    console.warn("No movies found in API response");
    return "NO_MOVIES";
  }

  const firstMovie = data.movies[0];
  if (!firstMovie.href) {
    console.warn("Movie found but no booking link available");
    return null;
  }

  return firstMovie.href;
}

/**
 * Waits for the watch div to be ready and injects the booking link or placeholder
 * @param {string} bookingLink - The booking URL to inject or 'NO_MOVIES' for placeholder
 */
async function waitForWatchDiv(bookingLink) {
  setTimeout(async () => {
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

    if (bookingLink === "NO_MOVIES") {
      await injectNoMoviesPlaceholder(watchDiv);
    } else {
      injectBookingLink(watchDiv, bookingLink);
    }
    setupContentObserver(watchDiv, bookingLink);
  }, CONFIG.TIMING.INITIAL_DELAY);
}

/**
 * Creates and injects the BookMyShow link into the watch div
 * @param {Element} watchDiv - The watch container element
 * @param {string} bookingLink - The booking URL
 * @param {boolean} animate - Whether to animate the transition
 */
function injectBookingLink(watchDiv, bookingLink, animate = false) {
  const existingLink = watchDiv.querySelector(CONFIG.SELECTORS.BMS_LINK);

  if (existingLink && animate) {
    // Fade out existing link
    existingLink.style.transition = `opacity ${CONFIG.ANIMATIONS.FADE_DURATION}ms ease-out`;
    existingLink.style.opacity = "0";

    setTimeout(() => {
      existingLink.remove();
      createAndInsertLink(watchDiv, bookingLink, animate);
    }, CONFIG.ANIMATIONS.FADE_DURATION);
  } else {
    if (existingLink) {
      existingLink.remove();
    }
    createAndInsertLink(watchDiv, bookingLink, animate);
  }
}

/**
 * Creates and inserts the actual link element
 * @param {Element} watchDiv - The watch container element
 * @param {string} bookingLink - The booking URL
 * @param {boolean} animate - Whether to animate the insertion
 */
function createAndInsertLink(watchDiv, bookingLink, animate) {
  const bmsLink = document.createElement("a");
  bmsLink.href = bookingLink;
  bmsLink.target = "_blank";
  bmsLink.className = "bms-link";

  applyLinkStyles(bmsLink);
  setLinkContent(bmsLink);

  if (animate) {
    bmsLink.style.opacity = "0";
    bmsLink.style.transition = `opacity ${CONFIG.ANIMATIONS.FADE_DURATION}ms ease-in`;
  }

  watchDiv.appendChild(bmsLink);

  if (animate) {
    requestAnimationFrame(() => {
      bmsLink.style.opacity = "1";
    });
  }
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
 * Creates and injects a skeleton loading state
 * @param {Element} watchDiv - The watch container element
 */
function injectSkeletonLoader(watchDiv) {
  const existingLink = watchDiv.querySelector(CONFIG.SELECTORS.BMS_LINK);
  if (existingLink) {
    existingLink.remove();
  }

  const skeletonLink = document.createElement("div");
  skeletonLink.className = "bms-link bms-skeleton";

  applySkeletonStyles(skeletonLink);
  setSkeletonContent(skeletonLink);

  watchDiv.appendChild(skeletonLink);
}

/**
 * Applies skeleton loading styles
 * @param {Element} skeletonElement - The skeleton element
 */
function applySkeletonStyles(skeletonElement) {
  skeletonElement.style.cssText = `
    font-size: 12px;
    padding: 12px 0px;
    margin-left: 10px;
    border-top: 1px solid #202830;
    display: flex;
    align-items: center;
    opacity: 0.6;
  `;
}

/**
 * Sets skeleton content with pulsing animation
 * @param {Element} skeletonElement - The skeleton element
 */
function setSkeletonContent(skeletonElement) {
  skeletonElement.innerHTML = `
    <div style="
      width: 23px; 
      height: 23px; 
      margin-right: 7px; 
      background: linear-gradient(90deg, #2c3440 25%, #3c4450 50%, #2c3440 75%);
      background-size: 200% 100%;
      animation: skeleton-pulse ${CONFIG.ANIMATIONS.SKELETON_PULSE_DURATION}ms ease-in-out infinite;
      border-radius: 3px;
    "></div>
    <div style="
      width: 80px;
      height: 12px;
      background: linear-gradient(90deg, #2c3440 25%, #3c4450 50%, #2c3440 75%);
      background-size: 200% 100%;
      animation: skeleton-pulse ${CONFIG.ANIMATIONS.SKELETON_PULSE_DURATION}ms ease-in-out infinite;
      border-radius: 2px;
    "></div>
    <style>
      @keyframes skeleton-pulse {
        0% { background-position: 200% 0; }
        100% { background-position: -200% 0; }
      }
    </style>
  `;
}

/**
 * Creates and injects a "no movies found" placeholder
 * @param {Element} watchDiv - The watch container element
 * @param {string} city - The current city
 */
async function injectNoMoviesPlaceholder(watchDiv, city = null) {
  const existingLink = watchDiv.querySelector(CONFIG.SELECTORS.BMS_LINK);
  if (existingLink) {
    existingLink.remove();
  }

  if (!city) {
    city = await getSelectedCity();
  }

  const placeholderDiv = document.createElement("div");
  placeholderDiv.className = "bms-link bms-no-movies";

  placeholderDiv.style.cssText = `
    font-size: 12px;
    padding: 12px 0px;
    margin-left: 10px;
    border-top: 1px solid #202830;
    display: flex;
    align-items: center;
    color: #9ab;
    opacity: 0.7;
  `;

  placeholderDiv.innerHTML = `
    <div style="
      width: 23px; 
      height: 23px; 
      margin-right: 7px; 
      background: #404040;
      border-radius: 3px;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 10px;
      color: #666;
    ">?</div>
    Not screening in ${city}
  `;

  watchDiv.appendChild(placeholderDiv);
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
        if (bookingLink === "NO_MOVIES") {
          injectNoMoviesPlaceholder(watchDiv);
        } else {
          injectBookingLink(watchDiv, bookingLink);
        }
      }
    }, CONFIG.TIMING.OBSERVER_DEBOUNCE);
  });

  mutationObserver.observe(watchDiv, {
    childList: true,
    subtree: true,
  });
}

/**
 * Handle messages from popup (city change notifications)
 */
chrome.runtime.onMessage.addListener((message) => {
  if (message.type === "CITY_CHANGED") {
    refreshBookingLink();
  }
});

/**
 * Refresh the booking link with current city selection
 */
async function refreshBookingLink() {
  const movieTitle =
    document.querySelector(CONFIG.SELECTORS.MOVIE_TITLE)?.innerHTML?.trim() ||
    null;

  if (!movieTitle) {
    console.log("Movie title not found for refresh");
    return;
  }

  const watchDiv = document.getElementById(CONFIG.SELECTORS.WATCH_DIV);
  if (!watchDiv) {
    console.log("Watch div not found for refresh");
    return;
  }

  injectSkeletonLoader(watchDiv);

  try {
    const movieData = await fetchMovieData(movieTitle);
    const bookingLink = validateAndExtractLink(movieData);

    if (bookingLink && bookingLink !== "NO_MOVIES") {
      injectBookingLink(watchDiv, bookingLink, true);
    } else if (bookingLink === "NO_MOVIES") {
      const skeleton = watchDiv.querySelector(".bms-skeleton");
      if (skeleton) {
        skeleton.style.transition = `opacity ${CONFIG.ANIMATIONS.FADE_DURATION}ms ease-out`;
        skeleton.style.opacity = "0";
        setTimeout(async () => {
          skeleton.remove();
          await injectNoMoviesPlaceholder(watchDiv);
        }, CONFIG.ANIMATIONS.FADE_DURATION);
      } else {
        await injectNoMoviesPlaceholder(watchDiv);
      }
    } else {
      const skeleton = watchDiv.querySelector(".bms-skeleton");
      if (skeleton) {
        skeleton.remove();
      }
    }
  } catch (error) {
    console.error("Error refreshing booking link:", error);
    const skeleton = watchDiv.querySelector(".bms-skeleton");
    if (skeleton) {
      skeleton.remove();
    }
  }
}

window.addEventListener("beforeunload", () => {
  if (mutationObserver) {
    mutationObserver.disconnect();
  }
});

main();
