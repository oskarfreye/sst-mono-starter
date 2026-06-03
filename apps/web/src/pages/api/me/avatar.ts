import type { APIRoute } from "astro";

import { forwardAuthenticatedJSON, forwardAuthenticatedRaw } from "../../../lib/api-proxy";

export const prerender = false;

// Multipart upload -> forward the raw body (preserves the boundary).
export const POST: APIRoute = (ctx) => forwardAuthenticatedRaw(ctx, "/api/me/avatar");

// Drop the visor: clear the generated avatar.
export const DELETE: APIRoute = (ctx) => forwardAuthenticatedJSON(ctx, "/api/me/avatar");

// Convenience status read.
export const GET: APIRoute = (ctx) => forwardAuthenticatedJSON(ctx, "/api/me/avatar");
