// SQL backend: Drizzle ORM over Aurora DSQL (Postgres-compatible).
//
// Usage:
//   import { db, users } from "@starter/database/sql";
//   const rows = await db.select().from(users);
export { db, getDb } from "./client";
export * from "./schema";
