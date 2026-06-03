package handlers

import (
	"context"
	"io"
	"log"
	"mime/multipart"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/theairlock/airlock/apps/api/internal/middleware"
)

const (
	maxAvatarUploadBytes = 5 * 1024 * 1024
	maxDisplayNameChars  = 60
	maxBioChars          = 280
)

// avatarStore covers boarder-state lookup (for the seat gate) plus the avatar
// and profile mutations. dynamoMissionStore satisfies it; tests inject a fake.
type avatarStore interface {
	LoadBoarderState(ctx context.Context, userID string, now time.Time) (boarderState, error)
	PutAvatarSource(ctx context.Context, bucket string, userID string, data []byte, contentType string) error
	PutAvatarPublic(ctx context.Context, bucket string, userID string, png []byte) error
	SetUserAvatar(ctx context.Context, userID string, url string, status string, now time.Time) error
	ClearUserAvatar(ctx context.Context, userID string, now time.Time) error
	UpdateUserProfile(ctx context.Context, userID string, displayName string, bio string, now time.Time) error
}

type AvatarHandlerConfig struct {
	TableName           string
	AWSRegion           string
	PublicBucket        string
	PrivateBucket       string
	PublicAssetsBaseURL string
	BedrockRegion       string
	BedrockImageModel   string
	LaunchUserID        string
	Clock               func() time.Time
	Store               avatarStore
	Generator           imageGenerator
}

type AvatarHandler struct {
	store               avatarStore
	generator           imageGenerator
	publicBucket        string
	privateBucket       string
	publicAssetsBaseURL string
	clock               func() time.Time
}

func NewAvatarHandler(cfg AvatarHandlerConfig) *AvatarHandler {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	store := cfg.Store
	if store == nil && cfg.TableName != "" {
		store = newDynamoMissionStore(cfg.TableName, cfg.AWSRegion, cfg.LaunchUserID)
	}
	generator := cfg.Generator
	if generator == nil && cfg.BedrockRegion != "" && cfg.PublicBucket != "" {
		generator = newBedrockImageGenerator(cfg.BedrockRegion, cfg.BedrockImageModel)
	}
	return &AvatarHandler{
		store:               store,
		generator:           generator,
		publicBucket:        cfg.PublicBucket,
		privateBucket:       cfg.PrivateBucket,
		publicAssetsBaseURL: strings.TrimRight(cfg.PublicAssetsBaseURL, "/"),
		clock:               clock,
	}
}

func (h *AvatarHandler) avatarURL(userID string) string {
	return h.publicAssetsBaseURL + "/avatars/" + userID + ".png"
}

// requireSeat resolves the caller, then 403s unless they hold an occupied seat.
// On success it returns the verified user and a nil response error. When the
// gate rejects the request it returns a nil user plus the response to send;
// callers must check `user == nil` (a written response always yields a nil
// fiber error, so the error return alone cannot signal rejection).
func (h *AvatarHandler) requireSeat(c *fiber.Ctx) (*middleware.User, error) {
	user, ok := middleware.GetUser(c)
	if !ok {
		return nil, c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	if h.store == nil {
		return nil, c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "avatar service unavailable"})
	}
	state, err := h.store.LoadBoarderState(c.UserContext(), user.ID, h.clock())
	if err != nil {
		log.Printf("avatar handler: load boarder state: %v", err)
		return nil, c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "avatar service unavailable"})
	}
	if !state.HasSeat {
		return nil, c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "an occupied seat is required"})
	}
	return user, nil
}

var allowedAvatarContentTypes = map[string]struct{}{
	"image/png":  {},
	"image/jpeg": {},
	"image/webp": {},
}

