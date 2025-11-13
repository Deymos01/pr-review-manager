package pull_request

import (
	"context"
	"errors"
	"log/slog"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/repository"
	"github.com/Deymos01/pr-review-manager/internal/usecase"
)

type Repository interface {
	UserExists(ctx context.Context, userID string) (bool, error)
	UserAssigned(ctx context.Context, prID, userID string) (bool, error)
	UserHasActiveTeam(ctx context.Context, authorID string) (bool, error)
	ReassignReviewer(ctx context.Context, prID, oldUserID string) (string, error)
	CreatePullRequest(ctx context.Context, prID, prName, authorID string) ([]string, error)
	PullRequestExists(ctx context.Context, prID string) (bool, error)
	PullRequestMerged(ctx context.Context, prID string) (bool, error)
	MergePullRequest(ctx context.Context, prID string) error
	GetPullRequestByID(ctx context.Context, prID string) (*domains.PullRequest, error)
}

type Service struct {
	log  *slog.Logger
	repo Repository
}

func New(log *slog.Logger, repo Repository) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) CreatePullRequest(ctx context.Context, prID, prName, authorID string) ([]string, error) {
	const op = "usecase.pull_request.CreatePullRequest"

	ok, err := s.repo.UserExists(ctx, authorID)
	if err != nil {
		s.log.Error("failed to check if author exists", slog.String("op", op), slog.String("err", err.Error()))
		return nil, err
	}
	if !ok {
		s.log.Warn("author does not exist", slog.String("author_id", authorID))
		return nil, usecase.ErrUserNotFound
	}

	ok, err = s.repo.UserHasActiveTeam(ctx, authorID)
	if err != nil {
		s.log.Error("failed to check if author has active team", slog.String("op", op), slog.String("err", err.Error()))
		return nil, err
	}
	if !ok {
		s.log.Warn("author does not have an active team", slog.String("author_id", authorID))
		return nil, usecase.ErrTeamNotFound
	}

	assignedReviewers, err := s.repo.CreatePullRequest(ctx, prID, prName, authorID)
	if err != nil {
		s.log.Error("failed to create pull request", slog.String("op", op), slog.String("err", err.Error()))
		return nil, err
	}

	s.log.Info("pull request created and reviewers assigned", slog.String("pr_id", prID))
	return assignedReviewers, nil
}

func (s *Service) MergePullRequest(ctx context.Context, prID string) (*domains.PullRequest, error) {
	const op = "usecase.pull_request.MergePullRequest"

	ok, err := s.repo.PullRequestExists(ctx, prID)
	if err != nil {
		s.log.Error("failed to check if pull request exists", slog.String("op", op), slog.String("err", err.Error()))
		return nil, err
	}
	if !ok {
		s.log.Warn("pull request does not exist", slog.String("pr_id", prID))
		return nil, usecase.ErrPullRequestNotFound
	}

	err = s.repo.MergePullRequest(ctx, prID)
	if err != nil {
		s.log.Error("failed to merge pull request", slog.String("op", op), slog.String("err", err.Error()))
		return nil, err
	}

	pr, err := s.repo.GetPullRequestByID(ctx, prID)
	if err != nil {
		s.log.Error("failed to get pull request", slog.String("op", op), slog.String("err", err.Error()))
		return nil, err
	}

	s.log.Info("pull request merged", slog.String("pr_id", prID))
	return pr, nil
}

func (s *Service) ReassignReviewer(ctx context.Context, prID, oldUserID string) (*domains.PullRequest, string, error) {
	const op = "usecase.pull_request.ReassignReviewer"

	ok, err := s.repo.PullRequestExists(ctx, prID)
	if err != nil {
		s.log.Error("failed to check if pull request exists", slog.String("op", op), slog.String("err", err.Error()))
		return nil, "", err
	}
	if !ok {
		s.log.Warn("pull request does not exist", slog.String("pr_id", prID))
		return nil, "", usecase.ErrPullRequestNotFound
	}

	ok, err = s.repo.PullRequestMerged(ctx, prID)
	if err != nil {
		s.log.Error("failed to check if pull request is merged", slog.String("op", op), slog.String("err", err.Error()))
		return nil, "", err
	}
	if ok {
		s.log.Warn("pull request is already merged", slog.String("pr_id", prID))
		return nil, "", usecase.ErrPRAlreadyMerged
	}

	ok, err = s.repo.UserExists(ctx, oldUserID)
	if err != nil {
		s.log.Error("failed to check if user exists", slog.String("op", op), slog.String("err", err.Error()))
		return nil, "", err
	}
	if !ok {
		s.log.Warn("user does not exist", slog.String("user_id", oldUserID))
		return nil, "", usecase.ErrUserNotFound
	}

	ok, err = s.repo.UserAssigned(ctx, prID, oldUserID)
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

	newUserID, err := s.repo.ReassignReviewer(ctx, prID, oldUserID)
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

	pr, err := s.repo.GetPullRequestByID(ctx, prID)
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
