# Feature: Clerk login + paid boarding → dashboard onboarding with astronaut-suit AI avatars

Decisions (locked):
- Image generation = **AWS Bedrock** (IAM-based, no new SST secret). Model: Amazon **Nova Canvas** (`amazon.nova-canvas-v1:0`), Titan Image Generator v2 as documented fallback.
- Access = **paid boarders only**. Every avatar/profile/goal write requires an OCCUPIED seat.
- Default (no upload) = a single shared **visor-down astronaut SVG** shipped statically (`/astronaut-default.svg`). Uploads → Bedrock generates the user's face inside an astronaut suit/helmet (clear visor).
- Region: Lambda runs in `eu-central-1`. Bedrock image model may not be in eu-central-1, so the Bedrock client uses a dedicated `BEDROCK_REGION` (default `us-east-1`). Cross-region invoke is fine.

This repo: SST monorepo. Go Fiber Lambda (`apps/api`), Astro web (`apps/web`), ElectroDB single-table (`packages/database`), SST infra (`infra/`).

------------------------------------------------------------------------
## SHARED DATA MODEL (the contract every layer agrees on)

UserEntity gains four attributes. Dynamo field names (snake_case) are authoritative
because the Go layer reads/writes raw Dynamo items with these exact names:

| concept        | TS attr (ElectroDB) | Dynamo field        | type / values                         |
|----------------|---------------------|---------------------|---------------------------------------|
| short bio      | `bio`               | `bio`               | string, optional, <= 280 chars        |
| avatar image   | `avatarUrl`         | `avatar_url`        | string, optional (full CDN URL)       |
| avatar state   | `avatarStatus`      | `avatar_status`     | "NONE"\|"PENDING"\|"READY"\|"FAILED"  |
| avatar updated | `avatarUpdatedAt`   | `avatar_updated_at` | string ISO, optional                   |

`avatarSeed` (existing) stays — used as the fallback tint seed for the default SVG.
Empty/absent `avatar_status` is treated as `"NONE"` by all readers.

avatar_url value = `${PUBLIC_ASSETS_BASE_URL}/avatars/<userId>.png`
(PUBLIC_ASSETS_BASE_URL = `${router.url}/cdn`; public bucket key = `avatars/<userId>.png`).

------------------------------------------------------------------------
## HTTP CONTRACT (Go API, all under `/api` and `/v2`, all auth-required)

- `PUT  /me/profile`  body `{ "display_name": string, "bio": string }`
      -> 200 `{ "ok": true, "profile": {handle, display_name, bio} }`
      validation: display_name 1..60 chars; bio 0..280 chars. 403 if no occupied seat.
- `POST /me/avatar`   multipart/form-data, file field name **`avatar`** (png/jpeg/webp, <= 5 MB)
      -> 200 `{ "avatar_url": string, "status": "READY" }`
      403 if no occupied seat; 400 invalid file; 502 `{ "error": "avatar generation failed" }` on Bedrock error.
      Side effects: store source to private bucket `avatar-src/<userId>`, generated png to public bucket
      `avatars/<userId>.png`, set user avatar_url + avatar_status=READY + avatar_updated_at.
- `DELETE /me/avatar` -> 200 `{ "status": "NONE" }`  (REMOVE avatar_url, set avatar_status=NONE). 403 if no seat.
- `GET  /me/avatar`   -> 200 `{ "avatar_url"?: string, "status": string }` (optional convenience).

Existing `GET /me/airlock` (boarder state) `profile` object MUST also now include
`avatar_url` (omitempty), `avatar_status`, and `bio` (omitempty).

Public `GET /api/state` occupant objects MUST include `avatar_url` (omitempty) when the
occupant's avatar_status == READY.

------------------------------------------------------------------------
## FILE OWNERSHIP (no two agents edit the same file)

### A — Database (TS)  [packages/database/src/index.ts]
Add the 4 attributes above to `UserEntity.attributes` (keep existing `avatarSeed`).
`avatarStatus` default `() => "NONE"`. Do not touch indexes.

