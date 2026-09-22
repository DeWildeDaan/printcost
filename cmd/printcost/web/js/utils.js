let currencySymbol = "€";

export function setCurrencySymbol(symbol) {
  currencySymbol = symbol || "€";
}

export function fmtMoney(n) {
  const v = Number(n) || 0;
  return `${currencySymbol}${v.toFixed(2)}`;
}

export function fmtNum(n, decimals = 2) {
  const v = Number(n) || 0;
  return v.toFixed(decimals);
}

export function fmtPct(n) {
  return `${(Number(n) || 0).toFixed(1)}%`;
}

export function minutesToHHMM(minutes) {
  const m = Math.round(Number(minutes) || 0);
  const h = Math.floor(m / 60);
  const mm = m % 60;
  return `${String(h).padStart(2, "0")}:${String(mm).padStart(2, "0")}`;
}

export function hhmmToMinutes(hhmm) {
  const parts = String(hhmm || "0:0").split(":");
  const h = parseInt(parts[0], 10) || 0;
  const m = parseInt(parts[1], 10) || 0;
  return h * 60 + m;
}

export function escapeHTML(s) {
  return String(s ?? "").replace(/[&<>"']/g, (c) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
  }[c]));
}

export function todayPlusMonths(n) {
  const d = new Date();
  d.setMonth(d.getMonth() + n);
  return d.toISOString().slice(0, 10);
}

export function fmtDate(s) {
  if (!s) return "—";
  return String(s).slice(0, 10);
}

export function statusBadge(status) {
  return `<span class="badge badge-${status}">${escapeHTML(status)}</span>`;
}

export function el(html) {
  const template = document.createElement("template");
  template.innerHTML = html.trim();
  return template.content.firstElementChild;
}
