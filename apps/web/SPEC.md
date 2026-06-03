# THE AIRLOCK · SPEC v1

> Single source of truth for the landing page, profile pages, mechanic, pricing, and data model. Implementation translates this — nothing else.

**Status:** All design and copy decisions locked 2026-05-27. Implementation pending. Start with section 15.

---

## 0. WHAT THIS IS

The Airlock is a deadline-and-public-log system for software founders who don't ship. A boarding pass buys a seat on a public roster. Every seat-holder declares one app they'll ship in 30 days. Miss the hatch and the system publishes it — permanently, under your real handle, on a page anyone can link to.

The product is not a course, a community, or a habit tracker. It is a public record that makes hiding expensive.

---

## 1. AUDIENCE & POSITIONING

**Built for:** founders and indie hackers building software people pay for. SaaS, AI tools, dev tools, marketplaces, consumer apps. Anyone whose unit of work is a deployed app that takes money.

**Rejects:** people without an idea yet, creators (essays / video / devlogs), freelancers, course-builders, productivity tourists, anyone whose "ship" isn't a URL with a buy button.

**Pitch (one sentence):** *Sign the manifest. Ship one app in 30 days, with a working buy button. Or your handle goes on the public airlock log forever.*

**Closest comp:** ship-or-die.com. They sell a 30-day Discord community for $249 one-time. We sell a 30-day public profile + permanent log for $42–$336 one-time. Their consequence is being kicked out of a Discord. Ours is a public registry under your real name.

---

## 1.5 DECISIONS LOCKED (2026-05-27)

The full decision log. If a future ambiguity arises, this is the authority.

**Copy / page presentation**
- L1. Mirror line *"Twitter threads about your stack aren't shipping"* — **KEEP.**
- L2. The Wake stack closer *"Probably the only one that actually works"* — **KEEP.**
- L3. Launch-day numbers — **HONEST.** Page ships with real small numbers: Oskar as seat 01 (DECLARED), every other seat open, fleet metrics genuinely tiny. No fake names, no padded counts.

**Mechanic edge cases**
- L4. Early ship — confirmed mission on day 5 does **not** bank remaining days. Boarder can coast or declare a new mission. Missions are strictly sequential, one per seat at a time.
- L5. Late proof grace — proof submitted **after** hatch close does **not** count. The hatch is the hatch.
- L6. Declaration URL ≠ proof — even if the URL declared in step 02 already has a working buy button on day 1, proof requires a **separate explicit submission** before the deadline. Prevents gaming.
- L7. Verification visibility — `PROOF PENDING` (manual admin review state) is **publicly visible** on the mission log during the review window. Transparency over polish.

**Profile + URLs**
- L8. Handle URL on re-board — single canonical `/c/[handle]` URL per person. Seat history (original seat, vacated date, current seat) rendered inside the profile. No sub-paths per seat.
- L9. Avatars — **auto-generated pixel avatars only**, seeded deterministically by handle. No custom uploads in v1.

**Payments**
- L10. Donation penalty stack — **deferred to v1.1.** v1 ships with two optional stacks only: CREW ALERT and THE WAKE. Card-on-file + future auto-charge is real compliance work, punt.
- L11. Refund policy — refund **only** when our system caused a false airlock. GDPR delete = account deletion but no refund (service was rendered). Fraud chargebacks = let the bank dispute, do not engage. Codified in Stripe terms, not surfaced on the page.

**The Wake**
- L12. The Wake post format — **preset auto-template** (`@handle · mission · date · airlocked`). No boarder-written custom text.
- L13. Cross-post to socials — **deferred to v1.1.** v1 posts only on `theairlock.space/airlock`. Manual share-to-socials button is acceptable (boarder clicks themselves); no OAuth-held tokens or auto-posting under linked accounts.

---

## 2. VOICE

- **Sci-fi mechanical.** The page is a ship, a manifest, a hatch, a registry. Vocabulary stays inside that world.
- **Dry. Cold. Brief.** Sentences end before they're comfortable. Two-word answers in the FAQ are correct.
- **Specific over general.** *"Stripe still in test mode"* beats *"haven't shipped yet."* *"247 commits and zero users"* beats *"lots of work, no traction."*
- **No softening, no hedging.** *"No refund. Ship or airlock."* not *"All sales final — but we're flexible if circumstances arise."*
- **No motivational mode.** The page is not on the reader's side. It's the airlock door.
- **Recurring motifs to use deliberately:**
  - *"Two weeks away"* — the lie. Appears in the Mirror and in the founder block as a callback.
  - *"No passengers. Only builders."* — the page's tagline.
  - *"Hatches not stacks."* — dismisses the "what tech do you use" objection.
  - *"Hiding is expensive."* — the closing line of the FAQ. Best mic-drop.
- **References we use without explaining:**
  - **$42** — Hitchhiker's. Never named.
  - **THE 100** — TV show. Never named. The brand is already sci-fi, the reference layers cleanly.
  - **Marc Lou** — named directly once in the founder block. Calculated. Signals we know the landscape.

---

## 3. THE MECHANIC

### 3.1 The Rule (in one sentence)

*30 days from boarding to ship one declared app. Miss the hatch and you're airlocked.*

### 3.2 The Mission — strict floor

The mission must be one of:

- **A new app:** deployed at a public URL, a stranger can sign up, **payment is live** (Stripe / Lemon Squeezy / equivalent), OR a paid waitlist that captures intent (i.e. takes a credit card hold, not just an email).
- **A customer-facing release of an existing app:** a feature a user would notice and pay for, behind a paywall or unlocking a paid tier. Refactors, internal tools, and infra work do not count.

Always: one shippable artifact, one URL, one sentence of declaration.

### 3.3 The 30-Day Clock

- Signing the manifest commits the boarder to declaring a mission within **24 hours**.
- The 30-day clock starts at mission declaration.
- The clock does not pause for any reason — illness, family, life, server outages.
- The mission cannot be edited after declaration. The hatch is built around it.

### 3.4 Proof

