package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/your-org/ventopanel/internal/domain/audit"
)

type StatusEventRepository struct {
	db *pgxpool.Pool
}

func NewStatusEventRepository(db *pgxpool.Pool) *StatusEventRepository {
	return &StatusEventRepository{db: db}
}

func (r *StatusEventRepository) WriteStatusEvent(event audit.StatusEvent) error {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}

	const query = `
		INSERT INTO status_events (resource_type, resource_id, from_status, to_status, reason, task_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(context.Background(), query,
		event.ResourceType,
		event.ResourceID,
		event.FromStatus,
		event.ToStatus,
		event.Reason,
		event.TaskID,
		event.CreatedAt,
	)
	return err
}

func (r *StatusEventRepository) ListStatusEvents(filter audit.StatusEventFilter) ([]audit.StatusEvent, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	conditions := make([]string, 0, 6)
	args := make([]any, 0, 8)

	if filter.ResourceType != "" {
		conditions = append(conditions, fmt.Sprintf("resource_type = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(filter.ResourceType))
	}
	if filter.ResourceID != "" {
		conditions = append(conditions, fmt.Sprintf("resource_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(filter.ResourceID))
	}
	if filter.FromStatus != "" {
		conditions = append(conditions, fmt.Sprintf("from_status = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(filter.FromStatus))
	}
	if filter.ToStatus != "" {
		conditions = append(conditions, fmt.Sprintf("to_status = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(filter.ToStatus))
	}
	if filter.Since != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", len(args)+1))
		args = append(args, filter.Since.UTC())
	}
	if filter.BeforeCreatedAt != nil {
		if strings.TrimSpace(filter.BeforeID) != "" {
			conditions = append(conditions, fmt.Sprintf("(created_at, id) < ($%d, $%d)", len(args)+1, len(args)+2))
			args = append(args, filter.BeforeCreatedAt.UTC(), strings.TrimSpace(filter.BeforeID))
		} else {
			conditions = append(conditions, fmt.Sprintf("created_at < $%d", len(args)+1))
			args = append(args, filter.BeforeCreatedAt.UTC())
		}
	}

	query := `
		SELECT id, resource_type, resource_id, from_status, to_status, reason, task_id, created_at
		FROM status_events
	`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", len(args)+1)
	args = append(args, limit)

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]audit.StatusEvent, 0, limit)
	for rows.Next() {
		var event audit.StatusEvent
		if err := rows.Scan(
			&event.ID,
			&event.ResourceType,
			&event.ResourceID,
			&event.FromStatus,
			&event.ToStatus,
			&event.Reason,
			&event.TaskID,
			&event.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func (r *StatusEventRepository) CountStatusEvents(filter audit.StatusEventFilter) (int64, error) {
	conditions := make([]string, 0, 5)
	args := make([]any, 0, 5)

	if filter.ResourceType != "" {
		conditions = append(conditions, fmt.Sprintf("resource_type = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(filter.ResourceType))
	}
	if filter.ResourceID != "" {
		conditions = append(conditions, fmt.Sprintf("resource_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(filter.ResourceID))
	}
	if filter.FromStatus != "" {
		conditions = append(conditions, fmt.Sprintf("from_status = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(filter.FromStatus))
	}
	if filter.ToStatus != "" {
		conditions = append(conditions, fmt.Sprintf("to_status = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(filter.ToStatus))
	}
	if filter.Since != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", len(args)+1))
		args = append(args, filter.Since.UTC())
	}

	query := `SELECT COUNT(*) FROM status_events`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int64
	if err := r.db.QueryRow(context.Background(), query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}
