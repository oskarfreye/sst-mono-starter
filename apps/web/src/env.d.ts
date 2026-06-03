/// <reference types="@clerk/astro/env" />

interface ImportMetaEnv {
  readonly DEV: boolean;
  readonly PROD: boolean;
  readonly PUBLIC_API_URL: string;
  readonly PUBLIC_CLERK_PUBLISHABLE_KEY: string;
  readonly CLERK_PUBLISHABLE_KEY: string;
  readonly CLERK_SECRET_KEY: string;
  readonly PUBLIC_CLERK_SIGN_IN_URL: string;
  readonly PUBLIC_CLERK_SIGN_UP_URL: string;
  readonly PUBLIC_ASSETS_BASE_URL: string;
  readonly PUBLIC_DISCORD_URL: string;
  readonly ADMIN_USER_IDS: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
