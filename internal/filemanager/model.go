package filemanager

import (
	"errors"
	"time"
)

// Sentinel errors returned by the service.
var (
	ErrForbidden = errors.New("path is outside the allowed root")
	ErrNotFound  = errors.New("path not found")
	ErrIsDir     = errors.New("path is a directory")
	ErrNotDir    = errors.New("path is not a directory")
)

// FileItem is the JSON-serialisable representation of a single filesystem entry.
type FileItem struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`    // relative to root
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	Ext     string    `json:"ext"` // empty for directories
}
