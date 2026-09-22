import { machinesApi, buildPlatesApi, nozzlesApi, amsUnitsApi, otherEquipmentApi } from "../api.js";
import { fmtMoney, fmtNum, escapeHTML } from "../utils.js";

const CONFIGS = {
  machines: {
    title: "Printers",
    api: machinesApi,
    fields: [
      { key: "name", label: "Name", type: "text", required: true },
      { key: "type_tag", label: "Type", type: "text", placeholder: "FDM, Resin, ..." },
      { key: "purchase_price", label: "Purchase price", type: "number", required: true, step: "0.01" },
      { key: "lifespan_hours", label: "Expected lifespan (hours)", type: "number", required: true, step: "1" },
      { key: "service_cost", label: "Total service cost", type: "number", step: "0.01" },
      { key: "power_kw", label: "Power consumption (kW)", type: "number", required: true, step: "0.01" },
      { key: "notes", label: "Notes", type: "textarea", span2: true },
    ],
    derivedKey: "hourly_depreciation",
    derivedLabel: (v) => `Hourly depreciation: ${fmtMoney(v)}/h`,
    computeDerived: (f) => (f.lifespan_hours > 0 ? (Number(f.purchase_price || 0) + Number(f.service_cost || 0)) / f.lifespan_hours : 0),
    columns: [
      { key: "name", label: "Name" },
      { key: "type_tag", label: "Type" },
      { key: "purchase_price", label: "Price", num: true, fmt: fmtMoney },
      { key: "lifespan_hours", label: "Lifespan (h)", num: true },
      { key: "hourly_depreciation", label: "€/hour", num: true, fmt: fmtMoney },
    ],
  },
  "build-plates": {
    title: "Build Plates",
    api: buildPlatesApi,
    fields: [
      { key: "name", label: "Name", type: "text", required: true },
      { key: "purchase_price", label: "Purchase price", type: "number", required: true, step: "0.01" },
      { key: "lifespan_hours", label: "Expected lifespan (hours)", type: "number", required: true, step: "1" },
      { key: "notes", label: "Notes", type: "textarea", span2: true },
    ],
    derivedKey: "hourly_cost",
    derivedLabel: (v) => `Hourly cost: ${fmtMoney(v)}/h`,
    computeDerived: (f) => (f.lifespan_hours > 0 ? Number(f.purchase_price || 0) / f.lifespan_hours : 0),
    columns: [
      { key: "name", label: "Name" },
      { key: "purchase_price", label: "Price", num: true, fmt: fmtMoney },
      { key: "lifespan_hours", label: "Lifespan (h)", num: true },
      { key: "hourly_cost", label: "€/hour", num: true, fmt: fmtMoney },
    ],
  },
  nozzles: {
    title: "Nozzles",
    api: nozzlesApi,
    fields: [
      { key: "name", label: "Name", type: "text", required: true },
      { key: "material", label: "Material", type: "text", placeholder: "Hardened steel, Brass, ..." },
      { key: "diameter_mm", label: "Diameter (mm)", type: "number", required: true, step: "0.01" },
      { key: "purchase_price", label: "Purchase price", type: "number", required: true, step: "0.01" },
      { key: "lifespan_hours", label: "Expected lifespan (hours)", type: "number", required: true, step: "1" },
      { key: "notes", label: "Notes", type: "textarea", span2: true },
    ],
    derivedKey: "hourly_cost",
    derivedLabel: (v) => `Hourly cost: ${fmtMoney(v)}/h`,
    computeDerived: (f) => (f.lifespan_hours > 0 ? Number(f.purchase_price || 0) / f.lifespan_hours : 0),
    columns: [
      { key: "name", label: "Name" },
      { key: "material", label: "Material" },
      { key: "diameter_mm", label: "Ø (mm)", num: true },
      { key: "purchase_price", label: "Price", num: true, fmt: fmtMoney },
      { key: "hourly_cost", label: "€/hour", num: true, fmt: fmtMoney },
    ],
  },
  "ams-units": {
    title: "AMS Units",
    api: amsUnitsApi,
    fields: [
      { key: "name", label: "Name", type: "text", required: true },
      { key: "purchase_price", label: "Purchase price", type: "number", required: true, step: "0.01" },
      { key: "lifespan_hours", label: "Expected lifespan (hours)", type: "number", required: true, step: "1" },
      { key: "service_cost", label: "Service cost over lifetime", type: "number", step: "0.01" },
      { key: "power_kw", label: "Power consumption (kW)", type: "number", step: "0.01" },
      { key: "notes", label: "Notes", type: "textarea", span2: true },
    ],
    derivedKey: "hourly_cost",
    derivedLabel: (v) => `Hourly cost: ${fmtMoney(v)}/h`,
    computeDerived: (f) => (f.lifespan_hours > 0 ? (Number(f.purchase_price || 0) + Number(f.service_cost || 0)) / f.lifespan_hours : 0),
    columns: [
      { key: "name", label: "Name" },
      { key: "purchase_price", label: "Price", num: true, fmt: fmtMoney },
      { key: "lifespan_hours", label: "Lifespan (h)", num: true },
      { key: "power_kw", label: "kW", num: true },
      { key: "hourly_cost", label: "€/hour", num: true, fmt: fmtMoney },
    ],
  },
  "other-equipment": {
    title: "Other Equipment",
    subtitle: "Ancillary equipment such as a filament dry cabinet. Filaments can be flagged as requiring one; the cost is added automatically to a quote when drying is enabled.",
    api: otherEquipmentApi,
    fields: [
      { key: "name", label: "Name", type: "text", required: true, placeholder: "Dry cabinet, ..." },
      { key: "purchase_price", label: "Purchase price", type: "number", required: true, step: "0.01" },
      { key: "lifespan_hours", label: "Expected lifespan (hours)", type: "number", required: true, step: "1" },
      { key: "service_cost", label: "Service cost over lifetime", type: "number", step: "0.01" },
      { key: "power_kw", label: "Power consumption (kW)", type: "number", step: "0.01" },
      { key: "notes", label: "Notes", type: "textarea", span2: true },
    ],
    derivedKey: "hourly_cost",
    derivedLabel: (v) => `Hourly cost: ${fmtMoney(v)}/h`,
    computeDerived: (f) => (f.lifespan_hours > 0 ? (Number(f.purchase_price || 0) + Number(f.service_cost || 0)) / f.lifespan_hours : 0),
    columns: [
      { key: "name", label: "Name" },
      { key: "purchase_price", label: "Price", num: true, fmt: fmtMoney },
      { key: "lifespan_hours", label: "Lifespan (h)", num: true },
      { key: "power_kw", label: "kW", num: true },
      { key: "hourly_cost", label: "€/hour", num: true, fmt: fmtMoney },
    ],
  },
};

