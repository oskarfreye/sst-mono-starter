package handlers_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/theairlock/airlock/apps/api/internal/app"
	"github.com/theairlock/airlock/apps/api/internal/config"
	"github.com/theairlock/airlock/apps/api/internal/handlers"
)

func TestStateRouteReturnsLaunchSnapshotPublicly(t *testing.T) {
	api := app.New(&config.Config{AuthURL: "invalid-url"})

	resp := requestState(t, api, "/v2/state")
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	cacheControl := resp.Header.Get("Cache-Control")
	if !strings.Contains(cacheControl, "max-age=10") {
		t.Fatalf("Cache-Control = %q, want 10s TTL", cacheControl)
	}

	var got stateSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(got.Seats) != 100 {
		t.Fatalf("len(seats) = %d, want 100", len(got.Seats))
	}
	first := got.Seats[0]
	if first.SeatLabel != "01" || first.Cohort != "THE_100" {
		t.Fatalf("seat 01 = %#v, want THE_100 seat 01", first)
	}
	if first.Occupant == nil || first.Occupant.Handle != "oskar" {
		t.Fatalf("seat 01 occupant = %#v, want @oskar", first.Occupant)
	}
	if first.Mission == nil {
		t.Fatalf("seat 01 mission = nil, want launch mission")
	}

	for i, seat := range got.Seats[1:] {
		if seat.Status != "OPEN" || seat.Occupant != nil || seat.Mission != nil {
			t.Fatalf("seat %02d = %#v, want open empty seat", i+2, seat)
		}
	}
	if got.NextSeat.SeatLabel != "02" || got.NextSeat.Price.Cents != 4200 || got.NextSeat.Price.Display != "$42" {
		t.Fatalf("next seat = %#v, want seat 02 at $42", got.NextSeat)
	}
	if len(got.PriceTiers) != 4 {
		t.Fatalf("len(price_tiers) = %d, want 4", len(got.PriceTiers))
	}
	if got.PriceTiers[0].Tier != "01" || got.PriceTiers[0].Price.Display != "$42" || got.PriceTiers[0].Filled != 1 || got.PriceTiers[0].Seats != 10 {
		t.Fatalf("price tier 01 = %#v, want $42 1/10", got.PriceTiers[0])
	}
	if got.TierProgress.Progress != "1/10" || got.TierProgress.Occupied != 1 || got.TierProgress.Capacity != 10 {
		t.Fatalf("tier progress = %#v, want 1/10", got.TierProgress)
	}
	if got.Metrics.LaunchesConfirmedLifetime != 0 {
		t.Fatalf("metrics = %#v, want no confirmed launches in launch seed", got.Metrics)
	}
}

func TestStateRouteSupportsSpecPath(t *testing.T) {
	api := app.New(&config.Config{AuthURL: "invalid-url"})
	resp := requestState(t, api, "/api/state")
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestSeatsRouteReturnsPublicSeatCollections(t *testing.T) {
	api := app.New(&config.Config{AuthURL: "invalid-url"})
	resp := requestState(t, api, "/v2/seats")
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got seatsSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got.Seats) != 100 || len(got.ActiveSeats) != 1 || got.NextSeat.SeatLabel != "02" {
		t.Fatalf("seats response = %#v, want public seat collections and next seat", got)
	}
}

