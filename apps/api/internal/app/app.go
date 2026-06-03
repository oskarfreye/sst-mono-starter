package app

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/theairlock/airlock/apps/api/internal/config"
	"github.com/theairlock/airlock/apps/api/internal/handlers"
	"github.com/theairlock/airlock/apps/api/internal/middleware"
)

// New builds the Fiber app. Routes are grouped under /api because the SST
// Router mounts this Lambda on the /api prefix (the path is NOT stripped).
// /v2 remains registered as a compatibility alias for older direct callers.
func New(cfg *config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
		// Cap request bodies to limit cheap DoS via giant payloads on Lambda.
		// Avatar uploads (multipart, <=5 MB) need headroom above the previous
		// 256 KB cap; 6 MB is the Lambda Function URL request-payload ceiling.
		// Routes carrying bodies this large are authenticated and rate-limited.
		BodyLimit: 6 * 1024 * 1024,
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

	rateLimitConfig := limiter.Config{
		Max:        60,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.Context().RemoteIP().String()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "rate limit exceeded"})
		},
	}
	stateHandler := handlers.NewStateHandler(handlers.StateHandlerConfig{
		TableName:    cfg.TableName,
		AWSRegion:    cfg.AWSRegion,
		LaunchUserID: cfg.LaunchUserID,
	})
	missionHandler := handlers.NewMissionHandler(handlers.MissionHandlerConfig{
		TableName:           cfg.TableName,
		AWSRegion:           cfg.AWSRegion,
		AdminUserIDs:        cfg.AdminUserIDs,
		LaunchUserID:        cfg.LaunchUserID,
		CrewAlertWebhookURL: cfg.CrewAlertWebhookURL,
	})
	hatchHandler := handlers.NewHatchHandler(handlers.HatchHandlerConfig{
		TableName:           cfg.TableName,
		AWSRegion:           cfg.AWSRegion,
		AdminUserIDs:        cfg.AdminUserIDs,
		LaunchUserID:        cfg.LaunchUserID,
		CrewAlertWebhookURL: cfg.CrewAlertWebhookURL,
	})
	stackHandler := handlers.NewStackHandler(handlers.StackHandlerConfig{
		TableName:    cfg.TableName,
		AWSRegion:    cfg.AWSRegion,
		LaunchUserID: cfg.LaunchUserID,
	})
	boarderStateHandler := handlers.NewBoarderStateHandler(handlers.BoarderStateHandlerConfig{
		TableName:    cfg.TableName,
		AWSRegion:    cfg.AWSRegion,
		LaunchUserID: cfg.LaunchUserID,
	})
	achievementsHandler := handlers.NewAchievementsHandler(handlers.AchievementsHandlerConfig{
		TableName:    cfg.TableName,
		AWSRegion:    cfg.AWSRegion,
		LaunchUserID: cfg.LaunchUserID,
	})
	analyticsHandler := handlers.NewAnalyticsHandler(handlers.AnalyticsHandlerConfig{
		TableName:    cfg.TableName,
		AWSRegion:    cfg.AWSRegion,
		LaunchUserID: cfg.LaunchUserID,
	})
	boardingHandler := handlers.NewBoardingHandler(handlers.BoardingHandlerConfig{
		TableName:    cfg.TableName,
		AWSRegion:    cfg.AWSRegion,
		LaunchUserID: cfg.LaunchUserID,
	})
	avatarHandler := handlers.NewAvatarHandler(handlers.AvatarHandlerConfig{
		TableName:           cfg.TableName,
		AWSRegion:           cfg.AWSRegion,
		PublicBucket:        cfg.PublicAssetsBucket,
		PrivateBucket:       cfg.PrivateAssetsBucket,
		PublicAssetsBaseURL: cfg.PublicAssetsBaseURL,
		BedrockRegion:       cfg.BedrockRegion,
		BedrockImageModel:   cfg.BedrockImageModel,
		LaunchUserID:        cfg.LaunchUserID,
	})

	// Public contract in apps/web/SPEC.md. The SST Router mounts this Lambda
	// on /api and does not strip the path prefix before Fiber sees it.
	apiCompat := app.Group("/api")
	apiCompat.Use(limiter.New(rateLimitConfig))
	apiCompat.Get("/state", stateHandler.GetState)
	apiCompat.Get("/seats", stateHandler.GetSeats)
	apiCompat.Get("/events", stateHandler.StreamEvents)
	apiCompat.Post("/c/:handle/view", analyticsHandler.IncrementView)

	v2 := app.Group("/v2")

	// Per-IP throttle on every public-facing /v2 route. Sits in front of the
	// auth guard so unauthenticated bursts can't pin the JWKS path either.
	v2.Use(limiter.New(rateLimitConfig))

	// Health is registered before constructing the auth guard so liveness
	// checks still respond if the issuer's JWKS endpoint is unreachable.
	v2.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"stage":   cfg.Stage,
			"service": "starter-api",
		})
	})
	v2.Get("/state", stateHandler.GetState)
	v2.Get("/seats", stateHandler.GetSeats)
	v2.Get("/events", stateHandler.StreamEvents)
	v2.Post("/c/:handle/view", analyticsHandler.IncrementView)

	authGuard, err := middleware.NewAuthGuard(cfg.AuthURL)
	if err != nil {
		log.Printf("auth guard: cold-start failed, fail-closed mode: %v", err)
		stub := middleware.FailClosedGuard(err)
		registerProtectedRoutes(apiCompat, stub, boardingHandler, missionHandler, stackHandler, hatchHandler, boarderStateHandler, avatarHandler, achievementsHandler, analyticsHandler)
		registerProtectedRoutes(v2, stub, boardingHandler, missionHandler, stackHandler, hatchHandler, boarderStateHandler, avatarHandler, achievementsHandler, analyticsHandler)
		apiCompat.Get("/public", stub, handlers.GetPublic)
		v2.Get("/public", stub, handlers.GetPublic)
		registerRootCompatRoutes(app, rateLimitConfig, stub, stub, stateHandler, boardingHandler, missionHandler, stackHandler, hatchHandler, boarderStateHandler, avatarHandler, achievementsHandler, analyticsHandler)
		return app
	}

	authRequired := authGuard.Middleware(&middleware.AuthGuardOptions{Optional: false})
	optionalAuth := authGuard.Middleware(&middleware.AuthGuardOptions{Optional: true})
	registerProtectedRoutes(apiCompat, authRequired, boardingHandler, missionHandler, stackHandler, hatchHandler, boarderStateHandler, avatarHandler, achievementsHandler, analyticsHandler)
	registerProtectedRoutes(v2, authRequired, boardingHandler, missionHandler, stackHandler, hatchHandler, boarderStateHandler, avatarHandler, achievementsHandler, analyticsHandler)
	apiCompat.Get("/public", optionalAuth, handlers.GetPublic)
	v2.Get("/public", optionalAuth, handlers.GetPublic)
	registerRootCompatRoutes(app, rateLimitConfig, authRequired, optionalAuth, stateHandler, boardingHandler, missionHandler, stackHandler, hatchHandler, boarderStateHandler, avatarHandler, achievementsHandler, analyticsHandler)

	return app
}

