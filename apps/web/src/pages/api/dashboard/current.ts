import type { APIRoute } from "astro";

import { forwardAuthenticatedJSON } from "../../../lib/api-proxy";

export const GET: APIRoute = (ctx) => forwardAuthenticatedJSON(ctx, "/api/me/airlock");
