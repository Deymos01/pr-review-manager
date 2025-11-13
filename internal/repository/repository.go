package repository

import "errors"

var (
	ErrTeamNotFound = errors.New("team not found")
	ErrNoCandidate  = errors.New("no available candidate for reassignment")
)
