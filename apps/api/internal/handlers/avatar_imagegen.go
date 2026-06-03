package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

// imageGenerator turns a user-supplied source photo into an astronaut-suit
// avatar PNG. Injected so handlers can be tested with a fake generator.
type imageGenerator interface {
	GenerateFromPhoto(ctx context.Context, src []byte, contentType string) ([]byte, error)
}

const (
	avatarPrompt = "a person wearing a white NASA-style spacesuit and helmet with a clear glass visor, " +
		"the person's face clearly visible through the visor, dark starfield background, " +
		"photorealistic centered portrait, professional studio lighting"
	avatarNegativePrompt = "blurry, distorted face, multiple people, extra limbs, text, watermark, low quality, deformed, closed visor, opaque visor"
)

type bedrockInvoker interface {
	InvokeModel(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error)
}

// bedrockImageGenerator calls Amazon Nova Canvas (OUTPAINTING task) to keep the
// uploaded face and regenerate the surrounding suit/helmet/background. The
// bedrockruntime client is built lazily on first use against BedrockRegion,
// mirroring the dynamoClient pattern in mission_store.go.
type bedrockImageGenerator struct {
	region  string
	modelID string
	client  bedrockInvoker
}

func newBedrockImageGenerator(region string, modelID string) *bedrockImageGenerator {
	if modelID == "" {
		modelID = "amazon.nova-canvas-v1:0"
	}
	return &bedrockImageGenerator{region: region, modelID: modelID}
}

type novaImageGenerationConfig struct {
	NumberOfImages int     `json:"numberOfImages,omitempty"`
	Quality        string  `json:"quality,omitempty"`
	CfgScale       float64 `json:"cfgScale,omitempty"`
	Seed           int     `json:"seed"`
	Width          int     `json:"width,omitempty"`
	Height         int     `json:"height,omitempty"`
}

type novaOutPaintingParams struct {
	Image           string `json:"image"`
	MaskPrompt      string `json:"maskPrompt,omitempty"`
	OutPaintingMode string `json:"outPaintingMode,omitempty"`
	Text            string `json:"text"`
	NegativeText    string `json:"negativeText,omitempty"`
}

type novaOutPaintingRequest struct {
	TaskType              string                    `json:"taskType"`
	OutPaintingParams     novaOutPaintingParams     `json:"outPaintingParams"`
	ImageGenerationConfig novaImageGenerationConfig `json:"imageGenerationConfig"`
}

// novaCanvasResponse is shared by all task types. error is present only when one
// or more images are blocked by AWS Responsible AI moderation, so it is a
// pointer and the images slice is always bounds-checked before indexing.
type novaCanvasResponse struct {
	Images []string `json:"images"`
	Error  *string  `json:"error,omitempty"`
}

func (g *bedrockImageGenerator) GenerateFromPhoto(ctx context.Context, src []byte, contentType string) ([]byte, error) {
	client, err := g.invoker(ctx)
	if err != nil {
		return nil, err
	}

	// Normalize to a square 1024px PNG so Nova Canvas always receives an
	// accepted format/size and every generated avatar shares one canvas.
	normalized, err := normalizeToSquarePNG(src, contentType)
	if err != nil {
		return nil, fmt.Errorf("normalize upload: %w", err)
	}

	reqBody := novaOutPaintingRequest{
		TaskType: "OUTPAINTING",
		OutPaintingParams: novaOutPaintingParams{
			Image:           base64.StdEncoding.EncodeToString(normalized),
			MaskPrompt:      "face",
			OutPaintingMode: "DEFAULT",
			Text:            avatarPrompt,
			NegativeText:    avatarNegativePrompt,
		},
		ImageGenerationConfig: novaImageGenerationConfig{
			NumberOfImages: 1,
			Quality:        "premium",
			CfgScale:       8,
			Width:          1024,
			Height:         1024,
			Seed:           0,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal nova canvas request: %w", err)
	}

	// Image generation can exceed the SDK's default read timeout; allow more.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	out, err := client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(g.modelID),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        body,
	})
	if err != nil {
		return nil, fmt.Errorf("invoke nova canvas: %w", err)
	}

	var resp novaCanvasResponse
	if err := json.Unmarshal(out.Body, &resp); err != nil {
		return nil, fmt.Errorf("decode nova canvas response: %w", err)
	}
	if len(resp.Images) == 0 {
		if resp.Error != nil {
			return nil, fmt.Errorf("nova canvas RAI block: %s", *resp.Error)
		}
		return nil, fmt.Errorf("nova canvas returned no images")
	}

	png, err := base64.StdEncoding.DecodeString(resp.Images[0])
	if err != nil {
		return nil, fmt.Errorf("decode nova canvas image: %w", err)
	}
	return png, nil
}

func (g *bedrockImageGenerator) invoker(ctx context.Context) (bedrockInvoker, error) {
	if g.client != nil {
		return g.client, nil
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(g.region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	g.client = bedrockruntime.NewFromConfig(cfg)
	return g.client, nil
}
