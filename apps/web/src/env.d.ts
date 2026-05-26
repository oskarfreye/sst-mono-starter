interface ImportMetaEnv {
  readonly PUBLIC_API_URL: string;
  readonly PUBLIC_ASSETS_BASE_URL: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
