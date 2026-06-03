import { clerkMiddleware, createRouteMatcher } from "@clerk/astro/server";
import { defineMiddleware, sequence } from "astro:middleware";
import { isAdminUserID } from "./lib/auth";

// Conservative-but-permissive baseline security headers.
// `style-src 'unsafe-inline'` is required because the landing markup uses
// inline `style="..."` attributes pervasively and Clerk injects runtime
// component styles. Clerk's CSP requirements also need its frontend API,
// hosted images, Cloudflare challenge frames, workers, and Stripe frames for
// billing UI.
const SECURITY_HEADERS: Record<string, string> = {
  "Strict-Transport-Security": "max-age=63072000; includeSubDomains; preload",
  "X-Content-Type-Options": "nosniff",
  "Referrer-Policy": "strict-origin-when-cross-origin",
  "X-Frame-Options": "DENY",
  "Content-Security-Policy":
    "default-src 'self'; " +
    "base-uri 'self'; " +
    "object-src 'none'; " +
    "script-src 'self' 'unsafe-inline' https: http:; " +
    "style-src 'self' 'unsafe-inline'; " +
    "font-src 'self'; " +
    "connect-src 'self' https://api.stripe.com https://clerk-telemetry.com https://*.clerk-telemetry.com https:; " +
    "img-src 'self' data: https://img.clerk.com https:; " +
    "worker-src 'self' blob:; " +
    // Stripe frames stay because Clerk Billing renders Stripe payment UI inside
    // its checkout. `https://*.clerk.accounts.dev` covers the dev/preview Clerk
    // checkout frame; in production add this instance's own Clerk domain
    // (e.g. `https://clerk.<APP_DOMAIN>` / its `accounts.<APP_DOMAIN>`).
    "frame-src 'self' https://challenges.cloudflare.com https://*.js.stripe.com https://js.stripe.com https://hooks.stripe.com https://*.clerk.accounts.dev; " +
    "form-action 'self' https:; " +
    "frame-ancestors 'none'",
};

const isProtectedRoute = createRouteMatcher(["/manifest(.*)", "/dashboard(.*)", "/admin/review(.*)"]);
const isAdminRoute = createRouteMatcher(["/admin/review(.*)"]);

const clerkAuth = clerkMiddleware((auth, ctx) => {
  const session = auth();
  if (!session.isAuthenticated && isProtectedRoute(ctx.request)) {
    const loginURL = new URL("/auth/login", ctx.url);
    loginURL.searchParams.set("return_to", `${ctx.url.pathname}${ctx.url.search}`);
    return Response.redirect(loginURL, 302);
  }
  if (isAdminRoute(ctx.request) && !isAdminUserID(session.userId)) {
    return new Response("Forbidden", { status: 403 });
  }
});

const securityHeaders = defineMiddleware(async (_ctx, next) => {
  const response = await next();
  // Astro's dev server injects inline HMR scripts that a strict `script-src`
  // would block; only apply CSP in production builds so dev iteration still
  // works. Test the production policy with `astro preview` or after deploy.
  if (import.meta.env.PROD) {
    for (const [name, value] of Object.entries(SECURITY_HEADERS)) {
      response.headers.set(name, value);
    }
  }
  return response;
});

export const onRequest = sequence(clerkAuth, securityHeaders);
