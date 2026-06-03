// eslint-disable-next-line @typescript-eslint/triple-slash-reference
/// <reference path="./.sst/platform/config.d.ts" />

// Stages we never want to accidentally tear down. Extend this list if you add
// long-lived stages (e.g. "preview", "demo"). Keep in sync with the matching
// const in infra/env.ts and infra/database.ts.
const protectedStages = ["production", "stage"];

export default $config({
  app(input) {
    const stage = input?.stage ?? "";
    const isProtected = protectedStages.includes(stage);
    return {
      name: "airlock",
      // M-Inf3: retain resources on every protected stage, not just production.
      removal: isProtected ? "retain" : "remove",
      // M-Inf3: SST `protect` blocks `sst remove` against the listed stages.
      protect: isProtected,
      home: "aws",
      providers: {
        aws: {
          region: "eu-central-1",
        },
      },
    };
  },
  async run() {
    const storage = await import("./infra/storage");
    const db = await import("./infra/database");
    const api = await import("./infra/api");
    await import("./infra/hatch");
    const frontend = await import("./infra/frontend");

    return {
      AppStage: $app.stage,
      Region: $app.providers?.aws.region,
      Table: db.table.name,
      PublicAssetsBucket: storage.publicAssetsBucket.name,
      ApiUrl: api.api.url,
      WebUrl: frontend.web.url,
    };
  },
  console: {
    autodeploy: {
      target(event) {
        // H-I4: only auto-deploy on real human pushes.
        // - Skip non-push events (branch deletions, tag events, etc.).
        // - Skip bot accounts. GitHub bots have usernames ending in `[bot]`
        //   (e.g. dependabot[bot], renovate[bot]). SST's GitSender doesn't
        //   expose a `type` field today, so we use the username convention.
        if (event.type !== "branch") return;
        if (event.action !== "pushed") return;
        if (event.sender.username.endsWith("[bot]")) return;

        if (event.branch === "production") return { stage: "production" };
        if (event.branch === "stage") return { stage: "stage" };
        if (event.branch === "dev") return { stage: "dev" };
      },
      async workflow({ $, event }) {
        await $`bun install`;
        if (event.action === "removed") {
          // H-I4: never let an autodeploy `sst remove` a protected stage.
          // Branch deletion of `production`/`stage` must be a deliberate,
          // human-driven operation — bail out so the workflow no-ops.
          const stageGuess =
            event.type === "branch" ? event.branch : undefined;
          if (stageGuess && protectedStages.includes(stageGuess)) {
            console.warn(
              `[autodeploy] refusing to sst-remove protected stage "${stageGuess}". ` +
                `Run \`sst remove --stage ${stageGuess}\` manually if you really mean it.`,
            );
            return;
          }
          await $`bun sst remove`;
        } else {
          await $`bun sst deploy`;
        }
      },
    },
  },
});
