package middleware

import (
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func TestUserFromClaimsAcceptsClerkSessionClaims(t *testing.T) {
	user, err := userFromClaims(jwt.MapClaims{
		"sub":                   "user_2xClerk",
		"email":                 "maya@example.com",
		"primary_email_address": "fallback@example.com",
	})
	if err != nil {
		t.Fatalf("userFromClaims returned error: %v", err)
	}
	if user.ID != "user_2xClerk" || user.Provider != "clerk" {
		t.Fatalf("user = %#v, want Clerk subject", user)
	}
	if user.Contact == nil || user.Contact.Email != "maya@example.com" {
		t.Fatalf("contact = %#v, want Clerk email", user.Contact)
	}
}

func TestUserFromClaimsAcceptsOpenAuthAccessClaims(t *testing.T) {
	user, err := userFromClaims(jwt.MapClaims{
		"type": "access",
		"properties": map[string]any{
			"id":       "usr_openauth",
			"provider": "code",
			"contact": map[string]any{
				"email": "oskar@example.com",
				"tel":   "+15555550123",
			},
		},
	})
	if err != nil {
		t.Fatalf("userFromClaims returned error: %v", err)
	}
	if user.ID != "usr_openauth" || user.Provider != "code" {
		t.Fatalf("user = %#v, want OpenAuth subject", user)
	}
	if user.Contact == nil || user.Contact.Email != "oskar@example.com" || user.Contact.Tel != "+15555550123" {
		t.Fatalf("contact = %#v, want OpenAuth contact", user.Contact)
	}
}

func TestUserFromClaimsRejectsNonAccessOpenAuthToken(t *testing.T) {
	_, err := userFromClaims(jwt.MapClaims{
		"type": "refresh",
		"properties": map[string]any{
			"id": "usr_openauth",
		},
	})
	if !errors.Is(err, jwt.ErrTokenInvalidClaims) {
		t.Fatalf("err = %v, want ErrTokenInvalidClaims", err)
	}
}

func TestUserFromClaimsRejectsClerkClaimsWithoutSubject(t *testing.T) {
	_, err := userFromClaims(jwt.MapClaims{
		"email": "maya@example.com",
	})
	if !errors.Is(err, jwt.ErrTokenInvalidClaims) {
		t.Fatalf("err = %v, want ErrTokenInvalidClaims", err)
	}
}

func TestUserFromClaimsParsesClerkBillingClaims(t *testing.T) {
	user, err := userFromClaims(jwt.MapClaims{
		"sub": "user_2xClerk",
		"pla": "u:boarder",
		"fea": "u:boarding, o:dashboard ,,  ",
	})
	if err != nil {
		t.Fatalf("userFromClaims returned error: %v", err)
	}
	if user.Plan != "u:boarder" {
		t.Fatalf("Plan = %q, want u:boarder", user.Plan)
	}
	want := []string{"u:boarding", "o:dashboard"}
	if len(user.Features) != len(want) {
		t.Fatalf("Features = %#v, want %#v", user.Features, want)
	}
	for i, f := range want {
		if user.Features[i] != f {
			t.Fatalf("Features = %#v, want %#v", user.Features, want)
		}
	}
}

func TestUserFromClaimsClerkBillingClaimsAbsent(t *testing.T) {
	user, err := userFromClaims(jwt.MapClaims{
		"sub": "user_2xClerk",
	})
	if err != nil {
		t.Fatalf("userFromClaims returned error: %v", err)
	}
	if user.Plan != "" {
		t.Fatalf("Plan = %q, want empty", user.Plan)
	}
	if user.Features != nil {
		t.Fatalf("Features = %#v, want nil", user.Features)
	}
}

func TestOpenAuthUserLeavesBillingClaimsZeroValued(t *testing.T) {
	user, err := userFromClaims(jwt.MapClaims{
		"type": "access",
		"properties": map[string]any{
			"id":       "usr_openauth",
			"provider": "code",
		},
	})
	if err != nil {
		t.Fatalf("userFromClaims returned error: %v", err)
	}
	if user.Plan != "" || user.Features != nil {
		t.Fatalf("billing claims = (%q, %#v), want zero-valued", user.Plan, user.Features)
	}
}

func TestUserHasFeature(t *testing.T) {
	user := &User{Features: []string{"u:boarding", "o:dashboard"}}

	cases := []struct {
		slug string
		want bool
	}{
		{"boarding", true},    // scope u
		{"dashboard", true},   // scope o
		{"missing", false},    // not present
		{"u:boarding", false}, // slug carries scope, no match on part after ':'
	}
	for _, tc := range cases {
		if got := user.HasFeature(tc.slug); got != tc.want {
			t.Errorf("HasFeature(%q) = %v, want %v", tc.slug, got, tc.want)
		}
	}

	// Bare slug entry (no scope prefix) is accepted.
	bare := &User{Features: []string{"boarding"}}
	if !bare.HasFeature("boarding") {
		t.Errorf("HasFeature(boarding) on bare entry = false, want true")
	}

	var nilUser *User
	if nilUser.HasFeature("boarding") {
		t.Errorf("nil user HasFeature = true, want false")
	}
}

func TestUserHasPlan(t *testing.T) {
	user := &User{Plan: "u:boarder"}
	if !user.HasPlan("boarder") {
		t.Errorf("HasPlan(boarder) = false, want true")
	}
	if user.HasPlan("free") {
		t.Errorf("HasPlan(free) = true, want false")
	}

	bare := &User{Plan: "boarder"}
	if !bare.HasPlan("boarder") {
		t.Errorf("HasPlan(boarder) on bare plan = false, want true")
	}

	empty := &User{}
	if empty.HasPlan("boarder") {
		t.Errorf("HasPlan on empty plan = true, want false")
	}
}

func TestUserPlanSlug(t *testing.T) {
	if got := (&User{Plan: "u:tier-04"}).PlanSlug(); got != "tier-04" {
		t.Errorf("PlanSlug(u:tier-04) = %q, want tier-04", got)
	}
	if got := (&User{Plan: "tier-01"}).PlanSlug(); got != "tier-01" {
		t.Errorf("PlanSlug(tier-01) = %q, want tier-01", got)
	}
	if got := (&User{}).PlanSlug(); got != "" {
		t.Errorf("PlanSlug(empty) = %q, want empty", got)
	}
	var nilUser *User
	if got := nilUser.PlanSlug(); got != "" {
		t.Errorf("PlanSlug(nil) = %q, want empty", got)
	}
}

func TestRequireFeatureEntitledCallsNext(t *testing.T) {
	app := fiber.New()
	app.Get("/gated", func(c *fiber.Ctx) error {
		WithUser(c, &User{ID: "u1", Features: []string{"u:boarding"}})
		return c.Next()
	}, RequireFeature("boarding"), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/gated", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Fatalf("body = %q, want ok", string(body))
	}
}

func TestRequireFeatureNotEntitledReturns402(t *testing.T) {
	app := fiber.New()
	app.Get("/gated", func(c *fiber.Ctx) error {
		WithUser(c, &User{ID: "u1", Features: []string{"u:other"}})
		return c.Next()
	}, RequireFeature("boarding"), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/gated", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402", resp.StatusCode)
	}
}

func TestRequireFeatureNoUserReturns401(t *testing.T) {
	app := fiber.New()
	app.Get("/gated", RequireFeature("boarding"), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/gated", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}
