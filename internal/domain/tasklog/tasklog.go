package tasklog

import (
	"context"
	"time"
)

type TaskLog struct {
	ID         string
	SiteID     string
	TaskType   string
	Status     string // running | success | failed
	Output     string
	StartedAt  time.Time
	FinishedAt *time.Time
}

type Repository interface {
	Create(ctx context.Context, log *TaskLog) error
	Finish(ctx context.Context, id, status, output string) error
	ListBySiteID(ctx context.Context, siteID string, limit int) ([]TaskLog, error)
}
