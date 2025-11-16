package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/repository"
	"github.com/lib/pq"
)

func (s *Storage) CreateTeam(ctx context.Context, team *domains.Team) error {
	const op = "storage.postgres.CreateTeam"

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer func() { _ = tx.Rollback() }()

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
	defer func() { _ = rows.Close() }()

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

func (s *Storage) DeactivateTeamMembers(
	ctx context.Context,
	teamName string,
	userIDs []string,
) (*domains.Team, []*domains.ReassignedPR, error) {
	const op = "storage.postgres.DeactivateTeamMembers"

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() { _ = tx.Rollback() }()

	userIDsPq := pq.Array(userIDs)
	// Ensure all users belong to the team
	query := `SELECT id FROM users
				WHERE team_name = $1 AND id = ANY($2)`

	rows, err := tx.QueryContext(ctx, query, teamName, userIDsPq)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	var found []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", op, err)
		}
		found = append(found, id)
	}
	_ = rows.Close()

	if len(found) != len(userIDs) {
		return nil, nil, repository.ErrTeamCompatibility
	}

	// Deactivate users
	_, err = tx.ExecContext(ctx,
		`UPDATE users
				SET is_active = FALSE
				WHERE id = ANY($1)`, userIDsPq)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	// Get available candidates for reassignment
	rows, err = tx.QueryContext(ctx,
		`SELECT id FROM users
				WHERE team_name = $1 AND is_active = TRUE`, teamName)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	var activeMembers []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", op, err)
		}
		activeMembers = append(activeMembers, id)
	}
	_ = rows.Close()

	// Find PRs where deactivated users are reviewers
	rows, err = tx.QueryContext(ctx,
		`SELECT user_id, pull_request_id FROM reviewers
				WHERE user_id = ANY($1)`, userIDsPq)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	type userPR struct {
		userID string
		prID   string
	}
	var affected []userPR

	for rows.Next() {
		var p userPR
		if err = rows.Scan(&p.userID, &p.prID); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", op, err)
		}
		affected = append(affected, p)
	}
	_ = rows.Close()

	var reassigned []*domains.ReassignedPR

	// Reassign PRs
	for _, a := range affected {
		if len(activeMembers) == 0 {
			// no active members → just remove reviewer
			_, err = tx.ExecContext(ctx, `
                DELETE FROM reviewers
                WHERE user_id = $1 AND pull_request_id = $2
            `, a.userID, a.prID)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %w", op, err)
			}
			continue
		}

		// other reviewers in this PR
		otherReviewers := make(map[string]struct{})
		rows, err = tx.QueryContext(ctx, `
			SELECT user_id FROM reviewers
			WHERE pull_request_id = $1 AND user_id != $2
		`, a.prID, a.userID)
		if err != nil {
			return nil, nil, err
		}

		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				return nil, nil, fmt.Errorf("%s: %w", op, err)
			}
			otherReviewers[id] = struct{}{}
		}
		_ = rows.Close()

		// Get author of the PR
		var prAuthor string
		err = tx.QueryRowContext(ctx, `
			SELECT author_id FROM pull_requests
			WHERE id = $1
		`, a.prID).Scan(&prAuthor)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", op, err)
		}

		// New reviewer should not be the PR author or an existing reviewer
		candidates := make([]string, 0, len(activeMembers))
		for _, member := range activeMembers {
			if _, ok := otherReviewers[member]; member != prAuthor && !ok {
				candidates = append(candidates, member)
			}
		}
		if len(candidates) == 0 {
			// no suitable candidates, just remove reviewer
			_, err = tx.ExecContext(ctx, `
				DELETE FROM reviewers
				WHERE user_id = $1 AND pull_request_id = $2
			`, a.userID, a.prID)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %w", op, err)
			}
			continue
		}

		newReviewer := candidates[rand.Intn(len(candidates))]

		// remove old reviewer
		_, err = tx.ExecContext(ctx, `
            DELETE FROM reviewers
            WHERE user_id = $1 AND pull_request_id = $2
        `, a.userID, a.prID)
		if err != nil {
			return nil, nil, err
		}

		// add new reviewer
		_, err = tx.ExecContext(ctx, `
            INSERT INTO reviewers (user_id, pull_request_id)
            VALUES ($1, $2)
        `, newReviewer, a.prID)
		if err != nil {
			return nil, nil, err
		}

		reassigned = append(reassigned, &domains.ReassignedPR{
			PrID:      a.prID,
			OldUserID: a.userID,
			NewUserID: newReviewer,
		})
	}

	// Get updated team
	team := &domains.Team{Name: teamName}

	rows, err = tx.QueryContext(ctx,
		`SELECT id, name, is_active FROM users
				WHERE team_name = $1`, teamName)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	var users []*domains.User
	for rows.Next() {
		var user domains.User
		if err := rows.Scan(&user.ID, &user.Name, &user.IsActive); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", op, err)
		}
		users = append(users, &user)
	}
	_ = rows.Close()

	team.Members = users

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	return team, reassigned, nil
}
