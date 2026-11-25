const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const assert = require("node:assert/strict");
const fs = require("node:fs");
const source = JSON.parse(fs.readFileSync("internal/swisseph/data/guruji-mangal-parihar.json", "utf8"));
const config = JSON.parse(fs.readFileSync("internal/swisseph/data/guruji-mangal-rules.json", "utf8"));
const wording = JSON.parse(fs.readFileSync("internal/platform/data/yoga-conclusions.json", "utf8"));

/** verify checks the live stateless API and both production Mangal views without saving birth details or creating orders. */
async function verify() {
  const browser = await chromium.launch({ executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe", headless: true });
  const errors = [];
  let views = 0;
  const created = [];
  try {
    const context = await browser.newContext({ viewport: { width: 1440, height: 1100 } });
    const page = await context.newPage();
    await page.addInitScript(() => { localStorage.setItem("m4y-language", "hi"); localStorage.setItem("m4y-analytics", "no"); });
    page.on("pageerror", (error) => errors.push(error.message));
    await page.route("**/api/**", (route) => route.request().method() === "GET" || new URL(route.request().url()).pathname === "/api/kundli/match" ? route.continue() : route.abort());
    await page.goto("http://localhost:5173/kundli-matching");
    for (const role of ["bride", "groom"]) {
      await page.locator(`#matching-${role}-name`).fill(`${role} Parihar QA`);
      await page.locator(`#matching-${role}-date`).fill(role === "bride" ? "1995-08-15" : "1992-02-29");
      await page.locator(`#matching-${role}-time`).fill(role === "bride" ? "10:30" : "18:15");
      const card = page.locator(".matching-birth-card").nth(role === "bride" ? 0 : 1);
      await card.getByRole("combobox").fill("New Delhi");
      await card.getByRole("option").first().click();
    }
    const wait = page.waitForResponse((response) => new URL(response.url()).pathname === "/api/kundli/match" && response.request().method() === "POST");
    await page.locator(".matching-submit").click();
    const response = await wait;
    assert.ok(response.ok(), await response.text());
    const result = await response.json();
    assert.equal(result.mangal.method, config.method);
    for (const role of ["bride", "groom"]) {
      const assessment = result.mangal[role];
      assert.deepEqual(assessment.references.map((reference) => reference.reference), ["lagna", "moon"]);
      assert.equal(assessment.base_classification, assessment.references[0].flagged ? "mangal" : assessment.references[1].flagged ? "partial" : "none");
      assert.equal(assessment.active_count, assessment.references.filter((reference) => reference.flagged && !reference.cancelled).length);
      for (const reference of assessment.references) {
        assert.equal(reference.flagged, [1, 4, 7, 8, 12].includes(reference.house));
        if (reference.cancelled) assert.ok(reference.exceptions.every((code) => assessment.rules.some((rule) => rule.code === code && rule.effect === "cancellation" && rule.evaluation === "met" && (rule.reference === "all" || rule.reference === reference.reference))));
      }
      for (const rule of assessment.rules) {
        const original = source.rules[rule.source_rule_number - 1];
        assert.equal(rule.heading, original.heading);
        assert.equal(rule.text, original.body);
      }
    }
    await page.locator("#matching-result-tab-mangal").click();
    await page.locator(".matching-mangal").waitFor();
    assert.deepEqual(await page.locator(".matching-mangal-people > article > .natal-yoga-status").allTextContents(), ["bride", "groom"].map((role) => wording[`mangal.guruji.${result.mangal[role].classification}`].hi));
    fs.writeFileSync(".local/mangal-parihar-live-fixture.json", JSON.stringify(result));
    for (const [original, target] of [["scripts/fixtures/mangal-parihar-qa.html", "web/.mangal-parihar-qa.html"], ["scripts/fixtures/mangal-parihar-qa.tsx", "web/src/.mangal-parihar-qa.tsx"]]) {
      fs.copyFileSync(original, target, fs.constants.COPYFILE_EXCL);
      created.push(target);
    }
    await page.goto("http://localhost:5173/.mangal-parihar-qa.html");
    await page.locator("#qa-hi").waitFor();
    for (const state of ["mangal", "partial", "cancelled", "none"]) {
      for (const language of ["hi", "en"]) {
        await page.locator(`#qa-state-${state}`).click();
        await page.locator(`#qa-${language}`).click();
        try {
          await page.waitForFunction((expected) => document.querySelector("#parihar-qa .matching-mangal-people .natal-yoga-status")?.textContent === expected, wording[`mangal.guruji.${state}`][language]);
        } catch (error) {
          console.error(JSON.stringify({ state, language, errors, statuses: await page.locator("#parihar-qa .natal-yoga-status").allTextContents() }));
          throw error;
        }
        assert.equal(await page.locator(".kundli-mangal-assessment .natal-yoga-status").textContent(), wording[`mangal.guruji.${state}`][language]);
        assert.equal(await page.locator("#kaal-sarp-heading").count(), 1, "Kaal Sarp must be shown even when absent");
        assert.equal(await page.locator("#rajyog-heading").count(), 0, "Absent Rajyog must be hidden");
        assert.equal(await page.locator(".yoga-pattern-grid article").count(), 1, "Only present additional patterns should appear");
        assert.equal(await page.locator(".pattern-absent").count(), 0);
        if (state !== "none") {
          const evidence = page.locator(".matching-mangal-people > article").first();
          const texts = await evidence.locator(".mangal-evidence-source").allTextContents();
          assert.deepEqual(texts, [source.rules[state === "cancelled" ? 23 : 2].body, source.rules[14].body]);
          assert.ok(await evidence.locator("[data-effect=review]").count());
          if (state === "cancelled") assert.equal(await evidence.locator(".mangal-evidence-basis p").textContent(), source.rules[23].body.split("\n").map((line) => line.trim()).find((line) => line.startsWith("2. ")));
        }
        for (const width of [320, 1440]) {
          await page.setViewportSize({ width, height: 1100 });
          await page.evaluate(() => document.fonts.ready);
          assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `${state}/${language}/${width} overflow`);
          views++;
          if (state === "cancelled" && language === "hi") await page.screenshot({ path: `.local/mangal-parihar-${width}.png`, fullPage: true });
        }
        if (state === "cancelled") {
          const audit = await new AxeBuilder({ page }).include("#parihar-qa").analyze();
          assert.deepEqual(audit.violations.filter((violation) => ["critical", "serious"].includes(violation.impact)).map((violation) => violation.id), []);
        }
      }
    }
    await page.locator("#qa-natal-present").click();
    assert.equal(await page.locator("#kaal-sarp-heading").count(), 1);
    assert.equal(await page.locator("#rajyog-heading").count(), 1);
    await page.locator("#qa-natal-absent").click();
    assert.equal(await page.locator("#kaal-sarp-heading").count(), 1);
    assert.equal(await page.locator("#rajyog-heading").count(), 0);
    await page.goto("http://localhost:5173/#kundli");
    const birthplace = page.locator("#birthplace");
    for (const [query, keyboard] of [["New Delhi", false], ["Mumbai", true]]) {
      await birthplace.fill(query);
      const option = page.locator("#place-options button").first();
      await option.waitFor();
      const selectedLookup = page.waitForResponse((response) => /\/api\/places\//.test(new URL(response.url()).pathname));
      if (keyboard) { await birthplace.press("ArrowDown"); await page.keyboard.press("Enter"); }
      else await option.click();
      await selectedLookup;
      assert.equal(await birthplace.getAttribute("aria-expanded"), "false");
      assert.equal(await page.locator("#place-options").count(), 0, "Choosing a location must close the recommendations");
      assert.ok((await birthplace.inputValue()).includes(query));
      await page.locator("#name").focus();
      await birthplace.focus();
      assert.equal(await birthplace.getAttribute("aria-expanded"), "false", "Refocusing a selected location must not reopen suggestions");
    }
    await birthplace.fill("Pune");
    await page.locator("#place-options button").first().waitFor();
    await birthplace.press("Escape");
    assert.equal(await page.locator("#place-options").count(), 0);
    await birthplace.focus();
    await birthplace.fill("Jaipur");
    await page.locator("#place-options button").first().waitFor();
    await page.locator("#name").focus();
    assert.equal(await page.locator("#place-options").count(), 0, "Leaving the field must close suggestions");
    assert.deepEqual(errors, []);
    fs.writeFileSync(".local/mangal-parihar-live-result.json", JSON.stringify(result, null, 2));
    console.log(JSON.stringify({ views, liveMethod: result.mangal.method, sourceTextUnchanged: true, cancelledLabel: true, absentYogasHidden: true, birthplaceMouseAndKeyboardClose: true, seriousAccessibilityViolations: 0 }));
  } finally { await browser.close(); for (const target of created) fs.unlinkSync(target); }
}

verify().catch((error) => { console.error(error); process.exitCode = 1; });
