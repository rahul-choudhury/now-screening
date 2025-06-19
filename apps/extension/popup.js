const STORAGE_KEY = "selectedCity";
const DEFAULT_CITY = "cuttack";

/**
 * Initialize popup when DOM is loaded
 */
document.addEventListener("DOMContentLoaded", async () => {
  await loadCurrentCity();
  setupEventListeners();
});

/**
 * Load and display the currently selected city
 */
async function loadCurrentCity() {
  try {
    const result = await chrome.storage.sync.get([STORAGE_KEY]);
    const selectedCity = result[STORAGE_KEY] || DEFAULT_CITY;

    const radioButton = document.querySelector(
      `input[name="city"][value="${selectedCity}"]`,
    );
    if (radioButton) {
      radioButton.checked = true;
    }
  } catch (error) {
    console.error("Error loading city selection:", error);
    const firstRadio = document.querySelector('input[name="city"]');
    if (firstRadio) {
      firstRadio.checked = true;
    }
  }
}

/**
 * Set up event listeners for city selection
 */
function setupEventListeners() {
  const cityRadios = document.querySelectorAll('input[name="city"]');

  cityRadios.forEach((radio) => {
    radio.addEventListener("change", handleCityChange);
  });
}

/**
 * Handle city selection change
 * @param {Event} event - The change event
 */
async function handleCityChange(event) {
  if (!event.target.checked) return;

  const selectedCity = event.target.value;

  try {
    await chrome.storage.sync.set({ [STORAGE_KEY]: selectedCity });
    console.log(`City selection saved: ${selectedCity}`);

    notifyContentScripts(selectedCity);
  } catch (error) {
    console.error("Error saving city selection:", error);
  }
}

/**
 * Notify content scripts about city change
 * @param {string} city - The newly selected city
 */
async function notifyContentScripts(city) {
  try {
    const tabs = await chrome.tabs.query({
      url: "*://letterboxd.com/film/*",
    });

    tabs.forEach((tab) => {
      chrome.tabs
        .sendMessage(tab.id, {
          type: "CITY_CHANGED",
          city: city,
        })
        .catch((error) => {
          console.debug("Could not notify tab:", tab.id, error);
        });
    });
  } catch (error) {
    console.error("Error notifying content scripts:", error);
  }
}

