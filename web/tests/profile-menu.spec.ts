import { test, expect } from "@playwright/test";
import type { Page } from "@playwright/test";

/** login verifies an isolated development customer with the server's simulated OTP. */
async function login(page: Page) {
  await page.locator("#phone").fill("+9197" + String(Date.now()).slice(-8));
  await page.getByRole("button", { name: "Get OTP", exact: true }).click();
  const code = await page.locator(".auth-card .notice b").textContent();
  await page.locator("#otp").fill(code!);
  await page.getByRole("button", { name: "Verify", exact: true }).click();
}

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem("m4y-language", "en");
    localStorage.setItem("m4y-analytics", "no");
  });
});

test("ordinary login returns home with hover and keyboard access to the customer sections", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto("/");
  await page.locator(".account-button").click();
  await login(page);
  await expect(page).toHaveURL(/\/$/);
  const trigger = page.getByRole("button", { name: "Profile options", exact: true });
  await expect(trigger).toBeVisible();
  await expect(page.locator(".account-button")).toHaveCount(0);
  await trigger.hover();
  const options = page.locator(".profile-options");
  await expect(options.getByRole("link")).toHaveText(["My Profile", "My Consultation", "My Orders"]);
  await expect(options.getByRole("button", { name: "Logout", exact: true })).toBeVisible();
  await page.screenshot({ path: "../.local/profile-menu-desktop.png" });
  await page.mouse.move(600, 300);
  await expect(options).toBeHidden();
  await trigger.focus();
  await trigger.press("ArrowDown");
  await expect(options.getByRole("link", { name: "My Profile", exact: true })).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(trigger).toBeFocused();
  await expect(trigger).toHaveAttribute("aria-expanded", "false");
  await trigger.press("Enter");
  await options.getByRole("link", { name: "My Profile", exact: true }).click();
  await expect(page).toHaveURL(/\/account$/);
  await expect(page.locator(".account-page h1")).toBeVisible();
  await page.getByRole("link", { name: "Back to website", exact: true }).click();
  await trigger.hover();
  await options.getByRole("link", { name: "My Consultation", exact: true }).click();
  await expect(page).toHaveURL(/\/account\?tab=bookings$/);
  await expect(page.getByRole("tab").nth(1)).toHaveAttribute("aria-selected", "true");
  await page.reload();
  await expect(page.getByRole("tab").nth(1)).toHaveAttribute("aria-selected", "true");
  await page.getByRole("link", { name: "Back to website", exact: true }).click();
  await trigger.hover();
  await options.getByRole("link", { name: "My Orders", exact: true }).click();
  await expect(page).toHaveURL(/\/account\?tab=orders$/);
  await expect(page.getByRole("tab").nth(2)).toHaveAttribute("aria-selected", "true");
  await expect(page.getByRole("link", { name: "My stone orders" })).toBeVisible();
  await page.goBack();
  await expect(trigger).toBeVisible();
  await page.reload();
  await expect(trigger).toBeVisible();
  await trigger.hover();
  await page.locator("#name").click();
  await expect(trigger).toHaveAttribute("aria-expanded", "false");
});

test.describe("touch profile navigation", () => {
  test.use({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true });

  test("protected consultation login, mobile taps, translated labels and logout", async ({ page, context }) => {
    await page.goto("/account?tab=bookings");
    await expect(page).toHaveURL(/\/login\?next=/);
    expect(new URL(page.url()).searchParams.get("next")).toBe("/account?tab=bookings");
    await login(page);
    await expect(page).toHaveURL(/\/account\?tab=bookings$/);
    await expect(page.getByRole("tab").nth(1)).toHaveAttribute("aria-selected", "true");
    await page.getByRole("link", { name: "Back to website", exact: true }).tap();
    const trigger = page.locator(".profile-trigger");
    const options = page.locator(".profile-options");
    await trigger.tap();
    await expect(options.getByRole("link", { name: "My Orders", exact: true })).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    await page.screenshot({ path: "../.local/profile-menu-mobile.png" });
    await trigger.tap();
    await expect(options).toBeHidden();
    await page.locator(".language-selector select").selectOption("hi");
    await trigger.tap();
    await expect(options.getByRole("link")).toHaveText(["मेरी प्रोफ़ाइल", "मेरा परामर्श", "मेरे ऑर्डर"]);
    await page.route("**/api/logout", (route) => route.fulfill({ status: 503, contentType: "application/json", body: JSON.stringify({ error: "Please retry logout" }) }));
    await options.getByRole("button", { name: "लॉगआउट", exact: true }).tap();
    await expect(options.getByRole("alert")).toBeVisible();
    expect((await context.request.get("/api/me")).status()).toBe(200);
    await page.unroute("**/api/logout");
    await options.getByRole("button", { name: "लॉगआउट", exact: true }).tap();
    await expect(page).toHaveURL(/\/$/);
    await expect(page.locator(".account-button")).toBeVisible();
    await expect(trigger).toHaveCount(0);
    expect((await context.request.get("/api/me")).status()).toBe(401);
    await page.reload();
    await expect(page.locator(".account-button")).toBeVisible();
  });
});
