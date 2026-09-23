import { settingsApi } from "./api.js";
import { setCurrencySymbol } from "./utils.js";
import { renderDashboard } from "./pages/dashboard.js";
import { renderEquipmentPage } from "./pages/equipment.js";
import { renderFilaments } from "./pages/filaments.js";
import { renderConsumables } from "./pages/consumables.js";
import { renderMaterialSettings } from "./pages/materialSettings.js";
import { renderQuoteList } from "./pages/quoteList.js";
import { renderQuoteBuilder } from "./pages/quoteBuilder.js";
import { renderQuoteDetail } from "./pages/quoteDetail.js";
import { renderSettings } from "./pages/settings.js";

const mainContent = document.getElementById("main-content");

const routes = [
  { pattern: /^dashboard$/, handler: () => renderDashboard(mainContent) },
  { pattern: /^quotes\/new$/, handler: () => renderQuoteBuilder(mainContent, null) },
  { pattern: /^quotes\/(\d+)\/edit$/, handler: (m) => renderQuoteBuilder(mainContent, m[1]) },
  { pattern: /^quotes\/(\d+)$/, handler: (m) => renderQuoteDetail(mainContent, m[1]) },
  { pattern: /^quotes$/, handler: () => renderQuoteList(mainContent) },
  { pattern: /^filaments$/, handler: () => renderFilaments(mainContent) },
  { pattern: /^material-settings$/, handler: () => renderMaterialSettings(mainContent) },
  { pattern: /^equipment\/(machines|build-plates|nozzles|ams-units|other-equipment)$/, handler: (m) => renderEquipmentPage(mainContent, m[1]) },
  { pattern: /^consumables$/, handler: () => renderConsumables(mainContent) },
  { pattern: /^settings$/, handler: () => renderSettings(mainContent) },
];

function currentPath() {
  return (location.hash || "#/dashboard").slice(2);
}

function updateActiveNav(path) {
  const links = [...document.querySelectorAll("#sidebar a[data-route]")];
  // Pick the single longest matching route so sibling routes that share a
  // prefix (e.g. "quotes" and "quotes/new") don't both light up.
  let best = null;
  for (const a of links) {
    const route = a.dataset.route;
    if (path === route || path.startsWith(route + "/")) {
      if (!best || route.length > best.dataset.route.length) best = a;
    }
  }
  links.forEach((a) => a.classList.toggle("active", a === best));
}

async function router() {
  const path = currentPath();
  updateActiveNav(path);
  for (const route of routes) {
    const m = path.match(route.pattern);
    if (m) {
      mainContent.innerHTML = "";
      try {
        await route.handler(m);
      } catch (err) {
        mainContent.innerHTML = `<div class="error-banner">${err.message || err}</div>`;
      }
      return;
    }
  }
  location.hash = "#/dashboard";
}

window.addEventListener("hashchange", () => {
  const navToggle = document.getElementById("nav-toggle");
  if (navToggle) navToggle.checked = false;
  router();
});

async function boot() {
  try {
    const settings = await settingsApi.get();
    setCurrencySymbol(settings.currency);
  } catch (err) {
    console.error("failed to load settings", err);
  }
  router();
}

boot();
