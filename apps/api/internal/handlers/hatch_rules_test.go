package handlers

import "testing"

func TestBuildHatchMutationsAirlocksOverdueMission(t *testing.T) {
	now := mustTime(t, "2026-06-20T12:00:00Z")
	records := stateRecords{
		Source: "test",
		Users:  []airlockUser{{UserID: "u1", Handle: "oskar"}},
		Seats:  []airlockSeat{{SeatID: "SEAT-001", SeatNumber: 1, OccupantUserID: "u1", Status: "OCCUPIED"}},
		Missions: []airlockMission{
			{MissionID: "MISSION-001", SeatID: "SEAT-001", Declaration: "Ship checkout", Status: "DECLARED", DeclaredAt: "2026-05-20T12:00:00Z", DeadlineAt: "2026-06-19T12:00:00Z"},
		},
	}

	got := buildHatchMutations(records, now, fixedID("EVENT-001"))

	if len(got) != 1 {
		t.Fatalf("len(mutations) = %d, want 1", len(got))
	}
	if got[0].Mission.Status != "AIRLOCKED" || got[0].Mission.AirlockedAt != "2026-06-20T12:00:00Z" {
		t.Fatalf("mission = %#v, want airlocked at now", got[0].Mission)
	}
	if got[0].Seat.Status != "VACATED" || got[0].Seat.VacatedOnMissionID != "MISSION-001" {
		t.Fatalf("seat = %#v, want vacated on mission", got[0].Seat)
	}
	if got[0].Event.Kind != "AIRLOCKED" || got[0].Event.EventID != "EVENT-001" {
		t.Fatalf("event = %#v, want airlock event", got[0].Event)
	}
}

func TestBuildHatchMutationsCarriesEnabledStackConsequences(t *testing.T) {
	now := mustTime(t, "2026-06-20T12:00:00Z")
	records := stateRecords{
		Users: []airlockUser{{UserID: "u1", Handle: "oskar"}},
		Seats: []airlockSeat{{SeatID: "SEAT-001", SeatNumber: 1, OccupantUserID: "u1", Status: "OCCUPIED"}},
		Stacks: []airlockStack{{
			SeatID:            "SEAT-001",
			CrewAlertEnabled:  true,
			CrewAlertContacts: []string{"cofounder@example.com", "Launch chat"},
			TheWakeEnabled:    true,
		}},
		Missions: []airlockMission{
			{MissionID: "MISSION-001", SeatID: "SEAT-001", Declaration: "Ship checkout", Status: "DECLARED", DeclaredAt: "2026-05-20T12:00:00Z", DeadlineAt: "2026-06-19T12:00:00Z"},
		},
	}

	got := buildHatchMutations(records, now, fixedID("EVENT-001"))

	if len(got) != 1 {
		t.Fatalf("len(mutations) = %d, want 1", len(got))
	}
	if !got[0].WakePosted || !got[0].Event.WakePosted {
		t.Fatalf("WakePosted = %#v / event %#v, want durable wake flag", got[0].WakePosted, got[0].Event.WakePosted)
	}
	if len(got[0].CrewAlertContacts) != 2 || got[0].CrewAlertContacts[0] != "cofounder@example.com" {
		t.Fatalf("CrewAlertContacts = %#v, want configured contacts", got[0].CrewAlertContacts)
	}
}

func TestBuildHatchMutationsSkipsProofBeforeDeadline(t *testing.T) {
	now := mustTime(t, "2026-06-19T11:59:59Z")
	records := stateRecords{
		Source: "test",
		Users:  []airlockUser{{UserID: "u1", Handle: "oskar"}},
		Seats:  []airlockSeat{{SeatID: "SEAT-001", SeatNumber: 1, OccupantUserID: "u1", Status: "OCCUPIED"}},
		Missions: []airlockMission{
			{MissionID: "MISSION-001", SeatID: "SEAT-001", Declaration: "Ship checkout", Status: "PROOF_PENDING", DeclaredAt: "2026-05-20T12:00:00Z", DeadlineAt: "2026-06-19T12:00:00Z"},
		},
	}

	got := buildHatchMutations(records, now, fixedID("EVENT-001"))

	if len(got) != 0 {
		t.Fatalf("len(mutations) = %d, want 0 before deadline", len(got))
	}
}

