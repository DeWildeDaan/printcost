import { quotesApi, consumablesApi } from "../api.js";
import { fmtMoney, fmtNum, fmtDate, fmtPct, statusBadge, escapeHTML, minutesToHHMM } from "../utils.js";

const COST_ITEMS = [
  { key: "cost_filament", label: "Filament", cls: "cost-filament" },
  { key: "cost_electricity", label: "Electricity", cls: "cost-electricity" },
  { key: "cost_depreciation", label: "Machine depreciation", cls: "cost-depreciation" },
  { key: "cost_labor", label: "Labour (pre/post processing)", cls: "cost-labor" },
  { key: "cost_consumables", label: "Consumables", cls: "cost-consumables" },
  { key: "cost_failure", label: "Failed print allowance", cls: "cost-failure" },
  { key: "cost_markup", label: "Markup", cls: "cost-markup" },
];

export async function renderQuoteDetail(container, id) {
  const [q, consumablesRes] = await Promise.all([quotesApi.get(id), consumablesApi.list()]);
  const consumableNames = Object.fromEntries(consumablesRes.data.map((c) => [c.id, c.name]));
  const total = q.cost_total || 1;

  const barSegments = COST_ITEMS.map((item) => {
    const pct = (q[item.key] / total) * 100;
    return `<div class="${item.cls}" style="width:${pct}%" title="${item.label}: ${fmtPct(pct)}"></div>`;
  }).join("");

  const legend = COST_ITEMS.map((item) => {
    const pct = (q[item.key] / total) * 100;
    return `<span><span class="swatch ${item.cls}"></span>${item.label}: ${fmtPct(pct)}</span>`;
  }).join("");

  const breakdownRows = COST_ITEMS.map(
    (item) => `<tr><td>${item.label}</td><td class="num">${fmtMoney(q[item.key])}</td></tr>`
  ).join("");

  const consumablesRows = q.consumables.length
    ? q.consumables
        .map((c) => `<tr><td>${escapeHTML(consumableNames[c.consumable_id] || `Consumable #${c.consumable_id}`)}</td><td class="num">${fmtNum(c.qty)}</td><td class="num">${fmtMoney(c.unit_cost)}</td><td class="num">${fmtMoney(c.qty * c.unit_cost)}</td></tr>`)
        .join("")
    : `<tr><td colspan="4" class="empty-state">None</td></tr>`;

  const stepsRows = q.processing_steps.length
    ? q.processing_steps.map((s) => `<tr><td>${escapeHTML(s.label)}</td><td class="num">${fmtNum(s.minutes, 0)} min</td></tr>`).join("")
    : `<tr><td colspan="2" class="empty-state">None</td></tr>`;

  container.innerHTML = `
    <div class="page-header no-print">
      <div>
        <h1>${escapeHTML(q.name)}</h1>
        <p class="page-subtitle">${statusBadge(q.status)} &nbsp; Created ${fmtDate(q.created_at)} &middot; Updated ${fmtDate(q.updated_at)}</p>
      </div>
      <div style="display:flex; gap:8px;">
        <a class="btn" href="#/quotes/${q.id}/edit">Edit</a>
        <button class="primary" id="print-btn">Print / Save as PDF</button>
      </div>
    </div>

    <div class="printable-quote">
      <div style="margin-bottom:16px;">
        <h1 class="print-only" style="display:none;">${escapeHTML(q.name)} ${statusBadge(q.status)}</h1>
        <p><strong>Client:</strong> ${escapeHTML(q.client_name || "—")} ${q.client_email ? `(${escapeHTML(q.client_email)})` : ""}</p>
        <p><strong>Valid until:</strong> ${fmtDate(q.valid_until)}</p>
        ${q.notes ? `<p><strong>Notes:</strong> ${escapeHTML(q.notes)}</p>` : ""}
      </div>

      <div class="detail-grid">
        <div>
          <div class="section">
            <h3 style="margin-top:0;">Print parameters</h3>
            <table>
              <tbody>
                <tr><td>Main filament</td><td class="num">${escapeHTML(q.filament_name)} — ${fmtNum(q.print_weight_g)} g, ${minutesToHHMM(q.print_time_min)}</td></tr>
                ${q.supports_enabled ? `<tr><td>Supports</td><td class="num">${escapeHTML(q.support_filament_name || "—")} — ${fmtNum(q.support_weight_g)} g</td></tr>` : ""}
                ${q.prototypes_enabled ? `<tr><td>Prototypes</td><td class="num">${fmtNum(q.prototype_count)} × ${escapeHTML(q.prototype_filament_name || "—")} — ${fmtNum(q.prototype_weight_g)} g, ${minutesToHHMM(q.prototype_time_min)}</td></tr>` : ""}
                ${q.drying_enabled ? `<tr><td>Drying</td><td class="num">${fmtNum(q.drying_hours)} h</td></tr>` : ""}
              </tbody>
            </table>
          </div>

          <div class="section">
            <h3 style="margin-top:0;">Equipment used</h3>
            <table>
              <tbody>
                <tr><td>Printer</td><td class="num">${escapeHTML(q.machine_name || "—")}</td></tr>
                <tr><td>Build plate</td><td class="num">${escapeHTML(q.build_plate_name || "—")}</td></tr>
                <tr><td>Nozzle</td><td class="num">${escapeHTML(q.nozzle_name || "—")}</td></tr>
                <tr><td>AMS unit</td><td class="num">${escapeHTML(q.ams_name || "—")}</td></tr>
                <tr><td>Dry cabinet</td><td class="num">${escapeHTML(q.dry_cabinet_name || "—")}</td></tr>
              </tbody>
            </table>
          </div>

          <div class="section">
            <h3 style="margin-top:0;">Pre/post-processing</h3>
            <table><tbody>${stepsRows}</tbody></table>
          </div>

          <div class="section">
            <h3 style="margin-top:0;">Consumables</h3>
            <table>
              <thead><tr><th>Consumable</th><th class="num">Qty</th><th class="num">Unit cost</th><th class="num">Line total</th></tr></thead>
              <tbody>${consumablesRows}</tbody>
            </table>
          </div>
        </div>

        <div>
          <div class="section">
            <h3 style="margin-top:0;">Cost breakdown</h3>
            <div class="cost-bar">${barSegments}</div>
            <div class="cost-legend">${legend}</div>
            <table>
              <tbody>
                ${breakdownRows}
                <tr style="font-weight:700;"><td>Total / Suggested price</td><td class="num">${fmtMoney(q.suggested_price)}</td></tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  `;

  container.querySelector("#print-btn").addEventListener("click", () => window.print());
}
