package middleware

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type AuthGuardOptions struct {
	Optional bool
}

// AuthGuard verifies issuer-signed JWTs via the issuer's JWKS endpoint
// (`$AUTH_URL/.well-known/jwks.json`). Keys are cached and refreshed in the
// background by `keyfunc`.
type AuthGuard struct {
	jwks   keyfunc.Keyfunc
	issuer string
}

// User is the subject extracted from a verified bearer token.
type User struct {
	ID       string
	Provider string
	Contact  *Contact
	// Plan is the active Clerk Billing plan claim (`pla`), a single
	// `scope:planslug` string (e.g. `u:boarder`). Zero-valued for OpenAuth.
	Plan string
	// Features holds the Clerk Billing feature claims (`fea`), each a
	// `scope:featureslug` entry (e.g. `u:boarding`). Zero-valued for OpenAuth.
	Features []string
}

type Contact struct {
	Email string
	Tel   string
}

type userContextKey struct{}

// NewAuthGuard fetches the JWKS at cold start. Returns an error so the
// caller can decide whether to fail fast or fall back to a fail-closed stub.
func NewAuthGuard(authURL string) (*AuthGuard, error) {
	issuer := strings.TrimRight(authURL, "/")

	// Reject plaintext issuer URLs unless they resolve to a loopback host —
	// JWKS over plain HTTP is a downgrade vector for token forgery.
	u, err := url.Parse(issuer)
	if err != nil {
		return nil, fmt.Errorf("auth guard: invalid AUTH_URL %q: %w", authURL, err)
	}
	switch u.Scheme {
	case "https":
	case "http":
		host := u.Hostname()
		if host != "localhost" && host != "127.0.0.1" && host != "::1" {
			return nil, fmt.Errorf("auth guard: AUTH_URL must use https outside localhost (got %q)", authURL)
		}
	default:
		return nil, fmt.Errorf("auth guard: AUTH_URL must be http(s) (got scheme %q)", u.Scheme)
	}

	jwksURL := issuer + "/.well-known/jwks.json"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Build the JWKS storage with an explicit 5m refresh so key rotations
	// propagate quickly without relying on jwkset's hour-long default.
	storage, err := jwkset.NewStorageFromHTTP(jwksURL, jwkset.HTTPClientStorageOptions{
		Ctx:                       context.Background(),
		NoErrorReturnFirstHTTPReq: false,
		RefreshErrorHandler: func(ctx context.Context, err error) {
			slog.Default().ErrorContext(ctx, "auth guard: JWKS refresh failed", "url", jwksURL, "error", err)
		},
		RefreshInterval: 5 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("auth guard: failed to load JWKS from %s: %w", jwksURL, err)
	}
	client, err := jwkset.NewHTTPClient(jwkset.HTTPClientOptions{
		HTTPURLs: map[string]jwkset.Storage{jwksURL: storage},
	})
	if err != nil {
		return nil, fmt.Errorf("auth guard: failed to build JWKS client: %w", err)
	}
	jwks, err := keyfunc.New(keyfunc.Options{Ctx: ctx, Storage: client})
	if err != nil {
		return nil, fmt.Errorf("auth guard: failed to load JWKS from %s: %w", jwksURL, err)
	}
	return &AuthGuard{jwks: jwks, issuer: issuer}, nil
}

// FailClosedGuard returns a middleware that rejects every request with 503.
// Used when the auth guard cannot be constructed at cold start so protected
// routes refuse traffic instead of silently allowing it through.
func FailClosedGuard(cause error) fiber.Handler {
	msg := "auth unavailable"
	if cause != nil {
		log.Printf("auth guard: fail-closed mode active: %v", cause)
	}
	return func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusServiceUnavailable, msg)
	}
}

func (ag *AuthGuard) Middleware(options *AuthGuardOptions) fiber.Handler {
	return func(c *fiber.Ctx) error {
		optional := options != nil && options.Optional

		header := c.Get("Authorization")
		if header == "" {
			if optional {
				return c.Next()
			}
			return unauth(c, "Unauthorized")
		}

		// RFC 6750 §2.1: the "Bearer" auth-scheme token is case-insensitive.
		const prefix = "Bearer "
		if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
			return unauth(c, "Invalid authorization header format")
		}
		token := strings.TrimSpace(header[len(prefix):])

		user, err := ag.verify(token)
		if err != nil {
			if optional {
				return c.Next()
			}
			log.Printf("auth guard: token verification failed: %v", err)
			return unauth(c, "Invalid token")
		}
		if user.ID == "" {
			if optional {
				return c.Next()
			}
			return unauth(c, "missing subject id")
		}

		c.Locals(userContextKey{}, user)
		return c.Next()
	}
}

// GetUser returns the subject attached by a successful auth-guard pass.
func GetUser(c *fiber.Ctx) (*User, bool) {
	user, ok := c.Locals(userContextKey{}).(*User)
	return user, ok
}

