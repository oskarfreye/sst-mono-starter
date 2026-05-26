# airlock

Opinionated SST v3 monorepo built on Bun workspaces.

Based on the layout of `fr3n-mono`, with:
- Go/Fiber Lambda API (the `api-v2` pattern), JWT verification via OpenAuth JWKS
- OpenAuth issuer Lambda (`sst.aws.Auth`)
- Astro frontend
- Modular `infra/` stacks composed in `sst.config.ts`
- Shared TS packages under `packages/`

## Quickstart

```sh
bun install
bun run dev            # sst dev (local)
bun run deploy --stage <stage>
```

## Structure

```
.
├── apps/
│   ├── api/            # Go Fiber Lambda
│   ├── auth/           # OpenAuth issuer
│   └── web/            # Astro
├── packages/
│   ├── core/           # shared TS
│   └── database/       # ElectroDB models
├── infra/
│   ├── storage.ts
│   ├── database.ts
│   ├── secrets.ts
│   ├── env.ts
│   ├── auth.ts
│   ├── api.ts
│   └── frontend.ts
├── sst.config.ts
├── package.json        # bun workspaces
└── tsconfig.json
```

## Workspace scripts

Run a script inside a single workspace:

```sh
bun run --filter @theairlock/web dev
bun run --filter @theairlock/api build
```

## Prerequisites

- Bun ≥ 1.1
- Go ≥ 1.24 (for `apps/api`)
- AWS credentials for `sst deploy`
