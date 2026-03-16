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
    MAX_WAIT_ATTEMPTS: 40,
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
let currentBookingState = null;
let watchDivTimeoutId = null;

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
    currentBookingState = validateAndExtractLink(movieData);
    if (currentBookingState) {
      scheduleWatchDivCheck();
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
async function fetchMovieData(movieTitle, selectedCity = null) {
  const city = selectedCity || (await getSelectedCity());
  const params = new URLSearchParams({
    city,
    query: movieTitle,
  });

  const response = await fetch(`${CONFIG.API_URL}?${params}`);

  if (!response.ok) {
    throw new Error(
      `API request failed: ${response.status} ${response.statusText}`,
    );
  }

  return response.json();
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
    data.movies.length === 0
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
 */
function scheduleWatchDivCheck(
  attempt = 0,
  delay = CONFIG.TIMING.INITIAL_DELAY,
) {
  if (watchDivTimeoutId) {
    clearTimeout(watchDivTimeoutId);
  }

  watchDivTimeoutId = setTimeout(() => {
    watchDivTimeoutId = null;
    waitForWatchDiv(attempt);
  }, delay);
}

async function waitForWatchDiv(attempt = 0) {
  if (attempt >= CONFIG.TIMING.MAX_WAIT_ATTEMPTS) {
    console.warn("Watch section was not ready after maximum retries");
    return;
  }

  const watchDiv = document.getElementById(CONFIG.SELECTORS.WATCH_DIV);
  if (!watchDiv) {
    scheduleWatchDivCheck(attempt + 1, CONFIG.TIMING.RETRY_DELAY);
    return;
  }

  const justWatchElement = watchDiv.querySelector(
    CONFIG.SELECTORS.JUSTWATCH_BRANDING,
  );
  if (!justWatchElement) {
    scheduleWatchDivCheck(attempt + 1, CONFIG.TIMING.READINESS_CHECK_DELAY);
    return;
  }

  if (currentBookingState === "NO_MOVIES") {
    await injectNoMoviesPlaceholder(watchDiv);
  } else if (currentBookingState) {
    injectBookingLink(watchDiv, currentBookingState);
  }

  setupContentObserver(watchDiv);
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
 * Removes an existing injected BookMyShow element from the watch section
 * @param {Element} watchDiv - The watch container element
 */
function removeExistingBmsElement(watchDiv) {
  const existingElement = watchDiv.querySelector(CONFIG.SELECTORS.BMS_LINK);
  if (existingElement) {
    existingElement.remove();
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

  applyBaseContainerStyles(bmsLink);
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
 * Applies shared container styling for injected BookMyShow UI
 * @param {Element} element - The element to style
 * @param {string} extraStyles - Additional styles appended to the base block
 */
function applyBaseContainerStyles(element, extraStyles = "") {
  element.style.cssText = `
    font-size: 12px;
    padding: 12px 0px;
    margin-left: 10px;
    border-top: 1px solid #202830;
    display: flex;
    align-items: center;
    ${extraStyles}
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
  removeExistingBmsElement(watchDiv);

  const skeletonLink = document.createElement("div");
  skeletonLink.className = "bms-link bms-skeleton";

  applyBaseContainerStyles(skeletonLink, "opacity: 0.6;");
  setSkeletonContent(skeletonLink);

  watchDiv.appendChild(skeletonLink);
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
  removeExistingBmsElement(watchDiv);

  if (!city) {
    city = await getSelectedCity();
  }

  const placeholderDiv = document.createElement("div");
  placeholderDiv.className = "bms-link bms-no-movies";

  applyBaseContainerStyles(placeholderDiv, "color: #9ab; opacity: 0.7;");

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
 */
function setupContentObserver(watchDiv) {
  if (mutationObserver) {
    mutationObserver.disconnect();
  }

  mutationObserver = new MutationObserver(() => {
    setTimeout(() => {
      if (!watchDiv.querySelector(CONFIG.SELECTORS.BMS_LINK)) {
        if (currentBookingState === "NO_MOVIES") {
          injectNoMoviesPlaceholder(watchDiv);
        } else if (currentBookingState) {
          injectBookingLink(watchDiv, currentBookingState);
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
    refreshBookingLink(message.city);
  }
});

/**
 * Refresh the booking link with current city selection
 */
async function refreshBookingLink(selectedCity = null) {
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

  if (watchDivTimeoutId) {
    clearTimeout(watchDivTimeoutId);
    watchDivTimeoutId = null;
  }

  currentBookingState = null;
  setupContentObserver(watchDiv);
  injectSkeletonLoader(watchDiv);

  try {
    const movieData = await fetchMovieData(movieTitle, selectedCity);
    const bookingLink = validateAndExtractLink(movieData);
    currentBookingState = bookingLink;

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

    setupContentObserver(watchDiv);
  } catch (error) {
    console.error("Error refreshing booking link:", error);
    const skeleton = watchDiv.querySelector(".bms-skeleton");
    if (skeleton) {
      skeleton.remove();
    }
  }
}

window.addEventListener("beforeunload", () => {
  if (watchDivTimeoutId) {
    clearTimeout(watchDivTimeoutId);
  }
  if (mutationObserver) {
    mutationObserver.disconnect();
  }
});

main();
