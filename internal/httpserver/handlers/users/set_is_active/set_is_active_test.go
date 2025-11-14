package set_is_active_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers/users/set_is_active"
	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers/users/set_is_active/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestSetIsActiveHandler(t *testing.T) {
	type testCase struct {
		name           string
		body           any
		mockUser       *domains.User
		mockError      error
		expectedStatus int
		expectedErr    string
	}

	cases := []testCase{
		{
			name: "Success",
			body: set_is_active.Request{
				UserID:   "u1",
				IsActive: true,
			},
			mockUser: &domains.User{
				ID:       "u1",
				Name:     "John",
				TeamName: ptr("team"),
				IsActive: true,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid JSON",
			body:           `{"user_id": 123}`,
			expectedStatus: http.StatusBadRequest,
			expectedErr:    "invalid JSON format",
		},
		{
			name: "User not found",
			body: set_is_active.Request{
				UserID:   "missing",
				IsActive: true,
			},
			mockError:      errors.New("not found"),
			expectedStatus: http.StatusNotFound,
			expectedErr:    "resource not found",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := mocks.NewUserService(t)

			var buf bytes.Buffer
			switch v := tc.body.(type) {
			case string:
				buf.WriteString(v)
			default:
				require.NoError(t, json.NewEncoder(&buf).Encode(v))
			}

			if req, ok := tc.body.(set_is_active.Request); ok {
				svc.On(
					"SetUserIsActive",
					mock.Anything,
					req.UserID,
					req.IsActive,
				).Return(tc.mockUser, tc.mockError).Once()
			}

			handler := set_is_active.New(discardLogger(), svc)

			req := httptest.NewRequest(
				http.MethodPost,
				"/users/setIsActive",
				&buf,
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

			user := resp["user"].(map[string]any)

			require.Equal(t, tc.mockUser.ID, user["user_id"])
			require.Equal(t, tc.mockUser.Name, user["username"])
			require.Equal(t, *tc.mockUser.TeamName, user["team_name"])
			require.Equal(t, tc.mockUser.IsActive, user["is_active"])
		})
	}
}

func ptr(s string) *string {
	return &s
}
