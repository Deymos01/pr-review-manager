package domains

type User struct {
	ID       string
	Name     string
	TeamName *string
	IsActive bool
}
