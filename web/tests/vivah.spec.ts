import { test, expect } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem("m4y-language", "en");
    localStorage.setItem("m4y-analytics", "no");
  });
});

test("marriage windows use live calculations, shareable filters and browser history", async ({ page }) => {
  await page.goto("/vivah-muhurat?month=2026-11&place_id=delhi");
  await expect(page.getByRole("heading", { name: "Vivah Muhurat", exact: true })).toBeVisible();
  await expect(page.locator(".vivah-day").first()).toBeVisible({ timeout: 60000 });
  await expect(page).toHaveTitle("Vivah Muhurat | Mishra 4 You");
  await expect(page.locator(".vivah-windows time").first()).toContainText("2026");
  await expect(page.locator(".vivah-windows time").first()).toHaveAttribute("datetime", /2026-/);
  const count = await page.locator(".vivah-day").count();
  expect(count).toBeGreaterThan(0);
  await expect(page.locator(".vivah-lagna dd").first()).toHaveText(/^(Taurus|Gemini|Virgo|Libra|Sagittarius|Pisces)$/);
  await expect(page.locator(".vivah-method")).toContainText("complete method is still under review");
  await page.screenshot({ path: "../.local/vivah-desktop.png", fullPage: true });
  const audit = await new AxeBuilder({ page }).include(".vivah-main").analyze();
  expect(audit.violations).toEqual([]);
  await page.locator("#vivah-month").fill("2026-01");
  await expect(page).toHaveURL(/month=2026-01/);
  await expect(page.getByRole("heading", { name: "No windows meet the selected rules this month" })).toBeVisible({ timeout: 60000 });
  await page.locator(".vivah-excluded summary").click();
  await expect(page.locator(".vivah-excluded article")).toHaveCount(31);
  await expect(page.locator(".vivah-excluded article").first()).toContainText("Shukra combustion");
  await page.goBack();
  await expect(page.locator("#vivah-month")).toHaveValue("2026-11");
  await expect(page.locator(".vivah-day")).toHaveCount(count);
  await page.locator("#vivah-sunrise").selectOption("hindu");
  await expect(page).toHaveURL(/sunrise=hindu/);
  await expect(page.locator(".vivah-day").first()).toBeVisible();
  await page.reload();
  await expect(page.locator("#vivah-sunrise")).toHaveValue("hindu");
  await expect(page.locator(".vivah-day").first()).toBeVisible();
  await page.locator(".language-selector select").selectOption("hi");
  await expect(page.getByRole("heading", { name: "विवाह मुहूर्त", exact: true })).toBeVisible();
  await expect(page).toHaveTitle("विवाह मुहूर्त | Mishra 4 You");
  await expect(page.locator(".vivah-method")).toContainText("गुरु और शुक्र");
  await expect(page.locator(".vivah-count")).toContainText("प्रारंभिक समय");
  await expect(page.locator("#vivah-month")).toHaveValue("2026-11");
});

test("mobile city search supports keyboard selection and the landing menu opens Vivah Muhurat", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");
  await page.locator(".menu-toggle").click();
  await page.getByRole("button", { name: "Vivah Muhurat", exact: true }).click();
  await expect(page).toHaveURL(/\/vivah-muhurat$/);
  await page.locator("#vivah-month").fill("2026-11");
  await page.locator("#vivah-city").fill("Mumbai");
  await expect(page.getByRole("option", { name: "Mumbai, Maharashtra, India", exact: true })).toBeVisible();
  await page.locator("#vivah-city").press("ArrowDown");
  await page.locator("#vivah-city").press("Enter");
  await expect(page).toHaveURL(/place_id=mumbai/);
  await expect(page.locator("#vivah-city")).toHaveValue(/Mumbai/);
  await expect(page.locator(".vivah-day").first()).toBeVisible({ timeout: 60000 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await page.screenshot({ path: "../.local/vivah-mobile.png", fullPage: true });
});

test("failed calculations retry and changed filters hide old results", async ({ page }) => {
  let fail = true;
  await page.route("**/api/panchang/vivah?**", async (route) => {
    if (fail) await route.fulfill({ status: 503, json: { error: "Unavailable" } });
    else await route.fulfill({ json: {
      month: "2026-11", zone: "Asia/Kolkata", place: { id: "delhi", name: "New Delhi", hindi: "नई दिल्ली", zone: "Asia/Kolkata", lat: 28.6, lon: 77.2 }, settings: {},
      days: [{ date: "2026-11-26", windows: [{ start: "2026-11-26T07:00:00+05:30", end: "2026-11-26T08:00:00+05:30", tithi: "Shukla Dwitiya", nakshatra: "Mrigashira", yoga: "Priti", karana: "Bava", lagna: "Taurus" }], reasons: [] }],
    } });
  });
  await page.goto("/vivah-muhurat?month=2026-11&place_id=delhi");
  await expect(page.getByRole("alert")).toContainText("Marriage calculations unavailable");
  fail = false;
  await page.getByRole("button", { name: "Try again", exact: true }).click();
  await expect(page.locator(".vivah-day").first()).toBeVisible({ timeout: 60000 });
  fail = true;
  await page.getByRole("button", { name: "Next month", exact: true }).click();
  await expect(page.locator(".vivah-day")).toHaveCount(0);
  await expect(page.getByRole("alert")).toBeVisible();
});
