// Read-only local QA. All writes are blocked; no customer/test data in Atlas.
const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const assert = require("node:assert/strict");
const fs = require("node:fs");
const { setTimeout: delay } = require("node:timers/promises");

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
      reducedMotion: "reduce",
      viewport: { width: 1440, height: 1000 },
    });
    await context.addInitScript(() => {
      localStorage.setItem("m4y-language", "en");
      localStorage.setItem("m4y-analytics", "no");
      localStorage.removeItem("m4y-panchang-place");
    });
    const page = await context.newPage();
    page.on("pageerror", (e) => errors.push(e.message));
    await page.route("**/api/**", (route) =>
      route.request().method() === "GET" ? route.continue() : route.abort(),
    );
    await page.goto(process.env.PANCHANG_QA_URL || "http://localhost:5173");
    const section = page.locator("#panchang");
    await section.locator(".daily-limbs").waitFor();
    await page.locator("#daily-date").fill("2026-10-18");
    await section.locator(".daily-limbs").waitFor();
    await page.locator("#name").fill("Keep my birth details");
    const originalSunrise = await section
      .locator(".daily-solar b")
      .first()
      .innerText();
    assert.match(await section.innerText(), /Shukla Saptami/);
    for (const language of ["en", "hi"]) {
      await page.locator(".language-selector select").selectOption(language);
      for (const width of [320, 339, 390, 650, 768, 901, 1100, 1440]) {
        await page.setViewportSize({ width, height: 1000 });
        await page.evaluate(() => document.fonts.ready);
        for (let tab = 0; tab < 4; tab++) {
          await section.locator(".daily-tabs [role=tab]").nth(tab).click();
          assert(!/\b\d{1,2}:\d{2}:\d{2}\b/.test(await section.innerText()),"Displayed Panchang clocks must omit seconds");
          for(const range of await section.locator('.time-range').all()) {
            assert.equal(await range.locator('.time-range-boundary > small').count(),2);
          }
          const metrics = await section.evaluate((el) => ({
            page: document.documentElement.scrollWidth - innerWidth,
            section: el.scrollWidth - el.clientWidth,
            overflow: [...el.querySelectorAll("input,button,[role=tabpanel]")]
              .filter((n) => {
                const r = n.getBoundingClientRect();
                return r.width && (r.left < -1 || r.right > innerWidth + 1);
              })
              .map((n) => n.outerHTML.slice(0, 100)),
            italics: [...el.querySelectorAll("*")].some(
              (n) => getComputedStyle(n).fontStyle === "italic",
            ),
            clippedData: [
              ...el.querySelectorAll(
                ".daily-periods li,.time-range,.daily-element,.daily-period-group",
              ),
            ]
              .filter((n) => n.scrollWidth > n.clientWidth + 1)
              .map((n) => n.className),
          }));
          assert(
            metrics.page <= 1 &&
              metrics.section <= 1 &&
              !metrics.overflow.length,
            JSON.stringify({ language, width, tab, metrics }),
          );
          assert(!metrics.italics);
          assert.deepEqual(
            metrics.clippedData,
            [],
            JSON.stringify({ language, width, tab, metrics }),
          );
          if (tab === 1) {
            assert.equal(
              await section.locator(".daily-period-track").count(),
              4,
            );
            const escaped = await section
              .locator(".daily-period-track > span")
              .evaluateAll((bars) =>
                bars.some((bar) => {
                  const track = bar.parentElement.getBoundingClientRect();
                  const rect = bar.getBoundingClientRect();
                  return (
                    rect.left < track.left - 1 ||
                    rect.right > track.right + 1 ||
                    rect.width <= 0
                  );
                }),
              );
            assert(!escaped, "Timing bars must stay within the daylight range");
          }
          if (tab === 2 || tab === 3) {
            assert.equal(
              await section
                .locator(".daily-periods li > .astro-symbol svg")
                .count(),
              tab === 2 ? 16 : 24,
            );
            assert.equal(
              await section
                .locator(".daily-periods li .time-range time")
                .count(),
              tab === 2 ? 32 : 48,
            );
            if (width > 650) {
              const alignment = await section.locator('.daily-split').evaluate(el => {
                const groups = [...el.querySelectorAll('.daily-period-group')];
                return groups.map(group => [
                  group.getBoundingClientRect(),
                  group.querySelector('.daily-group-heading').getBoundingClientRect(),
                  ...[...group.querySelectorAll('.daily-periods li')].map(row => row.getBoundingClientRect()),
                ].map(rect => ({top: rect.top, bottom: rect.bottom})));
              });
              for (let row = 0; row < alignment[0].length; row++) {
                assert(Math.abs(alignment[0][row].top - alignment[1][row].top) < 1 &&
                  Math.abs(alignment[0][row].bottom - alignment[1][row].bottom) < 1,
                  JSON.stringify({language, width, tab, row, alignment}));
              }
            }
          }
          if ([339, 1440].includes(width)) {
            await section.screenshot({
              path: `.local/data-${language}-${width}-tab-${tab}.png`,
            });
          }
          if (tab === 2)
            assert.equal(
              await section.locator(".daily-periods li").count(),
              16,
            );
          if (tab === 3)
            assert.equal(
              await section.locator(".daily-periods li").count(),
              24,
            );
          if ([320, 1440].includes(width)) {
            const axe = await new AxeBuilder({ page })
              .include("#panchang")
              .analyze();
            assert.deepEqual(
              axe.violations.map((v) => ({
                id: v.id,
                nodes: v.nodes.map((n) => n.target),
              })),
              [],
              `${language}/${width}/${tab}`,
            );
            audits.push({ language, width, tab });
          }
          layouts.push({ language, width, tab });
        }
      }
      assert.equal(
        await page.locator("#name").inputValue(),
        "Keep my birth details",
      );
      await section.locator(".daily-tabs [role=tab]").first().click();
      await section.screenshot({
        path: `.local/panchang-${language}-desktop.png`,
      });
      await page.setViewportSize({ width: 339, height: 950 });
      await section.screenshot({
        path: `.local/panchang-${language}-mobile.png`,
      });
    }
    await page.locator(".language-selector select").selectOption("en");
    await page.setViewportSize({ width: 390, height: 950 });
    // Tab keyboard behavior.
    await section.locator(".daily-tabs [role=tab]").first().focus();
    await page.keyboard.press("ArrowRight");
    assert.equal(
      await section.locator("[role=tab][aria-selected=true]").innerText(),
      "Daily timings",
    );
    await page.keyboard.press("End");
    assert.equal(
      await section.locator("[role=tab][aria-selected=true]").innerText(),
      "Eclipses",
    );
    await page.keyboard.press("Home");
    // Real nationwide search; select with keyboard and verify city-specific data.
    const city = page.locator("#daily-city");
    await city.fill("Mumbai");
    await page.waitForFunction(
      () => document.querySelectorAll("#daily-places [role=option]").length > 0,
    );
    await city.press("ArrowDown");
    const cityResponse = page.waitForResponse(
      (r) =>
        r.url().includes("/api/panchang?") &&
        r.url().includes("place_id=mumbai") &&
        r.status() === 200,
    );
    await city.press("Enter");
    await cityResponse;
    await section.locator(".daily-limbs").waitFor();
    assert.match(await city.inputValue(), /Mumbai/);
    assert.notEqual(
      await section.locator(".daily-solar b").first().innerText(),
      originalSunrise,
    );
    await city.fill("no-such-city-zzzz");
    await section
      .getByText("No locations found. Try a nearby town.", { exact: true })
      .waitFor();
    await city.press("Escape");
    assert.match(await city.inputValue(), /Mumbai/);
    // Sunrise convention recalculates without changing date or birth input.
    await section.locator(".daily-foot > details > summary").click();
    const oldSunrise = await section
      .locator(".daily-solar b")
      .first()
      .innerText();
    await page.locator("#daily-convention").selectOption("hindu");
    await section.locator(".daily-limbs").waitFor();
    assert.notEqual(
      await section.locator(".daily-solar b").first().innerText(),
      oldSunrise,
    );
    assert.equal(
      await page.locator("#name").inputValue(),
      "Keep my birth details",
    );
    // Delayed old response must not overwrite a newer selection.
    await page.route("**/api/panchang?**", async (route) => {
      if (route.request().url().includes("date=2026-10-19")) {
        const response = await route.fetch();
        await delay(500);
        await route.fulfill({ response });
      } else await route.continue();
    });
    await page.locator("#daily-date").fill("2026-10-19");
    await page.locator("#daily-date").fill("2026-10-20");
    await section.locator(".daily-limbs").waitFor();
    await page.waitForTimeout(650);
    assert.match(
      await section.locator(".daily-date-context strong").innerText(),
      /20 October/,
    );
    await page.unroute("**/api/panchang?**");
    // A failed request provides a working retry, without leaving stale results.
    let fail = true;
    await page.route("**/api/panchang?**", (route) =>
      fail
        ? route.fulfill({
            status: 503,
            contentType: "application/json",
            body: '{"error":"unavailable"}',
          })
        : route.continue(),
    );
    await page.locator("#daily-date").fill("2026-10-21");
    await section.getByRole("alert").waitFor();
    assert.equal(await section.locator(".daily-limbs").count(), 0);
    const errorAxe = await new AxeBuilder({ page })
      .include("#panchang")
      .analyze();
    assert.deepEqual(
      errorAxe.violations.map((v) => v.id),
      [],
      "Error-state accessibility",
    );
    fail = false;
    await section
      .getByRole("button", { name: "Try again", exact: true })
      .click();
    await section.locator(".daily-limbs").waitFor();
    assert.match(
      await section.locator(".daily-date-context strong").innerText(),
      /21 October/,
    );
    assert.deepEqual(errors, []);
    fs.writeFileSync(
      ".local/panchang-browser-results.json",
      JSON.stringify(
        {
          passed: true,
          layouts: layouts.length,
          axeAudits: audits.length + 1,
          keyboard: true,
          cityChanges: true,
          staleResponse: true,
          errorRetry: true,
          preservedBirthDetails: true,
          errors,
        },
        null,
        2,
      ),
    );
    console.log(
      fs.readFileSync(".local/panchang-browser-results.json", "utf8"),
    );
  } finally {
    await browser.close();
  }
})().catch((e) => {
  console.error(e.stack);
  process.exitCode = 1;
});
