package user

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/usecase/user/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestService_SetUserIsActive(t *testing.T) {
	userSample := &domains.User{
		ID:       "123",
		Name:     "John",
		IsActive: true,
	}

	type testCase struct {
		name     string
		userID   string
		isActive bool

		mockUser *domains.User
		mockErr  error

		expectedErr error
	}

	cases := []testCase{
		{
			name:     "Success",
			userID:   "123",
			isActive: true,
			mockUser: userSample,
		},
		{
			name:        "SetUserStatus returns error",
			userID:      "123",
			isActive:    false,
			mockErr:     errors.New("update error"),
			expectedErr: errors.New("update error"),
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRepo := mocks.NewUserRepository(t)

			userRepo.
				On("SetUserStatus", mock.Anything, tc.userID, tc.isActive).
				Return(tc.mockUser, tc.mockErr).
				Once()

			svc := New(discardLogger(), userRepo)
			user, err := svc.SetUserIsActive(context.Background(), tc.userID, tc.isActive)

			if tc.expectedErr != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedErr, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.mockUser.ID, user.ID)
			require.Equal(t, tc.mockUser.IsActive, user.IsActive)
		})
	}
}

func TestService_GetUsersReview(t *testing.T) {
	reviewSample := []*domains.PullRequest{
		{ID: "1", Name: "Fix bug"},
		{ID: "2", Name: "Add feature"},
	}

	type testCase struct {
		name   string
		userID string

		mockReviews []*domains.PullRequest
		mockErr     error

		expectedErr error
	}

	cases := []testCase{
		{
			name:        "Success",
			userID:      "123",
			mockReviews: reviewSample,
		},
		{
			name:        "UsersReview returns error",
			userID:      "123",
			mockErr:     errors.New("review error"),
			expectedErr: errors.New("review error"),
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRepo := mocks.NewUserRepository(t)

			userRepo.
				On("UsersReview", mock.Anything, tc.userID).
				Return(tc.mockReviews, tc.mockErr).
				Once()

			svc := New(discardLogger(), userRepo)
			reviews, err := svc.GetUsersReview(context.Background(), tc.userID)

			if tc.expectedErr != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedErr, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, len(tc.mockReviews), len(reviews))

			for i := range reviews {
				require.Equal(t, tc.mockReviews[i].ID, reviews[i].ID)
				require.Equal(t, tc.mockReviews[i].Name, reviews[i].Name)
			}
		})
	}
}
