package handlers

import (
	"testing"
	"time"
)

func TestBuildStateSnapshotUsesRecordsForNextSeatAndNearestHatch(t *testing.T) {
	now := mustTime(t, "2026-05-20T12:00:00Z")
	records := stateRecords{
		Source: "test",
		Users: []airlockUser{
			{UserID: "u1", Handle: "oskar", DisplayName: "Oskar"},
			{UserID: "u2", Handle: "maya", DisplayName: "Maya"},
		},
		Seats: []airlockSeat{
			{SeatID: "SEAT-001", SeatNumber: 1, Cohort: "THE_100", OccupantUserID: "u1", Status: "OCCUPIED", PricePaidCents: 4200},
			{SeatID: "SEAT-002", SeatNumber: 2, Cohort: "THE_100", OccupantUserID: "u2", Status: "OCCUPIED", PricePaidCents: 4200},
		},
		Missions: []airlockMission{
			{MissionID: "m1", SeatID: "SEAT-001", Declaration: "Ship Airlock", DeclarationURL: "https://theairlock.space", Status: "DECLARED", DeclaredAt: "2026-05-13T18:00:00Z", DeadlineAt: "2026-06-12T18:00:00Z"},
			{MissionID: "m2", SeatID: "SEAT-002", Declaration: "Ship billing", DeclarationURL: "https://billing.example.com", Status: "PROOF_PENDING", DeclaredAt: "2026-05-14T18:00:00Z", DeadlineAt: "2026-05-22T18:00:00Z"},
		},
		Events: []airlockEvent{
			{EventID: "e1", Kind: "DECLARED", UserID: "u2", SeatID: "SEAT-002", MissionID: "m2", OccurredAt: "2026-05-14T18:00:00Z"},
		},
	}

	got := buildStateSnapshot(records, now)

	if got.Source != "test" {
		t.Fatalf("source = %q, want test", got.Source)
	}
	if got.CohortSummary.SeatsOccupied != 2 || got.CohortSummary.SeatsOpen != 98 || got.CohortSummary.SeatsVacated != 0 {
		t.Fatalf("cohort summary = %#v, want 2 occupied, 98 open, 0 vacated", got.CohortSummary)
	}
	if got.NextSeat.SeatLabel != "03" || got.NextSeat.Price.Display != "$42" {
		t.Fatalf("next seat = %#v, want seat 03 at $42", got.NextSeat)
	}
	if got.NearestHatch == nil || got.NearestHatch.Handle != "maya" || got.NearestHatch.Status != "PROOF_PENDING" {
		t.Fatalf("nearest hatch = %#v, want maya proof pending", got.NearestHatch)
	}
	if got.Metrics.ProofPending != 1 || got.Metrics.ActiveMissions != 2 {
		t.Fatalf("metrics = %#v, want proof pending 1 and active missions 2", got.Metrics)
	}
	if got.Seats[1].Mission == nil || got.Seats[1].Mission.DeclarationURL != "https://billing.example.com" {
		t.Fatalf("seat 02 mission = %#v, want declaration URL exposed", got.Seats[1].Mission)
	}
}

func TestBuildStateSnapshotDoesNotInventAirlockBeforeHatchPersists(t *testing.T) {
	now := mustTime(t, "2026-06-13T12:00:00Z")
	records := launchStateRecords()

	got := buildStateSnapshot(records, now)

	if got.CohortSummary.SeatsOccupied != 1 || got.CohortSummary.SeatsVacated != 0 {
		t.Fatalf("cohort summary = %#v, want seat 01 still occupied until hatch persists", got.CohortSummary)
	}
	if got.Seats[0].Status != "OCCUPIED" || got.Seats[0].Mission == nil || got.Seats[0].Mission.Status != "DECLARED" {
		t.Fatalf("seat 01 = %#v, want stored status before hatch persists", got.Seats[0])
	}
}

