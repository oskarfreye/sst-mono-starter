package handlers

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	launchDeclaredAt = "2026-05-13T18:00:00Z"
	launchDeadlineAt = "2026-06-12T18:00:00Z"
)

type stateRecords struct {
	Source   string
	Users    []airlockUser
	Seats    []airlockSeat
	Stacks   []airlockStack
	Missions []airlockMission
	Events   []airlockEvent
}

type airlockUser struct {
	UserID       string
	Handle       string
	DisplayName  string
	Email        string
	AvatarSeed   string
	Bio          string
	AvatarURL    string
	AvatarStatus string
}

type airlockSeat struct {
	SeatID               string
	SeatNumber           int
	Cohort               string
	OccupantUserID       string
	Status               string
	TierPaid             int
	PricePaidCents       int
	BoardedAt            string
	VacatedAt            string
	VacatedOnMissionID   string
	ReboardedAsSeatID    string
	ReboardedAsSeatLabel string
}

type airlockMission struct {
	MissionID        string
	SeatID           string
	Declaration      string
	DeclarationURL   string
	DeclaredAt       string
	DeadlineAt       string
	Status           string
	ProofURL         string
	ProofSubmittedAt string
	ConfirmedAt      string
	AirlockedAt      string
	Redeemed         bool
}

type airlockStack struct {
	SeatID                  string
	DonationEnabled         bool
	DonationAmount          int
	DonationTarget          string
	CrewAlertEnabled        bool
	CrewAlertContacts       []string
	TheWakeEnabled          bool
	TheWakeCrosspost        []string
	ReentryChallengeEnabled bool
}

type airlockEvent struct {
	EventID    string
	Kind       string
	UserID     string
	SeatID     string
	MissionID  string
	OccurredAt string
	Message    string
	WakePosted bool
}

func launchStateRecords() stateRecords {
	return stateRecords{
		Source: "launch_fallback",
		Users: []airlockUser{
			{
				UserID:      "USER-OSKAR",
				Handle:      "oskar",
				DisplayName: "Oskar",
				AvatarSeed:  "oskar",
			},
		},
		Seats: []airlockSeat{
			{
				SeatID:         "SEAT-001",
				SeatNumber:     1,
				Cohort:         "THE_100",
				OccupantUserID: "USER-OSKAR",
				Status:         "OCCUPIED",
				TierPaid:       1,
				PricePaidCents: 4200,
				BoardedAt:      launchDeclaredAt,
			},
		},
		Stacks: []airlockStack{
			{
				SeatID: "SEAT-001",
			},
		},
		Missions: []airlockMission{
			{
				MissionID:      "MISSION-001",
				SeatID:         "SEAT-001",
				Declaration:    "Ship Airlock v1.0 + Clerk billing",
				DeclarationURL: "https://theairlock.space",
				Status:         "DECLARED",
				DeclaredAt:     launchDeclaredAt,
				DeadlineAt:     launchDeadlineAt,
			},
		},
		Events: []airlockEvent{
			{
				EventID:    "EVENT-001",
				Kind:       "BOARDED",
				UserID:     "USER-OSKAR",
				SeatID:     "SEAT-001",
				OccurredAt: launchDeclaredAt,
				Message:    "@oskar boarded THE 100 - seat 01",
			},
			{
				EventID:    "EVENT-002",
				Kind:       "DECLARED",
				UserID:     "USER-OSKAR",
				SeatID:     "SEAT-001",
				MissionID:  "MISSION-001",
				OccurredAt: launchDeclaredAt,
				Message:    "@oskar mission declared - Ship Airlock v1.0 + Clerk billing",
			},
		},
	}
}

func launchStateRecordsForUserID(userID string) stateRecords {
	records := launchStateRecords()
	if userID == "" {
		return records
	}
	if len(records.Users) > 0 {
		records.Users[0].UserID = userID
	}
	for i := range records.Seats {
		if records.Seats[i].SeatID == "SEAT-001" {
			records.Seats[i].OccupantUserID = userID
		}
	}
	for i := range records.Events {
		if records.Events[i].UserID == "USER-OSKAR" {
			records.Events[i].UserID = userID
		}
	}
	return records
}

