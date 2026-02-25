CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK(role IN ('admin','viewer')) DEFAULT 'viewer',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS servers (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    host            TEXT NOT NULL,
    port            INTEGER NOT NULL DEFAULT 22,
    username        TEXT NOT NULL,
    ssh_private_key TEXT NOT NULL,
    pre_command     TEXT NOT NULL DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS playbooks (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    file_path   TEXT NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS forms (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    playbook_id TEXT NOT NULL REFERENCES playbooks(id) ON DELETE CASCADE,
    server_id   TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS form_fields (
    id            TEXT PRIMARY KEY,
    form_id       TEXT NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    label         TEXT NOT NULL,
    field_type    TEXT NOT NULL CHECK(field_type IN ('text','number','bool','select')),
    default_value TEXT NOT NULL DEFAULT '',
    options       TEXT NOT NULL DEFAULT '[]',
    required      INTEGER NOT NULL DEFAULT 0,
    sort_order    INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS runs (
    id          TEXT PRIMARY KEY,
    form_id     TEXT REFERENCES forms(id) ON DELETE SET NULL,
    playbook_id TEXT NOT NULL REFERENCES playbooks(id),
    server_id   TEXT NOT NULL REFERENCES servers(id),
    variables   TEXT NOT NULL DEFAULT '{}',
    status      TEXT NOT NULL CHECK(status IN ('pending','running','success','failed')) DEFAULT 'pending',
    output      TEXT NOT NULL DEFAULT '',
    started_at  DATETIME,
    finished_at DATETIME
);
