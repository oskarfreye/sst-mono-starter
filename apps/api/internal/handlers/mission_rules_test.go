package handlers

import (
	"context"
	"errors"
	"testing"
)

func TestBuildMissionDeclarationCreatesSequentialMission(t *testing.T) {
	now := mustTime(t, "2026-05-27T12:00:00Z")
	records := stateRecords{
		Source: "test",
		Users: []airlockUser{
			{UserID: "u1", Handle: "oskar"},
		},
		Seats: []airlockSeat{
			{SeatID: "SEAT-001", SeatNumber: 1, Cohort: "THE_100", OccupantUserID: "u1", Status: "OCCUPIED"},
		},
		Missions: []airlockMission{
			{MissionID: "MISSION-OLD", SeatID: "SEAT-001", Declaration: "Ship checkout", Status: "CONFIRMED", DeclaredAt: "2026-04-01T12:00:00Z", DeadlineAt: "2026-05-01T12:00:00Z", ConfirmedAt: "2026-04-20T12:00:00Z"},
		},
	}

	got, err := buildMissionDeclaration(records, "u1", " Ship a paid onboarding release  ", "https://example.com/app#pricing", now, "MISSION-NEW", "EVENT-NEW")
	if err != nil {
		t.Fatalf("buildMissionDeclaration returned error: %v", err)
	}

	if got.Mission.Status != "DECLARED" || got.Mission.MissionID != "MISSION-NEW" {
		t.Fatalf("mission = %#v, want new declared mission", got.Mission)
	}
	if got.Mission.DeclarationURL != "https://example.com/app" {
		t.Fatalf("declaration url = %q, want normalized URL without fragment", got.Mission.DeclarationURL)
	}
	if got.Mission.DeclaredAt != "2026-05-27T12:00:00Z" || got.Mission.DeadlineAt != "2026-06-26T12:00:00Z" {
		t.Fatalf("mission timing = %#v, want 30-day deadline", got.Mission)
	}
	if got.Event.Kind != "DECLARED" || got.Event.UserID != "u1" || got.Event.SeatID != "SEAT-001" {
		t.Fatalf("event = %#v, want declaration event", got.Event)
	}
}

func TestBuildMissionDeclarationRejectsActiveMission(t *testing.T) {
	now := mustTime(t, "2026-05-27T12:00:00Z")
	records := stateRecords{
		Source: "test",
		Users:  []airlockUser{{UserID: "u1", Handle: "oskar"}},
		Seats:  []airlockSeat{{SeatID: "SEAT-001", SeatNumber: 1, OccupantUserID: "u1", Status: "OCCUPIED"}},
		Missions: []airlockMission{
			{MissionID: "MISSION-001", SeatID: "SEAT-001", Declaration: "Ship checkout", Status: "DECLARED", DeclaredAt: "2026-05-20T12:00:00Z", DeadlineAt: "2026-06-19T12:00:00Z"},
		},
	}

	_, err := buildMissionDeclaration(records, "u1", "Ship a paid onboarding release", "https://example.com/app", now, "MISSION-NEW", "EVENT-NEW")
	if !errors.Is(err, ErrMissionActive) {
		t.Fatalf("err = %v, want ErrMissionActive", err)
	}
}

func TestBuildMissionDeclarationRejectsMissingDeclarationURL(t *testing.T) {
	records := stateRecords{
		Source: "test",
		Users:  []airlockUser{{UserID: "u1", Handle: "oskar"}},
		Seats:  []airlockSeat{{SeatID: "SEAT-001", SeatNumber: 1, OccupantUserID: "u1", Status: "OCCUPIED"}},
	}

	_, err := buildMissionDeclaration(records, "u1", "Ship a paid onboarding release", "", mustTime(t, "2026-05-27T12:00:00Z"), "MISSION-NEW", "EVENT-NEW")
	if !errors.Is(err, ErrMissionInvalidDeclarationURL) {
		t.Fatalf("err = %v, want ErrMissionInvalidDeclarationURL", err)
	}
}

