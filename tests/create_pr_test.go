package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type PRRequest struct {
	PullRequestID     string   `json:"pull_request_id"`
	PullRequestName   string   `json:"pull_request_name"`
	AuthorID          string   `json:"author_id"`
	Status            string   `json:"status"`
	AssignedReviewers []string `json:"assigned_reviewers"`
}

type CreatePRResponse struct {
	PR PRRequest `json:"pr"`
}

type Member struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type TeamAddResponse struct {
	Team struct {
		Name    string   `json:"team_name"`
		Members []Member `json:"members"`
	} `json:"team"`
}

type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func ensureTeam(t *testing.T) {
	body := map[string]any{
		"team_name": "backend",
		"members": []map[string]any{
			{
				"user_id":   "u1",
				"username":  "Alice",
				"is_active": true,
			},
			{
				"user_id":   "u2",
				"username":  "Bob",
				"is_active": true,
			},
			{
				"user_id":   "u3",
				"username":  "Charlie",
				"is_active": true,
			},
		},
	}

	data, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", baseURL+"/team/add", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", "admin")

	resp, err := httpClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Parse and check response
	var out TeamAddResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))

	assert.Equal(t, "backend", out.Team.Name)
	assert.Equal(t, 3, len(out.Team.Members))
}

func TestCreatePR_Success(t *testing.T) {
	truncateAllTables(db)
	ensureTeam(t)

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
	ensureTeam(t)

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
	ensureTeam(t)

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

	// Parse and check error response
	var out ErrorResponse
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&out))

	assert.Equal(t, handlers.PrExists, out.Error.Code)
	assert.Equal(t, "pull request already exists", out.Error.Message)
}