Proof is a URL. The crew clicks it. They reach a working sign-up and a way to pay. A stranger could give the boarder a dollar without asking a question.

- Proof must be submitted before the hatch closes.
- Proof gets a brief verification window (`PROOF PENDING` status).
- Verification confirms only: URL works, sign-up functions, payment or paid waitlist is live.

### 3.5 Outcomes at Hatch

Two possible states:

- **CONFIRMED** — proof submitted, verified. Seat held. Boarder can declare a new mission whenever they want, or coast on the win.
- **AIRLOCKED** — no proof, or proof rejected. Floor consequences trigger automatically.

### 3.6 Consequences — THE FLOOR (automatic, two cards)

**F · 01 · MARKED**
A permanent `AIRLOCKED · [mission name] · [date]` stamp on the boarder's public profile. Stamps stack — multiple airlocks all show, in order. Nothing removes them.

**F · 02 · GROUNDED**
The seat is forfeit. The boarder is off the boat. To re-board, they buy a new pass at **full price ($336)**, regardless of which tier they originally paid. They come back as a new seat number — never the original 100 again.

### 3.7 Optional Stacks — extra pressure, picked at board

**v1 ships with two stacks. DONATION PENALTY is deferred to v1.1 (decision L10).**

**S · 01 · CREW ALERT** *(v1)*
Up to 8 named people (cofounder, partner, ex-boss, group chat). They get a direct push at the moment of hatch closing on a miss.

**S · 02 · THE WAKE** *(v1)*
A short automatic post on `theairlock.space/airlock` announcing the miss — preset auto-template (`@handle · mission · date · airlocked`). v1 posts on the airlock page only; manual share-to-socials button is allowed, but no OAuth auto-cross-post (decision L13).

**S · 03 · DONATION PENALTY** *(v1.1, deferred)*
Auto-charge a pre-set amount on miss, to a charity (or anti-charity) the boarder names at boarding. Requires Stripe card-on-file + future auto-charge. Not in v1.

**Page-copy note:** the landing page (section 6.5) still shows three stack cards in `[ 02 ] AIRLOCK PROTOCOL`, with DONATION PENALTY explicitly marked `COMING IN v1.1` or rendered in a muted/locked visual state. The card stays as part of the design promise; the implementation just hasn't shipped.

### 3.8 Redemption — The Re-entry Challenge (one note)

After re-boarding, a boarder may opt into the re-entry challenge: ship two consecutive missions in a row, no airlock. On completion, every prior `AIRLOCKED` stamp gets a `[REDEEMED]` tag appended. The stamps themselves stay forever — the comeback gets witnessed, the failure doesn't get erased.

---

## 4. PRICING & THE 100

### 4.1 The Ladder

Doubling tiers, all multiples of $42 (powers of 2 × $42):

| Tier | Price | Seats | Range |
|------|-------|-------|-------|
| 01 | $42 | 10 | 01–10 |
| 02 | $84 | 15 | 11–25 |
| 03 | $168 | 25 | 26–50 |
| 04 | $336 | 50 | 51–100 |
| ∞ | $336 (flat) | unlimited | 101+ |

Each boarder pays the price of their tier. The CTA on the page always reflects the *next* seat's price. The ladder ticks visibly as seats fill.

### 4.2 Lifetime Pass

A boarding pass is **one-time payment, lifetime seat**. There is no recurring billing.

A lifetime pass entitles the boarder to:

- A persistent public profile at `/c/[handle]`.
- Unlimited 30-day missions (one at a time, sequentially — no parallel missions).
- Full access to the mission log, the cabin grid, the event feed.

A lifetime pass does **not** entitle the boarder to skip the floor consequences on airlock. The pass is for boarding; staying on the boat is earned by shipping.

### 4.3 Death & Re-board

- Airlocking ends the lifetime pass. The seat is vacated.
- To return: purchase a new pass at the **current flat price ($336)**, regardless of which tier the original pass was bought at.
- The new pass comes with a new seat number (the next sequential number — 101, 102, etc. once THE 100 is sealed). The original seat number is never re-occupied.
- A re-boarder's profile retains its full history: original seat, all confirmed launches, all `AIRLOCKED` stamps, and a `RE-BOARDED AS SEAT [N]` link to the new seat.

### 4.4 THE 100 — The Original Cohort

THE 100 is the cohort of seats 01 through 100, ordered by boarding sequence. Membership is fixed by history:

- A boarder enters THE 100 by being among the first 100 to ever board.
- A boarder exits THE 100 by airlocking. Their seat is vacated and stays vacated forever.
- A vacated seat is **never refilled.** The grid keeps the empty square as a memorial.
- Post-airlock re-boarders return as seats 101+ — they are *not* in THE 100.

Seat number is part of identity. `THE 100 · SEAT 04` is a permanent mark. Seat 01 is reserved for Oskar (the founder).

### 4.5 Memorial Seats

When a THE 100 seat is vacated:

- The square on the 10×10 grid shows a struck-through pixel (`✕` or equivalent visual treatment).
- Hovering / clicking the square shows: former occupant's handle, the mission they died on, the date.
- The former occupant's profile retains the `THE 100 · SEAT [N] · VACATED [date]` line as part of its history.
- If the former occupant re-boards as a new seat, their original profile links forward: `RE-BOARDED AS SEAT [N]`.

The page literally decays over time. This is intentional.

---

## 5. SITE MAP

| URL | Purpose |
|-----|---------|
| `/` | Landing page |
| `/c/[handle]` | Public crew profile |
| `/airlock` | The Wake — public log of all airlock events |
| `/manifest` | Boarding flow (auth + payment + mission declaration) |
| `/dashboard` | Private boarder dashboard (declare mission, submit proof, manage stacks) |
| `/api/*` | Go Fiber Lambda backend (existing) |

Public, indexable, no auth required: `/`, `/c/[handle]`, `/airlock`.
Auth-gated: `/manifest`, `/dashboard`.

---

## 6. LANDING PAGE (`/`)

Top-down section-by-section. Numbered labels are visible on the page. The Mirror has no label by design.

### 6.0 TOP HUD (existing, keep)

