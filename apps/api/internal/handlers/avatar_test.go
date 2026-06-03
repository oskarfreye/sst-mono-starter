package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/theairlock/airlock/apps/api/internal/middleware"
)

type fakeAvatarStore struct {
	state    boarderState
	stateErr error

	putSourceErr   error
	putPublicErr   error
	setAvatarErr   error
	clearErr       error
	updateProfErr  error
	sourceContent  []byte
	publicContent  []byte
	setURL         string
	setStatus      string
	cleared        bool
	updatedDisplay string
	updatedBio     string
}

func (s *fakeAvatarStore) LoadBoarderState(context.Context, string, time.Time) (boarderState, error) {
	return s.state, s.stateErr
}

func (s *fakeAvatarStore) PutAvatarSource(_ context.Context, _ string, _ string, data []byte, _ string) error {
	s.sourceContent = data
	return s.putSourceErr
}

func (s *fakeAvatarStore) PutAvatarPublic(_ context.Context, _ string, _ string, png []byte) error {
	s.publicContent = png
	return s.putPublicErr
}

func (s *fakeAvatarStore) SetUserAvatar(_ context.Context, _ string, url string, status string, _ time.Time) error {
	s.setURL = url
	s.setStatus = status
	return s.setAvatarErr
}

func (s *fakeAvatarStore) ClearUserAvatar(_ context.Context, _ string, _ time.Time) error {
	s.cleared = true
	return s.clearErr
}

func (s *fakeAvatarStore) UpdateUserProfile(_ context.Context, _ string, displayName string, bio string, _ time.Time) error {
	s.updatedDisplay = displayName
	s.updatedBio = bio
	return s.updateProfErr
}

type fakeImageGenerator struct {
	png     []byte
	err     error
	gotSrc  []byte
	gotType string
}

func (g *fakeImageGenerator) GenerateFromPhoto(_ context.Context, src []byte, contentType string) ([]byte, error) {
	g.gotSrc = src
	g.gotType = contentType
	return g.png, g.err
}

func avatarTestApp(handler *AvatarHandler) *fiber.App {
	// Mirror the production BodyLimit so the handler's own 5 MB avatar check is
	// what rejects oversize uploads (not Fiber's default 4 MB body cap).
	app := fiber.New(fiber.Config{BodyLimit: 6 * 1024 * 1024})
	inject := func(c *fiber.Ctx) error {
		middleware.WithUser(c, &middleware.User{ID: "USER-MAYA", Provider: "clerk"})
		return c.Next()
	}
	app.Post("/me/avatar", inject, handler.Upload)
	app.Delete("/me/avatar", inject, handler.Reset)
	app.Get("/me/avatar", inject, handler.GetStatus)
	app.Put("/me/profile", inject, handler.UpdateProfile)
	return app
}

func newAvatarHandlerForTest(store avatarStore, generator imageGenerator) *AvatarHandler {
	return NewAvatarHandler(AvatarHandlerConfig{
		PublicBucket:        "public-bucket",
		PrivateBucket:       "private-bucket",
		PublicAssetsBaseURL: "https://cdn.example.com/cdn",
		Clock:               func() time.Time { return time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC) },
		Store:               store,
		Generator:           generator,
	})
}

func multipartAvatarRequest(t *testing.T, fieldName string, fileName string, contentType string, content []byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	header := make(map[string][]string)
	header["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name=%q; filename=%q`, fieldName, fileName)}
	header["Content-Type"] = []string{contentType}
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/me/avatar", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func decodeJSON(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var out map[string]any
	if len(data) > 0 {
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("decode body %q: %v", string(data), err)
		}
	}
	return out
}

func TestAvatarUploadRequiresSeat(t *testing.T) {
	store := &fakeAvatarStore{state: boarderState{HasSeat: false}}
	gen := &fakeImageGenerator{png: []byte("png")}
	app := avatarTestApp(newAvatarHandlerForTest(store, gen))

	req := multipartAvatarRequest(t, "avatar", "face.png", "image/png", []byte("fake-png-bytes"))
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	if gen.gotSrc != nil {
		t.Fatalf("generator was called despite missing seat")
	}
}