### B — Go data model  [state_snapshot.go, state_store.go, boarding_store.go, boarder_state.go]
- `airlockUser` struct (state_snapshot.go ~line 25): add `Bio string`, `AvatarURL string`, `AvatarStatus string`.
- state_store.go: the user-scan struct that has dynamodbav tags `handle`/`display_name`/`avatar_seed`
  (~line 170-180): add `Bio string \`dynamodbav:"bio"\``, `AvatarURL string \`dynamodbav:"avatar_url"\``,
  `AvatarStatus string \`dynamodbav:"avatar_status"\``, and map them into `airlockUser` wherever users are
  built from records (mirror how Handle/DisplayName/AvatarSeed are copied).
- state_snapshot.go `stateOccupant` struct (~line 846): add `AvatarURL string \`json:"avatar_url,omitempty"\``.
  Where occupant is built from `usersByID[...OccupantUserID]` (~lines 235, 262, 555): set
  `AvatarURL: <user>.AvatarURL` ONLY when `<user>.AvatarStatus == "READY"` (else empty).
- boarder_state.go `boarderStateResponse` profile map (~line 142): add `"avatar_url"` (only if non-empty),
  `"avatar_status"` (default "NONE" if empty), and `"bio"` (only if non-empty) from `state.AirlockUser`.
- boarding_store.go `userItem` (~line 205): DO NOT add avatar fields. Boarding creates the user with
  avatar unset (status reads as NONE). (Re-boarding intentionally resets to visor-down until re-upload —
  add a one-line comment noting this is deliberate.)
- Do NOT add new deps; do NOT edit app.go/config.go (owned by D).

### C — Infra (TS)  [infra/api.ts ONLY]
In `export const api = new sst.aws.Function("Api", {...})`:
- Add a `bedrock:InvokeModel` permission to `link`:
  ```ts
  const bedrockRegion = process.env.BEDROCK_REGION ?? "us-east-1";
  // ...inside link: [...]
  sst.aws.permission({
    actions: ["bedrock:InvokeModel"],
    resources: [
      `arn:aws:bedrock:${bedrockRegion}::foundation-model/amazon.nova-canvas-v1:0`,
      `arn:aws:bedrock:${bedrockRegion}::foundation-model/amazon.titan-image-generator-v2:0`,
    ],
  }),
  ```
- Add to `environment`:
  ```ts
  BEDROCK_REGION: process.env.BEDROCK_REGION ?? "us-east-1",
  BEDROCK_IMAGE_MODEL: process.env.BEDROCK_IMAGE_MODEL ?? "amazon.nova-canvas-v1:0",
  PUBLIC_ASSETS_BASE_URL: $interpolate`${router.url}/cdn`,
  ```
  (PRIVATE_ASSETS_BUCKET / PUBLIC_ASSETS_BUCKET env already exist — keep them.)
- Add a short comment: account must enable Bedrock model access for Nova Canvas in BEDROCK_REGION.

### D — Go avatar feature  [NEW: avatar.go, avatar_store.go, avatar_imagegen.go ; EDIT: config.go, app.go ; go.mod/go.sum]
Mirror existing handler style (interface-injected store + clock, like BoardingHandler/MissionHandler).
- config.go: add fields + load: `PrivateAssetsBucket` (env PRIVATE_ASSETS_BUCKET),
  `PublicAssetsBaseURL` (env PUBLIC_ASSETS_BASE_URL), `BedrockRegion` (env BEDROCK_REGION, default "us-east-1"),
  `BedrockImageModel` (env BEDROCK_IMAGE_MODEL, default "amazon.nova-canvas-v1:0"). (PublicAssetsBucket exists.)
