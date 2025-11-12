package domains

import (
	"time"

	"github.com/google/uuid"
)

type PullRequest struct {
	ID                uuid.UUID
	Name              string
	Author            *User
	Reviewers         []*Reviewer
	Status            string
	NeedMoreReviewers bool
	CreatedAt         time.Time
}