func buildStateSnapshot(records stateRecords, now time.Time) stateSnapshot {
	if records.Source == "" {
		records.Source = "dynamodb"
	}

	usersByID := make(map[string]airlockUser, len(records.Users))
	for _, user := range records.Users {
		usersByID[user.UserID] = user
	}

	missionsBySeatID := latestMissionBySeat(records.Missions)
	stacksBySeatID := make(map[string]airlockStack, len(records.Stacks))
	for _, stack := range records.Stacks {
		stacksBySeatID[stack.SeatID] = stack
	}
	eventsByID := make(map[string]airlockEvent, len(records.Events))
	for _, event := range records.Events {
		eventsByID[event.EventID] = event
	}

	seatsByNumber := make(map[int]airlockSeat, len(records.Seats))
	for _, seat := range records.Seats {
		if seat.SeatNumber <= 0 {
			continue
		}
		seatsByNumber[seat.SeatNumber] = seat
	}

	gridSeats := make([]stateSeat, 0, 100)
	activeSeats := make([]stateSeat, 0)
	vacatedSeats := make([]stateSeat, 0)
	the100OccupiedCount := 0
	the100VacatedCount := 0
	for seatNumber := 1; seatNumber <= 100; seatNumber++ {
		seatRecord, ok := seatsByNumber[seatNumber]
		if !ok {
			price := priceForSeat(seatNumber)
			gridSeats = append(gridSeats, stateSeat{
				SeatID:     fmt.Sprintf("SEAT-%03d", seatNumber),
				SeatNumber: seatNumber,
				SeatLabel:  seatLabel(seatNumber),
				Cohort:     "THE_100",
				Status:     "OPEN",
				NextPrice:  &price,
			})
			continue
		}

		user := usersByID[seatRecord.OccupantUserID]
		mission, hasMission := missionsBySeatID[seatRecord.SeatID]
		derivedStatus := deriveSeatStatus(seatRecord, mission, hasMission, now)
		gridSeat := toStateSeat(seatRecord, user, mission, hasMission, stacksBySeatID[seatRecord.SeatID], derivedStatus, now)
		gridSeats = append(gridSeats, gridSeat)
		if derivedStatus == "VACATED" {
			the100VacatedCount++
			if shouldListVacatedSeatAsActive(gridSeat, now) {
				activeSeats = append(activeSeats, gridSeat)
			} else {
				vacatedSeats = append(vacatedSeats, gridSeat)
			}
		} else if derivedStatus == "OCCUPIED" {
			the100OccupiedCount++
			activeSeats = append(activeSeats, gridSeat)
		}
	}

	post100SeatNumbers := make([]int, 0)
	for seatNumber := range seatsByNumber {
		if seatNumber > 100 {
			post100SeatNumbers = append(post100SeatNumbers, seatNumber)
		}
	}
	sort.Ints(post100SeatNumbers)
	for _, seatNumber := range post100SeatNumbers {
		seatRecord := seatsByNumber[seatNumber]
		user := usersByID[seatRecord.OccupantUserID]
		mission, hasMission := missionsBySeatID[seatRecord.SeatID]
		derivedStatus := deriveSeatStatus(seatRecord, mission, hasMission, now)
		stateSeat := toStateSeat(seatRecord, user, mission, hasMission, stacksBySeatID[seatRecord.SeatID], derivedStatus, now)
		gridSeats = append(gridSeats, stateSeat)
		if derivedStatus == "VACATED" {
			if shouldListVacatedSeatAsActive(stateSeat, now) {
				activeSeats = append(activeSeats, stateSeat)
			} else {
				vacatedSeats = append(vacatedSeats, stateSeat)
			}
		} else if derivedStatus == "OCCUPIED" {
			activeSeats = append(activeSeats, stateSeat)
		}
	}

	occupiedCount := the100OccupiedCount
	vacatedCount := the100VacatedCount
	openCount := 100 - occupiedCount - vacatedCount
	if openCount < 0 {
		openCount = 0
	}

	nextSeatNumber := nextAssignableSeatNumber(records, now)
	nextSeatCohort := "THE_100"
	if nextSeatNumber > 100 {
		nextSeatCohort = "POST_100"
	}
	nextSeatPrice := priceForSeat(nextSeatNumber)
	tier := tierForSeat(nextSeatNumber)

	return stateSnapshot{
		Snapshot: "live",
		Source:   records.Source,
		CohortSummary: stateCohortSummary{
			Cohort:        "THE_100",
			TotalSeats:    100,
			SeatsOccupied: occupiedCount,
			SeatsOpen:     openCount,
			SeatsVacated:  vacatedCount,
		},
		Seats:        gridSeats,
		ActiveSeats:  activeSeats,
		VacatedSeats: vacatedSeats,
		NextSeat: stateNextSeat{
			SeatNumber: nextSeatNumber,
			SeatLabel:  seatLabel(nextSeatNumber),
			Cohort:     nextSeatCohort,
			Price:      nextSeatPrice,
		},
		PriceTiers:   priceTierProgress(records.Seats),
		TierProgress: tierProgressFor(records.Seats, tier),
		NearestHatch: nearestHatch(activeSeats),
		Metrics: stateMetrics{
			SeatsOccupied:             occupiedCount,
			SeatsOpen:                 openCount,
			SeatsVacated:              vacatedCount,
			SeatsTotal:                100,
			ActiveMissions:            countMissions(records.Missions, now, "DECLARED", "PROOF_PENDING"),
			MissionsDeclared:          len(records.Missions),
			ProofPending:              countMissions(records.Missions, now, "PROOF_PENDING"),
			LaunchesConfirmedThisWeek: countConfirmedThisWeek(records.Missions, now),
			LaunchesConfirmedLifetime: countMissions(records.Missions, now, "CONFIRMED"),
			AirlocksLifetime:          countMissions(records.Missions, now, "AIRLOCKED"),
			RecentEvents:              len(records.Events),
			NextTierOpens: stateNextTier{
				Tier:         nextTierAfter(tier).ID,
				Price:        priceForTier(nextTierAfter(tier).ID),
				InSeats:      seatsUntilNextTier(nextSeatNumber),
				AtSeatNumber: nextTierAfter(tier).Start,
				AtSeatLabel:  seatLabel(nextTierAfter(tier).Start),
			},
		},
		MissionHistory: stateMissionHistoryFor(records.Missions, records.Seats, usersByID, now),
		RecentEvents:   stateEvents(records.Events, usersByID, records.Seats, records.Missions),
		AirlockEvents:  stateEventsByKind(records.Events, usersByID, records.Seats, records.Missions, "AIRLOCKED"),
	}
}