func (h *AvatarHandler) Upload(c *fiber.Ctx) error {
	user, err := h.requireSeat(c)
	if user == nil {
		return err
	}
	if h.generator == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "avatar generation is not configured"})
	}

	fileHeader, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "avatar file is required"})
	}
	if fileHeader.Size > maxAvatarUploadBytes {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "avatar file must be 5 MB or smaller"})
	}
	contentType := strings.ToLower(strings.TrimSpace(fileHeader.Header.Get("Content-Type")))
	if _, ok := allowedAvatarContentTypes[contentType]; !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "avatar must be a png, jpeg, or webp image"})
	}

	src, err := readMultipartFile(fileHeader)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "could not read avatar file"})
	}
	if len(src) > maxAvatarUploadBytes {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "avatar file must be 5 MB or smaller"})
	}

	ctx := c.UserContext()
	if h.privateBucket != "" {
		if err := h.store.PutAvatarSource(ctx, h.privateBucket, user.ID, src, contentType); err != nil {
			log.Printf("avatar handler: put source: %v", err)
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "avatar generation failed"})
		}
	}

	png, err := h.generator.GenerateFromPhoto(ctx, src, contentType)
	if err != nil {
		log.Printf("avatar handler: generate: %v", err)
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "avatar generation failed"})
	}

	if err := h.store.PutAvatarPublic(ctx, h.publicBucket, user.ID, png); err != nil {
		log.Printf("avatar handler: put public: %v", err)
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "avatar generation failed"})
	}

	url := h.avatarURL(user.ID)
	if err := h.store.SetUserAvatar(ctx, user.ID, url, "READY", h.clock()); err != nil {
		log.Printf("avatar handler: set user avatar: %v", err)
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "avatar generation failed"})
	}

	return c.JSON(fiber.Map{"avatar_url": url, "status": "READY"})
}

func (h *AvatarHandler) Reset(c *fiber.Ctx) error {
	user, err := h.requireSeat(c)
	if user == nil {
		return err
	}
	if err := h.store.ClearUserAvatar(c.UserContext(), user.ID, h.clock()); err != nil {
		log.Printf("avatar handler: clear user avatar: %v", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "avatar service unavailable"})
	}
	return c.JSON(fiber.Map{"status": "NONE"})
}

func (h *AvatarHandler) GetStatus(c *fiber.Ctx) error {
	user, ok := middleware.GetUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	if h.store == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "avatar service unavailable"})
	}
	state, err := h.store.LoadBoarderState(c.UserContext(), user.ID, h.clock())
	if err != nil {
		log.Printf("avatar handler: load boarder state: %v", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "avatar service unavailable"})
	}
	status := state.AirlockUser.AvatarStatus
	if status == "" {
		status = "NONE"
	}
	res := fiber.Map{"status": status}
	if state.AirlockUser.AvatarURL != "" {
		res["avatar_url"] = state.AirlockUser.AvatarURL
	}
	return c.JSON(res)
}

func (h *AvatarHandler) UpdateProfile(c *fiber.Ctx) error {
	user, err := h.requireSeat(c)
	if user == nil {
		return err
	}

	var req struct {
		DisplayName string `json:"display_name"`
		Bio         string `json:"bio"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	displayNameValue := strings.TrimSpace(req.DisplayName)
	bio := strings.TrimSpace(req.Bio)
	if n := len([]rune(displayNameValue)); n < 1 || n > maxDisplayNameChars {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "display_name must be 1-60 characters"})
	}
	if len([]rune(bio)) > maxBioChars {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bio must be 280 characters or fewer"})
	}

	if err := h.store.UpdateUserProfile(c.UserContext(), user.ID, displayNameValue, bio, h.clock()); err != nil {
		log.Printf("avatar handler: update profile: %v", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "profile service unavailable"})
	}

	state, err := h.store.LoadBoarderState(c.UserContext(), user.ID, h.clock())
	handle := ""
	if err == nil {
		handle = state.AirlockUser.Handle
	}
	return c.JSON(fiber.Map{
		"ok": true,
		"profile": fiber.Map{
			"handle":       handle,
			"display_name": displayNameValue,
			"bio":          bio,
		},
	})
}

func readMultipartFile(fileHeader *multipart.FileHeader) ([]byte, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(io.LimitReader(file, maxAvatarUploadBytes+1))
}
