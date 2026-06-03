# Clerk Billing config-of-record

`billing.json` is the apply artifact for the per-tier Clerk Billing plans
(`tier-01`..`tier-04`) that gate boarding on The Airlock. It is **not** imported
by `sst.config.ts` — it is reconciled into the live Clerk instance by a human
using the `clerk` CLI.

Plan amounts MUST stay in sync with the Go truth:
- `apps/api/internal/handlers/state_snapshot.go` → `priceForTier`
- `apps/api/internal/handlers/boarding_rules.go` → `planAmountCentsBySlug`

`tier-01`=4200, `tier-02`=8400, `tier-03`=16800, `tier-04`=33600. `tier-04`
($336) doubles as the flat post-100/reboard price. All four attach the same
`boarding` feature so `has({ feature: 'boarding' })` stays tier-agnostic.

## NOT applied by this workflow

This workflow only **wrote the files** in `infra/clerk/`. It did **not** run any
`clerk config patch`/`put`/`enable` or any other Clerk mutation. Applying
`billing.json` mutates the **live Clerk instance** and is a deliberate,
human-run step — follow the sequence below on the host.

## READ-ONLY pull → dry-run → apply

```bash
# 1. READ-ONLY sanity check
clerk --version   # expect 1.5.0; READ-ONLY sanity check

# 2. Capture current state-of-record (existing free_user + boarder plans).
#    If this warns about sandbox/agent-mode or fails auth/link, STOP and treat
#    the live schema as untrusted.
clerk config pull --keys billing > /tmp/clerk_billing_current.json

# 3. Eyeball what the patch will add/merge before touching the API.
diff <(jq -S . /tmp/clerk_billing_current.json) <(jq -S . infra/clerk/billing.json)

# 4. Preview the additive PATCH, no mutation; confirm tier-01..tier-04 are
#    ADDED and boarding/boarder are untouched.
clerk config patch --file infra/clerk/billing.json --dry-run

# 5. APPLY the additive tier plans (no --destructive needed — purely additive).
#    Maps merge by key, so free_user/shipping_updates/boarder are preserved here.
clerk config patch --file infra/clerk/billing.json --yes

# 6. Remove the legacy single-price `boarder` plan (superseded by tier-01..04 and
#    fail-closed to $0 in the Go enforcement). Deletion needs --destructive; the
#    null value names the only key to delete. DRY-RUN FIRST and confirm that only
#    `boarder` disappears (free_user + the four tiers must stay), then apply.
clerk config patch --json '{"billing":{"plans":{"boarder":null}}}' --destructive --dry-run
clerk config patch --json '{"billing":{"plans":{"boarder":null}}}' --destructive --yes

# 7. Verify tier-01..tier-04 exist and `boarder` is gone.
clerk config pull --keys billing | jq '.billing.plans | keys'

# To find the cplan_... IDs the web CheckoutButton needs (planId is a Clerk
# plan ID, NOT the slug):
clerk config pull --keys billing | jq -r '.billing.plans | to_entries[] | select(.key|startswith("tier-")) | "\(.key) -> \(.value.id // "NO_ID_FIELD: resolve via clerk api / dashboard")"'

# PROD: re-run the pull/dry-run/patch with --instance <prod> ONLY after Billing
# + a connected payment gateway are enabled on prod.
```

> A `patch` merges maps by key — omitted plans/features are preserved (this is
> why the additive tier patch in step 5 leaves `free_user`/`shipping_updates`
> untouched). To DELETE a plan, set its key to `null` AND pass `--destructive`
> (step 6); a destructive patch only affects the keys it names, so the other
> plans are safe. A per-plan `features` array REPLACES that plan's attachments.
