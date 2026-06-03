import type { APIRoute } from "astro";

export const GET: APIRoute = async ({ url }) => {
  const rawBase = import.meta.env.PUBLIC_API_URL;
  if (typeof rawBase !== "string" || rawBase.trim() === "") {
    return new Response("event: error\ndata: {\"error\":\"PUBLIC_API_URL is not configured\"}\n\n", {
      status: 503,
      headers: {
        "Content-Type": "text/event-stream; charset=utf-8",
        "Cache-Control": "no-store",
      },
    });
  }

  const upstream = new URL("/api/events", rawBase);
  if (url.searchParams.get("once") === "1") {
    upstream.searchParams.set("once", "1");
  }
  const response = await fetch(upstream, {
    headers: { accept: "text/event-stream" },
  });

  return new Response(response.body, {
    status: response.status,
    headers: {
      "Content-Type": response.headers.get("Content-Type") ?? "text/event-stream; charset=utf-8",
      "Cache-Control": "no-store",
      "X-Accel-Buffering": "no",
    },
  });
};
