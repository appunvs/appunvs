import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit()],
  optimizeDeps: {
    exclude: ['wa-sqlite']
  },
  server: {
    fs: {
      // wa-sqlite ships .wasm assets outside the normal module graph
      allow: ['..']
    }
  }
});
