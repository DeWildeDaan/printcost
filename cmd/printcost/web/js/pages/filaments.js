import { filamentsApi, buildPlatesApi } from "../api.js";
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
        <div class="field">
          <label>&nbsp;</label>
          <div class="checkbox-row">
            <input type="checkbox" id="glue_stick_recommended" name="glue_stick_recommended" ${v.glue_stick_recommended ? "checked" : ""}>
            <label for="glue_stick_recommended" style="margin:0;">Glue stick recommended</label>
          </div>
        </div>
        <div class="field span2"><label>Notes</label><textarea name="notes">${escapeHTML(v.notes)}</textarea></div>
      </div>

      <h3 style="margin-bottom:6px;">Colors</h3>
      <p class="hint" style="margin-top:0;">Every color this filament comes in, with an optional link to that color's tuned profile.</p>
      <div class="row-list" id="colors-list"></div>
      <button type="button" id="add-color" class="small" style="margin-top:10px;">+ Add color</button>

      <h3 style="margin-top:20px; margin-bottom:6px;">Recommended build plates</h3>
      <div id="plates-list" class="checkbox-row" style="flex-wrap:wrap; gap:14px;"></div>

      <div id="derived-preview" style="margin-top:12px; display:flex; gap:10px; flex-wrap:wrap;"></div>
      <div style="margin-top:16px; display:flex; gap:8px;">
        <button type="submit" class="primary">${item ? "Save changes" : "Add"}</button>
        <button type="button" id="cancel-form">Cancel</button>
      </div>
    </form>
  `;
}

function collectFormData(form, colors, buildPlates) {
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
    glue_stick_recommended: form.elements.glue_stick_recommended.checked,
    notes: form.elements.notes.value,
    colors: colors.filter((c) => c.hex),
    recommended_build_plate_ids: buildPlates
      .filter((p) => form.querySelector(`.plate-check[value="${p.id}"]`)?.checked)
      .map((p) => p.id),
  };
}

export async function renderFilaments(container) {
  let items = [];
  let buildPlates = [];
  let editingId = null;

  async function load() {
    const [filamentsRes, platesRes] = await Promise.all([filamentsApi.list(), buildPlatesApi.list()]);
    items = filamentsRes.data;
    buildPlates = platesRes.data;
    draw();
  }

  function colorSwatches(f) {
    if (!f.colors || !f.colors.length) return "—";
    return f.colors
      .map((c) => {
        const swatch = `<span class="swatch" title="${escapeHTML(c.hex)}" style="background:${escapeHTML(c.hex)}; border:1px solid var(--border);"></span>`;
        return c.profile_url ? `<a href="${escapeHTML(c.profile_url)}" target="_blank" rel="noopener" title="${escapeHTML(c.hex)} profile">${swatch}</a>` : swatch;
      })
      .join(" ");
  }

  function rowHTML(f) {
    return `
      <tr data-id="${f.id}">
        <td>${escapeHTML(f.name)}</td>
        <td>${escapeHTML(f.brand)}</td>
        <td>${escapeHTML(f.material)}</td>
        <td>${colorSwatches(f)}</td>
        <td class="num">${fmtMoney(f.price_per_kg)}</td>
        <td class="num">${fmtMoney(f.price_per_g)}</td>
        <td class="num">${fmtNum(f.length_per_roll_m, 1)} m</td>
        <td>${f.requires_dry_cabinet ? "Yes" : "—"}</td>
        <td>${f.glue_stick_recommended ? "Yes" : "—"}</td>
        <td>${(f.recommended_build_plate_names || []).map(escapeHTML).join(", ") || "—"}</td>
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
            <th>Name</th><th>Brand</th><th>Material</th><th>Colors</th>
            <th class="num">€/kg</th><th class="num">€/g</th><th class="num">Length/roll</th>
            <th>Dry cabinet</th><th>Glue stick</th><th>Recommended plates</th><th></th>
          </tr></thead>
          <tbody>
            ${items.length ? items.map(rowHTML).join("") : `<tr><td colspan="11" class="empty-state">No filaments yet.</td></tr>`}
          </tbody>
        </table>
      </div>
    `;

    if (!showForm) {
      container.querySelector("#add-btn").addEventListener("click", () => { editingId = "new"; draw(); });
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
      return;
    }

    const form = container.querySelector("#filament-form");
    let colors = (editingItem?.colors || []).map((c) => ({ ...c }));
    const selectedPlateIds = new Set(editingItem?.recommended_build_plate_ids || []);

    function colorRowHTML(c, idx) {
      return `
        <div class="row-item" data-idx="${idx}" style="grid-template-columns: 140px 1fr 32px;">
          <input type="text" class="color-hex" placeholder="#RRGGBB" value="${escapeHTML(c.hex || "")}">
          <input type="url" class="color-profile" placeholder="Link to this color's profile" value="${escapeHTML(c.profile_url || "")}">
          <button type="button" class="remove-row-btn remove-color" title="Remove">×</button>
        </div>`;
    }

    function drawColorRows() {
      const list = form.querySelector("#colors-list");
      list.innerHTML = colors.map(colorRowHTML).join("") || `<p class="hint">No colors added.</p>`;
      list.querySelectorAll(".row-item").forEach((rowEl) => {
        const idx = Number(rowEl.dataset.idx);
        rowEl.querySelector(".color-hex").addEventListener("input", (e) => { colors[idx].hex = e.target.value; });
        rowEl.querySelector(".color-profile").addEventListener("input", (e) => { colors[idx].profile_url = e.target.value; });
        rowEl.querySelector(".remove-color").addEventListener("click", () => { colors.splice(idx, 1); drawColorRows(); });
      });
    }
    drawColorRows();

    form.querySelector("#add-color").addEventListener("click", () => {
      colors.push({ hex: "", profile_url: "" });
      drawColorRows();
    });

    form.querySelector("#plates-list").innerHTML = buildPlates.length
      ? buildPlates
          .map(
            (p) => `
        <label style="display:flex; align-items:center; gap:6px; font-weight:400;">
          <input type="checkbox" class="plate-check" value="${p.id}" ${selectedPlateIds.has(p.id) ? "checked" : ""}>
          ${escapeHTML(p.name)}
        </label>`
          )
          .join("")
      : `<p class="hint">No build plates yet — add one under Equipment first.</p>`;

    const updatePreview = () => {
      const d = derivedValues(collectFormData(form, colors, buildPlates));
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
      const data = collectFormData(form, colors, buildPlates);
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

  await load();
}
