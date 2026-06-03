package handlers

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

var (
	ErrMissionInvalidDeclaration    = errors.New("mission declaration is invalid")
	ErrMissionInvalidDeclarationURL = errors.New("mission declaration url is invalid")
	ErrMissionInvalidProofURL       = errors.New("proof url is invalid")
	ErrMissionSeatRequired          = errors.New("occupied seat required")
	ErrMissionActive                = errors.New("active mission already exists")
	ErrMissionNotFound              = errors.New("mission not found")
	ErrMissionForbidden             = errors.New("mission does not belong to user")
	ErrMissionAdminRequired         = errors.New("mission admin required")
	ErrMissionHatchClosed           = errors.New("hatch is already closed")
	ErrMissionStateConflict         = errors.New("mission state conflict")
)

const (
	minDeclarationLength = 12
	maxDeclarationLength = 180
	maxMissionURLLength  = 2048
)

type missionDeclarationMutation struct {
	Mission airlockMission
	Event   airlockEvent
}

type missionProofMutation struct {
	Mission airlockMission
	Event   airlockEvent
}

type missionReviewMutation struct {
	Mission           airlockMission
	Seat              airlockSeat
	Event             airlockEvent
	Handle            string
	CrewAlertContacts []string
	CrewAlertSent     bool
	CrewAlertError    string
	WakePosted        bool
	RedeemedMissions  []airlockMission
	RedeemedEvent     *airlockEvent
}

func buildMissionDeclaration(records stateRecords, userID string, declaration string, declarationURL string, now time.Time, missionID string, eventID string) (missionDeclarationMutation, error) {
	declaration, err := normalizeDeclaration(declaration)
	if err != nil {
		return missionDeclarationMutation{}, err
	}
	declarationURL, err = normalizeDeclarationURL(declarationURL)
	if err != nil {
		return missionDeclarationMutation{}, err
	}
	if records.Source == "dynamodb_empty_launch_seed" {
		return missionDeclarationMutation{}, ErrMissionSeatRequired
	}

	usersByID := make(map[string]airlockUser, len(records.Users))
	for _, user := range records.Users {
		usersByID[user.UserID] = user
	}
	seat, mission, hasMission, ok := currentSeatForUser(records, userID, now)
	if !ok {
		return missionDeclarationMutation{}, ErrMissionSeatRequired
	}
	if hasMission {
		status := missionEffectiveStatus(mission, now)
		if status == "DECLARED" || status == "PROOF_PENDING" {
			return missionDeclarationMutation{}, ErrMissionActive
		}
	}

	nowISO := formatInstant(now)
	newMission := airlockMission{
		MissionID:      missionID,
		SeatID:         seat.SeatID,
		Declaration:    declaration,
		DeclarationURL: declarationURL,
		DeclaredAt:     nowISO,
		DeadlineAt:     formatInstant(now.AddDate(0, 0, 30)),
		Status:         "DECLARED",
		Redeemed:       false,
	}
	event := airlockEvent{
		EventID:    eventID,
		Kind:       "DECLARED",
		UserID:     userID,
		SeatID:     seat.SeatID,
		MissionID:  missionID,
		OccurredAt: nowISO,
		Message:    fmt.Sprintf("@%s mission declared - %s", eventHandle(usersByID[userID], userID), declaration),
	}
	return missionDeclarationMutation{Mission: newMission, Event: event}, nil
}

func buildMissionProof(records stateRecords, userID string, missionID string, proofURL string, now time.Time, eventID string) (missionProofMutation, error) {
	proofURL, err := normalizeProofURL(proofURL)
	if err != nil {
		return missionProofMutation{}, err
	}

	usersByID := make(map[string]airlockUser, len(records.Users))
	for _, user := range records.Users {
		usersByID[user.UserID] = user
	}
	seatsByID := make(map[string]airlockSeat, len(records.Seats))
	for _, seat := range records.Seats {
		seatsByID[seat.SeatID] = seat
	}

	var mission airlockMission
	found := false
	for _, candidate := range records.Missions {
		if candidate.MissionID == missionID {
			mission = candidate
			found = true
			break
		}
	}
	if !found {
		return missionProofMutation{}, ErrMissionNotFound
	}

	seat, ok := seatsByID[mission.SeatID]
	if !ok || seat.OccupantUserID != userID {
		return missionProofMutation{}, ErrMissionForbidden
	}
	if deriveSeatStatus(seat, mission, true, now) != "OCCUPIED" {
		return missionProofMutation{}, ErrMissionSeatRequired
	}

	deadline, err := time.Parse(time.RFC3339, mission.DeadlineAt)
	if err != nil || !now.UTC().Before(deadline) {
		return missionProofMutation{}, ErrMissionHatchClosed
	}
	if missionEffectiveStatus(mission, now) != "DECLARED" {
		return missionProofMutation{}, ErrMissionStateConflict
	}

	nowISO := formatInstant(now)
	updated := mission
	updated.Status = "PROOF_PENDING"
	updated.ProofURL = proofURL
	updated.ProofSubmittedAt = nowISO

	event := airlockEvent{
		EventID:    eventID,
		Kind:       "PROOF_SUBMITTED",
		UserID:     userID,
		SeatID:     seat.SeatID,
		MissionID:  missionID,
		OccurredAt: nowISO,
		Message:    fmt.Sprintf("@%s proof attached - %s", eventHandle(usersByID[userID], userID), proofURL),
	}
	return missionProofMutation{Mission: updated, Event: event}, nil
}

