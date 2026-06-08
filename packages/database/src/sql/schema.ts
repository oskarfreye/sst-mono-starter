// Drizzle schema for the Aurora DSQL (Postgres-compatible) database.
//
// This mirrors the ElectroDB `UserEntity` in `../dynamo/index.ts` so you can
// pick either backend without rewriting your domain model. Keep the two in
// sync if you add fields.
//
// DSQL notes:
//  - No sequences / `SERIAL` / `IDENTITY` columns — generate ids in app code
//    (e.g. a UUID/ULID) and use a `text` primary key.
//  - A table's primary key must be declared at creation time; it cannot be
//    added later with `ALTER TABLE`.
//  - `now()` / `current_timestamp` defaults are supported.
import { pgTable, text, timestamp } from "drizzle-orm/pg-core";

export const users = pgTable("users", {
  userId: text("user_id").primaryKey(),
  email: text("email"),
  createdAt: timestamp("created_at", { withTimezone: true })
    .defaultNow()
    .notNull(),
});

export type User = typeof users.$inferSelect;
export type NewUser = typeof users.$inferInsert;
