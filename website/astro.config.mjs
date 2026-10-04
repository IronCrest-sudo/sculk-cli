// @ts-check
import { defineConfig } from 'astro/config';

import tailwindcss from '@tailwindcss/vite';
import svelte from '@astrojs/svelte';
import icon from 'astro-icon';
import iconset from 'astro-iconset';

// https://astro.build/config
export default defineConfig({
  vite: {
    plugins: [tailwindcss()]
  },

  // `astro dev`/`astro preview` are used behind proxies (Codespaces, e2b,
  // ngrok, ...) whose hostname is not localhost, so accept any host there.
  // The built site is static and unaffected by this.
  server: { allowedHosts: true },
  preview: { allowedHosts: true },

  integrations: [svelte(), icon(), iconset()]
});