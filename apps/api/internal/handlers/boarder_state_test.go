package handlers

import "testing"

func TestBuildBoarderStateReturnsCurrentMissionAndStack(t *testing.T) {
	now := mustTime(t, "2026-05-27T12:00:00Z")
	records := stateRecords{
		Seats:  []airlockSeat{{SeatID: "SEAT-002", SeatNumber: 2, Cohort: "THE_100", OccupantUserID: "u1", Status: "OCCUPIED", BoardedAt: "2026-05-20T12:00:00Z", PricePaidCents: 4200}},
		Stacks: []airlockStack{{SeatID: "SEAT-002", CrewAlertEnabled: true, CrewAlertContacts: []string{"cofounder@example.com"}, TheWakeEnabled: true}},
		Missions: []airlockMission{
			{MissionID: "MISSION-001", SeatID: "SEAT-002", Declaration: "Ship checkout", Status: "DECLARED", DeclaredAt: "2026-05-20T12:00:00Z", DeadlineAt: "2026-06-19T12:00:00Z"},
		},
	}

	got := buildBoarderState(records, "u1", now)

	if !got.HasSeat || got.Seat.SeatID != "SEAT-002" {
		t.Fatalf("seat = %#v, want current occupied seat", got.Seat)
	}
	if !got.HasMission || got.Mission.MissionID != "MISSION-001" {
		t.Fatalf("mission = %#v, want latest mission", got.Mission)
	}
	if got.CanDeclare || !got.CanSubmit {
		t.Fatalf("CanDeclare/CanSubmit = %v/%v, want false/true for declared mission", got.CanDeclare, got.CanSubmit)
	}
	if !got.Stack.CrewAlertEnabled || !got.Stack.TheWakeEnabled || len(got.Stack.CrewAlertContacts) != 1 {
		t.Fatalf("stack = %#v, want configured stack", got.Stack)
	}
}

func TestBuildBoarderStateAllowsDeclareAfterConfirmedMission(t *testing.T) {
	now := mustTime(t, "2026-05-27T12:00:00Z")
	records := stateRecords{
		Seats: []airlockSeat{{SeatID: "SEAT-002", SeatNumber: 2, OccupantUserID: "u1", Status: "OCCUPIED"}},
		Missions: []airlockMission{
			{MissionID: "MISSION-001", SeatID: "SEAT-002", Declaration: "Ship checkout", Status: "CONFIRMED", DeclaredAt: "2026-05-20T12:00:00Z", DeadlineAt: "2026-06-19T12:00:00Z", ConfirmedAt: "2026-05-25T12:00:00Z"},
		},
	}

	got := buildBoarderState(records, "u1", now)

	if !got.CanDeclare || got.CanSubmit {
		t.Fatalf("CanDeclare/CanSubmit = %v/%v, want true/false after confirmation", got.CanDeclare, got.CanSubmit)
	}
}

func TestBuildBoarderStateMarksReboardedUser(t *testing.T) {
	now := mustTime(t, "2026-07-22T12:00:00Z")
	records := stateRecords{
		Seats: []airlockSeat{
			{SeatID: "SEAT-002", SeatNumber: 2, OccupantUserID: "u1", Status: "VACATED", VacatedAt: "2026-06-20T12:00:00Z"},
			{SeatID: "SEAT-101", SeatNumber: 101, OccupantUserID: "u1", Status: "OCCUPIED"},
		},
	}

	got := buildBoarderState(records, "u1", now)

	if !got.HasSeat || !got.HasReboarded {
		t.Fatalf("state = %#v, want active reboarded user", got)
	}
}

func TestBuildBoarderStateExposesDeclarationDeadlineForNewSeat(t *testing.T) {
	now := mustTime(t, "2026-05-27T12:00:00Z")
	records := stateRecords{
		Seats: []airlockSeat{{SeatID: "SEAT-002", SeatNumber: 2, OccupantUserID: "u1", Status: "OCCUPIED", BoardedAt: "2026-05-27T11:00:00Z"}},
	}

	got := buildBoarderState(records, "u1", now)

	if !got.HasSeat || got.HasMission || !got.CanDeclare {
		t.Fatalf("state = %#v, want seated user ready to declare", got)
	}
	if got.DeclareBy != "2026-05-28T11:00:00Z" {
		t.Fatalf("DeclareBy = %q, want 24h after boarding", got.DeclareBy)
	}
}

func TestBuildBoarderStateHandlesUnseatedUser(t *testing.T) {
	got := buildBoarderState(stateRecords{}, "u1", mustTime(t, "2026-05-27T12:00:00Z"))

	if got.HasSeat || got.HasMission || got.CanDeclare || got.CanSubmit {
		t.Fatalf("state = %#v, want empty unseated state", got)
	}
}
