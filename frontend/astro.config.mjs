// @ts-check
import { defineConfig } from 'astro/config';
import node from "@astrojs/node";
import svelte from '@astrojs/svelte';
import vercel from '@astrojs/vercel';

// https://astro.build/config
export default defineConfig({
  output: 'server',

  adapter: vercel({
    edgeMiddleware: false,
  }),
  // adapter: node({
  //   mode: "standalone"
  // }),

  integrations: [svelte()],
  prefetch: false
});