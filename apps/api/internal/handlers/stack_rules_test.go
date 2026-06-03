package handlers

import (
	"errors"
	"testing"
)

func TestBuildStackUpdateNormalizesContactsForOccupiedSeat(t *testing.T) {
	now := mustTime(t, "2026-05-27T12:00:00Z")
	records := stateRecords{
		Users: []airlockUser{{UserID: "u1", Handle: "oskar"}},
		Seats: []airlockSeat{{SeatID: "SEAT-001", SeatNumber: 1, OccupantUserID: "u1", Status: "OCCUPIED"}},
	}

	got, err := buildStackUpdate(records, "u1", stackUpdateInput{
		CrewAlertEnabled:        true,
		CrewAlertContacts:       []string{"  cofounder@example.com  ", "cofounder@example.com", " Launch   chat "},
		TheWakeEnabled:          true,
		ReentryChallengeEnabled: true,
	}, now)
	if err != nil {
		t.Fatalf("buildStackUpdate returned error: %v", err)
	}

	if got.SeatID != "SEAT-001" || !got.CrewAlertEnabled || !got.TheWakeEnabled || got.ReentryChallengeEnabled {
		t.Fatalf("stack = %#v, want enabled v1 stacks and no first-seat re-entry challenge", got)
	}
	if len(got.CrewAlertContacts) != 2 || got.CrewAlertContacts[0] != "cofounder@example.com" || got.CrewAlertContacts[1] != "Launch chat" {
		t.Fatalf("contacts = %#v, want trimmed unique contacts", got.CrewAlertContacts)
	}
}

func TestBuildStackUpdateAllowsReentryChallengeAfterReboard(t *testing.T) {
	now := mustTime(t, "2026-07-27T12:00:00Z")
	records := stateRecords{
		Seats: []airlockSeat{
			{SeatID: "SEAT-004", SeatNumber: 4, OccupantUserID: "u1", Status: "VACATED", VacatedAt: "2026-06-27T12:00:00Z"},
			{SeatID: "SEAT-101", SeatNumber: 101, OccupantUserID: "u1", Status: "OCCUPIED"},
		},
	}

	got, err := buildStackUpdate(records, "u1", stackUpdateInput{ReentryChallengeEnabled: true}, now)
	if err != nil {
		t.Fatalf("buildStackUpdate returned error: %v", err)
	}

	if got.SeatID != "SEAT-101" || !got.ReentryChallengeEnabled {
		t.Fatalf("stack = %#v, want re-entry challenge enabled for re-boarded seat", got)
	}
}

func TestBuildStackUpdateRejectsCrewAlertWithoutContacts(t *testing.T) {
	records := stateRecords{
		Seats: []airlockSeat{{SeatID: "SEAT-001", SeatNumber: 1, OccupantUserID: "u1", Status: "OCCUPIED"}},
	}

	_, err := buildStackUpdate(records, "u1", stackUpdateInput{CrewAlertEnabled: true}, mustTime(t, "2026-05-27T12:00:00Z"))
	if !errors.Is(err, ErrStackInvalidContacts) {
		t.Fatalf("err = %v, want ErrStackInvalidContacts", err)
	}
}

func TestBuildStackUpdateRejectsMoreThanEightContacts(t *testing.T) {
	records := stateRecords{
		Seats: []airlockSeat{{SeatID: "SEAT-001", SeatNumber: 1, OccupantUserID: "u1", Status: "OCCUPIED"}},
	}

	_, err := buildStackUpdate(records, "u1", stackUpdateInput{
		CrewAlertEnabled:  true,
		CrewAlertContacts: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"},
	}, mustTime(t, "2026-05-27T12:00:00Z"))
	if !errors.Is(err, ErrStackInvalidContacts) {
		t.Fatalf("err = %v, want ErrStackInvalidContacts", err)
	}
}

func TestBuildStackUpdateRequiresOccupiedSeat(t *testing.T) {
	_, err := buildStackUpdate(stateRecords{}, "u1", stackUpdateInput{TheWakeEnabled: true}, mustTime(t, "2026-05-27T12:00:00Z"))
	if !errors.Is(err, ErrMissionSeatRequired) {
		t.Fatalf("err = %v, want ErrMissionSeatRequired", err)
	}
}
