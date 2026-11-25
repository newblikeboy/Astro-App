// Read-only local QA: all customer/provider mutations are blocked and account state is a browser fixture.
const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const { setTimeout: delay } = require("node:timers/promises");
const guides = JSON.parse(fs.readFileSync(path.join(__dirname, "../web/src/data/puja-guides.json"), "utf8"));
const mantras = JSON.parse(fs.readFileSync(path.join(__dirname, "../web/src/data/puja-mantras.json"), "utf8"));

/** verifyLanguage checks all guide bodies and URL flows plus responsive/accessibility states in a fresh language-specific browser. */
async function verifyLanguage(language) {
  const browser = await chromium.launch({ executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe", headless: true, args: ["--disable-gpu", "--renderer-process-limit=2", "--js-flags=--max-old-space-size=128"] });
  const errors = [], layouts = [];
  let audits = 0, bodyRequests = 0, mantraRequests = 0;
  try {
    const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, reducedMotion: "reduce" });
    await context.addInitScript(lang => { localStorage.setItem("m4y-language", lang); localStorage.setItem("m4y-analytics", "no"); }, language);
    const page = await context.newPage();
    page.on("pageerror", error => errors.push(error.message));
    page.on("request", request => { if (request.resourceType() === "fetch" && request.url().includes("puja-guides")) bodyRequests++; });
    page.on("request", request => { if (request.resourceType() === "fetch" && request.url().includes("puja-mantras")) mantraRequests++; });
    await page.route("**/api/**", route => route.request().method() === "GET" ? route.continue() : route.abort());
    const base = process.env.DESIGN_QA_URL || "http://localhost:5173";
    assert(["localhost", "127.0.0.1"].includes(new URL(base).hostname));
    await page.goto(base + "/puja-vidhi");
    await page.locator(".puja-card").first().waitFor();
    assert.equal(bodyRequests, 0, "The catalogue must not fetch full guide bodies");
    assert.equal(mantraRequests, 0, "The catalogue must not fetch mantra bodies");
    assert.equal(await page.locator(".puja-card").count(), 12);
    assert.equal(await page.locator(".puja-card img[loading=lazy][decoding=async]").count(), 12);
    assert.equal(await page.locator('meta[name="robots"][content*="noindex"]').count(), 0);

    /** audit checks page overflow, one continuous reader surface, meaningful image loads and scoped WCAG rules at representative widths. */
    async function audit(view) {
      for (const width of [320, 390, 650, 768, 1281, 1440]) {
        await page.setViewportSize({ width, height: 1000 });
        await page.evaluate(() => document.fonts.ready);
        await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
        const metrics = await page.locator("#main").evaluate(el => ({
          page: document.documentElement.scrollWidth - innerWidth,
          main: el.scrollWidth - el.clientWidth,
          clipped: [...el.querySelectorAll("h1,h2,h3,.puja-reading-card,.puja-step,.puja-materials p,.puja-categories a")]
            .filter(node => node.scrollWidth > node.clientWidth + 1).map(node => node.className || node.tagName),
        }));
        assert(metrics.page <= 1 && metrics.main <= 1 && !metrics.clipped.length, JSON.stringify({ language, view, width, metrics }));
        if (view === "reader") {
          assert.equal(await page.locator(".puja-reading-card").count(), 1);
          const continuous = await page.locator(".puja-reading-card").evaluate(el => {
            const materials = el.querySelector(".puja-materials"), steps = el.querySelector(".puja-steps");
            const a = materials.getBoundingClientRect(), b = steps.getBoundingClientRect();
            return b.top > a.bottom && Math.abs(a.left - b.left) < 1 &&
              [materials, steps].every(node => getComputedStyle(node).borderTopWidth === "0px");
          });
          assert(continuous, "Materials must precede worship paragraphs within one card at every width");
          assert.equal(await page.locator(".puja-reading-card details, .puja-reading-card summary").count(), 0);
        }
        if ([320, 1440].includes(width)) {
          const result = await new AxeBuilder({ page }).setLegacyMode().include(".puja-layout").analyze();
          assert.deepEqual(result.violations.map(v => ({ id: v.id, nodes: v.nodes.map(n => n.target) })), [], `${language}/${view}/${width}`);
          audits++;
        }
        if ([390, 1440].includes(width)) {
          if (view === "reader") await page.locator(".puja-reading-card").evaluate(el => el.scrollIntoView({ block: "start" }));
          await page.screenshot({ path: `.local/puja-${language}-${view}-${width}.png`, fullPage: view === "list" });
        }
        layouts.push({ language, view, width });
      }
    }
    await audit("list");
    await page.locator(".devotional-pagination a[rel=next]").click();
    assert.equal(new URL(page.url()).searchParams.get("page"), "2");
    assert.equal(await page.locator(".puja-card h3").first().innerText(), guides[12].title[language]);
    await page.locator(".puja-card").first().click();
    await page.locator("[data-puja-step]").first().waitFor();
    await page.locator(".devotional-back").click();
    assert.equal(new URL(page.url()).searchParams.get("page"), "2");
    await page.reload();
    await page.locator(".puja-card").first().waitFor();
    assert.equal(await page.locator(".puja-card").count(), 12);
    await page.locator(".devotional-pagination a[rel=next]").click();
    assert.equal(await page.locator(".puja-card").count(), 3);
    await page.goBack();
    assert.equal(new URL(page.url()).searchParams.get("page"), "2");
    // Search starts on page one, covers the entire catalogue, and retains focus while typing.
    const search = page.getByRole("searchbox");
    await search.fill(language === "en" ? "homa" : "होम");
    assert.equal(await page.locator(".puja-card").count(), 2);
    assert.equal(new URL(page.url()).searchParams.get("page"), null);
    assert.equal(await search.evaluate(el => el === document.activeElement), true);
    await page.locator(".puja-card").first().click();
    await page.locator("[data-puja-step]").first().waitFor();
    await page.locator(".devotional-back").click();
    assert.equal(new URL(page.url()).searchParams.get("q"), language === "en" ? "homa" : "होम");
    await search.fill("no-such-ritual-zzzz");
    assert.equal(await page.locator(".puja-card").count(), 0);
    await page.locator(".devotional-empty button").click();
    for (const [kind, count] of Object.entries({ festival: 10, deity: 9, vrat: 6, homa: 2 })) {
      await page.locator(`.puja-categories a[href*="category=${kind}"]`).click();
      assert.equal(await page.locator(".puja-card").count(), count);
    }
    // Open every guide through normal search/navigation; hard-reloading every module for each guide can exhaust local static-request limits.
    await page.goto(base + "/puja-vidhi");
    await page.getByRole("searchbox").waitFor();
    for (const guide of guides) {
      await page.getByRole("searchbox").fill(guide.title[language]);
      await page.locator(".puja-card").filter({ has: page.getByRole("heading", { level: 3, name: guide.title[language], exact: true }) }).click();
      await page.locator("[data-puja-step]").first().waitFor();
      assert.equal(await page.locator("h1").innerText(), guide.title[language]);
      assert.deepEqual(await page.locator(".puja-step-instruction").allTextContents(), guide.steps[language]);
      assert.deepEqual(await page.locator(".puja-step-number").allTextContents(), guide.steps[language].map((_, index) => `${index + 1}.`));
      assert.equal(await page.locator(".puja-step-mantras h3").count(), guide.step_mantras.flat().length, "Only the specific mantra titles belong under each instruction");
      assert.equal(await page.locator(".puja-step-mantras a, .puja-mantra-group-title, .puja-mantra-reference").count(), 0, "References and repeated guide titles must not interrupt worship steps");
      assert.deepEqual(await page.locator(".puja-reading > .devotional-image-credits li a").evaluateAll(nodes => nodes.filter(node => node.href.includes("drikpanchang.com")).map(node => node.href)), guide.sources);
      assert.deepEqual(await page.locator(".puja-material-item").allTextContents(), guide.materials[language]);
      const available = new Map(guide.mantra_readings.flatMap(key => mantras[key].sections.map(section => [section.id, section])));
      const sections = guide.step_mantras.flatMap(ids => ids.map(id => available.get(id)));
      for (const [index, ids] of guide.step_mantras.entries()) {
        const step = page.locator(`[data-puja-step="${index + 1}"]`);
        assert.deepEqual(await step.locator(".puja-mantra-section").evaluateAll(nodes => nodes.map(node => node.dataset.mantraId)), ids, `${guide.slug}: mantras must follow instruction ${index + 1}`);
        assert.equal(await step.locator(".puja-step-mantras").count(), ids.length ? 1 : 0);
      }
      assert.equal(await page.locator(".puja-reading > .puja-mantras").count(), 0, "Mantras belong under instructions rather than in a separate list");
      assert.deepEqual(await page.locator(".puja-mantra-section").evaluateAll(nodes => nodes.map(node => node.dataset.mantraId)), sections.map(section => section.id));
      assert.deepEqual(await page.locator(".puja-mantra-text").allTextContents(), sections.map(section => section.text));
      assert.deepEqual(await page.locator(".puja-mantra-meaning > p").allTextContents(), sections.map(section => section.meaning[language]));
      assert.equal(await page.locator('.puja-mantra-text[lang="sa"]').count(), sections.length);
      assert.equal(await page.locator(".puja-mantra-guidance").textContent(), guide.mantra_note[language]);
      if (guide.slug === "ahoi-ashtami") assert.equal(sections.length, 0, "Ahoi reference must not receive an invented mantra");
      assert.equal(await page.locator(".puja-related a").first().getAttribute("href"), guide.related);
      assert.equal(await page.locator("iframe,video,a[href*=youtube],a[href*=youtu\\.be]").count(), 0);
      await page.locator(".devotional-back").click();
      await page.getByRole("searchbox").waitFor();
    }
    await page.goto(base + "/puja-vidhi/diwali");
    await page.locator("[data-puja-step]").first().waitFor();
    const beforeSwitch = bodyRequests;
    const mantrasBeforeSwitch = mantraRequests;
    await page.locator(".language-selector select").selectOption(language === "en" ? "hi" : "en");
    assert.equal(bodyRequests, beforeSwitch, "Switching language must reuse the loaded guide");
    assert.equal(mantraRequests, mantrasBeforeSwitch, "Switching language must reuse loaded mantras");
    await page.locator(".language-selector select").selectOption(language);
    assert.equal(await page.locator(".puja-step-mantras details, .puja-step-mantras summary, .puja-step-mantras button").count(), 0);
    assert(await page.locator(".puja-mantra-text").evaluateAll(nodes => nodes.length > 0 && nodes.every(node => node.getClientRects().length > 0)), "Every mantra must be visible without interaction");
    await audit("reader");
    await page.setViewportSize({ width: 320, height: 1000 });
    await page.locator(".devotional-reader-tools button").nth(1).click();
    await page.waitForFunction(() => getComputedStyle(document.querySelector(".puja-reading")).fontSize === "18px");
    for (let i = 0; i < 5; i++) {
      await page.locator(".devotional-reader-tools button").last().press("Enter");
      await page.waitForFunction(size => getComputedStyle(document.querySelector(".puja-reading")).fontSize === `${size}px`, 20 + 2 * i);
    }
    assert.equal(await page.locator(".puja-reading").evaluate(el => getComputedStyle(el).fontSize), "28px");
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    await page.reload();
    await page.locator("[data-puja-step]").first().waitFor();
    assert.equal(await page.locator("h1").innerText(), guides[0].title[language]);
    // Reader errors can retry a fresh static-asset request without reloading the application.
    let fail = true;
    await page.route("**/*puja-guides*.json*", route => route.request().resourceType() === "fetch" && fail
      ? route.fulfill({ status: 503, body: "Unavailable" }) : route.continue());
    await page.goto(base + "/puja-vidhi/lakshmi");
    await page.locator('[role="alert"]').waitFor();
    assert.equal(await page.locator("[data-puja-step]").count(), 0);
    fail = false;
    await page.locator('[role="alert"] button').click();
    await page.locator("[data-puja-step]").first().waitFor();
    assert.equal(await page.locator("h1").innerText(), guides[1].title[language]);
    await page.unroute("**/*puja-guides*.json*");
    // A failed mantra asset must fail the reader visibly; retry must restore both text and meanings.
    let failMantras = true;
    await page.route("**/*puja-mantras*.json*", route => route.request().resourceType() === "fetch" && failMantras
      ? route.fulfill({ status: 503, body: "Unavailable" }) : route.continue());
    await page.goto(base + "/puja-vidhi/ganesha");
    await page.locator('[role="alert"]').waitFor();
    assert.equal(await page.locator(".puja-mantra-text").count(), 0);
    failMantras = false;
    await page.locator('[role="alert"] button').click();
    await page.locator(".puja-mantra-section").first().waitFor({ state: "attached" });
    assert.equal(await page.locator(".puja-mantra-meaning > p").count(), mantras.ganesha.sections.length);
    await page.unroute("**/*puja-mantras*.json*");
    // A cancelled old reader request must not replace a later selection.
    let first = true;
    await page.route("**/*puja-guides*.json*", async route => {
      if (route.request().resourceType() === "fetch" && first) { first = false; await delay(500); }
      await route.continue().catch(() => {});
    });
    await page.locator(".devotional-back").click();
    await page.locator(".puja-card").first().click();
    await page.locator(".devotional-back").click();
    const finalGuide = guides.find(guide => guide.kind === "deity" && guide.title[language] !== guides[2].title[language]) || guides[2];
    await page.getByRole("searchbox").fill(finalGuide.title[language]);
    await page.locator(".puja-card").first().click();
    await page.locator("[data-puja-step]").first().waitFor();
    assert.deepEqual(await page.locator(".puja-step-instruction").allTextContents(), finalGuide.steps[language]);
    await page.unroute("**/*puja-guides*.json*");
    await page.goto(base + "/puja-vidhi/unknown-guide");
    await page.locator(".devotional-empty h1").waitFor();
    assert.equal(await page.locator(".puja-reader").count(), 0);
    await page.goto(base + "/puja-vidhi?page=999999");
    await page.locator(".puja-card").first().waitFor();
    assert.equal(await page.locator(".puja-card").count(), 3);
    await page.goto(base + "/puja-vidhi?page=-2");
    await page.locator(".puja-card").first().waitFor();
    assert.equal(await page.locator(".puja-card").count(), 12);
    // The existing devotional library remains reachable, and links back to the new collection.
    await page.locator('a[href="/bhakti"]').click();
    await page.locator(".devotional-card").first().waitFor();
    await page.locator(".puja-library-link").click();
    await page.locator(".puja-card").first().waitFor();
    // The signed-in profile trigger must fit beside the main-menu destinations at every header breakpoint.
    await page.route("**/api/me", route => route.fulfill({ json: { user: { id: "browser-fixture", phone: "", marketing: false }, csrf: "fixture", reports: [], orders: [], bookings: [], pdf_jobs: [] } }));
    await page.goto(base);
    await page.locator('.profile-trigger').waitFor();
    for (const width of [320, 390, 768, 1280, 1281, 1366, 1440, 1920]) {
      await page.setViewportSize({ width, height: 1000 });
      // Use the same one-pixel fractional layout tolerance as the catalogue audits above.
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `${language}/header/${width}`);
      const menu = page.locator(".site-header .menu-toggle");
      if (await menu.isVisible()) await menu.click();
      assert(await page.locator("#primary-navigation").getByRole("button", { name: language === "en" ? "Puja Vidhi" : "पूजा विधि", exact: true }).isVisible());
      if (await menu.isVisible()) await menu.click();
    }
    await page.setViewportSize({ width: 1440, height: 1000 });
    await page.locator("#primary-navigation").getByRole("button", { name: language === "en" ? "Puja Vidhi" : "पूजा विधि", exact: true }).click();
    await page.locator(".puja-card").first().waitFor();
    assert.deepEqual(errors, []);
    const result = { language, passed: true, guides: guides.length, layouts: layouts.length, axeAudits: audits, deferredBody: true, allReaderTexts: true, categories: true, pagination: true, search: true, history: true, reload: true, fontSize: true, retry: true, staleResponse: true, signedInMenu: true, textOnly: true, errors };
    console.log(JSON.stringify(result));
    return result;
  } catch (error) {
    const failedPage = browser.contexts()[0]?.pages()[0];
    await failedPage?.screenshot({ path: `.local/puja-${language}-failure.png`, fullPage: false }).catch(() => {});
    console.error(JSON.stringify({ language, url: failedPage?.url(), errors }));
    throw error;
  } finally { await browser.close(); }
}

/** run releases each language browser before the next pass and writes the combined read-only verification record. */
async function run() {
  const results = [];
  for (const language of ["en", "hi"]) results.push(await verifyLanguage(language));
  const result = { ...results[0], languages: ["en", "hi"], layouts: results.reduce((sum, r) => sum + r.layouts, 0), axeAudits: results.reduce((sum, r) => sum + r.axeAudits, 0) };
  fs.writeFileSync(".local/puja-library-results.json", JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result));
}
run().catch(error => { console.error(error.stack); process.exitCode = 1; });
