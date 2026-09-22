import { quotesApi } from "../api.js";
import { fmtMoney, fmtDate, statusBadge, escapeHTML } from "../utils.js";

const STATUSES = ["draft", "sent", "paid", "cancelled"];

export async function renderQuoteList(container) {
  let status = "all";
  let search = "";
  let sort = "created";
  let quotes = [];

  async function load() {
    const res = await quotesApi.list({ status, search, sort });
    quotes = res.data;
    drawTable();
  }

  function rowHTML(q) {
    return `
      <tr data-id="${q.id}">
        <td><a href="#/quotes/${q.id}">${escapeHTML(q.name)}</a></td>
        <td>${escapeHTML(q.client_name || "—")}</td>
        <td>
          <select class="status-select" data-id="${q.id}">
            ${STATUSES.map((s) => `<option value="${s}" ${s === q.status ? "selected" : ""}>${s}</option>`).join("")}
          </select>
        </td>
        <td class="num">${fmtMoney(q.suggested_price)}</td>
        <td>${fmtDate(q.created_at)}</td>
        <td>${fmtDate(q.updated_at)}</td>
        <td class="table-actions">
          <a class="btn small" href="#/quotes/${q.id}">View</a>
          <a class="btn small" href="#/quotes/${q.id}/edit">Edit</a>
          <button class="small duplicate-btn" data-id="${q.id}">Duplicate</button>
          <button class="small danger delete-btn" data-id="${q.id}">Delete</button>
        </td>
      </tr>`;
  }

  function drawTable() {
    const tbody = container.querySelector("#quotes-tbody");
    tbody.innerHTML = quotes.length
      ? quotes.map(rowHTML).join("")
      : `<tr><td colspan="7" class="empty-state">No quotes found.</td></tr>`;

    tbody.querySelectorAll("tr[data-id]").forEach((row) => {
      row.style.cursor = "pointer";
      row.addEventListener("click", (e) => {
        if (e.target.closest("a, button, select")) return;
        location.hash = `#/quotes/${row.dataset.id}`;
      });
    });
    tbody.querySelectorAll(".status-select").forEach((sel) => {
      sel.addEventListener("change", async () => {
        try {
          await quotesApi.setStatus(Number(sel.dataset.id), sel.value);
          await load();
        } catch (err) {
          alert(err.message);
        }
      });
    });
    tbody.querySelectorAll(".duplicate-btn").forEach((btn) => {
      btn.addEventListener("click", async () => {
        try {
          const copy = await quotesApi.duplicate(Number(btn.dataset.id));
          location.hash = `#/quotes/${copy.id}/edit`;
        } catch (err) {
          alert(err.message);
        }
      });
    });
    tbody.querySelectorAll(".delete-btn").forEach((btn) => {
      btn.addEventListener("click", async () => {
        if (!confirm("Delete this quote? This cannot be undone.")) return;
        try {
          await quotesApi.remove(Number(btn.dataset.id));
          await load();
        } catch (err) {
          alert(err.message);
        }
      });
    });
  }

  container.innerHTML = `
    <div class="page-header">
      <h1>Quotes</h1>
      <a class="btn primary" href="#/quotes/new">New Quote</a>
    </div>
    <div class="filters-bar">
      <select id="status-filter">
        <option value="all">All statuses</option>
        ${STATUSES.map((s) => `<option value="${s}">${s}</option>`).join("")}
      </select>
      <input id="search-input" type="text" placeholder="Search by name or client...">
      <select id="sort-select">
        <option value="created">Newest first</option>
        <option value="name">Name</option>
        <option value="price">Price</option>
      </select>
    </div>
    <div class="card">
      <table>
        <thead><tr>
          <th>Name</th><th>Client</th><th>Status</th><th class="num">Suggested price</th><th>Created</th><th>Updated</th><th></th>
        </tr></thead>
        <tbody id="quotes-tbody"></tbody>
      </table>
    </div>
  `;

  container.querySelector("#status-filter").addEventListener("change", (e) => { status = e.target.value; load(); });
  let searchTimer;
  container.querySelector("#search-input").addEventListener("input", (e) => {
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => { search = e.target.value; load(); }, 250);
  });
  container.querySelector("#sort-select").addEventListener("change", (e) => { sort = e.target.value; load(); });

  await load();
}