func TestBuildMissionProofMovesDeclaredMissionToProofPending(t *testing.T) {
	now := mustTime(t, "2026-05-27T12:00:00Z")
	records := stateRecords{
		Source: "test",
		Users:  []airlockUser{{UserID: "u1", Handle: "oskar"}},
		Seats:  []airlockSeat{{SeatID: "SEAT-001", SeatNumber: 1, OccupantUserID: "u1", Status: "OCCUPIED"}},
		Missions: []airlockMission{
			{MissionID: "MISSION-001", SeatID: "SEAT-001", Declaration: "Ship checkout", Status: "DECLARED", DeclaredAt: "2026-05-20T12:00:00Z", DeadlineAt: "2026-06-19T12:00:00Z"},
		},
	}

	got, err := buildMissionProof(records, "u1", "MISSION-001", "https://example.com/proof#details", now, "EVENT-002")
	if err != nil {
		t.Fatalf("buildMissionProof returned error: %v", err)
	}

	if got.Mission.Status != "PROOF_PENDING" || got.Mission.ProofURL != "https://example.com/proof" {
		t.Fatalf("mission = %#v, want proof pending without fragment", got.Mission)
	}
	if got.Mission.ProofSubmittedAt != "2026-05-27T12:00:00Z" {
		t.Fatalf("proof submitted at = %q, want now", got.Mission.ProofSubmittedAt)
	}
	if got.Event.Kind != "PROOF_SUBMITTED" || got.Event.MissionID != "MISSION-001" {
		t.Fatalf("event = %#v, want proof event", got.Event)
	}
}

func TestBuildMissionProofRejectsLateProofAtDeadline(t *testing.T) {
	now := mustTime(t, "2026-06-19T12:00:00Z")
	records := stateRecords{
		Source: "test",
		Users:  []airlockUser{{UserID: "u1", Handle: "oskar"}},
		Seats:  []airlockSeat{{SeatID: "SEAT-001", SeatNumber: 1, OccupantUserID: "u1", Status: "OCCUPIED"}},
		Missions: []airlockMission{
			{MissionID: "MISSION-001", SeatID: "SEAT-001", Declaration: "Ship checkout", Status: "DECLARED", DeclaredAt: "2026-05-20T12:00:00Z", DeadlineAt: "2026-06-19T12:00:00Z"},
		},
	}

	_, err := buildMissionProof(records, "u1", "MISSION-001", "https://example.com/proof", now, "EVENT-002")
	if !errors.Is(err, ErrMissionHatchClosed) {
		t.Fatalf("err = %v, want ErrMissionHatchClosed at deadline", err)
	}
}

func TestBuildMissionConfirmationConfirmsProofPendingMission(t *testing.T) {
	now := mustTime(t, "2026-05-28T12:00:00Z")
	records := stateRecords{
		Users: []airlockUser{{UserID: "u1", Handle: "oskar"}},
		Seats: []airlockSeat{{SeatID: "SEAT-001", SeatNumber: 1, OccupantUserID: "u1", Status: "OCCUPIED"}},
		Missions: []airlockMission{
			{
				MissionID:        "MISSION-001",
				SeatID:           "SEAT-001",
				Declaration:      "Ship checkout",
				Status:           "PROOF_PENDING",
				DeclaredAt:       "2026-05-20T12:00:00Z",
				DeadlineAt:       "2026-06-19T12:00:00Z",
				ProofURL:         "https://example.com/proof",
				ProofSubmittedAt: "2026-05-27T12:00:00Z",
			},
		},
	}

	got, err := buildMissionConfirmation(records, "MISSION-001", "admin", now, "EVENT-003")
	if err != nil {
		t.Fatalf("buildMissionConfirmation returned error: %v", err)
	}

	if got.Mission.Status != "CONFIRMED" || got.Mission.ConfirmedAt != "2026-05-28T12:00:00Z" {
		t.Fatalf("mission = %#v, want confirmed at now", got.Mission)
	}
	if got.Seat.Status != "OCCUPIED" {
		t.Fatalf("seat = %#v, want occupied seat unchanged", got.Seat)
	}
	if got.Event.Kind != "CONFIRMED" || got.Event.MissionID != "MISSION-001" {
		t.Fatalf("event = %#v, want confirmation event", got.Event)
	}
}

