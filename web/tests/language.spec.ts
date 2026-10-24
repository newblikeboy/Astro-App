import { test, expect } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

test("English is the default and Hindi selection persists without losing birth details", async ({
  page,
}) => {
  await page.goto("/");
  const selector = page.getByRole("combobox", { name: "Language / भाषा" });
  await expect(selector).toHaveValue("en");
  await expect(page.locator("html")).toHaveAttribute("lang", "en");
  await expect(
    page.getByRole("heading", {
      name: /Your Kundli.*A new perspective/,
    }),
  ).toBeVisible();
  expect(await page.locator("main").innerText()).not.toMatch(/[\u0900-\u097f]/);
  await page
    .getByRole("button", { name: "Essential only", exact: true })
    .click();
  await page.locator("#name").fill("Aarav Sharma");
  await page.locator("#birth-date").fill("1995-08-15");
  await page.locator("#birth-time").fill("10:30");
  await page.locator("#birthplace").fill("Delhi");
  await page.getByRole("button", { name: /^New Delhi, Delhi, India/ }).click();
  await selector.selectOption("hi");
  await expect(page.locator("html")).toHaveAttribute("lang", "hi");
  await expect(
    page.getByRole("heading", { name: /आपकी कुंडली। समझ की नई दिशा/ }),
  ).toBeVisible();
  await expect(page.locator("#name")).toHaveValue("Aarav Sharma");
  await expect(page.locator("#birth-date")).toHaveValue("1995-08-15");
  await expect(page.locator("#birth-time")).toHaveValue("10:30");
  await expect(page.locator("#birthplace")).toHaveValue(/नई दिल्ली/);
  await page.reload();
  await expect(selector).toHaveValue("hi");
  await expect(page.locator("#name")).toHaveValue("Aarav Sharma");
  await selector.selectOption("en");
  await expect(page.locator("#birthplace")).toHaveValue(
    "New Delhi, Delhi, India",
  );
  let calculations = 0;
  page.on("request", (r) => {
    if (r.url().endsWith("/api/kundli")) calculations++;
  });
  await page
    .getByRole("button", { name: "Create my free Kundli", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Aarav Sharma, your story." }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Your planetary positions" }),
  ).toBeVisible();
  await expect(
    page.getByRole("rowheader", { name: "Sun", exact: true }),
  ).toBeVisible();
  await selector.selectOption("hi");
  await expect(
    page.getByRole("heading", { name: /Aarav Sharma, आपकी कहानी/ }),
  ).toBeVisible();
  await expect(
    page.getByRole("rowheader", { name: "सूर्य", exact: true }),
  ).toBeVisible();
  expect(calculations).toBe(1);
  await selector.selectOption("en");
  await page
    .getByRole("button", { name: "Save to my account", exact: true })
    .click();
  await expect(page).toHaveURL(/\/login\?next=/);
  await expect(page.locator(".auth-card")).toBeVisible();
  await expect(page.getByLabel("Mobile number", { exact: true })).toBeVisible();
  await page.getByLabel("Mobile number", { exact: true }).fill("9876543210");
  const dialog = page.locator(".auth-page");
  await page
    .getByRole("combobox", { name: "Language / भाषा" })
    .selectOption("hi");
  await expect(dialog.getByLabel("मोबाइल नंबर", { exact: true })).toHaveValue(
    "9876543210",
  );
  await page
    .getByRole("combobox", { name: "Language / भाषा" })
    .selectOption("en");
  await page.goBack();
  await expect(page.locator(".kundli-summary")).toBeVisible();
});

test("English layouts and language selector work on phone, tablet and desktop", async ({
  page,
}) => {
  for (const [name, width, height] of [
    ["phone", 390, 844],
    ["tablet", 768, 1024],
    ["desktop", 1440, 1000],
  ] as const) {
    await page.setViewportSize({ width, height });
    await page.goto("/");
    await expect(
      page.getByRole("combobox", { name: "Language / भाषा" }),
    ).toBeVisible();
    await expect(
      page.getByRole("heading", { name: /Your Kundli.*A new perspective/ }),
    ).toBeVisible();
    await page
      .getByRole("button", { name: "Essential only", exact: true })
      .click({ timeout: 500 })
      .catch(() => {});
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBeTruthy();
    await page.screenshot({
      path: `../.local/english-${name}.png`,
      fullPage: true,
    });
    const audit = await new AxeBuilder({ page })
      .withTags(["wcag2a", "wcag2aa", "wcag21aa"])
      .analyze();
    expect(
      audit.violations.filter(
        (v) => v.impact === "serious" || v.impact === "critical",
      ),
    ).toEqual([]);
  }
});
