import { test, expect } from "@playwright/test";
import { customerReturn } from "../src/routing";

test("customer return paths stay inside the customer area", () => {
  for (const path of ["/", "/account", "/kundli", "/account/reports/rpt_123", "/account?tab=bookings", "/account?tab=orders", "/account?tab=privacy"])
    expect(customerReturn(path)).toBe(path);
  for (const path of [
    null,
    "https://other.example",
    "//other.example",
    "javascript:alert(1)",
    "/admin",
    "/admin/login",
    "/login",
    "/account/reports/../../admin",
    "/account/reports/rpt_123?next=/admin",
    "/account?tab=bookings&next=/admin",
    "/account?tab=unknown",
  ])
    expect(customerReturn(path)).toBe("/account");
});

test("direct protected URLs, reload and browser history use separate pages", async ({
  page,
}) => {
  await page.addInitScript(() => {
    localStorage.setItem("m4y-language", "en");
    localStorage.setItem("m4y-analytics", "no");
  });
  await page.goto("/");
  await expect(page.locator(".account-button")).toHaveAttribute(
    "href",
    "/login",
  );
  await expect(page.locator('a[href^="/admin"]')).toHaveCount(0);
  await page.locator("#name").fill("Preserved route draft");
  await page.locator(".account-button").click();
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.locator(".auth-card")).toBeVisible();
  await expect(page.locator(".site-header,.site-footer,dialog")).toHaveCount(0);
  await expect(page).toHaveTitle(/Log in to your account/);
  await page.reload();
  await expect(page.locator("#phone")).toBeVisible();
  await page.goBack();
  await expect(page.locator("#name")).toHaveValue("Preserved route draft");
  await page.goForward();
  await expect(page.locator(".auth-card")).toBeVisible();
  await page.goto("/account/reports/rpt_unowned");
  await expect(page).toHaveURL(/\/login\?next=/);
  expect(new URL(page.url()).searchParams.get("next")).toBe(
    "/account/reports/rpt_unowned",
  );
  await expect(page.locator(".kundli-workspace")).toHaveCount(0);
  await page.goto("/admin");
  await expect(page).toHaveURL(/\/admin\/login$/);
  await expect(page.locator("#admin-password")).toBeVisible();
  await expect(
    page.locator("#phone,.site-header,.site-footer,.consent"),
  ).toHaveCount(0);
  await page.reload();
  await expect(page.locator("#admin-password")).toBeVisible();
  await page.goto("/missing-page");
  await expect(
    page.getByRole("heading", { name: "Page not found" }),
  ).toBeVisible();
});

test("customer and administrator sessions do not grant or revoke each other's access", async ({
  page,
  context,
}) => {
  await context.addInitScript(() => {
    localStorage.setItem("m4y-language", "en");
    localStorage.setItem("m4y-analytics", "no");
  });
  const phone = "+9198" + String(Date.now()).slice(-8);
  /** customerLogin logs the isolated browser-test customer in using the explicit development OTP flow. */
  async function customerLogin() {
    await page.locator("#phone").fill(phone);
    await page.getByRole("button", { name: "Get OTP", exact: true }).click();
    const code = await page.locator(".auth-card .notice b").textContent();
    await page.locator("#otp").fill(code!);
    await page.getByRole("button", { name: "Verify", exact: true }).click();
    await expect(page).toHaveURL(/\/account$/);
    await expect(
      page.locator(".saved-reports,.account-page h1").first(),
    ).toBeVisible();
  }
  await page.goto("/account");
  await expect(page).toHaveURL(/\/login\?next=/);
  await customerLogin();
  expect((await context.request.get("/api/me")).status()).toBe(200);
  expect((await context.request.get("/api/admin")).status()).toBe(401);
  const admin = await context.newPage();
  await admin.goto("/admin");
  await expect(admin).toHaveURL(/\/admin\/login$/);
  await admin.locator("#admin-password").fill("local-admin-change-me");
  await admin.getByRole("button", { name: "Open admin", exact: true }).click();
  await expect(admin).toHaveURL(/\/admin$/);
  await expect(admin.locator(".admin-workspace")).toBeVisible();
  await page.getByRole("button", { name: "Log out", exact: true }).click();
  await expect(page).toHaveURL(/\/login$/);
  expect((await context.request.get("/api/me")).status()).toBe(401);
  await admin.reload();
  await expect(admin.locator(".admin-workspace")).toBeVisible();
  expect((await context.request.get("/api/admin")).status()).toBe(200);
  await page.goto("/login?next=https%3A%2F%2Fother.example");
  await customerLogin();
  await admin.getByRole("button", { name: "Log out", exact: true }).click();
  await expect(admin).toHaveURL(/\/admin\/login$/);
  expect((await context.request.get("/api/admin")).status()).toBe(401);
  await page.reload();
  await expect(page).toHaveURL(/\/account$/);
  await expect(page.locator(".account-page h1")).toBeVisible();
  expect((await context.request.get("/api/me")).status()).toBe(200);
});
