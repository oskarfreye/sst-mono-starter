import { DsqlSigner } from "@aws-sdk/dsql-signer";
import { drizzle, type PostgresJsDatabase } from "drizzle-orm/postgres-js";
import postgres from "postgres";
import { Resource } from "sst";

import * as schema from "./schema";

/**
 * Resolve the DSQL connection target.
 *
 * `Resource.Sql` comes from the `sst.aws.Dsql` component in
 * `infra/database.ts` (available on any function linked to `dsql`). We fall
 * back to env vars so the package also works outside of SST Link (local
 * scripts, Drizzle Kit migrations, etc.) — mirror of how the Dynamo client
 * reads `ELECTRO_TABLE_NAME`.
 */
function resolveConnection(): { endpoint: string; region: string } {
  const linked = (
    Resource as { Sql?: { endpoint: string; region: string } }
  ).Sql;

  const endpoint = linked?.endpoint ?? process.env.DSQL_ENDPOINT;
  const region =
    linked?.region ?? process.env.DSQL_REGION ?? process.env.AWS_REGION;

  if (!endpoint) {
    throw new Error(
      "DSQL endpoint not set (SST Link `Sql` or DSQL_ENDPOINT env var)",
    );
  }
  if (!region) {
    throw new Error(
      "DSQL region not set (SST Link `Sql`, DSQL_REGION, or AWS_REGION)",
    );
  }
  return { endpoint, region };
}

let _db: PostgresJsDatabase<typeof schema> | undefined;

/**
 * Lazily build a Drizzle client backed by Aurora DSQL.
 *
 * Auth is IAM-based: DSQL has no static password — instead we sign a short
 * lived (≈15 min) connection token with the caller's IAM identity. Because
 * `postgres.js` accepts an async `password` thunk, the token is regenerated
 * for every new physical connection, so a warm Lambda never serves a stale
 * one.
 *
 * The Lambda's execution role must allow `dsql:DbConnectAdmin` on the
 * cluster — `link: [dsql]` in `infra/api.ts` / `infra/auth.ts` grants it.
 */
export function getDb(): PostgresJsDatabase<typeof schema> {
  if (_db) return _db;

  const { endpoint, region } = resolveConnection();

  const sql = postgres({
    host: endpoint,
    port: 5432,
    database: "postgres",
    username: "admin",
    // DSQL only accepts TLS and presents a publicly-trusted certificate, so
    // full verification works with the default Node trust store — no custom
    // CA bundle required. Do not downgrade to `rejectUnauthorized: false`.
    ssl: { rejectUnauthorized: true },
    // Lambda: one connection per warm container is plenty and keeps us well
    // under DSQL's per-cluster connection ceiling. Tune up for long-lived
    // servers.
    max: 1,
    idle_timeout: 20,
    connect_timeout: 10,
    password: async () => {
      const signer = new DsqlSigner({ hostname: endpoint, region });
      // `admin` is a superuser-equivalent role → admin token. For a
      // least-privilege custom role, switch to `getDbConnectAuthToken()`.
      return signer.getDbConnectAdminAuthToken();
    },
  });

  _db = drizzle(sql, { schema });
  return _db;
}

/**
 * Convenience handle so callers can `import { db } from "@starter/database/sql"`
 * and use it directly. It's a lazy proxy around {@link getDb} — the
 * underlying connection is only opened on first query, not on import.
 */
export const db = new Proxy({} as PostgresJsDatabase<typeof schema>, {
  get(_target, prop, receiver) {
    const real = getDb();
    const value = Reflect.get(real, prop, receiver);
    return typeof value === "function" ? value.bind(real) : value;
  },
});
