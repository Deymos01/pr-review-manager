package add_test

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
	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers/teams/add"
	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers/teams/add/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestAddTeamHandler(t *testing.T) {
	type testCase struct {
		name           string
		body           string
		mockReturnTeam *domains.Team
		mockError      error
		expectedStatus int
		expectedErr    string
	}

	cases := []testCase{
		{
			name: "Success",
			body: `{
				"team_name":"team",
				"members":[
					{"user_id":"u1","username":"Alice","is_active":true},
					{"user_id":"u2","username":"Bob","is_active":false}
				]
			}`,
			mockReturnTeam: &domains.Team{
				Name: "team",
				Members: []*domains.User{
					{ID: "u1", Name: "Alice", TeamName: ptr("team"), IsActive: true},
					{ID: "u2", Name: "Bob", TeamName: ptr("team"), IsActive: false},
				},
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Invalid JSON",
			body:           `{"team_name":`,
			expectedStatus: http.StatusBadRequest,
			expectedErr:    "invalid JSON format",
		},
		{
			name:           "Team already exists",
			body:           `{"team_name":"team","members":[]}`,
			mockError:      errors.New("team exists"),
			expectedStatus: http.StatusBadRequest,
			expectedErr:    "team_name already exists",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := mocks.NewTeamService(t)

			if tc.expectedStatus != http.StatusBadRequest || tc.expectedErr == "team_name already exists" {
				svc.On(
					"AddTeam",
					mock.Anything,
					mock.AnythingOfType("*domains.Team")).
					Return(tc.mockReturnTeam, tc.mockError).
					Once()
			}

			handler := add.New(discardLogger(), svc)

			req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader([]byte(tc.body)))
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
				team := resp["team"].(map[string]any)
				require.Equal(t, "team", team["team_name"])

				members := team["members"].([]any)
				require.Len(t, members, 2)
			}
		})
	}
}

func ptr(s string) *string { return &s }
