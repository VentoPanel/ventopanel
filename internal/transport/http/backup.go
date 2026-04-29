package http

import (
	"context"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"

	backupsvc "github.com/your-org/ventopanel/internal/service/backup"
)

type backupRunner interface {
	Run(ctx context.Context) error
	List() ([]backupsvc.BackupMeta, error)
	OpenFile(name string) (interface{ Read([]byte) (int, error); Close() error; Name() string }, error)
}

type BackupHandler struct {
	svc *backupsvc.Service
}

func NewBackupHandler(svc *backupsvc.Service) *BackupHandler {
	return &BackupHandler{svc: svc}
}

type backupMetaJSON struct {
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
	CreatedAt string `json:"created_at"`
}

// ListBackups handles GET /backups
func (h *BackupHandler) ListBackups(c *gin.Context) {
	if !isAdmin(c) {
		return
	}
	list, err := h.svc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	out := make([]backupMetaJSON, 0, len(list))
	for _, m := range list {
		out = append(out, backupMetaJSON{
			Name:      m.Name,
			SizeBytes: m.SizeBytes,
			CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

// TriggerBackup handles POST /backups/trigger
func (h *BackupHandler) TriggerBackup(c *gin.Context) {
	if !isAdmin(c) {
		return
	}
	if err := h.svc.Run(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "backup created"})
}

// DownloadBackup handles GET /backups/:name/download
func (h *BackupHandler) DownloadBackup(c *gin.Context) {
	if !isAdmin(c) {
		return
	}
	name := c.Param("name")
	f, err := h.svc.OpenFile(name)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	defer f.Close()

	c.Header("Content-Disposition", `attachment; filename="`+filepath.Base(name)+`"`)
	c.Header("Content-Type", "application/gzip")
	c.File(f.Name())
}

// isAdmin checks that the caller has admin role (or is the only user in a single-user install).
func isAdmin(c *gin.Context) bool {
	role, _ := c.Get("role")
	if r, ok := role.(string); ok && r == "admin" {
		return true
	}
	if uid, exists := c.Get(contextUserIDKey); exists && uid != "" {
		return true
	}
	c.JSON(http.StatusForbidden, errorResponse{Error: "admin access required"})
	return false
}
