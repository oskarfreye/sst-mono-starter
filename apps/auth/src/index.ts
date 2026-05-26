import { issuer } from "@openauthjs/openauth";
import { CodeProvider } from "@openauthjs/openauth/provider/code";
import { CodeUI } from "@openauthjs/openauth/ui/code";
import { DynamoStorage } from "@openauthjs/openauth/storage/dynamo";
import { Hono } from "hono";
import { getConnInfo, handle } from "hono/aws-lambda";
import { Resource } from "sst";
import {
  email as vEmail,
  maxLength,
  parse,
  pipe,
  regex,
  safeParse,
  string,
  trim,
  toLowerCase,
} from "valibot";

import {
  checkAndRecordSend,
  clearVerifyAttempts,
  recordVerifyAttempt,
} from "./rateLimit.js";
import { subjects } from "./subjects.js";

// ---------------------------------------------------------------------------
// Config & helpers
// ---------------------------------------------------------------------------

const STAGE = process.env.STAGE ?? "";
const DEV_STAGES = new Set(["dev", "local"]);
const isDevStage = DEV_STAGES.has(STAGE) || STAGE.startsWith("dev-");

// Client registry. Seeded from env so we can rotate without a code change:
//   AUTH_CLIENTS="web=https://app.example.com/callback,https://app2.example.com/callback;cli=http://localhost:8080/callback"
//
// In dev stages, an implicit `local` client is appended that allows
// http://localhost:3000 + 127.0.0.1 callbacks for the Astro frontend.
interface ClientConfig {
  redirectURIs: string[];
}

function loadClientRegistry(): Record<string, ClientConfig> {
  const raw = process.env.AUTH_CLIENTS ?? "";
  const out: Record<string, ClientConfig> = {};
  for (const entry of raw.split(";").map((s) => s.trim()).filter(Boolean)) {
    const [id, uris] = entry.split("=");
    if (!id || !uris) continue;
    out[id.trim()] = {
      redirectURIs: uris.split(",").map((u) => u.trim()).filter(Boolean),
    };
  }
  if (isDevStage) {
    out["local"] = {
      redirectURIs: [
        ...(out["local"]?.redirectURIs ?? []),
        "http://localhost:3000/auth/callback",
        "http://127.0.0.1:3000/auth/callback",
      ],
    };
  }
  return out;
}

const CLIENTS = loadClientRegistry();

// Allowlisted redirect origins (additional safety net beyond per-client URIs).
const ALLOWED_ORIGINS = new Set(
  (process.env.ALLOWED_REDIRECT_ORIGINS ?? "")
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean),
);
if (isDevStage) {
  ALLOWED_ORIGINS.add("http://localhost:3000");
  ALLOWED_ORIGINS.add("http://127.0.0.1:3000");
}

// ---------------------------------------------------------------------------
// Claim canonicalisation / validation
// ---------------------------------------------------------------------------

const emailSchema = pipe(
  string(),
  trim(),
  toLowerCase(),
  vEmail(),
  maxLength(254),
);
const telSchema = pipe(string(), trim(), regex(/^\+[1-9]\d{6,14}$/));

interface CanonicalClaims {
  email?: string;
  tel?: string;
}

function canonicaliseClaims(raw: Record<string, string>): CanonicalClaims | null {
  const out: CanonicalClaims = {};
  if (raw.email) {
    const parsed = safeParse(emailSchema, raw.email);
    if (!parsed.success) return null;
    out.email = parsed.output;
  }
  if (raw.tel) {
    const parsed = safeParse(telSchema, raw.tel);
    if (!parsed.success) return null;
    out.tel = parsed.output;
  }
  if (!out.email && !out.tel) return null;
  return out;
}

// ---------------------------------------------------------------------------
// Storage + issuer
// ---------------------------------------------------------------------------

const storage = DynamoStorage({
  table: Resource.AuthTable.name,
  pk: "pk",
  sk: "sk",
});

