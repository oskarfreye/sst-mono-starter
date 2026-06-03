# DASHBOARD · "QUARTERS & COMMAND CONSOLE" — DESIGN

**Date:** 2026-06-04
**Approver:** Oskar Freye (oskar@fr3n.tech)
**Status:** Approved through §6. Ready for `/init-project` pickup.
**Parent spec:** `apps/web/SPEC.md` (THE AIRLOCK · SPEC v1) — this design details the `/dashboard` route, which the parent spec defines only as a one-liner ("declare mission, submit proof, manage stacks").

---

## 0. WHAT THIS IS

The behind-auth home base a boarder receives the moment they sign the manifest — *their quarters and personal command console aboard the ship.* The public hull stays a cold registry; this is the one layer that is personal, connected, and gamified. The aim is a place worth opening **daily** across the 30-day cycle, not a one-time form.

Register decision: **the ship from the inside.** Cold, in-world, mechanical — but *theirs*. Public is the hull and the airlock door; the dashboard is the cabin with the controls.

---

## 1. CONSCIOUS SPEC OVERRIDES

This design deliberately reverses two locked v1 decisions from `SPEC.md` — **only behind auth.** The public registry is untouched.

| Override | What the parent spec said | Decision here |
|----------|---------------------------|---------------|
| **Achievement tree** (gamification) | §13: *"Streak as a load-bearing mechanic… don't gamify."* | Overridden **privately only.** A trophy-wall tree lives behind auth. Public profile shows no trophies. |
| **Community + Discord** | §0/§13/FAQ#12: *"not a community… Public chat / forum / Discord… The Airlock is a registry, not a community."* | Overridden **privately only.** A crew activity feed + an off-site Discord link live behind auth. No in-app forum is built. |
| **Profile-view analytics** | (not addressed in parent spec) | **Additive, no override needed.** Cold telemetry fits the voice. |

Guardrail: gamification and the crew door must never bleed into the public hull, which is what does the selling.

---

## 2. GOAL & NON-GOALS

**Goal.** A private, in-world command console + quarters that makes the 30-day cycle feel alive and personal, and gives a paying boarder reasons to return daily.

**Non-goals (v1):**
- **Public stays cold.** Gamification is behind-auth only; nothing leaks to `/c/[handle]` or the registry.
- **No teeth on the tree.** Trophy wall = record only. No cosmetics, no perks, nothing that softens the floor consequences.
- **No in-app chat.** Discord is an off-site door we link to; no forum inside the ship.
- **Not a mechanic rewrite.** Same declare → proof → stack flow as today's dashboard, restructured — not reinvented.

---

## 3. ARCHITECTURE — FOUR-TAB CONSOLE

**Persistent top strip** (every tab): brand mark → `@handle` · `THE 100 · SEAT 04` seat-pill · admin link when applicable. Identity always pinned.

**Tabs** (names are a proposal, not load-bearing):

| # | Label | Role | Reuses |
|---|-------|------|--------|
| 1 (default) | `CONSOLE` | Cockpit. Live hatch countdown, current mission state, the one action owed next (declare / submit proof), stacks. State-machine driven. | existing dashboard forms + `/api/me/airlock` |
| 2 | `RECORD` | Trophy-wall achievement tree. Cold ledger; nodes light up as earned. | new (computed) |
| 3 | `QUARTERS` | Edit public face (display name, bio, suit/avatar) + cold telemetry: profile-view count. | existing avatar/profile forms + new counter |
| 4 | `CREW` | Crew-wide activity feed (boardings, ships, airlocks) + Discord door. | existing `EventFeed` + `/api/events` |

**Why Console is default:** the clock you can't move greets you first. The other three are rooms you walk into.

**Console cycle states** (the tab must handle all six):
no seat → boarded-undeclared → declared-active → proof-pending → confirmed-idle → airlocked.

---

## 4. THE RECORD — ACHIEVEMENT TREE TAXONOMY

A constellation of nodes in six branches. Earned nodes light (ice/signal); unearned sit dim/outlined. Every node is **derivable from existing data** (missions, seats, stacks, events) → computed, never hand-maintained.