func TestSeatsRouteSupportsSpecPath(t *testing.T) {
	api := app.New(&config.Config{AuthURL: "invalid-url"})
	resp := requestState(t, api, "/api/seats")
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestEventsRouteStreamsStateSnapshot(t *testing.T) {
	api := app.New(&config.Config{AuthURL: "invalid-url"})
	req := httptest.NewRequest("GET", "/v2/events?once=1", nil)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := api.Test(req)
	if err != nil {
		t.Fatalf("GET /v2/events: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", contentType)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "event: state") || !strings.Contains(text, `"snapshot":"live"`) || !strings.Contains(text, `"recent_events"`) {
		t.Fatalf("body = %q, want state SSE snapshot", text)
	}
}

func TestEventsRouteSupportsSpecPath(t *testing.T) {
	api := app.New(&config.Config{AuthURL: "invalid-url"})
	req := httptest.NewRequest("GET", "/api/events?once=1", nil)

	resp, err := api.Test(req)
	if err != nil {
		t.Fatalf("GET /api/events: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestEventsHTTPHandlerStreamsStateSnapshot(t *testing.T) {
	handler := handlers.NewStateHandler(handlers.StateHandlerConfig{})
	req := httptest.NewRequest("GET", "/api/events?once=1", nil)
	rec := httptest.NewRecorder()

	handler.StreamEventsHTTP(rec, req)

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s, want 200", rec.Code, body)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", contentType)
	}
	if !strings.Contains(body, "event: state") || !strings.Contains(body, `"snapshot":"live"`) || !strings.Contains(body, `"recent_events"`) {
		t.Fatalf("body = %q, want state SSE snapshot", body)
	}
}

func TestMissionRoutesFailClosedWhenAuthUnavailable(t *testing.T) {
	api := app.New(&config.Config{AuthURL: "invalid-url"})
	req := httptest.NewRequest("POST", "/v2/missions", strings.NewReader(`{"declaration":"Ship a paid onboarding release","declaration_url":"https://example.com/app"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.Test(req)
	if err != nil {
		t.Fatalf("POST /v2/missions: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestHatchRouteFailsClosedWhenAuthUnavailable(t *testing.T) {
	api := app.New(&config.Config{AuthURL: "invalid-url"})
	req := httptest.NewRequest("POST", "/v2/hatch/run", nil)

	resp, err := api.Test(req)
	if err != nil {
		t.Fatalf("POST /v2/hatch/run: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestAdminReviewRoutesFailClosedWhenAuthUnavailable(t *testing.T) {
	api := app.New(&config.Config{AuthURL: "invalid-url"})
	req := httptest.NewRequest("POST", "/v2/admin/missions/MISSION-001/confirm", nil)

	resp, err := api.Test(req)
	if err != nil {
		t.Fatalf("POST /v2/admin/missions/MISSION-001/confirm: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestBoarderStateRouteFailsClosedWhenAuthUnavailable(t *testing.T) {
	api := app.New(&config.Config{AuthURL: "invalid-url"})
	req := httptest.NewRequest("GET", "/v2/me/airlock", nil)

	resp, err := api.Test(req)
	if err != nil {
		t.Fatalf("GET /v2/me/airlock: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func requestState(t *testing.T, api interface {
	Test(*http.Request, ...int) (*http.Response, error)
}, path string) *http.Response {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	resp, err := api.Test(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

type stateSnapshot struct {
	Seats        []stateSeat   `json:"seats"`
	VacatedSeats []stateSeat   `json:"vacated_seats"`
	NextSeat     stateNextSeat `json:"next_seat"`
	PriceTiers   []stateTier   `json:"price_tiers"`
	TierProgress stateTier     `json:"tier_progress"`
	NearestHatch *stateHatch   `json:"nearest_hatch"`
	Metrics      stateMetrics  `json:"metrics"`
}

type seatsSnapshot struct {
	Seats       []stateSeat   `json:"seats"`
	ActiveSeats []stateSeat   `json:"active_seats"`
	NextSeat    stateNextSeat `json:"next_seat"`
}

type stateSeat struct {
	SeatLabel string         `json:"seat_label"`
	Cohort    string         `json:"cohort"`
	Status    string         `json:"status"`
	Occupant  *stateOccupant `json:"occupant"`
	Mission   *stateMission  `json:"mission"`
}

type stateOccupant struct {
	Handle string `json:"handle"`
}

type stateMission struct {
	Status string `json:"status"`
}

type stateNextSeat struct {
	SeatLabel string     `json:"seat_label"`
	Price     statePrice `json:"price"`
}

type statePrice struct {
	Cents   int    `json:"cents"`
	Display string `json:"display"`
}

type stateTier struct {
	Progress string     `json:"progress"`
	Tier     string     `json:"tier"`
	Price    statePrice `json:"price"`
	Filled   int        `json:"filled"`
	Seats    int        `json:"seats"`
	Occupied int        `json:"occupied"`
	Capacity int        `json:"capacity"`
}

type stateHatch struct {
	Handle string `json:"handle"`
	Status string `json:"status"`
}

type stateMetrics struct {
	SeatsOccupied             int `json:"seats_occupied"`
	SeatsVacated              int `json:"seats_vacated"`
	LaunchesConfirmedLifetime int `json:"launches_confirmed_lifetime"`
}
