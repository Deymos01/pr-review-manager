package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReassignUser_Success(t *testing.T) {
	truncateAllTables(db)
	ensureTeam(t, "backend", []string{"user1", "user2", "user3", "user4", "user5", "user6", "user7"})

	// Create new PR with user1 as author
	body := map[string]any{
		"pull_request_id":   "pr-1",
		"pull_request_name": "some pull request",
		"author_id":         "u1",
	}

	data, err := json.Marshal(body)
	require.NoError(t, err)

	resp := createPR(t, httpClient, baseURL, data)

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Parse and check response (get assigned reviewers)
	var prOut CreatePRResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&prOut))
	err = resp.Body.Close()
	require.NoError(t, err)

	require.Equal(t, len(prOut.PR.AssignedReviewers), 2)

	assert.Equal(t, "pr-1", prOut.PR.PullRequestID)
	assert.Equal(t, "some pull request", prOut.PR.PullRequestName)
	assert.Equal(t, "u1", prOut.PR.AuthorID)
	assert.Equal(t, "OPEN", prOut.PR.Status)

	// prepare map of users and their assignment status
	usersAssignments := map[string]bool{
		prOut.PR.AssignedReviewers[0]: true,
		prOut.PR.AssignedReviewers[1]: true,
	}
	for i := 0; i < 7; i++ {
		userID := "u" + strconv.Itoa(i+1)
		if userID != prOut.PR.AssignedReviewers[0] && userID != prOut.PR.AssignedReviewers[1] && userID != "u1" {
			usersAssignments[userID] = false
		}
	}

	// for all users check assignment to PR
	for userID, assigned := range usersAssignments {
		checkAssignment(t, httpClient, userID, prOut.PR.PullRequestID, assigned)
	}

	var oldUserID string
	for userID, assigned := range usersAssignments {
		if assigned {
			oldUserID = userID
			break
		}
	}

	// Reassign user
	resp = reassignUser(t, httpClient, oldUserID, prOut.PR.PullRequestID)

	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse and check response (get new assigned reviewers)
	var reassignOut ReassignUserResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&reassignOut))
	err = resp.Body.Close()
	require.NoError(t, err)

	require.Equal(t, "pr-1", reassignOut.Pr.PullRequestId)
	require.Equal(t, "some pull request", reassignOut.Pr.PullRequestName)
	require.Equal(t, "u1", reassignOut.Pr.AuthorId)
	require.Equal(t, "OPEN", reassignOut.Pr.Status)
	require.Equal(t, 2, len(reassignOut.Pr.AssignedReviewers))

	assert.NotEqual(t, oldUserID, reassignOut.ReplacedBy)

	// Check that old user is not assigned anymore
	checkAssignment(t, httpClient, oldUserID, prOut.PR.PullRequestID, false)

	// Check that new user is assigned
	checkAssignment(t, httpClient, reassignOut.ReplacedBy, prOut.PR.PullRequestID, true)

	usersAssignments[oldUserID] = false
	usersAssignments[reassignOut.ReplacedBy] = true

	// set not assigned users to inactive
	for userID, assigned := range usersAssignments {
		if !assigned {
			setActiveStatus(t, httpClient, userID, false)
		}
	}

	// Reassign user again
	resp = reassignUser(t, httpClient, reassignOut.ReplacedBy, "pr-1")

	require.Equal(t, http.StatusConflict, resp.StatusCode)

	var errorOut ErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errorOut))
	err = resp.Body.Close()
	require.NoError(t, err)

	require.Equal(t, handlers.NoCandidate, errorOut.Error.Code)
	require.Equal(t, "no active replacement candidate in team", errorOut.Error.Message)

	// merge pull request
	body = map[string]any{
		"pull_request_id": "pr-1",
	}

	data, err = json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", baseURL+"/pullRequest/merge", bytes.NewReader(data))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", "admin")

	resp, err = httpClient.Do(req)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NoError(t, err)

	var mergeOut MergePRResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&mergeOut))
	err = resp.Body.Close()
	require.NoError(t, err)

	require.Equal(t, "MERGED", mergeOut.Pr.Status)

	// try to reassign user after merge
	resp = reassignUser(t, httpClient, oldUserID, "pr-1")

	require.Equal(t, http.StatusConflict, resp.StatusCode)

	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errorOut))
	err = resp.Body.Close()
	require.NoError(t, err)

	require.Equal(t, handlers.PrMerged, errorOut.Error.Code)
	require.Equal(t, "cannot reassign on merged PR", errorOut.Error.Message)
}
