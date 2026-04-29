package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UptimeCheck struct {
	ID         string
	SiteID     string
	CheckedAt  time.Time
	Status     string // "up" | "down"
	LatencyMs  int
	StatusCode int
	Error      string
}

type UptimeRepository struct {
	db *pgxpool.Pool
}

func NewUptimeRepository(db *pgxpool.Pool) *UptimeRepository {
	return &UptimeRepository{db: db}
}

// Insert records one uptime check result.
func (r *UptimeRepository) Insert(ctx context.Context, c UptimeCheck) error {
	var errStr *string
	if c.Error != "" {
		errStr = &c.Error
	}
	var code *int
	if c.StatusCode != 0 {
		code = &c.StatusCode
	}
	var lat *int
	if c.LatencyMs != 0 {
		lat = &c.LatencyMs
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO site_uptime_checks (site_id, status, latency_ms, status_code, error)
		VALUES ($1, $2, $3, $4, $5)`,
		c.SiteID, c.Status, lat, code, errStr,
	)
	return err
}

// IsFirstCheck returns true when no previous checks exist for the site.
// Used to suppress notifications on the very first ping.
func (r *UptimeRepository) IsFirstCheck(ctx context.Context, siteID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM site_uptime_checks WHERE site_id = $1 LIMIT 1)`,
		siteID,
	).Scan(&exists)
	return !exists, err
}

// ListRecent returns the last `limit` checks for a site, newest first.
func (r *UptimeRepository) ListRecent(ctx context.Context, siteID string, limit int) ([]UptimeCheck, error) {
	if limit <= 0 {
		limit = 90
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, site_id, checked_at, status,
		       COALESCE(latency_ms, 0), COALESCE(status_code, 0), COALESCE(error, '')
		FROM site_uptime_checks
		WHERE site_id = $1
		ORDER BY checked_at DESC
		LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []UptimeCheck
	for rows.Next() {
		var c UptimeCheck
		if err := rows.Scan(&c.ID, &c.SiteID, &c.CheckedAt, &c.Status,
			&c.LatencyMs, &c.StatusCode, &c.Error); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UptimePct returns the percentage of 'up' checks in the last `limit` records.
func (r *UptimeRepository) UptimePct(ctx context.Context, siteID string, limit int) (float64, error) {
	if limit <= 0 {
		limit = 90
	}
	row := r.db.QueryRow(ctx, `
		WITH recent AS (
			SELECT status FROM site_uptime_checks
			WHERE site_id = $1
			ORDER BY checked_at DESC
			LIMIT $2
		)
		SELECT COUNT(*) FILTER (WHERE status='up')::float / NULLIF(COUNT(*),0) * 100
		FROM recent`, siteID, limit)

	var pct *float64
	if err := row.Scan(&pct); err != nil {
		return 0, err
	}
	if pct == nil {
		return 0, nil
	}
	return *pct, nil
}

// SiteOverview holds the aggregated uptime info for one site.
type SiteOverview struct {
	SiteID        string
	SiteName      string
	Domain        string
	LastStatus    string    // "up" | "down" | "" (never checked)
	LastCheckedAt time.Time
	LatencyMs     int
	UptimePct90   float64 // last 90 checks
}

// OverviewAll returns the latest check + uptime % for every site that has at least one check.
func (r *UptimeRepository) OverviewAll(ctx context.Context) ([]SiteOverview, error) {
	rows, err := r.db.Query(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (site_id)
				site_id,
				status,
				checked_at,
				COALESCE(latency_ms, 0) AS latency_ms
			FROM site_uptime_checks
			ORDER BY site_id, checked_at DESC
		),
		pct AS (
			SELECT
				site_id,
				COUNT(*) FILTER (WHERE status = 'up')::float / NULLIF(COUNT(*), 0) * 100 AS uptime_pct
			FROM (
				SELECT site_id, status,
					ROW_NUMBER() OVER (PARTITION BY site_id ORDER BY checked_at DESC) AS rn
				FROM site_uptime_checks
			) ranked
			WHERE rn <= 90
			GROUP BY site_id
		)
		SELECT
			s.id,
			s.name,
			s.domain,
			COALESCE(l.status, '')      AS last_status,
			COALESCE(l.checked_at, '1970-01-01'::timestamptz) AS last_checked_at,
			COALESCE(l.latency_ms, 0)   AS latency_ms,
			COALESCE(p.uptime_pct, 0)   AS uptime_pct
		FROM sites s
		LEFT JOIN latest l ON l.site_id = s.id
		LEFT JOIN pct    p ON p.site_id = s.id
		ORDER BY s.name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SiteOverview
	for rows.Next() {
		var o SiteOverview
		if err := rows.Scan(
			&o.SiteID, &o.SiteName, &o.Domain,
			&o.LastStatus, &o.LastCheckedAt, &o.LatencyMs, &o.UptimePct90,
		); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// Prune deletes records older than the most recent `keep` rows for a site.
// Uses checked_at offset instead of NOT IN to avoid large subquery result sets.
func (r *UptimeRepository) Prune(ctx context.Context, siteID string, keep int) error {
	if keep <= 0 {
		keep = 10_080 // 7 days at 1 check/min
	}
	_, err := r.db.Exec(ctx, `
		DELETE FROM site_uptime_checks
		WHERE site_id = $1
		  AND checked_at < (
			SELECT checked_at
			FROM site_uptime_checks
			WHERE site_id = $1
			ORDER BY checked_at DESC
			OFFSET $2
			LIMIT 1
		  )`,
		siteID, keep,
	)
	return err
}
