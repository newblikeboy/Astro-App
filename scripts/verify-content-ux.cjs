// Local UI verification. All checkout POSTs below are fulfilled in the browser and never reach the server.
const { chromium } = require("../web/node_modules/playwright");
const assert = require("node:assert/strict");
const fs = require("node:fs");

/** verifyContentUX checks navigation destinations, filter disclosure, non-duplicating cart actions and native checkout validation. */
async function verifyContentUX() {
  const base = process.env.DESIGN_QA_URL || "http://localhost:5173";
  assert(["localhost", "127.0.0.1"].includes(new URL(base).hostname));
  const browser = await chromium.launch({
    executablePath:
      process.env.CHROME_PATH ||
      "C:/Program Files/Google/Chrome/Application/chrome.exe",
    headless: true,
  });
  const results = { passed: false, checks: [], errors: [] };
  try {
    const context = await browser.newContext({
      viewport: { width: 390, height: 844 },
      reducedMotion: "reduce",
    });
    await context.addInitScript(() => {
      localStorage.setItem("m4y-language", "en");
      localStorage.setItem("m4y-analytics", "no");
    });
    await context.route("**/api/**", (route) => {
      assert.equal(route.request().method(), "GET");
      return route.continue();
    });
    const catalog = await (
      await context.request.get(base + "/api/shop")
    ).json();
    const ruby = catalog.products.find((product) => product.slug === "ruby");
    const page = await context.newPage();
    page.on("pageerror", (error) => results.errors.push(error.message));
    await page.goto(base);
    assert.equal(await page.locator(".dev-ribbon").count(), 0);
    await page.locator(".home-stone").first().waitFor();
    assert.equal(await page.locator(".home-stone").count(), 3);
    assert(
      await page.evaluate(() => {
        const ids = ["kundli", "gemstones", "panchang", "videos"];
        return ids.every(
          (id, index) =>
            !index ||
            document.getElementById(ids[index - 1]).offsetTop <
              document.getElementById(id).offsetTop,
        );
      }),
    );
    assert.equal(await page.locator('#guidance,#consultation,.offers-section').count(), 0);
    await page.locator(".hero-actions .button").click();
    await page.waitForFunction(() => document.activeElement.id === "name");
    results.checks.push(
      "Primary chart action focuses the first birth field; main content follows the customer journey",
    );
    await page.locator(".hero-actions .text-button").click();
    await page.locator('.auth-card').waitFor();
    assert.equal(new URL(page.url()).pathname, "/login");
    assert.equal(await page.evaluate(() => sessionStorage.getItem('m4y-login-intent')), 'consultation');
    await page.goto(base);
    await page.locator('.hero-shortcuts').waitFor();
    await page.locator(".hero-shortcuts button").first().click();
    await page.locator(".daily-limbs").waitFor();
    assert.equal(new URL(page.url()).hash, "#panchang");
    results.checks.push(
      "Landing guidance cards are absent; consultation enters the purchase/login flow and daily tools retain their destination",
    );
    await page.goto(base + "/shop");
    await page.locator(".gem-card").first().waitFor();
    assert.equal(
      await page.locator(".shop-extra-filters").getAttribute("open"),
      null,
    );
    await page.locator(".shop-extra-filters summary").click();
    await page.getByRole("button", { name: "Precious", exact: true }).click();
    assert.equal(
      await page.locator(".gem-card").count(),
      catalog.products.filter((product) => product.kind === "precious").length,
    );
    await page
      .getByRole("button", { name: "Clear filters", exact: true })
      .click();
    assert.equal(
      await page.locator(".gem-card").count(),
      catalog.products.length,
    );
    results.checks.push(
      "Secondary filters remain discoverable and can be cleared",
    );
    await page.goto(base + "/shop/stones/ruby");
    await page.locator(".stone-order-buttons").waitFor();
    await page
      .locator(".stone-main-photo img")
      .evaluate((image) => image.decode());
    assert.equal(
      await page.locator(".stone-main-photo img").getAttribute("loading"),
      "eager",
    );
    assert.equal(await page.locator(".stone-mobile-cart").count(), 0);
    assert.equal(await page.locator(".stone-order-buttons button").count(), 1);
    await page.locator(".stone-order-buttons button").click();
    assert.equal(await page.locator(".cart-count").innerText(), "1");
    await page.locator(".stone-info-grid").scrollIntoViewIfNeeded();
    await page.locator(".stone-mobile-cart").waitFor();
    await page.evaluate(() => scrollTo(0, 0));
    await page.waitForFunction(
      () => !document.querySelector(".stone-mobile-cart"),
    );
    await page.locator(".stone-view-cart").click();
    await page.locator(".checkout-items").waitFor();
    assert.equal(await page.locator(".cart-count").innerText(), "1");
    results.checks.push(
      "View cart never adds another item; the phone bar appears only after passing the main action",
    );
    await context.close();

    const checkout = await browser.newContext({
      viewport: { width: 390, height: 844 },
      reducedMotion: "reduce",
    });
    await checkout.addInitScript((id) => {
      localStorage.setItem("m4y-language", "en");
      localStorage.setItem("m4y-analytics", "no");
      sessionStorage.setItem(
        "m4y-shop-bag",
        JSON.stringify([{ product_id: id, quantity: 1 }]),
      );
    }, ruby.id);
    const requests = [];
    await checkout.route("**/api/**", (route) => {
      const request = route.request(),
        pathname = new URL(request.url()).pathname;
      if (pathname === "/api/shop/checkout" && request.method() === "POST") {
        requests.push(request.postDataJSON());
        return route.fulfill({
          status: 422,
          json: { error: "UI verification: no order created." },
        });
      }
      assert.equal(
        request.method(),
        "GET",
        "Only the mocked checkout may submit",
      );
      if (pathname === "/api/shop")
        return route.fulfill({
          json: {
            ...catalog,
            products: catalog.products.map((product) => ({
              ...product,
              sample: false,
            })),
            checkout_available: true,
            settings: {
              enabled: true,
              shipping_fee: 15000,
              tax_mode: "inclusive",
              tax_bps: 300,
              delivery_policy: "Preview dispatch policy.",
              return_policy: "Preview return policy.",
            },
          },
        });
      if (pathname === "/api/me")
        return route.fulfill({
          json: {
            user: { id: "ux-preview", phone: "+919876543210" },
            csrf: "ux-preview",
            reports: [],
            orders: [],
            bookings: [],
            pdf_jobs: [],
          },
        });
      return route.continue();
    });
    const delivery = await checkout.newPage();
    delivery.on("pageerror", (error) => results.errors.push(error.message));
    await delivery.goto(base + "/shop/bag");
    const submit = delivery.getByRole("button", {
      name: "Review order & payment",
      exact: true,
    });
    await submit.waitFor();
    assert.equal(await submit.count(), 1);
    assert.equal(
      await submit.evaluate((button) => button.form.id),
      "checkout-delivery",
    );
    assert(
      await delivery.evaluate(() => {
        const items = document
          .querySelector(".checkout-items")
          .getBoundingClientRect();
        const form = document
          .querySelector(".shop-delivery-form")
          .getBoundingClientRect();
        const summary = document
          .querySelector(".shop-order-summary")
          .getBoundingClientRect();
        return items.top < form.top && form.top < summary.top;
      }),
    );
    await submit.click();
    assert.equal(requests.length, 0);
    assert.equal(
      await delivery
        .getByLabel("Recipient name", { exact: true })
        .evaluate((input) => input === document.activeElement),
      true,
    );
    for (const [label, value] of [
      ["Recipient name", "UI Preview"],
      ["Delivery mobile number", "9876543210"],
      ["Address line 1", "Preview address"],
      ["City", "Delhi"],
      ["State", "Delhi"],
      ["PIN code", "110001"],
    ])
      await delivery.getByLabel(label, { exact: true }).fill(value);
    await submit.click();
    assert.equal(requests.length, 0);
    await delivery.getByRole("checkbox").check();
    await submit.click();
    await delivery.waitForFunction(() =>
      document
        .querySelector('[role="alert"]')
        ?.textContent.includes("UI verification"),
    );
    assert.equal(requests.length, 1);
    assert.deepEqual(requests[0].items, [{ product_id: ruby.id, quantity: 1 }]);
    assert(!("price" in requests[0]) && !("amount" in requests[0]));
    results.checks.push(
      "One summary action validates delivery and policy consent before submitting item IDs/quantities; the submission was mocked",
    );
    assert.deepEqual(results.errors, []);
    results.passed = true;
    console.log(JSON.stringify(results));
  } finally {
    fs.writeFileSync(
      ".local/content-ux-verification.json",
      JSON.stringify(results, null, 2),
    );
    await browser.close();
  }
}
verifyContentUX().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
