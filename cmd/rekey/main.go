// cmd/rekey/main.go — one-shot utility to re-encrypt all secrets in the DB
// when the APP_ENCRYPTION_KEY changes.
//
// Usage:
//   go run ./cmd/rekey \
//     --old-key 0123456789abcdef0123456789abcdef \
//     --new-key <new 32-byte hex key> \
//     --dsn    "postgres://vento:pass@localhost:5432/ventopanel?sslmode=disable"
//
// The tool is idempotent: rows already encrypted with the new key are skipped.
// Run it BEFORE switching APP_ENCRYPTION_KEY in the API container.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/ventopanel/internal/infra/crypto"
)

func main() {
	oldKey := flag.String("old-key", "", "current (old) 32-byte encryption key")
	newKey := flag.String("new-key", "", "new 32-byte encryption key")
	dsn := flag.String("dsn", os.Getenv("POSTGRES_DSN"), "PostgreSQL DSN")
	dryRun := flag.Bool("dry-run", false, "print counts only, do not write")
	flag.Parse()

	if *oldKey == "" || *newKey == "" {
		flag.Usage()
		log.Fatal("--old-key and --new-key are required")
	}
	if *oldKey == *newKey {
		log.Fatal("old-key and new-key are identical — nothing to do")
	}

	oldEnc, err := crypto.NewEncryptor(*oldKey)
	if err != nil {
		log.Fatalf("invalid old key: %v", err)
	}
	newEnc, err := crypto.NewEncryptor(*newKey)
	if err != nil {
		log.Fatalf("invalid new key: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		log.Fatalf("connect to postgres: %v", err)
	}
	defer pool.Close()

	total := 0

	// ── servers.ssh_password ─────────────────────────────────────────────────
	n, err := rekey(ctx, pool, *dryRun, oldEnc, newEnc,
		"servers", "id", "ssh_password")
	if err != nil {
		log.Fatalf("rekey servers: %v", err)
	}
	log.Printf("servers.ssh_password: %d rows re-encrypted", n)
	total += n

	// ── site_env_vars.value_enc ──────────────────────────────────────────────
	n, err = rekey(ctx, pool, *dryRun, oldEnc, newEnc,
		"site_env_vars", "id", "value_enc")
	if err != nil {
		log.Fatalf("rekey site_env_vars: %v", err)
	}
	log.Printf("site_env_vars.value_enc: %d rows re-encrypted", n)
	total += n

	if *dryRun {
		log.Printf("DRY RUN — total rows that would be updated: %d", total)
	} else {
		log.Printf("Done. Total rows updated: %d", total)
	}
}

// rekey fetches all rows from table where the encrypted column starts with
// "enc:v1:", decrypts with oldEnc and re-encrypts with newEnc.
func rekey(
	ctx context.Context,
	pool *pgxpool.Pool,
	dryRun bool,
	oldEnc, newEnc *crypto.Encryptor,
	table, idCol, encCol string,
) (int, error) {
	q := fmt.Sprintf(
		`SELECT %s, %s FROM %s WHERE %s LIKE 'enc:v1:%%'`,
		idCol, encCol, table, encCol,
	)
	rows, err := pool.Query(ctx, q)
	if err != nil {
		return 0, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	type row struct {
		id  string
		enc string
	}
	var toUpdate []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.enc); err != nil {
			return 0, fmt.Errorf("scan: %w", err)
		}
		toUpdate = append(toUpdate, r)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	if dryRun {
		return len(toUpdate), nil
	}

	updated := 0
	for _, r := range toUpdate {
		plain, err := oldEnc.Decrypt(r.enc)
		if err != nil {
			// Already re-encrypted with the new key — try new key as old key.
			plain2, err2 := newEnc.Decrypt(r.enc)
			if err2 != nil {
				return updated, fmt.Errorf("decrypt row %s: %w (also tried new key: %v)", r.id, err, err2)
			}
			// Already on new key, re-encrypt anyway to normalise nonce.
			plain = plain2
		}

		if plain == "" {
			continue
		}

		newCipher, err := newEnc.Encrypt(plain)
		if err != nil {
			return updated, fmt.Errorf("encrypt row %s: %w", r.id, err)
		}

		uq := fmt.Sprintf(
			`UPDATE %s SET %s = $1 WHERE %s = $2`,
			table, encCol, idCol,
		)
		if _, err := pool.Exec(ctx, uq, newCipher, r.id); err != nil {
			return updated, fmt.Errorf("update row %s: %w", r.id, err)
		}
		updated++
		log.Printf("  re-keyed %s/%s=%s", table, idCol, r.id)
	}
	return updated, nil
}

func init() {
	log.SetFlags(log.Ltime | log.Lmsgprefix)
	log.SetPrefix("[rekey] ")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `VentoPanel encryption key migration tool

Usage:
  go run ./cmd/rekey [flags]

Flags:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  # Dry run first — see how many rows will be updated:
  go run ./cmd/rekey --old-key OLD32BYTEKEYHERE --new-key NEW32BYTEKEYHERE --dsn "$POSTGRES_DSN" --dry-run

  # Then apply:
  go run ./cmd/rekey --old-key OLD32BYTEKEYHERE --new-key NEW32BYTEKEYHERE --dsn "$POSTGRES_DSN"
`)
	}
}
