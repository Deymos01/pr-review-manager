package pull_request

import (
	"context"
	"errors"
	"log/slog"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/repository"
	"github.com/Deymos01/pr-review-manager/internal/usecase"
)

//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name=UserRepository
type UserRepository interface {
	UserExists(ctx context.Context, userID string) (bool, error)
	UserAssigned(ctx context.Context, prID, userID string) (bool, error)
	UserHasActiveTeam(ctx context.Context, authorID string) (bool, error)
}

//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name=PullRequestRepository
type PullRequestRepository interface {
	CreatePullRequest(ctx context.Context, prID, prName, authorID string) ([]string, error)
	PullRequestExists(ctx context.Context, prID string) (bool, error)
	PullRequestMerged(ctx context.Context, prID string) (bool, error)
	MergePullRequest(ctx context.Context, prID string) error
	GetPullRequestByID(ctx context.Context, prID string) (*domains.PullRequest, error)
	ReassignReviewer(ctx context.Context, prID, oldUserID string) (string, error)
}

type Service struct {
	log      *slog.Logger
	userRepo UserRepository
	prRepo   PullRequestRepository
}

func New(log *slog.Logger, userRepo UserRepository, prRepo PullRequestRepository) *Service {
	return &Service{
		log:      log,
		userRepo: userRepo,
		prRepo:   prRepo,
	}
}

func (s *Service) CreatePullRequest(ctx context.Context, prID, prName, authorID string) ([]string, error) {
	const op = "usecase.pull_request.CreatePullRequest"

	ok, err := s.userRepo.UserExists(ctx, authorID)
	if err != nil {
		s.log.Error("failed to check if author exists", slog.String("op", op), slog.String("err", err.Error()))
		return nil, err
	}
	if !ok {
		s.log.Warn("author does not exist", slog.String("author_id", authorID))
		return nil, usecase.ErrUserNotFound
	}

	ok, err = s.userRepo.UserHasActiveTeam(ctx, authorID)
	if err != nil {
		s.log.Error("failed to check if author has active team", slog.String("op", op), slog.String("err", err.Error()))
		return nil, err
	}
	if !ok {
		s.log.Warn("author does not have an active team", slog.String("author_id", authorID))
		return nil, usecase.ErrTeamNotFound
	}

	assignedReviewers, err := s.prRepo.CreatePullRequest(ctx, prID, prName, authorID)
	if err != nil {
		s.log.Error("failed to create pull request", slog.String("op", op), slog.String("err", err.Error()))
		return nil, err
	}

	s.log.Info("pull request created and reviewers assigned", slog.String("pr_id", prID))
	return assignedReviewers, nil
}

func (s *Service) MergePullRequest(ctx context.Context, prID string) (*domains.PullRequest, error) {
	const op = "usecase.pull_request.MergePullRequest"

	ok, err := s.prRepo.PullRequestExists(ctx, prID)
	if err != nil {
		s.log.Error("failed to check if pull request exists", slog.String("op", op), slog.String("err", err.Error()))
		return nil, err
	}
	if !ok {
		s.log.Warn("pull request does not exist", slog.String("pr_id", prID))
		return nil, usecase.ErrPullRequestNotFound
	}

	err = s.prRepo.MergePullRequest(ctx, prID)
	if err != nil {
		s.log.Error("failed to merge pull request", slog.String("op", op), slog.String("err", err.Error()))
		return nil, err
	}

	pr, err := s.prRepo.GetPullRequestByID(ctx, prID)
	if err != nil {
		s.log.Error("failed to get pull request", slog.String("op", op), slog.String("err", err.Error()))
		return nil, err
	}

	s.log.Info("pull request merged", slog.String("pr_id", prID))
	return pr, nil
}

func (s *Service) ReassignReviewer(ctx context.Context, prID, oldUserID string) (*domains.PullRequest, string, error) {
	const op = "usecase.pull_request.ReassignReviewer"

	ok, err := s.prRepo.PullRequestExists(ctx, prID)
	if err != nil {
		s.log.Error("failed to check if pull request exists", slog.String("op", op), slog.String("err", err.Error()))
		return nil, "", err
	}
	if !ok {
		s.log.Warn("pull request does not exist", slog.String("pr_id", prID))
		return nil, "", usecase.ErrPullRequestNotFound
	}

	ok, err = s.prRepo.PullRequestMerged(ctx, prID)
	if err != nil {
		s.log.Error("failed to check if pull request is merged", slog.String("op", op), slog.String("err", err.Error()))
		return nil, "", err
	}
	if ok {
		s.log.Warn("pull request is already merged", slog.String("pr_id", prID))
		return nil, "", usecase.ErrPRAlreadyMerged
	}

	ok, err = s.userRepo.UserExists(ctx, oldUserID)
	if err != nil {
		s.log.Error("failed to check if user exists", slog.String("op", op), slog.String("err", err.Error()))
		return nil, "", err
	}
	if !ok {
		s.log.Warn("user does not exist", slog.String("user_id", oldUserID))
		return nil, "", usecase.ErrUserNotFound
	}

	ok, err = s.userRepo.UserAssigned(ctx, prID, oldUserID)
	if err != nil {
		s.log.Error("failed to check if user is assigned to the pull request",
			slog.String("op", op),
			slog.String("err", err.Error()))
		return nil, "", err
	}
	if !ok {
		s.log.Warn("user is not assigned to the pull request",
			slog.String("pr_id", prID),
			slog.String("user_id", oldUserID))
		return nil, "", usecase.ErrUserNotAssigned
	}

	newUserID, err := s.prRepo.ReassignReviewer(ctx, prID, oldUserID)
	if err != nil {
		if errors.Is(err, repository.ErrNoCandidate) {
			s.log.Warn("no available reviewer to reassign",
				slog.String("pr_id", prID),
				slog.String("old_user_id", oldUserID))
			return nil, "", usecase.ErrNoAvailableReviewer
		}
		s.log.Error("failed to reassign reviewer",
			slog.String("op", op),
			slog.String("err", err.Error()))
		return nil, "", err
	}

	pr, err := s.prRepo.GetPullRequestByID(ctx, prID)
	if err != nil {
		s.log.Error("failed to get pull request",
			slog.String("op", op),
			slog.String("err", err.Error()))
		return nil, "", err
	}

	s.log.Info("reviewer reassigned", slog.String("pr_id", prID),
		slog.String("old_user_id", oldUserID),
		slog.String("new_user_id", newUserID))

	return pr, newUserID, nil
}
