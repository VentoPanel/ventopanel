package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DashboardRepository provides aggregated statistics for the metrics dashboard.
type DashboardRepository struct {
	db *pgxpool.Pool
}

func NewDashboardRepository(db *pgxpool.Pool) *DashboardRepository {
	return &DashboardRepository{db: db}
}

// SiteSummary counts sites by status.
type SiteSummary struct {
	Total     int `json:"total"`
	Deployed  int `json:"deployed"`
	Failed    int `json:"failed"`
	Deploying int `json:"deploying"`
	Other     int `json:"other"`
}

// ServerSummary counts servers by status.
type ServerSummary struct {
	Total     int `json:"total"`
	Connected int `json:"connected"`
	Failed    int `json:"failed"`
	Other     int `json:"other"`
}

// UptimeSummary aggregates the latest uptime check per site.
type UptimeSummary struct {
	SitesUp   int     `json:"sites_up"`
	SitesDown int     `json:"sites_down"`
	AvgPct    float64 `json:"avg_pct"`
}

// DeploySummary counts deploy task results for different windows.
type DeploySummary struct {
	Today24hSuccess int `json:"today_24h_success"`
	Today24hFailed  int `json:"today_24h_failed"`
	AllTimeSuccess  int `json:"all_time_success"`
	AllTimeFailed   int `json:"all_time_failed"`
}

// UptimeTrendPoint is one hourly bucket in the uptime trend.
type UptimeTrendPoint struct {
	Hour       time.Time `json:"hour"`
	UpCount    int       `json:"up_count"`
	DownCount  int       `json:"down_count"`
	AvgLatency float64   `json:"avg_latency_ms"`
}

// DeployTrendPoint is one daily bucket in the deploy activity chart.
type DeployTrendPoint struct {
	Day     time.Time `json:"day"`
	Success int       `json:"success"`
	Failed  int       `json:"failed"`
}

// GetSiteSummary returns site counts grouped by status.
func (r *DashboardRepository) GetSiteSummary(ctx context.Context) (SiteSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT status, COUNT(*) AS cnt
		FROM sites
		GROUP BY status
	`)
	if err != nil {
		return SiteSummary{}, err
	}
	defer rows.Close()

	var s SiteSummary
	for rows.Next() {
		var status string
		var cnt int
		if err := rows.Scan(&status, &cnt); err != nil {
			continue
		}
		s.Total += cnt
		switch status {
		case "deployed", "ssl_active":
			s.Deployed += cnt
		case "deploy_failed":
			s.Failed += cnt
		case "deploying":
			s.Deploying += cnt
		default:
			s.Other += cnt
		}
	}
	return s, rows.Err()
}

// GetServerSummary returns server counts grouped by status.
func (r *DashboardRepository) GetServerSummary(ctx context.Context) (ServerSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT status, COUNT(*) AS cnt
		FROM servers
		GROUP BY status
	`)
	if err != nil {
		return ServerSummary{}, err
	}
	defer rows.Close()

	var s ServerSummary
	for rows.Next() {
		var status string
		var cnt int
		if err := rows.Scan(&status, &cnt); err != nil {
			continue
		}
		s.Total += cnt
		switch status {
		case "connected", "ready_for_deploy":
			s.Connected += cnt
		case "connection_failed", "provision_failed":
			s.Failed += cnt
		default:
			s.Other += cnt
		}
	}
	return s, rows.Err()
}

// GetUptimeSummary returns up/down counts based on the most recent check per site.
func (r *DashboardRepository) GetUptimeSummary(ctx context.Context) (UptimeSummary, error) {
	row := r.db.QueryRow(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (site_id) site_id, status
			FROM site_uptime_checks
			ORDER BY site_id, checked_at DESC
		),
		pct AS (
			SELECT
				site_id,
				ROUND(
					100.0 * COUNT(*) FILTER (WHERE status = 'up') / NULLIF(COUNT(*), 0),
					1
				) AS uptime_pct
			FROM site_uptime_checks
			WHERE checked_at > NOW() - INTERVAL '7 days'
			GROUP BY site_id
		)
		SELECT
			COUNT(*) FILTER (WHERE l.status = 'up')   AS sites_up,
			COUNT(*) FILTER (WHERE l.status = 'down') AS sites_down,
			COALESCE(AVG(p.uptime_pct), 100)::float8  AS avg_pct
		FROM latest l
		LEFT JOIN pct p ON p.site_id = l.site_id
	`)
	var s UptimeSummary
	err := row.Scan(&s.SitesUp, &s.SitesDown, &s.AvgPct)
	return s, err
}

// GetDeploySummary counts deploy task successes and failures.
func (r *DashboardRepository) GetDeploySummary(ctx context.Context) (DeploySummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			(started_at > NOW() - INTERVAL '24 hours') AS recent,
			status,
			COUNT(*) AS cnt
		FROM task_logs
		WHERE task_type IN ('deploy', 'rollback')
		GROUP BY recent, status
	`)
	if err != nil {
		return DeploySummary{}, err
	}
	defer rows.Close()

	var s DeploySummary
	for rows.Next() {
		var recent bool
		var status string
		var cnt int
		if err := rows.Scan(&recent, &status, &cnt); err != nil {
			continue
		}
		switch status {
		case "success":
			s.AllTimeSuccess += cnt
			if recent {
				s.Today24hSuccess += cnt
			}
		case "failed":
			s.AllTimeFailed += cnt
			if recent {
				s.Today24hFailed += cnt
			}
		}
	}
	return s, rows.Err()
}

// GetUptimeTrend returns per-hour uptime check counts for the last 24 hours.
func (r *DashboardRepository) GetUptimeTrend(ctx context.Context) ([]UptimeTrendPoint, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			date_trunc('hour', checked_at)             AS hour,
			COUNT(*) FILTER (WHERE status = 'up')      AS up_count,
			COUNT(*) FILTER (WHERE status = 'down')    AS down_count,
			COALESCE(AVG(latency_ms), 0)::float8       AS avg_latency
		FROM site_uptime_checks
		WHERE checked_at > NOW() - INTERVAL '24 hours'
		GROUP BY hour
		ORDER BY hour ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []UptimeTrendPoint
	for rows.Next() {
		var p UptimeTrendPoint
		if err := rows.Scan(&p.Hour, &p.UpCount, &p.DownCount, &p.AvgLatency); err != nil {
			continue
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// GetDeployTrend returns per-day deploy counts for the last 7 days.
func (r *DashboardRepository) GetDeployTrend(ctx context.Context) ([]DeployTrendPoint, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			date_trunc('day', started_at)                       AS day,
			COUNT(*) FILTER (WHERE status = 'success')          AS success,
			COUNT(*) FILTER (WHERE status = 'failed')           AS failed
		FROM task_logs
		WHERE task_type IN ('deploy', 'rollback')
		  AND started_at > NOW() - INTERVAL '7 days'
		GROUP BY day
		ORDER BY day ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []DeployTrendPoint
	for rows.Next() {
		var p DeployTrendPoint
		if err := rows.Scan(&p.Day, &p.Success, &p.Failed); err != nil {
			continue
		}
		points = append(points, p)
	}
	return points, rows.Err()
}
