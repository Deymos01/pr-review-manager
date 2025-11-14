package repository

import "errors"

var (
	ErrPRAlreadyExists = errors.New("pull request already exists")
	ErrTeamNotFound    = errors.New("team not found")
	ErrNoCandidate     = errors.New("no available candidate for reassignment")
)
