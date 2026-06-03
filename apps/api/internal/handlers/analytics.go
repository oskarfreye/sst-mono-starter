package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/theairlock/airlock/apps/api/internal/middleware"
)

// analyticsStore covers the public view-counter write plus the authed read,
// reusing boarder-state lookup to resolve the caller's handle.
// dynamoMissionStore satisfies it; tests inject a fake.
type analyticsStore interface {
	LoadBoarderState(ctx context.Context, userID string, now time.Time) (boarderState, error)
	IncrementProfileView(ctx context.Context, handle string, now time.Time) error
	GetProfileViews(ctx context.Context, handle string) (int, error)
}

type unavailableAnalyticsStore struct{}

func (unavailableAnalyticsStore) LoadBoarderState(context.Context, string, time.Time) (boarderState, error) {
	return boarderState{}, fmt.Errorf("analytics store unavailable")
}

func (unavailableAnalyticsStore) IncrementProfileView(context.Context, string, time.Time) error {
	return fmt.Errorf("analytics store unavailable")
}

func (unavailableAnalyticsStore) GetProfileViews(context.Context, string) (int, error) {
	return 0, fmt.Errorf("analytics store unavailable")
}

type AnalyticsHandlerConfig struct {
	TableName    string
	AWSRegion    string
	LaunchUserID string
	Clock        func() time.Time
	Store        analyticsStore
}

type AnalyticsHandler struct {
	store analyticsStore
	clock func() time.Time
}

func NewAnalyticsHandler(cfg AnalyticsHandlerConfig) *AnalyticsHandler {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	store := cfg.Store
	if store == nil {
		store = unavailableAnalyticsStore{}
		if cfg.TableName != "" {
			store = newDynamoMissionStore(cfg.TableName, cfg.AWSRegion, cfg.LaunchUserID)
		}
	}
	return &AnalyticsHandler{store: store, clock: clock}
}

// IncrementView records a public profile hit. The handle is validated against
// the same pattern boarding enforces; the count is never echoed back.
func (h *AnalyticsHandler) IncrementView(c *fiber.Ctx) error {
	handle := strings.ToLower(c.Params("handle"))
	if !handlePattern.MatchString(handle) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid handle"})
	}
	if err := h.store.IncrementProfileView(c.UserContext(), handle, h.clock()); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "unavailable"})
	}
	return c.JSON(fiber.Map{"ok": true})
}

// GetAnalytics returns the caller's own profile-view count. A handleless caller
// (never boarded) reports zero views rather than erroring.
func (h *AnalyticsHandler) GetAnalytics(c *fiber.Ctx) error {
	user, ok := middleware.GetUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	state, err := h.store.LoadBoarderState(c.UserContext(), user.ID, h.clock())
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "analytics unavailable"})
	}
	handle := state.AirlockUser.Handle
	if handle == "" {
		return c.JSON(fiber.Map{"profile_views": 0})
	}
	views, _ := h.store.GetProfileViews(c.UserContext(), handle)
	return c.JSON(fiber.Map{"profile_views": views})
}
