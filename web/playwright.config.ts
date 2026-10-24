import { defineConfig } from "@playwright/test";
const baseURL = process.env.E2E_URL || "http://127.0.0.1:18081";
const target = new URL(baseURL);
if (
  !["127.0.0.1", "localhost"].includes(target.hostname) ||
  target.protocol !== "http:"
)
  throw new Error("Browser tests require an isolated localhost URL");
export default defineConfig({
  testDir: "tests",
  timeout: 120000,
  expect: { timeout: 15000 },
  workers: 1,
  use: {
    baseURL,
    headless: true,
    launchOptions: {
      executablePath:
        process.env.CHROME_PATH ||
        "C:/Program Files/Google/Chrome/Application/chrome.exe",
    },
    trace: "retain-on-failure",
  },
  webServer: {
    command: "go run ./cmd/server",
    cwd: "..",
    url: baseURL + "/api/health",
    timeout: 60000,
    reuseExistingServer: false,
    env: {
      ENV_FILE: "none",
      GOMEMLIMIT: "256MiB",
      MONGODB_URI: "",
      APP_ENV: "development",
      HTTP_ADDR: `127.0.0.1:${target.port}`,
      APP_ORIGIN: baseURL,
      DATA_DIR: `.local/e2e-${Date.now()}`,
      ADMIN_PASSWORD: "local-admin-change-me",
      ADMIN_TOTP_SECRET: "",
      PAYMENT_MODE: "simulation",
      OTP_MODE: "simulation",
      ZOOM_MODE: "simulation",
      NOTIFICATION_URL: "",
      NOTIFICATION_TOKEN: "",
    },
  },
  reporter: [["list"], ["json", { outputFile: "../.local/e2e-results.json" }]],
});