- avatar_imagegen.go:
  - `type imageGenerator interface { GenerateFromPhoto(ctx context.Context, src []byte, contentType string) ([]byte, error) }`
  - `bedrockImageGenerator` (fields: region, modelID; lazily builds `*bedrockruntime.Client` via
    awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region)) — same pattern as dynamoClient).
    Use the VERIFIED Nova Canvas OUTPAINTING request shape provided in the workflow prompt
    (taskType OUTPAINTING, outPaintingParams.image=base64 src, maskPrompt:"face", text=astronaut prompt,
    outPaintingMode:"DEFAULT"; imageGenerationConfig 1024x1024, quality "premium", cfgScale 8,
    numberOfImages 1). Decode response.images[0] (base64) -> png bytes. Return clear error on response.error.
  - Astronaut prompt: white NASA-style spacesuit + helmet with a clear glass visor, face clearly visible
    through the visor, dark starfield background, photorealistic centered portrait. negativeText covers
    blurry/distorted face/multiple people/text/watermark.
- avatar_store.go: add METHODS on `*dynamoMissionStore` (it already has tableName/region + dynamoClient):
  - lazy `s3Client(ctx)` (service/s3, region = store.region i.e. AWS_REGION) on a new field.
  - `PutAvatarSource(ctx, bucket, userID, data, contentType) error` (key `avatar-src/<userID>`).
  - `PutAvatarPublic(ctx, bucket, userID, png []byte) error` (key `avatars/<userID>.png`,
    ContentType image/png, CacheControl "public, max-age=300").
  - `SetUserAvatar(ctx, userID, url, status string, now time.Time) error` — Dynamo UpdateItem on the user
    item (pk = electroPK("userid", userID), sk = "$user_1") SET avatar_url, avatar_status, avatar_updated_at,
    condition attribute_exists(pk).
  - `ClearUserAvatar(ctx, userID) error` — REMOVE avatar_url, SET avatar_status="NONE", avatar_updated_at.
  - `UpdateUserProfile(ctx, userID, displayName, bio string, now time.Time) error` — UpdateItem SET
    display_name + bio, condition attribute_exists(pk).
  (Reuse electroPK/stringAttr/numberAttr helpers already in the handlers package.)
- avatar.go:
  - `AvatarHandlerConfig { TableName, AWSRegion, PublicBucket, PrivateBucket, PublicAssetsBaseURL,
    BedrockRegion, BedrockImageModel string; LaunchUserID string; Clock func() time.Time;
    Store avatarStore; Generator imageGenerator }` where `avatarStore` is an interface covering
    LoadBoarderState + the avatar/profile store methods above (so tests inject a fake).
  - `NewAvatarHandler(cfg)` defaults: store = newDynamoMissionStore(...) when TableName set; generator =
    bedrockImageGenerator when BedrockRegion+PublicBucket set (else nil -> 503 on upload).
  - Seat gate helper: `requireSeat(c, userID)` uses store.LoadBoarderState(now); 403 if !HasSeat.
  - `Upload`, `Reset`, `UpdateProfile`, `GetStatus` per HTTP contract. Build avatar_url as
    `PublicAssetsBaseURL + "/avatars/" + userID + ".png"`.
  - File validation: content-type in {image/png,image/jpeg,image/webp}; size <= 5*1024*1024.
- app.go:
  - Bump `BodyLimit` from 256*1024 to `6 * 1024 * 1024` (authenticated + rate-limited; needed for uploads;
    keep the explaining comment, note 6MB is the Function-URL request ceiling).
  - Construct `avatarHandler := handlers.NewAvatarHandler(...)` in `New`.
  - Add `avatarHandler` param to `registerProtectedRoutes` and register inside it:
    `group.Put("/me/profile", auth, avatarHandler.UpdateProfile)`,
    `group.Post("/me/avatar", auth, avatarHandler.Upload)`,
    `group.Delete("/me/avatar", auth, avatarHandler.Reset)`,
    `group.Get("/me/avatar", auth, avatarHandler.GetStatus)`.
    Update BOTH call sites (fail-closed stub path + normal path).
- go.mod: `go get github.com/aws/aws-sdk-go-v2/service/s3 github.com/aws/aws-sdk-go-v2/service/bedrockruntime`
  then `go mod tidy`. Build with `go build ./...`.
- Add `avatar_test.go` covering: no-seat -> 403, bad content-type -> 400, oversize -> 400,
  happy path with fake store + fake generator -> 200 + READY + correct avatar_url, generator error -> 502,
  Reset -> NONE, UpdateProfile validation. Use the existing injectable-store/clock test style.

