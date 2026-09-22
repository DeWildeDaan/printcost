import { settingsApi } from "../api.js";
import { setCurrencySymbol, escapeHTML } from "../utils.js";

export async function renderSettings(container) {
  const s = await settingsApi.get();

  container.innerHTML = `
    <h1>Settings</h1>
    <p class="page-subtitle">These defaults apply to new quotes. Changing them does not retroactively alter saved quotes.</p>
    <form id="settings-form" class="card" style="max-width:520px;">
      <div class="form-grid single">
        <div class="field"><label>Energy cost per kWh</label><input type="number" step="0.01" name="energy_cost" value="${s.energy_cost}"></div>
        <div class="field"><label>Labour rate per hour</label><input type="number" step="0.01" name="labor_rate" value="${s.labor_rate}"></div>
        <div class="field"><label>Failure rate (%)</label><input type="number" step="0.1" name="failure_rate" value="${s.failure_rate}"></div>
        <div class="field"><label>Default markup (%)</label><input type="number" step="0.1" name="markup" value="${s.markup}"></div>
        <div class="field"><label>Currency symbol</label><input type="text" name="currency" value="${escapeHTML(s.currency)}"></div>
        <div class="field"><label>Company / maker name</label><input type="text" name="company_name" value="${escapeHTML(s.company_name)}"></div>
      </div>
      <div style="margin-top:16px;">
        <button type="submit" class="primary">Save settings</button>
      </div>
      <div id="save-confirm" style="margin-top:10px; color: var(--good); display:none;">Saved.</div>
    </form>
  `;

  const form = container.querySelector("#settings-form");
  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const data = {
      energy_cost: parseFloat(form.elements.energy_cost.value) || 0,
      labor_rate: parseFloat(form.elements.labor_rate.value) || 0,
      failure_rate: parseFloat(form.elements.failure_rate.value) || 0,
      markup: parseFloat(form.elements.markup.value) || 0,
      currency: form.elements.currency.value,
      company_name: form.elements.company_name.value,
    };
    try {
      await settingsApi.update(data);
      setCurrencySymbol(data.currency);
      const confirmEl = container.querySelector("#save-confirm");
      confirmEl.style.display = "block";
      setTimeout(() => (confirmEl.style.display = "none"), 2000);
    } catch (err) {
      alert(err.message);
    }
  });
}
