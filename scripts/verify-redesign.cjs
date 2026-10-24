const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const fs = require("node:fs");
const assert = require("node:assert/strict");
(async () => {
  const browser = await chromium.launch({
    executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe",
    headless: true,
  });
  const errors = [],
    results = [],
    audits = [];
  try {
    const context = await browser.newContext({
      reducedMotion: "reduce",
      viewport: { width: 1440, height: 1000 },
    });
    await context.addInitScript(() => {
      localStorage.setItem("m4y-analytics", "no");
      localStorage.setItem("m4y-language", "en");
    });
    const page = await context.newPage();
    page.on("pageerror", (e) => errors.push(e.message));
    await page.route("**/api/**", (r) =>
      r.request().method() === "GET" ? r.continue() : r.abort(),
    );
    await page.goto("http://localhost:5173");
    await page.locator(".daily-limbs").waitFor();
    await page.evaluate(() => document.fonts.ready);
    await page.screenshot({ path: ".local/redesign-opening-desktop.png" });
    for (const [id, name] of [
      ["kundli", "birth"],
      ["panchang", "tools"],
    ])
      await page
        .locator("#" + id)
        .screenshot({ path: ".local/redesign-" + name + "-desktop.png" });
    await page.locator("#name").fill("Keep these birth details");
    for (const language of ["en", "hi"]) {
      await page.locator(".language-selector select").selectOption(language);
      for (const width of [320, 339, 390, 650, 768, 900, 1024, 1100, 1440]) {
        await page.setViewportSize({ width, height: 950 });
        await page.evaluate(() => document.fonts.ready);
        const state = await page.evaluate(() => ({
          overflow: document.documentElement.scrollWidth - innerWidth,
          headers: document.querySelectorAll("h1").length,
          italic: [
            ...document.querySelectorAll("main *,header *,footer *"),
          ].some((el) => getComputedStyle(el).fontStyle === "italic"),
          clipped: [
            ...document.querySelectorAll(
              "#kundli input,#kundli .input-icon,header button,.daily-controls input",
            ),
          ]
            .filter((el) => {
              const r = el.getBoundingClientRect();
              return r.width && (r.left < -1 || r.right > innerWidth + 1);
            })
            .map((el) => el.outerHTML.slice(0, 130)),
          guide: getComputedStyle(document.querySelector(".step")).textAlign,
          footer: getComputedStyle(document.querySelector(".footer-note"))
            .textAlign,
        }));
        assert(
          state.overflow <= 1 &&
            !state.clipped.length &&
            !state.italic &&
            state.headers === 1,
          JSON.stringify({ language, width, state }),
        );
        assert.equal(
          await page.locator("#name").inputValue(),
          "Keep these birth details",
        );
        if (width <= 650) {
          assert.equal(state.guide, "center");
          assert.equal(state.footer, "center");
        }
        if ([390, 1440].includes(width)) {
          const axe = await new AxeBuilder({ page }).analyze();
          assert.deepEqual(
            axe.violations.map((v) => ({
              id: v.id,
              impact: v.impact,
              targets: v.nodes.map((n) => n.target),
            })),
            [],
            JSON.stringify({ language, width }),
          );
          audits.push({ language, width });
        }
        results.push({ language, width });
        if (width === 390) {
          await page.evaluate(() => scrollTo(0, 0));
          await page.screenshot({
            path: `.local/redesign-opening-${language}-mobile.png`,
          });
          await page.locator("#kundli").screenshot({
            path: `.local/redesign-birth-${language}-mobile.png`,
          });
          await page.locator("footer").screenshot({
            path: `.local/redesign-footer-${language}-mobile.png`,
          });
        }
      }
    }
    await page.locator(".language-selector select").selectOption("en");
    await page.setViewportSize({ width: 390, height: 950 });
    await page.locator(".menu-toggle").click();
    assert.equal(
      await page.locator(".menu-toggle").getAttribute("aria-expanded"),
      "true",
    );
    await page.keyboard.press("Escape");
    assert.equal(
      await page.locator(".menu-toggle").getAttribute("aria-expanded"),
      "false",
    );
    await page.locator(".hero-shortcuts button").nth(1).click();
    await page.locator(".sky-positions tbody tr").first().waitFor();
    assert.equal(
      await page.locator(".daily-tabs [aria-selected=true]").innerText(),
      "Planets",
    );
    await page.locator(".hero-shortcuts button").nth(2).click();
    await page.locator(".sky-eclipses article").first().waitFor();
    assert.equal(
      await page.locator(".daily-tabs [aria-selected=true]").innerText(),
      "Eclipses",
    );
    await page.locator(".hero-shortcuts button").first().click();
    await page.locator(".daily-limbs").waitFor();
    const rail = page.locator(".videos-grid");
    await rail.evaluate((el) => {
      el.scrollLeft = 0;
    });
    await page
      .getByRole("button", { name: "Next videos", exact: true })
      .click();
    assert(await rail.evaluate((el) => el.scrollLeft > 0));
    await page
      .getByRole("button", { name: "Previous videos", exact: true })
      .click();
    assert(await rail.evaluate((el) => el.scrollLeft < 2));
    await rail.focus();
    await page.keyboard.press("ArrowRight");
    await page.waitForFunction(
      () => document.querySelector(".videos-grid").scrollLeft > 0,
    );
    await page.locator(".account-button").click();
    await page.locator(".auth-card").waitFor();
    await page.locator("#phone").fill("9876543210");
    const modalAxe = await new AxeBuilder({ page }).analyze();
    assert.deepEqual(
      modalAxe.violations.map((v) => v.id),
      [],
    );
    audits.push({ view: "login" });
    await page
      .locator(".auth-card")
      .screenshot({ path: ".local/redesign-login-mobile.png" });
    await page.goBack();
    await page.locator(".footer-bottom button").first().click();
    await page.locator("dialog[open]").waitFor();
    await page.keyboard.press("Escape");
    assert.deepEqual(errors, []);
    const report = {
      passed: true,
      pageLayouts: results.length,
      axeAudits: audits.length,
      heroShortcuts: true,
      videoButtons: true,
      videoKeyboard: true,
      mobileMenu: true,
      preservedBirthDetails: true,
      errors,
    };
    fs.writeFileSync(
      ".local/redesign-results.json",
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
