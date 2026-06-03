package handlers

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrStackInvalidContacts = errors.New("crew alert contacts are invalid")
)

const (
	maxCrewAlertContacts = 8
	maxStackContactLen   = 160
)

type stackUpdateInput struct {
	CrewAlertEnabled        bool
	CrewAlertContacts       []string
	TheWakeEnabled          bool
	ReentryChallengeEnabled bool
}

func buildStackUpdate(records stateRecords, userID string, input stackUpdateInput, now time.Time) (airlockStack, error) {
	seat, _, _, ok := currentSeatForUser(records, userID, now)
	if !ok {
		return airlockStack{}, ErrMissionSeatRequired
	}

	contacts, err := normalizeCrewAlertContacts(input.CrewAlertEnabled, input.CrewAlertContacts)
	if err != nil {
		return airlockStack{}, err
	}
	reentryChallengeEnabled := input.ReentryChallengeEnabled && userHasVacatedSeat(records, userID)
	return airlockStack{
		SeatID:                  seat.SeatID,
		CrewAlertEnabled:        input.CrewAlertEnabled,
		CrewAlertContacts:       contacts,
		TheWakeEnabled:          input.TheWakeEnabled,
		TheWakeCrosspost:        []string{},
		ReentryChallengeEnabled: reentryChallengeEnabled,
	}, nil
}

func stackForUser(records stateRecords, userID string, now time.Time) (airlockStack, error) {
	seat, _, _, ok := currentSeatForUser(records, userID, now)
	if !ok {
		return airlockStack{}, ErrMissionSeatRequired
	}
	for _, stack := range records.Stacks {
		if stack.SeatID == seat.SeatID {
			return stack, nil
		}
	}
	return airlockStack{SeatID: seat.SeatID}, nil
}

func normalizeCrewAlertContacts(enabled bool, contacts []string) ([]string, error) {
	if !enabled {
		return []string{}, nil
	}
	out := make([]string, 0, len(contacts))
	seen := make(map[string]struct{}, len(contacts))
	for _, contact := range contacts {
		contact = strings.Join(strings.Fields(strings.TrimSpace(contact)), " ")
		if contact == "" {
			continue
		}
		if len(contact) > maxStackContactLen {
			return nil, ErrStackInvalidContacts
		}
		key := strings.ToLower(contact)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, contact)
	}
	if len(out) == 0 || len(out) > maxCrewAlertContacts {
		return nil, ErrStackInvalidContacts
	}
	return out, nil
}
