import { sveltekit } from "@sveltejs/kit/vite"; // SvelteKit plugin that wires Svelte into Vite
import { defineConfig } from "vite"; // helper that provides TypeScript types for the config object

// When running inside Docker the compose file sets API_URL / WORKER_URL to the
// container service names.  Outside Docker both fall back to localhost so the
// plain `bun run dev` workflow is unchanged.
const apiTarget = process.env.API_URL ?? "http://localhost:8080";
const workerTarget = process.env.WORKER_URL ?? "http://localhost:8081";

export default defineConfig({
  plugins: [sveltekit()], // register the SvelteKit plugin so Vite can process .svelte files
  server: {
    host: true, // bind to 0.0.0.0 so the dev server is reachable from outside the container
    proxy: {
      // /api/* is forwarded to the Go API server, stripping the /api prefix before forwarding
      "/api": {
        target: apiTarget, // resolved from API_URL env var or localhost fallback
        rewrite: (path) => path.replace(/^\/api/, ""), // strip /api so the API sees the original path (e.g. /health)
      },
      // /worker/* is forwarded to the Python AI worker health server, stripping the /worker prefix
      "/worker": {
        target: workerTarget, // resolved from WORKER_URL env var or localhost fallback
        rewrite: (path) => path.replace(/^\/worker/, ""), // strip /worker so the worker sees the original path (e.g. /health)
      },
    },
  },
});
