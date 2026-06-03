import type { APIContext } from "astro";

type ClerkAuthContext = {
  locals: APIContext["locals"];
};

export function adminUserIDs(): Set<string> {
  return new Set(splitIDs(import.meta.env.ADMIN_USER_IDS));
}

export function isAdminUserID(userID: string | null | undefined): boolean {
  return Boolean(userID && adminUserIDs().has(userID));
}

export async function clerkSessionToken(ctx: ClerkAuthContext): Promise<string | null> {
  const session = ctx.locals.auth();
  if (!session.isAuthenticated) return null;
  try {
    return await session.getToken();
  } catch {
    return null;
  }
}

export function normalizeReturnTo(value: string | null): string {
  if (!value || !value.startsWith("/") || value.startsWith("//")) return "/manifest";
  return value;
}

function splitIDs(value: string | undefined): string[] {
  if (!value) return [];
  return value
    .split(/[\s,]+/)
    .map((entry) => entry.trim())
    .filter(Boolean);
}
