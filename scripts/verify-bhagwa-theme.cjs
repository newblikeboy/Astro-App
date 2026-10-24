// Read-only brand/layout verification; never submits to the customer's database.
const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

/** verifyBhagwaTheme checks the shared saffron identity, bilingual layouts, contrast and preserved cart/form state. */
async function verifyBhagwaTheme() {
  process.chdir(path.resolve(__dirname, ".."));
  const base = process.env.DESIGN_QA_URL || "http://localhost:5173";
  assert(["localhost", "127.0.0.1"].includes(new URL(base).hostname));
  const browser = await chromium.launch({
    executablePath:
      process.env.CHROME_PATH ||
      "C:/Program Files/Google/Chrome/Application/chrome.exe",
    headless: true,
  });
  const evidence = { passed: false, layouts: [], audits: [], errors: [] };
  fs.mkdirSync(".local/bhagwa-preview", { recursive: true });
  try {
    const context = await browser.newContext({ reducedMotion: "reduce" });
    const catalog = await (
      await context.request.get(base + "/api/shop")
    ).json();
    const ruby = catalog.products.find((product) => product.slug === "ruby");
    assert(ruby, "Expected the existing ruby listing");
    await context.addInitScript((id) => {
      localStorage.setItem("m4y-analytics", "no");
      localStorage.setItem("m4y-language", "en");
      sessionStorage.setItem(
        "m4y-shop-bag",
        JSON.stringify([{ product_id: id, quantity: 1 }]),
      );
    }, ruby.id);
    let accountFixture = false;
    await context.route("**/api/**", (route) => {
      assert.equal(
        route.request().method(),
        "GET",
        "Design QA must not write application data",
      );
      // The account preview uses fictional browser data and never authenticates a real customer.
      if (accountFixture && new URL(route.request().url()).pathname === "/api/me")
        return route.fulfill({
          json: {
            user: { id: "qa-theme-customer", phone: "+919800000000", marketing: false },
            csrf: "browser-only-theme-qa",
            reports: [{
              id: "qa-theme-report",
              birth: { name: "QA saved Kundli", date: "2000-01-01", unknown_time: false },
              place: { name: "New Delhi", hindi: "नई दिल्ली" },
            }],
            orders: [],
            bookings: [],
            pdf_jobs: [],
          },
        });
      return route.continue();
    });
    const page = await context.newPage();
    page.on("pageerror", (error) => evidence.errors.push(error.message));
    for (const [route, name, ready] of [
      ["/", "home", ".daily-limbs"],
      ["/shop", "shop", ".gem-card"],
      ["/shop/stones/ruby", "stone", ".stone-buy-panel"],
      ["/shop/bag", "checkout", ".checkout-items"],
      ["/login", "login", ".auth-card"],
      ["/account", "account", ".saved-report"],
      ["/kundli-matching", "matching", ".matching-form-section"],
      ["/admin/login", "admin", ".admin-login"],
      ["/bhakti", "bhakti", ".devotional-card"],
    ]) {
      if (
        process.env.DESIGN_QA_PAGES &&
        !process.env.DESIGN_QA_PAGES.split(",").includes(name)
      )
        continue;
      accountFixture = name === "account";
      await page.goto(base + route);
      await page.locator(ready).first().waitFor();
      if (name === "home")
        await page.locator("#name").fill("Preserved Bhagwa design draft");
      for (const language of ["en", "hi"]) {
        await page
          .locator(".language-selector select")
          .first()
          .selectOption(language);
        for (const width of [320, 390, 650, 768, 1024, 1201, 1280, 1281, 1440]) {
          await page.setViewportSize({ width, height: 1000 });
          await page.evaluate(async () => {
            await document.fonts.ready;
            await new Promise((resolve) =>
              requestAnimationFrame(() => requestAnimationFrame(resolve)),
            );
          });
          const state = await page.evaluate(() => {
            const mark = document.querySelector("header .brand-mark");
            return {
              overflow: document.documentElement.scrollWidth - innerWidth,
              saffron: getComputedStyle(document.documentElement)
                .getPropertyValue("--bhagwa-saffron")
                .trim(),
              logo: mark.complete && mark.naturalWidth > 0,
              clipped: [
                ...document.querySelectorAll(
                  "header a,header button,header select, main input,main textarea",
                ),
              ]
                .filter((element) => {
                  const rect = element.getBoundingClientRect();
                  return (
                    rect.width &&
                    getComputedStyle(element).visibility !== "hidden" &&
                    (rect.left < -1 || rect.right > innerWidth + 1)
                  );
                })
                .map((element) => element.outerHTML.slice(0, 120)),
            };
          });
          assert(
            state.overflow <= 1 && state.logo && !state.clipped.length,
            JSON.stringify({ name, language, width, state }),
          );
          assert.equal(state.saffron, "#c94b0b");
          if (name === "home")
            assert.equal(
              await page.locator("#name").inputValue(),
              "Preserved Bhagwa design draft",
            );
          if ([390, 1440].includes(width)) {
            await page.evaluate(() => scrollTo(0, 0));
            await page.screenshot({
              path: `.local/bhagwa-preview/${name}-${language}-${width}.png`,
              fullPage: name !== "home",
            });
            const audit = await new AxeBuilder({ page })
              .withTags(["wcag2a", "wcag2aa", "wcag21aa"])
              .analyze();
            evidence.audits.push({
              name,
              language,
              width,
              violations: audit.violations.map((violation) => ({
                id: violation.id,
                impact: violation.impact,
                nodes: violation.nodes.map((node) => ({
                  target: node.target,
                  summary: node.failureSummary,
                })),
              })),
            });
          }
          evidence.layouts.push({ name, language, width });
        }
      }
      if (name === "home") {
        await page
          .locator(".language-selector select")
          .first()
          .selectOption("en");
        for (const id of ["kundli", "panchang", "gemstones"])
          await page
            .locator("#" + id)
            .screenshot({ path: `.local/bhagwa-preview/${id}-desktop.png` });
        await page
          .locator("footer")
          .screenshot({ path: ".local/bhagwa-preview/footer-desktop.png" });
      }
      if (name === "checkout") {
        await page
          .locator(".language-selector select")
          .first()
          .selectOption("en");
        await page
          .getByRole("button", { name: /Increase quantity/ })
          .first()
          .click();
        assert.equal(
          await page.getByRole("spinbutton").first().inputValue(),
          "2",
        );
        await page
          .getByRole("button", { name: /Decrease quantity/ })
          .first()
          .click();
        assert.equal(
          await page.getByRole("spinbutton").first().inputValue(),
          "1",
        );
      }
    }
    assert.deepEqual(evidence.errors, []);
    assert(
      evidence.audits.every((audit) => audit.violations.length === 0),
      JSON.stringify(
        evidence.audits.filter((audit) => audit.violations.length),
      ),
    );
    evidence.passed = true;
    console.log(
      JSON.stringify({
        passed: true,
        layouts: evidence.layouts.length,
        accessibilityAudits: evidence.audits.length,
        errors: evidence.errors,
      }),
    );
  } finally {
    fs.writeFileSync(
      process.env.DESIGN_QA_PAGES
        ? ".local/bhagwa-focused-verification.json"
        : ".local/bhagwa-verification.json",
      JSON.stringify(evidence, null, 2),
    );
    await browser.close();
  }
}
verifyBhagwaTheme().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
