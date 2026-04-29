package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// tables included in every backup, in safe dependency order.
var backupTables = []string{
	"users",
	"teams",
	"team_members",
	"servers",
	"sites",
	"site_env_vars",
	"site_uptime_checks",
	"task_logs",
	"status_events",
	"app_settings",
}

// BackupMeta describes a single backup archive.
type BackupMeta struct {
	Name      string    `json:"name"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

type notifier interface {
	NotifyAll(ctx context.Context, message string) error
}

// Service creates and manages database backup archives.
type Service struct {
	db        *pgxpool.Pool
	dir       string
	keepCount int
	notifier  notifier
}

func NewService(db *pgxpool.Pool, dir string, keepCount int, notifier notifier) *Service {
	if keepCount <= 0 {
		keepCount = 7
	}
	return &Service{db: db, dir: dir, keepCount: keepCount, notifier: notifier}
}

// Run creates one .tar.gz backup and prunes excess archives.
func (s *Service) Run(ctx context.Context) error {
	if err := os.MkdirAll(s.dir, 0o750); err != nil {
		return fmt.Errorf("backup: mkdir %s: %w", s.dir, err)
	}

	name := "ventopanel_" + time.Now().UTC().Format("2006-01-02_15-04-05") + ".tar.gz"
	path := filepath.Join(s.dir, name)

	if err := s.writeTarGz(ctx, path); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("backup: %w", err)
	}

	_ = s.prune()
	return nil
}

// List returns backup metadata sorted newest first.
func (s *Service) List() ([]BackupMeta, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var out []BackupMeta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tar.gz") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, BackupMeta{
			Name:      e.Name(),
			SizeBytes: info.Size(),
			CreatedAt: info.ModTime().UTC(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

// OpenFile returns an open *os.File for the named archive. Caller must close it.
func (s *Service) OpenFile(name string) (*os.File, error) {
	base := filepath.Base(name)
	if !strings.HasSuffix(base, ".tar.gz") || strings.ContainsAny(base, "/\\") {
		return nil, fmt.Errorf("invalid backup name")
	}
	return os.Open(filepath.Join(s.dir, base))
}

// writeTarGz streams each table as a CSV entry into a .tar.gz archive.
func (s *Service) writeTarGz(ctx context.Context, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	pgConn := conn.Conn().PgConn()
	for _, table := range backupTables {
		if err := copyTableToTar(ctx, pgConn, tw, table); err != nil {
			return fmt.Errorf("table %s: %w", table, err)
		}
	}
	return nil
}

// copyTableToTar buffers one table's CSV COPY output then writes it into the tar.
func copyTableToTar(ctx context.Context, pgConn *pgconn.PgConn, tw *tar.Writer, table string) error {
	tmp, err := os.CreateTemp("", "vpbk-"+table+"-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	if _, err := pgConn.CopyTo(ctx, tmp, "COPY "+table+" TO STDOUT CSV HEADER"); err != nil {
		return err
	}

	size, err := tmp.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return err
	}

	if err := tw.WriteHeader(&tar.Header{
		Name:    table + ".csv",
		Size:    size,
		Mode:    0o644,
		ModTime: time.Now().UTC(),
	}); err != nil {
		return err
	}

	_, err = io.Copy(tw, tmp)
	return err
}

// prune removes oldest archives beyond keepCount.
func (s *Service) prune() error {
	list, err := s.List()
	if err != nil || len(list) <= s.keepCount {
		return err
	}
	for _, old := range list[s.keepCount:] {
		_ = os.Remove(filepath.Join(s.dir, old.Name))
	}
	return nil
}
