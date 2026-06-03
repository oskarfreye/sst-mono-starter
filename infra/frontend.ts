import { router } from "./api";
import { publicAssetsBucket } from "./storage";
import { ClerkSecretKey } from "./secrets";
import { CLERK_PUBLISHABLE_KEY, DISCORD_INVITE_URL, HAS_CUSTOM_DOMAIN, WEB_DOMAIN } from "./env";

// Astro web app. `sst.aws.Astro` builds the project with the `astro-sst`
// adapter and deploys it as Lambda + CloudFront. Point API calls at the
// shared router so /api requests flow through the same domain in production.
//
// Hardening:
//  - C2: dropped `link: [publicAssetsBucket]`. The frontend doesn't need IAM
//    access to the bucket — it only needs the public URL to render assets,
//    which we now serve via the Router (`/cdn/*`, see infra/api.ts). If the
//    Astro SSR layer ever needs to read objects directly, add a scoped
//    permission via `permissions: [...]` rather than re-linking the bucket.
//  - M-Inf4: only env vars prefixed `PUBLIC_` are shipped to the browser by
//    Astro. STAGE is server-side only; expose via a private env var if you
//    need it during SSR.
//
// NOTE(apps/web): asset URLs should be constructed as
// `${PUBLIC_API_URL}/cdn/<key>` (the bucket is no longer publicly addressable
// directly). The bucket's domain name is exposed as
// PUBLIC_ASSETS_BUCKET_DOMAIN for tooling/debug only — do NOT use it for
// browser-facing asset URLs in production.
export const web = new sst.aws.Astro("Web", {
  path: "apps/web",
  // Bind the web app to its canonical host (prod apex; {stage}.theairlock.space
  // for dev stages) instead of the auto-generated CloudFront URL. Mirrors the
  // Router (infra/api.ts) and Auth (infra/auth.ts) domain wiring.
  ...(HAS_CUSTOM_DOMAIN ? { domain: { name: WEB_DOMAIN } } : {}),
  environment: {
    PUBLIC_API_URL: router.url,
    // Inlined per stage in infra/env.ts (public; no .env needed).
    PUBLIC_CLERK_PUBLISHABLE_KEY: CLERK_PUBLISHABLE_KEY,
    // Server-side only (SSR Lambda). Sourced from the SST secret so it never
    // lives in shell env or .env: `bun sst secret set ClerkSecretKey <sk_...>`.
    CLERK_SECRET_KEY: ClerkSecretKey.value,
    PUBLIC_DISCORD_URL: DISCORD_INVITE_URL,
    PUBLIC_CLERK_SIGN_IN_URL: "/auth/login",
    PUBLIC_CLERK_SIGN_UP_URL: "/auth/signup",
    PUBLIC_CLERK_SIGN_IN_FALLBACK_REDIRECT_URL: "/manifest",
    PUBLIC_CLERK_SIGN_UP_FALLBACK_REDIRECT_URL: "/manifest",
    ADMIN_USER_IDS: process.env.ADMIN_USER_IDS ?? "",
    // Browsers fetch assets through the Router/CDN under /cdn/*.
    PUBLIC_ASSETS_BASE_URL: $interpolate`${router.url}/cdn`,
    // Raw bucket DNS (s3.amazonaws.com) — kept as a non-public env var for
    // server-side use only. Do NOT prefix this with PUBLIC_.
    ASSETS_BUCKET_DOMAIN: publicAssetsBucket.domain,
  },
});