**Branch I · BOARDING**
- `MANIFEST SIGNED` — boarded (seat assigned)
- `THE 100` — seat ≤ 100
- `SUIT ON` — uploaded a face

**Branch II · SHIPPING** *(the spine)*
- `FIRST SHIP` — 1 confirmed
- `REPEAT SHIPPER` — 3 confirmed
- `VETERAN` — 5 confirmed
- `PROLIFIC` — 10 confirmed

**Branch III · STREAK**
- `BACK TO BACK` — 2 confirmed in a row, no airlock
- `HOT STREAK` — 3 in a row
- `UNBROKEN` — 5 in a row

**Branch IV · COURAGE** *(stacks armed)*
- `CREW ALERTED` — Crew Alert armed on a live mission
- `WAKE ARMED` — The Wake armed
- `ALL IN` — both stacks armed at once

**Branch V · REDEMPTION** *(the dark branch — honest about failure)*
- `AIRLOCKED` — first miss (lights in hatch-orange, not hidden)
- `RE-BOARDED` — bought back after a death
- `REDEEMED` — completed the re-entry challenge

**Branch VI · LONGEVITY**
- `30 DAYS ABOARD`
- `100 DAYS ON THE BOAT`
- `YEAR ONE`

~19 nodes total. The dark Redemption branch is the on-brand risk and is an approved, conscious permanent choice (honest > flattering).

---

## 5. DATA & API ADDITIONS

| Need | Backend work | New? |
|------|-------------|------|
| Console tab | Served by existing `/api/me/airlock`. Front-end restructure only. | No |
| Record tab | `GET /api/me/achievements` — computes earned/locked node states from existing data. Stateless, cacheable, no new storage. | New (read-only) |
| Quarters — edit | Existing `/api/me/profile` + avatar endpoints. Move, don't rebuild. | No |
| Quarters — analytics | Profile-view counter: increment on each public `/c/[handle]` load; read via `GET /api/me/analytics`. Needs a counter + bot throttle/dedup. | New (write path) |
| Crew tab | Existing `/api/events`. | No |
| Discord door | Config URL (env var). | Trivial |

Only two genuinely new surfaces: **computed achievements** (read-only, low risk) and the **profile-view counter** (a write on every public profile hit — the one item needing care).

---

## 6. v1 SCOPE vs LATER

**v1 — ships:**
- Four-tab restructure of `/dashboard` with persistent identity strip.
- **Console:** existing mission/proof/stack forms reorganized around a live hatch countdown; all six cycle states handled.
- **Record:** computed trophy-wall tree, ~19 nodes, six branches. `/api/me/achievements`.
- **Quarters:** existing profile + avatar forms moved here + profile-view counter (`/api/me/analytics`).
- **Crew:** existing `EventFeed` (crew-wide) + Discord link.

**Later (v1.1+) — explicitly not now:**
- Cosmetic unlocks / suit skins hanging off tree nodes (schema stays open for it).
- Referrer / geo analytics beyond a raw view count.
- Per-node share-to-socials.
- Selectively surfacing tree nodes on the public profile as flex.
- Real-time push on the Crew feed (SSE) — v1 may poll.

---

## 7. RISKS & OPEN QUESTIONS

1. **Dark Redemption branch** — own `AIRLOCKED` is a permanent private node. Approved, conscious, permanent.
2. **View-counter integrity** *(open)* — bots inflate raw counts. Recommendation: per-IP throttle + accept rough numbers (voice can own the roughness). Decide: throttle at launch, or ship raw and refine?
3. **Discord ownership** *(open)* — a paid door to a dead server is worse than no door. Who owns/moderates day one?
4. **Gamification ↔ identity drift** — tree + Discord + analytics steps toward "community/course," which the pitch rejects. Contained by the "public stays cold" non-goal; re-check each time the private layer grows.
5. **Tab naming** — `CONSOLE / RECORD / QUARTERS / CREW` is a proposal; finalize at implementation.

---

## 8. NEXT STEP

Existing codebase → **`/draht:init-project`** (or `/draht:plan-phase` if planning structure already exists) picks up this spec to produce execution plans. Most of v1 is a front-end restructure of forms that already work, plus wiring two new read endpoints and one counter.
