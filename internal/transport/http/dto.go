package http

type createServerRequest struct {
	Name        string `json:"name" binding:"required"`
	Host        string `json:"host" binding:"required"`
	Port        int    `json:"port"`
	Provider    string `json:"provider" binding:"required"`
	Status      string `json:"status"`
	SSHUser     string `json:"ssh_user"`
	SSHPassword string `json:"ssh_password"`
}

type updateServerRequest struct {
	Name        string `json:"name" binding:"required"`
	Host        string `json:"host" binding:"required"`
	Port        int    `json:"port"`
	Provider    string `json:"provider" binding:"required"`
	Status      string `json:"status"`
	SSHUser     string `json:"ssh_user"`
	SSHPassword string `json:"ssh_password"`
}

type createSiteRequest struct {
	ServerID      string `json:"server_id" binding:"required"`
	Name          string `json:"name" binding:"required"`
	Domain        string `json:"domain" binding:"required"`
	Runtime       string `json:"runtime" binding:"required"`
	RepositoryURL string `json:"repository_url"`
	Branch        string `json:"branch"`
	Status        string `json:"status"`
}

type updateSiteRequest struct {
	ServerID      string `json:"server_id" binding:"required"`
	Name          string `json:"name" binding:"required"`
	Domain        string `json:"domain" binding:"required"`
	Runtime       string `json:"runtime" binding:"required"`
	RepositoryURL string `json:"repository_url"`
	Branch        string `json:"branch"`
	Status        string `json:"status"`
}

type listResponse[T any] struct {
	Items []T `json:"items"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type devTokenRequest struct {
	UserID string `json:"user_id" binding:"required"`
	TeamID string `json:"team_id" binding:"required"`
	TTL    int64  `json:"ttl_seconds"`
}
