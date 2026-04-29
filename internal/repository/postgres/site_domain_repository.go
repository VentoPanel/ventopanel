package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SiteDomainRepository manages extra alias domains attached to a site.
type SiteDomainRepository struct {
	db *pgxpool.Pool
}

func NewSiteDomainRepository(db *pgxpool.Pool) *SiteDomainRepository {
	return &SiteDomainRepository{db: db}
}

// List returns all alias domains for a site, ordered by creation time.
func (r *SiteDomainRepository) List(ctx context.Context, siteID string) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT domain FROM site_domains WHERE site_id=$1 ORDER BY created_at`,
		siteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// Add attaches an alias domain to a site.
// Returns an error if the domain is already used by any site.
func (r *SiteDomainRepository) Add(ctx context.Context, siteID, domain string) error {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return fmt.Errorf("domain must not be empty")
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO site_domains (site_id, domain) VALUES ($1, $2)`,
		siteID, domain,
	)
	return err
}

// Remove detaches an alias domain from a site.
func (r *SiteDomainRepository) Remove(ctx context.Context, siteID, domain string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM site_domains WHERE site_id=$1 AND domain=$2`,
		siteID, domain,
	)
	return err
}