func latestMissionBySeat(missions []airlockMission) map[string]airlockMission {
	out := make(map[string]airlockMission)
	for _, mission := range missions {
		if mission.SeatID == "" {
			continue
		}
		current, ok := out[mission.SeatID]
		if !ok || missionSortKey(mission) > missionSortKey(current) {
			out[mission.SeatID] = mission
		}
	}
	return out
}

func missionSortKey(mission airlockMission) string {
	if mission.DeclaredAt != "" {
		return mission.DeclaredAt
	}
	return mission.MissionID
}

func deriveSeatStatus(seat airlockSeat, mission airlockMission, hasMission bool, now time.Time) string {
	if strings.EqualFold(seat.Status, "VACATED") || seat.VacatedAt != "" {
		return "VACATED"
	}
	if hasMission && normalizeStatus(mission.Status) == "AIRLOCKED" {
		return "VACATED"
	}
	if strings.EqualFold(seat.Status, "OCCUPIED") {
		return "OCCUPIED"
	}
	if seat.OccupantUserID != "" {
		return "OCCUPIED"
	}
	return "OPEN"
}

func toStateSeat(seat airlockSeat, user airlockUser, mission airlockMission, hasMission bool, stack airlockStack, derivedStatus string, now time.Time) stateSeat {
	out := stateSeat{
		SeatID:               seat.SeatID,
		SeatNumber:           seat.SeatNumber,
		SeatLabel:            seatLabel(seat.SeatNumber),
		Cohort:               cohortForSeat(seat.SeatNumber, seat.Cohort),
		Status:               derivedStatus,
		BoardedAt:            seat.BoardedAt,
		VacatedAt:            seat.VacatedAt,
		VacatedOnMissionID:   seat.VacatedOnMissionID,
		ReboardedAsSeatID:    seat.ReboardedAsSeatID,
		ReboardedAsSeatLabel: seat.ReboardedAsSeatLabel,
	}
	if out.SeatID == "" {
		out.SeatID = fmt.Sprintf("SEAT-%03d", seat.SeatNumber)
	}
	if user.Handle != "" {
		out.Occupant = &stateOccupant{
			Handle:        user.Handle,
			HandleDisplay: "@" + user.Handle,
			DisplayName:   displayName(user),
			Role:          "founder",
			ProfilePath:   "/c/" + user.Handle,
		}
		if user.AvatarStatus == "READY" {
			out.Occupant.AvatarURL = user.AvatarURL
		}
	}
	if hasMission {
		status := missionEffectiveStatus(mission, now)
		out.MissionStatus = status
		out.Mission = &stateMission{
			MissionID:        mission.MissionID,
			Declaration:      mission.Declaration,
			DeclarationURL:   mission.DeclarationURL,
			Status:           status,
			DeclaredAt:       mission.DeclaredAt,
			DeadlineAt:       mission.DeadlineAt,
			ProofURL:         mission.ProofURL,
			ProofSubmittedAt: mission.ProofSubmittedAt,
			ConfirmedAt:      mission.ConfirmedAt,
			AirlockedAt:      mission.AirlockedAt,
			Redeemed:         mission.Redeemed,
		}
	}
	if seat.PricePaidCents > 0 {
		price := statePrice{
			Cents:    seat.PricePaidCents,
			Display:  fmt.Sprintf("$%d", seat.PricePaidCents/100),
			Currency: "USD",
		}
		out.PricePaid = &price
	}
	if stateStack := publicStateStack(stack); stateStack != nil {
		out.Stack = stateStack
	}
	return out
}

