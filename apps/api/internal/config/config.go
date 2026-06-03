package config

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	Stage               string
	AuthURL             string
	TableName           string
	RateLimitTableName  string
	PublicAssetsBucket  string
	PrivateAssetsBucket string
	PublicAssetsBaseURL string
	BedrockRegion       string
	BedrockImageModel   string
	AWSRegion           string
	AllowedOrigins      string
	AdminUserIDs        []string
	LaunchUserID        string
	CrewAlertWebhookURL string
}

func Load() *Config {
	authURL := firstNonEmpty(os.Getenv("CLERK_ISSUER_URL"), os.Getenv("AUTH_URL"))
	if authURL == "" {
		log.Fatal("CLERK_ISSUER_URL or AUTH_URL environment variable is required")
	}

	tableName := os.Getenv("ELECTRO_TABLE_NAME")
	if tableName == "" {
		log.Fatal("ELECTRO_TABLE_NAME environment variable is required")
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "eu-central-1"
	}

	bedrockRegion := firstNonEmpty(os.Getenv("BEDROCK_REGION"), "us-east-1")
	bedrockImageModel := firstNonEmpty(os.Getenv("BEDROCK_IMAGE_MODEL"), "amazon.nova-canvas-v1:0")

	return &Config{
		Stage:               os.Getenv("STAGE"),
		AuthURL:             authURL,
		TableName:           tableName,
		RateLimitTableName:  os.Getenv("RATE_LIMIT_TABLE_NAME"),
		PublicAssetsBucket:  os.Getenv("PUBLIC_ASSETS_BUCKET"),
		PrivateAssetsBucket: os.Getenv("PRIVATE_ASSETS_BUCKET"),
		PublicAssetsBaseURL: os.Getenv("PUBLIC_ASSETS_BASE_URL"),
		BedrockRegion:       bedrockRegion,
		BedrockImageModel:   bedrockImageModel,
		AWSRegion:           region,
		AllowedOrigins:      os.Getenv("ALLOWED_ORIGINS"),
		AdminUserIDs:        splitCSV(os.Getenv("ADMIN_USER_IDS")),
		LaunchUserID:        strings.TrimSpace(os.Getenv("LAUNCH_USER_ID")),
		// Optional plain env — the crew-alert webhook is no longer an SST secret.
		// Unset means no external delivery; the hatch run records "not configured".
		CrewAlertWebhookURL: os.Getenv("CREW_ALERT_WEBHOOK_URL"),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func splitCSV(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t'
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			out = append(out, field)
		}
	}
	return out
}
