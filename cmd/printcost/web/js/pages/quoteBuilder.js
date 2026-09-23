import {
  machinesApi, buildPlatesApi, nozzlesApi, amsUnitsApi, otherEquipmentApi,
  filamentsApi, consumablesApi, settingsApi, quotesApi,
} from "../api.js";
import { fmtMoney, escapeHTML, minutesToHHMM, hhmmToMinutes, todayPlusMonths } from "../utils.js";
import { calculate } from "../calc.js";

function opt(id, label, selectedId) {
  return `<option value="${id}" ${String(id) === String(selectedId) ? "selected" : ""}>${escapeHTML(label)}</option>`;
}

function selectHTML(name, items, selectedId, emptyLabel = "— none —") {
  return `<select name="${name}"><option value="">${emptyLabel}</option>${items.map((i) => opt(i.id, i.name, selectedId)).join("")}</select>`;
}

export async function renderQuoteBuilder(container, quoteId) {
  const isEdit = !!quoteId;

  const [settings, machinesRes, platesRes, nozzlesRes, amsRes, otherEquipRes, filamentsRes, consumablesRes, existing] =
    await Promise.all([
      settingsApi.get(),
      machinesApi.list(),
      buildPlatesApi.list(),
      nozzlesApi.list(),
      amsUnitsApi.list(),
      otherEquipmentApi.list(),
      filamentsApi.list(),
      consumablesApi.list(),
      isEdit ? quotesApi.get(quoteId) : Promise.resolve(null),
    ]);

  const lookups = {
    machines: machinesRes.data,
    buildPlates: platesRes.data,
    nozzles: nozzlesRes.data,
    ams: amsRes.data,
    otherEquipment: otherEquipRes.data,
    filaments: filamentsRes.data,
    consumables: consumablesRes.data,
  };

  const byId = (list, id) => list.find((i) => String(i.id) === String(id)) || null;

  const q = existing || {
    name: "", status: "draft", client_name: "", client_email: "", notes: "",
    valid_until: todayPlusMonths(1),
    machine_id: lookups.machines[0]?.id ?? "",
    build_plate_id: lookups.buildPlates[0]?.id ?? "",
    nozzle_id: lookups.nozzles[0]?.id ?? "",
    ams_id: lookups.ams[0]?.id ?? "",
    dry_cabinet_id: "",
    ams_used_for_printing: true,
    filament_id: "", print_weight_g: "", print_time_min: 0,
    supports_enabled: false, support_filament_id: "", support_weight_g: "",
    prototypes_enabled: false, prototype_filament_id: "", prototype_count: "", prototype_weight_g: "", prototype_time_min: 0,
    drying_enabled: false, drying_hours: "",
    additional_cost: "", additional_cost_note: "",
    markup_pct: settings.markup,
    processing_steps: [],
    consumables: lookups.consumables
      .filter((c) => c.default_on_quote)
      .map((c) => ({ consumable_id: c.id, qty: 1, unit_cost: c.unit_cost })),
  };

  // Mutable working copies of the dynamic row lists.
  let steps = (q.processing_steps || []).map((s) => ({ ...s }));
  let quoteConsumables = (q.consumables || []).map((c) => ({ ...c }));

  function stepRowHTML(step, idx) {
    return `
      <div class="row-item" data-idx="${idx}">
        <input type="text" class="step-label" placeholder="Step label" value="${escapeHTML(step.label || "")}">
        <input type="number" step="1" class="step-minutes" placeholder="Minutes" value="${step.minutes ?? ""}">
        <button type="button" class="remove-row-btn remove-step" title="Remove">×</button>
      </div>`;
  }

  function consumableRowHTML(row, idx) {
    return `
      <div class="row-item consumable-row" data-idx="${idx}">
        ${selectHTML("consumable_select", lookups.consumables, row.consumable_id, "— select —")}
        <input type="number" step="0.01" class="cons-qty" placeholder="Qty" value="${row.qty ?? ""}">
        <input type="number" step="0.01" class="cons-unit-cost" placeholder="Unit cost" value="${row.unit_cost ?? ""}">
        <span class="num cons-line-total mono">${fmtMoney((row.qty || 0) * (row.unit_cost || 0))}</span>
        <button type="button" class="remove-row-btn remove-cons" title="Remove">×</button>
      </div>`;
  }

  function drawStepRows() {
    container.querySelector("#steps-list").innerHTML = steps.map(stepRowHTML).join("") || `<p class="hint">No processing steps added.</p>`;
    bindStepRowEvents();
    updateStepsTotal();
  }
  function updateStepsTotal() {
    const totalMinutes = steps.reduce((sum, st) => sum + (Number(st.minutes) || 0), 0);
    const h = Math.floor(totalMinutes / 60);
    const m = Math.round(totalMinutes % 60);
    container.querySelector("#steps-total").textContent = `Total: ${totalMinutes} min (${h}h ${m}m)`;
  }
  function drawConsumableRows() {
    container.querySelector("#cons-list").innerHTML = quoteConsumables.map(consumableRowHTML).join("") || `<p class="hint">No consumables added.</p>`;
    bindConsumableRowEvents();
  }

  function bindStepRowEvents() {
    const list = container.querySelector("#steps-list");
    list.querySelectorAll(".row-item").forEach((rowEl) => {
      const idx = Number(rowEl.dataset.idx);
      rowEl.querySelector(".step-label").addEventListener("input", (e) => { steps[idx].label = e.target.value; });
      rowEl.querySelector(".step-minutes").addEventListener("input", (e) => { steps[idx].minutes = parseFloat(e.target.value) || 0; updateStepsTotal(); updatePreview(); });
      rowEl.querySelector(".remove-step").addEventListener("click", () => { steps.splice(idx, 1); drawStepRows(); updatePreview(); });
    });
  }
  function bindConsumableRowEvents() {
    const list = container.querySelector("#cons-list");
    list.querySelectorAll(".row-item").forEach((rowEl) => {
      const idx = Number(rowEl.dataset.idx);
      const select = rowEl.querySelector(".consumable_select") || rowEl.querySelector("select");
      select.addEventListener("change", (e) => {
        quoteConsumables[idx].consumable_id = e.target.value ? Number(e.target.value) : "";
        const c = byId(lookups.consumables, e.target.value);
        if (c) {
          quoteConsumables[idx].unit_cost = c.unit_cost;
          rowEl.querySelector(".cons-unit-cost").value = c.unit_cost;
        }
        updateConsumableLineTotal(rowEl, idx);
        updatePreview();
      });
      rowEl.querySelector(".cons-qty").addEventListener("input", (e) => {
        quoteConsumables[idx].qty = parseFloat(e.target.value) || 0;
        updateConsumableLineTotal(rowEl, idx);
        updatePreview();
      });
      rowEl.querySelector(".cons-unit-cost").addEventListener("input", (e) => {
        quoteConsumables[idx].unit_cost = parseFloat(e.target.value) || 0;
        updateConsumableLineTotal(rowEl, idx);
        updatePreview();
      });
      rowEl.querySelector(".remove-cons").addEventListener("click", () => { quoteConsumables.splice(idx, 1); drawConsumableRows(); updatePreview(); });
    });
  }
  function updateConsumableLineTotal(rowEl, idx) {
    const row = quoteConsumables[idx];
    rowEl.querySelector(".cons-line-total").textContent = fmtMoney((row.qty || 0) * (row.unit_cost || 0));
  }

  function collapsibleSection(id, title, bodyHTML, enabledByDefault) {
    return `
      <div class="section">
        <div class="section-header" data-toggle="${id}">
          <h3>${title}</h3>
          <span class="toggle-hint">${enabledByDefault ? "Hide" : "Show"}</span>
        </div>
        <div class="section-body ${enabledByDefault ? "" : "collapsed"}" id="${id}">
          ${bodyHTML}
        </div>
      </div>`;
  }

  container.innerHTML = `
    <div class="page-header">
      <h1>${isEdit ? "Edit Quote" : "New Quote"}</h1>
    </div>
    ${isEdit ? `<div class="banner">Prices will be recalculated from current equipment and filament costs.</div>` : ""}
    <div id="error-slot"></div>
    <form id="quote-form">
      <div class="section">
        <h3 style="margin-top:0;">Quote details</h3>
        <div class="form-grid">
          <div class="field"><label>Name *</label><input type="text" name="name" required value="${escapeHTML(q.name)}"></div>
          <div class="field"><label>Status</label>
            <select name="status">
              ${["draft", "sent", "paid", "cancelled"].map((s) => `<option value="${s}" ${s === q.status ? "selected" : ""}>${s}</option>`).join("")}
            </select>
          </div>
          <div class="field"><label>Client name</label><input type="text" name="client_name" value="${escapeHTML(q.client_name)}"></div>
          <div class="field"><label>Client email</label><input type="email" name="client_email" value="${escapeHTML(q.client_email)}"></div>
          <div class="field"><label>Valid until</label><input type="date" name="valid_until" value="${q.valid_until || ""}"></div>
          <div class="field span2"><label>Notes / internal memo</label><textarea name="notes">${escapeHTML(q.notes)}</textarea></div>
        </div>
      </div>

      <div class="section">
        <h3 style="margin-top:0;">Equipment</h3>
        <div class="form-grid">
          <div class="field"><label>Printer</label>${selectHTML("machine_id", lookups.machines, q.machine_id)}</div>
          <div class="field"><label>Build plate</label>${selectHTML("build_plate_id", lookups.buildPlates, q.build_plate_id)}</div>
          <div class="field"><label>Nozzle</label>${selectHTML("nozzle_id", lookups.nozzles, q.nozzle_id)}</div>
          <div class="field"><label>AMS unit</label>${selectHTML("ams_id", lookups.ams, q.ams_id)}</div>
        </div>
        <div class="field" id="ams-toggle-field" style="margin-top:10px; display:none;">
          <div class="checkbox-row">
            <input type="checkbox" id="ams_used_for_printing" name="ams_used_for_printing" ${q.ams_used_for_printing ? "checked" : ""}>
            <label for="ams_used_for_printing" style="margin:0;">AMS is used for printing (adds its cost across the print + prototype time)</label>
          </div>
        </div>
      </div>

      <div class="section">
        <h3 style="margin-top:0;">Main print</h3>
        <div class="form-grid">
          <div class="field">
            <label>Filament *</label>${selectHTML("filament_id", lookups.filaments, q.filament_id, "— select —")}
            <div id="filament-colors" class="hint" style="margin-top:4px;"></div>
          </div>
          <div class="field"><label>Print weight (g) *</label><input type="number" step="0.1" name="print_weight_g" required value="${q.print_weight_g}"></div>
          <div class="field"><label>Print time (HH:MM) *</label><input type="text" name="print_time_hhmm" required pattern="\\d{1,3}:\\d{2}" value="${minutesToHHMM(q.print_time_min)}"></div>
        </div>
      </div>

      ${collapsibleSection("supports-body", "Support material", `
        <div class="checkbox-row" style="margin-bottom:12px;">
          <input type="checkbox" id="supports_enabled" name="supports_enabled" ${q.supports_enabled ? "checked" : ""}>
          <label for="supports_enabled" style="margin:0;">Enable supports</label>
        </div>
        <div class="form-grid">
          <div class="field"><label>Support filament</label>${selectHTML("support_filament_id", lookups.filaments, q.support_filament_id, "— select —")}</div>
          <div class="field"><label>Support weight (g)</label><input type="number" step="0.1" name="support_weight_g" value="${q.support_weight_g}"></div>
        </div>
      `, q.supports_enabled)}

      ${collapsibleSection("prototypes-body", "Prototype / first-layer prints", `
        <div class="checkbox-row" style="margin-bottom:12px;">
          <input type="checkbox" id="prototypes_enabled" name="prototypes_enabled" ${q.prototypes_enabled ? "checked" : ""}>
          <label for="prototypes_enabled" style="margin:0;">Enable prototypes</label>
        </div>
        <div class="form-grid">
          <div class="field"><label>Prototype filament</label>${selectHTML("prototype_filament_id", lookups.filaments, q.prototype_filament_id, "— select —")}</div>
          <div class="field"><label>Number of prototypes</label><input type="number" step="0.1" name="prototype_count" value="${q.prototype_count}"></div>
          <div class="field"><label>Prototype weight (g)</label><input type="number" step="0.1" name="prototype_weight_g" value="${q.prototype_weight_g}"></div>
          <div class="field"><label>Prototype print time (HH:MM)</label><input type="text" name="prototype_time_hhmm" pattern="\\d{1,3}:\\d{2}" value="${minutesToHHMM(q.prototype_time_min)}"></div>
        </div>
      `, q.prototypes_enabled)}

      ${collapsibleSection("drying-body", "Filament drying", `
        <div class="checkbox-row" style="margin-bottom:12px;">
          <input type="checkbox" id="drying_enabled" name="drying_enabled" ${q.drying_enabled ? "checked" : ""}>
          <label for="drying_enabled" style="margin:0;">Enable drying</label>
        </div>
        <div class="form-grid">
          <div class="field"><label>Drying hours</label><input type="number" step="0.1" name="drying_hours" value="${q.drying_hours}"></div>
          <div class="field"><label>Dry cabinet</label>${selectHTML("dry_cabinet_id", lookups.otherEquipment, q.dry_cabinet_id, "— none —")}</div>
        </div>
        <p class="hint">Drying is done in the dry cabinet OR the AMS, never both: the cabinet is used automatically if selected and a filament in this quote requires one, otherwise drying falls back to the AMS unit's hourly cost.</p>
      `, q.drying_enabled)}

      <div class="section">
        <h3 style="margin-top:0;">Pre- and post-processing</h3>
        <div class="row-list" id="steps-list"></div>
        <div id="steps-total" class="hint" style="margin-top:8px;"></div>
        <div style="display:flex; gap:6px; flex-wrap:wrap; margin-top:10px;">
          <button type="button" class="small quick-step" data-label="Model preparation (Fixing, CAD…)">+ Model preparation</button>
          <button type="button" class="small quick-step" data-label="Slicing (Supports, Parameters…)">+ Slicing</button>
          <button type="button" class="small quick-step" data-label="Support removal">+ Support removal</button>
          <button type="button" class="small quick-step" data-label="Additional work">+ Additional work</button>
          <button type="button" id="add-step" class="small">+ Custom step</button>
        </div>
      </div>

      <div class="section">
        <h3 style="margin-top:0;">Consumables used</h3>
        <div class="row-list" id="cons-list"></div>
        <button type="button" id="add-cons" class="small" style="margin-top:10px;">+ Add consumable</button>
      </div>

      <div class="section">
        <h3 style="margin-top:0;">Additional costs & markup</h3>
        <div class="form-grid">
          <div class="field"><label>Additional cost amount</label><input type="number" step="0.01" name="additional_cost" value="${q.additional_cost}"></div>
          <div class="field"><label>Additional cost note</label><input type="text" name="additional_cost_note" placeholder="Shipping, rush fee, ..." value="${escapeHTML(q.additional_cost_note)}"></div>
          <div class="field"><label>Markup (%)</label><input type="number" step="0.1" name="markup_pct" value="${q.markup_pct}"></div>
        </div>
      </div>

      <div class="section" id="preview-section">
        <h3 style="margin-top:0;">Cost preview</h3>
        <div id="cost-preview"></div>
      </div>

      <div style="display:flex; gap:8px; margin-bottom:40px;">
        <button type="submit" class="primary">${isEdit ? "Save changes" : "Create quote"}</button>
        <a class="btn" href="#/quotes">Cancel</a>
      </div>
    </form>
  `;

  const form = container.querySelector("#quote-form");

  container.querySelectorAll(".section-header[data-toggle]").forEach((header) => {
    header.addEventListener("click", () => {
      const body = container.querySelector(`#${header.dataset.toggle}`);
      body.classList.toggle("collapsed");
      header.querySelector(".toggle-hint").textContent = body.classList.contains("collapsed") ? "Show" : "Hide";
    });
  });

  container.querySelector("#add-step").addEventListener("click", () => {
    steps.push({ label: "", minutes: 0 });
    drawStepRows();
  });
  container.querySelectorAll(".quick-step").forEach((btn) => {
    btn.addEventListener("click", () => {
      steps.push({ label: btn.dataset.label, minutes: 0 });
      drawStepRows();
      updatePreview();
    });
  });
  container.querySelector("#add-cons").addEventListener("click", () => {
    quoteConsumables.push({ consumable_id: "", qty: 1, unit_cost: 0 });
    drawConsumableRows();
  });

  function drawFilamentColors() {
    const f = byId(lookups.filaments, form.elements.filament_id.value);
    const colorsEl = container.querySelector("#filament-colors");
    if (!f || !f.colors || !f.colors.length) {
      colorsEl.innerHTML = "";
      return;
    }
    colorsEl.innerHTML =
      "Available colors: " +
      f.colors
        .map((c) => {
          const swatch = `<span class="swatch" style="background:${escapeHTML(c.hex)}; border:1px solid var(--border);"></span>`;
          return c.profile_url ? `<a href="${escapeHTML(c.profile_url)}" target="_blank" rel="noopener">${swatch}${escapeHTML(c.hex)}</a>` : `${swatch}${escapeHTML(c.hex)}`;
        })
        .join(" &nbsp; ");
  }
  drawFilamentColors();

  // Pre-fill drying hours from the selected filament's default when it changes,
  // but only if the user hasn't already set a value (avoid clobbering edits).
  form.elements.filament_id.addEventListener("change", (e) => {
    const f = byId(lookups.filaments, e.target.value);
    const dryingHoursInput = form.elements.drying_hours;
    if (f && f.drying_time_hours != null && !dryingHoursInput.value) {
      dryingHoursInput.value = f.drying_time_hours;
    }
    drawFilamentColors();
    updatePreview();
  });

  // Prototype weight/time default to (main print value * prototype count),
  // but stop auto-updating a field the moment the user types into it directly.
  let protoWeightTouched = isEdit;
  let protoTimeTouched = isEdit;

  function recalcPrototypeFields() {
    const printWeight = parseFloat(form.elements.print_weight_g.value) || 0;
    const printTimeMin = hhmmToMinutes(form.elements.print_time_hhmm.value);
    const count = parseFloat(form.elements.prototype_count.value) || 0;
    if (!protoWeightTouched) {
      form.elements.prototype_weight_g.value = printWeight * count || "";
    }
    if (!protoTimeTouched) {
      form.elements.prototype_time_hhmm.value = minutesToHHMM(printTimeMin * count);
    }
  }
  form.elements.prototype_weight_g.addEventListener("input", () => { protoWeightTouched = true; });
  form.elements.prototype_time_hhmm.addEventListener("input", () => { protoTimeTouched = true; });
  ["print_weight_g", "prototype_count"].forEach((name) => {
    form.elements[name].addEventListener("input", recalcPrototypeFields);
  });
  form.elements.print_time_hhmm.addEventListener("input", recalcPrototypeFields);
  form.elements.prototypes_enabled.addEventListener("change", () => {
    if (form.elements.prototypes_enabled.checked) recalcPrototypeFields();
  });

  function updateAmsToggleVisibility() {
    container.querySelector("#ams-toggle-field").style.display = form.elements.ams_id.value ? "block" : "none";
  }
  form.elements.ams_id.addEventListener("change", () => { updateAmsToggleVisibility(); updatePreview(); });
  updateAmsToggleVisibility();

  form.addEventListener("input", updatePreview);
  form.addEventListener("change", updatePreview);

  function readFormState() {
    const fd = new FormData(form);
    const val = (name) => fd.get(name);
    return {
      machine_id: val("machine_id") || null,
      build_plate_id: val("build_plate_id") || null,
      nozzle_id: val("nozzle_id") || null,
      ams_id: val("ams_id") || null,
      dry_cabinet_id: val("dry_cabinet_id") || null,
      ams_used_for_printing: form.elements.ams_used_for_printing.checked,
      filament_id: val("filament_id") || null,
      print_weight_g: parseFloat(val("print_weight_g")) || 0,
      print_time_min: hhmmToMinutes(val("print_time_hhmm")),
      supports_enabled: form.elements.supports_enabled.checked,
      support_filament_id: val("support_filament_id") || null,
      support_weight_g: parseFloat(val("support_weight_g")) || 0,
      prototypes_enabled: form.elements.prototypes_enabled.checked,
      prototype_filament_id: val("prototype_filament_id") || null,
      prototype_count: parseFloat(val("prototype_count")) || 0,
      prototype_weight_g: parseFloat(val("prototype_weight_g")) || 0,
      prototype_time_min: hhmmToMinutes(val("prototype_time_hhmm")),
      drying_enabled: form.elements.drying_enabled.checked,
      drying_hours: parseFloat(val("drying_hours")) || 0,
      additional_cost: parseFloat(val("additional_cost")) || 0,
      additional_cost_note: val("additional_cost_note") || "",
      markup_pct: parseFloat(val("markup_pct")) || 0,
    };
  }

  function updatePreview() {
    const s = readFormState();
    const result = calculate({
      settings,
      machine: byId(lookups.machines, s.machine_id),
      buildPlate: byId(lookups.buildPlates, s.build_plate_id),
      nozzle: byId(lookups.nozzles, s.nozzle_id),
      ams: byId(lookups.ams, s.ams_id),
      dryCabinet: byId(lookups.otherEquipment, s.dry_cabinet_id),
      amsUsedForPrinting: s.ams_used_for_printing,
      mainFilament: byId(lookups.filaments, s.filament_id),
      printWeightG: s.print_weight_g,
      printTimeMin: s.print_time_min,
      supportsEnabled: s.supports_enabled,
      supportFilament: byId(lookups.filaments, s.support_filament_id),
      supportWeightG: s.support_weight_g,
      prototypesEnabled: s.prototypes_enabled,
      prototypeFilament: byId(lookups.filaments, s.prototype_filament_id),
      prototypeWeightG: s.prototype_weight_g,
      prototypeTimeMin: s.prototype_time_min,
      dryingEnabled: s.drying_enabled,
      dryingHours: s.drying_hours,
      processingMinutes: steps.reduce((sum, st) => sum + (Number(st.minutes) || 0), 0),
      consumablesCost: quoteConsumables.reduce((sum, c) => sum + (Number(c.qty) || 0) * (Number(c.unit_cost) || 0), 0),
      additionalCost: s.additional_cost,
      markupPct: s.markup_pct,
    });

    const categories = [
      { key: "costFilament", label: "Filament", color: "var(--cat-1)" },
      { key: "costElectricity", label: "Electricity", color: "var(--cat-2)" },
      { key: "costDepreciation", label: "Machine depreciation", color: "var(--cat-3)" },
      { key: "costLabor", label: "Labour", color: "var(--cat-4)" },
      { key: "costConsumables", label: "Consumables", color: "var(--cat-5)" },
      { key: "costFailure", label: "Failed print allowance", color: "var(--cat-6)" },
      { key: "costMarkup", label: "Markup", color: "var(--cat-7)" },
    ];
    const total = result.suggestedPrice || 1;

    let gradientPos = 0;
    const gradientStops = categories.map((c) => {
      const pct = (result[c.key] / total) * 100;
      const from = gradientPos;
      gradientPos += pct;
      return `${c.color} ${from}% ${gradientPos}%`;
    }).join(", ");

    container.querySelector("#cost-preview").innerHTML = `
      <div class="cost-preview-layout">
        <table class="cat-table">
          <tbody>
            ${categories.map((c) => {
              const lines = (result.breakdown[c.key] || []);
              return `
                <tr class="cat-row" data-cat="${c.key}">
                  <td><span class="cat-swatch" style="background:${c.color}"></span>${c.label}</td>
                  <td class="num">${fmtMoney(result[c.key])}</td>
                </tr>
                <tr class="cat-detail collapsed" data-cat-detail="${c.key}">
                  <td colspan="2">
                    ${lines.length
                      ? `<ul class="cat-detail-list">${lines.map((l) => `<li><span>${escapeHTML(l.label)}</span><span class="num">${fmtMoney(l.value)}</span></li>`).join("")}</ul>`
                      : `<p class="hint">Nothing contributes to this category yet.</p>`}
                  </td>
                </tr>`;
            }).join("")}
          </tbody>
        </table>
        <div class="pie-wrap">
          <div class="pie-chart" style="background: conic-gradient(${gradientStops})"></div>
          <ul class="pie-legend">
            ${categories.map((c) => `<li><span class="cat-swatch" style="background:${c.color}"></span>${c.label}</li>`).join("")}
          </ul>
        </div>
      </div>
      <div class="suggested-price-box">
        <span class="suggested-price-label">Suggested price</span>
        <span class="suggested-price-value">${fmtMoney(result.suggestedPrice)}</span>
      </div>
    `;

    container.querySelectorAll(".cat-row").forEach((row) => {
      row.addEventListener("click", () => {
        const detail = container.querySelector(`[data-cat-detail="${row.dataset.cat}"]`);
        detail.classList.toggle("collapsed");
      });
    });
  }

  drawStepRows();
  drawConsumableRows();
  updatePreview();

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const s = readFormState();
    const fd = new FormData(form);

    const payload = {
      name: fd.get("name"),
      status: fd.get("status"),
      client_name: fd.get("client_name"),
      client_email: fd.get("client_email"),
      notes: fd.get("notes"),
      valid_until: fd.get("valid_until") || null,
      machine_id: s.machine_id ? Number(s.machine_id) : null,
      build_plate_id: s.build_plate_id ? Number(s.build_plate_id) : null,
      nozzle_id: s.nozzle_id ? Number(s.nozzle_id) : null,
      ams_id: s.ams_id ? Number(s.ams_id) : null,
      dry_cabinet_id: s.dry_cabinet_id ? Number(s.dry_cabinet_id) : null,
      ams_used_for_printing: s.ams_used_for_printing,
      filament_id: s.filament_id ? Number(s.filament_id) : null,
      print_weight_g: s.print_weight_g,
      print_time_min: s.print_time_min,
      supports_enabled: s.supports_enabled,
      support_filament_id: s.support_filament_id ? Number(s.support_filament_id) : null,
      support_weight_g: s.support_weight_g,
      prototypes_enabled: s.prototypes_enabled,
      prototype_filament_id: s.prototype_filament_id ? Number(s.prototype_filament_id) : null,
      prototype_count: s.prototype_count,
      prototype_weight_g: s.prototype_weight_g,
      prototype_time_min: s.prototype_time_min,
      drying_enabled: s.drying_enabled,
      drying_hours: s.drying_hours,
      additional_cost: s.additional_cost,
      additional_cost_note: s.additional_cost_note,
      markup_pct: s.markup_pct,
      processing_steps: steps.filter((st) => st.label).map((st) => ({ label: st.label, minutes: Number(st.minutes) || 0 })),
      consumables: quoteConsumables.filter((c) => c.consumable_id).map((c) => ({
        consumable_id: Number(c.consumable_id), qty: Number(c.qty) || 0, unit_cost: Number(c.unit_cost) || 0,
      })),
    };

    try {
      const saved = isEdit ? await quotesApi.update(quoteId, payload) : await quotesApi.create(payload);
      location.hash = `#/quotes/${saved.id}`;
    } catch (err) {
      container.querySelector("#error-slot").innerHTML = `<div class="error-banner">${escapeHTML(err.message)}</div>`;
      window.scrollTo(0, 0);
    }
  });
}
