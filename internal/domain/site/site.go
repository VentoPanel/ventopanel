package site

import (
	"context"
	"errors"

	serverdomain "github.com/your-org/ventopanel/internal/domain/server"
)

var (
	ErrNotFound       = errors.New("site not found")
	ErrServerNotFound = errors.New("server not found")
)

type Site struct {
	ID               string
	ServerID         string
	Name             string
	Domain           string
	Runtime          string
	RepositoryURL    string
	Branch           string
	Status           string
	WebhookToken     string
	HealthcheckPath  string // URL path for uptime checks, e.g. "/health". Defaults to "/".
}

type Repository interface {
	Create(ctx context.Context, site *Site) error
	GetByID(ctx context.Context, id string) (*Site, error)
	List(ctx context.Context) ([]Site, error)
	Update(ctx context.Context, site *Site) error
	Delete(ctx context.Context, id string) error
}

type ServerRepository interface {
	GetByID(ctx context.Context, id string) (*serverdomain.Server, error)
}
