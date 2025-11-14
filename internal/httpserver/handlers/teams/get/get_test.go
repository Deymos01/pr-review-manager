package get_test

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers/teams/get"
	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers/teams/get/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestGetTeamHandler(t *testing.T) {
	type testCase struct {
		name           string
		teamName       string
		mockReturnTeam *domains.Team
		mockError      error
		expectedStatus int
		expectedErr    string
	}

	cases := []testCase{
		{
			name:     "Success",
			teamName: "team",
			mockReturnTeam: &domains.Team{
				Name: "team",
				Members: []*domains.User{
					{ID: "u1", Name: "Alice", IsActive: true},
					{ID: "u2", Name: "Bob", IsActive: false},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Not found",
			teamName:       "missing",
			mockError:      errors.New("not found"),
			expectedStatus: http.StatusNotFound,
			expectedErr:    "resource not found",
		},
		{
			name:           "Empty team name",
			teamName:       "",
			mockError:      errors.New("team name empty"),
			expectedStatus: http.StatusNotFound,
			expectedErr:    "resource not found",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := mocks.NewTeamService(t)

			svc.On(
				"GetTeam",
				mock.Anything,
				tc.teamName,
			).Return(tc.mockReturnTeam, tc.mockError).Once()

			handler := get.New(discardLogger(), svc)

			req := httptest.NewRequest(http.MethodGet, "/team/get?team_name="+tc.teamName, nil)
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
				require.Equal(t, tc.teamName, resp["team_name"])

				members := resp["members"].([]any)
				require.Len(t, members, 2)
			}
		})
	}
}