func TestAvatarUploadRejectsBadContentType(t *testing.T) {
	store := &fakeAvatarStore{state: boarderState{HasSeat: true}}
	gen := &fakeImageGenerator{png: []byte("png")}
	app := avatarTestApp(newAvatarHandlerForTest(store, gen))

	req := multipartAvatarRequest(t, "avatar", "face.gif", "image/gif", []byte("fake-gif-bytes"))
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	if gen.gotSrc != nil {
		t.Fatalf("generator was called for invalid content type")
	}
}

func TestAvatarUploadRejectsOversizeFile(t *testing.T) {
	store := &fakeAvatarStore{state: boarderState{HasSeat: true}}
	gen := &fakeImageGenerator{png: []byte("png")}
	app := avatarTestApp(newAvatarHandlerForTest(store, gen))

	oversize := bytes.Repeat([]byte("a"), maxAvatarUploadBytes+1)
	req := multipartAvatarRequest(t, "avatar", "face.png", "image/png", oversize)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	if gen.gotSrc != nil {
		t.Fatalf("generator was called for oversize file")
	}
}

func TestAvatarUploadHappyPath(t *testing.T) {
	store := &fakeAvatarStore{state: boarderState{HasSeat: true}}
	gen := &fakeImageGenerator{png: []byte("generated-png")}
	app := avatarTestApp(newAvatarHandlerForTest(store, gen))

	req := multipartAvatarRequest(t, "avatar", "face.jpg", "image/jpeg", []byte("fake-jpeg-bytes"))
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := decodeJSON(t, resp)
	wantURL := "https://cdn.example.com/cdn/avatars/USER-MAYA.png"
	if body["avatar_url"] != wantURL {
		t.Fatalf("avatar_url = %v, want %s", body["avatar_url"], wantURL)
	}
	if body["status"] != "READY" {
		t.Fatalf("status = %v, want READY", body["status"])
	}
	if string(store.sourceContent) != "fake-jpeg-bytes" {
		t.Fatalf("stored source = %q, want fake-jpeg-bytes", string(store.sourceContent))
	}
	if string(store.publicContent) != "generated-png" {
		t.Fatalf("stored public = %q, want generated-png", string(store.publicContent))
	}
	if store.setURL != wantURL || store.setStatus != "READY" {
		t.Fatalf("set avatar url=%q status=%q", store.setURL, store.setStatus)
	}
	if gen.gotType != "image/jpeg" {
		t.Fatalf("generator content type = %q, want image/jpeg", gen.gotType)
	}
}

func TestAvatarUploadGeneratorError(t *testing.T) {
	store := &fakeAvatarStore{state: boarderState{HasSeat: true}}
	gen := &fakeImageGenerator{err: fmt.Errorf("nova canvas RAI block: bad")}
	app := avatarTestApp(newAvatarHandlerForTest(store, gen))

	req := multipartAvatarRequest(t, "avatar", "face.png", "image/png", []byte("fake-png-bytes"))
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", resp.StatusCode)
	}
	body := decodeJSON(t, resp)
	if body["error"] != "avatar generation failed" {
		t.Fatalf("error = %v, want avatar generation failed", body["error"])
	}
	if store.publicContent != nil {
		t.Fatalf("public content stored despite generator error")
	}
}

