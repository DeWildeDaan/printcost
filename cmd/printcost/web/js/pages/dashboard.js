import { reportsApi } from "../api.js";
import { fmtMoney, fmtNum, fmtPct } from "../utils.js";

let chartJsPromise = null;
function loadChartJs() {
  if (window.Chart) return Promise.resolve();
  if (chartJsPromise) return chartJsPromise;
  chartJsPromise = new Promise((resolve, reject) => {
    const script = document.createElement("script");
    script.src = "https://cdnjs.cloudflare.com/ajax/libs/Chart.js/4.5.1/chart.umd.min.js";
    script.onload = resolve;
    script.onerror = reject;
    document.head.appendChild(script);
  });
  return chartJsPromise;
}

const COST_COLORS = {
  filament_pct: "#4d9de0",
  electricity_pct: "#e0c25d",
  depreciation_pct: "#9d6de0",
  labor_pct: "#5dbf7d",
  consumables_pct: "#e08d5d",
  failure_pct: "#e05d5d",
  markup_pct: "#5de0c2",
};
const COST_LABELS = {
  filament_pct: "Filament",
  electricity_pct: "Electricity",
  depreciation_pct: "Depreciation",
  labor_pct: "Labour",
  consumables_pct: "Consumables",
  failure_pct: "Failure",
  markup_pct: "Markup",
};

export async function renderDashboard(container) {
  container.innerHTML = `<h1>Dashboard</h1><p class="page-subtitle">Loading…</p>`;

  const [summary, revenue, breakdown, machineUtil, filamentUse] = await Promise.all([
    reportsApi.summary(),
    reportsApi.revenueOverTime(),
    reportsApi.costBreakdownTrends(),
    reportsApi.machineUtilisation(),
    reportsApi.filamentConsumption(),
  ]);

  container.innerHTML = `
    <h1>Dashboard</h1>
    <div class="card-row">
      <div class="card stat-card"><div class="stat-label">Total quotes</div><div class="stat-value">${summary.total_quotes}</div></div>
      <div class="card stat-card"><div class="stat-label">Total revenue (paid)</div><div class="stat-value">${fmtMoney(summary.total_revenue)}</div></div>
      <div class="card stat-card"><div class="stat-label">Average margin</div><div class="stat-value">${fmtPct(summary.average_margin_pct)}</div></div>
      <div class="card stat-card"><div class="stat-label">Most used filament</div><div class="stat-value" style="font-size:15px;">${summary.most_used_filament || "—"}</div></div>
    </div>

    <div class="card chart-card" style="margin-bottom:20px;">
      <h2 style="margin-top:0;">Revenue & quote volume (last 12 months)</h2>
      <canvas id="revenue-chart"></canvas>
    </div>

    <div class="card chart-card" style="margin-bottom:20px;">
      <h2 style="margin-top:0;">Cost breakdown trends</h2>
      <canvas id="breakdown-chart"></canvas>
    </div>

    <div class="detail-grid">
      <div class="card">
        <h2 style="margin-top:0;">Machine utilisation</h2>
        <table>
          <thead><tr><th>Machine</th><th class="num">Total hours</th><th class="num">Depreciation cost</th></tr></thead>
          <tbody>
            ${machineUtil.data.length ? machineUtil.data.map((m) => `
              <tr><td>${m.machine}</td><td class="num">${fmtNum(m.total_hours, 1)}</td><td class="num">${fmtMoney(m.total_depreciation)}</td></tr>
            `).join("") : `<tr><td colspan="3" class="empty-state">No data yet.</td></tr>`}
          </tbody>
        </table>
      </div>
      <div class="card">
        <h2 style="margin-top:0;">Filament consumption</h2>
        <table>
          <thead><tr><th>Filament</th><th class="num">Grams</th><th class="num">Cost</th><th class="num">Quotes</th></tr></thead>
          <tbody>
            ${filamentUse.data.length ? filamentUse.data.map((f) => `
              <tr><td>${f.name}</td><td class="num">${fmtNum(f.grams, 0)}</td><td class="num">${fmtMoney(f.cost)}</td><td class="num">${f.quote_count}</td></tr>
            `).join("") : `<tr><td colspan="4" class="empty-state">No data yet.</td></tr>`}
          </tbody>
        </table>
      </div>
    </div>
  `;

  await loadChartJs();
  const Chart = window.Chart;
  if (!Chart) return;

  const months = Array.from(new Set([...revenue.revenue_by_month.map((p) => p.month), ...revenue.quotes_by_month.map((p) => p.month)])).sort();
  const revenueByMonth = Object.fromEntries(revenue.revenue_by_month.map((p) => [p.month, p.value]));
  const countByMonth = Object.fromEntries(revenue.quotes_by_month.map((p) => [p.month, p.value]));

  new Chart(container.querySelector("#revenue-chart"), {
    data: {
      labels: months,
      datasets: [
        { type: "line", label: "Revenue (paid)", data: months.map((m) => revenueByMonth[m] || 0), borderColor: "#4d9de0", backgroundColor: "#4d9de0", yAxisID: "y", tension: 0.25 },
        { type: "bar", label: "Quotes created", data: months.map((m) => countByMonth[m] || 0), backgroundColor: "#2c5a7c", yAxisID: "y1" },
      ],
    },
    options: {
      scales: {
        y: { position: "left", ticks: { color: "#9aa0ab" }, grid: { color: "#2e323c" } },
        y1: { position: "right", ticks: { color: "#9aa0ab" }, grid: { display: false } },
        x: { ticks: { color: "#9aa0ab" }, grid: { color: "#2e323c" } },
      },
      plugins: { legend: { labels: { color: "#e4e6eb" } } },
    },
  });

  const breakdownKeys = Object.keys(COST_LABELS);
  new Chart(container.querySelector("#breakdown-chart"), {
    type: "bar",
    data: {
      labels: breakdown.map((b) => b.month),
      datasets: breakdownKeys.map((key) => ({
        label: COST_LABELS[key],
        data: breakdown.map((b) => b[key] || 0),
        backgroundColor: COST_COLORS[key],
      })),
    },
    options: {
      scales: {
        x: { stacked: true, ticks: { color: "#9aa0ab" }, grid: { color: "#2e323c" } },
        y: { stacked: true, ticks: { color: "#9aa0ab" }, grid: { color: "#2e323c" }, title: { display: true, text: "% of total cost", color: "#9aa0ab" } },
      },
      plugins: { legend: { labels: { color: "#e4e6eb" } } },
    },
  });
}
