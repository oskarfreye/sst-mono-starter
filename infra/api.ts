import { table, rateLimitTable } from "./database";
import { publicAssetsBucket, privateAssetsBucket } from "./storage";
import {
  API_DOMAIN,
  HAS_CUSTOM_DOMAIN,
  BEDROCK_REGION,
  BEDROCK_IMAGE_MODEL,
  CLERK_ISSUER,
} from "./env";

// Router fronts the API so you can later mount versioned or streaming handlers
// on the same domain without introducing a new CloudFront distribution.
//
// Custom domain (api.<APP_DOMAIN>) is wired when one is configured. With a
// custom domain SST defaults the CloudFront viewer cert to TLSv1.2_2021.
export const router = new sst.aws.Router("ApiRouter", {
  ...(HAS_CUSTOM_DOMAIN ? { domain: API_DOMAIN } : {}),
});

// Serve the public assets bucket through the Router so we can keep the
// bucket itself private (no `Principal: *` policy, S3 Block Public Access
// stays on). Anything under `/cdn/*` is rewritten to the bucket root and
// signed with CloudFront's Origin Access Control under the hood.
//
// NOTE(apps/web): the frontend should construct asset URLs as
// `${PUBLIC_API_URL}/cdn/<key>` rather than hitting the bucket domain
// directly.
router.routeBucket("/cdn", publicAssetsBucket, {
  rewrite: {
    regex: "^/cdn/(.*)$",
    to: "/$1",
  },
});

// Bedrock image generation (avatar suits) runs cross-region: the Lambda lives
// in eu-central-1, but Nova Canvas is GA only in us-east-1 / eu-west-1 /
// ap-northeast-1. BEDROCK_REGION (defaulted to eu-west-1 in infra/env.ts to keep
// uploads in the EU) drives both the IAM ARN below and the Lambda's env var, so
// the model region is configured in exactly one place.

// Go / Fiber lambda behind the router at /api.
// Mirror of fr3n-mono's `apps/api-v2` pattern: SST builds the Go handler
// from `apps/api/main.go`, fiberadapter bridges API Gateway v2 → Fiber.
export const api = new sst.aws.Function("Api", {
  runtime: "go",
  handler: "apps/api",
  url: {
    cors: false,
    router: {
      instance: router,
      path: "/api",
      readTimeout: "60 seconds",
    },
  },
  streaming: true,
  // C4: tightened links — only the things the API actually needs SDK access
  // to. Buckets are scoped to PutObject/GetObject (no full s3:* via link).
  link: [
    table,
    rateLimitTable,
    sst.aws.permission({
      actions: ["s3:PutObject", "s3:GetObject"],
      resources: [
        publicAssetsBucket.arn,
        $interpolate`${publicAssetsBucket.arn}/*`,
      ],
    }),
    sst.aws.permission({
      actions: ["s3:PutObject", "s3:GetObject"],
      resources: [
        privateAssetsBucket.arn,
        $interpolate`${privateAssetsBucket.arn}/*`,
      ],
    }),
    // Bedrock image generation for astronaut-suit avatars. Scoped to the two
    // foundation models we invoke (Nova Canvas + Titan Image v2 fallback) in
    // BEDROCK_REGION. Account-level model access must also be enabled (see note
    // above).
    sst.aws.permission({
      actions: ["bedrock:InvokeModel"],
      resources: [
        `arn:aws:bedrock:${BEDROCK_REGION}::foundation-model/amazon.nova-canvas-v1:0`,
        `arn:aws:bedrock:${BEDROCK_REGION}::foundation-model/amazon.titan-image-generator-v2:0`,
      ],
    }),
  ],
  environment: {
    STAGE: $app.stage,
    // Clerk Frontend API origin — JWT `iss` + JWKS host. Inlined in infra/env.ts.
    AUTH_URL: CLERK_ISSUER,
    ELECTRO_TABLE_NAME: table.name,
    RATE_LIMIT_TABLE_NAME: rateLimitTable.name,
    PUBLIC_ASSETS_BUCKET: publicAssetsBucket.name,
    PRIVATE_ASSETS_BUCKET: privateAssetsBucket.name,
    // Comma-separated list of origins the API's CORS middleware trusts.
    // Empty string disables CORS headers (fail-closed). Read by the Go
    // middleware in apps/api/internal/app/app.go via cfg.AllowedOrigins.
    ALLOWED_ORIGINS: process.env.ALLOWED_ORIGINS ?? "",
    // Comma/space-separated Clerk user IDs allowed to review proofs and
    // run manual hatch operations. Empty means admin routes fail closed.
    ADMIN_USER_IDS: process.env.ADMIN_USER_IDS ?? "",
    // Clerk user ID that owns the launch seed (Oskar / seat 01).
    // Empty leaves local fallback rendering intact but skips Dynamo seeding.
    LAUNCH_USER_ID: process.env.LAUNCH_USER_ID ?? "",
    // Region + model for Bedrock avatar generation, defined once in
    // infra/env.ts (default eu-west-1 / Nova Canvas) and reused for the IAM ARN.
    BEDROCK_REGION: BEDROCK_REGION,
    BEDROCK_IMAGE_MODEL: BEDROCK_IMAGE_MODEL,
    // Public base URL for generated assets, served through the Router's /cdn
    // route. avatar_url values are `${PUBLIC_ASSETS_BASE_URL}/avatars/<id>.png`.
    PUBLIC_ASSETS_BASE_URL: $interpolate`${router.url}/cdn`,
  },
  // Keep this aligned with the Router read timeout so /api/events can hold an
  // SSE connection open while ordinary REST routes still return immediately.
  timeout: "60 seconds",
  memory: "512 MB",
  // Re-add `concurrency: { reserved: N }` once the account's Lambda
  // concurrency quota is raised AND a CloudWatch alarm on Throttles is wired
  // up. Removed because the default AWS account quota leaves no headroom to
  // reserve anything without dropping unreserved below the 10 minimum.
});
