import type { APIRoute } from "astro";

export const GET: APIRoute = async () => {
  const rawBase = import.meta.env.PUBLIC_API_URL;
  if (typeof rawBase !== "string" || rawBase.trim() === "") {
    return new Response(JSON.stringify({ error: "PUBLIC_API_URL is not configured" }), {
      status: 503,
      headers: {
        "Content-Type": "application/json; charset=utf-8",
        "Cache-Control": "no-store",
      },
    });
  }

  const upstream = new URL("/api/state", rawBase);
  const response = await fetch(upstream, {
    headers: { accept: "application/json" },
  });

  return new Response(response.body, {
    status: response.status,
    headers: {
      "Content-Type": response.headers.get("Content-Type") ?? "application/json; charset=utf-8",
      "Cache-Control":
        response.headers.get("Cache-Control") ??
        "public, max-age=10, s-maxage=10, stale-while-revalidate=30",
    },
  });
};
