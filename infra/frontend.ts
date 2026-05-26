import { router } from "./api";
import { publicAssetsBucket } from "./storage";

// Astro web app. `sst.aws.Astro` builds the project with the `astro-sst`
// adapter and deploys it as Lambda + CloudFront. Point API calls at the
// shared router so /v2 requests flow through the same domain in production.
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
  environment: {
    PUBLIC_API_URL: router.url,
    // Browsers fetch assets through the Router/CDN under /cdn/*.
    PUBLIC_ASSETS_BASE_URL: $interpolate`${router.url}/cdn`,
    // Raw bucket DNS (s3.amazonaws.com) — kept as a non-public env var for
    // server-side use only. Do NOT prefix this with PUBLIC_.
    ASSETS_BUCKET_DOMAIN: publicAssetsBucket.domain,
  },
});
