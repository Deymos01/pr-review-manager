package team

import (
	"context"
	"errors"
	"log/slog"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/repository"
	"github.com/Deymos01/pr-review-manager/internal/usecase"
)

//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name=TeamRepository
type TeamRepository interface {
	CreateTeam(ctx context.Context, team *domains.Team) error
	TeamExists(ctx context.Context, name string) (bool, error)
	GetTeamByName(ctx context.Context, name string) (*domains.Team, error)
}

type Service struct {
	log  *slog.Logger
	repo TeamRepository
}

func New(log *slog.Logger, repo TeamRepository) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) AddTeam(ctx context.Context, team *domains.Team) (*domains.Team, error) {
	const op = "usecase.team.AddTeam"

	exists, err := s.repo.TeamExists(ctx, team.Name)
	if err != nil {
		s.log.Error("failed to check team existence", slog.String("op", op), slog.String("err", err.Error()))
		return nil, err
	}
	if exists {
		s.log.Warn("team already exists", slog.String("team", team.Name))
		return nil, usecase.ErrTeamAlreadyExists
	}

	if err := s.repo.CreateTeam(ctx, team); err != nil {
		s.log.Error("failed to create team", slog.String("op", op), slog.String("err", err.Error()))
		return nil, err
	}

	created, err := s.repo.GetTeamByName(ctx, team.Name)
	if err != nil {
		s.log.Error("failed to get created team", slog.String("op", op), slog.String("err", err.Error()))
		return nil, err
	}

	s.log.Info("team successfully created", slog.String("team", team.Name))
	return created, nil
}

func (s *Service) GetTeam(ctx context.Context, teamName string) (*domains.Team, error) {
	const op = "usecase.team.GetTeam"

	team, err := s.repo.GetTeamByName(ctx, teamName)
	if err != nil {
		if errors.Is(err, repository.ErrTeamNotFound) {
			s.log.Warn("team not found", slog.String("team", teamName))
			return nil, usecase.ErrTeamNotFound
		}

		s.log.Error("failed to get team by name", slog.String("op", op), slog.String("err", err.Error()))
		return nil, err
	}

	s.log.Info("team successfully retrieved", slog.String("team", teamName))
	return team, nil
}
