// Central place for non-secret configuration shared across stacks.
// Secrets go in ./secrets.ts — keep domain names, versions, constants here.

// Stages where a real domain is expected (must be kept in sync with the
// `protectedStages` const in sst.config.ts).
const PROTECTED_STAGES = ["production", "stage"] as const;

export const APP_DOMAIN = process.env.APP_DOMAIN ?? "example.com";

if (
  APP_DOMAIN === "example.com" &&
  PROTECTED_STAGES.includes($app.stage as (typeof PROTECTED_STAGES)[number])
) {
  // eslint-disable-next-line no-console
  console.warn(
    `[infra/env] APP_DOMAIN is still "example.com" while deploying to a protected stage ("${$app.stage}"). ` +
      `Set the APP_DOMAIN env var before deploying production/stage so custom domains and TLS are wired correctly.`,
  );
}

// Set to true when APP_DOMAIN has been customised away from the placeholder.
// Stacks gate domain blocks on this so the starter still deploys cleanly on
// the default CloudFront URLs for users who haven't configured a domain yet.
export const HAS_CUSTOM_DOMAIN = APP_DOMAIN !== "example.com";

export const API_DOMAIN = `api.${APP_DOMAIN}`;
export const WEB_DOMAIN = APP_DOMAIN;
export const AUTH_DOMAIN = `auth.${APP_DOMAIN}`;

export const VERSION = "0.0.1";
