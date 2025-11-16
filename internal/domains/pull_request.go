package domains

import (
	"time"
)

type PullRequest struct {
	ID                string
	Name              string
	Author            *User
	Reviewers         []*Reviewer
	Status            string
	NeedMoreReviewers bool
	CreatedAt         time.Time
	MergedAt          *time.Time
}

type ReassignedPR struct {
	PrID      string
	OldUserID string
	NewUserID string
}
