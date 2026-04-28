package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	auditdomain "github.com/your-org/ventopanel/internal/domain/audit"
	serverdomain "github.com/your-org/ventopanel/internal/domain/server"
	postgresrepo "github.com/your-org/ventopanel/internal/repository/postgres"
	auditsvc "github.com/your-org/ventopanel/internal/service/audit"
	serversvc "github.com/your-org/ventopanel/internal/service/server"
)

func TestAuditStatusEvents_IncludeTotalAndFilters(t *testing.T) {
	pool := openTestDB(t)
	repo := postgresrepo.NewStatusEventRepository(pool)
	truncateStatusEvents(t, pool)

	base := time.Now().UTC().Add(-1 * time.Hour)
	writeEvent(t, repo, "site", "11111111-1111-1111-1111-111111111111", "deploying", "deployed", base.Add(1*time.Minute))
	writeEvent(t, repo, "site", "11111111-1111-1111-1111-111111111111", "deployed", "ssl_pending", base.Add(2*time.Minute))
	writeEvent(t, repo, "site", "11111111-1111-1111-1111-111111111111", "ssl_pending", "deployed", base.Add(3*time.Minute))
	writeEvent(t, repo, "server", "22222222-2222-2222-2222-222222222222", "new", "provisioning", base.Add(4*time.Minute))

	engine := buildAuditTestEngine(repo)
	since := base.Format(time.RFC3339)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/status-events?resource_type=site&resource_id=11111111-1111-1111-1111-111111111111&since="+since+"&limit=2&include_total=true", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp auditResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items by limit, got %d", len(resp.Items))
	}
	if resp.TotalCount == nil {
		t.Fatal("expected total_count to be present when include_total=true")
	}
	if *resp.TotalCount != 3 {
		t.Fatalf("expected total_count=3, got %d", *resp.TotalCount)
	}
	if strings.TrimSpace(resp.NextCursor) == "" {
		t.Fatal("expected next_cursor to be set when page is full")
	}
	for _, it := range resp.Items {
		if it.ResourceType != "site" {
			t.Fatalf("expected only site events, got resource_type=%q", it.ResourceType)
		}
		if it.ResourceID != "11111111-1111-1111-1111-111111111111" {
			t.Fatalf("unexpected resource_id=%q", it.ResourceID)
		}
	}
}

func TestAuditStatusEvents_CursorPagination(t *testing.T) {
	pool := openTestDB(t)
	repo := postgresrepo.NewStatusEventRepository(pool)
	truncateStatusEvents(t, pool)

	base := time.Now().UTC().Add(-2 * time.Hour)
	writeEvent(t, repo, "site", "33333333-3333-3333-3333-333333333333", "a", "b", base.Add(1*time.Minute))
	writeEvent(t, repo, "site", "33333333-3333-3333-3333-333333333333", "b", "c", base.Add(2*time.Minute))
	writeEvent(t, repo, "site", "33333333-3333-3333-3333-333333333333", "c", "d", base.Add(3*time.Minute))

	engine := buildAuditTestEngine(repo)

	firstReq := httptest.NewRequest(http.MethodGet, "/api/v1/audit/status-events?resource_type=site&resource_id=33333333-3333-3333-3333-333333333333&limit=2", nil)
	firstRec := httptest.NewRecorder()
	engine.ServeHTTP(firstRec, firstReq)

	if firstRec.Code != http.StatusOK {
		t.Fatalf("first page expected 200, got %d: %s", firstRec.Code, firstRec.Body.String())
	}

	var first auditResponse
	if err := json.Unmarshal(firstRec.Body.Bytes(), &first); err != nil {
		t.Fatalf("unmarshal first page: %v", err)
	}
	if len(first.Items) != 2 {
		t.Fatalf("expected 2 items on first page, got %d", len(first.Items))
	}
	if strings.TrimSpace(first.NextCursor) == "" {
		t.Fatal("expected non-empty next_cursor for first page")
	}

	secondReq := httptest.NewRequest(http.MethodGet, "/api/v1/audit/status-events?resource_type=site&resource_id=33333333-3333-3333-3333-333333333333&limit=2&before="+first.NextCursor, nil)
	secondRec := httptest.NewRecorder()
	engine.ServeHTTP(secondRec, secondReq)

	if secondRec.Code != http.StatusOK {
		t.Fatalf("second page expected 200, got %d: %s", secondRec.Code, secondRec.Body.String())
	}

	var second auditResponse
	if err := json.Unmarshal(secondRec.Body.Bytes(), &second); err != nil {
		t.Fatalf("unmarshal second page: %v", err)
	}
	if len(second.Items) != 1 {
		t.Fatalf("expected 1 item on second page, got %d", len(second.Items))
	}
	if second.Items[0].ID == first.Items[0].ID || second.Items[0].ID == first.Items[1].ID {
		t.Fatal("cursor pagination returned duplicated item")
	}
	if second.NextCursor != "" {
		t.Fatalf("expected empty next_cursor on final page, got %q", second.NextCursor)
	}
}

