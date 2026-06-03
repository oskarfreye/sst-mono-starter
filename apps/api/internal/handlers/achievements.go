package handlers

import (
	"context"
	"sort"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/theairlock/airlock/apps/api/internal/middleware"
)

type AchievementsHandlerConfig struct {
	TableName    string
	AWSRegion    string
	LaunchUserID string
	Clock        func() time.Time
	Store        achievementsStore
}

type AchievementsHandler struct {
	store achievementsStore
	clock func() time.Time
}

type achievementsStore interface {
	LoadStateRecords(ctx context.Context) (stateRecords, error)
}

type achievementsResponse struct {
	EarnedCount int                 `json:"earned_count"`
	TotalCount  int                 `json:"total_count"`
	Branches    []achievementBranch `json:"branches"`
}

type achievementBranch struct {
	ID    string            `json:"id"`
	Label string            `json:"label"`
	Nodes []achievementNode `json:"nodes"`
}

type achievementNode struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Earned      bool   `json:"earned"`
	Tone        string `json:"tone"`
}

func NewAchievementsHandler(cfg AchievementsHandlerConfig) *AchievementsHandler {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	store := cfg.Store
	if store == nil {
		store = unavailableAchievementsStore{}
		if cfg.TableName != "" {
			store = newDynamoMissionStore(cfg.TableName, cfg.AWSRegion, cfg.LaunchUserID)
		}
	}
	return &AchievementsHandler{store: store, clock: clock}
}

func (h *AchievementsHandler) GetCurrent(c *fiber.Ctx) error {
	user, ok := middleware.GetUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	records, err := h.store.LoadStateRecords(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "achievements unavailable"})
	}
	return c.JSON(buildAchievements(records, user.ID, h.clock()))
}

