// Stateless preview QA: all customer/provider writes are blocked.
const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const assert = require("node:assert/strict");
const fs = require("node:fs");
/** verify checks a bounded group of extended panels and the shared date/error interactions in a fresh browser. */
async function verify(language, groups) {
  const browser = await chromium.launch({
    executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe",
    headless: true,
    args: ["--disable-gpu", "--renderer-process-limit=2"],
  });
  const errors = [],
    views = [];
  let audits = 0;
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
    page.on("pageerror", (e) => errors.push(e.message));
    await page.route("**/api/**", (r) =>
      r.request().method() === "GET" ? r.continue() : r.abort(),
    );
    await page.goto(process.env.PANCHANG_QA_URL || "http://127.0.0.1:18082");
    const section = page.locator("#panchang");
    await page.locator(".panchang-essentials").waitFor();
    await page.locator("#daily-date").fill("2026-10-18");
    /** openDetails waits for the selected response before expanding the optional extended results. */
    async function openDetails() {
      const date = await page.locator("#daily-date").inputValue();
      await page.locator(`.daily-date-context time[datetime="${date}"]`).waitFor();
      const disclosure = page.locator(".panchang-extended-disclosure");
      if (!(await disclosure.evaluate(el => el.open))) await disclosure.locator(":scope > summary").click();
      await page.locator(".panchang-details").waitFor();
    }
    await openDetails();
    /** check checks extended Panchang layout and verifies that displayed clocks omit seconds. */
    async function check(lang, width, view) {
      assert(
        !/\b\d{1,2}:\d{2}:\d{2}\b/.test(await section.innerText()),
        "Displayed extended clocks must omit seconds",
      );
      await page.setViewportSize({ width, height: 1000 });
      await page.evaluate(() => document.fonts.ready);
      await page.evaluate(
        () =>
          new Promise((resolve) =>
            requestAnimationFrame(() => requestAnimationFrame(resolve)),
          ),
      );
      const metrics = await section.evaluate((el) => ({
        page: document.documentElement.scrollWidth - innerWidth,
        section: el.scrollWidth - el.clientWidth,
        clipped: [
          ...el.querySelectorAll(
            ".panchang-detail-tabs button,.panchang-view-switch button,.panchang-calendar-grid button,.extended-periods li,.balam-controls select,.panchang-calendar-facts dd",
          ),
        ]
          .filter((n) => n.scrollWidth > n.clientWidth + 1)
          .map((n) => n.className || n.tagName),
      }));
      assert(
        metrics.page <= 1 && metrics.section <= 1,
        JSON.stringify({ lang, width, view, metrics }),
      );
      assert.deepEqual(
        metrics.clipped,
        [],
        JSON.stringify({ lang, width, view, metrics }),
      );
      if ([320, 1440].includes(width)) {
        const result = await new AxeBuilder({ page })
          .include("#panchang")
          .analyze();
        assert.deepEqual(
          result.violations.map((v) => ({
            id: v.id,
            nodes: v.nodes.map((n) => n.target),
          })),
          [],
          `${lang}/${width}/${view}`,
        );
        audits++;
      }
      if ([339, 1440].includes(width)) {
        await section.screenshot({
          path: `.local/panchang-extended-${lang}-${width}-${view}.png`,
          style: "header,.dev-strip { visibility:hidden!important; }",
        });
      }
      views.push({ lang, width, view });
    }
    for (const lang of [language]) {
      await page.locator(".language-selector select").selectOption(lang);
      await openDetails();
      for (const group of groups) {
        console.log(`Checking ${lang}/${group}`);
        await page.locator(`#panchang-detail-${group}`).click();
        if (group === "yogas") {
          await page.locator(".balam-controls select").nth(0).selectOption("1");
          await page.locator(".balam-controls select").nth(1).selectOption("1");
          assert(
            (await page
              .locator("#panchang-detail-panel .extended-periods li")
              .count()) > 1,
          );
        }
        if (group === "lagna")
          assert(
            (await page
              .locator("#panchang-detail-panel .extended-split section")
              .first()
              .locator("li")
              .count()) >= 12,
          );
        for (const width of [320, 339, 650, 768, 1100, 1440])
          await check(lang, width, group);
      }
      assert.equal(
        await page
          .locator(".panchang-view-switch,.panchang-calendar-grid")
          .count(),
        0,
      );
      await page.locator("#daily-date").fill("2026-10-18");
      await page.locator(".daily-date-nav button").last().click();
      assert.equal(
        await page.locator("#daily-date").inputValue(),
        "2026-10-19",
      );
      await page.locator(".daily-date-nav button").first().click();
      assert.equal(
        await page.locator("#daily-date").inputValue(),
        "2026-10-18",
      );
    }
    await page.locator(".language-selector select").selectOption("en");
    await openDetails();
    await page.locator("#panchang-detail-calendar").focus();
    await page.keyboard.press("End");
    assert.equal(
      await page
        .locator("#panchang-detail-directions")
        .getAttribute("aria-selected"),
      "true",
    );
    await page.keyboard.press("Home");
    // The revised calendar must retain real leap days and Adhika month metadata.
    await page.locator("#daily-date").fill("2023-08-07");
    await openDetails();
    await page.waitForFunction(() =>
      document
        .querySelector(".panchang-calendar-facts")
        ?.textContent.includes("Adhika month"),
    );
    assert.match(
      await page.locator(".panchang-calendar-facts").innerText(),
      /Shravana/,
    );
    // Daily date navigation must retain leap days without a weekly/monthly view.
    await page.locator("#daily-date").fill("2024-02-28");
    await page.locator(".daily-date-nav button").last().click();
    assert.equal(await page.locator("#daily-date").inputValue(), "2024-02-29");
    await page.locator(".daily-date-nav button").last().click();
    assert.equal(await page.locator("#daily-date").inputValue(), "2024-03-01");
    // A daily fetch failure must retain the controls and expose the existing retry.
    let fail = true;
    await page.route("**/api/panchang?**", (route) =>
      fail
        ? route.fulfill({ status: 503, json: { error: "unavailable" } })
        : route.continue(),
    );
    await page.locator("#daily-date").fill("2024-03-19");
    await page.locator(".daily-panel-content [role=alert]").waitFor();
    assert.equal(await page.locator("#daily-date").inputValue(), "2024-03-19");
    fail = false;
    await page.locator(".daily-panel-content .text-button").click();
    await openDetails();
    assert.deepEqual(errors, []);
    const result = {
      passed: true,
      views: views.length,
      axeAudits: audits,
      dailyOnly: true,
      dailyNavigation: true,
      adhika: true,
      leapDay: true,
      errorRetry: true,
      errors,
    };
    console.log(JSON.stringify(result));
    return result;
  } finally {
    await browser.close();
  }
}
/** run releases the browser between panel batches so extended accessibility scans fit the workstation memory limit. */
async function run() {
  const results = [];
  for (const lang of ["en", "hi"])
    for (const groups of [["calendar", "muhurats", "yogas"], ["lagna", "directions"]])
      results.push(await verify(lang, groups));
  const result = { ...results[0], views: results.reduce((n, r) => n + r.views, 0), axeAudits: results.reduce((n, r) => n + r.axeAudits, 0) };
  fs.writeFileSync(".local/panchang-extended-results.json", JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result));
}
run().catch((e) => {
  console.error(e.stack);
  process.exitCode = 1;
});
