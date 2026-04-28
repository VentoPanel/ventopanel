package lifecycle

import "testing"

func TestEnsureServerTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		wantErr bool
	}{
		{name: "pending to connected", from: "pending", to: "connected", wantErr: false},
		{name: "connected to provisioning", from: "connected", to: "provisioning", wantErr: false},
		{name: "pending to ready_for_deploy invalid", from: "pending", to: "ready_for_deploy", wantErr: true},
	}

	for _, tt := range tests {
		err := EnsureServerTransition(tt.from, tt.to)
		if (err != nil) != tt.wantErr {
			t.Fatalf("%s: expected err=%v, got %v", tt.name, tt.wantErr, err)
		}
	}
}

func TestEnsureSiteTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		wantErr bool
	}{
		{name: "draft to deploying", from: "draft", to: "deploying", wantErr: false},
		{name: "deploying to ssl_pending", from: "deploying", to: "ssl_pending", wantErr: false},
		{name: "draft to deployed invalid", from: "draft", to: "deployed", wantErr: true},
	}

	for _, tt := range tests {
		err := EnsureSiteTransition(tt.from, tt.to)
		if (err != nil) != tt.wantErr {
			t.Fatalf("%s: expected err=%v, got %v", tt.name, tt.wantErr, err)
		}
	}
}