func publicStateStack(stack airlockStack) *stateStack {
	if !stack.CrewAlertEnabled && !stack.TheWakeEnabled && !stack.ReentryChallengeEnabled {
		return nil
	}
	return &stateStack{
		CrewAlertEnabled:        stack.CrewAlertEnabled,
		CrewAlertContactCount:   len(stack.CrewAlertContacts),
		TheWakeEnabled:          stack.TheWakeEnabled,
		ReentryChallengeEnabled: stack.ReentryChallengeEnabled,
	}
}

func displayName(user airlockUser) string {
	if user.DisplayName != "" {
		return user.DisplayName
	}
	return user.Handle
}

func missionEffectiveStatus(mission airlockMission, now time.Time) string {
	status := normalizeStatus(mission.Status)
	if status == "" {
		return "DECLARED"
	}
	return status
}

func shouldListVacatedSeatAsActive(seat stateSeat, now time.Time) bool {
	if seat.Mission == nil || normalizeStatus(seat.Mission.Status) != "AIRLOCKED" || seat.Mission.AirlockedAt == "" {
		return false
	}
	airlockedAt, err := time.Parse(time.RFC3339, seat.Mission.AirlockedAt)
	if err != nil {
		return false
	}
	return now.UTC().Before(airlockedAt.UTC().Add(24 * time.Hour))
}

func normalizeStatus(status string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(status), " ", "_"))
}

func cohortForSeat(seatNumber int, cohort string) string {
	if cohort != "" {
		return cohort
	}
	if seatNumber <= 100 {
		return "THE_100"
	}
	return "POST_100"
}

func nearestHatch(activeSeats []stateSeat) *stateHatch {
	var nearest *stateHatch
	for _, seat := range activeSeats {
		if seat.Mission == nil || seat.Occupant == nil {
			continue
		}
		status := normalizeStatus(seat.Mission.Status)
		if status != "DECLARED" && status != "PROOF_PENDING" {
			continue
		}
		candidate := &stateHatch{
			Handle:        seat.Occupant.Handle,
			HandleDisplay: seat.Occupant.HandleDisplay,
			SeatNumber:    seat.SeatNumber,
			SeatLabel:     seat.SeatLabel,
			Cohort:        seat.Cohort,
			MissionID:     seat.Mission.MissionID,
			Mission:       seat.Mission.Declaration,
			Status:        status,
			DeadlineAt:    seat.Mission.DeadlineAt,
		}
		if nearest == nil || candidate.DeadlineAt < nearest.DeadlineAt {
			nearest = candidate
		}
	}
	return nearest
}

func countMissions(missions []airlockMission, now time.Time, statuses ...string) int {
	wanted := make(map[string]bool, len(statuses))
	for _, status := range statuses {
		wanted[normalizeStatus(status)] = true
	}
	count := 0
	for _, mission := range missions {
		if wanted[missionEffectiveStatus(mission, now)] {
			count++
		}
	}
	return count
}

