/**
 * `api` — small helpers that validate the runtime API base URL before issuing
 * any request and construct full URLs via `new URL(path, base)` (no string
 * concatenation, no trailing-slash footguns).
 *
 * The base URL is parsed at module init. If it is missing or invalid, this
 * throws immediately — fail closed rather than silently issuing requests to a
 * broken or attacker-controlled host.
 *
 * `http:` is only allowed for `localhost` / `127.0.0.1` / `[::1]`. Everything
 * else must be `https:`.
 */
const LOCAL_HOSTNAMES = new Set(["localhost", "127.0.0.1", "[::1]", "::1"]);

function parseBaseUrl(raw: unknown): URL {
  if (typeof raw !== "string" || raw.trim() === "") {
    throw new Error("api: PUBLIC_API_URL is not configured");
  }

  let parsed: URL;
  try {
    parsed = new URL(raw);
  } catch {
    throw new Error(`api: PUBLIC_API_URL is not a valid URL: ${raw}`);
  }

  const isLocal = LOCAL_HOSTNAMES.has(parsed.hostname);
  if (parsed.protocol !== "https:" && !(parsed.protocol === "http:" && isLocal)) {
    throw new Error(
      `api: refusing non-https PUBLIC_API_URL (${parsed.protocol}//${parsed.hostname}). ` +
        `http: is only allowed for localhost.`,
    );
  }

  return parsed;
}

export const apiBase = parseBaseUrl(import.meta.env.PUBLIC_API_URL);

/** Build a full URL from a path, resolved against the validated base. */
export function apiUrl(path: string): string {
  return new URL(path, apiBase).toString();
}
