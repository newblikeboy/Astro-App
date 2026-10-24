import { test, expect } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import fs from "node:fs";
test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem("m4y-language", "hi"));
});
test("responsive landing, keyboard and accessibility", async ({ page }) => {
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  for (const [name, width, height] of [
    ["phone", 390, 844],
    ["tablet", 768, 1024],
    ["desktop", 1440, 1000],
  ] as const) {
    await page.setViewportSize({ width, height });
    await page.goto("/");
    await expect(
      page.getByRole("heading", { name: "आपकी कुंडली। समझ की नई दिशा।" }),
    ).toBeVisible();
    await page
      .getByRole("button", { name: "केवल आवश्यक" })
      .click({ timeout: 500 })
      .catch(() => {});
    await page.evaluate(() => document.fonts.ready);
    await page.screenshot({
      path: `../.local/landing-${name}.png`,
      fullPage: true,
    });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBeTruthy();
    const accessibility = await new AxeBuilder({ page })
      .withTags(["wcag2a", "wcag2aa", "wcag21aa"])
      .analyze();
    fs.writeFileSync(
      `../.local/axe-${name}.json`,
      JSON.stringify(accessibility.violations, null, 2),
    );
    expect(
      accessibility.violations.filter(
        (v) => v.impact === "critical" || v.impact === "serious",
      ),
    ).toEqual([]);
  }
  expect(errors).toEqual([]);
});
test("guest Kundli, OTP, private save, paid demo PDF and independent booking", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByRole("button", { name: "केवल आवश्यक" }).click();
  await page.locator("#name").fill("आरव शर्मा");
  await page.locator("#birth-date").fill("1995-08-15");
  await page.locator("#birth-time").fill("10:30");
  await page.locator("#birthplace").fill("Varanasi");
  await page.getByRole("button", { name: /वाराणसी, उत्तर प्रदेश/ }).click();
  await page
    .getByRole("button", { name: "मेरी निःशुल्क कुंडली बनाएँ" })
    .click();
  await expect(
    page.getByRole("heading", { name: "आरव शर्मा, आपकी कहानी।" }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "आपकी ग्रह-सारणी" }),
  ).toBeVisible();
  await page.screenshot({
    path: "../.local/report-desktop.png",
    fullPage: true,
  });
  // Phone readers can choose every report section without discovering a hidden horizontal tab.
  await page.setViewportSize({ width: 390, height: 844 });
  const reportSection = page.locator(".kundli-section-select select");
  await expect(reportSection).toBeVisible();
  await expect(reportSection.locator("option")).toHaveCount(9);
  await reportSection.selectOption("dasha");
  await expect(page.locator("#kundli-panel")).toBeVisible();
  await reportSection.selectOption("overview");
  await page.screenshot({ path: "../.local/report-navigation-mobile.png" });
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.getByRole("button", { name: "खाते में सहेजें" }).click();
  await expect(page).toHaveURL(/\/login\?next=/);
  await page.reload();
  await page.locator("#phone").fill("+9198" + String(Date.now()).slice(-8));
  await page.getByRole("button", { name: "OTP पाएँ", exact: true }).click();
  const text = await page.locator(".auth-card .notice b").textContent();
  await page.locator("#otp").fill(text!);
  await page
    .getByRole("button", { name: "सत्यापित करें", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "खाते में सहेजी गई" }),
  ).toBeVisible();
  await expect(page).toHaveURL(/\/account\/reports\//);
  await page.reload();
  await page.getByRole("button", { name: "उपाय PDF", exact: true }).click();
  await page
    .getByRole("button", { name: "ऑर्डर बनाएँ और भुगतान देखें" })
    .click();
  await page.getByRole("button", { name: "डेमो भुगतान सफल करें" }).click();
  await expect(
    page.getByRole("heading", { name: "सहेजी गई कुंडलियाँ" }),
  ).toBeVisible();
  const downloadButton = page.getByRole("link", { name: "PDF डाउनलोड" });
  await expect(downloadButton).toBeVisible({ timeout: 80000 });
  const d = page.waitForEvent("download");
  await downloadButton.click();
  const download = await d;
  await download.saveAs("../.local/verified-hindi-report.pdf");
  await page.getByRole("tab", { name: "मेरे परामर्श" }).click();
  await page.getByRole("button", { name: "नया परामर्श" }).click();
  await page.locator("dialog input[type=radio]").first().check();
  await page
    .getByRole("button", { name: "ऑर्डर बनाएँ और भुगतान देखें" })
    .click();
  await page.getByRole("button", { name: "डेमो भुगतान सफल करें" }).click();
  await page.getByRole("tab", { name: "मेरे परामर्श" }).click();
  await expect(page.locator(".status-confirmed").first()).toBeVisible({
    timeout: 20000,
  });
  await page.screenshot({
    path: "../.local/account-desktop.png",
    fullPage: true,
  });
  await page
    .getByRole("button", { name: "समय बदलें", exact: true })
    .first()
    .click();
  await page.locator("dialog input[type=radio]").first().check();
  await page.getByRole("button", { name: "नया समय सुरक्षित करें" }).click();
  await expect(page.locator(".status-confirmed").first()).toBeVisible({
    timeout: 20000,
  });
  await page
    .getByRole("button", { name: "रद्द करने का अनुरोध" })
    .first()
    .click();
  await expect(page.locator(".status-cancelled").first()).toBeVisible({
    timeout: 20000,
  });
});
test("unknown birth time never renders fabricated chart", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "केवल आवश्यक" }).click();
  await page.locator("#name").fill("सीमा मिश्रा");
  await page.locator("#birth-date").fill("1990-01-01");
  await page.getByLabel("जन्म समय नहीं पता", { exact: true }).check();
  await page.locator("#birthplace").fill("Delhi");
  await page.getByRole("button", { name: /नई दिल्ली, भारत/ }).click();
  await page
    .getByRole("button", { name: "मेरी निःशुल्क कुंडली बनाएँ" })
    .click();
  await expect(
    page.getByRole("heading", { name: "सही जन्म समय के बिना गणना सीमित है।" }),
  ).toBeVisible();
  await expect(page.locator(".report-chart")).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "उपाय PDF", exact: true }),
  ).toBeDisabled();
});
test("admin settings, rule editing and accounting load behind separate authentication", async ({
  page,
}) => {
  await page.goto("/admin");
  await expect(page).toHaveURL(/\/admin\/login$/);
  await expect(page.locator(".site-header,.site-footer,.consent")).toHaveCount(
    0,
  );
  await expect(
    page.getByRole("heading", { name: "संचालक लॉग इन" }),
  ).toBeVisible();
  await page.locator("#admin-password").fill("local-admin-change-me");
  await page.getByRole("button", { name: "व्यवस्थापन खोलें" }).click();
  await expect(
    page.getByRole("heading", { name: "व्यवसाय अवलोकन", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "लेखा और खर्च", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "मासिक साझेदारी रिपोर्ट" }),
  ).toBeVisible();
  await page.screenshot({
    path: "../.local/admin-finance.png",
    fullPage: true,
  });
  await page
    .getByRole("button", { name: "व्याख्याएँ और उपाय", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "व्याख्या / उपाय संपादक" }),
  ).toBeVisible();
});
