# @theairlock/api

Go / Fiber Lambda API. Mirrors the `api-v2` pattern from `fr3n-mono`.

## Layout

```
apps/api/
├── main.go                     # Lambda Function URL entry (streaming SSE + Fiber)
├── go.mod
├── Makefile
├── package.json                # bun workspace stub
└── internal/
    ├── app/app.go              # Fiber app + routes
    ├── config/config.go        # env → Config
    ├── handlers/               # route handlers
    └── middleware/             # auth_guard, etc.
```

## Local

```sh
make run            # run Fiber directly (not Lambda)
make test
make fmt
```

## Deploy

Built as part of `sst deploy` — SST calls `go build -o bootstrap` for you.
The router in `infra/api.ts` mounts this function on `/api`.

Set `CLERK_ISSUER_URL` to the Clerk issuer / frontend API URL and
`LAUNCH_USER_ID` to Oskar's Clerk user ID before production deploys so
the first Dynamo-backed read persists the launch seed as real records.
