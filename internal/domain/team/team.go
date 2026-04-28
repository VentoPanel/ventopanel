package team

import "context"

type Team struct {
	ID   string
	Name string
}

type AccessGrant struct {
	TeamID string
	SiteID string
	Role   string
}

type Repository interface {
	HasSiteAccess(ctx context.Context, teamID, siteID string) (bool, error)
	GetSiteRole(ctx context.Context, teamID, siteID string) (string, error)
	GrantSiteAccess(ctx context.Context, teamID, siteID, role string) error

	HasServerAccess(ctx context.Context, teamID, serverID string) (bool, error)
	GetServerRole(ctx context.Context, teamID, serverID string) (string, error)
	GrantServerAccess(ctx context.Context, teamID, serverID, role string) error
}
