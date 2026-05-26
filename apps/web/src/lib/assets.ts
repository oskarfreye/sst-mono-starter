/**
 * `assets` — validates the runtime asset base URL and exposes a small helper
 * for constructing fully-qualified asset URLs via `new URL(path, base)` (no
 * string concatenation, no trailing-slash footguns).
 *
 * The base URL points at the Router's `/cdn` route, which fronts the private
 * public bucket via OAC. The raw S3 domain is no longer publicly addressable —
 * all asset references must go through this module.
 *
 * Parsed at module init. If missing or invalid, this throws immediately — fail
 * closed rather than silently emitting links to a broken or
 * attacker-controlled host.
 *
 * `http:` is only allowed for `localhost` / `127.0.0.1` / `[::1]`. Everything
 * else must be `https:`.
 */
const LOCAL_HOSTNAMES = new Set(["localhost", "127.0.0.1", "[::1]", "::1"]);

function parseBaseUrl(raw: unknown): URL {
  if (typeof raw !== "string" || raw.trim() === "") {
    throw new Error("assets: PUBLIC_ASSETS_BASE_URL is not configured");
  }

  let parsed: URL;
  try {
    parsed = new URL(raw);
  } catch {
    throw new Error(`assets: PUBLIC_ASSETS_BASE_URL is not a valid URL: ${raw}`);
  }

  const isLocal = LOCAL_HOSTNAMES.has(parsed.hostname);
  if (parsed.protocol !== "https:" && !(parsed.protocol === "http:" && isLocal)) {
    throw new Error(
      `assets: refusing non-https PUBLIC_ASSETS_BASE_URL (${parsed.protocol}//${parsed.hostname}). ` +
        `http: is only allowed for localhost.`,
    );
  }

  return parsed;
}

export const assetsBase = parseBaseUrl(import.meta.env.PUBLIC_ASSETS_BASE_URL);

/** Build a full asset URL from a path, resolved against the validated base. */
export function assetUrl(path: string): string {
  return new URL(path, assetsBase).toString();
}
