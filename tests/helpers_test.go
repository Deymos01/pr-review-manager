package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
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

func ensureTeam(t *testing.T, teamName string, membersNames []string) {
	members := make([]map[string]any, 0, len(membersNames))
	for i, name := range membersNames {
		member := map[string]any{
			"user_id":   "u" + strconv.Itoa(i+1),
			"username":  name,
			"is_active": true,
		}
		members = append(members, member)
	}

	body := map[string]any{
		"team_name": teamName,
		"members":   members,
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

	assert.Equal(t, teamName, out.Team.Name)
	assert.Equal(t, len(membersNames), len(out.Team.Members))
}

func createPR(t *testing.T, httpClient *http.Client, baseURL string, data []byte) *http.Response {
	req, _ := http.NewRequest("POST", baseURL+"/pullRequest/create", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", "admin")

	resp, err := httpClient.Do(req)
	require.NoError(t, err)

	return resp
}

func checkAssignment(
	t *testing.T,
	httpClient *http.Client,
	userID string,
	prID string,
	assigned bool,
) {
	req, err := http.NewRequest("GET", baseURL+"/users/getReview?user_id="+userID, nil)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", "admin")

	resp, err := httpClient.Do(req)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var reviewOut GetReviewResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&reviewOut))
	err = resp.Body.Close()
	require.NoError(t, err)

	require.Equal(t, userID, reviewOut.UserId)
	if assigned {
		require.Equal(t, 1, len(reviewOut.PullRequests))
		require.Equal(t, prID, reviewOut.PullRequests[0].PullRequestId)
	} else {
		require.Equal(t, 0, len(reviewOut.PullRequests))
	}
}

func setActiveStatus(t *testing.T, httpClient *http.Client, userID string, isActive bool) {
	body := map[string]any{
		"user_id":   userID,
		"is_active": isActive,
	}

	data, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", baseURL+"/users/setIsActive", bytes.NewReader(data))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", "admin")

	resp, err := httpClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func reassignUser(t *testing.T, httpClient *http.Client, oldUserID string, prID string) *http.Response {
	reassignBody := map[string]any{
		"pull_request_id": prID,
		"old_reviewer_id": oldUserID,
	}

	reassignData, err := json.Marshal(reassignBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", baseURL+"/pullRequest/reassign", bytes.NewReader(reassignData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", "admin")

	resp, err := httpClient.Do(req)
	require.NoError(t, err)

	return resp
}