- Brand mark + `THE AIRLOCK · v1.0` (left)
- Status pill `SHIP NOMINAL` (right)
- CTA button `Join the Crew → ` — updated to `Sign the Manifest · $42 → ` (or current next-seat price)

### 6.1 HERO — cinematic stage (existing)

Pinned cinematic canvas with sequence:

- `LAUNCH` (default)
- `OR AIRLOCK` (alt, on scroll)
- Subtitle `your app in 30 days` → `miss the deadline, get kicked out forever.`
- Scroll prompt `↓ scroll to start`

No copy changes. The visual sequence carries the hook.

### 6.2 MIRROR — no label, no heading, pure prose

Centered narrow column (`max-width: 620px`). Slots directly under the hero, before the anchor section. The HUD pauses here — no labels, no panels, no canvases. The page interrupts itself.

```
"I'm two weeks away."

You said that two weeks ago. And eight weeks before that.

247 commits. Zero users. Stripe still in test mode. The waitlist page is in its sixth Figma. The launch date is a calendar event that keeps getting dragged into next month.

"After auth."  "After the redesign."  "After the AI feature is in."

Polishing in private isn't shipping. Twitter threads about your stack aren't shipping. Pushing to `main` of a domain nobody knows isn't shipping.

You don't need another month.
You need a public hatch and a date you can't move.
```

### 6.3 ANCHOR SECTION — transition + `THE 100 · LIVE` panel

Two-column layout (existing `.anchor-row`).

**Left column (copy):**

- Label: `▲ PUBLIC ACCOUNTABILITY · BOARDING PROTOCOL`
- Heading: `No passengers.` / `<em>Only builders.</em>`
- Body: *A public roster of founders shipping one app every 30 days. Seats are numbered, missions are declared, hatches are enforced. When someone misses, the page records it — under their real handle, forever.*
- CTAs: `Sign the Manifest · $42 →` (primary), `See the Mission Log` (ghost)

**Right column (panel — replaces HATCH CONTROL):**

```
THE 100 · LIVE                                ● LIVE
─────────────────────────────────────────────────────
 ■ □ □ □ □ □ □ □ □ □
 □ □ □ □ □ □ □ □ □ □
 □ □ □ □ □ □ □ □ □ □
 □ □ □ □ □ □ □ □ □ □       ■ OCCUPIED   03
 □ □ □ □ □ □ □ □ □ □       ✕ VACATED    00
 □ □ □ □ □ □ □ □ □ □       □ OPEN       97
 □ □ □ □ □ □ □ □ □ □
 □ □ □ □ □ □ □ □ □ □
 □ □ □ □ □ □ □ □ □ □
 □ □ □ □ □ □ □ □ □ □
─────────────────────────────────────────────────────
NEAREST HATCH IN FLEET
@priya · SEAT 04 · 08h 14m until airlock
─────────────────────────────────────────────────────
NEXT SEAT · $42 · TIER 01
[ SIGN THE MANIFEST → ]
─────────────────────────────────────────────────────
● NO HEROES, NO HIDING                          UTC
```

The grid is the page's defining visual. Live counts. The "nearest hatch in fleet" is whoever in the active crew has the closest deadline — their handle is publicly visible, ticking down.

### 6.4 [ 01 ] THE 30 DAYS

**Label:** `[ 01 ] THE 30 DAYS`
**Heading:** `Declare one app. Ship it in 30 days. <em>Or face the airlock.</em>`
**Right subtitle:** *Four steps. One rule. The clock doesn't extend, the mission doesn't change, the floor doesn't soften.*

**STEP 01 · BOARD**
*Sign the manifest.*
Pay the next-seat price. A public profile goes live at `theairlock.space/c/[you]`. Pick your optional stacks if you want extra rope. The 30-day clock starts the moment you declare your mission.

**STEP 02 · DECLARE**
*One app. One URL. One sentence.*
A new app, or a customer-facing release of an existing one. Has to be live within 30 days. Has to take money — Stripe checkout or a paid waitlist counts. *"Refactor the backend"* doesn't. *"Ship Stripe checkout on v1.beta"* does. You don't renegotiate it.

**STEP 03 · SHIP**
*Drop the URL before the hatch.*
The crew clicks it. They reach a working sign-up. They reach a way to pay. A stranger could give you a dollar without asking you a question. Proof beats intention. A live Stripe key beats a sixth Figma.

**STEP 04 · OR**
*The hatch doesn't argue.*
Miss the deadline and you're airlocked. Profile marked. Seat lost. To come back, you buy a new boarding pass at full price ($336). The next section explains what that means and why we don't soften it.

### 6.5 [ 02 ] AIRLOCK PROTOCOL

**Label:** `[ 02 ] AIRLOCK PROTOCOL`
**Heading:** `When you miss, two things happen. <em>Every time.</em>`
**Right subtitle:** *The floor is automatic — you bought it with your boarding pass. The stacks are extra pressure you add yourself, at board, when you know you'll need more rope.*

#### THE FLOOR — mandatory, automatic

**F · 01 · MARKED**
*Profile marked AIRLOCKED.*
Your public profile gets a permanent `AIRLOCKED · [mission] · [date]` stamp. Future cycles can stack proof above it; nothing erases it. Friends, rivals, and Google all see it.

**F · 02 · GROUNDED**
*Seat forfeit. Re-board costs $336.*
You're off the boat. To come back, you buy a new boarding pass at full price — regardless of which tier you originally paid. You return as a new seat number. Your original seat is vacated forever, marked with your name on the public grid.

#### OPTIONAL STACKS — extra pressure, picked at board

**S · 01 · DONATION PENALTY**
Auto-charge a pre-set amount to a charity — or an anti-charity — you'd rather not fund. Skin in the game, denominated in dollars.

**S · 02 · CREW ALERT**
Nominate up to 8 people (cofounder, partner, ex-boss, group chat). They get a direct push the moment your hatch closes on a miss. Accountability isn't anonymous.

