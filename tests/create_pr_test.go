package tests

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatePR_Success(t *testing.T) {
	truncateAllTables(db)
	ensureTeam(t, "backend", []string{"alice", "bob", "charlie"})

	body := map[string]any{
		"pull_request_id":   "pr-1001",
		"pull_request_name": "Add search function",
		"author_id":         "u1",
	}

	data, err := json.Marshal(body)
	require.NoError(t, err)

	resp := createPR(t, httpClient, baseURL, data)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Parse and check response
	var out CreatePRResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))

	assert.Equal(t, "pr-1001", out.PR.PullRequestID)
	assert.Equal(t, "Add search function", out.PR.PullRequestName)
	assert.Equal(t, "u1", out.PR.AuthorID)
	assert.Equal(t, "OPEN", out.PR.Status)
	assert.Greater(t, len(out.PR.AssignedReviewers), 0, "expected at least one reviewer assigned")
}

func TestCreatePR_InvalidJSON(t *testing.T) {
	truncateAllTables(db)
	ensureTeam(t, "backend", []string{"alice", "bob", "charlie"})

	data := []byte(`{ "pull_request_id": 123, }`)

	resp := createPR(t, httpClient, baseURL, data)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Parse and check error response
	var out ErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))

	assert.Equal(t, handlers.InvalidRequest, out.Error.Code)
	assert.Equal(t, "invalid JSON format", out.Error.Message)
}

func TestCreatePR_AlreadyExists(t *testing.T) {
	truncateAllTables(db)
	ensureTeam(t, "backend", []string{"alice", "bob", "charlie"})

	body := map[string]any{
		"pull_request_id":   "pr-1003",
		"pull_request_name": "Initial",
		"author_id":         "u1",
	}
	data, _ := json.Marshal(body)

	resp1 := createPR(t, httpClient, baseURL, data)
	defer func() { _ = resp1.Body.Close() }()

	assert.Equal(t, http.StatusCreated, resp1.StatusCode)

	resp2 := createPR(t, httpClient, baseURL, data)
	defer func() { _ = resp2.Body.Close() }()

	assert.Equal(t, http.StatusConflict, resp2.StatusCode)

	var out ErrorResponse
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&out))

	assert.Equal(t, handlers.PrExists, out.Error.Code)
	assert.Equal(t, "pull request already exists", out.Error.Message)
}
