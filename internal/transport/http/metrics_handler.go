package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	gossh "golang.org/x/crypto/ssh"

	"github.com/your-org/ventopanel/internal/filemanager"
)

// ─── Data model ───────────────────────────────────────────────────────────────

// MetricsSnapshot is one point of server metrics collected per poll interval.
type MetricsSnapshot struct {
	Timestamp  int64   `json:"ts"`
	CPUPct     float64 `json:"cpu_pct"`
	RAMTotalMB int64   `json:"ram_total_mb"`
	RAMUsedMB  int64   `json:"ram_used_mb"`
	DiskTotal  string  `json:"disk_total"`
	DiskUsed   string  `json:"disk_used"`
	DiskPct    string  `json:"disk_pct"`
	Load1      float64 `json:"load1"`
	Load5      float64 `json:"load5"`
	NetRxKB    int64   `json:"net_rx_kb"` // KB received since last poll
	NetTxKB    int64   `json:"net_tx_kb"` // KB sent since last poll
}

// ─── Handler ──────────────────────────────────────────────────────────────────

type ServerMetricsHandler struct {
	servers serverGetter
}

func NewServerMetricsHandler(servers serverGetter) *ServerMetricsHandler {
	return &ServerMetricsHandler{servers: servers}
}

const metricsInterval = 3 * time.Second

// Stream handles GET /servers/:id/metrics/stream as an SSE endpoint.
// It emits a MetricsSnapshot JSON every metricsInterval until the client disconnects.
func (h *ServerMetricsHandler) Stream(c *gin.Context) {
	serverID := c.Param("id")

	if _, ok := c.Get(contextUserIDKey); !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	srv, err := h.servers.GetByID(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "server not found"})
		return
	}

	// Re-use the SFTP connection pool — avoids a new TCP+SSH handshake each time.
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

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // disable Nginx response buffering

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "streaming unsupported"})
		return
	}

	ctx := c.Request.Context()
	ticker := time.NewTicker(metricsInterval)
	defer ticker.Stop()

	// First snapshot immediately — don't wait for the first tick.
	if snap, err := collectMetrics(sshCli); err == nil {
		sseJSON(c.Writer, flusher, "metrics", snap)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snap, err := collectMetrics(sshCli)
			if err != nil {
				filemanager.GlobalPool.Invalidate(serverID)
				fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", err.Error())
				flusher.Flush()
				return
			}
			sseJSON(c.Writer, flusher, "metrics", snap)
		}
	}
}

// ─── Metric collection ────────────────────────────────────────────────────────

// metricsScript collects CPU (via /proc/stat delta), memory, disk, load and
// network in a single SSH round-trip. The 1-second sleep inside is needed to
// compute a meaningful CPU delta. Total execution time: ~1 s.
//
// Output (pipe-separated):
//
//	cpu_pct | ram_total_mb | ram_used_mb | disk_total | disk_used | disk_pct | load1 | load5 | rx_kb | tx_kb
const metricsScript = `
s1=$(awk '/^cpu /{print $2+$3+$4+$5+$6+$7+$8,$5}' /proc/stat)
n1=$(awk 'NR>2{r+=$2;t+=$10}END{print r,t}' /proc/net/dev)
sleep 1
s2=$(awk '/^cpu /{print $2+$3+$4+$5+$6+$7+$8,$5}' /proc/stat)
n2=$(awk 'NR>2{r+=$2;t+=$10}END{print r,t}' /proc/net/dev)
t1=$(echo $s1|awk '{print $1}') i1=$(echo $s1|awk '{print $2}')
t2=$(echo $s2|awk '{print $1}') i2=$(echo $s2|awk '{print $2}')
cpu=$(awk -v t1=$t1 -v i1=$i1 -v t2=$t2 -v i2=$i2 \
  'BEGIN{dt=t2-t1;if(dt>0)printf "%.1f",100*(dt-(i2-i1))/dt;else print "0"}')
r1=$(echo $n1|awk '{print $1}') x1=$(echo $n1|awk '{print $2}')
r2=$(echo $n2|awk '{print $1}') x2=$(echo $n2|awk '{print $2}')
mt=$(awk '/^MemTotal/{print int($2/1024)}' /proc/meminfo)
ma=$(awk '/^MemAvailable/{print int($2/1024)}' /proc/meminfo)
df_line=$(df -h / | awk 'NR==2{print $2,$3,$5}')
la=$(awk '{print $1,$2}' /proc/loadavg)
echo "$cpu|$mt|$((mt-ma))|$df_line|$la|$(( (r2-r1)/1024 ))|$(( (x2-x1)/1024 ))"
`

func collectMetrics(sshCli *gossh.Client) (*MetricsSnapshot, error) {
	out, err := sshOutput(sshCli, metricsScript)
	if err != nil {
		return nil, fmt.Errorf("collect metrics: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(out), "|")
	if len(parts) < 10 {
		return nil, fmt.Errorf("unexpected metrics output (%d parts): %q", len(parts), out)
	}

	snap := &MetricsSnapshot{Timestamp: time.Now().UnixMilli()}
	snap.CPUPct, _ = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	snap.RAMTotalMB, _ = strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	snap.RAMUsedMB, _ = strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)

	// disk: "20G 15G 78%"
	diskParts := strings.Fields(parts[3])
	if len(diskParts) >= 3 {
		snap.DiskTotal = diskParts[0]
		snap.DiskUsed  = diskParts[1]
		snap.DiskPct   = diskParts[2]
	}

	snap.Load1, _ = strconv.ParseFloat(strings.TrimSpace(parts[4]), 64)
	snap.Load5, _ = strconv.ParseFloat(strings.TrimSpace(parts[5]), 64)
	snap.NetRxKB, _ = strconv.ParseInt(strings.TrimSpace(parts[6]), 10, 64)
	snap.NetTxKB, _ = strconv.ParseInt(strings.TrimSpace(parts[7]), 10, 64)

	return snap, nil
}

// sshOutput runs cmd on the remote host and returns its combined output.
func sshOutput(client *gossh.Client, cmd string) (string, error) {
	sess, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()

	out, err := sess.Output(cmd)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// ─── SSE helper ───────────────────────────────────────────────────────────────

func sseJSON(w gin.ResponseWriter, flusher http.Flusher, event string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
	flusher.Flush()
}
