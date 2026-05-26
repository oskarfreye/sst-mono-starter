package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	fiberadapter "github.com/awslabs/aws-lambda-go-api-proxy/fiber"
	"github.com/theairlock/airlock/apps/api/internal/app"
	"github.com/theairlock/airlock/apps/api/internal/config"
)

var fiberLambda *fiberadapter.FiberLambda

func init() {
	cfg := config.Load()
	fiberApp := app.New(cfg)
	fiberLambda = fiberadapter.New(fiberApp)
	log.Println("API initialized for AWS Lambda")
}

func main() {
	lambda.Start(Handler)
}

// Handler proxies API Gateway v2 (HTTP API) events into Fiber.
func Handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return fiberLambda.ProxyWithContextV2(ctx, req)
}