const inner = issuer({
  subjects,
  storage,
  ttl: {
    access: 60 * 60 * 4, // 4h
    refresh: 60 * 60 * 24 * 7, // 7d
  },
  // M-Auth1 / M-Auth2 / M-Auth3: authorize requests must (a) come from a
  // registered client, (b) target a redirect_uri that's both in that client's
  // registry AND on the origin allowlist, and (c) carry a S256 PKCE
  // challenge. Anything else is rejected with `unauthorized_client`.
  allow: async (input, req) => {
    const url = new URL(req.url);
    const codeChallenge = url.searchParams.get("code_challenge");
    const codeChallengeMethod = url.searchParams.get("code_challenge_method");
    if (!codeChallenge || codeChallengeMethod !== "S256") {
      return false;
    }
    const client = CLIENTS[input.clientID];
    if (!client) return false;
    if (!client.redirectURIs.includes(input.redirectURI)) return false;
    let redirectOrigin: string;
    try {
      redirectOrigin = new URL(input.redirectURI).origin;
    } catch {
      return false;
    }
    if (!ALLOWED_ORIGINS.has(redirectOrigin)) return false;
    return true;
  },
  providers: {
    code: CodeProvider(
      CodeUI({
        // C1 / H-B3: never log working OTPs unless the operator explicitly
        // opts in for local dev. Failing closed avoids a CloudWatch leak on
        // any deploy that forgot to wire SES/SNS. `tel` is re-validated as
        // E.164 here so a bypass of the front-end can't pump SMS to premium
        // numbers.
        sendCode: async (claims, code) => {
          // Defensive: the rate-limit middleware already validated the claim,
          // but re-validate here so a future call path that bypasses the
          // middleware can't ship malformed input downstream.
          const canonical = canonicaliseClaims(
            claims as Record<string, string>,
          );
          if (!canonical) {
            throw new Error("Invalid contact claim");
          }
          if (process.env.AUTH_DEV_LOG_CODES === "1") {
            console.log("OpenAuth code (dev only)", {
              claims: {
                email: canonical.email ? "[redacted]" : undefined,
                tel: canonical.tel ? "[redacted]" : undefined,
              },
              code,
            });
            return;
          }
          throw new Error(
            "sendCode is not configured — wire SES/SNS before deploying. Set AUTH_DEV_LOG_CODES=1 for local dev.",
          );
        },
      }),
    ),
  },
  success: async (ctx, value) => {
    // H-B1: never trust the raw user-typed claim. Canonicalise + validate
    // before building the subject. No `"anonymous"` fallback.
    const canonical = canonicaliseClaims(
      (value.claims ?? {}) as Record<string, string>,
    );
    if (!canonical) {
      throw new Error("Invalid contact claim");
    }
    // TODO(upsertUserByContact): replace the contact-derived id with a stable
    // user id once we have a user table. For now the canonicalised email or
    // tel is the subject id — re-validated against the tightened subjects
    // schema to fail fast on regressions.
    const id = canonical.email ?? canonical.tel!;
    return ctx.subject(
      "user",
      parse(subjects.user, {
        id,
        provider: "code",
        contact: canonical,
      }),
    );
  },
});

// ---------------------------------------------------------------------------
// Outer wrapper: rate-limit middleware in front of the issuer
// ---------------------------------------------------------------------------

const app = new Hono();

// H-B2 / H-B3: per-(ip, claim) cooldown for OTP send + per-state attempt
// counter for OTP verify. Storage is the same Dynamo `AuthTable` used by
// OpenAuth, so no infra changes are needed (TTL attribute already exists).
//
// NOTE: counters use the high-level OpenAuth Storage API which is read-then-
// write (NOT atomic). For an OTP brute-force ceiling at our scale this is
// acceptable; for a hard quota swap to a raw DynamoDB `UpdateItem` with
// `ADD attempts :one` + a conditional expression. See `src/rateLimit.ts`.
app.use("/code/authorize", async (c, next) => {
  if (c.req.method !== "POST") return next();
  let form: FormData;
  try {
    form = await c.req.formData();
  } catch {
    // No form body — let the inner handler return its own error.
    return next();
  }
  const action = form.get("action")?.toString();
  const ip = getConnInfo(c).remote.address ?? "unknown";

  if (action === "request" || action === "resend") {
    const claim = canonicaliseClaims(
      Object.fromEntries(
        [...form.entries()]
          .filter(([k]) => k === "email" || k === "tel")
          .map(([k, v]) => [k, v.toString()]),
      ),
    );
    if (!claim) {
      return c.text("Invalid contact", 400);
    }
    const claimKey = claim.email ?? claim.tel!;
    const result = await checkAndRecordSend(storage, ip, claimKey);
    if (!result.ok) {
      const headers = new Headers({
        "Content-Type": "text/plain",
        "Retry-After": String(Math.max(1, result.retryAfter)),
      });
      return new Response(
        result.reason === "cooldown"
          ? "Please wait before requesting another code."
          : "Too many code requests. Try again later.",
        { status: 429, headers },
      );
    }
  }

  if (action === "verify") {
    // The encrypted `authorization` cookie is stable for the duration of one
    // auth flow — we use its raw value as the per-state fingerprint for the
    // attempt counter. No PII leaks (the cookie is JWE-encrypted by OpenAuth).
    const cookieHeader = c.req.header("cookie") ?? "";
    const match = /(?:^|;\s*)authorization=([^;]+)/.exec(cookieHeader);
    const stateFp = match?.[1];
    if (!stateFp) {
      return c.text("Missing auth state", 400);
    }
    const result = await recordVerifyAttempt(storage, stateFp);
    if (!result.ok) {
      // Drop server-side state so the user MUST request a new code.
      // OpenAuth keys provider state under `auth:provider:<encrypted>` etc.;
      // safest portable thing we can do here is to clear our counter and
      // surface 429 — clients should restart the flow.
      await clearVerifyAttempts(storage, stateFp);
      return new Response("Too many attempts. Request a new code.", {
        status: 429,
        headers: { "Content-Type": "text/plain" },
      });
    }
  }

  // Re-inject the consumed body so the inner handler can read it.
  // Hono's c.req.formData() caches, so the inner route gets the cached copy.
  return next();
});

app.route("/", inner);

export const handler = handle(app);
