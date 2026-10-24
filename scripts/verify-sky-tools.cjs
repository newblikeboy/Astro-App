// Read-only local browser QA. Customer/provider writes are always blocked.
const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const assert = require("node:assert/strict");
const fs = require("node:fs");
const { setTimeout: delay } = require("node:timers/promises");
(async () => {
  const browser = await chromium.launch({
    executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe",
    headless: true,
    args: ["--disable-gpu", "--renderer-process-limit=2"],
  });
  const results = [],
    errors = [];
  let audits = 0;
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
    await page.route("**/api/**", (r) =>
      r.request().method() === "GET" ? r.continue() : r.abort(),
    );
    await page.goto(process.env.PANCHANG_QA_URL || "http://localhost:5173");
    await page.locator(".daily-limbs").waitFor();
    await page.locator("#daily-date").fill("2026-10-18");
    await page.locator("#name").fill("Preserve birth details");
    const section = page.locator("#panchang");
    /** auditView checks visible sky controls, horizontal tab semantics and result layout for the selected tool. */
    async function auditView(view) {
      console.log(`Checking ${view}`);
      assert.equal(
        await section.locator(".daily-tabs").getAttribute("aria-orientation"),
        "horizontal",
      );
      assert.equal(
        await section
          .locator(".daily-sheet #daily-panel .sky-controls")
          .count(),
        1,
      );
      assert.equal(
        await section.locator(".container > .daily-controls").count(),
        0,
      );
      const positions = view === "positions";
      const eclipse = view.startsWith("eclipse");
      assert.equal(
        await section.locator("#daily-date").count(),
        positions ? 1 : 0,
      );
      assert.equal(
        await section.locator("#sky-clock").count(),
        positions ? 1 : 0,
      );
      assert.equal(
        await section.locator("#daily-city").count(),
        eclipse ? 1 : 0,
      );
      assert.equal(
        await section.locator("#sky-year").count(),
        positions || eclipse ? 0 : 1,
      );
      for (const language of ["en", "hi"]) {
        await page.locator(".language-selector select").selectOption(language);
        for (const width of [320, 339, 390, 650, 768, 901, 1100, 1440]) {
          await page.setViewportSize({ width, height: 1000 });
          await page.evaluate(() => document.fonts.ready);
          // Language + viewport changes can land before the browser's resize paint.
          await page.evaluate(
            () =>
              new Promise((resolve) =>
                { setTimeout(resolve, 500); requestAnimationFrame(() => requestAnimationFrame(resolve)); },
              ),
          );
          const state = await section.evaluate((el) => ({
            page: document.documentElement.scrollWidth - innerWidth,
            section: el.scrollWidth - el.clientWidth,
            bad: [...el.querySelectorAll("button,input,select,table,article")]
              .filter((n) => {
                const r = n.getBoundingClientRect();
                return r.width && (r.left < -1 || r.right > innerWidth + 1);
              })
              .map((n) => n.outerHTML.slice(0, 100)),
            italic: [...el.querySelectorAll("*")].some(
              (n) => getComputedStyle(n).fontStyle === "italic",
            ),
            clippedData: [
              ...el.querySelectorAll(
                ".sky-transits li,.eclipse-neighbour,.sky-positions .planet-identity,.sky-switch button",
              ),
            ]
              .filter((n) => n.scrollWidth > n.clientWidth + 1)
              .map((n) => n.className),
          }));
          assert(
            state.page <= 1 &&
              state.section <= 1 &&
              !state.bad.length &&
              !state.italic,
            JSON.stringify({ view, language, width, state }),
          );
          assert.deepEqual(
            state.clippedData,
            [],
            JSON.stringify({ view, language, width, state }),
          );
          if ([320, 1440].includes(width)) {
            const axe = await new AxeBuilder({ page }).setLegacyMode()
              .include("#panchang")
              .analyze();
            assert.deepEqual(
              axe.violations.map((v) => ({
                id: v.id,
                targets: v.nodes.map((n) => n.target),
              })),
              [],
              JSON.stringify({ view, language, width }),
            );
            audits++;
          }
          assert.equal(
            await page.locator("#name").inputValue(),
            "Preserve birth details",
          );
          results.push({ view, language, width });
          if (
            width === 339 &&
            language === "en" &&
            ["positions", "combustion", "retrograde", "eclipse-lunar"].includes(view)
          )
            await section.screenshot({ path: `.local/sky-${view}-mobile.png` });
          if (width === 1440 && language === "en")
            await section.screenshot({
              path: `.local/sky-${view}-desktop.png`,
            });
        }
      }
      await page.locator(".language-selector select").selectOption("en");
    }
    await section.locator(".daily-tabs [role=tab]").nth(4).click();
    await page.locator(".sky-positions tbody tr").first().waitFor();
    await page.locator("#sky-clock").fill("12:00");
    await page.locator(".sky-positions tbody tr").first().waitFor();
    assert.equal(await page.locator(".sky-positions tbody tr").count(), 9);
    assert.equal(
      await page
        .locator(".sky-positions tbody tr")
        .filter({ hasText: "Rahu" })
        .count(),
      1,
    );
    await auditView("positions");
    await page
      .getByRole("button", { name: "Planet transits", exact: true })
      .click();
    await page.locator(".sky-transits li").first().waitFor();
    assert.equal(await page.locator(".sky-transits li").count(), 12);
    await auditView("transits");
    await page.locator("#sky-planet").selectOption("Mercury");
    assert(
      await page
        .locator(".sky-transits")
        .innerText()
        .then((s) => s.includes("Retrograde begins")),
    );
    await page.locator("#sky-planet").selectOption("All planets");
    assert.equal(await page.locator(".sky-transits li").count(), 20);
    await page
      .getByRole("button", { name: "Show more events", exact: true })
      .click();
    assert.equal(await page.locator(".sky-transits li").count(), 40);
    await page.getByRole("button", { name: "Planet combustion", exact: true }).click();
    await page.locator(".sky-combustion-evidence").first().waitFor();
    assert.equal(await page.locator(".sky-transits li").count(), 20);
    assert.deepEqual(await page.locator("#sky-planet option").evaluateAll(options => options.map(option => option.value)),
      ["All planets", "Moon", "Mars", "Mercury", "Jupiter", "Venus", "Saturn"]);
    await page.getByRole("button", { name: "Show more events", exact: true }).click();
    assert.equal(await page.locator(".sky-transits li").count(), 40);
    await page.locator("#sky-planet").selectOption("Jupiter");
    assert.equal(await page.locator(".sky-transits li").count(), 2);
    const combustion = await page.locator(".sky-transits").innerText();
    assert(combustion.includes("Combustion begins") && combustion.includes("Combustion ends") && combustion.includes("11°"));
    await page.locator(".sky-method summary").click();
    await auditView("combustion");
    await page.getByRole("button", { name: "Planet retrograde", exact: true }).click();
    await page.locator(".sky-transits li").first().waitFor();
    assert.deepEqual(await page.locator("#sky-planet option").evaluateAll(options => options.map(option => option.value)),
      ["All planets", "Mars", "Mercury", "Jupiter", "Venus", "Saturn"]);
    const native = await (await context.request.get((process.env.PANCHANG_QA_URL || "http://localhost:5173") + "/api/sky/transits?year=2026&place_id=delhi")).json();
    const stations = native.events.filter(event => ["Retrograde begins", "Direct motion begins"].includes(event.kind));
    assert.equal(await page.locator(".sky-transits li").count(), Math.min(stations.length, 20));
    assert(!(await page.locator(".sky-transits").innerText()).includes("Sign ingress"));
    await page.locator("#sky-planet").selectOption("Mercury");
    assert.equal(await page.locator(".sky-transits li").count(), stations.filter(event => event.planet === "Mercury").length);
    const stationText = await page.locator(".sky-transits").innerText();
    assert(stationText.includes("Retrograde begins") && stationText.includes("Direct motion begins"));
    await page.locator("#sky-planet").selectOption("Mars");
    assert.equal(stations.filter(event => event.planet === "Mars").length, 0);
    await page.locator(".sky-empty").waitFor();
    await page.locator("#sky-planet").selectOption("All planets");
    await auditView("retrograde");
    let failCombustion = true;
    await page.route("**/api/sky/combustion?*", route => failCombustion
      ? route.fulfill({ status: 503, json: { error: "Fictional QA failure" } }) : route.continue());
    await page.getByRole("button", { name: "Planet combustion", exact: true }).click();
    await page.locator('.sky-tools [role="alert"]').waitFor();
    assert.equal(await page.locator(".sky-transits li").count(), 0);
    failCombustion = false;
    await page.getByRole("button", { name: "Try again", exact: true }).click();
    await page.locator(".sky-combustion-evidence").first().waitFor();
    assert.equal(await page.locator("#sky-planet").inputValue(), "Jupiter");
    await page.getByRole("button", { name: "Planet transits", exact: true }).click();
    await page.locator(".sky-transits li").first().waitFor();
    assert.equal(await page.locator("#sky-planet").inputValue(), "All planets");
    assert.equal(await page.locator(".sky-transits li").count(), 20);
    await page.getByRole("button", { name: "Show more events", exact: true }).click();
    await section.locator(".daily-tabs [role=tab]").nth(5).click();
    await page.locator(".eclipse-neighbour").first().waitFor();
    assert.equal(await page.locator(".eclipse-neighbour").count(), 4);
    assert.equal(await page.locator("#sky-year,.sky-visible,.sky-eclipse-details").count(), 0);
    await auditView("eclipse-summary");
    await section.locator(".daily-tabs [role=tab]").nth(4).click();
    await page.locator(".sky-transits li").first().waitFor();
    assert.equal(await page.locator("#sky-planet").inputValue(), "All planets");
    assert.equal(await page.locator(".sky-transits li").count(), 40);

    assert.deepEqual(errors, []);
    const report = {
      passed: true,
      views: results.length,
      axeAudits: audits,
      positions: 9,
      transits: true,
      retrograde: true,
      combustion: true,
      dedicatedStationFilter: true,
      failureRetry: true,
      pagination: true,
      previousAndNextEclipses: true,
      errors,
    };
    fs.writeFileSync(
      ".local/sky-browser-results.json",
      JSON.stringify(report, null, 2),
    );
    console.log(JSON.stringify(report));
  } finally {
    await browser.close();
  }
})().catch((e) => {
  console.error(e.stack);
  process.exitCode = 1;
});
