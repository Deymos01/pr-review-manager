package pull_request_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/repository"
	"github.com/Deymos01/pr-review-manager/internal/usecase"
	"github.com/Deymos01/pr-review-manager/internal/usecase/pull_request"
	"github.com/Deymos01/pr-review-manager/internal/usecase/pull_request/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCreatePullRequest(t *testing.T) {
	type testCase struct {
		name         string
		authorExists bool
		hasTeam      bool

		mockErrAuthor error
		mockErrTeam   error
		mockErrCreate error

		expectedErr error
	}

	cases := []testCase{
		{
			name:         "Success",
			authorExists: true,
			hasTeam:      true,
		},
		{
			name:         "Author does not exist",
			authorExists: false,
			expectedErr:  usecase.ErrUserNotFound,
		},
		{
			name:         "Author has no active team",
			authorExists: true,
			hasTeam:      false,
			expectedErr:  usecase.ErrTeamNotFound,
		},
		{
			name:          "UserExists returns error",
			mockErrAuthor: errors.New("user exists error"),
			expectedErr:   errors.New("user exists error"),
		},
		{
			name:         "UserHasActiveTeam returns error",
			authorExists: true,
			mockErrTeam:  errors.New("has active team error"),
			expectedErr:  errors.New("has active team error"),
		},
		{
			name:          "CreatePullRequest returns error",
			authorExists:  true,
			hasTeam:       true,
			mockErrCreate: errors.New("create pull request error"),
			expectedErr:   errors.New("create pull request error"),
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRepo := mocks.NewUserRepository(t)
			prRepo := mocks.NewPullRequestRepository(t)

			userRepo.
				On("UserExists", mock.Anything, "authorID").
				Return(tc.authorExists, tc.mockErrAuthor).
				Once()

			if tc.mockErrAuthor == nil && tc.authorExists {
				userRepo.
					On("UserHasActiveTeam", mock.Anything, "authorID").
					Return(tc.hasTeam, tc.mockErrTeam).
					Once()
			}

			if tc.mockErrTeam == nil && tc.authorExists && tc.hasTeam {
				if tc.mockErrCreate != nil {
					prRepo.
						On("CreatePullRequest", mock.Anything, "pr1", "Feature", "authorID").
						Return(nil, tc.mockErrCreate).
						Once()
				} else {
					prRepo.
						On("CreatePullRequest", mock.Anything, "pr1", "Feature", "authorID").
						Return([]string{"u1", "u2"}, nil).
						Once()
				}
			}

			svc := pull_request.New(discardLogger(), userRepo, prRepo)

			res, err := svc.CreatePullRequest(context.Background(), "pr1", "Feature", "authorID")

			if tc.expectedErr != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedErr, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, []string{"u1", "u2"}, res)
		})
	}
}

func TestMergePullRequest(t *testing.T) {
	type testCase struct {
		name   string
		exists bool

		mockErrExists error
		mockErrMerge  error
		mockErrGet    error

		expectedErr error
	}

	cases := []testCase{
		{
			name:   "Success",
			exists: true,
		},
		{
			name:        "PR does not exist",
			exists:      false,
			expectedErr: usecase.ErrPullRequestNotFound,
		},
		{
			name:          "Exists check error",
			mockErrExists: errors.New("exists err"),
			expectedErr:   errors.New("exists err"),
		},
		{
			name:         "MergePullRequest error",
			exists:       true,
			mockErrMerge: errors.New("merge err"),
			expectedErr:  errors.New("merge err"),
		},
		{
			name:        "GetPullRequestByID error",
			exists:      true,
			mockErrGet:  errors.New("get err"),
			expectedErr: errors.New("get err"),
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRepo := mocks.NewUserRepository(t)
			prRepo := mocks.NewPullRequestRepository(t)

			prRepo.
				On("PullRequestExists", mock.Anything, "pr1").
				Return(tc.exists, tc.mockErrExists).
				Once()

			if tc.mockErrExists == nil && tc.exists {
				prRepo.
					On("MergePullRequest", mock.Anything, "pr1").
					Return(tc.mockErrMerge).
					Once()
			}

			if tc.mockErrMerge == nil && tc.exists {
				if tc.mockErrGet != nil {
					prRepo.
						On("GetPullRequestByID", mock.Anything, "pr1").
						Return(nil, tc.mockErrGet).
						Once()
				} else {
					prRepo.
						On("GetPullRequestByID", mock.Anything, "pr1").
						Return(&domains.PullRequest{ID: "pr1"}, nil).
						Once()
				}
			}

			svc := pull_request.New(discardLogger(), userRepo, prRepo)
			pr, err := svc.MergePullRequest(context.Background(), "pr1")

			if tc.expectedErr != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedErr, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, "pr1", pr.ID)
		})
	}
}

