package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/your-org/ventopanel/internal/domain/tasklog"
)

type TaskLogRepository struct {
	db *pgxpool.Pool
}

func NewTaskLogRepository(db *pgxpool.Pool) *TaskLogRepository {
	return &TaskLogRepository{db: db}
}

func (r *TaskLogRepository) Create(ctx context.Context, log *tasklog.TaskLog) error {
	const q = `
		INSERT INTO task_logs (id, site_id, task_type, status, output, started_at)
		VALUES (gen_random_uuid(), $1, $2, 'running', '', NOW())
		RETURNING id, started_at`
	return r.db.QueryRow(ctx, q, log.SiteID, log.TaskType).Scan(&log.ID, &log.StartedAt)
}

func (r *TaskLogRepository) Finish(ctx context.Context, id, status, output string) error {
	now := time.Now()
	const q = `
		UPDATE task_logs
		SET status = $1, output = $2, finished_at = $3
		WHERE id = $4`
	_, err := r.db.Exec(ctx, q, status, output, now, id)
	return err
}

func (r *TaskLogRepository) ListBySiteID(ctx context.Context, siteID string, limit int) ([]tasklog.TaskLog, error) {
	const q = `
		SELECT id, site_id, task_type, status, output, started_at, finished_at
		FROM task_logs
		WHERE site_id = $1
		ORDER BY started_at DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, q, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []tasklog.TaskLog
	for rows.Next() {
		var l tasklog.TaskLog
		if err := rows.Scan(&l.ID, &l.SiteID, &l.TaskType, &l.Status, &l.Output, &l.StartedAt, &l.FinishedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}
