CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS machines (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL,
    type_tag        TEXT NOT NULL DEFAULT '',
    purchase_price  REAL NOT NULL,
    lifespan_hours  REAL NOT NULL,
    service_cost    REAL NOT NULL DEFAULT 0,
    power_kw        REAL NOT NULL,
    notes           TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS build_plates (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    name             TEXT NOT NULL,
    purchase_price   REAL NOT NULL,
    lifespan_hours   REAL NOT NULL,
    notes            TEXT NOT NULL DEFAULT '',
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at       TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS nozzles (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL,
    material        TEXT NOT NULL DEFAULT '',
    diameter_mm     REAL NOT NULL,
    purchase_price  REAL NOT NULL,
    lifespan_hours  REAL NOT NULL,
    notes           TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS ams_units (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL,
    purchase_price  REAL NOT NULL,
    lifespan_hours  REAL NOT NULL,
    service_cost    REAL NOT NULL DEFAULT 0,
    power_kw        REAL NOT NULL DEFAULT 0,
    notes           TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS other_equipment (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL,
    purchase_price  REAL NOT NULL,
    lifespan_hours  REAL NOT NULL,
    service_cost    REAL NOT NULL DEFAULT 0,
    power_kw        REAL NOT NULL DEFAULT 0,
    notes           TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS filaments (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    name                 TEXT NOT NULL,
    brand                TEXT NOT NULL DEFAULT '',
    material             TEXT NOT NULL,
    diameter_mm          REAL NOT NULL DEFAULT 1.75,
    spool_price          REAL NOT NULL,
    spool_weight_kg      REAL NOT NULL DEFAULT 1.0,
    density_gcm3         REAL NOT NULL DEFAULT 1.24,
    drying_temp_c        INTEGER,
    drying_time_hours    REAL,
    requires_dry_cabinet INTEGER NOT NULL DEFAULT 0,
    notes                TEXT NOT NULL DEFAULT '',
    created_at           TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at           TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS consumables (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    name             TEXT NOT NULL,
    unit_label       TEXT NOT NULL,
    unit_cost        REAL NOT NULL,
    notes            TEXT NOT NULL DEFAULT '',
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at       TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS quotes (
    id                      INTEGER PRIMARY KEY AUTOINCREMENT,
    name                    TEXT NOT NULL,
    status                  TEXT NOT NULL DEFAULT 'draft',
    client_name             TEXT NOT NULL DEFAULT '',
    client_email            TEXT NOT NULL DEFAULT '',
    notes                   TEXT NOT NULL DEFAULT '',
    valid_until             TEXT,

    machine_id              INTEGER REFERENCES machines(id),
    build_plate_id          INTEGER REFERENCES build_plates(id),
    nozzle_id               INTEGER REFERENCES nozzles(id),
    ams_id                  INTEGER REFERENCES ams_units(id),
    dry_cabinet_id          INTEGER REFERENCES other_equipment(id),
    ams_used_for_printing   INTEGER NOT NULL DEFAULT 1,

    filament_id             INTEGER NOT NULL REFERENCES filaments(id),
    print_weight_g          REAL NOT NULL,
    print_time_min          REAL NOT NULL,

    supports_enabled        INTEGER NOT NULL DEFAULT 0,
    support_filament_id     INTEGER REFERENCES filaments(id),
    support_weight_g        REAL NOT NULL DEFAULT 0,

    prototypes_enabled       INTEGER NOT NULL DEFAULT 0,
    prototype_filament_id    INTEGER REFERENCES filaments(id),
    prototype_count          REAL NOT NULL DEFAULT 0,
    prototype_weight_g       REAL NOT NULL DEFAULT 0,
    prototype_time_min       REAL NOT NULL DEFAULT 0,

    drying_enabled          INTEGER NOT NULL DEFAULT 0,
    drying_hours            REAL NOT NULL DEFAULT 0,

    additional_cost         REAL NOT NULL DEFAULT 0,
    additional_cost_note    TEXT NOT NULL DEFAULT '',
    markup_pct              REAL NOT NULL,

    energy_cost_snapshot    REAL NOT NULL,
    labor_rate_snapshot     REAL NOT NULL,
    failure_rate_snapshot   REAL NOT NULL,

    cost_filament           REAL NOT NULL DEFAULT 0,
    cost_electricity        REAL NOT NULL DEFAULT 0,
    cost_depreciation       REAL NOT NULL DEFAULT 0,
    cost_labor              REAL NOT NULL DEFAULT 0,
    cost_consumables        REAL NOT NULL DEFAULT 0,
    cost_failure            REAL NOT NULL DEFAULT 0,
    cost_markup             REAL NOT NULL DEFAULT 0,
    cost_total              REAL NOT NULL DEFAULT 0,
    suggested_price         REAL NOT NULL DEFAULT 0,

    created_at              TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at              TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS quote_processing_steps (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    quote_id    INTEGER NOT NULL REFERENCES quotes(id) ON DELETE CASCADE,
    label       TEXT NOT NULL,
    minutes     REAL NOT NULL,
    sort_order  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS quote_consumables (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    quote_id       INTEGER NOT NULL REFERENCES quotes(id) ON DELETE CASCADE,
    consumable_id  INTEGER NOT NULL REFERENCES consumables(id),
    qty            REAL NOT NULL,
    unit_cost      REAL NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_quotes_status ON quotes(status);
CREATE INDEX IF NOT EXISTS idx_quotes_created_at ON quotes(created_at);
CREATE INDEX IF NOT EXISTS idx_qps_quote_id ON quote_processing_steps(quote_id);
CREATE INDEX IF NOT EXISTS idx_qc_quote_id ON quote_consumables(quote_id);
