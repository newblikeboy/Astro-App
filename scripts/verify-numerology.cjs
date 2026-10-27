// Stateless guest calculations and synthetic browser snapshots only; no account, purchase or report writes.
const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const assert = require("node:assert/strict");
const fs = require("node:fs");
const copy = require("../internal/platform/data/numerology-copy.json");
const base = process.env.NUMEROLOGY_QA_URL || "http://127.0.0.1:5173";

/** guest calculates an unsaved date-number fixture with or without a known birth time. */
async function guest(unknown) {
  const response = await fetch(base + "/api/kundli", {
    method: "POST", headers: { "Content-Type": "application/json", Origin: base },
    body: JSON.stringify({ name: "Raja Babu", date: "1995-11-29", time: "12:35", place_id: "geonames-1273491", unknown_time: unknown }),
  });
  assert(response.ok, await response.clone().text());
  const report = await response.json();
  assert.equal(report.numerology.mulank.value, 2);
  assert.equal(report.numerology.bhagyank.value, 1);
  assert.deepEqual(report.numerology.mulank.steps, [11, 2]);
  assert.deepEqual(report.numerology.bhagyank.steps, [37, 10, 1]);
  assert.equal(report.numerology.extended.naamank.number.value, 7);
  assert.deepEqual(report.numerology.extended.naamank.number.steps, [16, 7]);
  assert.deepEqual(report.numerology.extended.lucky_dates, [1, 2, 10, 11, 19, 20, 28, 29]);
  return report;
}

