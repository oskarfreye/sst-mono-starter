import { dsql } from "./database";
import { AUTH_DOMAIN, HAS_CUSTOM_DOMAIN } from "./env";

// Dedicated table for OpenAuth storage (sessions, codes, refresh tokens).
// Kept separate from the application Dynamo table to keep access patterns
// simple and to make blast-radius smaller if you ever want to reset it.
//
// Hardening (H-I1):
//  - PITR is on by default in `sst.aws.Dynamo`.
//  - Deletion protection enabled on protected stages.
//
// TODO(L-Auth3): Consider encrypting this table with a customer-managed KMS
// key (CMK) so credential rotation and key access are auditable separately
// from the AWS-owned default key. Wire it via `transform.table` here:
//   transform: { table: (args) => { args.serverSideEncryption = {
//     enabled: true, kmsKeyArn: myCmk.arn }; } }
// Skipped now — adds a KMS resource + IAM scope and recurring cost.
const isProtectedStage = ["production", "stage"].includes($app.stage);

export const authTable = new sst.aws.Dynamo("AuthTable", {
  fields: {
    pk: "string",
    sk: "string",
  },
  primaryIndex: { hashKey: "pk", rangeKey: "sk" },
  ttl: "ttl",
  deletionProtection: isProtectedStage,
});

// OpenAuth issuer Lambda. `sst.aws.Auth` sets up a Function + Router +
// CloudFront distribution and exposes `.url` for the issuer.
//
// Hardening (H-I2):
//  - Custom domain (auth.<APP_DOMAIN>) when one is configured. CloudFront
//    issues an ACM cert and uses the SST default minimumProtocolVersion of
//    TLSv1.2_2021 (see .sst/platform/src/components/aws/cdn.ts).
//  - Without a custom domain we keep the default CloudFront URL so the
//    starter still deploys for users who haven't set APP_DOMAIN.
export const auth = new sst.aws.Auth("Auth", {
  issuer: {
    handler: "apps/auth/src/index.handler",
    // `dsql` linked so the issuer can persist users via the Drizzle client in
    // `@starter/database/sql` (the package apps/auth already depends on). Drop
    // it if you keep auth on the Dynamo backend.
    link: [authTable, dsql],
    environment: {
      STAGE: $app.stage,
      DSQL_ENDPOINT: dsql.endpoint,
      DSQL_REGION: dsql.region,
      // Gates the dev-mode OTP logger in apps/auth/src/index.ts. Must never
      // be "1" on shared stages — `$dev` is only true under `sst dev`.
      AUTH_DEV_LOG_CODES: $dev ? "1" : "0",
      // Comma-separated list of origins the issuer is allowed to redirect
      // back to after auth. Set via env at deploy time.
      ALLOWED_REDIRECT_ORIGINS: process.env.ALLOWED_REDIRECT_ORIGINS ?? "",
      // Client registry. Format: "id=uri1,uri2;id2=uri1"
      AUTH_CLIENTS: process.env.AUTH_CLIENTS ?? "",
    },
  },
  ...(HAS_CUSTOM_DOMAIN ? { domain: AUTH_DOMAIN } : {}),
});
