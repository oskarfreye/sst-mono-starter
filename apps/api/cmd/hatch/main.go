package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/theairlock/airlock/apps/api/internal/config"
	"github.com/theairlock/airlock/apps/api/internal/handlers"
)

var hatchHandler *handlers.HatchHandler

func init() {
	cfg := config.Load()
	hatchHandler = handlers.NewHatchHandler(handlers.HatchHandlerConfig{
		TableName:           cfg.TableName,
		AWSRegion:           cfg.AWSRegion,
		AdminUserIDs:        cfg.AdminUserIDs,
		LaunchUserID:        cfg.LaunchUserID,
		CrewAlertWebhookURL: cfg.CrewAlertWebhookURL,
	})
	log.Println("Hatch cron initialized for AWS Lambda")
}

func main() {
	lambda.Start(Handler)
}

func Handler(ctx context.Context, _ map[string]any) (handlers.HatchRunResult, error) {
	return hatchHandler.RunHatch(ctx)
}
