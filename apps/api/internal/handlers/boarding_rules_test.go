package handlers

import (
	"errors"
	"testing"
)

func TestBuildBoardingAssignsNextSeatAtFlatPrice(t *testing.T) {
	now := mustTime(t, "2026-05-27T12:00:00Z")
	got, err := buildBoarding(launchStateRecords(), "USER-MAYA", "maya@example.com", "Maya.Space", "  Maya   Space  ", 33600, now)
	if err != nil {
		t.Fatalf("buildBoarding returned error: %v", err)
	}

	if got.Seat.SeatNumber != 2 || got.Seat.SeatID != "SEAT-002" {
		t.Fatalf("seat = %#v, want next open seat 2", got.Seat)
	}
	if got.Seat.PricePaidCents != boardingDisplayPriceCents || got.Seat.TierPaid != 1 {
		t.Fatalf("seat = %#v, want flat $42 display price in tier 1", got.Seat)
	}
	if got.Seat.Cohort != "THE_100" || got.Seat.Status != "OCCUPIED" {
		t.Fatalf("seat = %#v, want occupied THE_100 seat", got.Seat)
	}
	if got.User.Handle != "maya-space" || got.User.DisplayName != "Maya Space" {
		t.Fatalf("user = %#v, want normalized handle and display name", got.User)
	}
	if got.User.Email != "maya@example.com" {
		t.Fatalf("user = %#v, want email carried onto user", got.User)
	}
	if got.Event.EventID != "EVENT-BOARDED-SEAT-002" || got.Event.Kind != "BOARDED" || got.Event.SeatID != "SEAT-002" {
		t.Fatalf("event = %#v, want deterministic boarding event for seat", got.Event)
	}
}

func TestBuildBoardingRejectsOccupiedUser(t *testing.T) {
	records := stateRecords{
		Seats: []airlockSeat{
			{SeatID: "SEAT-002", SeatNumber: 2, OccupantUserID: "USER-MAYA", Status: "OCCUPIED"},
		},
	}

	_, err := buildBoarding(records, "USER-MAYA", "maya@example.com", "maya", "Maya", 33600, mustTime(t, "2026-05-27T12:00:00Z"))
	if !errors.Is(err, ErrBoardingAlreadySeated) {
		t.Fatalf("err = %v, want ErrBoardingAlreadySeated", err)
	}
}

func TestBuildBoardingRejectsTakenHandle(t *testing.T) {
	records := stateRecords{
		Users: []airlockUser{
			{UserID: "USER-OTHER", Handle: "maya"},
		},
	}

	_, err := buildBoarding(records, "USER-MAYA", "maya@example.com", "maya", "Maya", 33600, mustTime(t, "2026-05-27T12:00:00Z"))
	if !errors.Is(err, ErrBoardingHandleTaken) {
		t.Fatalf("err = %v, want ErrBoardingHandleTaken", err)
	}
}

func TestBuildBoardingRejectsInvalidHandle(t *testing.T) {
	_, err := buildBoarding(stateRecords{}, "USER-MAYA", "", "!!", "Maya", 33600, mustTime(t, "2026-05-27T12:00:00Z"))
	if !errors.Is(err, ErrBoardingInvalidHandle) {
		t.Fatalf("err = %v, want ErrBoardingInvalidHandle", err)
	}
}

func TestBuildBoardingKeepsNewBoardersInThe100AfterPost100Reboard(t *testing.T) {
	now := mustTime(t, "2026-05-27T12:00:00Z")
	records := stateRecords{
		Seats: []airlockSeat{
			{SeatID: "SEAT-002", SeatNumber: 2, OccupantUserID: "USER-OLD", Status: "VACATED", VacatedAt: "2026-05-26T12:00:00Z"},
			{SeatID: "SEAT-101", SeatNumber: 101, OccupantUserID: "USER-OLD", Status: "OCCUPIED"},
		},
	}

	got, err := buildBoarding(records, "USER-MAYA", "maya@example.com", "maya", "Maya", 33600, now)
	if err != nil {
		t.Fatalf("buildBoarding returned error: %v", err)
	}
	if got.Seat.SeatNumber != 3 || got.Seat.TierPaid != 1 || got.Seat.PricePaidCents != boardingDisplayPriceCents {
		t.Fatalf("seat = %#v, want first-time boarder to continue THE 100 at seat 3", got.Seat)
	}
	if got.Event.Kind != "BOARDED" {
		t.Fatalf("event = %#v, want BOARDED for a first-time boarder", got.Event)
	}
}

