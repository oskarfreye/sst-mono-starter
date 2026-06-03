import type { APIRoute } from "astro";

import { forwardAuthenticatedJSON } from "../../../../../lib/api-proxy";

export const POST: APIRoute = (ctx) => {
  const missionId = ctx.params.missionId ?? "";
  return forwardAuthenticatedJSON(ctx, `/api/admin/missions/${encodeURIComponent(missionId)}/confirm`);
};
