const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const fs = require("node:fs");
const assert = require("node:assert/strict");
(async () => {
  const fixture = JSON.parse(
    fs
      .readFileSync(
        process.env.KUNDLI_QA_FIXTURE || ".local/kundli-v2-fixture.json",
        "utf8",
      )
      .replace(/^\uFEFF/, ""),
  );
  fixture.birth.name = "Aarav Sharma";
  let response = fixture;
  const browser = await chromium.launch({
    executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe",
    headless: true,
  });
  const results = [];
  const errors = [];
  let axeAudits = 0;
  try {
    const context = await browser.newContext({
      reducedMotion: "reduce",
      isMobile: true,
      hasTouch: true,
    });
    await context.addInitScript((birth) => {
      localStorage.setItem("m4y-language", "en");
      localStorage.setItem("m4y-analytics", "no");
      sessionStorage.setItem("m4y-birth", JSON.stringify(birth));
    }, fixture.birth);
    const page = await context.newPage();
    /** selectReportSection uses the visible phone selector or desktop tabs to open a report section. */
    async function selectReportSection(id) {
      const selector = page.locator(".kundli-section-select select");
      if (await selector.isVisible()) await selector.selectOption(id);
      else await page.locator("#kundli-tab-" + id).click();
    }
    page.on("pageerror", (e) => errors.push(e.message));
    await page.route("**/api/kundli", (route) =>
      route.fulfill({ json: response }),
    );
    await page.route("**/api/**", (route) => {
      if (
        route.request().method() !== "GET" &&
        !["/api/kundli", "/api/kundli/dasha"].includes(
          new URL(route.request().url()).pathname,
        )
      )
        return route.abort();
      return route.fallback();
    });
    await page.goto(process.env.PANCHANG_QA_URL || "http://localhost:5173");
    await page.locator("#birthplace").waitFor();
    await page.waitForFunction(
      () => document.querySelector("#birthplace").value.length > 0,
    );
    await page.locator(".birth-form button[type=submit]").click();
    await page.locator(".kundli-summary").waitFor();
    for (const lang of ["en", "hi"]) {
      await page.locator(".language-selector select").selectOption(lang);
      for (const width of [320, 339, 390, 650, 768, 1440]) {
        await page.setViewportSize({ width, height: 844 });
        await page.evaluate(() => document.fonts.ready);
        for (const tab of [
          "overview",
          "charts",
          "reading",
          "dasha",
          "checks",
          "transits",
          "settings",
        ]) {
          if (process.env.KUNDLI_QA_PROGRESS === "1")
            console.log(`${lang}/${width}/${tab}`);
          await selectReportSection(tab);
          if (tab === "charts")
            await page.locator(".chart-switch button").nth(1).click();
          if (tab === "reading") {
            await page.locator("#faladesh-topic-career").click();
            assert.equal(
              await page.locator(".faladesh-topic-tabs [role=tab]").count(),
              8,
            );
            assert.equal(
              await page.locator(".faladesh-topic-tabs [role=tab] svg").count(),
              8,
            );
            assert.equal(
              await page.locator(".faladesh-panel select").count(),
              0,
            );
            const clippedTopics = await page
              .locator(".faladesh-topic-tabs button")
              .evaluateAll((buttons) =>
                buttons
                  .filter(
                    (button) => button.scrollWidth > button.clientWidth + 1,
                  )
                  .map((button) => button.id),
              );
            assert.deepEqual(
              clippedTopics,
              [],
              `${lang}/${width}: clipped life-area tabs`,
            );
          }
          const overflow = await page.evaluate(
            () => document.documentElement.scrollWidth > innerWidth,
          );
          assert(!overflow, `${lang}/${width}/${tab}: page overflow`);
          if (tab === "overview" && [339, 1440].includes(width)) {
            await page.locator(".report-chart").screenshot({
              path: `.local/explorer-chart-${lang}-${width}.png`,
            });
            await page.locator(".natal-table").screenshot({
              path: `.local/explorer-planets-${lang}-${width}.png`,
            });
          }
          const escape = await page.locator(".kundli-panel").evaluate((el) =>
            [
              ...el.querySelectorAll(
                "h2,h3,summary,.fact-list,.chart-switch,.dasha-summary,.placement-list article,.reading-status,.report-section-heading,.faladesh-topic-tabs,.faladesh-topic-tabs button,.chart-legend button,.chart-house-detail,.natal-table",
              ),
            ]
              .filter((n) => {
                const r = n.getBoundingClientRect();
                return r.width && (r.left < -0.5 || r.right > innerWidth + 0.5);
              })
              .map((n) => n.className || n.tagName),
          );
          assert.deepEqual(
            escape,
            [],
            `${lang}/${width}/${tab}: content escapes`,
          );
          if (tab === "dasha") {
            await page
              .locator(".dasha-tree > .dasha-period-list > .dasha-item")
              .first()
              .waitFor();
            assert.equal(
              await page
                .locator(".dasha-tree > .dasha-period-list > .dasha-item")
                .count(),
              fixture.dasha.periods.length,
            );
            assert.equal(
              await page.locator(".dasha-current-chain > li").count(),
              5,
            );
          }
          if (tab === "checks")
            assert.equal(
              await page.locator(".placement-list article").count(),
              4,
            );
          if (tab === "transits")
            assert.equal(
              await page.locator(".transit-panel tbody tr").count(),
              9,
            );
          if (lang === "en") {
            const body = await page.locator(".kundli-panel").innerText();
            assert(
              !/[\u0900-\u097f]/.test(body),
              `English text fallback: ${tab}: ${body.match(/[\u0900-\u097f]+/g)}`,
            );
          }
          if ([339, 1440].includes(width)) {
            await page.locator(".kundli-workspace").evaluate((el) =>
              window.scrollTo({
                top: el.offsetTop - 95,
                behavior: "instant",
              }),
            );
            await page.screenshot({
              path: `.local/kundli-v2-${lang}-${width}-${tab}.png`,
            });
          }
          if (
            [339, 1440].includes(width) &&
            ["overview", "dasha", "reading", "charts"].includes(tab)
          ) {
            const audit = await new AxeBuilder({ page })
              .include(".kundli-workspace")
              .withTags(["wcag2a", "wcag2aa", "wcag21aa"])
              .analyze();
            axeAudits++;
            assert.deepEqual(
              audit.violations
                .filter((v) => ["serious", "critical"].includes(v.impact))
                .map((v) => ({
                  id: v.id,
                  nodes: v.nodes.map((n) => n.target),
                })),
              [],
              `${lang}/${width}/${tab}: accessibility`,
            );
          }
          results.push({ lang, width, tab, passed: true });
        }
      }
    }
    // A selected planet must locate its actual house in every chart reference.
    await page.locator(".language-selector select").selectOption("en");
    await page.locator("#kundli-tab-charts").click();
    for (let mode = 0; mode < 3; mode++) {
      await page.locator(".chart-switch button").nth(mode).click();
      const planets =
        mode === 1 ? fixture.navamsa.planets : fixture.chart.planets;
      const moon = fixture.chart.planets.find((p) => p.name === "Moon");
      for (let i = 0; i < planets.length; i++) {
        const p = planets[i];
        const house =
          mode === 2
            ? ((p.rashi_number - moon.rashi_number + 12) % 12) + 1
            : p.house;
        await page.locator(".chart-legend button").nth(i).click();
        assert.equal(
          await page.locator(".chart-legend [aria-pressed=true]").count(),
          1,
        );
        assert.match(
          await page.locator(".chart-house-detail strong").innerText(),
          new RegExp(`^House ${house} `),
        );
        assert(
          (await page.locator(".chart-house-detail").innerText()).includes(
            p.name,
          ),
        );
        assert.equal(await page.locator(".chart-house-selected").count(), 1);
      }
    }
    await page.locator(".language-selector select").selectOption("hi");
    await page.locator("#kundli-tab-reading").click();
    await page.setViewportSize({ width: 390, height: 844 });
    for (const topic of [
      "general",
      "personality",
      "education",
      "career",
      "relationships",
      "family",
      "wellbeing",
      "dasha",
    ]) {
      await page.locator("#faladesh-topic-" + topic).click();
      assert.equal(
        await page.locator(".faladesh-topic-tabs [aria-selected=true]").count(),
        1,
      );
      assert.equal(
        await page
          .locator("#faladesh-topic-panel")
          .getAttribute("aria-labelledby"),
        "faladesh-topic-" + topic,
      );
      assert(
        await page.locator("#faladesh-topic-panel .chart-facts").isVisible(),
      );
    }
    await page.locator("#faladesh-topic-dasha").focus();
    await page.keyboard.press("Home");
    assert.equal(
      await page
        .locator("#faladesh-topic-general")
        .getAttribute("aria-selected"),
      "true",
    );
    await page.keyboard.press("ArrowDown");
    assert.equal(
      await page
        .locator("#faladesh-topic-education")
        .getAttribute("aria-selected"),
      "true",
    );
    await page.keyboard.press("ArrowUp");
    assert.equal(
      await page
        .locator("#faladesh-topic-general")
        .getAttribute("aria-selected"),
      "true",
    );
    await page.keyboard.press("End");
    await page.keyboard.press("ArrowRight");
    assert.equal(
      await page
        .locator("#faladesh-topic-general")
        .getAttribute("aria-selected"),
      "true",
    );
    // Arrow-key tab navigation applies to the visible desktop tab row.
    await page.setViewportSize({ width: 1440, height: 844 });
    await page.locator("#kundli-tab-overview").click();
    await page.locator("#kundli-tab-overview").focus();
    await page.keyboard.press("ArrowRight");
    assert.equal(
      await page.locator("#kundli-tab-birth").getAttribute("aria-selected"),
      "true",
    );
    await page.keyboard.press("ArrowRight");
    assert.equal(
      await page.locator("#kundli-tab-charts").getAttribute("aria-selected"),
      "true",
    );
    await page.keyboard.press("End");
    assert.equal(
      await page.locator("#kundli-tab-settings").getAttribute("aria-selected"),
      "true",
    );
    await page.keyboard.press("Home");
    assert.equal(
      await page.locator("#kundli-tab-overview").getAttribute("aria-selected"),
      "true",
    );
    // Older snapshots calculate Dasha from validated birth inputs without
    // modifying their stored snapshot or inventing unavailable chart fields.
    response = { ...fixture };
    delete response.navamsa;
    delete response.dasha;
    delete response.checks;
    delete response.transits;
    delete response.details;
    await page.goto(process.env.PANCHANG_QA_URL || "http://localhost:5173");
    await page.waitForFunction(
      () => document.querySelector("#birthplace")?.value.length > 0,
    );
    await page.locator(".birth-form button[type=submit]").click();
    await page.locator("#kundli-tab-dasha").click();
    await page.locator(".dasha-current-chain > li").last().waitFor();
    assert.equal(await page.locator(".dasha-current-chain > li").count(), 5);
    assert.equal(
      await page.evaluate(
        () => JSON.parse(sessionStorage.getItem("m4y-guest-report")).dasha,
      ),
      undefined,
    );
    response = {
      ...fixture,
      chart: null,
      rules: [],
      navamsa: undefined,
      dasha: undefined,
      checks: undefined,
      transits: undefined,
      details: undefined,
      birth: { ...fixture.birth, unknown_time: true, time: "" },
    };
    await page.goto(process.env.PANCHANG_QA_URL || "http://localhost:5173");
    await page.waitForFunction(
      () => document.querySelector("#birthplace")?.value.length > 0,
    );
    await page.locator(".birth-form button[type=submit]").click();
    await page.locator(".kundli-panel .empty-state").waitFor();
    assert.equal(
      await page.locator(".kundli-panel svg.kundli-chart").count(),
      0,
    );
    assert(
      await page
        .getByRole("button", { name: "Remedies PDF", exact: true })
        .isDisabled(),
    );
    assert.deepEqual(errors, []);
    fs.writeFileSync(
      process.env.KUNDLI_QA_RESULTS || ".local/kundli-v2-browser-results.json",
      JSON.stringify(
        {
          scenarios: results.length,
          axeAudits,
          engine: fixture.engine,
          results,
          errors,
          keyboard: true,
          legacy: true,
          unknownTime: true,
        },
        null,
        2,
      ),
    );
    console.log(
      JSON.stringify({
        scenarios: results.length,
        axeAudits,
        engine: fixture.engine,
        keyboard: true,
        legacy: true,
        unknownTime: true,
        errors,
        passed: true,
      }),
    );
  } finally {
    await browser.close();
  }
})().catch((e) => {
  console.error(e.stack);
  process.exitCode = 1;
});
