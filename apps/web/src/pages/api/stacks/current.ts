import type { APIContext, APIRoute } from "astro";

import { clerkSessionToken } from "../../../lib/auth";
import { forwardAuthenticatedJSON } from "../../../lib/api-proxy";

export const GET: APIRoute = (ctx) => forwardAuthenticatedJSON(ctx, "/api/stacks/current");

export const POST: APIRoute = (ctx) => updateStack(ctx);
export const PUT: APIRoute = (ctx) => updateStack(ctx);

async function updateStack(ctx: APIContext): Promise<Response> {
  const token = await clerkSessionToken(ctx);
  if (!token) {
    return json({ error: "Unauthorized" }, 401);
  }

  const rawBase = import.meta.env.PUBLIC_API_URL;
  if (typeof rawBase !== "string" || rawBase.trim() === "") {
    return json({ error: "PUBLIC_API_URL is not configured" }, 503);
  }

  const payload = await readStackPayload(ctx.request);
  const response = await fetch(new URL("/api/stacks/current", rawBase), {
    method: "PUT",
    headers: {
      accept: "application/json",
      "content-type": "application/json",
      authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(payload),
  });

  return new Response(response.body, {
    status: response.status,
    headers: {
      "Content-Type": response.headers.get("Content-Type") ?? "application/json; charset=utf-8",
      "Cache-Control": "no-store",
    },
  });
}

async function readStackPayload(request: Request) {
  const contentType = request.headers.get("content-type") ?? "";
  const raw = contentType.includes("application/json")
    ? ((await request.json().catch(() => ({}))) as Record<string, unknown>)
    : Object.fromEntries(await request.formData());

  return {
    crew_alert_enabled: parseBoolean(raw.crew_alert_enabled),
    crew_alert_contacts: parseContacts(raw.crew_alert_contacts),
    the_wake_enabled: parseBoolean(raw.the_wake_enabled),
    reentry_challenge_enabled: parseBoolean(raw.reentry_challenge_enabled),
  };
}

function parseBoolean(value: unknown): boolean {
  return value === true || value === "true" || value === "on" || value === "1";
}

function parseContacts(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.flatMap((entry) => parseContacts(entry));
  }
  if (typeof value !== "string") return [];
  return value
    .split(/\r?\n|,/)
    .map((entry) => entry.trim())
    .filter(Boolean);
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
