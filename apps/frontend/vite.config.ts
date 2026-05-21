import { sveltekit } from "@sveltejs/kit/vite"; // SvelteKit plugin that wires Svelte into Vite
import { defineConfig } from "vite"; // helper that provides TypeScript types for the config object

export default defineConfig({
  plugins: [sveltekit()], // register the SvelteKit plugin so Vite can process .svelte files
  server: {
    proxy: {
      // /api/* is forwarded to the Go API server, stripping the /api prefix before forwarding
      "/api": {
        target: "http://localhost:8080", // Go API listens on this port
        rewrite: (path) => path.replace(/^\/api/, ""), // strip /api so the API sees the original path (e.g. /health)
      },
      // /worker/* is forwarded to the Python AI worker health server, stripping the /worker prefix
      "/worker": {
        target: "http://localhost:8081", // Python worker listens on this port
        rewrite: (path) => path.replace(/^\/worker/, ""), // strip /worker so the worker sees the original path (e.g. /health)
      },
    },
  },
});