func TestReassignReviewer(t *testing.T) {
	type testCase struct {
		name         string
		prExists     bool
		prMerged     bool
		userExists   bool
		userAssigned bool

		mockErrExists   error
		mockErrMerged   error
		mockErrUser     error
		mockErrAssigned error
		mockErrReassign error
		mockErrGet      error

		expectedErr error
	}

	cases := []testCase{
		{
			name:         "Success",
			prExists:     true,
			prMerged:     false,
			userExists:   true,
			userAssigned: true,
		},
		{
			name:        "PR does not exist",
			prExists:    false,
			expectedErr: usecase.ErrPullRequestNotFound,
		},
		{
			name:          "PullRequestExists returns error",
			prExists:      false,
			mockErrExists: errors.New("pr exists returns err"),
			expectedErr:   errors.New("pr exists returns err"),
		},
		{
			name:        "PR is already merged",
			prExists:    true,
			prMerged:    true,
			expectedErr: usecase.ErrPRAlreadyMerged,
		},
		{
			name:          "PullRequestMerged returns error",
			prExists:      true,
			mockErrMerged: errors.New("pr merged returns err"),
			expectedErr:   errors.New("pr merged returns err"),
		},
		{
			name:        "User does not exist",
			prExists:    true,
			prMerged:    false,
			userExists:  false,
			expectedErr: usecase.ErrUserNotFound,
		},
		{
			name:        "UserExists returns error",
			prExists:    true,
			prMerged:    false,
			mockErrUser: errors.New("user exists returns err"),
			expectedErr: errors.New("user exists returns err"),
		},
		{
			name:         "User not assigned",
			prExists:     true,
			prMerged:     false,
			userExists:   true,
			userAssigned: false,
			expectedErr:  usecase.ErrUserNotAssigned,
		},
		{
			name:            "UserAssigned returns error",
			prExists:        true,
			prMerged:        false,
			userExists:      true,
			mockErrAssigned: errors.New("user assigned returns err"),
			expectedErr:     errors.New("user assigned returns err"),
		},
		{
			name:            "Reassign returns ErrNoCandidate",
			prExists:        true,
			prMerged:        false,
			userExists:      true,
			userAssigned:    true,
			mockErrReassign: repository.ErrNoCandidate,
			expectedErr:     usecase.ErrNoAvailableReviewer,
		},
		{
			name:            "Reassign returns other error",
			prExists:        true,
			prMerged:        false,
			userExists:      true,
			userAssigned:    true,
			mockErrReassign: errors.New("reassign err"),
			expectedErr:     errors.New("reassign err"),
		},
		{
			name:         "GetPullRequestByID error",
			prExists:     true,
			prMerged:     false,
			userExists:   true,
			userAssigned: true,
			mockErrGet:   errors.New("get err"),
			expectedErr:  errors.New("get err"),
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRepo := mocks.NewUserRepository(t)
			prRepo := mocks.NewPullRequestRepository(t)

			prRepo.
				On("PullRequestExists", mock.Anything, "pr1").
				Return(tc.prExists, tc.mockErrExists).
				Once()

			if tc.mockErrExists == nil && tc.prExists {
				prRepo.
					On("PullRequestMerged", mock.Anything, "pr1").
					Return(tc.prMerged, tc.mockErrMerged).
					Once()
			}

			if tc.mockErrMerged == nil && tc.prExists && !tc.prMerged {
				userRepo.
					On("UserExists", mock.Anything, "old").
					Return(tc.userExists, tc.mockErrUser).
					Once()
			}

			if tc.mockErrUser == nil && tc.userExists && tc.prExists && !tc.prMerged {
				userRepo.
					On("UserAssigned", mock.Anything, "pr1", "old").
					Return(tc.userAssigned, tc.mockErrAssigned).
					Once()
			}

			if tc.mockErrAssigned == nil && tc.userAssigned {
				if tc.mockErrReassign != nil {
					prRepo.
						On("ReassignReviewer", mock.Anything, "pr1", "old").
						Return("", tc.mockErrReassign).
						Once()
				} else {
					prRepo.
						On("ReassignReviewer", mock.Anything, "pr1", "old").
						Return("newUser", nil).
						Once()
				}
			}

			if tc.mockErrReassign == nil && tc.userAssigned {
				if tc.mockErrGet != nil {
					prRepo.
						On("GetPullRequestByID", mock.Anything, "pr1").
						Return(nil, tc.mockErrGet).
						Once()
				} else {
					prRepo.
						On("GetPullRequestByID", mock.Anything, "pr1").
						Return(&domains.PullRequest{ID: "pr1"}, nil).
						Once()
				}
			}

			svc := pull_request.New(discardLogger(), userRepo, prRepo)
			pr, newID, err := svc.ReassignReviewer(context.Background(), "pr1", "old")

			if tc.expectedErr != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedErr, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, "pr1", pr.ID)
			require.Equal(t, "newUser", newID)
		})
	}
}
