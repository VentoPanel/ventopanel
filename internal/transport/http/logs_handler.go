package http

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/your-org/ventopanel/internal/filemanager"
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

// logShellescape wraps s in single quotes, escaping any existing single quotes.
func logShellescape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// ─── Handler ──────────────────────────────────────────────────────────────────

type LogsHandler struct {
	servers serverGetter
}

func NewLogsHandler(servers serverGetter) *LogsHandler {
	return &LogsHandler{servers: servers}
}

// Stream godoc
// GET /servers/:id/logs/stream
//
// Query params:
//
//	source  = journal | docker | file   (default: journal)
//	unit    = systemd unit name          (source=journal)
//	container = docker container name/id (source=docker)
//	path    = absolute file path         (source=file,  default: /var/log/syslog)
//	lines   = tail N lines               (default: 200)
func (h *LogsHandler) Stream(c *gin.Context) {
	if _, ok := c.Get(contextUserIDKey); !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	serverID := c.Param("id")
	source := c.DefaultQuery("source", "journal")
	lines := c.DefaultQuery("lines", "200")
	unit := c.Query("unit")
	container := c.Query("container")
	filePath := c.DefaultQuery("path", "/var/log/syslog")

	// Build the remote command.
	var cmd string
	switch source {
	case "journal":
		if unit == "" {
			unit = "sshd"
		}
		cmd = fmt.Sprintf(
			"journalctl -u %s -n %s -f --no-pager --output=short-iso 2>&1",
			logShellescape(unit), logShellescape(lines),
		)
	case "docker":
		if container == "" {
			c.JSON(http.StatusBadRequest, errorResponse{Error: "container param required"})
			return
		}
		cmd = fmt.Sprintf("docker logs -f --tail %s %s 2>&1", logShellescape(lines), logShellescape(container))
	case "file":
		cmd = fmt.Sprintf("tail -n %s -f %s 2>&1", logShellescape(lines), logShellescape(filePath))
	default:
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid source (journal|docker|file)"})
		return
	}

	srv, err := h.servers.GetByID(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "server not found"})
		return
	}

	_, sshCli, err := filemanager.GlobalPool.Get(serverID, filemanager.ServerDialConfig{
		Host:     srv.Host,
		Port:     srv.Port,
		User:     srv.SSHUser,
		Password: srv.SSHPassword,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: "SSH: " + err.Error()})
		return
	}

	// Open a dedicated SSH session for the streaming command.
	sess, err := sshCli.NewSession()
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: "SSH session: " + err.Error()})
		return
	}

	stdout, err := sess.StdoutPipe()
	if err != nil {
		sess.Close()
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	if err := sess.Start(cmd); err != nil {
		sess.Close()
		c.JSON(http.StatusBadGateway, errorResponse{Error: "start cmd: " + err.Error()})
		return
	}

	// Ensure session closes when client disconnects.
	ctx := c.Request.Context()
	go func() {
		<-ctx.Done()
		sess.Close()
	}()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		sess.Close()
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "streaming unsupported"})
		return
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			sess.Close()
			return
		default:
		}
		line := scanner.Text()
		// Escape newlines inside the SSE data field.
		line = strings.ReplaceAll(line, "\n", "\\n")
		fmt.Fprintf(c.Writer, "event: log\ndata: %s\n\n", line)
		flusher.Flush()
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
	}
	sess.Close()
}

// ListUnits godoc
// GET /servers/:id/logs/units
// Returns a list of active systemd service units on the remote server.
func (h *LogsHandler) ListUnits(c *gin.Context) {
	if _, ok := c.Get(contextUserIDKey); !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	serverID := c.Param("id")
	srv, err := h.servers.GetByID(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "server not found"})
		return
	}

	_, sshCli, err := filemanager.GlobalPool.Get(serverID, filemanager.ServerDialConfig{
		Host:     srv.Host,
		Port:     srv.Port,
		User:     srv.SSHUser,
		Password: srv.SSHPassword,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: "SSH: " + err.Error()})
		return
	}

	out, err := sshOutput(sshCli, `systemctl list-units --type=service --state=loaded --no-pager --no-legend 2>/dev/null | awk '{print $1}' | sed 's/\.service//' | head -100`)
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}

	units := []string{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			units = append(units, line)
		}
	}

	c.JSON(http.StatusOK, gin.H{"units": units})
}

// ListContainers godoc
// GET /servers/:id/logs/containers
// Returns a list of docker container names on the remote server.
func (h *LogsHandler) ListContainers(c *gin.Context) {
	if _, ok := c.Get(contextUserIDKey); !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	serverID := c.Param("id")
	srv, err := h.servers.GetByID(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "server not found"})
		return
	}

	_, sshCli, err := filemanager.GlobalPool.Get(serverID, filemanager.ServerDialConfig{
		Host:     srv.Host,
		Port:     srv.Port,
		User:     srv.SSHUser,
		Password: srv.SSHPassword,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: "SSH: " + err.Error()})
		return
	}

	out, err := sshOutput(sshCli, `docker ps --format '{{.Names}}' 2>/dev/null || echo ""`)
	if err != nil {
		// Docker may not be installed — return empty list gracefully.
		c.JSON(http.StatusOK, gin.H{"containers": []string{}})
		return
	}

	containers := []string{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			containers = append(containers, line)
		}
	}

	c.JSON(http.StatusOK, gin.H{"containers": containers})
}