/** verify checks bilingual meanings, arithmetic, mobile layout, unknown-time access and old-snapshot messaging. */
async function verify() {
  const known = await guest(false);
  const unknown = await guest(true);
  assert.equal(unknown.chart, null);
  assert.equal(unknown.details, undefined);
  assert.deepEqual(known.numerology, unknown.numerology);
  const browser = await chromium.launch({ executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe", headless: true });
  const results = { passed: false, layouts: [], states: [], errors: [] };
  try {
    const context = await browser.newContext({ reducedMotion: "reduce" });
    await context.addInitScript((report) => {
      localStorage.setItem("m4y-language", localStorage.getItem("m4y-language") || "en");
      localStorage.setItem("m4y-analytics", "no");
      if (!sessionStorage.getItem("m4y-guest-report")) {
        sessionStorage.setItem("m4y-guest-report", JSON.stringify(report));
        sessionStorage.setItem("m4y-birth", JSON.stringify(report.birth));
      }
    }, known);
    const page = await context.newPage();
    page.on("pageerror", (error) => results.errors.push(error.message));
    await page.route("**/api/**", (route) => {
      assert.equal(route.request().method(), "GET");
      return route.continue();
    });
    /** show renders a local snapshot and selects Numerology using the viewport's visible navigation. */
    async function show(report) {
      await page.goto(base + "/kundli");
      await page.evaluate((value) => sessionStorage.setItem("m4y-guest-report", JSON.stringify(value)), report);
      await page.reload();
      const selector = page.locator(".kundli-section-select select");
      if (await selector.isVisible()) await selector.selectOption("numerology");
      else await page.locator("#kundli-tab-numerology").click();
      await page.locator("#numerology-heading").waitFor();
    }
    for (const language of ["en", "hi"]) {
      await show(known);
      await page.locator(".language-selector select").selectOption(language);
      assert.equal(await page.locator("#numerology-heading").innerText(), copy.Numerology[language]);
      assert.equal(await page.locator(".numerology-panel .natal-yoga-status").innerText(), copy["numerology.summary"][language].replace("{v1}", "2").replace("{v2}", "1") + " " + copy["numerology.name-summary"][language].replace("{v1}", "7"));
      assert.deepEqual(await page.locator(".numerology-value").allTextContents(), ["2", "1", "7"]);
      await page.locator(".numerology-number details").evaluateAll((nodes) => nodes.forEach((node) => { node.open = true; }));
      assert.deepEqual(await page.locator('[data-number-kind="mulank"] li').allTextContents(), ["2 + 9 = 11", "1 + 1 = 2"]);
      assert.deepEqual(await page.locator('[data-number-kind="bhagyank"] li').allTextContents(), ["2 + 9 + 1 + 1 + 1 + 9 + 9 + 5 = 37", "3 + 7 = 10", "1 + 0 = 1"]);
      assert.deepEqual(await page.locator('[data-number-kind="naamank"] li').allTextContents(), ["2 + 1 + 1 + 1 + 2 + 1 + 2 + 6 = 16", "1 + 6 = 7"]);
      assert.equal(await page.locator("[data-numerology-topic]").count(), 9);
      assert.equal(await page.locator(".numerology-facts > div").count(), 11);
      assert.deepEqual(await page.locator(".numerology-yantra td").allTextContents(), ["7", "2", "9", "8", "6", "4", "3", "10", "5"]);
      if (language === "en") assert(!/[\u0900-\u097f]/u.test(await page.locator(".numerology-panel").innerText()));
      for (const width of [320, 390, 768, 1440]) {
        await page.setViewportSize({ width, height: 1000 });
        await page.evaluate(() => document.fonts.ready);
        const overflow = await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1 ? [] : Array.from(document.querySelectorAll(".numerology-panel *")).filter((node) => node.getBoundingClientRect().right > innerWidth + 1).map((node) => ({ tag: node.tagName, class: node.className, right: node.getBoundingClientRect().right, text: node.textContent.slice(0, 90) })));
        assert.deepEqual(overflow, [], `${language}/${width}: horizontal overflow`);
        const audit = await new AxeBuilder({ page }).include(".numerology-panel").analyze();
        assert.deepEqual(audit.violations.map((item) => ({ id: item.id, impact: item.impact })), []);
        if (width === 390 || width === 1440) await page.locator(".numerology-panel").screenshot({ path: `.local/numerology-${language}-${width}.png` });
        results.layouts.push(`${language}/${width}`);
      }
      await show(unknown);
      assert(await page.locator(".numerology-panel .notice").isVisible());
      assert.deepEqual(await page.locator(".numerology-value").allTextContents(), ["2", "1", "7"]);
      results.states.push(`${language}/unknown-time`);
      const unnamed = structuredClone(known);
      delete unnamed.numerology.extended.naamank;
      unnamed.numerology.extended.name_status = "needs_latin_spelling";
      await show(unnamed);
      assert.equal(await page.locator('[data-number-kind="naamank"] .notice').innerText(), copy["numerology.name-missing"][language]);
      assert.deepEqual(await page.locator(".numerology-value").allTextContents(), ["2", "1"]);
      results.states.push(`${language}/missing-name-spelling`);
      const dateOnly = structuredClone(known);
      delete dateOnly.numerology.extended;
      await show(dateOnly);
      assert.equal(await page.locator(".numerology-panel .notice").innerText(), copy["numerology.extended-missing"][language]);
      assert.equal(await page.locator("[data-numerology-topic]").count(), 0);
      results.states.push(`${language}/date-only-snapshot`);
      const old = structuredClone(known);
      delete old.numerology;
      await show(old);
      assert.equal(await page.locator(".numerology-panel .notice").innerText(), copy["numerology.missing"][language]);
      assert.equal(await page.locator(".numerology-value").count(), 0);
      results.states.push(`${language}/older-report`);
    }
    const formPage = await browser.newPage();
    await formPage.goto(base + "/#kundli");
    await formPage.locator("#name").waitFor();
    assert.equal(await formPage.locator("#numerology-name").count(), 0);
    assert.equal(await formPage.locator('label[for="numerology-name"]').count(), 0);
    results.states.push("single-main-name-field");
    await formPage.close();
    assert.deepEqual(results.errors, []);
    results.passed = true;
    fs.writeFileSync(".local/numerology-browser-results.json", JSON.stringify(results, null, 2));
    console.log(JSON.stringify(results));
  } finally { await browser.close(); }
}

verify().catch((error) => { console.error(error); process.exitCode = 1; });
