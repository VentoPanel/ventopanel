package http

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
// Uses a poll-based approach (same pattern as the metrics handler) rather than
// long-lived SSH pipe streaming. Every pollInterval the handler runs a one-shot
// SSH command that returns new log lines since the last poll, and sends them as
// SSE events. This is reliable across all proxy/buffering setups.
//
// Query params:
//
//	source    = journal | docker | file   (default: file)
//	unit      = systemd unit name          (source=journal)
//	container = docker container name/id   (source=docker)
//	path      = absolute file path         (source=file, default: /var/log/syslog)
//	lines     = initial tail N lines       (default: 200)
func (h *LogsHandler) Stream(c *gin.Context) {
	if _, ok := c.Get(contextUserIDKey); !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	serverID  := c.Param("id")
	source    := c.DefaultQuery("source", "file")
	lines     := c.DefaultQuery("lines", "200")
	unit      := c.Query("unit")
	container := c.Query("container")
	filePath  := c.DefaultQuery("path", "/var/log/syslog")

	srv, err := h.servers.GetByID(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "server not found"})
		return
	}

	dialCfg := filemanager.ServerDialConfig{
		Host:     srv.Host,
		Port:     srv.Port,
		User:     srv.SSHUser,
		Password: srv.SSHPassword,
	}

	// ── Open SSE response immediately (before any SSH work) ─────────────────
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "streaming unsupported"})
		return
	}

	// Send a visible "log" event immediately so the frontend sees data
	// the moment the HTTP response is committed — confirms SSE transport works.
	fmt.Fprintf(c.Writer, "event: log\ndata: [ventopanel] connecting to %s…\n\n", srv.Host)
	flusher.Flush()

	// ── Get SSH client ───────────────────────────────────────────────────────
	_, sshCli, err := filemanager.GlobalPool.Get(serverID, dialCfg)
	if err != nil {
		fmt.Fprintf(c.Writer, "event: error\ndata: SSH: %s\n\n", err.Error())
		flusher.Flush()
		return
	}

	fmt.Fprintf(c.Writer, "event: log\ndata: [ventopanel] SSH ok — fetching %s lines via %s\n\n", lines, source)
	flusher.Flush()

	ctx := c.Request.Context()

	// ── Helper: run one-shot SSH command, always return output ───────────────
	// Unlike sshOutput(), this does NOT discard stdout when the exit code is
	// non-zero (journalctl can exit 1 while still printing valid log lines).
	run := func(cmd string) []string {
		sess, serr := sshCli.NewSession()
		if serr != nil {
			return nil
		}
		defer sess.Close()

		var buf bytes.Buffer
		sess.Stdout = &buf
		sess.Stderr = &buf
		_ = sess.Run(cmd) // ignore exit code

		var result []string
		for _, l := range strings.Split(buf.String(), "\n") {
			l = strings.TrimRight(l, "\r")
			// Skip empty lines and journalctl meta-lines (e.g. "-- No entries --",
			// "-- Boot ... --", "-- Reboot --").
			if l == "" || (strings.HasPrefix(l, "-- ") && strings.HasSuffix(l, " --")) {
				continue
			}
			result = append(result, l)
		}
		return result
	}

	// ── Send initial batch ──────────────────────────────────────────────────
	var initCmd string
	switch source {
	case "journal":
		if unit == "" || unit == "_all" {
			initCmd = fmt.Sprintf("journalctl -b -n %s --no-pager --output=cat 2>&1 || journalctl -n %s --no-pager --output=cat 2>&1", logShellescape(lines), logShellescape(lines))
		} else {
			initCmd = fmt.Sprintf("journalctl -b -u %s -n %s --no-pager --output=cat 2>&1 || journalctl -u %s -n %s --no-pager --output=cat 2>&1", logShellescape(unit), logShellescape(lines), logShellescape(unit), logShellescape(lines))
		}
	case "docker":
		if container == "" {
			fmt.Fprintf(c.Writer, "event: error\ndata: container param required\n\n")
			flusher.Flush()
			return
		}
		initCmd = fmt.Sprintf("docker logs --tail %s %s 2>&1", logShellescape(lines), logShellescape(container))
	default: // file
		initCmd = fmt.Sprintf("tail -n %s %s 2>&1", logShellescape(lines), logShellescape(filePath))
	}

	initLines := run(initCmd)
	for _, l := range initLines {
		fmt.Fprintf(c.Writer, "event: log\ndata: %s\n\n", l)
	}
	if len(initLines) == 0 {
		src := source
		if source == "file" {
			src = filePath
		}
		fmt.Fprintf(c.Writer, "event: log\ndata: [ventopanel] no log entries found for: %s\n\n", src)
	}
	flusher.Flush()

	// For file source: track line count so each poll only emits new lines.
	var fileLineCount int64
	if source == "file" {
		if cl := run(fmt.Sprintf("wc -l < %s 2>/dev/null", logShellescape(filePath))); len(cl) > 0 {
			n, _ := strconv.ParseInt(strings.TrimSpace(cl[0]), 10, 64)
			fileLineCount = n
		}
	}

	// ── Poll for new lines every 2 seconds ──────────────────────────────────
	since := time.Now()
	pollTick  := time.NewTicker(2 * time.Second)
	keepalive := time.NewTicker(30 * time.Second)
	defer pollTick.Stop()
	defer keepalive.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-keepalive.C:
			fmt.Fprintf(c.Writer, ": ping\n\n")
			flusher.Flush()

		case <-pollTick.C:
			var pollCmd string
			sinceStr := since.UTC().Format("2006-01-02 15:04:05")
			switch source {
			case "journal":
				if unit == "" || unit == "_all" {
					pollCmd = fmt.Sprintf(
						"journalctl -b --since=%s --no-pager --output=cat 2>&1",
						logShellescape(sinceStr),
					)
				} else {
					pollCmd = fmt.Sprintf(
						"journalctl -b -u %s --since=%s --no-pager --output=cat 2>&1",
						logShellescape(unit), logShellescape(sinceStr),
					)
				}
			case "docker":
				pollCmd = fmt.Sprintf(
					"docker logs --since=%s %s 2>&1",
					logShellescape(since.UTC().Format(time.RFC3339)), logShellescape(container),
				)
			default: // file
				// awk 'NR>N' reads only lines after the last known count.
				pollCmd = fmt.Sprintf("awk 'NR>%d' %s 2>/dev/null", fileLineCount, logShellescape(filePath))
			}

			newLines := run(pollCmd)
			since = time.Now()
			if len(newLines) > 0 {
				if source == "file" {
					fileLineCount += int64(len(newLines))
				}
				for _, l := range newLines {
					fmt.Fprintf(c.Writer, "event: log\ndata: %s\n\n", l)
				}
				flusher.Flush()
			}

			// Refresh SSH client from pool (reconnects if connection died).
			if _, newSSH, rerr := filemanager.GlobalPool.Get(serverID, dialCfg); rerr == nil {
				sshCli = newSSH
			}
		}
	}
}

// ListUnits godoc
// GET /servers/:id/logs/units
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
		Host: srv.Host, Port: srv.Port, User: srv.SSHUser, Password: srv.SSHPassword,
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
		if line = strings.TrimSpace(line); line != "" {
			units = append(units, line)
		}
	}
	c.JSON(http.StatusOK, gin.H{"units": units})
}

// ListContainers godoc
// GET /servers/:id/logs/containers
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
		Host: srv.Host, Port: srv.Port, User: srv.SSHUser, Password: srv.SSHPassword,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: "SSH: " + err.Error()})
		return
	}

	out, err := sshOutput(sshCli, `docker ps --format '{{.Names}}' 2>/dev/null || echo ""`)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"containers": []string{}})
		return
	}

	containers := []string{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			containers = append(containers, line)
		}
	}
	c.JSON(http.StatusOK, gin.H{"containers": containers})
}