func countConfirmedThisWeek(missions []airlockMission, now time.Time) int {
	start := now.AddDate(0, 0, -7)
	count := 0
	for _, mission := range missions {
		if missionEffectiveStatus(mission, now) != "CONFIRMED" || mission.ConfirmedAt == "" {
			continue
		}
		confirmedAt, err := time.Parse(time.RFC3339, mission.ConfirmedAt)
		if err == nil && confirmedAt.After(start) {
			count++
		}
	}
	return count
}

func stateMissionHistoryFor(missions []airlockMission, seats []airlockSeat, users map[string]airlockUser, now time.Time) []stateMissionHistoryEntry {
	seatsByID := make(map[string]airlockSeat, len(seats))
	for _, seat := range seats {
		seatsByID[seat.SeatID] = seat
	}

	out := make([]stateMissionHistoryEntry, 0, len(missions))
	for _, mission := range missions {
		seat := seatsByID[mission.SeatID]
		if seat.SeatID == "" {
			continue
		}
		user := users[seat.OccupantUserID]
		entry := stateMissionHistoryEntry{
			MissionID:        mission.MissionID,
			SeatID:           mission.SeatID,
			SeatNumber:       seat.SeatNumber,
			SeatLabel:        seatLabel(seat.SeatNumber),
			Cohort:           cohortForSeat(seat.SeatNumber, seat.Cohort),
			Handle:           user.Handle,
			HandleDisplay:    "@" + user.Handle,
			Declaration:      mission.Declaration,
			DeclarationURL:   mission.DeclarationURL,
			Status:           missionEffectiveStatus(mission, now),
			DeclaredAt:       mission.DeclaredAt,
			DeadlineAt:       mission.DeadlineAt,
			ProofURL:         mission.ProofURL,
			ProofSubmittedAt: mission.ProofSubmittedAt,
			ConfirmedAt:      mission.ConfirmedAt,
			AirlockedAt:      mission.AirlockedAt,
			Redeemed:         mission.Redeemed,
		}
		if entry.Handle == "" {
			entry.Handle = eventHandle(user, seat.OccupantUserID)
			entry.HandleDisplay = "@" + entry.Handle
		}
		out = append(out, entry)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].DeclaredAt > out[j].DeclaredAt
	})
	return out
}

func stateEvents(events []airlockEvent, users map[string]airlockUser, seats []airlockSeat, missions []airlockMission) []stateEvent {
	return stateEventsFiltered(events, users, seats, missions, "", 12)
}

func stateEventsByKind(events []airlockEvent, users map[string]airlockUser, seats []airlockSeat, missions []airlockMission, kind string) []stateEvent {
	return stateEventsFiltered(events, users, seats, missions, kind, 0)
}

func stateEventsFiltered(events []airlockEvent, users map[string]airlockUser, seats []airlockSeat, missions []airlockMission, kind string, limit int) []stateEvent {
	seatsByID := make(map[string]airlockSeat, len(seats))
	for _, seat := range seats {
		seatsByID[seat.SeatID] = seat
	}
	missionsByID := make(map[string]airlockMission, len(missions))
	for _, mission := range missions {
		missionsByID[mission.MissionID] = mission
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].OccurredAt > events[j].OccurredAt
	})

	out := make([]stateEvent, 0, len(events))
	for _, event := range events {
		if kind != "" && normalizeStatus(event.Kind) != normalizeStatus(kind) {
			continue
		}
		if limit > 0 && len(out) >= limit {
			break
		}
		user := users[event.UserID]
		seat := seatsByID[event.SeatID]
		mission := missionsByID[event.MissionID]
		handle := user.Handle
		if handle == "" {
			handle = "unknown"
		}
		out = append(out, stateEvent{
			EventID:      event.EventID,
			Kind:         normalizeStatus(event.Kind),
			Handle:       handle,
			SeatNumber:   seat.SeatNumber,
			SeatLabel:    seatLabel(seat.SeatNumber),
			Cohort:       cohortForSeat(seat.SeatNumber, seat.Cohort),
			MissionID:    mission.MissionID,
			Mission:      mission.Declaration,
			OccurredAt:   event.OccurredAt,
			Message:      eventMessage(event, handle, mission),
			DisplayLabel: eventDisplayLabel(event, seat),
			WakePosted:   event.WakePosted,
		})
	}
	return out
}

