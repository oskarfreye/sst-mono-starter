# Migration: remove Stripe → gate boarding via Clerk Billing

Decision (locked): drop Stripe entirely. Boarding is gated by a **Clerk Billing feature**
(`boarding`) attached to a paid "Boarder" plan. Payment is handled 100% by Clerk (Clerk
connects Stripe behind the scenes; our app never touches Stripe APIs/keys).

## How the gate works (verified against Clerk docs)
- Clerk's DEFAULT session token (v2) carries billing claims that CANNOT be spoofed via custom
  JWT templates:
  - `pla` — a SINGLE string `scope:planslug` (e.g. `u:boarder`). One active plan per user.
  - `fea` — comma-separated `scope:featureslug` list (e.g. `u:boarding,o:dashboard`).
- Our Go API already verifies the default Clerk session token via JWKS, so it can trust `pla`/`fea`.
- We gate on the FEATURE `boarding` (additive, future-proof), not a single plan slug.
- Frontend uses `<PricingTable />` from `@clerk/astro/components` to subscribe and
  `Astro.locals.auth().has({ feature: 'boarding' })` to branch the manifest UI.

## Product scope decisions (state these; they are deliberate)
- Seat numbering (THE_100 = 1..100, POST_100 = 101+), cohort, and reboard linkage are KEPT
  unchanged — that scarcity/identity model is independent of payment.
- The escalating price LADDER no longer drives charges (Clerk plan is a fixed price). The
  pricing functions (priceForSeat/priceForTier/tierForSeat) stay for the marketing ladder /
  state-snapshot tiers display, but boarding stamps a flat display price.
- `seat.price_paid` is stamped from a flat constant `boardingDisplayPriceCents = 4200` ($42) so the
  existing "TOTAL PAID $42" profile UI keeps working. The real charge lives in Clerk.
- Reservations (a Stripe-checkout artifact) are removed. Concurrent seat assignment stays correct
  via the existing optimistic `seatCounterUpdate` conditional + a retry loop.
- Re-boarding still routes to the next POST_100 seat and links the vacated seat.

## HTTP contract change
- REMOVE `POST /api/boarding/checkout` and `POST /api/stripe/webhook` (both `/api` and `/v2`).
- ADD `POST /api/boarding/board` — auth-required AND requires the Clerk `boarding` feature.
  Body `{ "handle": string, "display_name": string }`.
  -> 200 `{ "seat_id", "seat_number", "seat_label", "cohort", "profile_path": "/c/<handle>", "created": bool }`
  - Idempotent: if the user already holds an OCCUPIED seat, return it with `created:false` (200).
  - 400 invalid handle; 409 handle taken; 402/403 if not entitled (handled by middleware before handler).

------------------------------------------------------------------------
## FILE OWNERSHIP

### Agent 1 — Go auth claims  [apps/api/internal/middleware/auth_guard.go (+ auth_guard_test.go)]
- Extend `User` with `Plan string` and `Features []string`.
- In `clerkUserFromClaims`: set `Plan = asString(claims["pla"])`; parse `fea` (a single comma-separated
  string) into `Features` (trimmed, non-empty entries). OpenAuth path leaves them zero-valued.
- Add `func (u *User) HasFeature(slug string) bool` — true if any entry of Features equals `<scope>:<slug>`
  for scope in {u,o} (match on the part after the first ':'; also accept a bare slug). Add `HasPlan(slug string) bool` similarly against `Plan`.
- Add `func RequireFeature(slug string) fiber.Handler` — reads GetUser(c); if missing -> 401; if
  `!user.HasFeature(slug)` -> 402 `{ "error": "boarding requires an active plan" }`; else c.Next().
  (Must be chained AFTER the auth guard so the user is in Locals.)
- Tests: pla/fea parsing, HasFeature/HasPlan, RequireFeature (entitled→next, not→402, no-user→401).
- Independent package — no dependency on `handlers`.

