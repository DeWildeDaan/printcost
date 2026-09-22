import { filamentsApi } from "../api.js";
import { fmtMoney, fmtNum, escapeHTML } from "../utils.js";

function derivedValues(f) {
  const spoolWeightKg = Number(f.spool_weight_kg) || 0;
  const spoolPrice = Number(f.spool_price) || 0;
  const density = Number(f.density_gcm3) || 0;
  const diameter = Number(f.diameter_mm) || 0;

  const pricePerKg = spoolWeightKg > 0 ? spoolPrice / spoolWeightKg : 0;
  const pricePerG = spoolWeightKg > 0 ? spoolPrice / (spoolWeightKg * 1000) : 0;

  const radiusCm = (diameter / 10) / 2;
  const volumeCm3PerM = Math.PI * radiusCm * radiusCm * 100;
  const lengthPerRollM = density > 0 && radiusCm > 0 ? (spoolWeightKg * 1000) / (density * volumeCm3PerM) : 0;

  return { pricePerKg, pricePerG, lengthPerRollM };
}

function formHTML(item) {
  const v = item || {};
  return `
    <form id="filament-form" class="card" style="margin-bottom:20px;">
      <h2 style="margin-top:0;">${item ? "Edit" : "Add"} Filament</h2>
      <p class="hint" style="margin-top:-6px;">Name is generated automatically from brand + material.</p>
      <div class="form-grid">
        <div class="field"><label>Brand</label><input name="brand" value="${escapeHTML(v.brand)}"></div>
        <div class="field"><label>Material *</label><input name="material" required placeholder="ABS, PLA, PETG, TPU" value="${escapeHTML(v.material)}"></div>
        <div class="field"><label>Diameter (mm)</label><input type="number" step="0.01" name="diameter_mm" value="${v.diameter_mm ?? 1.75}"></div>
        <div class="field"><label>Spool price *</label><input type="number" step="0.01" required name="spool_price" value="${v.spool_price ?? ""}"></div>
        <div class="field"><label>Spool weight (kg)</label><input type="number" step="0.01" name="spool_weight_kg" value="${v.spool_weight_kg ?? 1.0}"></div>
        <div class="field"><label>Density (g/cm³)</label><input type="number" step="0.01" name="density_gcm3" value="${v.density_gcm3 ?? 1.24}"></div>
        <div class="field"><label>Drying temperature (°C)</label><input type="number" step="1" name="drying_temp_c" value="${v.drying_temp_c ?? ""}"></div>
        <div class="field"><label>Drying time (hours)</label><input type="number" step="0.1" name="drying_time_hours" value="${v.drying_time_hours ?? ""}"></div>
        <div class="field">
          <label>&nbsp;</label>
          <div class="checkbox-row">
            <input type="checkbox" id="requires_dry_cabinet" name="requires_dry_cabinet" ${v.requires_dry_cabinet ? "checked" : ""}>
            <label for="requires_dry_cabinet" style="margin:0;">Requires dry cabinet</label>
          </div>
        </div>
        <div class="field span2"><label>Notes</label><textarea name="notes">${escapeHTML(v.notes)}</textarea></div>
      </div>
      <div id="derived-preview" style="margin-top:12px; display:flex; gap:10px; flex-wrap:wrap;"></div>
      <div style="margin-top:16px; display:flex; gap:8px;">
        <button type="submit" class="primary">${item ? "Save changes" : "Add"}</button>
        <button type="button" id="cancel-form">Cancel</button>
      </div>
    </form>
  `;
}

