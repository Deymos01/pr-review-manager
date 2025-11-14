package create_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers/pull_requests/create"
	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers/pull_requests/create/mocks"
	"github.com/Deymos01/pr-review-manager/internal/usecase"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCreateHandler(t *testing.T) {
	type testCase struct {
		name           string
		body           string
		mockReturnIDs  []string
		mockError      error
		expectedStatus int
		expectedErr    string
	}

	cases := []testCase{
		{
			name:           "Success",
			body:           `{"pull_request_id":"1","pull_request_name":"test","author_id":"1"}`,
			mockReturnIDs:  []string{"u1", "u2"},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Invalid JSON",
			body:           `{"pull_request_id":`,
			expectedStatus: http.StatusBadRequest,
			expectedErr:    "invalid JSON format",
		},
		{
			name:           "ErrUserNotFound",
			body:           `{"pull_request_id":"1","pull_request_name":"test","author_id":"1"}`,
			mockError:      usecase.ErrUserNotFound,
			expectedStatus: http.StatusNotFound,
			expectedErr:    "resource not found",
		},
		{
			name:           "ErrTeamNotFound",
			body:           `{"pull_request_id":"1","pull_request_name":"test","author_id":"1"}`,
			mockError:      usecase.ErrTeamNotFound,
			expectedStatus: http.StatusNotFound,
			expectedErr:    "resource not found",
		},
		{
			name:           "ErrPRAlreadyExists",
			body:           `{"pull_request_id":"1","pull_request_name":"test","author_id":"1"}`,
			mockError:      usecase.ErrPRAlreadyExists,
			expectedStatus: http.StatusConflict,
			expectedErr:    "pull request already exists",
		},
		{
			name:           "Unknown error",
			body:           `{"pull_request_id":"1","pull_request_name":"test","author_id":"1"}`,
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
					"CreatePullRequest",
					mock.Anything, "1", "test", "1").
					Return(tc.mockReturnIDs, tc.mockError).
					Once()
			}

			handler := create.New(discardLogger(), svc)

			req := httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewReader([]byte(tc.body)))
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

			if rr.Code == http.StatusCreated {
				pr := resp["pr"].(map[string]any)
				require.Equal(t, "1", pr["pull_request_id"])
				require.Equal(t, "test", pr["pull_request_name"])
				require.Equal(t, "1", pr["author_id"])
				require.Equal(t, "OPEN", pr["status"])
			}
		})
	}
}
