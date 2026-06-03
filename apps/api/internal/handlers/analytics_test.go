package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/theairlock/airlock/apps/api/internal/middleware"
)

type fakeAnalyticsStore struct {
	state    boarderState
	stateErr error

	views          int
	viewsErr       error
	incrementErr   error
	incrementedAs  string
	incrementCalls int
}

func (s *fakeAnalyticsStore) LoadBoarderState(context.Context, string, time.Time) (boarderState, error) {
	return s.state, s.stateErr
}

func (s *fakeAnalyticsStore) IncrementProfileView(_ context.Context, handle string, _ time.Time) error {
	s.incrementCalls++
	s.incrementedAs = handle
	return s.incrementErr
}

func (s *fakeAnalyticsStore) GetProfileViews(context.Context, string) (int, error) {
	return s.views, s.viewsErr
}

func analyticsTestApp(handler *AnalyticsHandler) *fiber.App {
	app := fiber.New()
	inject := func(c *fiber.Ctx) error {
		middleware.WithUser(c, &middleware.User{ID: "USER-MAYA", Provider: "clerk"})
		return c.Next()
	}
	app.Post("/c/:handle/view", handler.IncrementView)
	app.Get("/me/analytics", inject, handler.GetAnalytics)
	return app
}

func newAnalyticsHandlerForTest(store analyticsStore) *AnalyticsHandler {
	return NewAnalyticsHandler(AnalyticsHandlerConfig{
		Clock: func() time.Time { return time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC) },
		Store: store,
	})
}

func TestIncrementViewHandleValidation(t *testing.T) {
	tests := []struct {
		name          string
		handle        string
		wantStatus    int
		wantIncrement bool
		wantStoredAs  string
	}{
		{name: "valid lowercases through", handle: "maya", wantStatus: http.StatusOK, wantIncrement: true, wantStoredAs: "maya"},
		{name: "mixed case lowercased", handle: "Maya", wantStatus: http.StatusOK, wantIncrement: true, wantStoredAs: "maya"},
		{name: "empty rejected", handle: "%20", wantStatus: http.StatusBadRequest},
		{name: "illegal char rejected", handle: "A!", wantStatus: http.StatusBadRequest},
		{name: "too long rejected", handle: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", wantStatus: http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeAnalyticsStore{}
			app := analyticsTestApp(newAnalyticsHandlerForTest(store))

			req := httptest.NewRequest(http.MethodPost, "/c/"+tc.handle+"/view", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}
			if tc.wantIncrement {
				if store.incrementCalls != 1 {
					t.Fatalf("increment calls = %d, want 1", store.incrementCalls)
				}
				if store.incrementedAs != tc.wantStoredAs {
					t.Fatalf("incremented handle = %q, want %q", store.incrementedAs, tc.wantStoredAs)
				}
				body := decodeJSON(t, resp)
				if body["ok"] != true {
					t.Fatalf("ok = %v, want true", body["ok"])
				}
			} else if store.incrementCalls != 0 {
				t.Fatalf("increment called for invalid handle %q", tc.handle)
			}
		})
	}
}

func TestIncrementViewStoreErrorIsUnavailable(t *testing.T) {
	store := &fakeAnalyticsStore{incrementErr: context.DeadlineExceeded}
	app := analyticsTestApp(newAnalyticsHandlerForTest(store))

	req := httptest.NewRequest(http.MethodPost, "/c/maya/view", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
	body := decodeJSON(t, resp)
	if body["error"] != "unavailable" {
		t.Fatalf("error = %v, want unavailable", body["error"])
	}
}

func TestGetAnalyticsReturnsViews(t *testing.T) {
	store := &fakeAnalyticsStore{
		state: boarderState{AirlockUser: airlockUser{Handle: "maya"}},
		views: 7,
	}
	app := analyticsTestApp(newAnalyticsHandlerForTest(store))

	req := httptest.NewRequest(http.MethodGet, "/me/analytics", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := decodeJSON(t, resp)
	if body["profile_views"] != float64(7) {
		t.Fatalf("profile_views = %v, want 7", body["profile_views"])
	}
}

func TestGetAnalyticsEmptyHandleReturnsZero(t *testing.T) {
	store := &fakeAnalyticsStore{state: boarderState{}, views: 99}
	app := analyticsTestApp(newAnalyticsHandlerForTest(store))

	req := httptest.NewRequest(http.MethodGet, "/me/analytics", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := decodeJSON(t, resp)
	if body["profile_views"] != float64(0) {
		t.Fatalf("profile_views = %v, want 0 for handleless caller", body["profile_views"])
	}
}
