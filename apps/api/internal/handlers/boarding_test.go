package handlers

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/theairlock/airlock/apps/api/internal/middleware"
)

type fakeBoardingStore struct {
	result          boardingResult
	err             error
	called          bool
	userID          string
	email           string
	handle          string
	displayName     string
	planAmountCents int
}

func (s *fakeBoardingStore) Board(_ context.Context, userID string, email string, handle string, displayName string, planAmountCents int, _ time.Time) (boardingResult, error) {
	s.called = true
	s.userID = userID
	s.email = email
	s.handle = handle
	s.displayName = displayName
	s.planAmountCents = planAmountCents
	return s.result, s.err
}

func boardWithUser(handler *BoardingHandler, user *middleware.User) *fiber.App {
	app := fiber.New()
	app.Post("/boarding/board", func(c *fiber.Ctx) error {
		if user != nil {
			middleware.WithUser(c, user)
		}
		return c.Next()
	}, handler.Board)
	return app
}

func TestBoardCreatesSeat(t *testing.T) {
	store := &fakeBoardingStore{result: boardingResult{
		SeatID:     "SEAT-002",
		SeatNumber: 2,
		SeatLabel:  "02",
		Cohort:     "THE_100",
		Handle:     "maya",
		Created:    true,
	}}
	handler := NewBoardingHandler(BoardingHandlerConfig{
		Store: store,
		Clock: func() time.Time { return mustTime(t, "2026-05-27T12:00:00Z") },
	})
	app := boardWithUser(handler, &middleware.User{ID: "USER-MAYA", Contact: &middleware.Contact{Email: "maya@example.com"}})

	req := httptest.NewRequest(http.MethodPost, "/boarding/board", bytes.NewReader([]byte(`{"handle":"maya","display_name":"Maya"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s, want 200", resp.StatusCode, string(body))
	}
	if !strings.Contains(string(body), `"created":true`) || !strings.Contains(string(body), `"seat_id":"SEAT-002"`) {
		t.Fatalf("body = %s, want created seat", string(body))
	}
	if !strings.Contains(string(body), `"profile_path":"/c/maya"`) {
		t.Fatalf("body = %s, want profile path", string(body))
	}
	if !store.called || store.userID != "USER-MAYA" || store.email != "maya@example.com" || store.handle != "maya" || store.displayName != "Maya" {
		t.Fatalf("store call = %#v, want forwarded user, email, handle, display name", store)
	}
}

func TestBoardIdempotentReturnsExistingSeat(t *testing.T) {
	store := &fakeBoardingStore{result: boardingResult{
		SeatID:     "SEAT-002",
		SeatNumber: 2,
		SeatLabel:  "02",
		Cohort:     "THE_100",
		Handle:     "maya",
		Created:    false,
	}}
	handler := NewBoardingHandler(BoardingHandlerConfig{Store: store})
	app := boardWithUser(handler, &middleware.User{ID: "USER-MAYA"})

	req := httptest.NewRequest(http.MethodPost, "/boarding/board", bytes.NewReader([]byte(`{"handle":"maya"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s, want 200", resp.StatusCode, string(body))
	}
	if !strings.Contains(string(body), `"created":false`) || !strings.Contains(string(body), `"seat_id":"SEAT-002"`) {
		t.Fatalf("body = %s, want idempotent existing seat with created:false", string(body))
	}
}

func TestBoardRequiresAuthenticatedUser(t *testing.T) {
	store := &fakeBoardingStore{}
	handler := NewBoardingHandler(BoardingHandlerConfig{Store: store})
	app := boardWithUser(handler, nil)

	req := httptest.NewRequest(http.MethodPost, "/boarding/board", bytes.NewReader([]byte(`{"handle":"maya"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	if store.called {
		t.Fatalf("store.Board called for unauthenticated request")
	}
}

func TestBoardMapsHandleTakenTo409(t *testing.T) {
	store := &fakeBoardingStore{err: ErrBoardingHandleTaken}
	handler := NewBoardingHandler(BoardingHandlerConfig{Store: store})
	app := boardWithUser(handler, &middleware.User{ID: "USER-MAYA"})

	req := httptest.NewRequest(http.MethodPost, "/boarding/board", bytes.NewReader([]byte(`{"handle":"maya"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
}

func TestBoardMapsInvalidHandleTo400(t *testing.T) {
	store := &fakeBoardingStore{err: ErrBoardingInvalidHandle}
	handler := NewBoardingHandler(BoardingHandlerConfig{Store: store})
	app := boardWithUser(handler, &middleware.User{ID: "USER-MAYA"})

	req := httptest.NewRequest(http.MethodPost, "/boarding/board", bytes.NewReader([]byte(`{"handle":"!!"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestBoardForwardsPlanAmountCents(t *testing.T) {
	cases := []struct {
		plan string
		want int
	}{
		{plan: "u:tier-04", want: 33600},
		{plan: "u:tier-01", want: 4200},
	}
	for _, tc := range cases {
		store := &fakeBoardingStore{result: boardingResult{SeatID: "SEAT-002", SeatNumber: 2, Handle: "maya", Created: true}}
		handler := NewBoardingHandler(BoardingHandlerConfig{Store: store})
		app := boardWithUser(handler, &middleware.User{ID: "USER-MAYA", Plan: tc.plan})

		req := httptest.NewRequest(http.MethodPost, "/boarding/board", bytes.NewReader([]byte(`{"handle":"maya"}`)))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test returned error: %v", err)
		}
		resp.Body.Close()

		if !store.called {
			t.Fatalf("plan %q: store.Board not called", tc.plan)
		}
		if store.planAmountCents != tc.want {
			t.Fatalf("plan %q: store.planAmountCents = %d, want %d", tc.plan, store.planAmountCents, tc.want)
		}
	}
}

func TestBoardMapsPlanUnderpaidTo402(t *testing.T) {
	store := &fakeBoardingStore{err: ErrBoardingPlanUnderpaid}
	handler := NewBoardingHandler(BoardingHandlerConfig{Store: store})
	app := boardWithUser(handler, &middleware.User{ID: "USER-MAYA", Plan: "u:tier-01"})

	req := httptest.NewRequest(http.MethodPost, "/boarding/board", bytes.NewReader([]byte(`{"handle":"maya"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "does not cover") {
		t.Fatalf("body = %s, want underpaid message", string(body))
	}
}
