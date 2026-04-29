package filemanager

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/afero"
)

// Service provides jailed filesystem operations backed by an afero.Fs.
// All paths supplied by callers are relative to RootPath; any attempt to
// escape the root via ".." or absolute paths is rejected with ErrForbidden.
type Service struct {
	fs       afero.Fs
	RootPath string // absolute path on the host OS
}

// NewService creates a new Service.
// rootPath is the absolute directory that all operations are restricted to.
func NewService(rootPath string) *Service {
	return &Service{
		fs:       afero.NewBasePathFs(afero.NewOsFs(), rootPath),
		RootPath: rootPath,
	}
}

// safePath validates that a caller-supplied relative path does not escape the
// root after cleaning. It returns the cleaned relative path or ErrForbidden.
func safePath(rel string) (string, error) {
	// Normalise: strip leading slash so it stays relative.
	rel = filepath.ToSlash(filepath.Clean("/" + rel))
	// After cleaning, the path must still be rooted (starts with /).
	// afero.BasePathFs will prepend the root, so we just need to make sure
	// there are no ".." segments left.
	if strings.Contains(rel, "..") {
		return "", ErrForbidden
	}
	return rel, nil
}

// ListDir returns the contents of the directory at relPath.
func (s *Service) ListDir(relPath string) ([]FileItem, error) {
	clean, err := safePath(relPath)
	if err != nil {
		return nil, err
	}

	entries, err := afero.ReadDir(s.fs, clean)
	if err != nil {
		return nil, mapErr(err)
	}

	items := make([]FileItem, 0, len(entries))
	for _, e := range entries {
		itemPath := filepath.ToSlash(filepath.Join(clean, e.Name()))
		ext := ""
		if !e.IsDir() {
			ext = strings.ToLower(filepath.Ext(e.Name()))
		}
		items = append(items, FileItem{
			Name:    e.Name(),
			Path:    itemPath,
			IsDir:   e.IsDir(),
			Size:    e.Size(),
			ModTime: e.ModTime(),
			Ext:     ext,
		})
	}

	// Directories first, then files; both sorted alphabetically.
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return items[i].Name < items[j].Name
	})
	return items, nil
}

// ReadFile returns the contents of the file at relPath.
func (s *Service) ReadFile(relPath string) ([]byte, error) {
	clean, err := safePath(relPath)
	if err != nil {
		return nil, err
	}
	data, err := afero.ReadFile(s.fs, clean)
	return data, mapErr(err)
}

// WriteFile writes content to the file at relPath, creating it if necessary.
func (s *Service) WriteFile(relPath string, content []byte) error {
	clean, err := safePath(relPath)
	if err != nil {
		return err
	}
	// Ensure parent directory exists.
	if mkErr := s.fs.MkdirAll(filepath.Dir(clean), 0755); mkErr != nil {
		return mkErr
	}
	return afero.WriteFile(s.fs, clean, content, 0644)
}

// Delete removes a file or directory (recursively) at relPath.
func (s *Service) Delete(relPath string) error {
	clean, err := safePath(relPath)
	if err != nil {
		return err
	}
	if clean == "/" {
		return ErrForbidden // never delete the root
	}
	return mapErr(s.fs.RemoveAll(clean))
}

// CreateDir creates the directory tree at relPath.
func (s *Service) CreateDir(relPath string) error {
	clean, err := safePath(relPath)
	if err != nil {
		return err
	}
	return mapErr(s.fs.MkdirAll(clean, 0755))
}

// Rename moves/renames oldPath to newPath (both relative).
func (s *Service) Rename(oldPath, newPath string) error {
	cleanOld, err := safePath(oldPath)
	if err != nil {
		return err
	}
	cleanNew, err := safePath(newPath)
	if err != nil {
		return err
	}
	return mapErr(s.fs.Rename(cleanOld, cleanNew))
}

// Upload streams reader into the file at relPath using io.Copy.
func (s *Service) Upload(relPath string, r io.Reader) (int64, error) {
	clean, err := safePath(relPath)
	if err != nil {
		return 0, err
	}
	if mkErr := s.fs.MkdirAll(filepath.Dir(clean), 0755); mkErr != nil {
		return 0, mkErr
	}
	f, err := s.fs.Create(clean)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(f, r)
}

// Download opens the file at relPath and returns a ReadCloser + its size.
func (s *Service) Download(relPath string) (afero.File, int64, error) {
	clean, err := safePath(relPath)
	if err != nil {
		return nil, 0, err
	}
	info, err := s.fs.Stat(clean)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	if info.IsDir() {
		return nil, 0, ErrIsDir
	}
	f, err := s.fs.Open(clean)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	return f, info.Size(), nil
}

// mapErr converts afero/os errors to domain sentinel errors where applicable.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "no such file") || strings.Contains(msg, "not found") {
		return fmt.Errorf("%w: %s", ErrNotFound, err)
	}
	return err
}