func TestAuditStatusEvents_InvalidIncludeTotal(t *testing.T) {
	pool := openTestDB(t)
	repo := postgresrepo.NewStatusEventRepository(pool)
	truncateStatusEvents(t, pool)

	engine := buildAuditTestEngine(repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/status-events?include_total=definitely-not-bool", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestServerConnect_WritesAuditOnSuccess(t *testing.T) {
	pool := openTestDB(t)
	ensureServersTable(t, pool)
	repo := postgresrepo.NewServerRepository(pool, testCipher{})
	auditRepo := postgresrepo.NewStatusEventRepository(pool)
	truncateStatusEvents(t, pool)
	truncateServers(t, pool)

	serverService := serversvc.NewService(repo, sshExecutorStub{}, auditRepo)
	created, err := serverService.Create(t.Context(), serverdomain.Server{
		Name:        "srv-connect-ok",
		Host:        "10.0.0.10",
		Port:        22,
		Provider:    "hetzner",
		Status:      "pending",
		SSHUser:     "root",
		SSHPassword: "secret",
	})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	h := NewServerHandler(serverService, nil, nil, nil, nil)
	engine := gin.New()
	engine.POST("/api/v1/servers/:id/connect", h.Connect)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/servers/"+created.ID+"/connect", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var got serverdomain.Server
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Status != "connected" {
		t.Fatalf("expected status connected, got %q", got.Status)
	}

	events, err := auditRepo.ListStatusEvents(auditdomain.StatusEventFilter{
		ResourceType: "server",
		ResourceID:   created.ID,
		Limit:        10,
	})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected at least one audit event")
	}
	ev := events[0]
	if ev.FromStatus != "pending" || ev.ToStatus != "connected" {
		t.Fatalf("unexpected transition: %q -> %q", ev.FromStatus, ev.ToStatus)
	}
	if ev.Reason != "ssh_connect_success" || ev.TaskID != "connect" {
		t.Fatalf("unexpected audit metadata: reason=%q task_id=%q", ev.Reason, ev.TaskID)
	}
}

func TestServerConnect_WritesAuditOnFailure(t *testing.T) {
	pool := openTestDB(t)
	ensureServersTable(t, pool)
	repo := postgresrepo.NewServerRepository(pool, testCipher{})
	auditRepo := postgresrepo.NewStatusEventRepository(pool)
	truncateStatusEvents(t, pool)
	truncateServers(t, pool)

	serverService := serversvc.NewService(repo, sshExecutorStub{testConnErr: errors.New("ssh timeout")}, auditRepo)
	created, err := serverService.Create(t.Context(), serverdomain.Server{
		Name:        "srv-connect-fail",
		Host:        "10.0.0.11",
		Port:        22,
		Provider:    "do",
		Status:      "pending",
		SSHUser:     "root",
		SSHPassword: "secret",
	})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	h := NewServerHandler(serverService, nil, nil, nil, nil)
	engine := gin.New()
	engine.POST("/api/v1/servers/:id/connect", h.Connect)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/servers/"+created.ID+"/connect", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d: %s", rec.Code, rec.Body.String())
	}

	updated, err := repo.GetByID(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("get server after failed connect: %v", err)
	}
	if updated.Status != "connection_failed" {
		t.Fatalf("expected status connection_failed, got %q", updated.Status)
	}

	events, err := auditRepo.ListStatusEvents(auditdomain.StatusEventFilter{
		ResourceType: "server",
		ResourceID:   created.ID,
		Limit:        10,
	})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected at least one audit event")
	}
	ev := events[0]
	if ev.FromStatus != "pending" || ev.ToStatus != "connection_failed" {
		t.Fatalf("unexpected transition: %q -> %q", ev.FromStatus, ev.ToStatus)
	}
	if ev.Reason != "ssh_connect_failed" || ev.TaskID != "connect" {
		t.Fatalf("unexpected audit metadata: reason=%q task_id=%q", ev.Reason, ev.TaskID)
	}
}

func TestServerConnect_InvalidTransition_DoesNotWriteAudit(t *testing.T) {
	pool := openTestDB(t)
	ensureServersTable(t, pool)
	repo := postgresrepo.NewServerRepository(pool, testCipher{})
	auditRepo := postgresrepo.NewStatusEventRepository(pool)
	truncateStatusEvents(t, pool)
	truncateServers(t, pool)

	serverService := serversvc.NewService(repo, sshExecutorStub{}, auditRepo)
	created, err := serverService.Create(t.Context(), serverdomain.Server{
		Name:        "srv-invalid-transition",
		Host:        "10.0.0.12",
		Port:        22,
		Provider:    "e2e",
		Status:      "ready_for_deploy",
		SSHUser:     "root",
		SSHPassword: "secret",
	})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	h := NewServerHandler(serverService, nil, nil, nil, nil)
	engine := gin.New()
	engine.POST("/api/v1/servers/:id/connect", h.Connect)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/servers/"+created.ID+"/connect", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502 for invalid transition, got %d: %s", rec.Code, rec.Body.String())
	}

	current, err := repo.GetByID(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("get server after invalid transition: %v", err)
	}
	if current.Status != "ready_for_deploy" {
		t.Fatalf("server status changed unexpectedly: %q", current.Status)
	}

	events, err := auditRepo.ListStatusEvents(auditdomain.StatusEventFilter{
		ResourceType: "server",
		ResourceID:   created.ID,
		Limit:        10,
	})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no audit events, got %d", len(events))
	}
}

