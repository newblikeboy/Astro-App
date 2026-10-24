const { chromium } = require("../web/node_modules/playwright");
const fs = require("node:fs");
const assert = require("node:assert/strict");
(async () => {
  const browser = await chromium.launch({
    executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe",
    headless: true,
  });
  try {
    const context = await browser.newContext({
      viewport: { width: 1440, height: 1000 },
      reducedMotion: "reduce",
    });
    const report = JSON.parse(
      fs.readFileSync(".local/kundli-v2-fixture.json", "utf8"),
    );
    report.birth.name = "Aarav Sharma";
    await context.addInitScript((report) => {
      localStorage.setItem("m4y-language", "en");
      localStorage.setItem("m4y-analytics", "no");
      sessionStorage.setItem("m4y-guest-report", JSON.stringify(report));
    }, report);
    const page = await context.newPage();
    await page.route("**/api/**", (route) =>
      route.request().method() === "GET" ? route.continue() : route.abort(),
    );
    await page.goto("http://localhost:5173/kundli");
    await page.locator(".chart-viewer").waitFor();
    for (const lang of ["en", "hi"]) {
      await page.locator(".language-selector select").selectOption(lang);
      for (const width of [320, 339, 1440]) {
        await page.setViewportSize({ width, height: 1000 });
        await page.evaluate(() => document.fonts.ready);
        await page.evaluate(
          () =>
            new Promise((resolve) =>
              requestAnimationFrame(() => requestAnimationFrame(resolve)),
            ),
        );
        const clipped = await page
          .locator(".chart-legend button")
          .evaluateAll((buttons) =>
            buttons
              .filter((b) => b.scrollWidth > b.clientWidth + 1)
              .map((b) => b.textContent),
          );
        assert.deepEqual(
          clipped,
          [],
          `${lang}/${width}: clipped planet labels`,
        );
        await page
          .locator(".report-chart")
          .screenshot({ path: `.local/explorer-chart-${lang}-${width}.png` });
        await page.locator(".natal-table").screenshot({
          path: `.local/explorer-planets-${lang}-${width}.png`,
          style: "header,.dev-strip { visibility:hidden!important; }",
        });
      }
    }
    console.log("Captured chart and natal table, both languages and sizes.");
  } finally {
    await browser.close();
  }
})().catch((e) => {
  console.error(e);
  process.exitCode = 1;
});
