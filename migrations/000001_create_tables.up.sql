CREATE TABLE IF NOT EXISTS teams
(
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS users
(
    id        TEXT PRIMARY KEY,
    name      TEXT NOT NULL,
    team_name TEXT REFERENCES teams (name) ON DELETE SET NULL,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS statuses
(
    id   SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS pull_requests
(
    id                  TEXT PRIMARY KEY,
    name                TEXT NOT NULL,
    author_id           TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    status_id           INT  NOT NULL REFERENCES statuses (id),
    need_more_reviewers BOOLEAN   DEFAULT FALSE,
    created_at          TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reviewers
(
    user_id         TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    pull_request_id TEXT NOT NULL REFERENCES pull_requests (id) ON DELETE CASCADE,
    assigned_at     TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (user_id, pull_request_id)
);