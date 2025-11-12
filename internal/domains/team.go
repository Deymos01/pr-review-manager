package domains

import "github.com/google/uuid"

type Team struct {
	ID      uuid.UUID
	Name    string
	Members []*User
}