function collectFormData(form) {
  return {
    brand: form.elements.brand.value,
    material: form.elements.material.value,
    diameter_mm: parseFloat(form.elements.diameter_mm.value) || 1.75,
    spool_price: parseFloat(form.elements.spool_price.value) || 0,
    spool_weight_kg: parseFloat(form.elements.spool_weight_kg.value) || 1.0,
    density_gcm3: parseFloat(form.elements.density_gcm3.value) || 1.24,
    drying_temp_c: form.elements.drying_temp_c.value === "" ? null : parseInt(form.elements.drying_temp_c.value, 10),
    drying_time_hours: form.elements.drying_time_hours.value === "" ? null : parseFloat(form.elements.drying_time_hours.value),
    requires_dry_cabinet: form.elements.requires_dry_cabinet.checked,
    notes: form.elements.notes.value,
  };
}

export async function renderFilaments(container) {
  let items = [];
  let editingId = null;

  async function load() {
    const res = await filamentsApi.list();
    items = res.data;
    draw();
  }

  function rowHTML(f) {
    return `
      <tr data-id="${f.id}">
        <td>${escapeHTML(f.name)}</td>
        <td>${escapeHTML(f.brand)}</td>
        <td>${escapeHTML(f.material)}</td>
        <td class="num">${fmtMoney(f.price_per_kg)}</td>
        <td class="num">${fmtMoney(f.price_per_g)}</td>
        <td class="num">${fmtNum(f.length_per_roll_m, 1)} m</td>
        <td>${f.requires_dry_cabinet ? "Yes" : "—"}</td>
        <td class="table-actions">
          <button class="small edit-btn" data-id="${f.id}">Edit</button>
          <button class="small danger delete-btn" data-id="${f.id}">Delete</button>
        </td>
      </tr>`;
  }

  function draw() {
    const showForm = editingId !== null;
    const editingItem = editingId && editingId !== "new" ? items.find((i) => i.id === editingId) : null;

    container.innerHTML = `
      <div class="page-header">
        <h1>Filament Library</h1>
        ${!showForm ? `<button class="primary" id="add-btn">Add Filament</button>` : ""}
      </div>
      ${showForm ? formHTML(editingItem) : ""}
      <div class="card">
        <table>
          <thead><tr>
            <th>Name</th><th>Brand</th><th>Material</th>
            <th class="num">€/kg</th><th class="num">€/g</th><th class="num">Length/roll</th>
            <th>Dry cabinet</th><th></th>
          </tr></thead>
          <tbody>
            ${items.length ? items.map(rowHTML).join("") : `<tr><td colspan="8" class="empty-state">No filaments yet.</td></tr>`}
          </tbody>
        </table>
      </div>
    `;

    if (!showForm) {
      container.querySelector("#add-btn").addEventListener("click", () => { editingId = "new"; draw(); });
    } else {
      const form = container.querySelector("#filament-form");
      const updatePreview = () => {
        const d = derivedValues(collectFormData(form));
        container.querySelector("#derived-preview").innerHTML = `
          <span class="derived-value">${fmtMoney(d.pricePerKg)}/kg</span>
          <span class="derived-value">${fmtMoney(d.pricePerG)}/g</span>
          <span class="derived-value">${fmtNum(d.lengthPerRollM, 1)} m/roll</span>
        `;
      };
      form.addEventListener("input", updatePreview);
      updatePreview();

      form.addEventListener("submit", async (e) => {
        e.preventDefault();
        const data = collectFormData(form);
        try {
          if (editingItem) {
            await filamentsApi.update(editingItem.id, data);
          } else {
            await filamentsApi.create(data);
          }
          editingId = null;
          await load();
        } catch (err) {
          alert(err.message);
        }
      });
      container.querySelector("#cancel-form").addEventListener("click", () => { editingId = null; draw(); });
    }

    container.querySelectorAll(".edit-btn").forEach((btn) => {
      btn.addEventListener("click", () => { editingId = Number(btn.dataset.id); draw(); });
    });
    container.querySelectorAll(".delete-btn").forEach((btn) => {
      btn.addEventListener("click", async () => {
        if (!confirm("Delete this filament? This cannot be undone.")) return;
        try {
          await filamentsApi.remove(Number(btn.dataset.id));
          await load();
        } catch (err) {
          alert(err.message);
        }
      });
    });
  }

  await load();
}
