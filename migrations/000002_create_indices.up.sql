CREATE INDEX IF NOT EXISTS idx_teams_name ON teams (name);
CREATE INDEX IF NOT EXISTS idx_users_team_name ON users (team_name);
CREATE INDEX IF NOT EXISTS idx_pull_requests_author_id ON pull_requests (author_id);
CREATE INDEX IF NOT EXISTS idx_reviewers_user_id ON reviewers (user_id);
CREATE INDEX IF NOT EXISTS idx_reviewers_pull_request_id ON reviewers (pull_request_id);