### Agent 2 — Go boarding + state (the big one; package `handlers` + `config`)
Files: boarding.go, boarding_store.go, boarding_rules.go, state_snapshot.go, state_store.go,
config.go, boarding_test.go, boarding_rules_test.go.  DO NOT touch app.go (Agent 3) or auth_guard.go (Agent 1).
- boarding.go: remove ALL Stripe (CheckoutSessionCreator, stripeSessionCreator, CreateCheckout,
  StripeWebhook, checkoutParams, validateCheckoutURL/validateCheckoutURLs, stripe imports,
  StripeSecretKey/StripeWebhookSecret fields in BoardingHandlerConfig/BoardingHandler). Trim
  `boardingError` to the remaining errors. Add `Board(c *fiber.Ctx) error`:
  GetUser → parse JSON {handle, display_name} → store.Board(ctx, user.ID, email, handle, displayName, now)
  → map errors → JSON result per contract. (Feature gate is middleware in app.go, not here.)
- boarding_store.go: change `boardingStore` interface to
  `Board(ctx, userID, email, handle, displayName string, now time.Time) (boardingResult, error)`.
  Remove: stripeCheckoutSession, LoadQuote, FulfillCheckoutSession, ExpireCheckoutSession,
  putBoardingReservation, expireBoardingReservation, all reservation* items/updates/keys,
  reservationFulfilledUpdate, stripeCheckoutSessionFrom, checkoutEventExists, newReservationID.
  Keep/adapt: putBoardingFulfillment (always seatCounterUpdate branch; no reservation/stripe param),
  userItem, handleLockItem, seatItem (remove the 3 stripe_* fields), stackItem, seatCounterUpdate,
  seatCounterKey, reboardedSeatUpdate, putWithCondition, numberAttr, putItem.
  Add `boardingResult { SeatID, SeatNumber int, SeatLabel, Cohort, Handle string; Created bool }`
  and a `Board` method: 5-attempt retry loop — loadRecords; if user already OCCUPIED+!vacated →
  return that seat {Created:false}; else buildBoarding(...) → putBoardingFulfillment; on conditional
  failure retry; success → {Created:true}.
- boarding_rules.go: remove buildBoardingQuote, buildBoardingFulfillment(stripe), reservationForCheckout,
  reservationIsActive, boardingQuote, boardingReservationTTL, and stripe-only errors
  (ErrBoardingPaymentIncomplete, ErrBoardingMissingMetadata, ErrBoardingPaymentMismatch,
  ErrBoardingReservationActive, ErrBoardingInvalidReturnURL). Remove the `records.Reservations` loops
  in nextAssignableSeatNumber/nextPost100SeatNumber. Add:
  `buildBoarding(records, userID, email, handle, displayName string, now) (boardingFulfillment, error)`
  — normalizeHandle + displayName cleanup; already-seated check (→ErrBoardingAlreadySeated);
  handle-taken check; seatNumber via nextAssignableSeatNumber, reboard via userHasVacatedSeat →
  nextPost100SeatNumber; build airlockUser (no stripe), airlockSeat (no stripe fields,
  PricePaidCents=boardingDisplayPriceCents, TierPaid=tierPaidForSeat(seatNumber)), airlockEvent
  (kind BOARDED/RE_BOARDED, EventID = eventIDForBoarding(seatID)); LinkedVacatedSeats via
  vacatedSeatsToLinkForReboard. Add `const boardingDisplayPriceCents = 4200`. Add
  `func eventIDForBoarding(seatID string) string { return "EVENT-BOARDED-" + seatID }`.
  Keep: nextAssignableSeatNumber, nextPost100SeatNumber, userHasVacatedSeat, normalizeHandle,
  tierPaidForSeat, seatIDForNumber, vacatedSeatsToLinkForReboard, handlePattern, firstPost100SeatNumber.
- state_snapshot.go: remove StripeCheckoutSessionID/StripeCustomerID/StripePaymentIntentID from
  `airlockSeat`; remove the `airlockReservation` struct; remove `Reservations []airlockReservation`
  from `stateRecords`. (avatar fields added previously stay.)
- state_store.go: in loadRecords remove the boarding_reservation parsing branch and the seat
  stripe_* field scanning; remove any dynamoReservationRecord.
- config.go: remove StripeSecretKey + StripeWebhookSecret from Config and Load() (keep readSecret if
  still used by CrewAlertWebhookURL — it is). 
- boarding_test.go / boarding_rules_test.go: rewrite to cover Board + buildBoarding (happy path,
  already-seated idempotency returns Created:false, handle taken→409 mapping, reboard→POST_100).
  Remove all Stripe webhook/checkout/quote tests.
