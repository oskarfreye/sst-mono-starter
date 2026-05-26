import { auth } from "./auth";
import { table, rateLimitTable } from "./database";
import { publicAssetsBucket, privateAssetsBucket } from "./storage";
// import { EXAMPLE_API_KEY } from "./secrets"; // L-Inf2: re-enable when wired
import { API_DOMAIN, HAS_CUSTOM_DOMAIN } from "./env";

// Router fronts the API so you can later mount /v1, /v2, streaming handlers
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

// Go / Fiber lambda behind the router at /v2.
// Mirror of fr3n-mono's `apps/api-v2` pattern: SST builds the Go handler
// from `apps/api/main.go`, fiberadapter bridges API Gateway v2 → Fiber.
export const api = new sst.aws.Function("Api", {
  runtime: "go",
  handler: "apps/api",
  url: {
    cors: false,
    router: {
      instance: router,
      path: "/v2",
    },
  },
  // C4: tightened links — only the things the API actually needs SDK access
  // to. Buckets are scoped to PutObject/GetObject (no full s3:* via link).
  // `auth` is NOT linked — we only need its URL, passed via env (the API
  // verifies tokens against the issuer's public JWKS, no IAM call needed).
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
    // L-Inf2: EXAMPLE_API_KEY is currently unused by apps/api. Re-add to the
    // link array (and re-import at the top) once a real secret is wired.
    // EXAMPLE_API_KEY,
  ],
  environment: {
    STAGE: $app.stage,
    AUTH_URL: auth.url,
    ELECTRO_TABLE_NAME: table.name,
    RATE_LIMIT_TABLE_NAME: rateLimitTable.name,
    PUBLIC_ASSETS_BUCKET: publicAssetsBucket.name,
    PRIVATE_ASSETS_BUCKET: privateAssetsBucket.name,
    // Comma-separated list of origins the API's CORS middleware trusts.
    // Empty string disables CORS headers (fail-closed). Read by the Go
    // middleware in apps/api/internal/app/app.go via cfg.AllowedOrigins.
    ALLOWED_ORIGINS: process.env.ALLOWED_ORIGINS ?? "",
  },
  // M-Inf2: shorter timeout caps cost on a Slowloris-style abuse and matches
  // what a sync REST handler should ever need. If you add streaming or
  // long-running routes later, override per-route.
  timeout: "10 seconds",
  memory: "512 MB",
  // M-Inf2: cap blast radius / cost. Tune upwards once you have load data;
  // remove only after you've added an alarm on Throttles.
  concurrency: { reserved: 50 },
});