**S · 03 · THE WAKE**
A short, automatic post on `theairlock.space/airlock` announcing your miss — handle, mission, date. Optional cross-post to your linked socials. The bravest stack. Probably the only one that actually works.

#### THE RE-ENTRY CHALLENGE — optional redemption

*Ship two missions in a row after you re-board and every prior AIRLOCKED stamp gets a `[REDEEMED]` tag. The failure stays — the comeback gets witnessed.*

### 6.6 [ 03 ] MISSION LOG

**Label:** `[ 03 ] MISSION LOG`
**Heading:** `Every seat. Every mission. <em>Every airlock. Public.</em>`
**Right subtitle:** *A read-only window into what THE 100 is shipping right now. No filters, no PR — current missions, hatch countdowns, and every seat that's been vacated. The page that tells the truth.*

**Tabs:**
```
ACTIVE · [n]    VACATED · [n]    ALL HISTORY    SHOWING [n] OF [n] · LIVE
```

**Table columns:**
```
SEAT          CREW MEMBER          CURRENT MISSION                       HATCH                STATUS
```

**Sample rows (replace with live data; these are mocks for design):**

```
THE 100 · 01    @oskar  founder      Ship Airlock v1.0 + Stripe checkout      in 18d 14h           DECLARED
THE 100 · 02    @maya   founder      Ship Bolt AI feature behind paywall      in 12d 03h           DECLARED
THE 100 · 03    @devon  indie dev    Ship dev tool v0.1 + paid waitlist       in 06d 18h           DECLARED
THE 100 · 04    @priya  founder      Ship marketplace v1 + Stripe             TODAY 23:59          PROOF PENDING
THE 100 · 05    @aki    indie dev    Ship AI tool launch + paid signup        hatched 06h ago      CONFIRMED
THE 100 · 06    @renaud founder      Ship landing page redesign               hatched 18h ago      AIRLOCKED — seat vacated · re-board $336
```

**Status semantics:**
- `DECLARED` — manifest signed, mission declared, no proof yet, deadline future.
- `PROOF PENDING` — proof submitted, awaiting verification.
- `CONFIRMED` — shipped, verified, mission complete.
- `AIRLOCKED` — missed the hatch. Row tinted orange. Stays in `ACTIVE` for 24h, then moves to `VACATED`.

**Visual treatment:**
- `AIRLOCKED` rows get `var(--hatch)` background tint at ~3% opacity.
- Seat number cell on AIRLOCKED row renders struck-through.
- `PROOF PENDING` rows get a faint pulse animation on the status badge.

### 6.7 [ 04 ] THE CABIN (was CREW MODE — full rewrite)

**Label:** `[ 04 ] THE CABIN`
**Heading:** `Small room. <em>Numbered seats. Empty chairs.</em>`
**Right subtitle:** *The 100 has 100 seats. When a seat is vacated, it stays vacated forever. The page literally shrinks over time. The names of the dead stay on the grid.*

**Main visual:** Full-width 10×10 seat grid (much larger than the hero panel version). Each square is interactive.

**Seat states:**
- `OCCUPIED` — solid pixel, hatch color or signal color depending on current status. Hover shows: handle, current mission, hatch countdown, click → profile.
- `OPEN` — empty outline. Hover shows: `SEAT [N] · NEXT TO FILL AT $[price]`. Click → boarding flow.
- `VACATED` — struck-through pixel. Hover shows: former occupant handle, mission they died on, date vacated. Click → memorial profile.

**Below the grid:**
- A small legend.
- A short paragraph: *Active boarders → [n]. Vacated → [n]. Open seats → [n]. After seat 100 is sealed, new boarders fill seats 101+ at $336 — they don't enter this grid.*

### 6.8 [ 05 ] WHAT GETS SHIPPED (was WHAT GETS LAUNCHED)

**Label:** `[ 05 ] WHAT GETS SHIPPED`
**Heading:** `The mission is yours. <em>The proof is non-negotiable.</em>`
**Right subtitle:** *Four mission archetypes. The Airlock doesn't care which one you pick — only that you ship it with a working buy button.*

Four cards, each one a mission archetype (replaces the old persona-based cards):

**MISSION TYPE 01 · NEW SaaS**
*"Ship a new SaaS. v1, Stripe live, first dollar earned."*
Proof: deployed URL + Stripe checkout reachable.

**MISSION TYPE 02 · NEW AI TOOL**
*"Ship a new AI tool or wrapper. Paid signup or paid waitlist mandatory."*
Proof: deployed URL + paid waitlist or live checkout.

**MISSION TYPE 03 · MAJOR RELEASE**
*"Ship a customer-facing release behind a paywall on an existing product."*
Proof: deployed feature + paywall behavior verified.

**MISSION TYPE 04 · SMALL BET**
*"Ship one feature with one buy button. Validate before you build."*
Proof: deployed URL + payment flow.

### 6.9 [ 06 ] SHIP TELEMETRY

**Label:** `[ 06 ] SHIP TELEMETRY`
**Heading:** `The crew never sleeps. <em>The log never stops scrolling.</em>`
**Right subtitle:** *Every boarding, every launch, every airlock — broadcast across the ship in real time.*

**Left panel: EVENT FEED · LIVE**

Real-time event stream. Each entry: `[timestamp] [icon] [handle] [event] [tag]`.

Sample entries (replace with live data):

```
18:42:07  ▲ @aki     LAUNCH CONFIRMED · AI tool live with paid signup       SEAT 05 · +CONFIRMED
18:38:51  ▶ @priya   proof attached · marketplace.beta + stripe              SEAT 04 · PENDING REVIEW
18:21:13  ▼ @renaud  AIRLOCKED · no proof before hatch                       SEAT 06 · VACATED
17:46:19  → @devon   mission declared · dev tool v0.1 + paid waitlist        SEAT 03 · DECLARED
17:33:08  ▲ @maya    launch confirmed · Bolt AI feature paywall              SEAT 02 · +CONFIRMED
17:21:55  ◯ @kel     boarding pass purchased · seat 07 taken                 THE 100 · +1
```

**Right panel: FLEET METRICS · LIVE**