func TestServerConnect_NotFound(t *testing.T) {
	pool := openTestDB(t)
	ensureServersTable(t, pool)
	repo := postgresrepo.NewServerRepository(pool, testCipher{})
	auditRepo := postgresrepo.NewStatusEventRepository(pool)
	truncateStatusEvents(t, pool)
	truncateServers(t, pool)

	serverService := serversvc.NewService(repo, sshExecutorStub{}, auditRepo)
	h := NewServerHandler(serverService, nil, nil, nil, nil)

	engine := gin.New()
	engine.POST("/api/v1/servers/:id/connect", h.Connect)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/servers/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/connect", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

type auditResponse struct {
	Items      []auditdomain.StatusEvent `json:"items"`
	NextCursor string                    `json:"next_cursor"`
	TotalCount *int64                    `json:"total_count,omitempty"`
}

func buildAuditTestEngine(repo *postgresrepo.StatusEventRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	svc := auditsvc.NewService(repo)
	h := NewAuditHandler(svc)

	engine := gin.New()
	engine.GET("/api/v1/audit/status-events", h.ListStatusEvents)
	return engine
}

type sshExecutorStub struct {
	testConnErr error
}

func (s sshExecutorStub) Run(_ context.Context, _ serverdomain.Server, _ string) error {
	return nil
}

func (s sshExecutorStub) RunScript(_ context.Context, _ serverdomain.Server, _ []string) error {
	return nil
}

func (s sshExecutorStub) TestConnection(_ context.Context, _ serverdomain.Server) error {
	return s.testConnErr
}

type testCipher struct{}

func (testCipher) Encrypt(plaintext string) (string, error) {
	return plaintext, nil
}

func (testCipher) Decrypt(value string) (string, error) {
	return value, nil
}

func openTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("TEST_POSTGRES_DSN"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("POSTGRES_DSN"))
	}
	if dsn == "" {
		t.Skip("set TEST_POSTGRES_DSN (or POSTGRES_DSN) to run integration tests")
	}

	pool, err := pgxpool.New(t.Context(), dsn)
	if err != nil {
		t.Skipf("postgres unavailable (%v)", err)
	}

	if _, err := pool.Exec(t.Context(), "CREATE EXTENSION IF NOT EXISTS pgcrypto"); err != nil {
		pool.Close()
		t.Skipf("cannot create pgcrypto extension (%v)", err)
	}
	if _, err := pool.Exec(t.Context(), `
		CREATE TABLE IF NOT EXISTS status_events (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			resource_type TEXT NOT NULL,
			resource_id UUID NOT NULL,
			from_status TEXT NOT NULL,
			to_status TEXT NOT NULL,
			reason TEXT NOT NULL DEFAULT '',
			task_id TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		pool.Close()
		t.Skipf("cannot ensure status_events table (%v)", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})
	return pool
}

func truncateStatusEvents(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(t.Context(), "TRUNCATE TABLE status_events"); err != nil {
		t.Fatalf("truncate status_events: %v", err)
	}
}

func ensureServersTable(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(t.Context(), `
		CREATE TABLE IF NOT EXISTS servers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			host TEXT NOT NULL UNIQUE,
			port INTEGER NOT NULL DEFAULT 22,
			provider TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			ssh_user TEXT NOT NULL DEFAULT 'root',
			ssh_password TEXT NOT NULL DEFAULT '',
			last_renew_at TIMESTAMPTZ NULL,
			last_renew_status TEXT NOT NULL DEFAULT 'unknown',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		t.Fatalf("ensure servers table: %v", err)
	}
}

func truncateServers(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(t.Context(), "TRUNCATE TABLE servers CASCADE"); err != nil {
		t.Fatalf("truncate servers: %v", err)
	}
}

func writeEvent(
	t *testing.T,
	repo *postgresrepo.StatusEventRepository,
	resourceType, resourceID, from, to string,
	createdAt time.Time,
) {
	t.Helper()
	err := repo.WriteStatusEvent(auditdomain.StatusEvent{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		FromStatus:   from,
		ToStatus:     to,
		Reason:       "test",
		TaskID:       "task-test",
		CreatedAt:    createdAt.UTC(),
	})
	if err != nil {
		t.Fatalf("write status event: %v", err)
	}
}
