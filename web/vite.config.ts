import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

/** sourceRevision identifies the published source of this build, including builds from source archives. */
function sourceRevision(): string {
  const revision = process.env.SOURCE_REVISION || execFileSync(
    "git", ["rev-parse", "HEAD"],
    { cwd: fileURLToPath(new URL("..", import.meta.url)), encoding: "utf8" },
  ).trim();
  if (!/^[a-f0-9]{40}$/.test(revision)) {
    throw new Error("SOURCE_REVISION must be the full published Git commit SHA.");
  }
  return revision;
}

export default defineConfig({
  plugins: [react()],
  define: { __APP_SOURCE_REVISION__: JSON.stringify(sourceRevision()) },
  server: {
    // OneDrive/formatters can briefly expose an empty file during a save.
    watch: { awaitWriteFinish: { stabilityThreshold: 300, pollInterval: 50 } },
    proxy: {
      "/api": "http://127.0.0.1:8080",
      "/fonts/noto.ttf": "http://127.0.0.1:8080",
    },
  },
});
