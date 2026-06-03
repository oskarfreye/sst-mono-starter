package handlers

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/theairlock/airlock/apps/api/internal/middleware"
)

type MissionHandlerConfig struct {
	TableName           string
	AWSRegion           string
	AdminUserIDs        []string
	LaunchUserID        string
	Clock               func() time.Time
	NewID               func(prefix string) string
	Store               missionStore
	CrewAlertWebhookURL string
	CrewAlertSender     crewAlertSender
}

type MissionHandler struct {
	store           missionStore
	crewAlertSender crewAlertSender
	clock           func() time.Time
	newID           func(prefix string) string
	adminUserIDs    map[string]struct{}
}

func NewMissionHandler(cfg MissionHandlerConfig) *MissionHandler {
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
		store = unavailableMissionStore{}
		if cfg.TableName != "" {
			store = newDynamoMissionStore(cfg.TableName, cfg.AWSRegion, cfg.LaunchUserID)
		}
	}
	crewAlertSender := cfg.CrewAlertSender
	if crewAlertSender == nil && cfg.CrewAlertWebhookURL != "" {
		crewAlertSender = newWebhookCrewAlertSender(cfg.CrewAlertWebhookURL)
	}
	return &MissionHandler{
		store:           store,
		crewAlertSender: crewAlertSender,
		clock:           clock,
		newID:           newID,
		adminUserIDs:    adminUserIDSet(cfg.AdminUserIDs),
	}
}

func (h *MissionHandler) DeclareMission(c *fiber.Ctx) error {
	user, ok := middleware.GetUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req struct {
		Declaration    string `json:"declaration"`
		DeclarationURL string `json:"declaration_url"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	mission, err := h.store.DeclareMission(
		c.UserContext(),
		user.ID,
		req.Declaration,
		req.DeclarationURL,
		h.clock(),
		h.newID("MISSION"),
		h.newID("EVENT"),
	)
	if err != nil {
		return missionError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"mission": missionResponse(mission)})
}

func (h *MissionHandler) SubmitProof(c *fiber.Ctx) error {
	user, ok := middleware.GetUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	missionID := strings.TrimSpace(c.Params("mission_id"))
	if missionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "mission id is required"})
	}

	var req struct {
		ProofURL string `json:"proof_url"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	mission, err := h.store.SubmitProof(
		c.UserContext(),
		user.ID,
		missionID,
		req.ProofURL,
		h.clock(),
		h.newID("EVENT"),
	)
	if err != nil {
		return missionError(c, err)
	}
	return c.JSON(fiber.Map{"mission": missionResponse(mission)})
}

func (h *MissionHandler) ConfirmProof(c *fiber.Ctx) error {
	user, ok := middleware.GetUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	if !h.isAdmin(user.ID) {
		return missionError(c, ErrMissionAdminRequired)
	}

	missionID := strings.TrimSpace(c.Params("mission_id"))
	if missionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "mission id is required"})
	}

	mission, err := h.store.ConfirmProof(c.UserContext(), missionID, user.ID, h.clock(), h.newID("EVENT"))
	if err != nil {
		return missionError(c, err)
	}
	return c.JSON(fiber.Map{"mission": missionResponse(mission)})
}

func (h *MissionHandler) RejectProof(c *fiber.Ctx) error {
	user, ok := middleware.GetUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	if !h.isAdmin(user.ID) {
		return missionError(c, ErrMissionAdminRequired)
	}

	missionID := strings.TrimSpace(c.Params("mission_id"))
	if missionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "mission id is required"})
	}

	mutation, err := h.store.RejectProof(c.UserContext(), missionID, user.ID, h.clock(), h.newID("EVENT"))
	if err != nil {
		return missionError(c, err)
	}
	h.sendMissionCrewAlert(c.UserContext(), &mutation)
	res := fiber.Map{"mission": missionResponse(mutation.Mission)}
	if mutation.WakePosted {
		res["wake_posted"] = true
	}
	if len(mutation.CrewAlertContacts) > 0 {
		res["crew_alert_contacts"] = mutation.CrewAlertContacts
		res["crew_alert_sent"] = mutation.CrewAlertSent
		if mutation.CrewAlertError != "" {
			res["crew_alert_error"] = mutation.CrewAlertError
		}
	}
	return c.JSON(res)
}

func (h *MissionHandler) sendMissionCrewAlert(ctx context.Context, mutation *missionReviewMutation) {
	if len(mutation.CrewAlertContacts) == 0 {
		return
	}
	if h.crewAlertSender == nil {
		mutation.CrewAlertError = "crew alert webhook is not configured"
		return
	}
	alert := CrewAlert{
		EventID:    mutation.Event.EventID,
		Handle:     mutation.Handle,
		SeatID:     mutation.Seat.SeatID,
		MissionID:  mutation.Mission.MissionID,
		Mission:    mutation.Mission.Declaration,
		OccurredAt: mutation.Event.OccurredAt,
		Contacts:   mutation.CrewAlertContacts,
	}
	if err := h.crewAlertSender.SendCrewAlert(ctx, alert); err != nil {
		mutation.CrewAlertError = err.Error()
		log.Printf("crew alert: event %s delivery failed: %v", mutation.Event.EventID, err)
		return
	}
	mutation.CrewAlertSent = true
}

func (h *MissionHandler) isAdmin(userID string) bool {
	if userID == "" {
		return false
	}
	_, ok := h.adminUserIDs[userID]
	return ok
}

func missionError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrMissionInvalidDeclaration):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "declaration must be 12-180 characters"})
	case errors.Is(err, ErrMissionInvalidDeclarationURL):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "declaration_url must be an http(s) URL"})
	case errors.Is(err, ErrMissionInvalidProofURL):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "proof_url must be an http(s) URL"})
	case errors.Is(err, ErrMissionSeatRequired):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "occupied seat required"})
	case errors.Is(err, ErrMissionForbidden):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "mission does not belong to user"})
	case errors.Is(err, ErrMissionAdminRequired):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	case errors.Is(err, ErrMissionNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "mission not found"})
	case errors.Is(err, ErrMissionActive):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "active mission already exists"})
	case errors.Is(err, ErrMissionHatchClosed):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "hatch is already closed"})
	case errors.Is(err, ErrMissionStateConflict):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "mission state conflict"})
	default:
		log.Printf("mission handler: %v", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "mission service unavailable"})
	}
}

func missionResponse(mission airlockMission) fiber.Map {
	res := fiber.Map{
		"mission_id":  mission.MissionID,
		"seat_id":     mission.SeatID,
		"declaration": mission.Declaration,
		"declared_at": mission.DeclaredAt,
		"deadline_at": mission.DeadlineAt,
		"status":      mission.Status,
	}
	if mission.DeclarationURL != "" {
		res["declaration_url"] = mission.DeclarationURL
	}
	if mission.ProofURL != "" {
		res["proof_url"] = mission.ProofURL
	}
	if mission.ProofSubmittedAt != "" {
		res["proof_submitted_at"] = mission.ProofSubmittedAt
	}
	if mission.ConfirmedAt != "" {
		res["confirmed_at"] = mission.ConfirmedAt
	}
	if mission.AirlockedAt != "" {
		res["airlocked_at"] = mission.AirlockedAt
	}
	if mission.Redeemed {
		res["redeemed"] = true
	}
	return res
}

func adminUserIDSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}
