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
      service: "airlock",
    },
    attributes: {
      entityType: { type: "string", field: "entity_type", default: () => "user" },
      userId: { type: "string", field: "user_id", required: true },
      handle: { type: "string", required: true },
      displayName: { type: "string", field: "display_name" },
      email: { type: "string" },
      joinedAt: { type: "string", field: "joined_at", default: () => new Date().toISOString() },
      avatarSeed: { type: "string", field: "avatar_seed", required: true },
      bio: { type: "string", field: "bio" },
      avatarUrl: { type: "string", field: "avatar_url" },
      avatarStatus: {
        type: ["NONE", "PENDING", "READY", "FAILED"] as const,
        field: "avatar_status",
        default: () => "NONE",
      },
      avatarUpdatedAt: { type: "string", field: "avatar_updated_at" },
    },
    indexes: {
      primary: {
        pk: { field: "pk", composite: ["userId"] },
        sk: { field: "sk", composite: [] },
      },
      byHandle: {
        index: "GSI1",
        pk: { field: "gsi1pk", composite: ["handle"] },
        sk: { field: "gsi1sk", composite: [] },
      },
    },
  },
  { client: dynamo, table: tableName },
);

export const SeatEntity = new Entity(
  {
    model: {
      entity: "seat",
      version: "1",
      service: "airlock",
    },
    attributes: {
      entityType: { type: "string", field: "entity_type", default: () => "seat" },
      seatId: { type: "string", field: "seat_id", required: true },
      seatNumber: { type: "number", field: "seat_number", required: true },
      cohort: { type: ["THE_100", "POST_100"] as const, required: true },
      occupantUserId: { type: "string", field: "occupant_user_id" },
      status: { type: ["OCCUPIED", "VACATED"] as const, required: true },
      tierPaid: { type: "number", field: "tier_paid", required: true },
      pricePaid: { type: "number", field: "price_paid", required: true },
      boardedAt: { type: "string", field: "boarded_at", required: true },
      vacatedAt: { type: "string", field: "vacated_at" },
      vacatedOnMissionId: { type: "string", field: "vacated_on_mission_id" },
      reboardedAsSeatId: { type: "string", field: "reboarded_as_seat_id" },
      reboardedAsSeatLabel: { type: "string", field: "reboarded_as_seat_label" },
    },
    indexes: {
      primary: {
        pk: { field: "pk", composite: ["seatId"] },
        sk: { field: "sk", composite: [] },
      },
      byOccupant: {
        index: "GSI1",
        pk: { field: "gsi1pk", composite: ["occupantUserId"] },
        sk: { field: "gsi1sk", composite: ["boardedAt"] },
      },
      byStatus: {
        index: "GSI2",
        pk: { field: "gsi2pk", composite: ["status"] },
        sk: { field: "gsi2sk", composite: ["boardedAt"] },
      },
    },
  },
  { client: dynamo, table: tableName },
);

export const MissionEntity = new Entity(
  {
    model: {
      entity: "mission",
      version: "1",
      service: "airlock",
    },
    attributes: {
      entityType: { type: "string", field: "entity_type", default: () => "mission" },
      missionId: { type: "string", field: "mission_id", required: true },
      seatId: { type: "string", field: "seat_id", required: true },
      declaration: { type: "string", required: true },
      declarationUrl: { type: "string", field: "declaration_url", required: true },
      declaredAt: { type: "string", field: "declared_at", required: true },
      deadlineAt: { type: "string", field: "deadline_at", required: true },
      status: {
        type: ["DECLARED", "PROOF_PENDING", "CONFIRMED", "AIRLOCKED"] as const,
        required: true,
      },
      proofUrl: { type: "string", field: "proof_url" },
      proofSubmittedAt: { type: "string", field: "proof_submitted_at" },
      confirmedAt: { type: "string", field: "confirmed_at" },
      airlockedAt: { type: "string", field: "airlocked_at" },
      redeemed: { type: "boolean", default: () => false },
    },
    indexes: {
      primary: {
        pk: { field: "pk", composite: ["missionId"] },
        sk: { field: "sk", composite: [] },
      },
      bySeat: {
        index: "GSI1",
        pk: { field: "gsi1pk", composite: ["seatId"] },
        sk: { field: "gsi1sk", composite: ["declaredAt"] },
      },
      byStatusDeadline: {
        index: "GSI2",
        pk: { field: "gsi2pk", composite: ["status"] },
        sk: { field: "gsi2sk", composite: ["deadlineAt"] },
      },
    },
  },
  { client: dynamo, table: tableName },
);

