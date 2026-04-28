package server

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("server not found")

type Server struct {
	ID          string
	Name        string
	Host        string
	Port        int
	Provider    string
	Status      string
	SSHUser     string
	SSHPassword string
	LastRenewAt *time.Time
	LastRenewStatus string
}

type Repository interface {
	Ping(ctx context.Context) error
	Create(ctx context.Context, server *Server) error
	GetByID(ctx context.Context, id string) (*Server, error)
	List(ctx context.Context) ([]Server, error)
	Update(ctx context.Context, server *Server) error
	Delete(ctx context.Context, id string) error
}

type SSHExecutor interface {
	Run(ctx context.Context, server Server, command string) error
	RunScript(ctx context.Context, server Server, commands []string) error
	TestConnection(ctx context.Context, server Server) error
}