func buildAchievements(records stateRecords, userID string, now time.Time) achievementsResponse {
	userSeatIDs := make(map[string]bool)
	userSeats := make([]airlockSeat, 0)
	for _, seat := range records.Seats {
		if seat.OccupantUserID == userID {
			userSeats = append(userSeats, seat)
			userSeatIDs[seat.SeatID] = true
		}
	}
	userMissions := make([]airlockMission, 0)
	for _, mission := range records.Missions {
		if userSeatIDs[mission.SeatID] {
			userMissions = append(userMissions, mission)
		}
	}
	userStacks := make([]airlockStack, 0)
	for _, stack := range records.Stacks {
		if userSeatIDs[stack.SeatID] {
			userStacks = append(userStacks, stack)
		}
	}
	userEvents := make([]airlockEvent, 0)
	for _, event := range records.Events {
		if event.UserID == userID {
			userEvents = append(userEvents, event)
		}
	}

	user := airlockUser{}
	for _, candidate := range records.Users {
		if candidate.UserID == userID {
			user = candidate
			break
		}
	}

	confirmedCount := 0
	airlockedCount := 0
	for _, mission := range userMissions {
		switch missionEffectiveStatus(mission, now) {
		case "CONFIRMED":
			confirmedCount++
		case "AIRLOCKED":
			airlockedCount++
		}
	}

	longestStreak := longestConfirmedStreak(userMissions, now)
	daysAboard := daysAboardForSeats(userSeats, now)

	reboarded := false
	for _, seat := range userSeats {
		if seat.ReboardedAsSeatID != "" {
			reboarded = true
			break
		}
	}
	if !reboarded {
		for _, event := range userEvents {
			if event.Kind == "RE_BOARDED" {
				reboarded = true
				break
			}
		}
	}

	redeemed := false
	for _, mission := range userMissions {
		if mission.Redeemed {
			redeemed = true
			break
		}
	}
	if !redeemed {
		for _, event := range userEvents {
			if event.Kind == "REDEEMED" {
				redeemed = true
				break
			}
		}
	}

	seatHundredOrLower := false
	for _, seat := range userSeats {
		if seat.SeatNumber <= 100 {
			seatHundredOrLower = true
			break
		}
	}

	crewAlerted := false
	wakeArmed := false
	allIn := false
	for _, stack := range userStacks {
		if stack.CrewAlertEnabled {
			crewAlerted = true
		}
		if stack.TheWakeEnabled {
			wakeArmed = true
		}
		if stack.CrewAlertEnabled && stack.TheWakeEnabled {
			allIn = true
		}
	}

	branches := []achievementBranch{
		{
			ID:    "BOARDING",
			Label: "BOARDING",
			Nodes: []achievementNode{
				{ID: "MANIFEST_SIGNED", Label: "MANIFEST SIGNED", Description: "boarded (seat assigned)", Earned: len(userSeats) >= 1, Tone: "ice"},
				{ID: "THE_100", Label: "THE 100", Description: "seat 100 or lower", Earned: seatHundredOrLower, Tone: "ice"},
				{ID: "SUIT_ON", Label: "SUIT ON", Description: "uploaded a face", Earned: user.AvatarStatus == "READY", Tone: "ice"},
			},
		},
		{
			ID:    "SHIPPING",
			Label: "SHIPPING",
			Nodes: []achievementNode{
				{ID: "FIRST_SHIP", Label: "FIRST SHIP", Description: "1 confirmed", Earned: confirmedCount >= 1, Tone: "ice"},
				{ID: "REPEAT_SHIPPER", Label: "REPEAT SHIPPER", Description: "3 confirmed", Earned: confirmedCount >= 3, Tone: "ice"},
				{ID: "VETERAN", Label: "VETERAN", Description: "5 confirmed", Earned: confirmedCount >= 5, Tone: "ice"},
				{ID: "PROLIFIC", Label: "PROLIFIC", Description: "10 confirmed", Earned: confirmedCount >= 10, Tone: "ice"},
			},
		},
		{
			ID:    "STREAK",
			Label: "STREAK",
			Nodes: []achievementNode{
				{ID: "BACK_TO_BACK", Label: "BACK TO BACK", Description: "2 in a row", Earned: longestStreak >= 2, Tone: "ice"},
				{ID: "HOT_STREAK", Label: "HOT STREAK", Description: "3 in a row", Earned: longestStreak >= 3, Tone: "ice"},
				{ID: "UNBROKEN", Label: "UNBROKEN", Description: "5 in a row", Earned: longestStreak >= 5, Tone: "ice"},
			},
		},
		{
			ID:    "COURAGE",
			Label: "COURAGE",
			Nodes: []achievementNode{
				{ID: "CREW_ALERTED", Label: "CREW ALERTED", Description: "Crew Alert armed", Earned: crewAlerted, Tone: "ice"},
				{ID: "WAKE_ARMED", Label: "WAKE ARMED", Description: "The Wake armed", Earned: wakeArmed, Tone: "ice"},
				{ID: "ALL_IN", Label: "ALL IN", Description: "both stacks armed at once", Earned: allIn, Tone: "ice"},
			},
		},
		{
			ID:    "REDEMPTION",
			Label: "REDEMPTION",
			Nodes: []achievementNode{
				{ID: "AIRLOCKED", Label: "AIRLOCKED", Description: "first miss", Earned: airlockedCount >= 1 || userHasVacatedSeat(records, userID), Tone: "hatch"},
				{ID: "RE_BOARDED", Label: "RE-BOARDED", Description: "bought back after a death", Earned: reboarded, Tone: "ice"},
				{ID: "REDEEMED", Label: "REDEEMED", Description: "completed re-entry challenge", Earned: redeemed, Tone: "ice"},
			},
		},
		{
			ID:    "LONGEVITY",
			Label: "LONGEVITY",
			Nodes: []achievementNode{
				{ID: "DAYS_30", Label: "30 DAYS ABOARD", Description: "30 days aboard", Earned: daysAboard >= 30, Tone: "ice"},
				{ID: "DAYS_100", Label: "100 DAYS ON THE BOAT", Description: "100 days aboard", Earned: daysAboard >= 100, Tone: "ice"},
				{ID: "YEAR_ONE", Label: "YEAR ONE", Description: "365 days aboard", Earned: daysAboard >= 365, Tone: "ice"},
			},
		},
	}

	earned := 0
	total := 0
	for _, branch := range branches {
		for _, node := range branch.Nodes {
			total++
			if node.Earned {
				earned++
			}
		}
	}
	return achievementsResponse{
		EarnedCount: earned,
		TotalCount:  total,
		Branches:    branches,
	}
}

func longestConfirmedStreak(missions []airlockMission, now time.Time) int {
	sorted := append([]airlockMission(nil), missions...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].DeclaredAt < sorted[j].DeclaredAt
	})
	cur := 0
	max := 0
	for _, mission := range sorted {
		switch missionEffectiveStatus(mission, now) {
		case "CONFIRMED":
			cur++
			if cur > max {
				max = cur
			}
		case "AIRLOCKED":
			cur = 0
		}
	}
	return max
}

func daysAboardForSeats(seats []airlockSeat, now time.Time) int {
	var total time.Duration
	for _, seat := range seats {
		boardedAt, err := time.Parse(time.RFC3339, seat.BoardedAt)
		if err != nil {
			continue
		}
		end := now
		if seat.VacatedAt != "" {
			vacatedAt, err := time.Parse(time.RFC3339, seat.VacatedAt)
			if err != nil {
				continue
			}
			end = vacatedAt
		}
		span := end.Sub(boardedAt)
		if span < 0 {
			continue
		}
		total += span
	}
	return int(total / (24 * time.Hour))
}
