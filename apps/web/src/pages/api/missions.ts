import type { APIRoute } from "astro";

import { forwardAuthenticatedJSON } from "../../lib/api-proxy";

export const POST: APIRoute = (ctx) => forwardAuthenticatedJSON(ctx, "/api/missions");