func buildMissionConfirmation(records stateRecords, missionID string, reviewerID string, now time.Time, eventID string) (missionReviewMutation, error) {
	mission, seat, user, err := pendingMissionForReview(records, missionID)
	if err != nil {
		return missionReviewMutation{}, err
	}

	nowISO := formatInstant(now)
	updated := mission
	updated.Status = "CONFIRMED"
	updated.ConfirmedAt = nowISO

	handle := eventHandle(user, seat.OccupantUserID)
	event := airlockEvent{
		EventID:    eventID,
		Kind:       "CONFIRMED",
		UserID:     seat.OccupantUserID,
		SeatID:     seat.SeatID,
		MissionID:  mission.MissionID,
		OccurredAt: nowISO,
		Message:    fmt.Sprintf("@%s confirmed - %s", handle, mission.Declaration),
	}
	var redeemedMissions []airlockMission
	if stackBySeatID(records.Stacks, seat.SeatID).ReentryChallengeEnabled {
		redeemedMissions = buildRedeemedMissions(records, updated, seat.OccupantUserID)
	}
	var redeemedEvent *airlockEvent
	if len(redeemedMissions) > 0 {
		redeemedEvent = &airlockEvent{
			EventID:    eventID + "-REDEEMED",
			Kind:       "REDEEMED",
			UserID:     seat.OccupantUserID,
			SeatID:     seat.SeatID,
			MissionID:  mission.MissionID,
			OccurredAt: nowISO,
			Message:    fmt.Sprintf("@%s redeemed %d prior airlock stamp(s)", handle, len(redeemedMissions)),
		}
	}
	return missionReviewMutation{
		Mission:          updated,
		Seat:             seat,
		Event:            event,
		Handle:           handle,
		RedeemedMissions: redeemedMissions,
		RedeemedEvent:    redeemedEvent,
	}, nil
}

func buildMissionRejection(records stateRecords, missionID string, reviewerID string, now time.Time, eventID string) (missionReviewMutation, error) {
	mission, seat, user, err := pendingMissionForReview(records, missionID)
	if err != nil {
		return missionReviewMutation{}, err
	}

	nowISO := formatInstant(now)
	updatedMission := mission
	updatedMission.Status = "AIRLOCKED"
	updatedMission.AirlockedAt = nowISO

	updatedSeat := seat
	updatedSeat.Status = "VACATED"
	updatedSeat.VacatedAt = nowISO
	updatedSeat.VacatedOnMissionID = mission.MissionID

	handle := eventHandle(user, seat.OccupantUserID)
	event := airlockEvent{
		EventID:    eventID,
		Kind:       "AIRLOCKED",
		UserID:     seat.OccupantUserID,
		SeatID:     seat.SeatID,
		MissionID:  mission.MissionID,
		OccurredAt: nowISO,
		Message:    fmt.Sprintf("@%s · %s · %s · airlocked", handle, mission.Declaration, nowISO),
	}
	mutation := missionReviewMutation{
		Mission: updatedMission,
		Seat:    updatedSeat,
		Event:   event,
		Handle:  handle,
	}
	return withMissionReviewStackConsequences(mutation, stackBySeatID(records.Stacks, seat.SeatID)), nil
}

func withMissionReviewStackConsequences(mutation missionReviewMutation, stack airlockStack) missionReviewMutation {
	mutation.WakePosted = stack.TheWakeEnabled
	mutation.Event.WakePosted = stack.TheWakeEnabled
	if stack.CrewAlertEnabled {
		mutation.CrewAlertContacts = append([]string{}, stack.CrewAlertContacts...)
	}
	return mutation
}

func stackBySeatID(stacks []airlockStack, seatID string) airlockStack {
	for _, stack := range stacks {
		if stack.SeatID == seatID {
			return stack
		}
	}
	return airlockStack{}
}

