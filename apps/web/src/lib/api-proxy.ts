import type { APIContext } from "astro";

import { clerkSessionToken } from "./auth";

export async function forwardAuthenticatedJSON(ctx: APIContext, upstreamPath: string): Promise<Response> {
  const token = await clerkSessionToken(ctx);
  if (!token) {
    return json({ error: "Unauthorized" }, 401);
  }

  const rawBase = import.meta.env.PUBLIC_API_URL;
  if (typeof rawBase !== "string" || rawBase.trim() === "") {
    return json({ error: "PUBLIC_API_URL is not configured" }, 503);
  }

  const upstream = new URL(upstreamPath, rawBase);
  const method = ctx.request.method;
  const body = method === "GET" || method === "HEAD" ? undefined : await ctx.request.text();
  const response = await fetch(upstream, {
    method,
    headers: {
      accept: "application/json",
      "content-type": ctx.request.headers.get("content-type") ?? "application/json",
      authorization: `Bearer ${token}`,
    },
    ...(body === undefined ? {} : { body }),
  });

  return new Response(response.body, {
    status: response.status,
    headers: {
      "Content-Type": response.headers.get("Content-Type") ?? "application/json; charset=utf-8",
      "Cache-Control": "no-store",
    },
  });
}

// Like forwardAuthenticatedJSON, but forwards the raw request body (e.g. multipart
// file uploads) and preserves the original content-type header so the upstream Go
// API receives the boundary intact.
export async function forwardAuthenticatedRaw(ctx: APIContext, upstreamPath: string): Promise<Response> {
  const token = await clerkSessionToken(ctx);
  if (!token) {
    return json({ error: "Unauthorized" }, 401);
  }

  const rawBase = import.meta.env.PUBLIC_API_URL;
  if (typeof rawBase !== "string" || rawBase.trim() === "") {
    return json({ error: "PUBLIC_API_URL is not configured" }, 503);
  }

  const upstream = new URL(upstreamPath, rawBase);
  const method = ctx.request.method;
  const body = method === "GET" || method === "HEAD" ? undefined : await ctx.request.arrayBuffer();
  const headers: Record<string, string> = {
    accept: "application/json",
    authorization: `Bearer ${token}`,
  };
  const contentType = ctx.request.headers.get("content-type");
  if (contentType) {
    headers["content-type"] = contentType;
  }

  const response = await fetch(upstream, {
    method,
    headers,
    ...(body === undefined ? {} : { body }),
  });

  return new Response(response.body, {
    status: response.status,
    headers: {
      "Content-Type": response.headers.get("Content-Type") ?? "application/json; charset=utf-8",
      "Cache-Control": "no-store",
    },
  });
}

function json(payload: unknown, status: number): Response {
  return new Response(JSON.stringify(payload), {
    status,
    headers: {
      "Content-Type": "application/json; charset=utf-8",
      "Cache-Control": "no-store",
    },
  });
}