func TestBuildStateSnapshotExposesOnlyPublicStackSummary(t *testing.T) {
	now := mustTime(t, "2026-05-20T12:00:00Z")
	records := stateRecords{
		Users: []airlockUser{{UserID: "u1", Handle: "oskar", DisplayName: "Oskar"}},
		Seats: []airlockSeat{{SeatID: "SEAT-001", SeatNumber: 1, Cohort: "THE_100", OccupantUserID: "u1", Status: "OCCUPIED", PricePaidCents: 4200}},
		Stacks: []airlockStack{{
			SeatID:                  "SEAT-001",
			CrewAlertEnabled:        true,
			CrewAlertContacts:       []string{"cofounder@example.com", "Launch chat"},
			TheWakeEnabled:          true,
			ReentryChallengeEnabled: true,
		}},
		Missions: []airlockMission{
			{MissionID: "m1", SeatID: "SEAT-001", Declaration: "Ship Airlock", Status: "DECLARED", DeclaredAt: "2026-05-13T18:00:00Z", DeadlineAt: "2026-06-12T18:00:00Z"},
		},
	}

	got := buildStateSnapshot(records, now)

	if got.Seats[0].Stack == nil {
		t.Fatalf("stack = nil, want public stack summary")
	}
	if !got.Seats[0].Stack.CrewAlertEnabled || got.Seats[0].Stack.CrewAlertContactCount != 2 || !got.Seats[0].Stack.TheWakeEnabled || !got.Seats[0].Stack.ReentryChallengeEnabled {
		t.Fatalf("stack = %#v, want enabled summary with contact count", got.Seats[0].Stack)
	}
}

func TestBuildStateSnapshotIncludesAirlockEvents(t *testing.T) {
	now := mustTime(t, "2026-06-20T12:00:00Z")
	records := stateRecords{
		Users: []airlockUser{{UserID: "u1", Handle: "oskar"}},
		Seats: []airlockSeat{{SeatID: "SEAT-001", SeatNumber: 1, Cohort: "THE_100", OccupantUserID: "u1", Status: "VACATED", VacatedAt: "2026-06-20T12:00:00Z"}},
		Missions: []airlockMission{
			{MissionID: "m1", SeatID: "SEAT-001", Declaration: "Ship Airlock", Status: "AIRLOCKED", DeclaredAt: "2026-05-13T18:00:00Z", DeadlineAt: "2026-06-12T18:00:00Z", AirlockedAt: "2026-06-20T12:00:00Z"},
		},
		Events: []airlockEvent{
			{EventID: "e-declared", Kind: "DECLARED", UserID: "u1", SeatID: "SEAT-001", MissionID: "m1", OccurredAt: "2026-05-13T18:00:00Z"},
			{EventID: "e-airlocked", Kind: "AIRLOCKED", UserID: "u1", SeatID: "SEAT-001", MissionID: "m1", OccurredAt: "2026-06-20T12:00:00Z", WakePosted: true},
		},
	}

	got := buildStateSnapshot(records, now)

	if len(got.AirlockEvents) != 1 {
		t.Fatalf("len(airlock_events) = %d, want 1", len(got.AirlockEvents))
	}
	if got.AirlockEvents[0].EventID != "e-airlocked" || got.AirlockEvents[0].Kind != "AIRLOCKED" {
		t.Fatalf("airlock event = %#v, want AIRLOCKED event", got.AirlockEvents[0])
	}
	if !got.AirlockEvents[0].WakePosted {
		t.Fatalf("airlock event = %#v, want wake_posted flag", got.AirlockEvents[0])
	}
	if len(got.ActiveSeats) != 1 || len(got.VacatedSeats) != 0 {
		t.Fatalf("active/vacated seats = %#v/%#v, want fresh airlock in active for 24h", got.ActiveSeats, got.VacatedSeats)
	}
}

func TestBuildStateSnapshotIncludesMissionHistory(t *testing.T) {
	now := mustTime(t, "2026-07-22T12:00:00Z")
	records := stateRecords{
		Users: []airlockUser{{UserID: "u1", Handle: "oskar"}},
		Seats: []airlockSeat{{SeatID: "SEAT-001", SeatNumber: 1, Cohort: "THE_100", OccupantUserID: "u1", Status: "OCCUPIED"}},
		Missions: []airlockMission{
			{MissionID: "m1", SeatID: "SEAT-001", Declaration: "Ship first", Status: "CONFIRMED", DeclaredAt: "2026-05-13T18:00:00Z", DeadlineAt: "2026-06-12T18:00:00Z", ConfirmedAt: "2026-06-01T12:00:00Z"},
			{MissionID: "m2", SeatID: "SEAT-001", Declaration: "Miss second", Status: "AIRLOCKED", DeclaredAt: "2026-06-13T18:00:00Z", DeadlineAt: "2026-07-13T18:00:00Z", AirlockedAt: "2026-07-14T12:00:00Z", Redeemed: true},
		},
	}

	got := buildStateSnapshot(records, now)

	if len(got.MissionHistory) != 2 {
		t.Fatalf("len(mission_history) = %d, want 2", len(got.MissionHistory))
	}
	if got.MissionHistory[0].MissionID != "m2" || got.MissionHistory[0].Status != "AIRLOCKED" || !got.MissionHistory[0].Redeemed {
		t.Fatalf("first mission history entry = %#v, want newest redeemed airlock", got.MissionHistory[0])
	}
	if got.MissionHistory[1].MissionID != "m1" || got.MissionHistory[1].Status != "CONFIRMED" {
		t.Fatalf("second mission history entry = %#v, want older confirmed mission", got.MissionHistory[1])
	}
}

