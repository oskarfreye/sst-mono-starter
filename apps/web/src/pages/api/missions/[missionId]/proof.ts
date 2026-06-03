import type { APIRoute } from "astro";

import { forwardAuthenticatedJSON } from "../../../../lib/api-proxy";

export const POST: APIRoute = (ctx) => {
  const missionID = ctx.params.missionId;
  if (!missionID) {
    return new Response(JSON.stringify({ error: "mission id is required" }), {
      status: 400,
      headers: { "Content-Type": "application/json; charset=utf-8" },
    });
  }
  return forwardAuthenticatedJSON(ctx, `/api/missions/${encodeURIComponent(missionID)}/proof`);
};