Honest, small numbers at launch. Don't fake.

```
SEATS OCCUPIED       03 / 100
SEATS VACATED        00
LAUNCHES CONFIRMED   00          (this week / lifetime)
NEXT TIER OPENS      $84 in 7 seats
```

Drop the old "cycle progress" bar — there is no shared cycle. Replace with: *next tier opens in [n] seats · price climbs to $[next-tier]*.

### 6.10 [ 07 ] BOARDING PASS

**Label:** `[ 07 ] BOARDING PASS`
**Heading:** `Sign the manifest. Take your seat. <em>Start the clock.</em>`
**Right subtitle:** *$42 buys the first seat. Every tier doubles the last. Ship and the seat is yours for life. Die and you buy back at full price.*

**Left card — pricing:**

```
NEXT SEAT · $42
SEAT 04 OF THE 100 · 96 SEATS LEFT IN THE ORIGINAL COHORT

What it gets you:
- A public crew profile at theairlock.space/c/[you]
- A lifetime boarding pass — unlimited 30-day missions, forever
- A 30-day hatch clock the system actually enforces — no extensions, no exceptions
- Live access to the mission log, crew deck, and event feed
- THE 100: the first 100 boarders carry a permanent THE 100 · SEAT [N] mark
- No refund. Die on a mission and the seat is forfeit — re-boarding costs $336, regardless of where the ladder is.

[ SIGN THE MANIFEST · $42 → ]

Signing commits you to declaring a mission within 24 hours. Once the mission is declared, the 30-day clock is non-negotiable.
```

**Right panel — price ladder, live:**

```
PRICE LADDER · LIVE

TIER 01 · $42       ███░░░░░░░    3 / 10     SEATS 01–10
TIER 02 · $84       ░░░░░░░░░░    0 / 15     SEATS 11–25
TIER 03 · $168      ░░░░░░░░░░    0 / 25     SEATS 26–50
TIER 04 · $336      ░░░░░░░░░░    0 / 50     SEATS 51–100
                                  ──────────
                                  3 / 100 · THE 100

▸ NEXT BUMP IN 7 SEATS · TIER 02 OPENS AT $84
▸ AFTER SEAT 100: price flattens at $336 · THE 100 closes forever
```

### 6.11 [ 08 ] BUILT BY THE GUY IN SEAT 01

**Label:** `[ 08 ] FOUNDER`
**Heading:** `Built by the guy in seat 01.`

Single card layout (left: portrait or pixel-avatar at seat 01, right: copy + three product receipts).

**Copy:**

```
Oskar Freye — software engineer. Co-founder of fr3n.fan and creavings.com. Freelance at freye.tech.

I'm not Marc Lou. I don't have 30 apps and $50K MRR to wave at you.

What I have is the specific knowledge that "I'm two weeks away" is the most expensive lie a builder tells themselves. I was two weeks away for four months. The branch was open. The landing page was in its third Figma file. The deploy was almost configured. None of it was shipping.

The mechanic that broke me out of it was unreasonable: declare in public, set a date you can't move, pay the cost in front of people when you miss. The Airlock is the room that enforces it.

I board with you. Same 30-day clock. Same public log. Same airlock if I miss. If I die, I pay $336 to come back. Seat 01 won't come with me.

Seat 01 is mine. Seats 02 through 100 are open. After that, the original cohort is sealed.
```

**Three receipt cards below body:**
- `fr3n.fan` — Co-founder, 2024 — thumbnail + link
- `creavings.com` — Co-founder, 2025 — thumbnail + link
- `freye.tech` — Freelance studio — thumbnail + link

### 6.12 [ 09 ] BRIEFING (FAQ)

**Label:** `[ 09 ] BRIEFING`
**Heading:** `Read before you sign the manifest.`
**Right subtitle:** *Twelve objections the airlock has heard before. If yours isn't here, you're either the first — or you're not asking the right one.*

Twelve Q/A pairs, single-column, narrower column width (`max-width: 720px`).

**1. What if I don't have an idea yet?**
Wrong room. The Airlock isn't where you find a mission — it's where you ship one. Come back when you can finish the sentence *"In 30 days I will ship ___."*

**2. Can I change my mission mid-cycle?**
No. You declared it. The hatch is built around it. If your real idea shows up on day 12, write it down and board it next cycle.

**3. What counts as shipping?**
A URL a stranger can visit, sign up to, and pay you — Stripe, Lemon Squeezy, paid waitlist. If you have to explain why it doesn't have a buy button yet, it's not shipped.

**4. Does an existing app count?**
Yes, if the cycle ships a customer-facing release a user would notice and pay for. *"I refactored the backend"* doesn't count. *"I shipped the AI feature behind a paywall on production"* does.

**5. Do I need to be a technical founder?**
You need a working app live at a URL in 30 days. How it got built — code, no-code, vibe-coded with Claude — is your business. The Airlock cares about hatches, not stacks.

**6. What if real life happens — emergency, illness, kid?**
The clock doesn't pause. We're not unsympathetic, but every pause is a back door, and back doors break the mechanic for everyone else. The 30 days are 30 days.

**7. Can I extend the deadline?**
No.

**8. Can I get a refund?**
If you're asking, you're already drifting.

**9. What is THE 100?**
The first 100 builders ever boarded. The original cohort. The mark stays on your profile forever — alongside every launch and every airlock event. After seat 100, the door seals. Nobody else gets the badge, regardless of price.

**10. What happens after seat 100?**
Price flattens at $336. New boarders pay it every time, forever. THE 100 is sealed. You don't join later — you missed the boat.

**11. If I'm airlocked, does my THE 100 mark stay?**
No. THE 100 is about active seats in the original cohort. When you die, your seat is vacated and you exit THE 100. If you re-board at $336, you come back as a new seat — never the original 100 again. The vacated seat stays on the public log forever, marked with your name and the date you went out.

**12. Is this a course?**
No. There's nothing to learn. The Airlock is a 30-day deadline, a public profile, a system that publishes when you miss, and a room with 99 other people doing the same thing. Courses let you feel productive while staying hidden. This makes hiding expensive.

