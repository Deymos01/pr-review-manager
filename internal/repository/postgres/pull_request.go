package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"

	"github.com/Deymos01/pr-review-manager/internal/domains"
	"github.com/Deymos01/pr-review-manager/internal/repository"
)

const (
	numReviewersToAssign = 2
)

func (s *Storage) CreatePullRequest(ctx context.Context, prID, prName, authorID string) ([]string, error) {
	const op = "repository.postgres.CreatePullRequest"

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() { _ = tx.Rollback() }()

	queryPRExists := `SELECT EXISTS(SELECT 1 FROM pull_requests WHERE id = $1)`
	var exists bool
	if err = tx.QueryRowContext(ctx, queryPRExists, prID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if exists {
		return nil, repository.ErrPRAlreadyExists
	}

	queryGetMembers := `
		SELECT id
		FROM users
		WHERE is_active = TRUE
		  AND team_name = (
				SELECT team_name
				FROM users
				WHERE id = $1
			)
		  AND id <> $1
		FOR UPDATE
	`
	rows, err := tx.QueryContext(ctx, queryGetMembers, authorID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() { _ = rows.Close() }()

	var members []string
	for rows.Next() {
		var memberID string
		if err := rows.Scan(&memberID); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		members = append(members, memberID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if len(members) == 0 {
		return nil, fmt.Errorf("%s: no active teammates found for author %s", op, authorID)
	}

	numReviewers := numReviewersToAssign
	if len(members) < numReviewers {
		numReviewers = len(members)
	}
	rand.Shuffle(len(members), func(i, j int) {
		members[i], members[j] = members[j], members[i]
	})
	reviewers := members[:numReviewers]

	queryStatusOpen := `SELECT id FROM statuses WHERE name = 'OPEN'`
	var statusID string
	if err = tx.QueryRowContext(ctx, queryStatusOpen).Scan(&statusID); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	queryCreatePR := `
		INSERT INTO pull_requests (id, name, author_id, status_id)
		VALUES ($1, $2, $3, $4)
	`

	if _, err = tx.ExecContext(ctx, queryCreatePR, prID, prName, authorID, statusID); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	queryAssignReviewer := `
		INSERT INTO reviewers (pull_request_id, user_id)
		VALUES ($1, $2)
	`
	for _, reviewerID := range reviewers {
		if _, err = tx.ExecContext(ctx, queryAssignReviewer, prID, reviewerID); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return reviewers, nil
}

func (s *Storage) PullRequestExists(ctx context.Context, prID string) (bool, error) {
	const op = "repository.postgres.PullRequestExists"

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM pull_requests WHERE id = $1)`
	err := s.db.QueryRowContext(ctx, query, prID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return exists, nil
}

func (s *Storage) PullRequestMerged(ctx context.Context, prID string) (bool, error) {
	const op = "repository.postgres.PullRequestMerged"

	var isMerged bool
	query := `SELECT CASE WHEN st.name = 'MERGED' THEN TRUE ELSE FALSE END
				FROM pull_requests pr
				JOIN statuses st ON pr.status_id = st.id
				WHERE pr.id = $1`
	err := s.db.QueryRowContext(ctx, query, prID).Scan(&isMerged)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return isMerged, nil
}

func (s *Storage) MergePullRequest(ctx context.Context, prID string) error {
	const op = "repository.postgres.MergePullRequest"

	query := `UPDATE pull_requests
				SET status_id = (SELECT id FROM statuses WHERE name = 'MERGED'),
				 	merged_at = NOW()
				WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, prID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) GetPullRequestByID(ctx context.Context, prID string) (*domains.PullRequest, error) {
	const op = "repository.postgres.GetPullRequestByID"

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() { _ = tx.Rollback() }()

	queryPR := `SELECT pr.id, pr.name, pr.author_id, st.name, pr.merged_at
				FROM pull_requests pr
				JOIN statuses st ON pr.status_id = st.id
				WHERE pr.id = $1`

	var pr domains.PullRequest
	pr.Author = &domains.User{}
	err = tx.QueryRowContext(ctx, queryPR, prID).
		Scan(&pr.ID, &pr.Name, &pr.Author.ID, &pr.Status, &pr.MergedAt)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	queryReviewers := `SELECT user_id FROM reviewers WHERE pull_request_id = $1`
	rows, err := tx.QueryContext(ctx, queryReviewers, prID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() { _ = rows.Close() }()

	var reviewers []*domains.Reviewer
	for rows.Next() {
		var reviewerID string
		if err := rows.Scan(&reviewerID); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		reviewers = append(reviewers, &domains.Reviewer{User: &domains.User{ID: reviewerID}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	pr.Reviewers = reviewers

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &pr, nil
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
