// Apply pending Drizzle migrations against the DSQL cluster.
//
// Run with the cluster reachable and AWS creds in the environment, e.g.:
//   DSQL_ENDPOINT=<id>.dsql.<region>.on.aws DSQL_REGION=<region> \
//     bun run --filter @starter/database db:migrate
//
// Inside SST you can instead `sst shell -- bun run --filter @starter/database
// db:migrate`, which injects the linked `Sql` resource so the endpoint/region
// are resolved automatically.
import { migrate } from "drizzle-orm/postgres-js/migrator";

import { getDb } from "./client";

await migrate(getDb(), { migrationsFolder: "./migrations" });

// postgres.js keeps the event loop alive; exit once migrations land.
process.exit(0);
