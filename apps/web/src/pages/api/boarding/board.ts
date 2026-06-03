import type { APIRoute } from "astro";

import { forwardAuthenticatedJSON } from "../../../lib/api-proxy";

export const prerender = false;

// The manifest page submits the boarding details. Accept either a JSON body
// (preferred) or an HTML form post and normalize to the JSON contract the Go
// API expects: { handle, display_name }. Then forward with the Clerk session
// token attached. The `boarding` feature gate is enforced upstream by the Go
// middleware (402/403 if not entitled).
export const POST: APIRoute = async (ctx) => {
  const contentType = ctx.request.headers.get("content-type") ?? "";
  if (contentType.includes("application/x-www-form-urlencoded") || contentType.includes("multipart/form-data")) {
    const form = await ctx.request.formData();
    const body = JSON.stringify({
      handle: form.get("handle")?.toString() ?? "",
      display_name: form.get("display_name")?.toString() ?? "",
    });
    const jsonRequest = new Request(ctx.request.url, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body,
    });
    return forwardAuthenticatedJSON({ ...ctx, request: jsonRequest }, "/api/boarding/board");
  }

  return forwardAuthenticatedJSON(ctx, "/api/boarding/board");
};
