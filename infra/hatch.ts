import { table } from "./database";
import { CLERK_ISSUER } from "./env";

export const hatchCron = new sst.aws.CronV2("HatchCron", {
  schedule: "rate(5 minutes)",
  retries: 1,
  function: {
    runtime: "go",
    handler: "apps/api/cmd/hatch",
    link: [table],
    environment: {
      STAGE: $app.stage,
      AUTH_URL: CLERK_ISSUER,
      ELECTRO_TABLE_NAME: table.name,
      LAUNCH_USER_ID: process.env.LAUNCH_USER_ID ?? "",
    },
    timeout: "30 seconds",
    memory: "512 MB",
  },
});
