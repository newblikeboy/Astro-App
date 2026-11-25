import { test, expect } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

for (const language of ["en", "hi"] as const) {
  test(`email/password registration, errors and returning login (${language})`, async ({ page, context }) => {
    await page.addInitScript(language => {
      localStorage.setItem("m4y-language", language);
      localStorage.setItem("m4y-analytics", "no");
    }, language);
    const email = `login-${language}-${Date.now()}@example.test`;
    await page.goto("/login?next=%2Faccount%3Ftab%3Dprivacy");
    await expect(page.locator("#customer-email")).toBeVisible();
    await expect(page.locator('#phone,#otp,input[autocomplete="one-time-code"]')).toHaveCount(0);
    await page.locator(".auth-switch button").click();
    await page.locator("#customer-email").fill(email);
    await page.locator("#customer-password").fill("Browser!123");
    await page.locator("#customer-confirm-password").fill("Mismatch!123");
    await page.locator('.auth-card form button[type="submit"]').click();
    await expect(page.locator('.auth-card [role="alert"]')).toContainText(language === "en" ? "Both passwords must match." : "दोनों पासवर्ड एक जैसे होने चाहिए।");
    await page.locator("#customer-confirm-password").fill("Browser!123");
    await page.locator(".customer-password-field button").click();
    await expect(page.locator("#customer-password")).toHaveAttribute("type", "text");
    await page.locator(".customer-password-field button").click();
    for (const width of [1440, 390, 320]) {
      await page.setViewportSize({ width, height: 1000 });
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
      expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([]);
      await page.screenshot({ path: `../.local/customer-signup-${language}-${width}.png`, fullPage: true });
    }
    await page.locator('.auth-card form button[type="submit"]').click();
    await expect(page).toHaveURL(/\/account\?tab=privacy$/);
    await expect(page.locator(".workspace-heading")).toContainText(email);
    await page.reload();
    await expect(page.locator(".workspace-heading")).toContainText(email);
    const profile = await (await context.request.get("/api/me")).json();
    expect(JSON.stringify(profile)).not.toMatch(/password_hash|Browser!123/);
    const loggedOut = await context.request.post("/api/logout", { headers: { "X-CSRF-Token": profile.csrf } });
    expect(loggedOut.ok()).toBe(true);
    await page.goto("/login?next=%2Faccount%3Ftab%3Dorders");
    await page.locator("#customer-email").fill(email.toUpperCase());
    await page.locator("#customer-password").fill("Wrong!123");
    await page.locator('.auth-card form button[type="submit"]').click();
    await expect(page.locator('.auth-card [role="alert"]')).toContainText(language === "en" ? "The email or password is incorrect." : "ईमेल या पासवर्ड सही नहीं है।");
    await page.locator("#customer-password").fill("Browser!123");
    await page.locator('.auth-card form button[type="submit"]').click();
    await expect(page).toHaveURL(/\/account\?tab=orders$/);
    expect((await (await context.request.get("/api/me")).json()).user.id).toBe(profile.user.id);
    await page.reload();
    await expect(page.locator(".workspace-heading")).toContainText(email);
  });
}