### 6.13 FINAL CTA (existing — minor rewrite)

**Label:** `● FINAL TRANSMISSION`
**Heading:** `The ship is moving. <em>Are you boarding?</em>`
**Body:** *Sign the manifest. Declare your mission. Ship in 30 days. Or watch the hatch open from the wrong side of the glass.*
**Buttons:** `[ Sign the Manifest · $42 → ]` (primary), `[ See the Mission Log first ]` (ghost)
**Micro under buttons:** `NO PASSENGERS · ONLY BUILDERS · THE 100 IS BOARDING`

### 6.14 FOOTER (existing — minor edits)

- `© THE AIRLOCK · theairlock.space`
- Links: `PROTOCOL · MISSION LOG · THE CABIN · THE WAKE · CHANGELOG · CONTACT`
- Status: `SHIP NOMINAL`

---

## 7. PROFILE PAGE (`/c/[handle]`)

Public, indexable. Shareable URL. The thing a boarder puts on their CV or pins to their Twitter bio.

### 7.1 Header

```
[ AVATAR ]   @handle   Name (optional)
             THE 100 · SEAT 04   ·   BOARDED 2026-05-12   ·   FOUNDER
             [ SHARE PROFILE → ]
```

If post-100 boarder: `SEAT 147 · BOARDED [date]` (no THE 100 prefix).
If re-boarder: shows current seat + `PREVIOUSLY THE 100 · SEAT 04 · VACATED [date]` below.

### 7.2 Current Mission

If active:

```
CURRENT MISSION                                              HATCH IN 18d 14h
"Ship Airlock v1.0 + Stripe checkout"                        STATUS: DECLARED
DECLARED 2026-05-13   ·   DEADLINE 2026-06-12 18:00 UTC
[ optional: PROOF link if PROOF PENDING ]
```

If idle (between missions):

```
SEAT HELD · NO ACTIVE MISSION
Last shipped: 2026-04-22 · "Ship Bolt AI feature behind paywall"
[ DECLARE NEW MISSION → ] (only visible to seat owner)
```

If airlocked (memorial profile):

```
SEAT VACATED · 2026-05-04
Died on: "Ship landing page redesign"
[ RE-BOARDED AS SEAT 147 → ] (if applicable)
```

### 7.3 Launch Wall

Every confirmed mission, newest first. Each entry:

```
[ thumbnail ]   "Ship Bolt AI feature behind paywall"
                LAUNCHED 2026-04-22   ·   MISSION 02 OF LIFETIME
                PROOF: bolt.example.com/ai   ·   30 DAYS · $84 PAID

[ thumbnail ]   "Ship Airlock v1.0 + Stripe checkout"
                LAUNCHED 2026-03-15   ·   MISSION 01 OF LIFETIME
                PROOF: theairlock.space   ·   30 DAYS · $42 PAID
```

### 7.4 Airlock Log

Every AIRLOCKED stamp, newest first. Each entry:

```
[ ✕ ]   AIRLOCKED · "Ship marketplace v1 + Stripe"
        2026-05-04   ·   30 DAYS, 0 PROOF
        [ optional: REDEEMED · 2026-07-22 ] tag if re-entry challenge completed
```

If the boarder has no airlock events: `NO AIRLOCK EVENTS · clean log` — small text, muted.

### 7.5 Lifetime Stats

```
MISSIONS DECLARED      4
MISSIONS CONFIRMED     3
MISSIONS AIRLOCKED     1
LONGEST STREAK         3 missions
TIME ON THE BOAT       142 days
TOTAL PAID             $42 + $336 = $378
```

### 7.6 Active Stacks

```
THIS BOARDER'S STACKS

✓ DONATION PENALTY    $50 → effectivealtruism.org
✓ CREW ALERT          3 contacts named
✗ THE WAKE            (not enabled)
```

Visible publicly — having stacks active is a flex. Hiding them is allowed; the section just doesn't render.

### 7.7 Memorial Profile (for vacated seats)

When a seat is vacated and the former occupant has not re-boarded:

