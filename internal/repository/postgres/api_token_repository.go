package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// APIToken represents a personal access token for programmatic API access.
type APIToken struct {
	ID         string
	UserID     string
	Name       string
	TokenHash  string
	LastUsedAt *time.Time
	CreatedAt  time.Time
}

type APITokenRepository struct {
	db *pgxpool.Pool
}

func NewAPITokenRepository(db *pgxpool.Pool) *APITokenRepository {
	return &APITokenRepository{db: db}
}

// GenerateToken creates a new random token in the format "vp_<64 hex chars>".
// Returns the plaintext token and its SHA-256 hash.
func GenerateToken() (plaintext, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	plaintext = "vp_" + hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(plaintext))
	hash = hex.EncodeToString(sum[:])
	return
}

// HashToken returns the SHA-256 hex digest of a plaintext token.
func HashToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// Create stores a new token. The tokenHash must already be computed by the caller.
func (r *APITokenRepository) Create(ctx context.Context, userID, name, tokenHash string) (*APIToken, error) {
	t := &APIToken{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO api_tokens (user_id, name, token_hash)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, name, token_hash, last_used_at, created_at`,
		userID, name, tokenHash,
	).Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.LastUsedAt, &t.CreatedAt)
	return t, err
}

// GetByHash looks up a token by its hash, returning nil if not found.
func (r *APITokenRepository) GetByHash(ctx context.Context, hash string) (*APIToken, error) {
	t := &APIToken{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, name, token_hash, last_used_at, created_at
		FROM api_tokens WHERE token_hash = $1`, hash,
	).Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.LastUsedAt, &t.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return t, nil
}

// ListByUser returns all tokens for a user, ordered by creation time.
func (r *APITokenRepository) ListByUser(ctx context.Context, userID string) ([]APIToken, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, name, token_hash, last_used_at, created_at
		FROM api_tokens WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []APIToken
	for rows.Next() {
		var t APIToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.LastUsedAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// TouchLastUsed updates last_used_at for a token (best-effort, async-safe).
func (r *APITokenRepository) TouchLastUsed(ctx context.Context, id string) {
	_, _ = r.db.Exec(ctx,
		`UPDATE api_tokens SET last_used_at = NOW() WHERE id = $1`, id)
}

// Delete removes a token by id, restricted to the owning user.
func (r *APITokenRepository) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM api_tokens WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}
