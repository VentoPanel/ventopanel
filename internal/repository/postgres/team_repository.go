package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TeamRepository struct {
	db *pgxpool.Pool
}

func NewTeamRepository(db *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) HasSiteAccess(ctx context.Context, teamID, siteID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM team_site_access
			WHERE team_id = $1 AND site_id = $2
		)
	`

	var allowed bool
	if err := r.db.QueryRow(ctx, query, teamID, siteID).Scan(&allowed); err != nil {
		return false, err
	}
	return allowed, nil
}

func (r *TeamRepository) GetSiteRole(ctx context.Context, teamID, siteID string) (string, error) {
	const query = `
		SELECT role
		FROM team_site_access
		WHERE team_id = $1 AND site_id = $2
		LIMIT 1
	`

	var role string
	if err := r.db.QueryRow(ctx, query, teamID, siteID).Scan(&role); err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(role), nil
}

func (r *TeamRepository) HasServerAccess(ctx context.Context, teamID, serverID string) (bool, error) {
	// Check direct server grant first, then fall back to site-based access.
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM team_server_access
			WHERE team_id = $1 AND server_id = $2
		) OR EXISTS (
			SELECT 1
			FROM team_site_access tsa
			INNER JOIN sites s ON s.id = tsa.site_id
			WHERE tsa.team_id = $1 AND s.server_id = $2
		)
	`

	var allowed bool
	if err := r.db.QueryRow(ctx, query, teamID, serverID).Scan(&allowed); err != nil {
		return false, err
	}
	return allowed, nil
}

func (r *TeamRepository) GetServerRole(ctx context.Context, teamID, serverID string) (string, error) {
	// Direct server grant takes precedence; fall back to highest site-based role.
	const query = `
		SELECT role FROM (
			SELECT role,
				CASE LOWER(role)
					WHEN 'owner' THEN 3
					WHEN 'admin' THEN 2
					WHEN 'viewer' THEN 1
					ELSE 0
				END AS rank
			FROM team_server_access
			WHERE team_id = $1 AND server_id = $2
			UNION ALL
			SELECT tsa.role,
				CASE LOWER(tsa.role)
					WHEN 'owner' THEN 3
					WHEN 'admin' THEN 2
					WHEN 'viewer' THEN 1
					ELSE 0
				END AS rank
			FROM team_site_access tsa
			INNER JOIN sites s ON s.id = tsa.site_id
			WHERE tsa.team_id = $1 AND s.server_id = $2
		) combined
		ORDER BY rank DESC
		LIMIT 1
	`

	var role string
	if err := r.db.QueryRow(ctx, query, teamID, serverID).Scan(&role); err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(role), nil
}

func (r *TeamRepository) GrantServerAccess(ctx context.Context, teamID, serverID, role string) error {
	const query = `
		INSERT INTO team_server_access (team_id, server_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (team_id, server_id) DO UPDATE SET role = EXCLUDED.role
	`
	_, err := r.db.Exec(ctx, query, teamID, serverID, role)
	return err
}

func (r *TeamRepository) GrantSiteAccess(ctx context.Context, teamID, siteID, role string) error {
	const query = `
		INSERT INTO team_site_access (team_id, site_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (team_id, site_id) DO UPDATE SET role = EXCLUDED.role
	`
	_, err := r.db.Exec(ctx, query, teamID, siteID, role)
	return err
}
