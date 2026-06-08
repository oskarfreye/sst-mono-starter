import { DynamoDBClient } from "@aws-sdk/client-dynamodb";
import { DynamoDBDocumentClient } from "@aws-sdk/lib-dynamodb";
import { Entity, Service } from "electrodb";
import { Resource } from "sst";

export const dynamo = new DynamoDBClient({});

/**
 * DocumentClient with explicit, safe marshalling defaults.
 *
 * - `removeUndefinedValues: true` — drop `undefined` fields instead of throwing
 *   so partial updates don't blow up.
 * - `convertClassInstanceToMap: false` — never silently marshal class instances
 *   (e.g. `Date`, custom classes). Forces callers to convert explicitly.
 * - `convertEmptyValues: false` — keep empty strings / Buffers as-is.
 *   WARNING: do NOT flip this to `true`. Setting it `true` rewrites empty
 *   strings to `NULL`, which breaks ownership / existence checks like
 *   `attribute_exists(field)` and `attribute_not_exists(field)` and can
 *   silently overwrite records you thought you owned.
 * - `wrapNumbers: false` — return JS numbers directly. Acceptable here because
 *   our schemas don't store integers beyond `Number.MAX_SAFE_INTEGER`.
 */
export const documentClient = DynamoDBDocumentClient.from(dynamo, {
  marshallOptions: {
    removeUndefinedValues: true,
    convertClassInstanceToMap: false,
    convertEmptyValues: false,
  },
  unmarshallOptions: {
    wrapNumbers: false,
  },
});

// `Resource.electro.name` comes from the Dynamo resource defined in
// `infra/database.ts` — it's available on any function linked to `table`.
const tableName =
  (Resource as { electro?: { name: string } }).electro?.name ??
  process.env.ELECTRO_TABLE_NAME;
if (!tableName) {
  throw new Error(
    "electro table name not set (SST Link or ELECTRO_TABLE_NAME)",
  );
}

export const UserEntity = new Entity(
  {
    model: {
      entity: "user",
      version: "1",
      service: "starter",
    },
    attributes: {
      userId: { type: "string", required: true },
      email: { type: "string" },
      createdAt: { type: "string", default: () => new Date().toISOString() },
    },
    indexes: {
      primary: {
        pk: { field: "pk", composite: ["userId"] },
        sk: { field: "sk", composite: [] },
      },
    },
  },
  { client: dynamo, table: tableName },
);

export const starterService = new Service(
  { user: UserEntity },
  { client: dynamo, table: tableName },
);

export type StarterService = typeof starterService;
