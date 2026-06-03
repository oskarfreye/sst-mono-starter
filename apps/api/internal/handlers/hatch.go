package handlers

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/theairlock/airlock/apps/api/internal/middleware"
)

type HatchHandlerConfig struct {
	TableName           string
	AWSRegion           string
	AdminUserIDs        []string
	LaunchUserID        string
	CrewAlertWebhookURL string
	Clock               func() time.Time
	NewID               func(prefix string) string
	Store               hatchStore
	CrewAlertSender     crewAlertSender
}

type HatchHandler struct {
	store           hatchStore
	crewAlertSender crewAlertSender
	clock           func() time.Time
	newID           func(prefix string) string
	adminUserIDs    map[string]struct{}
}

func NewHatchHandler(cfg HatchHandlerConfig) *HatchHandler {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	newID := cfg.NewID
	if newID == nil {
		newID = func(prefix string) string {
			return prefix + "-" + strings.ToUpper(uuid.NewString())
		}
	}
	store := cfg.Store
	if store == nil {
		store = unavailableHatchStore{}
		if cfg.TableName != "" {
			store = newDynamoMissionStore(cfg.TableName, cfg.AWSRegion, cfg.LaunchUserID)
		}
	}
	crewAlertSender := cfg.CrewAlertSender
	if crewAlertSender == nil && cfg.CrewAlertWebhookURL != "" {
		crewAlertSender = newWebhookCrewAlertSender(cfg.CrewAlertWebhookURL)
	}
	return &HatchHandler{
		store:           store,
		crewAlertSender: crewAlertSender,
		clock:           clock,
		newID:           newID,
		adminUserIDs:    adminUserIDSet(cfg.AdminUserIDs),
	}
}

func (h *HatchHandler) Run(c *fiber.Ctx) error {
	user, ok := middleware.GetUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	if !h.isAdmin(user.ID) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	result, err := h.RunHatch(c.UserContext())
	if err != nil {
		log.Printf("hatch handler: %v", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "hatch unavailable"})
	}
	return c.JSON(result)
}

func (h *HatchHandler) RunHatch(ctx context.Context) (HatchRunResult, error) {
	result, err := h.store.RunHatch(ctx, h.clock(), h.newID)
	if err != nil {
		return result, err
	}
	h.sendCrewAlerts(ctx, &result)
	return result, nil
}

func (h *HatchHandler) sendCrewAlerts(ctx context.Context, result *HatchRunResult) {
	for i := range result.Events {
		event := &result.Events[i]
		if len(event.CrewAlertContacts) == 0 {
			continue
		}
		if h.crewAlertSender == nil {
			event.CrewAlertError = "crew alert webhook is not configured"
			continue
		}
		alert := CrewAlert{
			EventID:    event.EventID,
			Handle:     event.Handle,
			SeatID:     event.SeatID,
			MissionID:  event.MissionID,
			Mission:    event.Mission,
			OccurredAt: event.OccurredAt,
			Contacts:   event.CrewAlertContacts,
		}
		if err := h.crewAlertSender.SendCrewAlert(ctx, alert); err != nil {
			event.CrewAlertError = err.Error()
			log.Printf("crew alert: event %s delivery failed: %v", event.EventID, err)
			continue
		}
		event.CrewAlertSent = true
	}
}

func (h *HatchHandler) isAdmin(userID string) bool {
	if userID == "" {
		return false
	}
	_, ok := h.adminUserIDs[userID]
	return ok
}
