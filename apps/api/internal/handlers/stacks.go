package handlers

import (
	"errors"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/theairlock/airlock/apps/api/internal/middleware"
)

type StackHandlerConfig struct {
	TableName    string
	AWSRegion    string
	LaunchUserID string
	Clock        func() time.Time
	Store        stackStore
}

type StackHandler struct {
	store stackStore
	clock func() time.Time
}

func NewStackHandler(cfg StackHandlerConfig) *StackHandler {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	store := cfg.Store
	if store == nil {
		store = unavailableStackStore{}
		if cfg.TableName != "" {
			store = newDynamoMissionStore(cfg.TableName, cfg.AWSRegion, cfg.LaunchUserID)
		}
	}
	return &StackHandler{store: store, clock: clock}
}

func (h *StackHandler) GetCurrent(c *fiber.Ctx) error {
	user, ok := middleware.GetUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	stack, err := h.store.LoadStack(c.UserContext(), user.ID, h.clock())
	if err != nil {
		return stackError(c, err)
	}
	return c.JSON(fiber.Map{"stack": stackResponse(stack, true)})
}

func (h *StackHandler) UpdateCurrent(c *fiber.Ctx) error {
	user, ok := middleware.GetUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req struct {
		CrewAlertEnabled        bool     `json:"crew_alert_enabled"`
		CrewAlertContacts       []string `json:"crew_alert_contacts"`
		TheWakeEnabled          bool     `json:"the_wake_enabled"`
		ReentryChallengeEnabled bool     `json:"reentry_challenge_enabled"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	stack, err := h.store.UpdateStack(c.UserContext(), user.ID, stackUpdateInput{
		CrewAlertEnabled:        req.CrewAlertEnabled,
		CrewAlertContacts:       req.CrewAlertContacts,
		TheWakeEnabled:          req.TheWakeEnabled,
		ReentryChallengeEnabled: req.ReentryChallengeEnabled,
	}, h.clock())
	if err != nil {
		return stackError(c, err)
	}
	return c.JSON(fiber.Map{"stack": stackResponse(stack, true)})
}

func stackError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrMissionSeatRequired):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "occupied seat required"})
	case errors.Is(err, ErrStackInvalidContacts):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "crew alert needs 1-8 contacts, each 160 characters or fewer"})
	case errors.Is(err, ErrMissionStateConflict):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "stack state conflict"})
	default:
		log.Printf("stack handler: %v", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "stack service unavailable"})
	}
}

func stackResponse(stack airlockStack, includeContacts bool) fiber.Map {
	res := fiber.Map{
		"seat_id":                   stack.SeatID,
		"crew_alert_enabled":        stack.CrewAlertEnabled,
		"crew_alert_contact_count":  len(stack.CrewAlertContacts),
		"the_wake_enabled":          stack.TheWakeEnabled,
		"reentry_challenge_enabled": stack.ReentryChallengeEnabled,
	}
	if includeContacts {
		res["crew_alert_contacts"] = stack.CrewAlertContacts
		res["the_wake_crosspost"] = stack.TheWakeCrosspost
	}
	return res
}
