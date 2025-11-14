package team

import (
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/repository"
	"github.com/Deymos01/pr-review-manager/internal/usecase"
	"github.com/Deymos01/pr-review-manager/internal/usecase/team/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestService_AddTeam(t *testing.T) {
	teamSample := &domains.Team{
		Name: "team",
		Members: []*domains.User{
			{Name: "user1"},
			{Name: "user2"},
		},
	}

	type testCase struct {
		name       string
		teamExists bool
		team       *domains.Team

		mockErrCreate error
		mockErrExist  error
		mockErrGet    error

		expectedErr error
	}

	cases := []testCase{
		{
			name:       "Success",
			teamExists: false,
			team:       teamSample,
		},
		{
			name:        "Team already exists",
			teamExists:  true,
			team:        teamSample,
			expectedErr: usecase.ErrTeamAlreadyExists,
		},
		{
			name:         "TeamExists returns error",
			team:         teamSample,
			mockErrExist: errors.New("team exists error"),
			expectedErr:  errors.New("team exists error"),
		},
		{
			name:          "CreateTeam returns error",
			teamExists:    false,
			team:          teamSample,
			mockErrCreate: errors.New("create team error"),
			expectedErr:   errors.New("create team error"),
		},
		{
			name:        "GetTeamByName returns error",
			teamExists:  false,
			team:        teamSample,
			mockErrGet:  errors.New("get team error"),
			expectedErr: errors.New("get team error"),
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			teamRepo := mocks.NewTeamRepository(t)

			teamRepo.
				On("TeamExists", mock.Anything, tc.team.Name).
				Return(tc.teamExists, tc.mockErrExist).
				Once()

			if tc.mockErrExist == nil && !tc.teamExists {
				teamRepo.
					On("CreateTeam", mock.Anything, mock.AnythingOfType("*domains.Team")).
					Return(tc.mockErrCreate).
					Once()
			}

			if tc.mockErrExist == nil && tc.mockErrCreate == nil && !tc.teamExists {
				teamRepo.
					On("GetTeamByName", mock.Anything, tc.team.Name).
					Return(tc.team, tc.mockErrGet).
					Once()
			}

			svc := New(discardLogger(), teamRepo)
			team, err := svc.AddTeam(nil, tc.team)

			if tc.expectedErr != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedErr, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.team.Name, team.Name)
			require.Equal(t, len(tc.team.Members), len(team.Members))
			for i, member := range tc.team.Members {
				require.Equal(t, member.Name, team.Members[i].Name)
			}
		})

	}
}

func TestService_GetTeam(t *testing.T) {
	teamSample := &domains.Team{
		Name: "team",
		Members: []*domains.User{
			{Name: "user1"},
			{Name: "user2"},
		},
	}

	type testCase struct {
		name string
		team *domains.Team

		mockErrTeam error

		expectedErr error
	}

	cases := []testCase{
		{
			name: "Success",
			team: teamSample,
		},
		{
			name:        "Team not found",
			mockErrTeam: repository.ErrTeamNotFound,
			expectedErr: usecase.ErrTeamNotFound,
		},
		{
			name:        "GetTeamByName returns error",
			mockErrTeam: errors.New("get team error"),
			expectedErr: errors.New("get team error"),
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			teamRepo := mocks.NewTeamRepository(t)

			teamRepo.
				On("GetTeamByName", mock.Anything, "team").
				Return(tc.team, tc.mockErrTeam).
				Once()

			svc := New(discardLogger(), teamRepo)
			team, err := svc.GetTeam(nil, "team")

			if tc.expectedErr != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedErr, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.team.Name, team.Name)
			require.Equal(t, len(tc.team.Members), len(team.Members))
			for i, member := range tc.team.Members {
				require.Equal(t, member.Name, team.Members[i].Name)
			}
		})
	}
}
