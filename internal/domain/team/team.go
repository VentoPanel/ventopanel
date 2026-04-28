package team

type Team struct {
	ID   string
	Name string
}

type AccessGrant struct {
	TeamID string
	SiteID string
	Role   string
}

type Repository interface{}
