import { materialSettingsApi, filamentsApi } from "../api.js";
import { escapeHTML } from "../utils.js";

function materialOptionsHTML(materials, selected) {
  const options = materials.includes(selected) || !selected ? materials : [selected, ...materials];
  return (
    `<option value="">— select —</option>` +
    options.map((m) => `<option value="${escapeHTML(m)}" ${m === selected ? "selected" : ""}>${escapeHTML(m)}</option>`).join("")
  );
}

function formHTML(item, materials) {
  const v = item || {};
  return `
    <form id="material-settings-form" class="card" style="margin-bottom:20px;">
      <h2 style="margin-top:0;">${item ? "Edit" : "Add"} Material Settings</h2>
      <div class="form-grid">
        <div class="field"><label>Material *</label><select name="material" required>${materialOptionsHTML(materials, v.material)}</select></div>
        <div class="field"><label>Supports</label><input name="supports" placeholder="Yes, organic" value="${escapeHTML(v.supports)}"></div>
        <div class="field"><label>Fuzzy skin</label><input name="fuzzy_skin" placeholder="Off" value="${escapeHTML(v.fuzzy_skin)}"></div>
        <div class="field"><label>Ironing</label><input name="ironing" placeholder="Top surfaces only" value="${escapeHTML(v.ironing)}"></div>
      </div>
      <div style="margin-top:16px; display:flex; gap:8px;">
        <button type="submit" class="primary">${item ? "Save changes" : "Add"}</button>
        <button type="button" id="cancel-form">Cancel</button>
      </div>
    </form>
  `;
}

function collectFormData(form) {
  return {
    material: form.elements.material.value,
    supports: form.elements.supports.value,
    fuzzy_skin: form.elements.fuzzy_skin.value,
    ironing: form.elements.ironing.value,
  };
}

export async function renderMaterialSettings(container) {
  let items = [];
  let materials = [];
  let editingId = null;

  async function load() {
    const [settingsRes, filamentsRes] = await Promise.all([materialSettingsApi.list(), filamentsApi.list()]);
    items = settingsRes.data;
    materials = [...new Set(filamentsRes.data.map((f) => f.material).filter(Boolean))].sort();
    draw();
  }

  function rowHTML(m) {
    return `
      <tr data-id="${m.id}">
        <td>${escapeHTML(m.material)}</td>
        <td>${escapeHTML(m.supports)}</td>
        <td>${escapeHTML(m.fuzzy_skin)}</td>
        <td>${escapeHTML(m.ironing)}</td>
        <td class="table-actions">
          <button class="small edit-btn" data-id="${m.id}">Edit</button>
          <button class="small danger delete-btn" data-id="${m.id}">Delete</button>
        </td>
      </tr>`;
  }

  function draw() {
    const showForm = editingId !== null;
    const editingItem = editingId && editingId !== "new" ? items.find((i) => i.id === editingId) : null;

    container.innerHTML = `
      <div class="page-header">
        <h1>Material Settings</h1>
        ${!showForm && materials.length ? `<button class="primary" id="add-btn">Add Material Settings</button>` : ""}
      </div>
      <p class="page-subtitle">Print-setting recommendations shared by every filament of that material.</p>
      ${!materials.length ? `<p class="hint">Add a filament first — materials come from the Filament Library.</p>` : ""}
      ${showForm ? formHTML(editingItem, materials) : ""}
      <div class="card">
        <table>
          <thead><tr>
            <th>Material</th><th>Supports</th><th>Fuzzy skin</th><th>Ironing</th><th></th>
          </tr></thead>
          <tbody>
            ${items.length ? items.map(rowHTML).join("") : `<tr><td colspan="5" class="empty-state">No material settings yet.</td></tr>`}
          </tbody>
        </table>
      </div>
    `;

    if (!showForm) {
      container.querySelector("#add-btn")?.addEventListener("click", () => { editingId = "new"; draw(); });
    } else {
      const form = container.querySelector("#material-settings-form");
      form.addEventListener("submit", async (e) => {
        e.preventDefault();
        const data = collectFormData(form);
        try {
          if (editingItem) {
            await materialSettingsApi.update(editingItem.id, data);
          } else {
            await materialSettingsApi.create(data);
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
        if (!confirm("Delete this material's settings? This cannot be undone.")) return;
        try {
          await materialSettingsApi.remove(Number(btn.dataset.id));
          await load();
        } catch (err) {
          alert(err.message);
        }
      });
    });
  }

  await load();
}
