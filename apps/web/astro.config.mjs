// @ts-check
import { defineConfig } from "astro/config";
import clerk from "@clerk/astro";
import aws from "astro-sst";

// SST deploys via the `astro-sst` adapter (Lambda + CloudFront). `sst.aws.Astro`
// (see infra/frontend.ts) builds this project and wires the output up.
// https://astro.build/config
export default defineConfig({
  output: "server",
  integrations: [clerk({ signInUrl: "/auth/login", signUpUrl: "/auth/signup" })],
  adapter: aws(),
});