func TestAvatarResetSetsNone(t *testing.T) {
	store := &fakeAvatarStore{state: boarderState{HasSeat: true}}
	app := avatarTestApp(newAvatarHandlerForTest(store, &fakeImageGenerator{}))

	req := httptest.NewRequest(http.MethodDelete, "/me/avatar", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := decodeJSON(t, resp)
	if body["status"] != "NONE" {
		t.Fatalf("status = %v, want NONE", body["status"])
	}
	if !store.cleared {
		t.Fatalf("ClearUserAvatar was not called")
	}
}

func TestAvatarResetRequiresSeat(t *testing.T) {
	store := &fakeAvatarStore{state: boarderState{HasSeat: false}}
	app := avatarTestApp(newAvatarHandlerForTest(store, &fakeImageGenerator{}))

	req := httptest.NewRequest(http.MethodDelete, "/me/avatar", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	if store.cleared {
		t.Fatalf("ClearUserAvatar called without a seat")
	}
}

func TestAvatarGetStatusReturnsReady(t *testing.T) {
	store := &fakeAvatarStore{state: boarderState{
		HasSeat: true,
		AirlockUser: airlockUser{
			AvatarStatus: "READY",
			AvatarURL:    "https://cdn.example.com/cdn/avatars/USER-MAYA.png",
		},
	}}
	app := avatarTestApp(newAvatarHandlerForTest(store, &fakeImageGenerator{}))

	req := httptest.NewRequest(http.MethodGet, "/me/avatar", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := decodeJSON(t, resp)
	if body["status"] != "READY" {
		t.Fatalf("status = %v, want READY", body["status"])
	}
	if body["avatar_url"] != "https://cdn.example.com/cdn/avatars/USER-MAYA.png" {
		t.Fatalf("avatar_url = %v", body["avatar_url"])
	}
}

func TestAvatarGetStatusDefaultsNone(t *testing.T) {
	store := &fakeAvatarStore{state: boarderState{HasSeat: false}}
	app := avatarTestApp(newAvatarHandlerForTest(store, &fakeImageGenerator{}))

	req := httptest.NewRequest(http.MethodGet, "/me/avatar", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := decodeJSON(t, resp)
	if body["status"] != "NONE" {
		t.Fatalf("status = %v, want NONE", body["status"])
	}
	if _, ok := body["avatar_url"]; ok {
		t.Fatalf("avatar_url should be omitted when empty")
	}
}

func TestUpdateProfileHappyPath(t *testing.T) {
	store := &fakeAvatarStore{state: boarderState{
		HasSeat:     true,
		AirlockUser: airlockUser{Handle: "maya"},
	}}
	app := avatarTestApp(newAvatarHandlerForTest(store, &fakeImageGenerator{}))

	req := httptest.NewRequest(http.MethodPut, "/me/profile", strings.NewReader(`{"display_name":"Maya R","bio":"to the moon"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := decodeJSON(t, resp)
	if body["ok"] != true {
		t.Fatalf("ok = %v, want true", body["ok"])
	}
	profile, _ := body["profile"].(map[string]any)
	if profile["display_name"] != "Maya R" || profile["bio"] != "to the moon" || profile["handle"] != "maya" {
		t.Fatalf("profile = %#v", profile)
	}
	if store.updatedDisplay != "Maya R" || store.updatedBio != "to the moon" {
		t.Fatalf("store display=%q bio=%q", store.updatedDisplay, store.updatedBio)
	}
}

func TestUpdateProfileRejectsEmptyDisplayName(t *testing.T) {
	store := &fakeAvatarStore{state: boarderState{HasSeat: true}}
	app := avatarTestApp(newAvatarHandlerForTest(store, &fakeImageGenerator{}))

	req := httptest.NewRequest(http.MethodPut, "/me/profile", strings.NewReader(`{"display_name":"  ","bio":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	if store.updatedDisplay != "" {
		t.Fatalf("UpdateUserProfile called with invalid display name")
	}
}

func TestUpdateProfileRejectsLongBio(t *testing.T) {
	store := &fakeAvatarStore{state: boarderState{HasSeat: true}}
	app := avatarTestApp(newAvatarHandlerForTest(store, &fakeImageGenerator{}))

	longBio := strings.Repeat("x", maxBioChars+1)
	req := httptest.NewRequest(http.MethodPut, "/me/profile", strings.NewReader(fmt.Sprintf(`{"display_name":"Maya","bio":%q}`, longBio)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	if store.updatedBio != "" {
		t.Fatalf("UpdateUserProfile called with invalid bio")
	}
}

func TestUpdateProfileRequiresSeat(t *testing.T) {
	store := &fakeAvatarStore{state: boarderState{HasSeat: false}}
	app := avatarTestApp(newAvatarHandlerForTest(store, &fakeImageGenerator{}))

	req := httptest.NewRequest(http.MethodPut, "/me/profile", strings.NewReader(`{"display_name":"Maya","bio":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}
