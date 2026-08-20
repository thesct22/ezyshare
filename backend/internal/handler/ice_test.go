package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/thesct22/ezyshare/backend/internal/handler"
)

func TestHandleICEServers(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/ice-servers", nil)
	rec := httptest.NewRecorder()

	handler.HandleICEServers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp handler.ICEResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if len(resp.ICEServers) == 0 {
		t.Fatalf("expected at least 1 STUN server in response")
	}
}

func TestHandleICEServersWithTURN(t *testing.T) {
	os.Setenv("TURN_SERVER_URL", "turn:turn.ezyshare.dev:3478")
	os.Setenv("TURN_SECRET", "my-secret-key")
	defer os.Unsetenv("TURN_SERVER_URL")
	defer os.Unsetenv("TURN_SECRET")

	req := httptest.NewRequest("GET", "/api/v1/ice-servers", nil)
	rec := httptest.NewRecorder()

	handler.HandleICEServers(rec, req)

	var resp handler.ICEResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	if len(resp.ICEServers) < 2 {
		t.Fatalf("expected STUN + TURN servers, got %d", len(resp.ICEServers))
	}
	if resp.ICEServers[1].Username == "" || resp.ICEServers[1].Credential == "" {
		t.Fatalf("expected non-empty TURN credentials")
	}
}
