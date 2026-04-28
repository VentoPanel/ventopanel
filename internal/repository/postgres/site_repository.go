package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/your-org/ventopanel/internal/domain/site"
)

type SiteRepository struct {
	db *pgxpool.Pool
}

func NewSiteRepository(db *pgxpool.Pool) *SiteRepository {
	return &SiteRepository{db: db}
}

func (r *SiteRepository) Create(ctx context.Context, site *domain.Site) error {
	const query = `
		INSERT INTO sites (server_id, name, domain, runtime, repository_url, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	return r.db.QueryRow(ctx, query,
		site.ServerID,
		site.Name,
		site.Domain,
		site.Runtime,
		site.RepositoryURL,
		site.Status,
	).Scan(&site.ID)
}

func (r *SiteRepository) GetByID(ctx context.Context, id string) (*domain.Site, error) {
	const query = `
		SELECT id, server_id, name, domain, runtime, repository_url, status
		FROM sites
		WHERE id = $1
	`

	var site domain.Site
	err := r.db.QueryRow(ctx, query, id).Scan(
		&site.ID,
		&site.ServerID,
		&site.Name,
		&site.Domain,
		&site.Runtime,
		&site.RepositoryURL,
		&site.Status,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}

		return nil, err
	}

	return &site, nil
}

func (r *SiteRepository) List(ctx context.Context) ([]domain.Site, error) {
	const query = `
		SELECT id, server_id, name, domain, runtime, repository_url, status
		FROM sites
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sites := make([]domain.Site, 0)
	for rows.Next() {
		var site domain.Site
		if err := rows.Scan(
			&site.ID,
			&site.ServerID,
			&site.Name,
			&site.Domain,
			&site.Runtime,
			&site.RepositoryURL,
			&site.Status,
		); err != nil {
			return nil, err
		}

		sites = append(sites, site)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sites, nil
}

func (r *SiteRepository) Update(ctx context.Context, site *domain.Site) error {
	const query = `
		UPDATE sites
		SET server_id = $2, name = $3, domain = $4, runtime = $5, repository_url = $6, status = $7, updated_at = NOW()
		WHERE id = $1
	`

	tag, err := r.db.Exec(ctx, query,
		site.ID,
		site.ServerID,
		site.Name,
		site.Domain,
		site.Runtime,
		site.RepositoryURL,
		site.Status,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *SiteRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM sites WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}
