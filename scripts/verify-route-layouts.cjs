// Read-only visual QA: never creates accounts or changes the user's Atlas data.
const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const assert = require("node:assert/strict");
const fs = require("node:fs");
(async () => {
  const browser = await chromium.launch({
    executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe",
    headless: true,
  });
  const errors = [],
    layouts = [],
    audits = [];
  try {
    const context = await browser.newContext({
      viewport: { width: 1440, height: 1000 },
      reducedMotion: "reduce",
    });
    await context.addInitScript(() => {
      localStorage.setItem("m4y-language", "en");
      localStorage.setItem("m4y-analytics", "no");
    });
    const page = await context.newPage();
    page.on("pageerror", (error) => errors.push(error.message));
    await page.route("**/api/**", (route) =>
      route.request().method() === "GET" ? route.continue() : route.abort(),
    );
    for (const [path, name, ready] of [
      ["/login", "customer-login", ".auth-card"],
      ["/admin/login", "admin-login", ".admin-login"],
    ]) {
      const requests = [];
      /** record records requested API paths to verify route/session separation. */
      const record = (request) =>
        requests.push(new URL(request.url()).pathname);
      page.on("request", record);
      await page.goto("http://localhost:5173" + path);
      await page.locator(ready).waitFor();
      assert.equal(
        await page
          .locator(".site-header,.site-footer,.hero,.consent,dialog")
          .count(),
        0,
      );
      if (name === "admin-login")
        assert(
          !requests.some(
            (path) => path === "/api/me" || path === "/api/events",
          ),
        );
      page.off("request", record);
      for (const language of ["en", "hi"]) {
        await page.locator(".language-selector select").selectOption(language);
        for (const width of [320, 339, 390, 650, 768, 901, 1100, 1440]) {
          await page.setViewportSize({ width, height: 950 });
          await page.evaluate(async () => {
            await document.fonts.ready;
            await new Promise((resolve) =>
              requestAnimationFrame(() => requestAnimationFrame(resolve)),
            );
          });
          const state = await page.evaluate(() => ({
            overflow: document.documentElement.scrollWidth - innerWidth,
            h1: document.querySelectorAll("h1").length,
            clipped: [
              ...document.querySelectorAll("input,button,.portal-header a"),
            ]
              .filter((node) => {
                const r = node.getBoundingClientRect();
                return r.width && (r.left < -1 || r.right > innerWidth + 1);
              })
              .map((node) => node.outerHTML.slice(0, 100)),
          }));
          assert(
            state.overflow <= 1 && state.h1 === 1 && !state.clipped.length,
            JSON.stringify({ name, language, width, state }),
          );
          if ([320, 1440].includes(width)) {
            const axe = await new AxeBuilder({ page }).analyze();
            assert.deepEqual(
              axe.violations.map((v) => ({
                id: v.id,
                nodes: v.nodes.map((n) => n.target),
              })),
              [],
              JSON.stringify({ name, language, width }),
            );
            audits.push({ name, language, width });
            await page.screenshot({
              path: `.local/routes-${name}-${language}-${width}.png`,
              fullPage: true,
            });
          }
          layouts.push({ name, language, width });
        }
      }
    }
    await page.goto("http://localhost:5173/admin");
    await page.locator(".admin-login").waitFor();
    assert.equal(new URL(page.url()).pathname, "/admin/login");
    await page.goto("http://localhost:5173/account");
    await page.locator(".auth-card").waitFor();
    assert.equal(new URL(page.url()).pathname, "/login");
    assert.equal(new URL(page.url()).searchParams.get("next"), "/account");
    await page.goto("http://localhost:5173");
    await page.locator(".hero").waitFor();
    assert.equal(await page.locator('a[href^="/admin"]').count(), 0);
    assert.equal(
      await page.locator(".account-button").getAttribute("href"),
      "/login",
    );
    assert.deepEqual(errors, []);
    const result = {
      passed: true,
      layouts: layouts.length,
      axeAudits: audits.length,
      separateShells: true,
      adminDoesNotLoadCustomer: true,
      routeGuards: true,
      publicAdminLinks: 0,
      errors,
    };
    fs.writeFileSync(
      ".local/routes-layout-results.json",
      JSON.stringify(result, null, 2),
    );
    console.log(JSON.stringify(result));
  } finally {
    await browser.close();
  }
})().catch((error) => {
  console.error(error.stack);
  process.exitCode = 1;
});
