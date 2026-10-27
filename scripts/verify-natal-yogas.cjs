// Read-only guest calculations and synthetic browser sessions; no account, order or provider writes.
const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const assert = require("node:assert/strict");
const fs = require("node:fs");
const conclusions = require("../internal/platform/data/yoga-conclusions.json");
const base = process.env.KUNDLI_YOGA_QA_URL || "http://127.0.0.1:5173";

/** guest calculates a synthetic, unsaved report using the site's public calculation endpoint. */
async function guest(unknown = false) {
  const response = await fetch(base + "/api/kundli", {
    method: "POST",
    headers: { "Content-Type": "application/json", Origin: base },
    body: JSON.stringify({ name: "Natal Yoga QA", date: "1995-08-15", time: "10:30", place_id: "delhi", unknown_time: unknown }),
  });
  assert(response.ok, await response.clone().text());
  return response.json();
}

/** verify checks real calculation rendering in both languages, responsive evidence and older-snapshot messaging. */
async function verify() {
  const report = await guest();
  assert.equal(report.details.yoga_checks.kaal_sarp.planets.length, 7);
  assert(report.details.yoga_checks.rajyog.pairs.length > 0);
  assert.equal((await guest(true)).details, undefined);
  const browser = await chromium.launch({ executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe", headless: true });
  const results = { passed: false, layouts: [], conclusionStates: [], errors: [] };
  try {
    const context = await browser.newContext({ reducedMotion: "reduce" });
    await context.addInitScript((report) => {
      localStorage.setItem("m4y-language", localStorage.getItem("m4y-language") || "en");
      localStorage.setItem("m4y-analytics", "no");
      if (!sessionStorage.getItem("m4y-guest-report")) {
        sessionStorage.setItem("m4y-guest-report", JSON.stringify(report));
        sessionStorage.setItem("m4y-birth", JSON.stringify(report.birth));
      }
    }, report);
    const page = await context.newPage();
    page.on("pageerror", (error) => results.errors.push(error.message));
    // Browser actions must not write accounts, reports or orders.
    await page.route("**/api/**", (route) => {
      assert.equal(route.request().method(), "GET");
      return route.continue();
    });
    await page.goto(base + "/kundli");
    await page.locator("#kundli-tab-checks").click();
    /** showSnapshot renders synthetic saved evidence locally without writing an account or report. */
    async function showSnapshot(snapshot) {
      await page.evaluate((value) => sessionStorage.setItem("m4y-guest-report", JSON.stringify(value)), snapshot);
      await page.reload();
      await page.locator("#kundli-tab-checks").click();
    }
    for (const language of ["en", "hi"]) {
      await page.locator(".language-selector select").selectOption(language);
      assert.equal(await page.locator("#kaal-sarp-heading").innerText(), language === "en" ? "Kaal Sarp Dosha" : "कालसर्प दोष");
      assert.equal(await page.locator("#rajyog-heading").innerText(), language === "en" ? "Rajyog · Kendra–Trikona connections" : "राजयोग · केन्द्र–त्रिकोण सम्बन्ध");
      assert(await page.locator(".kundli-mangal-assessment .yoga-conclusion").isVisible());
      assert.equal(await page.locator('[aria-labelledby="kaal-sarp-heading"] details[open]').count(), 0);
      await page.setViewportSize({ width: 390, height: 1000 });
      await page.locator('[aria-labelledby="kaal-sarp-heading"]').screenshot({ path: `.local/yoga-conclusion-${language}.png` });
      await page.locator(".natal-yoga-checks details").evaluateAll((nodes) => nodes.forEach((node) => { node.open = true; }));
      assert.equal(await page.locator('[aria-labelledby="kaal-sarp-heading"] article').count(), 7);
      for (const width of [320, 390, 768, 1440]) {
        await page.setViewportSize({ width, height: 1000 });
        await page.evaluate(() => document.fonts.ready);
        assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `${language}/${width}: page overflow`);
        const audit = await new AxeBuilder({ page }).include(".natal-yoga-checks").analyze();
        assert.deepEqual(audit.violations.map((v) => ({ id: v.id, impact: v.impact })), []);
        if (width === 1440 && language === "en" || width === 390 && language === "hi") {
          await page.locator(".natal-yoga-checks").screenshot({ path: `.local/natal-yogas-${language}-${width}.png` });
        }
        results.layouts.push({ language, width });
      }
      for (let house = 1; house <= 12; house++) {
        const snapshot = structuredClone(report);
        Object.assign(snapshot.details.yoga_checks.kaal_sarp, { status: "present", rahu_house: house, ketu_house: (house + 5) % 12 + 1 });
        delete snapshot.details.yoga_checks.kaal_sarp.type;
        await showSnapshot(snapshot);
        const name = conclusions[`kaal-sarp.type.${house}`][language];
        assert.equal(await page.locator('[aria-labelledby="kaal-sarp-heading"] .natal-yoga-status').innerText(), conclusions["kaal-sarp.present"][language].replace("{v1}", name));
        assert.equal(await page.locator('[aria-labelledby="kaal-sarp-heading"] tbody tr').count(), 12);
        assert.equal((await page.locator('[aria-labelledby="kaal-sarp-heading"] tr[aria-current="true"] th').textContent()).trim(), name);
        if (house === 9) await page.locator('[aria-labelledby="kaal-sarp-heading"]').screenshot({ path: `.local/yoga-shankhchud-${language}.png` });
        results.conclusionStates.push(`${language}/type-${house}`);
      }
      for (const [index, status] of ["present", "absent", "boundary"].entries()) {
        const snapshot = structuredClone(report);
        Object.assign(snapshot.details.yoga_checks.kaal_sarp, { status, rahu_house: 9, ketu_house: 3 });
        const mangalState = ["present", "absent", "cancelled"][index];
        Object.assign(snapshot.details.mangal, { base_count: index === 1 ? 0 : 2, active_count: index === 0 ? 1 : 0 });
        snapshot.details.yoga_checks.rajyog.matches = index === 0;
        await showSnapshot(snapshot);
        const kaalText = await page.locator('[aria-labelledby="kaal-sarp-heading"] .yoga-conclusion').innerText();
        assert(kaalText.includes(conclusions[status === "present" ? "kaal-sarp.meaning" : `kaal-sarp.${status}-meaning`][language]));
        assert.equal(await page.locator('[aria-labelledby="kaal-sarp-heading"] tr[aria-current="true"]').count(), status === "present" ? 1 : 0);
        assert.equal(await page.locator(".kundli-mangal-assessment .natal-yoga-status").innerText(), conclusions[`mangal.${mangalState}`][language]);
        assert.equal(await page.locator('[aria-labelledby="rajyog-heading"] .natal-yoga-status').innerText(), conclusions[index === 0 ? "rajyog.present" : "rajyog.absent"][language]);
        assert(!/kaal-sarp\.|yoga\.(?:name|meaning)|placement\.|\{v\d+\}/.test(await page.locator(".placement-panel").innerText()));
        results.conclusionStates.push(`${language}/${status}/${mangalState}`);
      }
    }
    const older = structuredClone(report);
    delete older.details.yoga_checks;
    await page.evaluate((older) => sessionStorage.setItem("m4y-guest-report", JSON.stringify(older)), older);
    await page.reload();
    await page.locator("#kundli-tab-checks").click();
    assert(await page.locator(".natal-yoga-checks .notice").isVisible());
    assert.equal(await page.locator("#kaal-sarp-heading").count(), 0);
    assert.deepEqual(results.errors, []);
    results.passed = true;
    fs.writeFileSync(".local/natal-yogas-browser-results.json", JSON.stringify(results, null, 2));
    console.log(JSON.stringify(results));
  } finally { await browser.close(); }
}

verify().catch((error) => { console.error(error); process.exitCode = 1; });