- Profile header shows `SEAT 04 · VACATED 2026-05-04`.
- "Current Mission" block replaced with the airlock event.
- Launch Wall and Airlock Log still render (the former occupant's history).
- A muted line at the bottom: *"This profile is a memorial. The seat was vacated and never refilled. Nobody else gets seat 04."*

When the former occupant has re-boarded:

- The original profile still exists at its old `/c/[handle]` URL.
- It shows `RE-BOARDED AS SEAT 147 — visit current profile →` linking to the new seat's profile.
- Or: handles are unique per user, so re-boarding keeps the same `/c/[handle]` URL but the seat number on the page updates and shows the historical record.

**Implementation note:** Handles are unique per user, not per seat. A user has one `/c/[handle]` URL across their lifetime. Within that URL, seat history is displayed in chronological order.

---

## 8. THE WAKE — `/airlock`

Public log of all airlock events. The Wake stack (S · 03) posts here automatically.

**Header:** `THE WAKE — public log of every airlock event since boarding began.`

**Stream (reverse chronological):**

```
2026-05-04 18:21 UTC
@renaud · THE 100 · SEAT 06 · VACATED
Died on: "Ship landing page redesign"
30 days, no proof, hatch opened 06h after deadline.
[ link to profile ]

2026-04-18 12:42 UTC
@kel · SEAT 119 · VACATED
Died on: "Ship AI scheduler v1"
30 days, no proof, hatch opened on time.
[ link to profile ]
```

Filterable by: `THE 100 ONLY` / `POST-100` / `ALL`.

Indexed by Google. Shareable per-entry.

---

## 9. DATA MODEL

DynamoDB single-table, via `@theairlock/database` (existing).

### 9.1 `users`

| field | type | notes |
|-------|------|-------|
| `user_id` | string (uuid) | PK |
| `handle` | string | unique, lowercase, URL-safe |
| `display_name` | string | optional |
| `email` | string | from Clerk when present on session/user metadata |
| `joined_at` | ISO timestamp | account creation, not boarding |
| `avatar_seed` | string | deterministic pixel avatar input |

### 9.2 `seats`

| field | type | notes |
|-------|------|-------|
| `seat_id` | string | PK — sequential, e.g. `SEAT-001` |
| `seat_number` | int | 1..N |
| `cohort` | enum | `THE_100` (1-100) or `POST_100` (101+) |
| `occupant_user_id` | string | nullable if vacated |
| `status` | enum | `OCCUPIED` / `VACATED` |
| `tier_paid` | int | 1..4, or 5 for post-100 flat |
| `price_paid` | int (cents) | actual amount paid |
| `boarded_at` | ISO timestamp | |
| `vacated_at` | ISO timestamp | nullable |
| `vacated_on_mission_id` | string | nullable, FK |

### 9.3 `missions`

| field | type | notes |
|-------|------|-------|
| `mission_id` | string | PK |
| `seat_id` | string | FK |
| `declaration` | string | the one-sentence mission |
| `declared_at` | ISO timestamp | clock starts here |
| `deadline_at` | ISO timestamp | declared_at + 30 days |
| `status` | enum | `DECLARED` / `PROOF_PENDING` / `CONFIRMED` / `AIRLOCKED` |
| `proof_url` | string | nullable |
| `proof_submitted_at` | ISO timestamp | nullable |
| `confirmed_at` | ISO timestamp | nullable |
| `airlocked_at` | ISO timestamp | nullable |
| `redeemed` | boolean | true if part of a redemption pair |

### 9.4 `stacks` (per-seat, optional)

| field | type | notes |
|-------|------|-------|
| `seat_id` | string | PK |
| `donation_enabled` | bool | |
| `donation_amount` | int (cents) | |
| `donation_target` | string | charity name / URL |
| `crew_alert_enabled` | bool | |
| `crew_alert_contacts` | string[] | up to 8 |
| `the_wake_enabled` | bool | |
| `the_wake_crosspost` | string[] | linked social accounts |

### 9.5 `events`

| field | type | notes |
|-------|------|-------|
| `event_id` | string | PK |
| `kind` | enum | `BOARDED` / `DECLARED` / `PROOF_SUBMITTED` / `CONFIRMED` / `AIRLOCKED` / `RE_BOARDED` / `REDEEMED` |
| `user_id` | string | |
| `seat_id` | string | |
| `mission_id` | string | nullable |
| `occurred_at` | ISO timestamp | |
| `payload` | json | event-specific details |

Drives event feed (`[ 06 ] SHIP TELEMETRY`) and The Wake (`/airlock`).

---

## 10. VISUAL SYSTEM

### 10.1 Colors (existing CSS vars in `apps/web/public/styles.css`)

Continue using:
- `--text` — primary
- `--muted` — secondary copy
- `--dim` — tertiary
- `--signal` — green (CONFIRMED, success)
- `--hatch` — orange (AIRLOCKED, warning, hatch events)
- `--ice` — cyan/blue (LIVE, telemetry)
- `--line` — borders
- background gradients + stars (existing)

**Status badge colors:**
- `DECLARED` → muted blue
- `PROOF PENDING` → amber, pulse
- `CONFIRMED` → `--signal` (green)
- `AIRLOCKED` → `--hatch` (orange)
- `VACATED` → struck-through, `--dim`

### 10.2 Typography

Existing: JetBrains Mono for labels/HUD, sans for body, display weight for headings.

### 10.3 Components to build

- **`SeatGrid`** — 10×10 (small for hero, large for `[ 04 ] THE CABIN`). Props: `seats`, interactive, hover/click handlers.
- **`MissionLogTable`** — sortable, filterable, real-time. Props: `rows`, `tab`, `liveUpdates`.
- **`PriceLadder`** — live progress bars per tier, next-tier preview. Props: `tiers`, `currentSeat`.
- **`EventFeed`** — streaming entries with icons and tags.
- **`StatusBadge`** — `DECLARED` / `PROOF PENDING` / `CONFIRMED` / `AIRLOCKED` / `VACATED`.
- **`SeatNumberPill`** — `THE 100 · SEAT 04` styling.
- **`CountdownInline`** — relative time, updates every 60s (or 1s if < 1h).
- **`ProductReceiptCard`** — for the founder block (fr3n.fan / creavings.com / freye.tech).

---

## 11. INTERACTIONS

- **`SeatGrid` hover:** tooltip with handle + current mission + hatch countdown (or memorial info if vacated).
- **`SeatGrid` click:** navigate to `/c/[handle]` (or boarding flow if `OPEN`).
- **Mission log row click:** navigate to `/c/[handle]`.
- **Event feed:** new entries slide in from the top. Old entries scroll out the bottom (max 8-12 visible at a time).
- **Price ladder bars:** smooth-animate fill on seat purchase.
- **Vacated transition:** when a seat airlocks, the grid square animates from `OCCUPIED` to `VACATED` (struck-through, color drains).
- **Countdown ticking:** "nearest hatch in fleet" updates every 1s when < 1h, every 60s otherwise.
- **AIRLOCKED row tint:** persistent, not animated.

**Real-time strategy:**
- Page polls `/api/state` every 30s for active mission counts, ladder progress, nearest hatch.
- Event feed connects via SSE (Server-Sent Events) for push updates. WebSockets are overkill for v1 unless the feed becomes high-frequency.
- Mock data behind a feature flag for the pre-launch period (so the page can be live with realistic-looking data before there are 3 real boarders).

---

## 12. IMPLEMENTATION

### 12.1 Stack (existing, no changes)

- `apps/web` — Astro static site + islands for interactivity (SeatGrid, MissionLogTable, EventFeed, etc.)
- `apps/api` — Go Fiber Lambda, exposes `/api/state`, `/api/missions`, `/api/seats`, `/api/events`, `/api/boarding`
- Auth — Clerk Astro SDK; API verifies Clerk session JWTs against the Clerk issuer JWKS
- `packages/database` — DynamoDB/ElectroDB
- Payments — Stripe checkout for v1 boarding because the pass is a one-time dynamic-price purchase tied to atomic seat assignment. Clerk Billing is subscription-plan oriented today, so switching the actual boarding charge to Clerk Billing requires changing the product model or adding a Clerk-compatible reservation handoff.

### 12.2 Auth flow

- `/manifest` requires Clerk auth → checkout → seat assignment → mission declaration UI.
- `/dashboard` requires JWT, gates mission management.

### 12.3 Payment flow

- Stripe Checkout Session, one-time charge at the current next-seat price.
- Webhook → assign seat (sequential next number) → create user record if needed → start 24h declaration window.

### 12.4 Seat assignment

- Atomic counter (DynamoDB conditional update or a dedicated `seats_counter` item).
- Race condition: two simultaneous purchases at tier boundary. Resolution: the second purchase falls into the next tier; charge a delta or refund partial. Simplest: charge the next-tier price at the moment Stripe session starts, not when webhook lands. Acceptable for v1.

### 12.5 Mission verification

- Manual review for v1. Boarder submits URL. Admin (Oskar) checks within 24h: does the URL load, does it have a working sign-up, does payment work.
- Future: automated checker (load page, look for Stripe `pk_live_*` keys, etc.). Out of scope for v1.

### 12.6 Hatch trigger

- Cron job (every 5 minutes) scans for missions where `deadline_at < now()` and `status IN ('DECLARED', 'PROOF_PENDING')`. Transitions them to `AIRLOCKED`, vacates the seat, fires stack consequences (donation charge, crew alert email, The Wake post).

---

## 13. OUT OF SCOPE FOR v1

These are deliberate omissions. Don't build them unless explicitly added later.

- Squad / sub-crew structure. The 100 is the only group.
- Streak as a load-bearing mechanic. Track it as a stat on profiles, don't gamify.
- Daily / weekly cadence options. 30 days is the only cadence.
- Refunds (except false airlocks caused by our system — see decision L11).
- Mission pausing or extension UX. No UI exists; the system literally has no path to extend.
- Public chat / forum / Discord. The Airlock is a registry, not a community.
- Mobile app. Web-only for v1.
- Onboarding tutorial. The FAQ is the tutorial.
- Email digests, weekly summaries. The page is the digest.
- Automated proof verification (Stripe key detection, sign-up flow scraping). Manual for v1.
- Pricing for sub-tier or split-payment options. One number, one payment.
- Custom mission types beyond the four archetypes. Anything else gets manually approved by the admin during the verification step or rejected as out-of-scope.
- **Donation Penalty stack** (S · 03) — deferred to v1.1. Card-on-file + future auto-charge.
- **OAuth cross-post to socials** for The Wake — deferred to v1.1. Manual share button only in v1.
- **Custom avatar uploads** — deferred. Auto-generated pixel avatars only.
- **Banked days from early ship** — explicitly not a feature. Confirmed mission ends the cycle; remaining days don't carry forward.
- **Late-proof grace period** — explicitly not a feature.

---

## 14. OPEN QUESTIONS

All core design questions resolved on 2026-05-27 (see section 1.5). Remaining items below are operational, not blocking:

- **Manual mission verification at scale.** Oskar handles all verifications in v1. At ~3 verifications/day average across 100 active builders, this is sustainable. Plan for partial automation (Stripe key detection, URL liveness check) when verification load exceeds ~10/day.
- **Stripe customer object retention.** Even with the donation stack deferred, decide whether v1 captures a Stripe customer object at checkout — gives v1.1 a path to launch the donation stack without re-prompting boarders for cards. Recommend: yes, capture it at checkout.
- **Memorial profile discoverability.** Decided (L8): single `/c/[handle]` URL per person, history inside. Implementation note: the profile page rendering logic needs to handle three states — currently boarded (active mission or idle), currently airlocked-no-rebuild (pure memorial), and re-boarded (current seat + history of prior vacated seats). Three render paths, one URL.

---

## 15. SHIPPING THIS SPEC

Spec is locked (see section 1.5). Implementation order:

1. **Port copy into `apps/web/src/pages/index.astro`.** Full rewrite of the body. Keep the cinematic hero scripts (`scene.js`, `sprites.js`, `reveals.js`) untouched. Use the existing CSS variables / classes; add inline styles only for genuinely new visual elements (10×10 grid, mirror narrow column, price ladder bars). Honest data only — Oskar as seat 01 (DECLARED), every other seat open, fleet metrics genuinely small. Do **not** seed mock crew members.
2. **Build `SeatGrid` as an Astro island** (DOM-based; CSS grid of 100 squares). Hover tooltips, click → `/c/[handle]` or boarding flow. Wire to `/api/state`.
3. **Build `MissionLogTable` and `PriceLadder` as Astro islands.** Both pull from `/api/state`.
4. **Stand up `/api/state`** in `apps/api` — returns the live snapshot: active seats, vacated seats, current next-seat price, nearest hatch, recent events. Cached aggressively (5–15s TTL).
5. **Wire Stripe Checkout** → seat assignment (atomic counter) → mission declaration UI. Routes: `/manifest` (post-checkout flow), `/dashboard` (private).
6. **Add the cron hatch trigger.** Runs every 5 min, scans for missions where `deadline_at < now()` and status ∈ `{DECLARED, PROOF_PENDING}`, transitions to `AIRLOCKED`, fires stack consequences.
7. **Build profile page `/c/[handle]`** with three render states (active, memorial, re-boarded). Build The Wake `/airlock` as reverse-chronological event stream.
8. **Switch from any mock-data scaffolding to live data.** No feature flag needed — the page is honest from day one (just Oskar boarded).
9. **Oskar boards as seat 01**, declares mission 01: *"Ship Airlock v1.0 + Stripe checkout."* The Airlock goes public.

**Hand-off note (for the implementation session):** read this entire spec top-to-bottom before touching code. The voice rules (section 2), copy callbacks (Mirror ↔ founder block ↔ FAQ), and visual mechanics (vacated-seat decay on the grid) are interconnected — implementing one section in isolation will produce a worse page than reading the whole spec first.
