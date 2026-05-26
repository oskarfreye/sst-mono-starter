package app

import (
	"log"
	"time"

	"github.com/theairlock/airlock/apps/api/internal/config"
	"github.com/theairlock/airlock/apps/api/internal/handlers"
	"github.com/theairlock/airlock/apps/api/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// New builds the Fiber app. Routes are grouped under /v2 because the SST
// Router mounts this Lambda on the /v2 prefix (the path is NOT stripped).
func New(cfg *config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
		// Cap request bodies to limit cheap DoS via giant payloads on Lambda.
		BodyLimit: 256 * 1024,
		// API Gateway terminates TLS and rewrites the source IP into
		// X-Forwarded-For; trust it so c.IP() reflects the real client.
		EnableTrustedProxyCheck: true,
		TrustedProxies:          []string{"0.0.0.0/0"},
		ProxyHeader:             fiber.HeaderXForwardedFor,
	})

	app.Use(recover.New())
	// Explicit format prevents future logger upgrades from auto-including
	// headers (notably Authorization) in access logs.
	app.Use(logger.New(logger.Config{
		Format: "${time} ${status} - ${latency} ${method} ${path}\n",
	}))
	if cfg.AllowedOrigins != "" {
		app.Use(cors.New(cors.Config{
			AllowOrigins: cfg.AllowedOrigins,
			AllowHeaders: "Content-Type, Authorization",
			AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		}))
	}

	v2 := app.Group("/v2")

	// Per-IP throttle on every public-facing /v2 route. Sits in front of the
	// auth guard so unauthenticated bursts can't pin the JWKS path either.
	v2.Use(limiter.New(limiter.Config{
		Max:        60,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.Context().RemoteIP().String()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "rate limit exceeded"})
		},
	}))

	// Health is registered before constructing the auth guard so liveness
	// checks still respond if the issuer's JWKS endpoint is unreachable.
	v2.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"stage":   cfg.Stage,
			"service": "starter-api",
		})
	})

	authGuard, err := middleware.NewAuthGuard(cfg.AuthURL)
	if err != nil {
		log.Printf("auth guard: cold-start failed, fail-closed mode: %v", err)
		stub := middleware.FailClosedGuard(err)
		v2.Get("/me", stub, handlers.GetMe)
		v2.Get("/public", stub, handlers.GetPublic)
		return app
	}

	v2.Get("/me", authGuard.Middleware(&middleware.AuthGuardOptions{Optional: false}), handlers.GetMe)
	v2.Get("/public", authGuard.Middleware(&middleware.AuthGuardOptions{Optional: true}), handlers.GetPublic)

	return app
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}
	// Never echo internal failure detail back to clients — server-side errors
	// can leak file paths, library names, or stack frames.
	if code >= 500 {
		message = "Internal Server Error"
	}
	return c.Status(code).JSON(fiber.Map{"error": message})
}
