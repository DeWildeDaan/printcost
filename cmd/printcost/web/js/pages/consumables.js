import { consumablesApi } from "../api.js";
import { fmtMoney, escapeHTML } from "../utils.js";

function formHTML(item) {
  const v = item || {};
  return `
    <form id="consumable-form" class="card" style="margin-bottom:20px;">
      <h2 style="margin-top:0;">${item ? "Edit" : "Add"} Consumable</h2>
      <div class="form-grid">
        <div class="field"><label>Name *</label><input name="name" required placeholder="Glue stick, IPA wipe, ..." value="${escapeHTML(v.name)}"></div>
        <div class="field"><label>Unit label *</label><input name="unit_label" required placeholder="piece, ml, g, sheet" value="${escapeHTML(v.unit_label)}"></div>
        <div class="field"><label>Unit cost *</label><input type="number" step="0.01" required name="unit_cost" value="${v.unit_cost ?? ""}"></div>
        <div class="field">
          <label>&nbsp;</label>
          <div class="checkbox-row">
            <input type="checkbox" id="default_on_quote" name="default_on_quote" ${v.default_on_quote ? "checked" : ""}>
            <label for="default_on_quote" style="margin:0;">Add to new quotes by default</label>
          </div>
        </div>
        <div class="field span2"><label>Notes</label><textarea name="notes">${escapeHTML(v.notes)}</textarea></div>
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
    name: form.elements.name.value,
    unit_label: form.elements.unit_label.value,
    unit_cost: parseFloat(form.elements.unit_cost.value) || 0,
    default_on_quote: form.elements.default_on_quote.checked,
    notes: form.elements.notes.value,
  };
}

export async function renderConsumables(container) {
  let items = [];
  let editingId = null;

  async function load() {
    const res = await consumablesApi.list();
    items = res.data;
    draw();
  }

  function rowHTML(c) {
    return `
      <tr data-id="${c.id}">
        <td>${escapeHTML(c.name)}</td>
        <td>${escapeHTML(c.unit_label)}</td>
        <td class="num">${fmtMoney(c.unit_cost)}</td>
        <td>${c.default_on_quote ? "Yes" : "—"}</td>
        <td class="table-actions">
          <button class="small edit-btn" data-id="${c.id}">Edit</button>
          <button class="small danger delete-btn" data-id="${c.id}">Delete</button>
        </td>
      </tr>`;
  }

  function draw() {
    const showForm = editingId !== null;
    const editingItem = editingId && editingId !== "new" ? items.find((i) => i.id === editingId) : null;

    container.innerHTML = `
      <div class="page-header">
        <h1>Consumables</h1>
        ${!showForm ? `<button class="primary" id="add-btn">Add Consumable</button>` : ""}
      </div>
      ${showForm ? formHTML(editingItem) : ""}
      <div class="card">
        <table>
          <thead><tr>
            <th>Name</th><th>Unit</th><th class="num">Unit cost</th><th>Default on quote</th><th></th>
          </tr></thead>
          <tbody>
            ${items.length ? items.map(rowHTML).join("") : `<tr><td colspan="5" class="empty-state">No consumables yet.</td></tr>`}
          </tbody>
        </table>
      </div>
    `;

    if (!showForm) {
      container.querySelector("#add-btn").addEventListener("click", () => { editingId = "new"; draw(); });
    } else {
      const form = container.querySelector("#consumable-form");
      form.addEventListener("submit", async (e) => {
        e.preventDefault();
        const data = collectFormData(form);
        try {
          if (editingItem) {
            await consumablesApi.update(editingItem.id, data);
          } else {
            await consumablesApi.create(data);
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
        if (!confirm("Delete this consumable? This cannot be undone.")) return;
        try {
          await consumablesApi.remove(Number(btn.dataset.id));
          await load();
        } catch (err) {
          alert(err.message);
        }
      });
    });
  }

  await load();
}
