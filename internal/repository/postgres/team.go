package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/repository"
)

func (s *Storage) CreateTeam(ctx context.Context, team *domains.Team) error {
	const op = "storage.postgres.CreateTeam"

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer tx.Rollback()

	query := `INSERT INTO teams (name) VALUES ($1)`
	_, err = tx.ExecContext(ctx, query, team.Name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	for _, member := range team.Members {
		query = `INSERT INTO users (id, name, is_active, team_name) VALUES ($1, $2, $3, $4)
					ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, team_name = EXCLUDED.team_name`
		_, err := tx.ExecContext(ctx, query, member.ID, member.Name, member.IsActive, member.TeamName)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) TeamExists(ctx context.Context, name string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM teams WHERE name = $1)`
	err := s.db.QueryRowContext(ctx, query, name).Scan(&exists)
	if err != nil {
		return false, repository.ErrTeamNotFound
	}

	return exists, nil
}

func (s *Storage) GetTeamByName(ctx context.Context, name string) (*domains.Team, error) {
	const op = "storage.postgres.GetTeamByName"

	var team domains.Team
	query := `SELECT users.id, users.name, users.is_active FROM teams
				JOIN users ON teams.name = users.team_name
				WHERE teams.name = $1`
	rows, err := s.db.QueryContext(ctx, query, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrTeamNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var users []*domains.User
	for rows.Next() {
		var user domains.User
		if err := rows.Scan(&user.ID, &user.Name, &user.IsActive); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		user.TeamName = &name
		users = append(users, &user)
	}
	team.Name = name
	team.Members = users

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &team, nil
}
