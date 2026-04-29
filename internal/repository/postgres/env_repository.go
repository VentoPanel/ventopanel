package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EnvVar is a decrypted key-value pair belonging to a site.
type EnvVar struct {
	ID        string
	SiteID    string
	Key       string
	Value     string // plaintext; never stored directly
	CreatedAt time.Time
	UpdatedAt time.Time
}

type EnvRepository struct {
	db        *pgxpool.Pool
	encryptor passwordCipher // reuse the same interface defined in server_repository.go
}

func NewEnvRepository(db *pgxpool.Pool, encryptor passwordCipher) *EnvRepository {
	return &EnvRepository{db: db, encryptor: encryptor}
}

// ListBySiteID returns all env vars for a site with values decrypted.
func (r *EnvRepository) ListBySiteID(ctx context.Context, siteID string) ([]EnvVar, error) {
	const q = `
		SELECT id, site_id, key, value_enc, created_at, updated_at
		FROM site_env_vars
		WHERE site_id = $1
		ORDER BY key
	`
	rows, err := r.db.Query(ctx, q, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []EnvVar
	for rows.Next() {
		var v EnvVar
		var encVal string
		if err := rows.Scan(&v.ID, &v.SiteID, &v.Key, &encVal, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		plain, err := r.encryptor.Decrypt(encVal)
		if err != nil {
			return nil, err
		}
		v.Value = plain
		out = append(out, v)
	}
	return out, rows.Err()
}

// Upsert inserts or updates an env var (keyed by site_id + key).
func (r *EnvRepository) Upsert(ctx context.Context, siteID, key, value string) error {
	enc, err := r.encryptor.Encrypt(value)
	if err != nil {
		return err
	}
	const q = `
		INSERT INTO site_env_vars (site_id, key, value_enc, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (site_id, key)
		DO UPDATE SET value_enc = EXCLUDED.value_enc, updated_at = NOW()
	`
	_, err = r.db.Exec(ctx, q, siteID, key, enc)
	return err
}

// Delete removes a single env var by key.
func (r *EnvRepository) Delete(ctx context.Context, siteID, key string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM site_env_vars WHERE site_id=$1 AND key=$2`, siteID, key)
	return err
}