func TestBuildMissionConfirmationDoesNotRedeemPriorAirlocksWithoutOptIn(t *testing.T) {
	now := mustTime(t, "2026-07-22T12:00:00Z")
	records := stateRecords{
		Users: []airlockUser{{UserID: "u1", Handle: "oskar"}},
		Seats: []airlockSeat{
			{SeatID: "SEAT-001", SeatNumber: 1, OccupantUserID: "u1", Status: "VACATED", VacatedAt: "2026-06-20T12:00:00Z"},
			{SeatID: "SEAT-101", SeatNumber: 101, OccupantUserID: "u1", Status: "OCCUPIED"},
		},
		Missions: []airlockMission{
			{
				MissionID:   "MISSION-AIRLOCKED",
				SeatID:      "SEAT-001",
				Declaration: "Missed checkout",
				Status:      "AIRLOCKED",
				DeclaredAt:  "2026-05-20T12:00:00Z",
				DeadlineAt:  "2026-06-19T12:00:00Z",
				AirlockedAt: "2026-06-20T12:00:00Z",
			},
			{
				MissionID:   "MISSION-WIN-1",
				SeatID:      "SEAT-101",
				Declaration: "Ship comeback one",
				Status:      "CONFIRMED",
				DeclaredAt:  "2026-06-21T12:00:00Z",
				DeadlineAt:  "2026-07-21T12:00:00Z",
				ConfirmedAt: "2026-07-01T12:00:00Z",
			},
			{
				MissionID:        "MISSION-WIN-2",
				SeatID:           "SEAT-101",
				Declaration:      "Ship comeback two",
				Status:           "PROOF_PENDING",
				DeclaredAt:       "2026-07-02T12:00:00Z",
				DeadlineAt:       "2026-08-01T12:00:00Z",
				ProofURL:         "https://example.com/proof",
				ProofSubmittedAt: "2026-07-20T12:00:00Z",
			},
		},
	}

	got, err := buildMissionConfirmation(records, "MISSION-WIN-2", "admin", now, "EVENT-CONFIRMED")
	if err != nil {
		t.Fatalf("buildMissionConfirmation returned error: %v", err)
	}

	if len(got.RedeemedMissions) != 0 || got.RedeemedEvent != nil {
		t.Fatalf("redemption = %#v / %#v, want no automatic redemption without opt-in", got.RedeemedMissions, got.RedeemedEvent)
	}
}

func TestBuildMissionConfirmationRedeemsPriorAirlocksAfterOptedInTwoConfirmedMissions(t *testing.T) {
	now := mustTime(t, "2026-07-22T12:00:00Z")
	records := stateRecords{
		Users: []airlockUser{{UserID: "u1", Handle: "oskar"}},
		Seats: []airlockSeat{
			{SeatID: "SEAT-001", SeatNumber: 1, OccupantUserID: "u1", Status: "VACATED", VacatedAt: "2026-06-20T12:00:00Z"},
			{SeatID: "SEAT-101", SeatNumber: 101, OccupantUserID: "u1", Status: "OCCUPIED"},
		},
		Stacks: []airlockStack{{SeatID: "SEAT-101", ReentryChallengeEnabled: true}},
		Missions: []airlockMission{
			{
				MissionID:   "MISSION-AIRLOCKED",
				SeatID:      "SEAT-001",
				Declaration: "Missed checkout",
				Status:      "AIRLOCKED",
				DeclaredAt:  "2026-05-20T12:00:00Z",
				DeadlineAt:  "2026-06-19T12:00:00Z",
				AirlockedAt: "2026-06-20T12:00:00Z",
			},
			{
				MissionID:   "MISSION-WIN-1",
				SeatID:      "SEAT-101",
				Declaration: "Ship comeback one",
				Status:      "CONFIRMED",
				DeclaredAt:  "2026-06-21T12:00:00Z",
				DeadlineAt:  "2026-07-21T12:00:00Z",
				ConfirmedAt: "2026-07-01T12:00:00Z",
			},
			{
				MissionID:        "MISSION-WIN-2",
				SeatID:           "SEAT-101",
				Declaration:      "Ship comeback two",
				Status:           "PROOF_PENDING",
				DeclaredAt:       "2026-07-02T12:00:00Z",
				DeadlineAt:       "2026-08-01T12:00:00Z",
				ProofURL:         "https://example.com/proof",
				ProofSubmittedAt: "2026-07-20T12:00:00Z",
			},
		},
	}

	got, err := buildMissionConfirmation(records, "MISSION-WIN-2", "admin", now, "EVENT-CONFIRMED")
	if err != nil {
		t.Fatalf("buildMissionConfirmation returned error: %v", err)
	}

	if len(got.RedeemedMissions) != 1 {
		t.Fatalf("len(RedeemedMissions) = %d, want 1", len(got.RedeemedMissions))
	}
	if got.RedeemedMissions[0].MissionID != "MISSION-AIRLOCKED" || !got.RedeemedMissions[0].Redeemed {
		t.Fatalf("redeemed mission = %#v, want prior airlock redeemed", got.RedeemedMissions[0])
	}
	if got.RedeemedEvent == nil || got.RedeemedEvent.Kind != "REDEEMED" || got.RedeemedEvent.EventID != "EVENT-CONFIRMED-REDEEMED" {
		t.Fatalf("redeemed event = %#v, want REDEEMED side event", got.RedeemedEvent)
	}
}