func eventMessage(event airlockEvent, handle string, mission airlockMission) string {
	if event.Message != "" {
		return event.Message
	}
	switch normalizeStatus(event.Kind) {
	case "BOARDED":
		return fmt.Sprintf("@%s boarded THE 100 - seat %s", handle, seatLabelFromID(event.SeatID))
	case "RE_BOARDED":
		return fmt.Sprintf("@%s re-boarded THE AIRLOCK - seat %s", handle, seatLabelFromID(event.SeatID))
	case "DECLARED":
		return fmt.Sprintf("@%s mission declared - %s", handle, mission.Declaration)
	case "CONFIRMED":
		return fmt.Sprintf("@%s launch confirmed - %s", handle, mission.Declaration)
	case "AIRLOCKED":
		return fmt.Sprintf("@%s airlocked - %s", handle, mission.Declaration)
	case "REDEEMED":
		return fmt.Sprintf("@%s redeemed prior airlock stamps", handle)
	default:
		return fmt.Sprintf("@%s %s", handle, strings.ToLower(event.Kind))
	}
}

func eventDisplayLabel(event airlockEvent, seat airlockSeat) string {
	switch normalizeStatus(event.Kind) {
	case "BOARDED":
		return "THE 100 +1"
	case "RE_BOARDED":
		return fmt.Sprintf("SEAT %s - RE-BOARDED", seatLabel(seat.SeatNumber))
	case "DECLARED":
		return fmt.Sprintf("SEAT %s - DECLARED", seatLabel(seat.SeatNumber))
	case "CONFIRMED":
		return fmt.Sprintf("SEAT %s - CONFIRMED", seatLabel(seat.SeatNumber))
	case "AIRLOCKED":
		return fmt.Sprintf("SEAT %s - VACATED", seatLabel(seat.SeatNumber))
	case "REDEEMED":
		return fmt.Sprintf("SEAT %s - REDEEMED", seatLabel(seat.SeatNumber))
	default:
		return normalizeStatus(event.Kind)
	}
}

func seatLabelFromID(seatID string) string {
	if strings.HasPrefix(seatID, "SEAT-") {
		return strings.TrimPrefix(seatID, "SEAT-")
	}
	return seatID
}

func tierProgressFor(seats []airlockSeat, tier priceTier) stateTierProgress {
	filled := 0
	for _, seat := range seats {
		if seat.SeatNumber >= tier.Start && seat.SeatNumber <= tier.End {
			filled++
		}
	}
	capacity := tier.End - tier.Start + 1
	remaining := capacity - filled
	if remaining < 0 {
		remaining = 0
	}
	return stateTierProgress{
		Tier:      tier.ID,
		Price:     priceForTier(tier.ID),
		SeatRange: fmt.Sprintf("%02d-%02d", tier.Start, tier.End),
		Occupied:  filled,
		Capacity:  capacity,
		Remaining: remaining,
		Progress:  fmt.Sprintf("%d/%d", filled, capacity),
	}
}

func priceTierProgress(seats []airlockSeat) []statePriceTier {
	tiers := make([]statePriceTier, 0, 5)
	for _, tier := range priceTiers {
		if tier.ID == "∞" {
			continue
		}
		progress := tierProgressFor(seats, tier)
		tiers = append(tiers, statePriceTier{
			ID:        tier.ID,
			Tier:      tier.ID,
			Price:     progress.Price,
			SeatRange: progress.SeatRange,
			Filled:    progress.Occupied,
			Seats:     progress.Capacity,
			Occupied:  progress.Occupied,
			Capacity:  progress.Capacity,
			Remaining: progress.Remaining,
			Progress:  progress.Progress,
		})
	}
	return tiers
}

type priceTier struct {
	ID    string
	Start int
	End   int
}

var priceTiers = []priceTier{
	{ID: "01", Start: 1, End: 10},
	{ID: "02", Start: 11, End: 25},
	{ID: "03", Start: 26, End: 50},
	{ID: "04", Start: 51, End: 100},
	{ID: "∞", Start: 101, End: 1_000_000},
}

func tierForSeat(seatNumber int) priceTier {
	for _, tier := range priceTiers {
		if seatNumber >= tier.Start && seatNumber <= tier.End {
			return tier
		}
	}
	return priceTiers[len(priceTiers)-1]
}

func nextTierAfter(current priceTier) priceTier {
	for i, tier := range priceTiers {
		if tier.ID == current.ID && i+1 < len(priceTiers) {
			return priceTiers[i+1]
		}
	}
	return current
}

func seatsUntilNextTier(seatNumber int) int {
	tier := tierForSeat(seatNumber)
	if tier.ID == "∞" {
		return 0
	}
	return tier.End - seatNumber + 1
}

