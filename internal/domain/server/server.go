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

// ServerStats holds live resource usage pulled from the remote server via SSH.
type ServerStats struct {
	CPUCores   int     `json:"cpu_cores"`
	LoadAvg1   float64 `json:"load_avg_1"`
	RAMTotalMB int64   `json:"ram_total_mb"`
	RAMUsedMB  int64   `json:"ram_used_mb"`
	DiskTotal  string  `json:"disk_total"`
	DiskUsed   string  `json:"disk_used"`
	DiskFree   string  `json:"disk_free"`
	DiskPct    string  `json:"disk_pct"`
	Uptime     string  `json:"uptime"`
}

type SSHExecutor interface {
	Run(ctx context.Context, server Server, command string) error
	RunOutput(ctx context.Context, server Server, command string) (string, error)
	RunScript(ctx context.Context, server Server, commands []string) error
	TestConnection(ctx context.Context, server Server) error
}