export async function renderEquipmentPage(container, resourceKey) {
  const config = CONFIGS[resourceKey];
  let items = [];
  let editingId = null;

  async function load() {
    const res = await config.api.list();
    items = res.data;
    draw();
  }

  function fieldInput(field, value) {
    const val = value ?? "";
    if (field.type === "textarea") {
      return `<textarea name="${field.key}" ${field.span2 ? "" : ""}>${escapeHTML(val)}</textarea>`;
    }
    const step = field.step ? ` step="${field.step}"` : "";
    const req = field.required ? " required" : "";
    const placeholder = field.placeholder ? ` placeholder="${escapeHTML(field.placeholder)}"` : "";
    return `<input type="${field.type}" name="${field.key}"${step}${req}${placeholder} value="${escapeHTML(val)}">`;
  }

  function formHTML(item) {
    const fields = config.fields
      .map((f) => `
        <div class="field ${f.span2 ? "span2" : ""}">
          <label>${f.label}${f.required ? " *" : ""}</label>
          ${fieldInput(f, item ? item[f.key] : "")}
        </div>`)
      .join("");
    return `
      <form id="equip-form" class="card" style="margin-bottom:20px;">
        <h2 style="margin-top:0;">${item ? "Edit" : "Add"} ${config.title.replace(/s$/, "")}</h2>
        <div class="form-grid">${fields}</div>
        <div id="derived-preview" style="margin-top:12px;"></div>
        <div style="margin-top:16px; display:flex; gap:8px;">
          <button type="submit" class="primary">${item ? "Save changes" : "Add"}</button>
          <button type="button" id="cancel-form">Cancel</button>
        </div>
      </form>
    `;
  }

  function rowHTML(item) {
    const cells = config.columns
      .map((c) => `<td class="${c.num ? "num" : ""}">${c.fmt ? c.fmt(item[c.key]) : escapeHTML(item[c.key] ?? "")}</td>`)
      .join("");
    return `
      <tr data-id="${item.id}">
        ${cells}
        <td class="table-actions">
          <button class="small edit-btn" data-id="${item.id}">Edit</button>
          <button class="small danger delete-btn" data-id="${item.id}">Delete</button>
        </td>
      </tr>`;
  }

  function draw() {
    const showForm = editingId !== null;
    const editingItem = editingId && editingId !== "new" ? items.find((i) => i.id === editingId) : null;

    container.innerHTML = `
      <div class="page-header">
        <div>
          <h1>${config.title}</h1>
          ${config.subtitle ? `<p class="page-subtitle">${config.subtitle}</p>` : ""}
        </div>
        ${!showForm ? `<button class="primary" id="add-btn">Add ${config.title.replace(/s$/, "")}</button>` : ""}
      </div>
      ${showForm ? formHTML(editingItem) : ""}
      <div class="card">
        <table>
          <thead><tr>${config.columns.map((c) => `<th class="${c.num ? "num" : ""}">${c.label}</th>`).join("")}<th></th></tr></thead>
          <tbody>
            ${items.length ? items.map(rowHTML).join("") : `<tr><td colspan="${config.columns.length + 1}" class="empty-state">No records yet.</td></tr>`}
          </tbody>
        </table>
      </div>
    `;

    if (!showForm) {
      container.querySelector("#add-btn").addEventListener("click", () => {
        editingId = "new";
        draw();
      });
    } else {
      const form = container.querySelector("#equip-form");
      const updatePreview = () => {
        const data = collectFormData(form);
        const value = config.computeDerived(data);
        container.querySelector("#derived-preview").innerHTML = `<span class="derived-value">${config.derivedLabel(value)}</span>`;
      };
      form.addEventListener("input", updatePreview);
      updatePreview();

      form.addEventListener("submit", async (e) => {
        e.preventDefault();
        const data = collectFormData(form);
        try {
          if (editingItem) {
            await config.api.update(editingItem.id, data);
          } else {
            await config.api.create(data);
          }
          editingId = null;
          await load();
        } catch (err) {
          alert(err.message);
        }
      });
      container.querySelector("#cancel-form").addEventListener("click", () => {
        editingId = null;
        draw();
      });
    }

    container.querySelectorAll(".edit-btn").forEach((btn) => {
      btn.addEventListener("click", () => {
        editingId = Number(btn.dataset.id);
        draw();
      });
    });
    container.querySelectorAll(".delete-btn").forEach((btn) => {
      btn.addEventListener("click", async () => {
        if (!confirm("Delete this record? This cannot be undone.")) return;
        try {
          await config.api.remove(Number(btn.dataset.id));
          await load();
        } catch (err) {
          alert(err.message);
        }
      });
    });
  }

  function collectFormData(form) {
    const data = {};
    for (const field of config.fields) {
      const el = form.elements[field.key];
      if (field.type === "number") {
        data[field.key] = el.value === "" ? 0 : parseFloat(el.value);
      } else {
        data[field.key] = el.value;
      }
    }
    return data;
  }

  await load();
}
