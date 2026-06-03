// Declare secrets here. Set values with: `bun sst secret set <Name> <value>`.
// Pass the Secret resource into `link: [...]` on functions that need it;
// access it at runtime via `Resource.<Name>.value`.

// Clerk server-side secret key, consumed by the Astro SSR runtime (see
// infra/frontend.ts). Set with: `bun sst secret set ClerkSecretKey <sk_...>`.
// The publishable key and Clerk issuer URL are public, so they stay as plain
// env vars rather than secrets.
export const ClerkSecretKey = new sst.Secret("ClerkSecretKey");
