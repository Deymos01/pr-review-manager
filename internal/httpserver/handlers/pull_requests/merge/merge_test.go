package merge_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers/pull_requests/merge"
	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers/pull_requests/merge/mocks"
	"github.com/Deymos01/pr-review-manager/internal/usecase"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestMergeHandler(t *testing.T) {
	type testCase struct {
		name           string
		body           string
		mockReturnPR   *domains.PullRequest
		mockError      error
		expectedStatus int
		expectedErr    string
	}

	now := time.Now()

	cases := []testCase{
		{
			name: "Success",
			body: `{"pull_request_id":"1"}`,
			mockReturnPR: &domains.PullRequest{
				ID:   "1",
				Name: "test",
				Author: &domains.User{
					ID: "1",
				},
				Status: "MERGED",
				Reviewers: []*domains.Reviewer{
					{User: &domains.User{ID: "u1"}},
					{User: &domains.User{ID: "u2"}},
				},
				MergedAt: &now,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid JSON",
			body:           `{"pull_request_id":`,
			expectedStatus: http.StatusBadRequest,
			expectedErr:    "invalid JSON format",
		},
		{
			name:           "Pull request not found",
			body:           `{"pull_request_id":"1"}`,
			mockError:      usecase.ErrPullRequestNotFound,
			expectedStatus: http.StatusNotFound,
			expectedErr:    "resource not found",
		},
		{
			name:           "Unknown error",
			body:           `{"pull_request_id":"1"}`,
			mockError:      errors.New("unexpected"),
			expectedStatus: http.StatusInternalServerError,
			expectedErr:    "internal server error",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := mocks.NewPRService(t)

			if tc.expectedStatus != http.StatusBadRequest {
				svc.On(
					"MergePullRequest",
					mock.Anything,
					"1",
				).Return(tc.mockReturnPR, tc.mockError).Once()
			}

			handler := merge.New(discardLogger(), svc)

			req := httptest.NewRequest(http.MethodPost, "/pullRequest/merge", bytes.NewBufferString(tc.body))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.expectedStatus, rr.Code)

			var resp map[string]any
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

			if tc.expectedErr != "" {
				errResp := resp["error"].(map[string]any)
				require.Equal(t, tc.expectedErr, errResp["message"])
				return
			}

			if rr.Code == http.StatusOK {
				pr := resp["pr"].(map[string]any)

				require.Equal(t, tc.mockReturnPR.ID, pr["pull_request_id"])
				require.Equal(t, tc.mockReturnPR.Name, pr["pull_request_name"])
				require.Equal(t, tc.mockReturnPR.Author.ID, pr["author_id"])
				require.Equal(t, tc.mockReturnPR.Status, pr["status"])

				arr := pr["assigned_reviewers"].([]any)
				require.Len(t, arr, len(tc.mockReturnPR.Reviewers))
				require.Equal(t, "u1", arr[0])
				require.Equal(t, "u2", arr[1])

				require.NotEmpty(t, pr["mergedAt"])
			}
		})
	}
}