func TestBuildBoardingRoutesReboardToPost100AndLinksVacatedSeat(t *testing.T) {
	now := mustTime(t, "2026-05-27T12:00:00Z")
	records := stateRecords{
		Users: []airlockUser{
			{UserID: "USER-MAYA", Handle: "maya"},
		},
		Seats: []airlockSeat{
			{SeatID: "SEAT-002", SeatNumber: 2, OccupantUserID: "USER-MAYA", Status: "VACATED", VacatedAt: "2026-06-20T12:00:00Z"},
			{SeatID: "SEAT-003", SeatNumber: 3, OccupantUserID: "USER-OTHER", Status: "OCCUPIED"},
		},
	}

	got, err := buildBoarding(records, "USER-MAYA", "maya@example.com", "maya", "Maya", 33600, now)
	if err != nil {
		t.Fatalf("buildBoarding returned error: %v", err)
	}
	if got.Seat.SeatNumber != 101 || got.Seat.SeatID != "SEAT-101" || got.Seat.TierPaid != 5 {
		t.Fatalf("seat = %#v, want reboard as post-100 seat 101 at tier 5", got.Seat)
	}
	if got.Seat.PricePaidCents != boardingDisplayPriceCents {
		t.Fatalf("seat = %#v, want flat display price on reboard", got.Seat)
	}
	if got.Seat.Cohort != "POST_100" {
		t.Fatalf("seat = %#v, want POST_100 cohort", got.Seat)
	}
	if got.Event.Kind != "RE_BOARDED" {
		t.Fatalf("event = %#v, want RE_BOARDED", got.Event)
	}
	if len(got.LinkedVacatedSeats) != 1 {
		t.Fatalf("linked seats = %#v, want one linked prior vacated seat", got.LinkedVacatedSeats)
	}
	linked := got.LinkedVacatedSeats[0]
	if linked.SeatID != "SEAT-002" || linked.ReboardedAsSeatID != "SEAT-101" || linked.ReboardedAsSeatLabel != "101" {
		t.Fatalf("linked seat = %#v, want prior seat linked to SEAT-101", linked)
	}
}

func TestBuildBoardingAssignsSequentialPost100Reboards(t *testing.T) {
	now := mustTime(t, "2026-05-27T12:00:00Z")
	records := stateRecords{
		Seats: []airlockSeat{
			{SeatID: "SEAT-002", SeatNumber: 2, OccupantUserID: "USER-MAYA", Status: "VACATED", VacatedAt: "2026-05-26T12:00:00Z"},
			{SeatID: "SEAT-101", SeatNumber: 101, OccupantUserID: "USER-OLD", Status: "OCCUPIED"},
		},
	}

	got, err := buildBoarding(records, "USER-MAYA", "maya@example.com", "maya", "Maya", 33600, now)
	if err != nil {
		t.Fatalf("buildBoarding returned error: %v", err)
	}
	if got.Seat.SeatNumber != 102 {
		t.Fatalf("seat = %#v, want next sequential post-100 seat", got.Seat)
	}
}

func TestNextAssignableSeatNumberContinuesThe100(t *testing.T) {
	now := mustTime(t, "2026-05-27T12:00:00Z")
	records := stateRecords{
		Seats: []airlockSeat{
			{SeatID: "SEAT-002", SeatNumber: 2, OccupantUserID: "USER-ONE", Status: "OCCUPIED"},
			{SeatID: "SEAT-003", SeatNumber: 3, OccupantUserID: "USER-TWO", Status: "OCCUPIED"},
		},
	}
	if got := nextAssignableSeatNumber(records, now); got != 4 {
		t.Fatalf("nextAssignableSeatNumber = %d, want 4", got)
	}
}

func TestBuildBoardingRejectsUnderpaidPlan(t *testing.T) {
	now := mustTime(t, "2026-05-27T12:00:00Z")

	// Next seat is in tier-01 ($42); an unmapped/0-cent plan must be rejected.
	if _, err := buildBoarding(launchStateRecords(), "USER-MAYA", "maya@example.com", "maya", "Maya", 0, now); !errors.Is(err, ErrBoardingPlanUnderpaid) {
		t.Fatalf("err = %v, want ErrBoardingPlanUnderpaid for tier-01 seat with 0 cents", err)
	}

	// A reboard routes to a post-100 seat that requires the flat $336 (33600).
	reboardRecords := stateRecords{
		Users: []airlockUser{{UserID: "USER-MAYA", Handle: "maya"}},
		Seats: []airlockSeat{
			{SeatID: "SEAT-002", SeatNumber: 2, OccupantUserID: "USER-MAYA", Status: "VACATED", VacatedAt: "2026-06-20T12:00:00Z"},
			{SeatID: "SEAT-003", SeatNumber: 3, OccupantUserID: "USER-OTHER", Status: "OCCUPIED"},
		},
	}
	if _, err := buildBoarding(reboardRecords, "USER-MAYA", "maya@example.com", "maya", "Maya", 16800, now); !errors.Is(err, ErrBoardingPlanUnderpaid) {
		t.Fatalf("err = %v, want ErrBoardingPlanUnderpaid for post-100 reboard with tier-03 plan", err)
	}

	got, err := buildBoarding(reboardRecords, "USER-MAYA", "maya@example.com", "maya", "Maya", 33600, now)
	if err != nil {
		t.Fatalf("buildBoarding returned error: %v", err)
	}
	if got.Seat.SeatNumber != 101 {
		t.Fatalf("seat = %#v, want post-100 reboard allowed with tier-04 plan", got.Seat)
	}
}

func TestPlanAmountCentsForSlug(t *testing.T) {
	cases := map[string]int{
		"tier-01": 4200,
		"tier-02": 8400,
		"tier-03": 16800,
		"tier-04": 33600,
		"boarder": 0,
		"":        0,
	}
	for slug, want := range cases {
		if got := planAmountCentsForSlug(slug); got != want {
			t.Errorf("planAmountCentsForSlug(%q) = %d, want %d", slug, got, want)
		}
	}
}