func buildRedeemedMissions(records stateRecords, confirmedMission airlockMission, userID string) []airlockMission {
	if userID == "" {
		return nil
	}
	seatIDs := make(map[string]bool)
	for _, seat := range records.Seats {
		if seat.OccupantUserID == userID {
			seatIDs[seat.SeatID] = true
		}
	}
	if len(seatIDs) == 0 {
		return nil
	}

	missions := make([]airlockMission, 0)
	foundCurrent := false
	for _, mission := range records.Missions {
		if !seatIDs[mission.SeatID] {
			continue
		}
		if mission.MissionID == confirmedMission.MissionID {
			mission = confirmedMission
			foundCurrent = true
		}
		missions = append(missions, mission)
	}
	if !foundCurrent {
		missions = append(missions, confirmedMission)
	}
	sortMissionsByDeclaration(missions)

	currentIndex := -1
	latestAirlockIndex := -1
	for i, mission := range missions {
		if mission.MissionID == confirmedMission.MissionID {
			currentIndex = i
		}
		if currentIndex == -1 && normalizeStatus(mission.Status) == "AIRLOCKED" {
			latestAirlockIndex = i
		}
	}
	if currentIndex < 0 || latestAirlockIndex < 0 || currentIndex-latestAirlockIndex < 2 {
		return nil
	}

	last := missions[currentIndex]
	previous := missions[currentIndex-1]
	if normalizeStatus(last.Status) != "CONFIRMED" || normalizeStatus(previous.Status) != "CONFIRMED" {
		return nil
	}

	redeemed := make([]airlockMission, 0)
	for i := 0; i <= latestAirlockIndex; i++ {
		if normalizeStatus(missions[i].Status) != "AIRLOCKED" || missions[i].Redeemed {
			continue
		}
		mission := missions[i]
		mission.Redeemed = true
		redeemed = append(redeemed, mission)
	}
	return redeemed
}

func sortMissionsByDeclaration(missions []airlockMission) {
	sort.SliceStable(missions, func(i, j int) bool {
		return missionSortKey(missions[i]) < missionSortKey(missions[j])
	})
}

func pendingMissionForReview(records stateRecords, missionID string) (airlockMission, airlockSeat, airlockUser, error) {
	usersByID := make(map[string]airlockUser, len(records.Users))
	for _, user := range records.Users {
		usersByID[user.UserID] = user
	}
	seatsByID := make(map[string]airlockSeat, len(records.Seats))
	for _, seat := range records.Seats {
		seatsByID[seat.SeatID] = seat
	}

	var mission airlockMission
	found := false
	for _, candidate := range records.Missions {
		if candidate.MissionID == missionID {
			mission = candidate
			found = true
			break
		}
	}
	if !found {
		return airlockMission{}, airlockSeat{}, airlockUser{}, ErrMissionNotFound
	}
	if normalizeStatus(mission.Status) != "PROOF_PENDING" || mission.ProofURL == "" {
		return airlockMission{}, airlockSeat{}, airlockUser{}, ErrMissionStateConflict
	}
	seat, ok := seatsByID[mission.SeatID]
	if !ok || normalizeStatus(seat.Status) != "OCCUPIED" || seat.VacatedAt != "" {
		return airlockMission{}, airlockSeat{}, airlockUser{}, ErrMissionSeatRequired
	}
	return mission, seat, usersByID[seat.OccupantUserID], nil
}

func currentSeatForUser(records stateRecords, userID string, now time.Time) (airlockSeat, airlockMission, bool, bool) {
	missionsBySeatID := latestMissionBySeat(records.Missions)
	for _, seat := range records.Seats {
		if seat.OccupantUserID != userID {
			continue
		}
		mission, hasMission := missionsBySeatID[seat.SeatID]
		if deriveSeatStatus(seat, mission, hasMission, now) == "OCCUPIED" {
			return seat, mission, hasMission, true
		}
	}
	return airlockSeat{}, airlockMission{}, false, false
}

func normalizeDeclaration(value string) (string, error) {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len(value) < minDeclarationLength || len(value) > maxDeclarationLength {
		return "", ErrMissionInvalidDeclaration
	}
	return value, nil
}

func normalizeProofURL(value string) (string, error) {
	return normalizeMissionURL(value, ErrMissionInvalidProofURL)
}

func normalizeDeclarationURL(value string) (string, error) {
	return normalizeMissionURL(value, ErrMissionInvalidDeclarationURL)
}

func normalizeMissionURL(value string, invalid error) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) == 0 || len(value) > maxMissionURLLength {
		return "", invalid
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return "", invalid
	}
	switch parsed.Scheme {
	case "https", "http":
	default:
		return "", invalid
	}
	parsed.Fragment = ""
	return parsed.String(), nil
}

func eventHandle(user airlockUser, fallback string) string {
	if user.Handle != "" {
		return user.Handle
	}
	return fallback
}

func formatInstant(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