func TestBuildMissionRejectionAirlocksProofPendingMission(t *testing.T) {
	now := mustTime(t, "2026-05-28T12:00:00Z")
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
			{
				MissionID:        "MISSION-001",
				SeatID:           "SEAT-001",
				Declaration:      "Ship checkout",
				Status:           "PROOF_PENDING",
				DeclaredAt:       "2026-05-20T12:00:00Z",
				DeadlineAt:       "2026-06-19T12:00:00Z",
				ProofURL:         "https://example.com/proof",
				ProofSubmittedAt: "2026-05-27T12:00:00Z",
			},
		},
	}

	got, err := buildMissionRejection(records, "MISSION-001", "admin", now, "EVENT-004")
	if err != nil {
		t.Fatalf("buildMissionRejection returned error: %v", err)
	}

	if got.Mission.Status != "AIRLOCKED" || got.Mission.AirlockedAt != "2026-05-28T12:00:00Z" {
		t.Fatalf("mission = %#v, want airlocked at now", got.Mission)
	}
	if got.Seat.Status != "VACATED" || got.Seat.VacatedOnMissionID != "MISSION-001" {
		t.Fatalf("seat = %#v, want vacated on rejected mission", got.Seat)
	}
	if got.Event.Kind != "AIRLOCKED" || got.Event.MissionID != "MISSION-001" {
		t.Fatalf("event = %#v, want airlock event", got.Event)
	}
	if got.Event.Message != "@oskar · Ship checkout · 2026-05-28T12:00:00Z · airlocked" {
		t.Fatalf("event message = %q, want preset wake template", got.Event.Message)
	}
	if got.Handle != "oskar" || !got.WakePosted || !got.Event.WakePosted {
		t.Fatalf("stack consequence metadata = %#v, want handle and wake post", got)
	}
	if len(got.CrewAlertContacts) != 2 || got.CrewAlertContacts[0] != "cofounder@example.com" {
		t.Fatalf("crew alert contacts = %#v, want configured contacts", got.CrewAlertContacts)
	}
}

func TestMissionRejectionSendsCrewAlertAfterStoreMutation(t *testing.T) {
	sender := &recordingCrewAlertSender{}
	handler := NewMissionHandler(MissionHandlerConfig{CrewAlertSender: sender})
	mutation := missionReviewMutation{
		Mission: airlockMission{
			MissionID:   "MISSION-001",
			SeatID:      "SEAT-001",
			Declaration: "Ship checkout",
			Status:      "AIRLOCKED",
		},
		Seat:              airlockSeat{SeatID: "SEAT-001"},
		Event:             airlockEvent{EventID: "EVENT-004", OccurredAt: "2026-05-28T12:00:00Z"},
		Handle:            "oskar",
		CrewAlertContacts: []string{"cofounder@example.com"},
	}

	handler.sendMissionCrewAlert(context.Background(), &mutation)

	if len(sender.alerts) != 1 {
		t.Fatalf("alerts = %#v, want one crew alert delivery", sender.alerts)
	}
	alert := sender.alerts[0]
	if alert.EventID != "EVENT-004" || alert.Handle != "oskar" || alert.MissionID != "MISSION-001" || alert.Mission != "Ship checkout" {
		t.Fatalf("alert = %#v, want mission rejection crew alert payload", alert)
	}
	if !mutation.CrewAlertSent || mutation.CrewAlertError != "" {
		t.Fatalf("mutation = %#v, want sent crew alert state", mutation)
	}
}

func TestBuildMissionReviewRejectsNonPendingMission(t *testing.T) {
	records := stateRecords{
		Users: []airlockUser{{UserID: "u1", Handle: "oskar"}},
		Seats: []airlockSeat{{SeatID: "SEAT-001", SeatNumber: 1, OccupantUserID: "u1", Status: "OCCUPIED"}},
		Missions: []airlockMission{
			{MissionID: "MISSION-001", SeatID: "SEAT-001", Declaration: "Ship checkout", Status: "DECLARED"},
		},
	}

	_, err := buildMissionConfirmation(records, "MISSION-001", "admin", mustTime(t, "2026-05-28T12:00:00Z"), "EVENT-003")
	if !errors.Is(err, ErrMissionStateConflict) {
		t.Fatalf("err = %v, want ErrMissionStateConflict", err)
	}
}
