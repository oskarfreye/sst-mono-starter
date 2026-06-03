// Central place for non-secret configuration shared across stacks.
// Secrets go in ./secrets.ts — keep domain names, versions, constants here.

// Domain handling (kintura pattern): APP_DOMAIN is the repo's source of truth
// and DNS is managed in Route53, so deploys need no shell exports. Prod is the
// bare apex; every other stage — the `dev` stage and personal `sst dev` stages
// like `exe008` — gets its own subdomain so concurrent deploys never collide on
// DNS. SST issues the ACM cert per stack, DNS-validated against the
// theairlock.space Route53 hosted zone.
//
//   production → theairlock.space        · api.theairlock.space        · auth.theairlock.space
//   exe008     → exe008.theairlock.space · api.exe008.theairlock.space · auth.exe008.theairlock.space
const stage = $app.stage;
const isProd = stage === "production";
const stageSegment = isProd ? "" : `.${stage}`;

export const APP_DOMAIN = "theairlock.space";
export const HAS_CUSTOM_DOMAIN = true;

export const API_DOMAIN = `api${stageSegment}.${APP_DOMAIN}`;
export const WEB_DOMAIN = isProd ? APP_DOMAIN : `${stage}.${APP_DOMAIN}`;
export const AUTH_DOMAIN = `auth${stageSegment}.${APP_DOMAIN}`;

export const VERSION = "0.0.1";

// ── Clerk (Identity + Billing) ──────────────────────────────────────────
// Public Clerk config is inlined per stage (the kintura pattern): the repo is
// the source of truth, so `sst dev` / `sst deploy` need no shell exports or
// .env files. Only the backend secret is an SST secret (ClerkSecretKey, in
// ./secrets.ts, set via `bun sst secret set ClerkSecretKey <sk_...> --stage …`).
//
//  • CLERK_PUBLISHABLE_KEY — safe in the browser bundle; shipped as the Astro
//    PUBLIC_CLERK_PUBLISHABLE_KEY env (see ./frontend.ts).
//  • CLERK_ISSUER — the JWT `iss` claim + JWKS host
//    (`<CLERK_ISSUER>/.well-known/jwks.json`), verified by the Go API + hatch.
//
// The test app is shared by `sst dev` and the `dev` stage; production gets live
// values once the Clerk prod app (and a clerk.<APP_DOMAIN> domain) exist.
export const CLERK_PUBLISHABLE_KEY = isProd
  ? "pk_live_Y2xlcmsudGhlYWlybG9jay5zcGFjZSQ"
  : "pk_test_a2V5LXNlYWd1bGwtOTUuY2xlcmsuYWNjb3VudHMuZGV2JA";

export const CLERK_ISSUER = isProd
  ? "https://clerk.theairlock.space"
  : "https://key-seagull-95.clerk.accounts.dev";

// Bedrock image generation (astronaut-suit avatars). Amazon Nova Canvas is GA
// only in us-east-1, eu-west-1, and ap-northeast-1 — NOT eu-central-1 where the
// API Lambda runs, and it has no cross-region inference profile. We therefore
// invoke Bedrock cross-region; eu-west-1 (Ireland) keeps uploaded photos inside
// the EU, so it's our default. Set the env var only to override for another
// stage/account. NOTE: the AWS account must also enable model access for these
// models in this region via the Bedrock console — IAM permission alone is not
// sufficient.
export const BEDROCK_REGION = process.env.BEDROCK_REGION ?? "eu-west-1";
export const BEDROCK_IMAGE_MODEL =
  process.env.BEDROCK_IMAGE_MODEL ?? "amazon.nova-canvas-v1:0";

// Discord invite link surfaced in the web UI (PUBLIC_DISCORD_URL). Optional —
// blank until a server invite exists; set via DISCORD_INVITE_URL in the deploy env.
export const DISCORD_INVITE_URL = process.env.DISCORD_INVITE_URL ?? "";
