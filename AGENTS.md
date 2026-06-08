# AGENTS.md

SST mono-repo starter. Bun workspaces.

## Layout

- `apps/api` — Go Fiber Lambda (api-v2 pattern), JWT auth guard against OpenAuth JWKS
- `apps/auth` — OpenAuth issuer Lambda
- `apps/web` — Nuxt 3
- `packages/core` — shared TS utils
- `packages/database` — pick your backend, both provisioned in `infra/database.ts`:
  - `@starter/database/dynamo` — DynamoDB single-table via ElectroDB
  - `@starter/database/sql` — Aurora DSQL (Postgres) via Drizzle ORM, IAM-authed
- `infra/` — modular SST stacks, imported by `sst.config.ts`

## Commands

- `bun install` — install workspace deps
- `bun run dev` — `sst dev`
- `bun run deploy` — `sst deploy --stage <stage>`
- `bun run --filter @starter/web dev` — run a single workspace script
