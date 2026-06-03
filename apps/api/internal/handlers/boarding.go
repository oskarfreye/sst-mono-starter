package handlers

import (
	"errors"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/theairlock/airlock/apps/api/internal/middleware"
)

type BoardingHandlerConfig struct {
	TableName    string
	AWSRegion    string
	LaunchUserID string
	Clock        func() time.Time
	Store        boardingStore
}

type BoardingHandler struct {
	store boardingStore
	clock func() time.Time
}

func NewBoardingHandler(cfg BoardingHandlerConfig) *BoardingHandler {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	store := cfg.Store
	if store == nil {
		store = unavailableBoardingStore{}
		if cfg.TableName != "" {
			store = newDynamoMissionStore(cfg.TableName, cfg.AWSRegion, cfg.LaunchUserID)
		}
	}
	return &BoardingHandler{
		store: store,
		clock: clock,
	}
}

func (h *BoardingHandler) Board(c *fiber.Ctx) error {
	user, ok := middleware.GetUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req struct {
		Handle      string `json:"handle"`
		DisplayName string `json:"display_name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	email := ""
	if user.Contact != nil {
		email = user.Contact.Email
	}

	planAmountCents := planAmountCentsForSlug(user.PlanSlug())

	result, err := h.store.Board(c.UserContext(), user.ID, email, req.Handle, req.DisplayName, planAmountCents, h.clock())
	if err != nil {
		return boardingError(c, err)
	}

	return c.JSON(fiber.Map{
		"seat_id":      result.SeatID,
		"seat_number":  result.SeatNumber,
		"seat_label":   result.SeatLabel,
		"cohort":       result.Cohort,
		"profile_path": "/c/" + result.Handle,
		"created":      result.Created,
	})
}

func boardingError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrBoardingInvalidHandle):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "handle must be 3-32 lowercase URL-safe characters"})
	case errors.Is(err, ErrBoardingHandleTaken):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "handle is already taken"})
	case errors.Is(err, ErrMissionStateConflict):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "boarding state conflict"})
	case errors.Is(err, ErrBoardingPlanUnderpaid):
		return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{"error": "your plan does not cover this seat's price"})
	default:
		log.Printf("boarding handler: %v", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "boarding service unavailable"})
	}
}
