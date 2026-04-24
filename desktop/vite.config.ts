import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

// Tauri expects a fixed port, fails if that port is not available.
// https://v2.tauri.app/start/frontend/sveltekit/
declare const process: { env: Record<string, string | undefined> };
const host = process.env.TAURI_DEV_HOST;

export default defineConfig({
  plugins: [sveltekit()],

  // Prevent vite from obscuring rust errors
  clearScreen: false,
  server: {
    port: 1420,
    strictPort: true,
    host: host || false,
    hmr: host
      ? {
          protocol: 'ws',
          host,
          port: 1421
        }
      : undefined,
    watch: {
      // Tell vite to ignore watching `src-tauri`
      ignored: ['**/src-tauri/**']
    }
  }
});
