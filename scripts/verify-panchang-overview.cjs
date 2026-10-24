// Read-only UI regression: native API results are read, and all mutations are blocked.
const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const assert = require("node:assert/strict");
const fs = require("node:fs");

/** verify checks progressive disclosure, native interval preservation, retained selections and bilingual responsive layouts. */
async function verify(language) {
  const browser = await chromium.launch({ executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe", headless: true, args: ["--disable-gpu", "--renderer-process-limit=2"] });
  const errors = [], views = [];
  let audits = 0;
  try {
    const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, reducedMotion: "reduce" });
    await context.addInitScript(() => {
      localStorage.setItem("m4y-language", "en");
      localStorage.setItem("m4y-analytics", "no");
      localStorage.removeItem("m4y-panchang-place");
      window.overviewRealNow = Date.now;
      Date.now = () => new Date("2026-10-18T08:00:00Z").getTime();
    });
    const page = await context.newPage();
    page.on("pageerror", error => errors.push(error.message));
    await page.route("**/api/**", route => route.request().method() === "GET" ? route.continue() : route.abort());
    await page.goto(process.env.PANCHANG_QA_URL || "http://localhost:5173");
    const section = page.locator("#panchang");
    await section.locator(".panchang-essentials").waitFor();
    /** loadDay waits for the selected native response and its rendered date rather than accepting stale results. */
    async function loadDay(date) {
      const response = page.waitForResponse(r => r.url().includes(`/api/panchang?date=${date}&`) && r.status() === 200);
      await page.locator("#daily-date").fill(date);
      const data = await (await response).json();
      await page.locator(`.daily-date-context time[datetime="${date}"]`).waitFor();
      return data;
    }
    /** checkValues compares displayed names and precise interval ends with the native response at the pinned instant. */
    async function checkValues(data, current) {
      assert.equal(await section.locator(".panchang-essential").count(), 4);
      assert.equal(await section.locator(".panchang-snapshot-label").innerText(), current ? "Current period" : "At sunrise");
      for (const [index, key] of ["tithi", "nakshatra", "yoga", "karana"].entries()) {
        const instant = Date.parse("2026-10-18T08:00:00Z");
        const expected = (current && data[key].find(p => instant >= Date.parse(p.start) && instant < Date.parse(p.end))) || data[key][0];
        const card = section.locator(".panchang-essential").nth(index);
        assert.equal(await card.locator("h4").innerText(), expected.name);
        assert.equal(await card.locator("time").getAttribute("datetime"), expected.end);
      }
    }
    await checkValues(await loadDay("2026-10-17"), false);
    const data = await loadDay("2026-10-18");
    await checkValues(data, true);
    // Restore the wall clock before accessibility scanners run their own timers.
    await page.evaluate(() => { Date.now = window.overviewRealNow; });
    assert.equal(await section.locator(".panchang-disclosure[open]").count(), 0);
    assert.equal(await section.locator(".daily-limb,.panchang-details,.daily-lunar-facts").count(), 0, "Optional content must mount only when requested");
    const timeline = section.locator(".panchang-timeline-disclosure");
    const extended = section.locator(".panchang-extended-disclosure");
    for (const lang of [language]) {
      await page.locator(".language-selector select").selectOption(lang);
      for (const mode of ["summary", "timeline", "extended"]) {
        console.log(`Checking ${lang}/${mode}`);
        if (mode === "timeline") {
          await timeline.locator(":scope > summary").click();
          await timeline.locator(".daily-limb").first().waitFor();
          const times = await timeline.locator(".time-range time").evaluateAll(nodes => nodes.map(n => n.getAttribute("datetime")));
          assert.deepEqual(times, [...["tithi", "nakshatra", "yoga", "karana"].flatMap(key => data[key].flatMap(p => [p.start, p.end])), data.sunrise, data.next_sunrise]);
        }
        if (mode === "extended") {
          await timeline.locator(":scope > summary").click();
          await extended.locator(":scope > summary").click();
          await page.locator(".panchang-details").waitFor();
          await page.locator("#panchang-detail-yogas").click();
          await page.locator(".balam-controls select").nth(0).selectOption("1");
          await page.locator(".balam-controls select").nth(1).selectOption("2");
          await extended.locator(":scope > summary").click();
          assert.equal(await page.locator(".balam-controls").isVisible(), false);
          await extended.locator(":scope > summary").click();
          assert.equal(await page.locator(".balam-controls select").nth(0).inputValue(), "1");
          assert.equal(await page.locator(".balam-controls select").nth(1).inputValue(), "2");
        }
        for (const width of [320, 390, 650, 651, 768, 901, 1440]) {
          await page.setViewportSize({ width, height: 1000 });
          await page.evaluate(() => document.fonts.ready);
          await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
          const metrics = await section.evaluate(el => ({
            page: document.documentElement.scrollWidth - innerWidth,
            section: el.scrollWidth - el.clientWidth,
            clipped: [...el.querySelectorAll(".panchang-essential h4,.panchang-essential p,.panchang-disclosure-copy,.daily-limb,.daily-controls input")]
              .filter(n => n.getBoundingClientRect().width && n.scrollWidth > n.clientWidth + 1).map(n => n.className || n.tagName),
          }));
          assert(metrics.page <= 1 && metrics.section <= 1 && !metrics.clipped.length, JSON.stringify({ lang, mode, width, metrics }));
          if ([320, 1440].includes(width)) {
            const result = await new AxeBuilder({ page }).include("#panchang").analyze();
            assert.deepEqual(result.violations.map(v => ({ id: v.id, nodes: v.nodes.map(n => n.target) })), [], `${lang}/${mode}/${width}`);
            audits++;
          }
          if ([390, 1440].includes(width)) await section.screenshot({ path: `.local/panchang-refresh-${lang}-${width}-${mode}.png`, style: "header,.dev-strip{visibility:hidden!important}" });
          if (lang === "en" && mode === "summary" && width === 1440) fs.writeFileSync(".local/panchang-height-after.json", JSON.stringify(await section.boundingBox()));
          views.push({ lang, mode, width });
        }
        if (mode === "extended") await extended.locator(":scope > summary").click();
      }
    }
    // Both native disclosures must remain available from the keyboard.
    await timeline.locator(":scope > summary").focus();
    await page.keyboard.press("Enter");
    assert.equal(await timeline.evaluate(el => el.open), true);
    await page.keyboard.press("Space");
    assert.equal(await timeline.evaluate(el => el.open), false);
    assert.deepEqual(errors, []);
    const result = { passed: true, views: views.length, axeAudits: audits, nativeValues: true, allIntervalsPreserved: true, lazyMount: true, retainedBalam: true, keyboard: true, errors };
    console.log(JSON.stringify({ language, ...result }));
    return result;
  } finally { await browser.close(); }
}
/** run releases each language's browser before the next accessibility pass to bound memory on the local workstation. */
async function run() {
  const results = [];
  for (const language of ["en", "hi"]) results.push(await verify(language));
  const combined = { ...results[0], views: results.reduce((n, r) => n + r.views, 0), axeAudits: results.reduce((n, r) => n + r.axeAudits, 0) };
  fs.writeFileSync(".local/panchang-overview-results.json", JSON.stringify(combined, null, 2));
  console.log(JSON.stringify(combined));
}
run().catch(error => { console.error(error.stack); process.exitCode = 1; });
