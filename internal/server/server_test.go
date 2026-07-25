package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ctrl-research/getbud/internal/auth"
	"github.com/ctrl-research/getbud/internal/server"
	"github.com/ctrl-research/getbud/internal/store"
	"github.com/ctrl-research/getbud/internal/store/storetest"
)

// TestAPIRequiresAuth walks every /api/v1 route and verifies it rejects
// unauthenticated requests.
func TestAPIRequiresAuth(t *testing.T) {
	pool := storetest.Pool(t)
	authSvc, err := auth.NewService(context.Background(), store.NewUsers(pool), store.NewSessions(pool), auth.Options{
		BaseURL: "http://localhost:8080", LocalAuth: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(server.New(pool, authSvc))
	defer srv.Close()

	routes := []struct{ method, path string }{
		{"GET", "/api/v1/me"},
		{"GET", "/api/v1/accounts"},
		{"POST", "/api/v1/accounts"},
		{"PATCH", "/api/v1/accounts/00000000-0000-0000-0000-000000000000"},
		{"DELETE", "/api/v1/accounts/00000000-0000-0000-0000-000000000000"},
		{"GET", "/api/v1/accounts/00000000-0000-0000-0000-000000000000/snapshots"},
		{"PUT", "/api/v1/accounts/00000000-0000-0000-0000-000000000000/snapshots"},
		{"GET", "/api/v1/categories"},
		{"POST", "/api/v1/categories"},
		{"GET", "/api/v1/transactions"},
		{"POST", "/api/v1/transactions"},
		{"POST", "/api/v1/transactions/transfer"},
		{"GET", "/api/v1/contribution-room"},
		{"PUT", "/api/v1/contribution-room/tfsa/2026"},
		{"POST", "/api/v1/imports/preview"},
		{"POST", "/api/v1/imports"},
		{"GET", "/api/v1/imports"},
		{"GET", "/api/v1/reports/summary"},
		{"GET", "/api/v1/reports/sankey"},
		{"GET", "/api/v1/reports/trends"},
		{"GET", "/api/v1/reports/net-worth"},
	}
	for _, route := range routes {
		req, err := http.NewRequest(route.method, srv.URL+route.path, strings.NewReader("{}"))
		if err != nil {
			t.Fatal(err)
		}
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", route.method, route.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s without session = %d, want 401", route.method, route.path, resp.StatusCode)
		}
	}
}

// TestHealthzOpen verifies the health endpoint needs no session.
func TestHealthzOpen(t *testing.T) {
	pool := storetest.Pool(t)
	authSvc, err := auth.NewService(context.Background(), store.NewUsers(pool), store.NewSessions(pool), auth.Options{
		BaseURL: "http://localhost:8080", LocalAuth: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(server.New(pool, authSvc))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz = %d, want 200", resp.StatusCode)
	}
}
