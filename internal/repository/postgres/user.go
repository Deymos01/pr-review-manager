package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/repository"
)

func (s *Storage) UserExists(ctx context.Context, userID string) (bool, error) {
	const op = "repository.postgres.UserExists"

	query := `
		SELECT EXISTS (
			SELECT 1
			FROM users
			WHERE id = $1
		);
	`

	var exists bool
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return exists, nil
}

func (s *Storage) UserHasActiveTeam(ctx context.Context, userID string) (bool, error) {
	const op = "repository.postgres.UserHasActiveTeam"

	query := `
		SELECT EXISTS (
			SELECT 1
			FROM users
			WHERE id = $1 AND team_name IS NOT NULL
		)
	`

	var exists bool
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return exists, nil
}

func (s *Storage) SetUserStatus(ctx context.Context, userID string, isActive bool) (*domains.User, error) {
	const op = "repository.postgres.user.SetIsActive"

	query := `
		UPDATE users
		SET is_active = $1
		WHERE id = $2
		RETURNING id, name, team_name, is_active
	`

	var user domains.User
	err := s.db.QueryRowContext(ctx, query, isActive, userID).
		Scan(&user.ID, &user.Name, &user.TeamName, &user.IsActive)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &user, nil
}

func (s *Storage) UsersReview(ctx context.Context, userID string) ([]*domains.PullRequest, error) {
	const op = "repository.postgres.user.GetUsersReview"

	query := `SELECT pr.id, pr.name, pr.author_id, st.name
				FROM pull_requests pr
				JOIN statuses st ON pr.status_id = st.id
				JOIN reviewers rev ON pr.id = rev.pull_request_id
				WHERE rev.user_id = $1`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() { _ = rows.Close() }()

	var reviews []*domains.PullRequest
	for rows.Next() {
		var pr domains.PullRequest
		pr.Author = &domains.User{}
		if err := rows.Scan(&pr.ID, &pr.Name, &pr.Author.ID, &pr.Status); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		reviews = append(reviews, &pr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return reviews, nil
}

func (s *Storage) UserAssigned(ctx context.Context, prID, userID string) (bool, error) {
	const op = "repository.postgres.user.UserAssigned"

	query := `
		SELECT EXISTS (
			SELECT 1
			FROM reviewers
			WHERE pull_request_id = $1 AND user_id = $2
		);
	`

	var exists bool
	err := s.db.QueryRowContext(ctx, query, prID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return exists, nil
}

func (s *Storage) ReassignReviewer(ctx context.Context, prID, oldUserID string) (string, error) {
	const op = "repository.postgres.user.ReassignReviewer"

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	defer func() { _ = tx.Rollback() }()

	querySelect := `
		SELECT u.id
		FROM users u
		WHERE u.team_name = (SELECT team_name FROM users WHERE id = $1) AND
		      u.id != $1 AND u.is_active AND
		      u.id NOT IN (SELECT author_id FROM pull_requests pr WHERE pr.id = $2) AND 
		      u.id NOT IN (SELECT user_id 
    						FROM reviewers 
    						WHERE pull_request_id = $2)
		ORDER BY RANDOM()
		LIMIT 1;
	`

	var newUserID string
	err = tx.QueryRowContext(ctx, querySelect, oldUserID, prID).Scan(&newUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", repository.ErrNoCandidate
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}

	queryUpdate := `
		UPDATE reviewers
		SET user_id = $1,
		 	assigned_at = NOW()
		WHERE pull_request_id = $2 AND user_id = $3;
	`

	_, err = tx.ExecContext(ctx, queryUpdate, newUserID, prID, oldUserID)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return newUserID, nil
}
