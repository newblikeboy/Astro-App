import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
export default defineConfig({
  plugins: [react()],
  server: {
    // OneDrive/formatters can briefly expose an empty file during a save.
    watch: { awaitWriteFinish: { stabilityThreshold: 300, pollInterval: 50 } },
    proxy: {
      "/api": "http://127.0.0.1:8080",
      "/fonts/noto.ttf": "http://127.0.0.1:8080",
    },
  },
});
