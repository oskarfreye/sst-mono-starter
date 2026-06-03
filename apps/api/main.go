package main

import (
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/lambdaurl"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/theairlock/airlock/apps/api/internal/app"
	"github.com/theairlock/airlock/apps/api/internal/config"
	"github.com/theairlock/airlock/apps/api/internal/handlers"
)

var httpHandler http.Handler

func init() {
	cfg := config.Load()
	fiberApp := app.New(cfg)
	stateHandler := handlers.NewStateHandler(handlers.StateHandlerConfig{
		TableName:    cfg.TableName,
		AWSRegion:    cfg.AWSRegion,
		LaunchUserID: cfg.LaunchUserID,
	})
	fiberHandler := adaptor.FiberApp(fiberApp)
	httpHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/events", "/v2/events", "/events":
			stateHandler.StreamEventsHTTP(w, r)
		default:
			fiberHandler.ServeHTTP(w, r)
		}
	})
	log.Println("API initialized for AWS Lambda Function URL")
}

func main() {
	lambdaurl.Start(httpHandler, lambdaurl.WithDetectContentType(false))
}
