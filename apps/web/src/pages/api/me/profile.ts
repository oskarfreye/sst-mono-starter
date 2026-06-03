import type { APIRoute } from "astro";

import { forwardAuthenticatedJSON } from "../../../lib/api-proxy";

export const prerender = false;

export const PUT: APIRoute = (ctx) => forwardAuthenticatedJSON(ctx, "/api/me/profile");
