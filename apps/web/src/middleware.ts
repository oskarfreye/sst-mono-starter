import { defineMiddleware } from "astro:middleware";

// Conservative-but-permissive baseline security headers. The CSP allows
// `connect-src https:` because the API URL varies per stage; once we have a
// stable per-stage origin we should narrow this to the exact host.
const SECURITY_HEADERS: Record<string, string> = {
  "Strict-Transport-Security": "max-age=63072000; includeSubDomains; preload",
  "X-Content-Type-Options": "nosniff",
  "Referrer-Policy": "strict-origin-when-cross-origin",
  "X-Frame-Options": "DENY",
  "Content-Security-Policy":
    "default-src 'self'; connect-src 'self' https:; img-src 'self' data: https:; frame-ancestors 'none'",
};

export const onRequest = defineMiddleware(async (_ctx, next) => {
  const response = await next();
  for (const [name, value] of Object.entries(SECURITY_HEADERS)) {
    response.headers.set(name, value);
  }
  return response;
});