func TestBuildHatchMutationsSkipsConfirmedMission(t *testing.T) {
	now := mustTime(t, "2026-06-20T12:00:00Z")
	records := stateRecords{
		Source: "test",
		Users:  []airlockUser{{UserID: "u1", Handle: "oskar"}},
		Seats:  []airlockSeat{{SeatID: "SEAT-001", SeatNumber: 1, OccupantUserID: "u1", Status: "OCCUPIED"}},
		Missions: []airlockMission{
			{MissionID: "MISSION-001", SeatID: "SEAT-001", Declaration: "Ship checkout", Status: "CONFIRMED", DeclaredAt: "2026-05-20T12:00:00Z", DeadlineAt: "2026-06-19T12:00:00Z", ConfirmedAt: "2026-06-01T12:00:00Z"},
		},
	}

	got := buildHatchMutations(records, now, fixedID("EVENT-001"))

	if len(got) != 0 {
		t.Fatalf("len(mutations) = %d, want 0 for confirmed mission", len(got))
	}
}

func TestBuildHatchMutationsAirlocksBoarderWhoMissesDeclarationWindow(t *testing.T) {
	now := mustTime(t, "2026-05-28T12:00:00Z")
	records := stateRecords{
		Source: "test",
		Users:  []airlockUser{{UserID: "u1", Handle: "maya"}},
		Seats:  []airlockSeat{{SeatID: "SEAT-002", SeatNumber: 2, OccupantUserID: "u1", Status: "OCCUPIED", BoardedAt: "2026-05-27T11:00:00Z"}},
	}

	got := buildHatchMutations(records, now, fixedIDs(map[string]string{
		"MISSION": "MISSION-NO-DECLARE",
		"EVENT":   "EVENT-NO-DECLARE",
	}))

	if len(got) != 1 {
		t.Fatalf("len(mutations) = %d, want 1", len(got))
	}
	if !got[0].CreateMission {
		t.Fatalf("CreateMission = false, want true for missed declaration")
	}
	if got[0].Mission.MissionID != "MISSION-NO-DECLARE" || got[0].Mission.Status != "AIRLOCKED" {
		t.Fatalf("mission = %#v, want synthetic airlocked mission", got[0].Mission)
	}
	if got[0].Mission.Declaration != "No mission declared within 24 hours" {
		t.Fatalf("declaration = %q, want missed declaration text", got[0].Mission.Declaration)
	}
	if got[0].Mission.DeadlineAt != "2026-05-28T11:00:00Z" || got[0].Mission.AirlockedAt != "2026-05-28T12:00:00Z" {
		t.Fatalf("mission timing = %#v, want 24h declaration deadline and airlocked now", got[0].Mission)
	}
	if got[0].Seat.Status != "VACATED" || got[0].Seat.VacatedOnMissionID != "MISSION-NO-DECLARE" {
		t.Fatalf("seat = %#v, want vacated on synthetic mission", got[0].Seat)
	}
	if got[0].Event.EventID != "EVENT-NO-DECLARE" || got[0].Event.MissionID != "MISSION-NO-DECLARE" {
		t.Fatalf("event = %#v, want synthetic airlock event", got[0].Event)
	}
	if scanned := hatchScanCount(records); scanned != 1 {
		t.Fatalf("hatchScanCount = %d, want declaration-window seat counted", scanned)
	}
}

func TestBuildHatchMutationsSkipsBoarderStillInsideDeclarationWindow(t *testing.T) {
	now := mustTime(t, "2026-05-28T10:59:59Z")
	records := stateRecords{
		Source: "test",
		Users:  []airlockUser{{UserID: "u1", Handle: "maya"}},
		Seats:  []airlockSeat{{SeatID: "SEAT-002", SeatNumber: 2, OccupantUserID: "u1", Status: "OCCUPIED", BoardedAt: "2026-05-27T11:00:00Z"}},
	}

	got := buildHatchMutations(records, now, fixedIDs(map[string]string{
		"MISSION": "MISSION-NO-DECLARE",
		"EVENT":   "EVENT-NO-DECLARE",
	}))

	if len(got) != 0 {
		t.Fatalf("len(mutations) = %d, want 0 inside declaration window", len(got))
	}
}

func fixedID(id string) func(string) string {
	return func(string) string { return id }
}

func fixedIDs(values map[string]string) func(string) string {
	return func(prefix string) string {
		if value := values[prefix]; value != "" {
			return value
		}
		return prefix + "-ID"
	}
}
