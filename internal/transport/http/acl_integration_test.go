package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	serverdomain "github.com/your-org/ventopanel/internal/domain/server"
	sitedomain "github.com/your-org/ventopanel/internal/domain/site"
	postgresrepo "github.com/your-org/ventopanel/internal/repository/postgres"
	serversvc "github.com/your-org/ventopanel/internal/service/server"
	sitesvc "github.com/your-org/ventopanel/internal/service/site"
	teamsvc "github.com/your-org/ventopanel/internal/service/team"
)

func withAuthFallback(engine *gin.Engine) {
	engine.Use(AuthContextMiddleware("", true))
}

func TestSiteGetByID_ACLAllowed(t *testing.T) {
	pool := openTestDB(t)
	ensureServersTable(t, pool)
	ensureSitesTable(t, pool)
	ensureTeamsAccessTables(t, pool)
	truncateServers(t, pool)

	serverRepo := postgresrepo.NewServerRepository(pool, testCipher{})
	siteRepo := postgresrepo.NewSiteRepository(pool)
	teamRepo := postgresrepo.NewTeamRepository(pool)

	server := createTestServer(t, serverRepo, "10.0.1.10")
	site := createTestSite(t, siteRepo, server.ID, "allowed.example.com")
	teamID := createTestTeam(t, pool, "team-acl-ok")
	grantSiteAccess(t, pool, teamID, site.ID, "owner")

	siteService := sitesvc.NewService(siteRepo, serverRepo)
	teamService := teamsvc.NewService(teamRepo)
	handler := NewSiteHandler(siteService, nil, teamService)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	withAuthFallback(engine)
	engine.GET("/api/v1/sites/:id", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sites/"+site.ID, nil)
	req.Header.Set("X-Team-ID", teamID)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSiteGetByID_ACLForbidden(t *testing.T) {
	pool := openTestDB(t)
	ensureServersTable(t, pool)
	ensureSitesTable(t, pool)
	ensureTeamsAccessTables(t, pool)
	truncateServers(t, pool)

	serverRepo := postgresrepo.NewServerRepository(pool, testCipher{})
	siteRepo := postgresrepo.NewSiteRepository(pool)
	teamRepo := postgresrepo.NewTeamRepository(pool)

	server := createTestServer(t, serverRepo, "10.0.1.11")
	site := createTestSite(t, siteRepo, server.ID, "forbidden.example.com")
	_ = createTestTeam(t, pool, "team-acl-no-grant")

	siteService := sitesvc.NewService(siteRepo, serverRepo)
	teamService := teamsvc.NewService(teamRepo)
	handler := NewSiteHandler(siteService, nil, teamService)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	withAuthFallback(engine)
	engine.GET("/api/v1/sites/:id", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sites/"+site.ID, nil)
	req.Header.Set("X-Team-ID", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSiteUpdate_ACLAdminAllowed(t *testing.T) {
	pool := openTestDB(t)
	ensureServersTable(t, pool)
	ensureSitesTable(t, pool)
	ensureTeamsAccessTables(t, pool)
	truncateServers(t, pool)

	serverRepo := postgresrepo.NewServerRepository(pool, testCipher{})
	siteRepo := postgresrepo.NewSiteRepository(pool)
	teamRepo := postgresrepo.NewTeamRepository(pool)

	server := createTestServer(t, serverRepo, "10.0.1.12")
	site := createTestSite(t, siteRepo, server.ID, "admin-write.example.com")
	teamID := createTestTeam(t, pool, "team-acl-admin")
	grantSiteAccess(t, pool, teamID, site.ID, "admin")

	siteService := sitesvc.NewService(siteRepo, serverRepo)
	teamService := teamsvc.NewService(teamRepo)
	handler := NewSiteHandler(siteService, nil, teamService)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	withAuthFallback(engine)
	engine.PUT("/api/v1/sites/:id", handler.Update)

	body := `{"server_id":"` + server.ID + `","name":"updated-name","domain":"admin-write.example.com","runtime":"node","repository_url":"https://example.com/repo.git","status":"draft"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/sites/"+site.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Team-ID", teamID)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSiteUpdate_ACLViewerForbidden(t *testing.T) {
	pool := openTestDB(t)
	ensureServersTable(t, pool)
	ensureSitesTable(t, pool)
	ensureTeamsAccessTables(t, pool)
	truncateServers(t, pool)

	serverRepo := postgresrepo.NewServerRepository(pool, testCipher{})
	siteRepo := postgresrepo.NewSiteRepository(pool)
	teamRepo := postgresrepo.NewTeamRepository(pool)

	server := createTestServer(t, serverRepo, "10.0.1.13")
	site := createTestSite(t, siteRepo, server.ID, "viewer-write.example.com")
	teamID := createTestTeam(t, pool, "team-acl-viewer")
	grantSiteAccess(t, pool, teamID, site.ID, "viewer")

	siteService := sitesvc.NewService(siteRepo, serverRepo)
	teamService := teamsvc.NewService(teamRepo)
	handler := NewSiteHandler(siteService, nil, teamService)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	withAuthFallback(engine)
	engine.PUT("/api/v1/sites/:id", handler.Update)

	body := `{"server_id":"` + server.ID + `","name":"should-not-update","domain":"viewer-write.example.com","runtime":"node","repository_url":"https://example.com/repo.git","status":"draft"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/sites/"+site.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Team-ID", teamID)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSiteDelete_ACLAdminAllowed(t *testing.T) {
	pool := openTestDB(t)
	ensureServersTable(t, pool)
	ensureSitesTable(t, pool)
	ensureTeamsAccessTables(t, pool)
	truncateServers(t, pool)

	serverRepo := postgresrepo.NewServerRepository(pool, testCipher{})
	siteRepo := postgresrepo.NewSiteRepository(pool)
	teamRepo := postgresrepo.NewTeamRepository(pool)

	server := createTestServer(t, serverRepo, "10.0.1.14")
	site := createTestSite(t, siteRepo, server.ID, "admin-delete.example.com")
	teamID := createTestTeam(t, pool, "team-acl-admin-delete")
	grantSiteAccess(t, pool, teamID, site.ID, "admin")

	siteService := sitesvc.NewService(siteRepo, serverRepo)
	teamService := teamsvc.NewService(teamRepo)
	handler := NewSiteHandler(siteService, nil, teamService)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	withAuthFallback(engine)
	engine.DELETE("/api/v1/sites/:id", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/sites/"+site.ID, nil)
	req.Header.Set("X-Team-ID", teamID)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSiteDeploy_ACLViewerForbidden(t *testing.T) {
	pool := openTestDB(t)
	ensureServersTable(t, pool)
	ensureSitesTable(t, pool)
	ensureTeamsAccessTables(t, pool)
	truncateServers(t, pool)

	serverRepo := postgresrepo.NewServerRepository(pool, testCipher{})
	siteRepo := postgresrepo.NewSiteRepository(pool)
	teamRepo := postgresrepo.NewTeamRepository(pool)

	server := createTestServer(t, serverRepo, "10.0.1.15")
	site := createTestSite(t, siteRepo, server.ID, "viewer-deploy.example.com")
	teamID := createTestTeam(t, pool, "team-acl-viewer-deploy")
	grantSiteAccess(t, pool, teamID, site.ID, "viewer")

	siteService := sitesvc.NewService(siteRepo, serverRepo)
	teamService := teamsvc.NewService(teamRepo)
	handler := NewSiteHandler(siteService, nil, teamService)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	withAuthFallback(engine)
	engine.POST("/api/v1/sites/:id/deploy", handler.Deploy)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/"+site.ID+"/deploy", nil)
	req.Header.Set("X-Team-ID", teamID)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestServerGetByID_ACLAllowedViaSiteGrant(t *testing.T) {
	pool := openTestDB(t)
	ensureServersTable(t, pool)
	ensureSitesTable(t, pool)
	ensureTeamsAccessTables(t, pool)
	truncateServers(t, pool)

	serverRepo := postgresrepo.NewServerRepository(pool, testCipher{})
	siteRepo := postgresrepo.NewSiteRepository(pool)
	teamRepo := postgresrepo.NewTeamRepository(pool)

	server := createTestServer(t, serverRepo, "10.0.1.16")
	site := createTestSite(t, siteRepo, server.ID, "server-acl-allowed.example.com")
	teamID := createTestTeam(t, pool, "team-server-read")
	grantSiteAccess(t, pool, teamID, site.ID, "viewer")

	serverService := serversvc.NewService(serverRepo, sshExecutorStub{}, nil)
	teamService := teamsvc.NewService(teamRepo)
	handler := NewServerHandler(serverService, nil, nil, teamService)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	withAuthFallback(engine)
	engine.GET("/api/v1/servers/:id", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+server.ID, nil)
	req.Header.Set("X-Team-ID", teamID)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestServerUpdate_ACLViewerForbidden(t *testing.T) {
	pool := openTestDB(t)
	ensureServersTable(t, pool)
	ensureSitesTable(t, pool)
	ensureTeamsAccessTables(t, pool)
	truncateServers(t, pool)

	serverRepo := postgresrepo.NewServerRepository(pool, testCipher{})
	siteRepo := postgresrepo.NewSiteRepository(pool)
	teamRepo := postgresrepo.NewTeamRepository(pool)

	server := createTestServer(t, serverRepo, "10.0.1.17")
	site := createTestSite(t, siteRepo, server.ID, "server-acl-viewer.example.com")
	teamID := createTestTeam(t, pool, "team-server-viewer")
	grantSiteAccess(t, pool, teamID, site.ID, "viewer")

	serverService := serversvc.NewService(serverRepo, sshExecutorStub{}, nil)
	teamService := teamsvc.NewService(teamRepo)
	handler := NewServerHandler(serverService, nil, nil, teamService)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	withAuthFallback(engine)
	engine.PUT("/api/v1/servers/:id", handler.Update)

	body := `{"name":"new-name","host":"10.0.1.17","port":22,"provider":"hetzner","status":"pending","ssh_user":"root","ssh_password":"secret"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/servers/"+server.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Team-ID", teamID)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSiteList_ACLFiltersInaccessibleItems(t *testing.T) {
	pool := openTestDB(t)
	ensureServersTable(t, pool)
	ensureSitesTable(t, pool)
	ensureTeamsAccessTables(t, pool)
	truncateServers(t, pool)

	serverRepo := postgresrepo.NewServerRepository(pool, testCipher{})
	siteRepo := postgresrepo.NewSiteRepository(pool)
	teamRepo := postgresrepo.NewTeamRepository(pool)

	server := createTestServer(t, serverRepo, "10.0.1.18")
	allowedSite := createTestSite(t, siteRepo, server.ID, "allowed-list.example.com")
	_ = createTestSite(t, siteRepo, server.ID, "denied-list.example.com")
	teamID := createTestTeam(t, pool, "team-site-list")
	grantSiteAccess(t, pool, teamID, allowedSite.ID, "viewer")

	siteService := sitesvc.NewService(siteRepo, serverRepo)
	teamService := teamsvc.NewService(teamRepo)
	handler := NewSiteHandler(siteService, nil, teamService)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	withAuthFallback(engine)
	engine.GET("/api/v1/sites", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sites", nil)
	req.Header.Set("X-Team-ID", teamID)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp listResponse[sitedomain.Site]
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 site after ACL filter, got %d", len(resp.Items))
	}
	if resp.Items[0].ID != allowedSite.ID {
		t.Fatalf("unexpected site in response: %s", resp.Items[0].ID)
	}
}

func TestServerList_ACLFiltersInaccessibleItems(t *testing.T) {
	pool := openTestDB(t)
	ensureServersTable(t, pool)
	ensureSitesTable(t, pool)
	ensureTeamsAccessTables(t, pool)
	truncateServers(t, pool)

	serverRepo := postgresrepo.NewServerRepository(pool, testCipher{})
	siteRepo := postgresrepo.NewSiteRepository(pool)
	teamRepo := postgresrepo.NewTeamRepository(pool)

	allowedServer := createTestServer(t, serverRepo, "10.0.1.19")
	deniedServer := createTestServer(t, serverRepo, "10.0.1.20")
	allowedSite := createTestSite(t, siteRepo, allowedServer.ID, "server-allowed-list.example.com")
	_ = createTestSite(t, siteRepo, deniedServer.ID, "server-denied-list.example.com")
	teamID := createTestTeam(t, pool, "team-server-list")
	grantSiteAccess(t, pool, teamID, allowedSite.ID, "viewer")

	serverService := serversvc.NewService(serverRepo, sshExecutorStub{}, nil)
	teamService := teamsvc.NewService(teamRepo)
	handler := NewServerHandler(serverService, nil, nil, teamService)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	withAuthFallback(engine)
	engine.GET("/api/v1/servers", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	req.Header.Set("X-Team-ID", teamID)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp listResponse[serverdomain.Server]
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 server after ACL filter, got %d", len(resp.Items))
	}
	if resp.Items[0].ID != allowedServer.ID {
		t.Fatalf("unexpected server in response: %s", resp.Items[0].ID)
	}
}

func ensureSitesTable(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(t.Context(), `
		CREATE TABLE IF NOT EXISTS sites (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			domain TEXT NOT NULL UNIQUE,
			runtime TEXT NOT NULL,
			repository_url TEXT,
			status TEXT NOT NULL DEFAULT 'draft',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		t.Fatalf("ensure sites table: %v", err)
	}
}

func ensureTeamsAccessTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(t.Context(), `
		CREATE TABLE IF NOT EXISTS teams (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		t.Fatalf("ensure teams table: %v", err)
	}
	if _, err := pool.Exec(t.Context(), `
		CREATE TABLE IF NOT EXISTS team_site_access (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
			site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
			role TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (team_id, site_id)
		)
	`); err != nil {
		t.Fatalf("ensure team_site_access table: %v", err)
	}
}

func createTestServer(t *testing.T, repo *postgresrepo.ServerRepository, host string) *serverdomain.Server {
	t.Helper()
	srv := &serverdomain.Server{
		Name:        "acl-server",
		Host:        host,
		Port:        22,
		Provider:    "hetzner",
		Status:      "pending",
		SSHUser:     "root",
		SSHPassword: "secret",
	}
	if err := repo.Create(t.Context(), srv); err != nil {
		t.Fatalf("create server: %v", err)
	}
	return srv
}

func createTestSite(t *testing.T, repo *postgresrepo.SiteRepository, serverID, domain string) *sitedomain.Site {
	t.Helper()
	site := &sitedomain.Site{
		ServerID:      serverID,
		Name:          "acl-site",
		Domain:        domain,
		Runtime:       "node",
		RepositoryURL: "https://example.com/repo.git",
		Status:        "draft",
	}
	if err := repo.Create(t.Context(), site); err != nil {
		t.Fatalf("create site: %v", err)
	}
	return site
}

func createTestTeam(t *testing.T, pool *pgxpool.Pool, name string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(t.Context(), `INSERT INTO teams (name) VALUES ($1) RETURNING id`, name).Scan(&id); err != nil {
		t.Fatalf("create team: %v", err)
	}
	return id
}

func grantSiteAccess(t *testing.T, pool *pgxpool.Pool, teamID, siteID, role string) {
	t.Helper()
	if _, err := pool.Exec(t.Context(),
		`INSERT INTO team_site_access (team_id, site_id, role) VALUES ($1, $2, $3)`,
		teamID, siteID, role,
	); err != nil {
		t.Fatalf("grant site access: %v", err)
	}
}