- After editing run `cd apps/api && go build ./...` (it's OK if app.go (Agent 3) hasn't caught up yet —
  if app.go references removed symbols that's expected and Agent 3 fixes it; report build status).

### Agent 3 — Go app wiring  [apps/api/internal/app/app.go]  (RUNS AFTER Agents 1 & 2)
- Remove `boardingHandler.StripeWebhook` routes (`/stripe/webhook` on apiCompat AND v2) and the
  `/boarding/checkout` registration. Remove StripeSecretKey/StripeWebhookSecret from the
  NewBoardingHandler config.
- Register `group.Post("/boarding/board", auth, middleware.RequireFeature("boarding"), boardingHandler.Board)`
  inside `registerBoardingRoutes`. (Keep avatar/mission/stack/admin/hatch routes untouched.)
- Then run `cd apps/api && go mod tidy` (drops stripe-go), `go build ./...`, `go vet ./...`,
  `go test ./...` and report. Keep the go1.24.4 toolchain pin intact.

### Agent 5 — Database (TS)  [packages/database/src/index.ts]
- Remove the three `stripe*` attributes from SeatEntity. Remove `BoardingReservationEntity` entirely
  and its entry in the `airlockService` Service map. Keep everything else (UserEntity avatar fields,
  CounterEntity, etc.).

### Agent 6 — Infra (TS)  [infra/secrets.ts, infra/api.ts]
- secrets.ts: remove `StripeSecretKey` and `StripeWebhookSecret` exports (keep EXAMPLE_API_KEY,
  CrewAlertWebhookUrl).
- api.ts: remove the `StripeSecretKey, StripeWebhookSecret` import and their two entries in the
  Function `link` array. Leave Bedrock + bucket perms + CrewAlertWebhookUrl intact.

### Agent 7 — Web (Astro)  [src/pages/manifest.astro, NEW src/pages/api/boarding/board.ts,
###   DELETE src/pages/api/boarding/checkout.ts, src/middleware.ts (CSP), grep for stripe/checkout refs]
- Read manifest.astro first. Rework the boarding UI by entitlement state (use
  `Astro.locals.auth().has({ feature: 'boarding' })` server-side):
  - signed out → existing sign-in CTA.
  - signed in WITHOUT the boarding feature → render `<PricingTable />` from `@clerk/astro/components`
    (set `client:load` if it needs hydration; `newSubscriptionRedirectUrl="/manifest?subscribed=1"`).
  - has feature but NO seat → show the handle + display_name form posting to `/api/boarding/board`
    (prefill display name from the Clerk user if available), then on success go to `/dashboard`.
  - already has a seat → link straight to `/dashboard`.
- NEW src/pages/api/boarding/board.ts: `export const prerender = false; export const POST = (ctx) =>
  forwardAuthenticatedJSON(ctx, "/api/boarding/board")` (JSON passthrough; the page collects handle +
  display_name as JSON, or convert form→JSON in the route). DELETE checkout.ts.
- Remove any client JS that posted to /api/boarding/checkout (grep airlock.js / inline scripts).
- middleware.ts CSP: KEEP the existing Stripe frame/connect entries (Clerk Billing renders Stripe
  payment UI). ADD Clerk checkout frame support: append `https://*.clerk.accounts.dev` to `frame-src`
  (dev) and a comment that production also needs the instance's `https://clerk.<APP_DOMAIN>`/accounts
  domain. connect-src already includes `https:` so Clerk frontend-API calls are fine.

------------------------------------------------------------------------
## VERIFY (must pass)
- `cd apps/api && go build ./... && go vet ./... && go test ./...` (toolchain go1.24.4)
- `cd apps/web && bun run typecheck`
- grep: no remaining `stripe`/`Stripe` references in apps/api (except possibly historical comments),
  none in infra link, none in web boarding flow. `go.mod` no longer requires stripe-go.

## Operator notes (document, can't enforce in code)
- In the Clerk dashboard: enable Billing, connect Stripe, create a "Boarder" plan, attach a
  `boarding` feature to it. The feature slug MUST be `boarding` to match the Go gate (or change the
  slug passed to RequireFeature/PricingTable consistently).
- The $42 in copy is now the Clerk plan price; keep them in sync manually.