func registerRootCompatRoutes(app *fiber.App, rateLimitConfig limiter.Config, auth fiber.Handler, optionalAuth fiber.Handler, stateHandler *handlers.StateHandler, boardingHandler *handlers.BoardingHandler, missionHandler *handlers.MissionHandler, stackHandler *handlers.StackHandler, hatchHandler *handlers.HatchHandler, boarderStateHandler *handlers.BoarderStateHandler, avatarHandler *handlers.AvatarHandler, achievementsHandler *handlers.AchievementsHandler, analyticsHandler *handlers.AnalyticsHandler) {
	// The SST Router route is configured at /api. Depending on whether a call
	// reaches this Lambda through the router edge or the raw Function URL, Fiber
	// can see either /api/<route> or the prefix-stripped /<route>. Register the
	// stripped shape last so it does not change /api or /v2 route middleware
	// ordering.
	rootCompat := app.Group("")
	rootCompat.Use(limiter.New(rateLimitConfig))
	rootCompat.Get("/state", stateHandler.GetState)
	rootCompat.Get("/seats", stateHandler.GetSeats)
	rootCompat.Get("/events", stateHandler.StreamEvents)
	rootCompat.Post("/c/:handle/view", analyticsHandler.IncrementView)
	registerProtectedRoutes(rootCompat, auth, boardingHandler, missionHandler, stackHandler, hatchHandler, boarderStateHandler, avatarHandler, achievementsHandler, analyticsHandler)
	rootCompat.Get("/public", optionalAuth, handlers.GetPublic)
}