### E — Frontend (Astro)  [dashboard.astro, c/[handle].astro, lib/api-proxy.ts, NEW pages/api/me/avatar.ts,
###     NEW pages/api/me/profile.ts, NEW public/astronaut-default.svg, NEW public/avatar.js, styles.css append]
- lib/api-proxy.ts: ADD `forwardAuthenticatedRaw(ctx, upstreamPath)` — like forwardAuthenticatedJSON but
  forwards the raw request body via `await ctx.request.arrayBuffer()` and preserves the original
  `content-type` header (for multipart). Keep the existing JSON forwarder.
- pages/api/me/avatar.ts: `export const POST = (ctx) => forwardAuthenticatedRaw(ctx, "/api/me/avatar")`;
  `export const DELETE = (ctx) => forwardAuthenticatedJSON(ctx, "/api/me/avatar")`;
  `export const GET = (ctx) => forwardAuthenticatedJSON(ctx, "/api/me/avatar")`. `export const prerender = false`.
- pages/api/me/profile.ts: `export const PUT = (ctx) => forwardAuthenticatedJSON(ctx, "/api/me/profile")`.
  `export const prerender = false`.
- dashboard.astro: extend `DashboardState.profile` type with `avatar_url?`, `avatar_status?`, `bio?`.
  Add two panels in the profile-grid:
    1) IDENTITY / SUIT panel: an `<img id="avatar-img" class="suit-avatar" src={avatarUrl || "/astronaut-default.svg"}>`,
       a file input (accept image/png,image/jpeg,image/webp) + "GENERATE SUIT" button, a "DROP VISOR (use default)"
       button, a status line. Explain: everyone boards in a suit; upload a face to ride visor-up, or stay
       visor-down. Only show when `dashboard.has_seat` (else "BOARD FIRST").
    2) PROFILE form (display_name + bio textarea<=280) posting PUT /api/me/profile.
  Relabel the existing mission declare card to emphasise "30-DAY GOAL".
  Load `/astronaut-default.svg` fallback when avatar_url empty.
- public/avatar.js (script-src 'self'): wires the file input -> `fetch('/api/me/avatar', {method:'POST', body: FormData})`,
  shows a PENDING/loading state, on success swaps #avatar-img src (cache-bust with ?t=) and the status line,
  on failure shows the error; wires the "drop visor" button -> DELETE then sets src to /astronaut-default.svg.
  Wire profile form via existing [data-airlock-form] (read public/dashboard.js to confirm it handles PUT/JSON;
  if it only does POST, give the profile form its own small handler in avatar.js).
- c/[handle].astro: the StateSeat occupant type -> add `avatar_url?: string`. Replace the
  `<canvas class="pixel-avatar profile-avatar" data-avatar=...>` with
  `<img class="suit-avatar profile-avatar" src={current?.occupant?.avatar_url || "/astronaut-default.svg"}
   alt={`${profileHandle} in their astronaut suit`} width="96" height="96" />`.
- public/astronaut-default.svg: a clean unified astronaut bust, visor DOWN (opaque reflective gold/tinted
  shield, no face visible), helmet + suit shoulders + small "AIRLOCK" patch, dark background. Square viewBox.
  (A high-fidelity version may be supplied/overwritten by the orchestrator — produce a solid one regardless.)
- styles.css: append a small `.suit-avatar { width:64px;height:64px;border-radius:10px;object-fit:cover;
  background:#0b0e14;border:1px solid var(--border, #222) }` and `.profile-avatar.suit-avatar{width:96px;height:96px}`
  plus minimal layout for the new panels (reuse existing .panel/.profile-card classes).
- Note: CSP already allows `img-src https:` and inline form posts to same-origin, so no CSP change needed.

------------------------------------------------------------------------
## VERIFY (must pass)
- Go: `cd apps/api && go build ./... && go vet ./... && go test ./...`
- Web: `cd apps/web && bun run typecheck` (astro check) and `bun run build` if feasible.
- No secrets committed. Bedrock via IAM only.
