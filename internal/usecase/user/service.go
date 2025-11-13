package user

import (
	"context"
	"log/slog"

	"github.com/Deymos01/pr-review-manager/internal/domains"
)

type Repository interface {
	SetUserStatus(ctx context.Context, userID string, isActive bool) (*domains.User, error)
	UsersReview(ctx context.Context, userID string) ([]*domains.PullRequest, error)
}

type Service struct {
	log  *slog.Logger
	repo Repository
}

func New(log *slog.Logger, repo Repository) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) SetIsActive(ctx context.Context, userID string, isActive bool) (*domains.User, error) {
	const op = "usecase.user.SetIsActive"

	user, err := s.repo.SetUserStatus(ctx, userID, isActive)
	if err != nil {
		s.log.Error("failed to set user is_active", slog.String("op", op), slog.String("err", err.Error()))
		return nil, err
	}

	s.log.Info("user is_active successfully updated", slog.String("user_id", userID), slog.Bool("is_active", isActive))
	return user, nil
}

func (s *Service) GetUsersReview(ctx context.Context, userID string) ([]*domains.PullRequest, error) {
	const op = "usecase.user.GetUsersReview"

	reviews, err := s.repo.UsersReview(ctx, userID)
	if err != nil {
		s.log.Error("failed to get user reviews", slog.String("op", op), slog.String("err", err.Error()))
		return nil, err
	}

	s.log.Info("user reviews successfully retrieved", slog.String("user_id", userID), slog.Int("reviews_count", len(reviews)))
	return reviews, nil
}
