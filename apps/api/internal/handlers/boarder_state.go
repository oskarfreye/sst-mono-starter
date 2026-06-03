package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/theairlock/airlock/apps/api/internal/middleware"
)

type BoarderStateHandlerConfig struct {
	TableName    string
	AWSRegion    string
	LaunchUserID string
	Clock        func() time.Time
	Store        boarderStateStore
}

type BoarderStateHandler struct {
	store boarderStateStore
	clock func() time.Time
}

type boarderStateStore interface {
	LoadBoarderState(ctx context.Context, userID string, now time.Time) (boarderState, error)
}

type boarderState struct {
	AirlockUser  airlockUser
	Seat         airlockSeat
	HasSeat      bool
	Mission      airlockMission
	HasMission   bool
	Stack        airlockStack
	CanDeclare   bool
	CanSubmit    bool
	DeclareBy    string
	HasReboarded bool
}

type unavailableBoarderStateStore struct{}

func (unavailableBoarderStateStore) LoadBoarderState(context.Context, string, time.Time) (boarderState, error) {
	return boarderState{}, fmt.Errorf("boarder state store unavailable")
}

func NewBoarderStateHandler(cfg BoarderStateHandlerConfig) *BoarderStateHandler {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	store := cfg.Store
	if store == nil {
		store = unavailableBoarderStateStore{}
		if cfg.TableName != "" {
			store = newDynamoMissionStore(cfg.TableName, cfg.AWSRegion, cfg.LaunchUserID)
		}
	}
	return &BoarderStateHandler{store: store, clock: clock}
}

func (h *BoarderStateHandler) GetCurrent(c *fiber.Ctx) error {
	user, ok := middleware.GetUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	state, err := h.store.LoadBoarderState(c.UserContext(), user.ID, h.clock())
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "boarder state unavailable"})
	}
	return c.JSON(boarderStateResponse(user, state))
}

func (s *dynamoMissionStore) LoadBoarderState(ctx context.Context, userID string, now time.Time) (boarderState, error) {
	client, err := s.dynamoClient(ctx)
	if err != nil {
		return boarderState{}, err
	}
	records, err := s.loadRecords(ctx, client)
	if err != nil {
		return boarderState{}, err
	}
	return buildBoarderState(records, userID, now), nil
}

func buildBoarderState(records stateRecords, userID string, now time.Time) boarderState {
	usersByID := make(map[string]airlockUser, len(records.Users))
	for _, user := range records.Users {
		usersByID[user.UserID] = user
	}
	seat, mission, hasMission, ok := currentSeatForUser(records, userID, now)
	if !ok {
		return boarderState{AirlockUser: usersByID[userID]}
	}
	stack := airlockStack{SeatID: seat.SeatID}
	for _, candidate := range records.Stacks {
		if candidate.SeatID == seat.SeatID {
			stack = candidate
			break
		}
	}
	status := ""
	if hasMission {
		status = missionEffectiveStatus(mission, now)
	}
	declareBy := ""
	if !hasMission {
		declareBy = declarationDeadlineForSeat(seat)
	}
	return boarderState{
		AirlockUser:  usersByID[userID],
		Seat:         seat,
		HasSeat:      true,
		Mission:      mission,
		HasMission:   hasMission,
		Stack:        stack,
		CanDeclare:   !hasMission || status == "CONFIRMED",
		CanSubmit:    hasMission && status == "DECLARED",
		DeclareBy:    declareBy,
		HasReboarded: userHasVacatedSeat(records, userID),
	}
}

func boarderStateResponse(user *middleware.User, state boarderState) fiber.Map {
	res := fiber.Map{
		"user":             userToMap(user),
		"has_seat":         state.HasSeat,
		"has_mission":      state.HasMission,
		"has_reboarded":    state.HasReboarded,
		"can_declare":      state.CanDeclare,
		"can_submit_proof": state.CanSubmit,
	}
	if state.HasSeat {
		res["seat"] = seatResponse(state.Seat)
		res["stack"] = stackResponse(state.Stack, true)
		if state.DeclareBy != "" {
			res["declare_by"] = state.DeclareBy
		}
	}
	if state.AirlockUser.Handle != "" {
		profile := fiber.Map{
			"handle":       state.AirlockUser.Handle,
			"display_name": displayName(state.AirlockUser),
			"profile_path": "/c/" + state.AirlockUser.Handle,
		}
		avatarStatus := state.AirlockUser.AvatarStatus
		if avatarStatus == "" {
			avatarStatus = "NONE"
		}
		profile["avatar_status"] = avatarStatus
		if state.AirlockUser.AvatarURL != "" {
			profile["avatar_url"] = state.AirlockUser.AvatarURL
		}
		if state.AirlockUser.Bio != "" {
			profile["bio"] = state.AirlockUser.Bio
		}
		res["profile"] = profile
	}
	if state.HasMission {
		res["mission"] = missionResponse(state.Mission)
	}
	return res
}

func declarationDeadlineForSeat(seat airlockSeat) string {
	if seat.BoardedAt == "" {
		return ""
	}
	boardedAt, err := time.Parse(time.RFC3339, seat.BoardedAt)
	if err != nil {
		return ""
	}
	return formatInstant(boardedAt.Add(24 * time.Hour))
}

func seatResponse(seat airlockSeat) fiber.Map {
	res := fiber.Map{
		"seat_id":          seat.SeatID,
		"seat_number":      seat.SeatNumber,
		"seat_label":       seatLabel(seat.SeatNumber),
		"cohort":           cohortForSeat(seat.SeatNumber, seat.Cohort),
		"status":           normalizeStatus(seat.Status),
		"occupant_user_id": seat.OccupantUserID,
		"boarded_at":       seat.BoardedAt,
	}
	if seat.PricePaidCents > 0 {
		res["price_paid"] = fiber.Map{
			"cents":    seat.PricePaidCents,
			"display":  fmt.Sprintf("$%d", seat.PricePaidCents/100),
			"currency": "USD",
		}
	}
	return res
}