func TestBuildStateSnapshotExposesReboardHistory(t *testing.T) {
	now := mustTime(t, "2026-07-22T12:00:00Z")
	records := stateRecords{
		Users: []airlockUser{{UserID: "u1", Handle: "maya"}},
		Seats: []airlockSeat{
			{
				SeatID:               "SEAT-002",
				SeatNumber:           2,
				Cohort:               "THE_100",
				OccupantUserID:       "u1",
				Status:               "VACATED",
				BoardedAt:            "2026-05-13T18:00:00Z",
				VacatedAt:            "2026-06-20T12:00:00Z",
				VacatedOnMissionID:   "m1",
				ReboardedAsSeatID:    "SEAT-101",
				ReboardedAsSeatLabel: "101",
			},
			{
				SeatID:         "SEAT-101",
				SeatNumber:     101,
				Cohort:         "POST_100",
				OccupantUserID: "u1",
				Status:         "OCCUPIED",
				BoardedAt:      "2026-07-21T12:00:00Z",
			},
		},
		Missions: []airlockMission{
			{MissionID: "m1", SeatID: "SEAT-002", Declaration: "Miss first", Status: "AIRLOCKED", DeclaredAt: "2026-05-13T18:00:00Z", DeadlineAt: "2026-06-12T18:00:00Z", AirlockedAt: "2026-06-20T12:00:00Z"},
		},
		Events: []airlockEvent{
			{EventID: "e-reboard", Kind: "RE_BOARDED", UserID: "u1", SeatID: "SEAT-101", OccurredAt: "2026-07-21T12:00:00Z"},
		},
	}

	got := buildStateSnapshot(records, now)

	if len(got.VacatedSeats) != 1 {
		t.Fatalf("vacated seats = %#v, want one vacated seat", got.VacatedSeats)
	}
	if len(got.ActiveSeats) != 1 || got.ActiveSeats[0].SeatID != "SEAT-101" || got.ActiveSeats[0].SeatNumber != 101 {
		t.Fatalf("active seats = %#v, want post-100 reboard seat exposed", got.ActiveSeats)
	}
	vacated := got.VacatedSeats[0]
	if vacated.VacatedAt != "2026-06-20T12:00:00Z" || vacated.ReboardedAsSeatID != "SEAT-101" || vacated.ReboardedAsSeatLabel != "101" {
		t.Fatalf("vacated seat = %#v, want reboard fields exposed", vacated)
	}
	if got.RecentEvents[0].Kind != "RE_BOARDED" || got.RecentEvents[0].DisplayLabel != "SEAT 101 - RE-BOARDED" {
		t.Fatalf("recent event = %#v, want reboard event label", got.RecentEvents[0])
	}
}

func TestBuildStateSnapshotNextSeatContinuesThe100AfterPost100Reboard(t *testing.T) {
	now := mustTime(t, "2026-07-22T12:00:00Z")
	records := stateRecords{
		Users: []airlockUser{{UserID: "u1", Handle: "maya"}},
		Seats: []airlockSeat{
			{SeatID: "SEAT-002", SeatNumber: 2, Cohort: "THE_100", OccupantUserID: "u1", Status: "VACATED", VacatedAt: "2026-06-20T12:00:00Z"},
			{SeatID: "SEAT-101", SeatNumber: 101, Cohort: "POST_100", OccupantUserID: "u1", Status: "OCCUPIED", BoardedAt: "2026-07-21T12:00:00Z"},
		},
	}

	got := buildStateSnapshot(records, now)

	if got.NextSeat.SeatNumber != 3 || got.NextSeat.Cohort != "THE_100" || got.NextSeat.Price.Cents != 4200 {
		t.Fatalf("next seat = %#v, want first-time boarders to continue THE 100 at seat 3", got.NextSeat)
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}
