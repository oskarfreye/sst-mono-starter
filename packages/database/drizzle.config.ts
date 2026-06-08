import { defineConfig } from "drizzle-kit";

// Only used by `drizzle-kit generate`, which diffs the schema and emits SQL
// into ./migrations — it does NOT need a live connection. Applying those
// migrations is done by `db:migrate` (see src/sql/migrate.ts), which connects
// with an IAM auth token (drizzle-kit can't sign DSQL tokens itself).
export default defineConfig({
  dialect: "postgresql",
  schema: "./src/sql/schema.ts",
  out: "./migrations",
});