func seatLabel(seatNumber int) string {
	if seatNumber <= 100 {
		return fmt.Sprintf("%02d", seatNumber)
	}
	return fmt.Sprintf("%d", seatNumber)
}

func priceForSeat(seatNumber int) statePrice {
	return priceForTier(tierForSeat(seatNumber).ID)
}

func priceForTier(tier string) statePrice {
	switch tier {
	case "01":
		return statePrice{Cents: 4200, Display: "$42", Currency: "USD"}
	case "02":
		return statePrice{Cents: 8400, Display: "$84", Currency: "USD"}
	case "03":
		return statePrice{Cents: 16800, Display: "$168", Currency: "USD"}
	default:
		return statePrice{Cents: 33600, Display: "$336", Currency: "USD"}
	}
}

type stateSnapshot struct {
	Snapshot       string                     `json:"snapshot"`
	Source         string                     `json:"source"`
	CohortSummary  stateCohortSummary         `json:"cohort_summary"`
	Seats          []stateSeat                `json:"seats"`
	ActiveSeats    []stateSeat                `json:"active_seats"`
	VacatedSeats   []stateSeat                `json:"vacated_seats"`
	NextSeat       stateNextSeat              `json:"next_seat"`
	PriceTiers     []statePriceTier           `json:"price_tiers"`
	TierProgress   stateTierProgress          `json:"tier_progress"`
	NearestHatch   *stateHatch                `json:"nearest_hatch"`
	Metrics        stateMetrics               `json:"metrics"`
	MissionHistory []stateMissionHistoryEntry `json:"mission_history"`
	RecentEvents   []stateEvent               `json:"recent_events"`
	AirlockEvents  []stateEvent               `json:"airlock_events"`
}

type stateCohortSummary struct {
	Cohort        string `json:"cohort"`
	TotalSeats    int    `json:"total_seats"`
	SeatsOccupied int    `json:"seats_occupied"`
	SeatsOpen     int    `json:"seats_open"`
	SeatsVacated  int    `json:"seats_vacated"`
}

type stateSeat struct {
	SeatID               string         `json:"seat_id"`
	SeatNumber           int            `json:"seat_number"`
	SeatLabel            string         `json:"seat_label"`
	Cohort               string         `json:"cohort"`
	Status               string         `json:"status"`
	BoardedAt            string         `json:"boarded_at,omitempty"`
	VacatedAt            string         `json:"vacated_at,omitempty"`
	VacatedOnMissionID   string         `json:"vacated_on_mission_id,omitempty"`
	ReboardedAsSeatID    string         `json:"reboarded_as_seat_id,omitempty"`
	ReboardedAsSeatLabel string         `json:"reboarded_as_seat_label,omitempty"`
	MissionStatus        string         `json:"mission_status,omitempty"`
	Occupant             *stateOccupant `json:"occupant,omitempty"`
	Mission              *stateMission  `json:"mission,omitempty"`
	Stack                *stateStack    `json:"stack,omitempty"`
	PricePaid            *statePrice    `json:"price_paid,omitempty"`
	NextPrice            *statePrice    `json:"next_price,omitempty"`
}

type stateOccupant struct {
	Handle        string `json:"handle"`
	HandleDisplay string `json:"handle_display"`
	DisplayName   string `json:"display_name"`
	Role          string `json:"role"`
	ProfilePath   string `json:"profile_path"`
	AvatarURL     string `json:"avatar_url,omitempty"`
}

type stateMission struct {
	MissionID        string `json:"mission_id"`
	Declaration      string `json:"declaration"`
	DeclarationURL   string `json:"declaration_url,omitempty"`
	Status           string `json:"status"`
	DeclaredAt       string `json:"declared_at"`
	DeadlineAt       string `json:"deadline_at"`
	ProofURL         string `json:"proof_url,omitempty"`
	ProofSubmittedAt string `json:"proof_submitted_at,omitempty"`
	ConfirmedAt      string `json:"confirmed_at,omitempty"`
	AirlockedAt      string `json:"airlocked_at,omitempty"`
	Redeemed         bool   `json:"redeemed,omitempty"`
}

