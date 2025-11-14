package get_review_test

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers/users/get_review"
	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers/users/get_review/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestGetReviewHandler(t *testing.T) {
	type testCase struct {
		name           string
		userID         string
		mockReturnPRs  []*domains.PullRequest
		mockError      error
		expectedStatus int
		expectedErr    string
	}

	cases := []testCase{
		{
			name:   "Success (one PR)",
			userID: "u1",
			mockReturnPRs: []*domains.PullRequest{
				{
					ID:     "pr1",
					Name:   "Implement feature",
					Status: "OPEN",
					Author: &domains.User{ID: "author1"},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "User not found",
			userID:         "missing",
			mockError:      errors.New("not found"),
			expectedStatus: http.StatusNotFound,
			expectedErr:    "resource not found",
		},
		{
			name:           "Empty user_id",
			userID:         "",
			mockError:      errors.New("empty"),
			expectedStatus: http.StatusNotFound,
			expectedErr:    "resource not found",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := mocks.NewUserService(t)

			svc.On(
				"GetUsersReview",
				mock.Anything,
				tc.userID,
			).Return(tc.mockReturnPRs, tc.mockError).Once()

			handler := get_review.New(discardLogger(), svc)

			req := httptest.NewRequest(
				http.MethodGet,
				"/users/review?user_id="+tc.userID,
				nil,
			)

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

			require.Equal(t, tc.userID, resp["user_id"])

			pr := resp["pull_requests"].(map[string]any)

			require.Equal(t, tc.mockReturnPRs[0].ID, pr["pull_request_id"])
			require.Equal(t, tc.mockReturnPRs[0].Name, pr["pull_request_name"])
			require.Equal(t, tc.mockReturnPRs[0].Author.ID, pr["author_id"])
			require.Equal(t, tc.mockReturnPRs[0].Status, pr["status"])
		})
	}
}
