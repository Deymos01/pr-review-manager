package domains

import "github.com/google/uuid"

type User struct {
	ID       uuid.UUID
	Name     string
	TeamID   *uuid.UUID
	IsActive bool
}
