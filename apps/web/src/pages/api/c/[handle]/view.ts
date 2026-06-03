import type { APIRoute } from "astro";

export const prerender = false;

export const POST: APIRoute = async ({ params }) => {
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

  const handle = params.handle ?? "";
  const upstream = new URL(`/api/c/${encodeURIComponent(handle)}/view`, rawBase);
  try {
    const response = await fetch(upstream, {
      method: "POST",
      headers: {
        accept: "application/json",
        "content-type": "application/json",
      },
      body: "{}",
    });
    return new Response(response.body, {
      status: response.status,
      headers: {
        "Content-Type": response.headers.get("Content-Type") ?? "application/json; charset=utf-8",
        "Cache-Control": "no-store",
      },
    });
  } catch {
    return new Response(JSON.stringify({ error: "unavailable" }), {
      status: 503,
      headers: {
        "Content-Type": "application/json; charset=utf-8",
        "Cache-Control": "no-store",
      },
    });
  }
};