func registerProtectedRoutes(group fiber.Router, auth fiber.Handler, boardingHandler *handlers.BoardingHandler, missionHandler *handlers.MissionHandler, stackHandler *handlers.StackHandler, hatchHandler *handlers.HatchHandler, boarderStateHandler *handlers.BoarderStateHandler, avatarHandler *handlers.AvatarHandler, achievementsHandler *handlers.AchievementsHandler, analyticsHandler *handlers.AnalyticsHandler) {
	registerBoardingRoutes(group, auth, boardingHandler)
	registerMissionRoutes(group, auth, missionHandler)
	registerStackRoutes(group, auth, stackHandler)
	registerAdminRoutes(group, auth, missionHandler)
	registerHatchRoutes(group, auth, hatchHandler)
	registerAvatarRoutes(group, auth, avatarHandler)
	group.Get("/me/airlock", auth, boarderStateHandler.GetCurrent)
	group.Get("/me/achievements", auth, achievementsHandler.GetCurrent)
	group.Get("/me/analytics", auth, analyticsHandler.GetAnalytics)
	group.Get("/me", auth, handlers.GetMe)
}

func registerAvatarRoutes(group fiber.Router, auth fiber.Handler, avatarHandler *handlers.AvatarHandler) {
	group.Put("/me/profile", auth, avatarHandler.UpdateProfile)
	group.Post("/me/avatar", auth, avatarHandler.Upload)
	group.Delete("/me/avatar", auth, avatarHandler.Reset)
	group.Get("/me/avatar", auth, avatarHandler.GetStatus)
}

func registerBoardingRoutes(group fiber.Router, auth fiber.Handler, boardingHandler *handlers.BoardingHandler) {
	group.Post("/boarding/board", auth, middleware.RequireFeature("boarding"), boardingHandler.Board)
}

func registerMissionRoutes(group fiber.Router, auth fiber.Handler, missionHandler *handlers.MissionHandler) {
	group.Post("/missions", auth, missionHandler.DeclareMission)
	group.Post("/missions/:mission_id/proof", auth, missionHandler.SubmitProof)
}

func registerStackRoutes(group fiber.Router, auth fiber.Handler, stackHandler *handlers.StackHandler) {
	group.Get("/stacks/current", auth, stackHandler.GetCurrent)
	group.Put("/stacks/current", auth, stackHandler.UpdateCurrent)
}

func registerAdminRoutes(group fiber.Router, auth fiber.Handler, missionHandler *handlers.MissionHandler) {
	group.Post("/admin/missions/:mission_id/confirm", auth, missionHandler.ConfirmProof)
	group.Post("/admin/missions/:mission_id/reject", auth, missionHandler.RejectProof)
}

func registerHatchRoutes(group fiber.Router, auth fiber.Handler, hatchHandler *handlers.HatchHandler) {
	group.Post("/hatch/run", auth, hatchHandler.Run)
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
