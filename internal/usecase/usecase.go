package usecase

import "errors"

var (
	ErrTeamAlreadyExists   = errors.New("team already exists")
	ErrTeamNotFound        = errors.New("team not found")
	ErrUserNotFound        = errors.New("user not found")
	ErrPRAlreadyExists     = errors.New("pull request already exists")
	ErrPullRequestNotFound = errors.New("pull request not found")
)
