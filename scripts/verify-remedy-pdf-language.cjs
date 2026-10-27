// Synthetic account and downloads in the browser only. No customer, order or provider writes.
const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const assert = require("node:assert/strict");
const fs = require("node:fs");
const base = process.env.REMEDY_PDF_QA_URL || "http://127.0.0.1:5173";

/** verify checks the language chooser, independent website language, download filenames, busy/error states and responsive accessibility. */
async function verify() {
  const config = await (await fetch(base + "/api/config")).json();
  const browser = await chromium.launch({ executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe", headless: true });
  const results = { layouts: [], requested: [], errors: [], passed: false };
  try {
    const context = await browser.newContext({ acceptDownloads: true, reducedMotion: "reduce" });
    await context.addInitScript(() => {
      localStorage.setItem("m4y-language", "en");
      localStorage.setItem("m4y-analytics", "no");
    });
    const page = await context.newPage();
    page.on("pageerror", (error) => results.errors.push(error.message));
    let mode = "success";
    await page.route("**/api/**", async (route) => {
      assert.equal(route.request().method(), "GET", "Unexpected account write");
      const url = new URL(route.request().url());
      if (url.pathname === "/api/config") return route.fulfill({ json: config });
      if (url.pathname === "/api/me") return route.fulfill({ json: {
        user: { id: "language-qa", phone: "+919000000000", marketing: false }, csrf: "synthetic",
        reports: [], bookings: [],
        orders: [{ id: "pdf-language-qa", title: "Personal remedies PDF", product: "pdf", status: "paid", amount: 9900, demo: true }],
        pdf_jobs: [{ id: "pdf:pdf-language-qa:1", ref: "pdf-language-qa", status: "done" }],
      } });
      if (url.pathname === "/api/orders/pdf-language-qa/pdf") {
        const language = url.searchParams.get("language");
        results.requested.push(language);
        await new Promise((resolve) => setTimeout(resolve, 300));
        if (mode === "error") return route.fulfill({ status: 503, json: { error: "The remedy PDF could not be prepared. Please try again." } });
        return route.fulfill({ status: 200, contentType: "application/pdf", body: `%PDF-1.7\nSynthetic ${language} download` });
      }
      return route.fulfill({ json: {} });
    });
    await page.goto(base + "/account");
    const chooser = page.locator(".remedy-pdf-download");
    await chooser.waitFor();
    const select = chooser.locator("select");
    assert.equal(await select.inputValue(), "en");
    for (const language of ["en", "hi"]) {
      await page.locator(".language-selector select").selectOption(language);
      for (const width of [320, 390, 768, 1440]) {
        await page.setViewportSize({ width, height: 1000 });
        await page.evaluate(() => document.fonts.ready);
        await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
        const overflow = await page.evaluate(() => [...document.querySelectorAll("body *")].filter((node) => {
          const box = node.getBoundingClientRect();
          return box.width && box.right > innerWidth + 1;
        }).slice(-12).map((node) => ({ tag: node.tagName, class: node.className, text: node.textContent.slice(0, 70), right: node.getBoundingClientRect().right })));
        if (overflow.length) await page.screenshot({ path: ".local/remedy-pdf-overflow.png", fullPage: true });
        assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `${language}/${width}: page overflow ${JSON.stringify(overflow)}`);
        const audit = await new AxeBuilder({ page }).include(".remedy-pdf-download").analyze();
        assert.deepEqual(audit.violations.map((v) => ({ id: v.id, impact: v.impact })), []);
        if (width === 390) await page.screenshot({ path: `.local/remedy-pdf-language-${language}.png`, fullPage: true });
        results.layouts.push({ language, width });
      }
    }
    // PDF language is independent of the website language, including after switching the website to Hindi.
    assert.equal(await select.inputValue(), "en");
    for (const language of ["en", "hi"]) {
      await select.selectOption(language);
      const download = page.waitForEvent("download");
      await chooser.locator("button").click();
      assert(await select.isDisabled());
      assert(await chooser.locator("button").isDisabled());
      assert.equal((await download).suggestedFilename(), `Mishra4You-remedies-${language}.pdf`);
      await chooser.locator("button:not([disabled])").waitFor();
    }
    mode = "error";
    await chooser.locator("button").click();
    await chooser.locator('[role="alert"]').waitFor();
    assert.equal(await chooser.locator('[role="alert"]').innerText(), "उपाय PDF तैयार नहीं हो सकी। कृपया दोबारा प्रयास करें।");
    mode = "success";
    const retried = page.waitForEvent("download");
    await chooser.locator("button").click();
    await retried;
    await chooser.locator("button:not([disabled])").waitFor();
    assert.equal(await chooser.locator('[role="alert"]').count(), 0);
    assert.deepEqual(results.requested, ["en", "hi", "hi", "hi"]);
    assert.deepEqual(results.errors, []);
    results.passed = true;
    fs.writeFileSync(".local/remedy-pdf-language-browser-results.json", JSON.stringify(results, null, 2));
    console.log(JSON.stringify(results));
  } finally { await browser.close(); }
}

verify().catch((error) => { console.error(error); process.exitCode = 1; });