export const StackEntity = new Entity(
  {
    model: {
      entity: "stack",
      version: "1",
      service: "airlock",
    },
    attributes: {
      entityType: { type: "string", field: "entity_type", default: () => "stack" },
      seatId: { type: "string", field: "seat_id", required: true },
      donationEnabled: { type: "boolean", field: "donation_enabled", default: () => false },
      donationAmount: { type: "number", field: "donation_amount" },
      donationTarget: { type: "string", field: "donation_target" },
      crewAlertEnabled: { type: "boolean", field: "crew_alert_enabled", default: () => false },
      crewAlertContacts: {
        type: "list",
        field: "crew_alert_contacts",
        items: { type: "string" },
        default: () => [],
      },
      theWakeEnabled: { type: "boolean", field: "the_wake_enabled", default: () => false },
      reentryChallengeEnabled: {
        type: "boolean",
        field: "reentry_challenge_enabled",
        default: () => false,
      },
      theWakeCrosspost: {
        type: "list",
        field: "the_wake_crosspost",
        items: { type: "string" },
        default: () => [],
      },
    },
    indexes: {
      primary: {
        pk: { field: "pk", composite: ["seatId"] },
        sk: { field: "sk", composite: [] },
      },
    },
  },
  { client: dynamo, table: tableName },
);

export const EventEntity = new Entity(
  {
    model: {
      entity: "event",
      version: "1",
      service: "airlock",
    },
    attributes: {
      entityType: { type: "string", field: "entity_type", default: () => "event" },
      eventId: { type: "string", field: "event_id", required: true },
      kind: {
        type: ["BOARDED", "DECLARED", "PROOF_SUBMITTED", "CONFIRMED", "AIRLOCKED", "RE_BOARDED", "REDEEMED"] as const,
        required: true,
      },
      userId: { type: "string", field: "user_id", required: true },
      seatId: { type: "string", field: "seat_id", required: true },
      missionId: { type: "string", field: "mission_id" },
      occurredAt: { type: "string", field: "occurred_at", required: true },
      message: { type: "string" },
      wakePosted: { type: "boolean", field: "wake_posted", default: () => false },
      payload: { type: "map", properties: {} },
    },
    indexes: {
      primary: {
        pk: { field: "pk", composite: ["eventId"] },
        sk: { field: "sk", composite: [] },
      },
      byKind: {
        index: "GSI1",
        pk: { field: "gsi1pk", composite: ["kind"] },
        sk: { field: "gsi1sk", composite: ["occurredAt"] },
      },
      byUser: {
        index: "GSI2",
        pk: { field: "gsi2pk", composite: ["userId"] },
        sk: { field: "gsi2sk", composite: ["occurredAt"] },
      },
      bySeat: {
        index: "GSI3",
        pk: { field: "gsi3pk", composite: ["seatId"] },
        sk: { field: "gsi3sk", composite: ["occurredAt"] },
      },
    },
  },
  { client: dynamo, table: tableName },
);

export const CounterEntity = new Entity(
  {
    model: {
      entity: "counter",
      version: "1",
      service: "airlock",
    },
    attributes: {
      entityType: { type: "string", field: "entity_type", default: () => "counter" },
      counterId: { type: "string", field: "counter_id", required: true },
      lastSeatNumber: { type: "number", field: "last_seat_number" },
    },
    indexes: {
      primary: {
        pk: { field: "pk", composite: ["counterId"] },
        sk: { field: "sk", composite: [] },
      },
    },
  },
  { client: dynamo, table: tableName },
);

export const ProfileViewEntity = new Entity(
  {
    model: {
      entity: "profileView",
      version: "1",
      service: "airlock",
    },
    attributes: {
      entityType: { type: "string", field: "entity_type", default: () => "profile_view" },
      handle: { type: "string", required: true },
      views: { type: "number", default: () => 0 },
      updatedAt: { type: "string", field: "updated_at" },
    },
    indexes: {
      primary: {
        pk: { field: "pk", composite: ["handle"] },
        sk: { field: "sk", composite: [] },
      },
    },
  },
  { client: dynamo, table: tableName },
);

export const airlockService = new Service(
  {
    user: UserEntity,
    seat: SeatEntity,
    mission: MissionEntity,
    stack: StackEntity,
    event: EventEntity,
    counter: CounterEntity,
    profileView: ProfileViewEntity,
  },
  { client: dynamo, table: tableName },
);

export const starterService = airlockService;

export type AirlockService = typeof airlockService;
export type StarterService = typeof starterService;
