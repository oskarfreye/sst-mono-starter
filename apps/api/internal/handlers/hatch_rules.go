package handlers

import (
	"fmt"
	"time"
)

type hatchMutation struct {
	Mission           airlockMission
	Seat              airlockSeat
	Event             airlockEvent
	Handle            string
	CrewAlertContacts []string
	WakePosted        bool
	CreateMission     bool
}

func buildHatchMutations(records stateRecords, now time.Time, newID func(prefix string) string) []hatchMutation {
	usersByID := make(map[string]airlockUser, len(records.Users))
	for _, user := range records.Users {
		usersByID[user.UserID] = user
	}

	seatsByID := make(map[string]airlockSeat, len(records.Seats))
	for _, seat := range records.Seats {
		seatsByID[seat.SeatID] = seat
	}
	stacksBySeatID := make(map[string]airlockStack, len(records.Stacks))
	for _, stack := range records.Stacks {
		stacksBySeatID[stack.SeatID] = stack
	}
	missionsBySeatID := latestMissionBySeat(records.Missions)

	nowISO := formatInstant(now)
	out := make([]hatchMutation, 0)
	for _, mission := range records.Missions {
		status := normalizeStatus(mission.Status)
		if status != "DECLARED" && status != "PROOF_PENDING" {
			continue
		}

		deadline, err := time.Parse(time.RFC3339, mission.DeadlineAt)
		if err != nil || !now.UTC().After(deadline) {
			continue
		}

		seat, ok := seatsByID[mission.SeatID]
		if !ok || normalizeStatus(seat.Status) != "OCCUPIED" || seat.VacatedAt != "" {
			continue
		}

		user := usersByID[seat.OccupantUserID]
		airlockedMission := mission
		airlockedMission.Status = "AIRLOCKED"
		airlockedMission.AirlockedAt = nowISO

		vacatedSeat := seat
		vacatedSeat.Status = "VACATED"
		vacatedSeat.VacatedAt = nowISO
		vacatedSeat.VacatedOnMissionID = mission.MissionID

		handle := eventHandle(user, seat.OccupantUserID)
		event := airlockEvent{
			EventID:    newID("EVENT"),
			Kind:       "AIRLOCKED",
			UserID:     seat.OccupantUserID,
			SeatID:     seat.SeatID,
			MissionID:  mission.MissionID,
			OccurredAt: nowISO,
			Message:    fmt.Sprintf("@%s · %s · %s · airlocked", handle, mission.Declaration, nowISO),
		}
		mutation := hatchMutation{
			Mission: airlockedMission,
			Seat:    vacatedSeat,
			Event:   event,
			Handle:  handle,
		}
		out = append(out, withStackConsequences(mutation, stacksBySeatID[seat.SeatID]))
	}

	for _, seat := range records.Seats {
		if _, hasMission := missionsBySeatID[seat.SeatID]; hasMission {
			continue
		}
		if normalizeStatus(seat.Status) != "OCCUPIED" || seat.VacatedAt != "" || seat.OccupantUserID == "" {
			continue
		}
		boardedAt, err := time.Parse(time.RFC3339, seat.BoardedAt)
		if err != nil {
			continue
		}
		declarationDeadline := boardedAt.Add(24 * time.Hour)
		if now.UTC().Before(declarationDeadline) {
			continue
		}

		user := usersByID[seat.OccupantUserID]
		handle := eventHandle(user, seat.OccupantUserID)
		missionID := newID("MISSION")
		declaration := "No mission declared within 24 hours"
		airlockedMission := airlockMission{
			MissionID:   missionID,
			SeatID:      seat.SeatID,
			Declaration: declaration,
			DeclaredAt:  formatInstant(boardedAt),
			DeadlineAt:  formatInstant(declarationDeadline),
			Status:      "AIRLOCKED",
			AirlockedAt: nowISO,
		}
		vacatedSeat := seat
		vacatedSeat.Status = "VACATED"
		vacatedSeat.VacatedAt = nowISO
		vacatedSeat.VacatedOnMissionID = missionID
		event := airlockEvent{
			EventID:    newID("EVENT"),
			Kind:       "AIRLOCKED",
			UserID:     seat.OccupantUserID,
			SeatID:     seat.SeatID,
			MissionID:  missionID,
			OccurredAt: nowISO,
			Message:    fmt.Sprintf("@%s · %s · %s · airlocked", handle, declaration, nowISO),
		}
		mutation := hatchMutation{
			Mission:       airlockedMission,
			Seat:          vacatedSeat,
			Event:         event,
			Handle:        handle,
			CreateMission: true,
		}
		out = append(out, withStackConsequences(mutation, stacksBySeatID[seat.SeatID]))
	}
	return out
}

func hatchScanCount(records stateRecords) int {
	missionsBySeatID := latestMissionBySeat(records.Missions)
	scanned := len(records.Missions)
	for _, seat := range records.Seats {
		if _, hasMission := missionsBySeatID[seat.SeatID]; hasMission {
			continue
		}
		if normalizeStatus(seat.Status) == "OCCUPIED" && seat.VacatedAt == "" && seat.OccupantUserID != "" {
			scanned++
		}
	}
	return scanned
}

func withStackConsequences(mutation hatchMutation, stack airlockStack) hatchMutation {
	mutation.WakePosted = stack.TheWakeEnabled
	mutation.Event.WakePosted = stack.TheWakeEnabled
	if stack.CrewAlertEnabled {
		mutation.CrewAlertContacts = append([]string{}, stack.CrewAlertContacts...)
	}
	return mutation
}
