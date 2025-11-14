package tests

import (
	"bytes"
	"database/sql"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func truncateAllTables(db *sql.DB) {
	queries := []string{
		`TRUNCATE TABLE reviewers RESTART IDENTITY CASCADE;`,
		`TRUNCATE TABLE pull_requests RESTART IDENTITY CASCADE;`,
		`TRUNCATE TABLE users RESTART IDENTITY CASCADE;`,
		`TRUNCATE TABLE teams RESTART IDENTITY CASCADE;`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			panic(err)
		}
	}
}

func createPR(t *testing.T, httpClient *http.Client, baseURL string, data []byte) *http.Response {
	req, _ := http.NewRequest("POST", baseURL+"/pullRequest/create", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", "admin")

	resp, err := httpClient.Do(req)
	require.NoError(t, err)

	return resp
}
