import { test, expect } from "@playwright/test";
import catalog from "../../internal/platform/data/gemstone-catalog.json" with { type: "json" };

test("starter catalogue supports cart estimates, responsive detail pages and manual admin updates", async ({
  page,
  context,
  baseURL,
}) => {
  test.setTimeout(180000);
  const headers = { Origin: baseURL! };
  const login = await context.request.post("/api/admin/login", {
    headers,
    data: { password: "local-admin-change-me" },
  });
  expect(login.ok()).toBeTruthy();
  const { csrf } = await login.json();
  for (const product of catalog) {
    const result = await context.request.post("/api/admin/shop/products", {
      headers: { ...headers, "X-CSRF-Token": csrf },
      data: { ...product, id: "", version: 0 },
    });
    expect(result.ok(), await result.text()).toBeTruthy();
  }
  await context.addInitScript(() => {
    if (!localStorage.getItem("m4y-language"))
      localStorage.setItem("m4y-language", "en");
    localStorage.setItem("m4y-analytics", "no");
  });
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto("/shop");
  await expect(page.locator(".gem-card")).toHaveCount(16);
  await expect(page.locator(".gem-photo-label")).toHaveCount(0);
  await expect(page.locator("body")).not.toContainText("Indicative price");
  await page.getByLabel("Stone collection").selectOption("navratna");
  await expect(page.locator(".gem-card")).toHaveCount(9);
  await page.getByLabel("Stone collection").selectOption("alternatives");
  await expect(page.locator(".gem-card")).toHaveCount(7);
  await page.goto("/shop/stones/ruby");
  await expect(page.locator(".stone-buy-panel h1")).toHaveText("Ruby");
  await expect(page.locator(".stone-reference")).toHaveCount(0);
  await expect(page.locator("body")).not.toContainText("AI-generated sample");
  await page
    .getByRole("button", { name: "Increase quantity", exact: true })
    .click();
  await page
    .locator(".stone-order-buttons")
    .getByRole("button", { name: "Add to cart", exact: true })
    .click();
  await expect(page.locator(".cart-count")).toHaveText("2");
  await page.getByRole("button", { name: "Enlarge image" }).click();
  await expect(page.locator(".stone-photo-dialog")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.locator(".stone-photo-dialog")).not.toBeVisible();
  await page.locator(".cart-link").click();
  await expect(page.locator(".shop-bag-lines article")).toHaveCount(1);
  await expect(
    page.locator('.checkout-progress [aria-current="step"]'),
  ).toContainText("Bag");
  await page
    .getByRole("button", { name: "Increase quantity Ruby", exact: true })
    .click();
  await expect(page.locator(".cart-count")).toHaveText("3");
  await page
    .getByRole("button", { name: "Decrease quantity Ruby", exact: true })
    .click();
  await expect(page.locator(".cart-count")).toHaveText("2");
  await page.getByRole("spinbutton").fill("3");
  await expect(page.locator(".cart-count")).toHaveText("3");
  await expect(
    page.getByText("Stock confirmation", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Review order & payment" }),
  ).toHaveCount(0);
  await page.locator(".shop-header .brand").click();
  await expect(page.locator(".cart-count")).toHaveText("3");
  await page
    .locator(".home-stone .cart-purchase-control button")
    .first()
    .click();
  await expect(page.locator(".cart-count")).toHaveText("4");
  await page.reload();
  await expect(page.locator(".cart-count")).toHaveText("4");

  for (const product of catalog) {
    await page.goto("/shop/stones/" + product.slug);
    await expect(page.locator(".stone-buy-panel h1")).toBeVisible();
    await page
      .locator(".stone-main-photo img")
      .evaluate((image: HTMLImageElement) => image.decode());
    for (const language of ["en", "hi"]) {
      await page.locator(".language-selector select").selectOption(language);
      for (const width of [320, 1440]) {
        await page.setViewportSize({ width, height: 950 });
        await page.evaluate(() => document.fonts.ready);
        expect(
          await page.evaluate(
            () => document.documentElement.scrollWidth <= innerWidth + 1,
          ),
          product.slug + "/" + language + "/" + width,
        ).toBeTruthy();
      }
    }
  }
  await page.locator(".language-selector select").selectOption("en");
  await page.goto("/admin");
  await page
    .getByRole("button", { name: "Gemstone shop", exact: true })
    .click();
  await page
    .locator(".shop-product-list button")
    .filter({ has: page.getByText("Ruby", { exact: true }) })
    .click();
  await page.getByLabel("Price per piece (INR)").fill("12345");
  await page
    .getByLabel("Description (English)")
    .fill("A manually updated Ruby description from the merchant.");
  await page.locator(".shop-guide-editor summary").click();
  await page
    .getByRole("textbox", { name: "Everyday care (English)", exact: true })
    .fill("Merchant care guidance saved from the editor.");
  await page.getByRole("button", { name: "Save stone", exact: true }).click();
  await expect(
    page.getByText("Saved successfully", { exact: true }),
  ).toBeVisible();
  await page.goto("/shop/stones/ruby");
  await expect(page.locator(".stone-price strong")).toContainText("12,345");
  await expect(page.locator(".stone-summary")).toHaveText(
    "A manually updated Ruby description from the merchant.",
  );
  await expect(
    page.getByText("Merchant care guidance saved from the editor."),
  ).toBeVisible();
  expect(errors).toEqual([]);
});