type stateMissionHistoryEntry struct {
	MissionID        string `json:"mission_id"`
	SeatID           string `json:"seat_id"`
	SeatNumber       int    `json:"seat_number"`
	SeatLabel        string `json:"seat_label"`
	Cohort           string `json:"cohort"`
	Handle           string `json:"handle"`
	HandleDisplay    string `json:"handle_display"`
	Declaration      string `json:"declaration"`
	DeclarationURL   string `json:"declaration_url,omitempty"`
	Status           string `json:"status"`
	DeclaredAt       string `json:"declared_at"`
	DeadlineAt       string `json:"deadline_at"`
	ProofURL         string `json:"proof_url,omitempty"`
	ProofSubmittedAt string `json:"proof_submitted_at,omitempty"`
	ConfirmedAt      string `json:"confirmed_at,omitempty"`
	AirlockedAt      string `json:"airlocked_at,omitempty"`
	Redeemed         bool   `json:"redeemed,omitempty"`
}

type stateStack struct {
	CrewAlertEnabled        bool `json:"crew_alert_enabled"`
	CrewAlertContactCount   int  `json:"crew_alert_contact_count"`
	TheWakeEnabled          bool `json:"the_wake_enabled"`
	ReentryChallengeEnabled bool `json:"reentry_challenge_enabled"`
}

type statePrice struct {
	Cents    int    `json:"cents"`
	Display  string `json:"display"`
	Currency string `json:"currency"`
}

type stateNextSeat struct {
	SeatNumber int        `json:"seat_number"`
	SeatLabel  string     `json:"seat_label"`
	Cohort     string     `json:"cohort"`
	Price      statePrice `json:"price"`
}

type stateTierProgress struct {
	Tier      string     `json:"tier"`
	Price     statePrice `json:"price"`
	SeatRange string     `json:"seat_range"`
	Occupied  int        `json:"occupied"`
	Capacity  int        `json:"capacity"`
	Remaining int        `json:"remaining"`
	Progress  string     `json:"progress"`
}

type statePriceTier struct {
	ID        string     `json:"id"`
	Tier      string     `json:"tier"`
	Price     statePrice `json:"price"`
	SeatRange string     `json:"seat_range"`
	Filled    int        `json:"filled"`
	Seats     int        `json:"seats"`
	Occupied  int        `json:"occupied"`
	Capacity  int        `json:"capacity"`
	Remaining int        `json:"remaining"`
	Progress  string     `json:"progress"`
}

type stateHatch struct {
	Handle        string `json:"handle"`
	HandleDisplay string `json:"handle_display"`
	SeatNumber    int    `json:"seat_number"`
	SeatLabel     string `json:"seat_label"`
	Cohort        string `json:"cohort"`
	MissionID     string `json:"mission_id"`
	Mission       string `json:"mission"`
	Status        string `json:"status"`
	DeadlineAt    string `json:"deadline_at"`
}

type stateMetrics struct {
	SeatsOccupied             int           `json:"seats_occupied"`
	SeatsOpen                 int           `json:"seats_open"`
	SeatsVacated              int           `json:"seats_vacated"`
	SeatsTotal                int           `json:"seats_total"`
	ActiveMissions            int           `json:"active_missions"`
	MissionsDeclared          int           `json:"missions_declared"`
	ProofPending              int           `json:"proof_pending"`
	LaunchesConfirmedThisWeek int           `json:"launches_confirmed_this_week"`
	LaunchesConfirmedLifetime int           `json:"launches_confirmed_lifetime"`
	AirlocksLifetime          int           `json:"airlocks_lifetime"`
	RecentEvents              int           `json:"recent_events"`
	NextTierOpens             stateNextTier `json:"next_tier_opens"`
}

type stateNextTier struct {
	Tier         string     `json:"tier"`
	Price        statePrice `json:"price"`
	InSeats      int        `json:"in_seats"`
	AtSeatNumber int        `json:"at_seat_number"`
	AtSeatLabel  string     `json:"at_seat_label"`
}

type stateEvent struct {
	EventID      string `json:"event_id"`
	Kind         string `json:"kind"`
	Handle       string `json:"handle"`
	SeatNumber   int    `json:"seat_number"`
	SeatLabel    string `json:"seat_label"`
	Cohort       string `json:"cohort"`
	MissionID    string `json:"mission_id,omitempty"`
	Mission      string `json:"mission,omitempty"`
	OccurredAt   string `json:"occurred_at"`
	Message      string `json:"message"`
	DisplayLabel string `json:"display_label"`
	WakePosted   bool   `json:"wake_posted,omitempty"`
}
