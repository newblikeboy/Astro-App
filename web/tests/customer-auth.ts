import type { Page } from "@playwright/test";

/** createCustomer registers a unique isolated browser-test account through the real customer form. */
export async function createCustomer(page: Page, email = `qa-${Date.now()}-${Math.random().toString(36).slice(2)}@example.test`, password = "Browser!123") {
  await page.locator(".auth-switch button").click();
  await page.locator("#customer-email").fill(email);
  await page.locator("#customer-password").fill(password);
  await page.locator("#customer-confirm-password").fill(password);
  await page.locator('.auth-card form button[type="submit"]').click();
  return { email, password };
}
