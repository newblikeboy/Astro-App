// Synthetic local QA only. No customer/account/provider writes.
const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const fs = require("node:fs");
const assert = require("node:assert/strict");
const base = process.env.KUNDLI_DETAILS_QA_URL || "http://127.0.0.1:18086";
const results = {
  passed: false,
  layouts: [],
  axeAudits: 0,
  checks: [],
  errors: [],
};
(async () => {
  const response = await fetch(base + "/api/kundli", {
    method: "POST",
    headers: { "Content-Type": "application/json", Origin: base },
    body: JSON.stringify({
      name: "Expanded Kundli QA",
      date: "1995-08-15",
      time: "10:30:47",
      place_id: "delhi",
      unknown_time: false,
    }),
  });
  assert(response.ok, await response.clone().text());
  const report = await response.json();
  assert.equal(report.details.vargas.length, 16);
  assert.equal(report.details.ashtakavarga.total, 337);
  assert.equal(report.details.analysis.varga_strength.length, 7);
  fs.writeFileSync(
    ".local/kundli-details-fixture.json",
    JSON.stringify(report, null, 2),
  );
  const browser = await chromium.launch({
    executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe",
    headless: true,
  });
  try {
    const context = await browser.newContext({
      viewport: { width: 1440, height: 1000 },
      reducedMotion: "reduce",
    });
    await context.addInitScript((report) => {
      localStorage.setItem("m4y-language", "en");
      localStorage.setItem("m4y-analytics", "no");
      if (!sessionStorage.getItem("m4y-guest-report")) {
        sessionStorage.setItem("m4y-guest-report", JSON.stringify(report));
        sessionStorage.setItem("m4y-birth", JSON.stringify(report.birth));
      }
    }, report);
    const page = await context.newPage();
    page.on("pageerror", (e) => results.errors.push(e.message));
    let timelineMode = "real",
      timelineCalls = 0;
    let timeline;
    const births = [];
    await page.route("**/api/**", async (route) => {
      const req = route.request(),
        path = new URL(req.url()).pathname;
      if (req.method() === "GET") return route.continue();
      assert(
        ["/api/kundli/dasha", "/api/kundli/saturn"].includes(path),
        "Unexpected write " + path,
      );
      births.push(req.postDataJSON().birth.time);
      if (path === "/api/kundli/saturn") {
        timelineCalls++;
        if (timelineMode === "error")
          return route.fulfill({
            status: 503,
            json: { error: "Synthetic temporary failure" },
          });
        if (timelineMode === "delay")
          await new Promise((r) => setTimeout(r, 500));
        const response = await route.fetch();
        assert(response.ok(), await response.text());
        timeline = await response.json();
        return route.fulfill({ response });
      }
      return route.continue();
    });
    await page.goto(base + "/kundli");
    /** tab opens a Kundli details tab by its stable DOM identifier. */
    const tab = (id) => page.locator("#kundli-tab-" + id).click();
    /** settle waits for queued browser rendering frames before inspecting report geometry. */
    const settle = () =>
      page.evaluate(
        () =>
          new Promise((r) =>
            requestAnimationFrame(() => requestAnimationFrame(r)),
          ),
      );
    /** inspect checks the selected Kundli view across its requested layout/accessibility scenarios. */
    const inspect = async (
      name,
      language,
      { axe = true, shot = false } = {},
    ) => {
      for (const width of [320, 339, 390, 650, 768, 1440]) {
        await page.setViewportSize({ width, height: 1000 });
        await page.evaluate(() => document.fonts.ready);
        await settle();
        assert(
          await page.evaluate(
            () => document.documentElement.scrollWidth <= innerWidth + 1,
          ),
          `${name}/${language}/${width}: page overflow`,
        );
        const clipped = await page.locator("#kundli-panel").evaluate((el) =>
          [
            ...el.querySelectorAll(
              "h2,h3,h4,.birth-record-strip,.birth-limb-grid article,.birth-chakra-grid>div,.planet-condition-grid article,.sarva-grid article,.saturn-period-list article,.yogini-periods>details,.time-range",
            ),
          ]
            .filter((n) => {
              const box = n.getBoundingClientRect();
              return (
                box.width &&
                (box.left < 0 ||
                  box.right > innerWidth + 1 ||
                  n.scrollWidth > n.clientWidth + 1)
              );
            })
            .map((n) => n.className || n.tagName),
        );
        assert.deepEqual(
          clipped,
          [],
          `${name}/${language}/${width}: clipped content`,
        );
        assert(
          !/\b\d{1,2}:\d{2}:\d{2}\b/.test(
            await page.locator("#kundli-panel").innerText(),
          ),
          "Visible clock seconds",
        );
        results.layouts.push({ name, language, width });
        if (shot && [339, 1440].includes(width)) {
          await page.locator("#kundli-panel").scrollIntoViewIfNeeded();
          await page.screenshot({
            path: `.local/kundli-details-${name}-${language}-${width}.png`,
          });
        }
      }
      if (axe) {
        const out = await new AxeBuilder({ page })
          .include(".kundli-workspace")
          .withTags(["wcag2a", "wcag2aa", "wcag21aa"])
          .analyze();
        results.axeAudits++;
        assert.deepEqual(
          out.violations.map((v) => ({
            id: v.id,
            nodes: v.nodes.map((n) => n.target),
          })),
          [],
          `${name}/${language}: Axe`,
        );
      }
    };
    for (const language of ["en", "hi"]) {
      await page.locator(".language-selector select").selectOption(language);
      await tab("birth");
      await page.locator(".birth-limb-grid").waitFor();
      assert.equal(await page.locator(".birth-limb-grid article").count(), 4);
      assert.equal(await page.locator(".birth-chakra-grid>div").count(), 16);
      const limbs = await page
        .locator(".birth-limb-grid time")
        .evaluateAll((nodes) => nodes.map((n) => n.dateTime));
      assert.deepEqual(
        limbs,
        report.details.birth.limbs.flatMap((p) => [p.start, p.end]),
      );
      await inspect("birth", language, { shot: true });
      await tab("charts");
      for (const factor of [7, 10]) {
        await page.locator("#kundli-chart-choice").selectOption("d" + factor);
        await inspect("d" + factor, language, { shot: factor === 10 });
      }
      await page.locator("#kundli-chart-choice").selectOption("chalit");
      for (let i = 0; i < 9; i++) {
        await page.locator(".chart-legend button").nth(i).click();
        const p = report.details.chalit.planets[i];
        assert.equal(await page.locator(".chart-house-selected").count(), 1);
        const text = await page
          .locator(".chart-house-detail strong")
          .innerText();
        assert(text.includes(String(p.house)), text);
      }
      await inspect("chalit", language, { shot: true });
      await tab("planets");
      assert.equal(
        await page.locator(".planet-condition-grid article").count(),
        9,
      );
      assert.equal(await page.locator(".sarva-grid article").count(), 12);
      const sav = await page.locator(".sarva-grid strong").allTextContents();
      assert.deepEqual(sav.map(Number), report.details.ashtakavarga.sarva);
      const meters = await page
        .locator(".varga-strength-list meter")
        .evaluateAll((nodes) => nodes.map((n) => n.value));
      assert.deepEqual(
        meters,
        report.details.analysis.varga_strength.map((p) => p.total),
      );
      await inspect("planets", language, { shot: true });
      // Open every detailed calculation panel, including a contributor table.
      await page
        .locator(".planet-analysis-panel > details")
        .evaluateAll((nodes) => nodes.forEach((n) => (n.open = true)));
      await page
        .locator(".prastara-planet")
        .first()
        .evaluate((n) => (n.open = true));
      await inspect("analysis-tables", language);
      await tab("checks");
      assert.equal(
        await page.locator(".yoga-pattern-grid article").count(),
        11,
      );
      await inspect("patterns", language);
      await tab("dasha");
      await page.locator(".dasha-system-select select").selectOption("yogini");
      assert.equal(await page.locator(".yogini-periods>details").count(), 8);
      assert.equal(
        await page
          .locator(".yogini-periods>details>summary .time-range time")
          .count(),
        16,
      );
      await page
        .locator(".yogini-periods>details")
        .first()
        .locator("summary")
        .click();
      assert.equal(
        await page.locator(".yogini-children article:visible").count(),
        8,
      );
      await inspect("yogini", language, { shot: true });
      await tab("transits");
      const form = page.locator(".saturn-window");
      await form.locator("input").first().fill("2020");
      await form.locator("input").last().fill("2030");
      await form.locator("button").click();
      await page.locator(".saturn-period-list").waitFor();
      const periods = await page
        .locator(".saturn-period-list time")
        .evaluateAll((nodes) => nodes.map((n) => n.dateTime));
      assert.deepEqual(
        periods,
        timeline.periods.flatMap((p) => [p.start, p.end]),
      );
      await inspect("saturn", language, { shot: true });
    }
    await page.locator(".language-selector select").selectOption("en");
    await tab("charts");
    // Every varga's nine table signs and chart selections match native source.
    for (const varga of report.details.vargas) {
      await page
        .locator("#kundli-chart-choice")
        .selectOption("d" + varga.factor);
      assert.equal(await page.locator(".chart-legend button").count(), 9);
      for (let i = 0; i < 9; i++) {
        const p = varga.planets[i];
        await page.locator(".chart-legend button").nth(i).click();
        assert.match(
          await page.locator(".chart-house-detail strong").innerText(),
          new RegExp("^House " + p.house + " "),
        );
      }
    }
    results.checks.push(
      "All sixteen native vargas and Chalit planet/house selections; actual-birth Panchang boundaries; exact SAV and Vimshopaka values; all labelled times omit seconds",
    );
    // Retry a new timeline window, then reuse it from cache.
    await tab("transits");
    const form = page.locator(".saturn-window");
    await form.locator("input").first().fill("2000");
    await form.locator("input").last().fill("2010");
    timelineMode = "error";
    await form.locator("button").click();
    await page.locator(".saturn-timeline [role=alert]").waitFor();
    timelineMode = "real";
    await page
      .locator(".saturn-timeline")
      .getByRole("button", { name: "Try again", exact: true })
      .click();
    await page.locator(".saturn-period-list").waitFor();
    const count = timelineCalls;
    await form.locator("button").click();
    await page.locator(".saturn-period-list").waitFor();
    assert.equal(timelineCalls, count, "Timeline cache missed");
    await form.locator("input").last().fill("2011");
    timelineMode = "delay";
    await form.locator("button").click();
    await tab("overview");
    await page.waitForTimeout(650);
    assert.equal(await page.locator(".saturn-timeline").count(), 0);
    timelineMode = "real";
    // Keep the request context alive until the deliberately cancelled native
    // request has finished; browser teardown must not interrupt route.fetch.
    await page.unrouteAll({ behavior: "wait" });
    await tab("dasha");
    await page.locator(".dasha-system-select select").selectOption("vimshottari");
    await page.locator('.dasha-period-list[data-level="1"]').waitFor();
    assert.equal(await page.locator(".yogini-periods").count(), 0);
    results.checks.push(
      "Saturn retry, bounded in-memory cache and cancellation on leaving the view; source birth seconds retained",
    );
    assert(births.every((time) => time === "10:30:47"));
    const stored = await page.evaluate(() =>
      JSON.parse(sessionStorage.getItem("m4y-guest-report")),
    );
    assert.equal(stored.birth.time, "10:30:47");
    assert.equal(stored.details.method, report.details.method);
    // Older saved reports still display their original core chart and Dasha.
    await page.evaluate((report) => {
      const old = structuredClone(report);
      delete old.details;
      sessionStorage.setItem("m4y-guest-report", JSON.stringify(old));
    }, report);
    await page.reload();
    await tab("birth");
    assert(
      await page
        .getByText(
          "Recreate this Kundli from its birth details to include the new calculations. Your saved report remains unchanged.",
          { exact: true },
        )
        .isVisible(),
    );
    await tab("charts");
    assert.equal(await page.locator(".chart-switch button").count(), 3);
    assert.deepEqual(results.errors, []);
    results.passed = true;
    console.log(
      JSON.stringify(
        {
          passed: true,
          layouts: results.layouts.length,
          axeAudits: results.axeAudits,
          checks: results.checks,
        },
        null,
        2,
      ),
    );
  } finally {
    fs.writeFileSync(
      ".local/kundli-details-browser-results.json",
      JSON.stringify(results, null, 2),
    );
    await browser.close();
  }
})().catch((e) => {
  console.error(e);
  process.exitCode = 1;
});
