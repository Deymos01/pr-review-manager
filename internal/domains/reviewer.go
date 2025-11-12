package domains

import (
	"time"
)

type Reviewer struct {
	User       *User
	AssignedAt time.Time
}