// WithUser attaches a verified subject to the request context. It is the
// companion setter to GetUser, used by middleware that authenticates a request
// before handlers run (and by tests that exercise handlers in isolation).
func WithUser(c *fiber.Ctx, user *User) {
	c.Locals(userContextKey{}, user)
}

// HasFeature reports whether the user holds the Clerk Billing feature `slug`.
// Clerk feature claims are `<scope>:<slug>` (scope in {u,o}); a bare slug is
// also accepted. Matching is done on the part after the first ':'.
func (u *User) HasFeature(slug string) bool {
	if u == nil {
		return false
	}
	for _, entry := range u.Features {
		if claimMatchesSlug(entry, slug) {
			return true
		}
	}
	return false
}

// HasPlan reports whether the user's active Clerk Billing plan is `slug`.
// The plan claim is `<scope>:<slug>` (scope in {u,o}); a bare slug is also
// accepted. Matching is done on the part after the first ':'.
func (u *User) HasPlan(slug string) bool {
	if u == nil {
		return false
	}
	return claimMatchesSlug(u.Plan, slug)
}

// PlanSlug returns the slug portion of the active Clerk Billing plan claim
// (`pla`), i.e. the part after the first ':' (scope is dropped). A bare value
// with no scope prefix is returned as-is. Empty for OpenAuth / no plan.
func (u *User) PlanSlug() string {
	if u == nil || u.Plan == "" {
		return ""
	}
	if _, after, found := strings.Cut(u.Plan, ":"); found {
		return after
	}
	return u.Plan
}

// claimMatchesSlug compares a `scope:slug` (or bare `slug`) claim entry against
// a target slug, ignoring the scope prefix.
func claimMatchesSlug(entry, slug string) bool {
	if entry == "" {
		return false
	}
	if _, after, found := strings.Cut(entry, ":"); found {
		return after == slug
	}
	return entry == slug
}

// RequireFeature returns a fiber middleware that gates a route on a Clerk
// Billing feature. It must be chained AFTER the auth guard so the user is
// present in Locals. Responds 401 when no user is attached, 402 when the user
// lacks the feature, and otherwise calls c.Next().
func RequireFeature(slug string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := GetUser(c)
		if !ok || user == nil {
			return unauth(c, "Unauthorized")
		}
		if !user.HasFeature(slug) {
			return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
				"error": "boarding requires an active plan",
			})
		}
		return c.Next()
	}
}

func (ag *AuthGuard) verify(tokenStr string) (*User, error) {
	parsed, err := jwt.Parse(
		tokenStr,
		ag.jwks.Keyfunc,
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(ag.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(30*time.Second),
		// TODO: add jwt.WithAudience("api") once issuer mints aud claims
	)
	if err != nil {
		return nil, err
	}
	if !parsed.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return userFromClaims(claims)
}

func userFromClaims(claims jwt.MapClaims) (*User, error) {
	if tokenType := asString(claims["type"]); tokenType != "" {
		if tokenType != "access" {
			return nil, jwt.ErrTokenInvalidClaims
		}
		return openAuthUserFromClaims(claims)
	}
	return clerkUserFromClaims(claims)
}

func openAuthUserFromClaims(claims jwt.MapClaims) (*User, error) {
	// OpenAuth packs the subject under `properties` (the value you returned
	// from the issuer's `success` handler).
	props, _ := claims["properties"].(map[string]any)
	if props == nil {
		props = map[string]any{}
	}

	user := &User{
		ID:       asString(props["id"]),
		Provider: asString(props["provider"]),
	}
	if contact, ok := props["contact"].(map[string]any); ok {
		user.Contact = &Contact{
			Email: asString(contact["email"]),
			Tel:   asString(contact["tel"]),
		}
	}
	if user.ID == "" {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return user, nil
}

func clerkUserFromClaims(claims jwt.MapClaims) (*User, error) {
	user := &User{
		ID:       asString(claims["sub"]),
		Provider: "clerk",
	}
	if user.ID == "" {
		return nil, jwt.ErrTokenInvalidClaims
	}

	email := firstString(claims["email"], claims["primary_email_address"], claims["email_address"])
	tel := firstString(claims["phone_number"], claims["phone"])
	if email != "" || tel != "" {
		user.Contact = &Contact{Email: email, Tel: tel}
	}

	// Clerk Billing claims: `pla` is a single `scope:planslug` string; `fea`
	// is a single comma-separated `scope:featureslug` list.
	user.Plan = asString(claims["pla"])
	user.Features = parseClerkList(asString(claims["fea"]))

	return user, nil
}

// parseClerkList splits a Clerk comma-separated claim into trimmed, non-empty
// entries. Returns nil when no entries remain.
func parseClerkList(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if entry := strings.TrimSpace(part); entry != "" {
			out = append(out, entry)
		}
	}
	return out
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func firstString(values ...any) string {
	for _, value := range values {
		if s := asString(value); s != "" {
			return s
		}
	}
	return ""
}

func unauth(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": msg})
}
