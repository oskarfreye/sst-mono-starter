# @theairlock/auth

OpenAuth issuer Lambda. Stateless Hono app deployed via `sst.aws.Auth`.

- Subject schema: `src/subjects.ts`
- Issuer + providers: `src/index.ts`
- OTP rate limiter: `src/rateLimit.ts`

## Required environment

The issuer expects the following environment variables (set in `infra/auth.ts`):

| Var                          | Required | Description                                                                                                       |
| ---------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------- |
| `STAGE`                      | yes      | SST stage name. `dev` / `local` / `dev-*` enable local-loopback redirects and the `local` client.                 |
| `AUTH_CLIENTS`               | prod     | `client_id=uri1,uri2;client_id2=uri1` registry. Authorize requests must match a registered client + redirect URI. |
| `ALLOWED_REDIRECT_ORIGINS`   | prod     | Comma-separated origins (`https://app.example.com`) the issuer is willing to redirect back to.                    |
| `AUTH_DEV_LOG_CODES`         | dev only | `1` to log generated OTPs to CloudWatch. **Never** set this in a shared stage — it leaks working codes.           |

## Sending codes

`sendCode` ships **failing closed**: any deploy without an SES/SNS integration
will throw `sendCode is not configured` on the first OTP request. This is
deliberate — it prevents a forgotten integration from silently logging valid
codes to CloudWatch.

For local development, set `AUTH_DEV_LOG_CODES=1` and the issuer will print
the code to the function logs. Email/phone are redacted; only the OTP is
printed. Do **not** set this var anywhere a teammate could read the logs.

To wire a real sender, replace the `throw` block in `src/index.ts` with your
SES / SNS / Postmark / Twilio client.

## Adding a provider

```ts
import { PasswordProvider } from "@openauthjs/openauth/provider/password";
import { GoogleProvider } from "@openauthjs/openauth/provider/google";

providers: {
  password: PasswordProvider(PasswordUI({ /* ... */ })),
  google: GoogleProvider({
    clientID: Resource.GOOGLE_CLIENT_ID.value,
    clientSecret: Resource.GOOGLE_CLIENT_SECRET.value,
    scopes: ["email", "profile"],
  }),
}
```

Then `link: [...]` the new secrets in `infra/auth.ts`.

### Verifying social-provider emails

When you add a federated provider (Google, Microsoft, etc.), do **not** trust
`tokenset.claims.email` blindly — providers can return an email a user has
not confirmed. In `success`, gate on the verification flag:

```ts
if (value.provider === "google") {
  if (value.tokenset.claims.email_verified !== true) {
    throw new Error("Email not verified by provider");
  }
  // ... map to subject
}
```

## OAuth client requirements

The issuer's `allow()` callback rejects authorize requests that:

1. are missing a `code_challenge`, or use a method other than `S256` (PKCE
   is mandatory),
2. carry a `client_id` not present in the registry built from `AUTH_CLIENTS`,
3. point at a `redirect_uri` not registered for that client OR not on the
   `ALLOWED_REDIRECT_ORIGINS` list.

In dev stages an implicit `local` client is registered with
`http://localhost:3000/auth/callback` so `bun run dev` works out of the box.

## OTP rate limits

The issuer rate-limits the OTP flow against the `AuthTable`:

- **Send (`request` / `resend`)**: per-(IP, contact) cooldown — min 60s
  between sends, max 5 per rolling hour.
- **Verify**: per-state attempt counter — 5 wrong codes burns the counter and
  forces the user to start a new flow.

Both counters use the OpenAuth `Storage` API (read-then-write, not atomic).
At our scale this is fine; for hard quotas swap to a raw DynamoDB
`UpdateItem ADD attempts :one` with a conditional expression.

## Verifying tokens

Clients send the access token as `Authorization: Bearer <jwt>`.
The Go API verifies against the issuer's JWKS at
`$AUTH_URL/.well-known/jwks.json` — see
`apps/api/internal/middleware/auth_guard.go`.

## Rotating signing keys

OpenAuth keeps signing keys in `AuthTable` under the `signing:key` partition
(see `@openauthjs/openauth/keys.ts`). To rotate:

1. Add a new key row with a fresh `kid` and `private`/`public` JWK pair, e.g.
   via a one-shot Lambda or `aws dynamodb put-item`.
2. Wait until the JWKS endpoint advertises both old and new keys (clients
   cache JWKS for ~10m).
3. Mark the old `signing:key` row as expired by setting an `exp` field in the
   past, or delete it once you're confident no in-flight token is signed
   with it.

Refresh tokens issued under the old key keep working until they expire — they
re-derive verification through the JWKS list.
