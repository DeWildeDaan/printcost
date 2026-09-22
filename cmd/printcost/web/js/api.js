const BASE = "/api/v1";

async function request(method, path, body) {
  const opts = { method, headers: {} };
  if (body !== undefined) {
    opts.headers["Content-Type"] = "application/json";
    opts.body = JSON.stringify(body);
  }
  const res = await fetch(BASE + path, opts);
  if (res.status === 204) return null;
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error || `Request failed (${res.status})`);
  }
  return data;
}

export const api = {
  get: (path) => request("GET", path),
  post: (path, body) => request("POST", path, body),
  put: (path, body) => request("PUT", path, body),
  patch: (path, body) => request("PATCH", path, body),
  del: (path) => request("DELETE", path),
};

// Simple resource client factory for the seven CRUD-only entities.
export function crudClient(path) {
  return {
    list: () => api.get(path),
    get: (id) => api.get(`${path}/${id}`),
    create: (data) => api.post(path, data),
    update: (id, data) => api.put(`${path}/${id}`, data),
    remove: (id) => api.del(`${path}/${id}`),
  };
}

export const machinesApi = crudClient("/machines");
export const buildPlatesApi = crudClient("/build-plates");
export const nozzlesApi = crudClient("/nozzles");
export const amsUnitsApi = crudClient("/ams-units");
export const otherEquipmentApi = crudClient("/other-equipment");
export const filamentsApi = crudClient("/filaments");
export const consumablesApi = crudClient("/consumables");

export const settingsApi = {
  get: () => api.get("/settings"),
  update: (data) => api.put("/settings", data),
};

export const quotesApi = {
  list: (params) => api.get(`/quotes${params ? "?" + new URLSearchParams(params) : ""}`),
  get: (id) => api.get(`/quotes/${id}`),
  create: (data) => api.post("/quotes", data),
  update: (id, data) => api.put(`/quotes/${id}`, data),
  remove: (id) => api.del(`/quotes/${id}`),
  duplicate: (id) => api.post(`/quotes/${id}/duplicate`),
  setStatus: (id, status) => api.patch(`/quotes/${id}/status`, { status }),
};

export const reportsApi = {
  summary: () => api.get("/reports/summary"),
  revenueOverTime: () => api.get("/reports/revenue-over-time"),
  costBreakdownTrends: () => api.get("/reports/cost-breakdown-trends"),
  machineUtilisation: () => api.get("/reports/machine-utilisation"),
  filamentConsumption: () => api.get("/reports/filament-consumption"),
};
