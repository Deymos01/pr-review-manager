CREATE TABLE IF NOT EXISTS teams
(
    id   UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS users
(
    id        UUID PRIMARY KEY,
    name      TEXT NOT NULL,
    team_id   UUID REFERENCES teams (id) ON DELETE SET NULL,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS statuses
(
    id   SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS pull_requests
(
    id                  UUID PRIMARY KEY,
    name                TEXT NOT NULL,
    author_id           UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    status_id           INT  NOT NULL REFERENCES statuses (id),
    need_more_reviewers BOOLEAN   DEFAULT FALSE,
    created_at          TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reviewers
(
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    pull_request_id UUID NOT NULL REFERENCES pull_requests (id) ON DELETE CASCADE,
    assigned_at     TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (user_id, pull_request_id)
);