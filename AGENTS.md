# AGENTS.md

SST mono-repo starter. Bun workspaces.

## Layout

- `apps/api` — Go Fiber Lambda (api-v2 pattern), JWT auth guard against OpenAuth JWKS
- `apps/auth` — OpenAuth issuer Lambda
- `apps/web` — Astro
- `packages/core` — shared TS utils
- `packages/database` — DynamoDB / ElectroDB client
- `infra/` — modular SST stacks, imported by `sst.config.ts`

## Commands

- `bun install` — install workspace deps
- `bun run dev` — `sst dev`
- `bun run deploy` — `sst deploy --stage <stage>`
- `bun run --filter @theairlock/web dev` — run a single workspace script
