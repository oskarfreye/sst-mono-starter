package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/theairlock/airlock/apps/api/internal/config"
)

func TestSpecAPIProtectedRoutesExistInFailClosedMode(t *testing.T) {
	api := New(&config.Config{
		AuthURL:   "http://example.com",
		TableName: "test-table",
	})

	for _, route := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/boarding/board"},
		{method: http.MethodPost, path: "/api/missions"},
		{method: http.MethodPost, path: "/api/missions/MISSION-001/proof"},
		{method: http.MethodGet, path: "/api/stacks/current"},
		{method: http.MethodGet, path: "/api/me/airlock"},
	} {
		t.Run(route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			resp, err := api.Test(req)
			if err != nil {
				t.Fatalf("api.Test returned error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusServiceUnavailable {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("status = %d, body = %s, want registered protected route to fail closed", resp.StatusCode, string(body))
			}
		})
	}
}

func TestSpecAPIProtectedRoutesExistWhenRouterStripsAPIPrefix(t *testing.T) {
	api := New(&config.Config{
		AuthURL:   "http://example.com",
		TableName: "test-table",
	})

	for _, route := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/boarding/board"},
		{method: http.MethodPost, path: "/missions"},
		{method: http.MethodPost, path: "/missions/MISSION-001/proof"},
		{method: http.MethodGet, path: "/stacks/current"},
		{method: http.MethodGet, path: "/me/airlock"},
	} {
		t.Run(route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			resp, err := api.Test(req)
			if err != nil {
				t.Fatalf("api.Test returned error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusServiceUnavailable {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("status = %d, body = %s, want registered protected route to fail closed", resp.StatusCode, string(body))
			}
		})
	}
}
