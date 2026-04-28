package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/your-org/ventopanel/internal/domain/server"
)

type ServerRepository struct {
	db        *pgxpool.Pool
	encryptor passwordCipher
}

type passwordCipher interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(value string) (string, error)
}

func NewServerRepository(db *pgxpool.Pool, encryptor passwordCipher) *ServerRepository {
	return &ServerRepository{
		db:        db,
		encryptor: encryptor,
	}
}

func (r *ServerRepository) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}

func (r *ServerRepository) Create(ctx context.Context, server *domain.Server) error {
	encryptedPassword, err := r.encryptor.Encrypt(server.SSHPassword)
	if err != nil {
		return fmt.Errorf("encrypt ssh password: %w", err)
	}

	const query = `
		INSERT INTO servers (name, host, port, provider, status, ssh_user, ssh_password, last_renew_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	return r.db.QueryRow(ctx, query,
		server.Name,
		server.Host,
		server.Port,
		server.Provider,
		server.Status,
		server.SSHUser,
		encryptedPassword,
		server.LastRenewStatus,
	).Scan(&server.ID)
}

func (r *ServerRepository) GetByID(ctx context.Context, id string) (*domain.Server, error) {
	const query = `
		SELECT id, name, host, port, provider, status, ssh_user, ssh_password, last_renew_at, last_renew_status
		FROM servers
		WHERE id = $1
	`

	var server domain.Server
	err := r.db.QueryRow(ctx, query, id).Scan(
		&server.ID,
		&server.Name,
		&server.Host,
		&server.Port,
		&server.Provider,
		&server.Status,
		&server.SSHUser,
		&server.SSHPassword,
		&server.LastRenewAt,
		&server.LastRenewStatus,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}

		return nil, err
	}

	server.SSHPassword, err = r.encryptor.Decrypt(server.SSHPassword)
	if err != nil {
		return nil, fmt.Errorf("decrypt ssh password: %w", err)
	}

	return &server, nil
}

func (r *ServerRepository) List(ctx context.Context) ([]domain.Server, error) {
	const query = `
		SELECT id, name, host, port, provider, status, ssh_user, ssh_password, last_renew_at, last_renew_status
		FROM servers
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	servers := make([]domain.Server, 0)
	for rows.Next() {
		var server domain.Server
		if err := rows.Scan(
			&server.ID,
			&server.Name,
			&server.Host,
			&server.Port,
			&server.Provider,
			&server.Status,
			&server.SSHUser,
			&server.SSHPassword,
			&server.LastRenewAt,
			&server.LastRenewStatus,
		); err != nil {
			return nil, err
		}

		server.SSHPassword, err = r.encryptor.Decrypt(server.SSHPassword)
		if err != nil {
			return nil, fmt.Errorf("decrypt ssh password: %w", err)
		}

		servers = append(servers, server)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return servers, nil
}

func (r *ServerRepository) Update(ctx context.Context, server *domain.Server) error {
	encryptedPassword, err := r.encryptor.Encrypt(server.SSHPassword)
	if err != nil {
		return fmt.Errorf("encrypt ssh password: %w", err)
	}

	const query = `
		UPDATE servers
		SET name = $2, host = $3, port = $4, provider = $5, status = $6, ssh_user = $7, ssh_password = $8, last_renew_at = $9, last_renew_status = $10, updated_at = NOW()
		WHERE id = $1
	`

	tag, err := r.db.Exec(ctx, query,
		server.ID,
		server.Name,
		server.Host,
		server.Port,
		server.Provider,
		server.Status,
		server.SSHUser,
		encryptedPassword,
		server.LastRenewAt,
		server.LastRenewStatus,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *ServerRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM servers WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}
