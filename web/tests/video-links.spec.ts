import { test, expect } from "@playwright/test";
import type { Page } from "@playwright/test";
import { customerReturn } from "../src/routing";
import { videoReadings, videoSign } from "../src/video-links";
import type { VideoLink } from "../src/video-links";
import type { Report, Rule } from "../src/types";

/** enterBirth fills the real public birth form and chooses a trusted location from server search. */
async function enterBirth(page: Page, unknown = false) {
  await page.locator("#name").fill("Video visitor");
  await page.locator("#birth-date").fill("1995-08-15");
  if (unknown) await page.locator(".birth-form input[type=checkbox]").check();
  else await page.locator("#birth-time").fill("10:30");
  await page.locator("#birthplace").fill("Varanasi");
  await page.getByRole("button", { name: /Varanasi, Uttar Pradesh/ }).first().click();
  await page.locator(".birth-form button[type=submit]").click();
}

test("video return paths and focused rules preserve chart matching and approvals", () => {
  for (const path of ["/?video=karj-rahat", "/kundli?video=mesh-raashi", "/account/reports/rpt_123?video=promotion-video"]) expect(customerReturn(path)).toBe(path);
  for (const path of ["/admin?video=karj-rahat", "/kundli?video=../../admin", "/kundli?video=karj&next=/admin", "/kundli?video=xx", "/kundli?video=" + "a".repeat(65)]) expect(customerReturn(path)).toBe("/account");
  const link: VideoLink = { slug: "mesh-raashi", title: "Mesh", topic: "zodiac", sign: 1, sign_basis: "moon", rule_ids: ["selected", "draft", "demo"], enabled: true, version: 1 };
  const rule = { id: "selected", status: "approved", approver: "Guruji", approved_at: "2026-10-22T00:00:00Z", demo: false } as Rule;
  const report = { chart: { lagna: { rashi_number: 3 }, planets: [{ name: "Moon", rashi_number: 1 }, { name: "Sun", rashi_number: 2 }] }, rules: [rule, { ...rule, id: "unrelated" }, { ...rule, id: "draft", status: "draft" }, { ...rule, id: "demo", demo: true }] } as Report;
  expect(videoReadings(report, link)).toEqual([rule]);
  expect(videoSign(report, { ...link, sign_basis: "lagna" })).toBe(3);
  expect(videoSign(report, { ...link, sign_basis: "sun" })).toBe(2);
  expect(videoReadings(report, { ...link, sign: 2 })).toEqual([]);
  expect(videoReadings({ ...report, chart: null }, link)).toEqual([]);
});

test("admin publishes a topic link; a visitor gets only matching guidance and can continue to full Kundli and purchases", async ({ page, context }) => {
  await context.addInitScript(() => {
    localStorage.setItem("m4y-language", "en");
    localStorage.setItem("m4y-analytics", "no");
  });
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto("/admin/login");
  await page.locator("#admin-password").fill("local-admin-change-me");
  await page.getByRole("button", { name: "Open admin", exact: true }).click();
  await expect(page.locator(".admin-workspace")).toBeVisible();
  const admin = await (await context.request.get("/api/admin")).json();
  const sample = await (await context.request.post("/api/kundli", { data: { name: "Fixture", date: "1995-08-15", time: "10:30", place_id: "varanasi", unknown_time: false } })).json();
  expect(sample.chart).toBeTruthy();
  const moonSign = sample.chart.planets.find((planet: { name: string }) => planet.name === "Moon").rashi_number;
  const suffix = String(Date.now());
  const matchID = "video-debt-" + suffix;
  const missID = "video-other-" + suffix;
  for (const [id, sign, topic, english] of [
    [matchID, moonSign, "कर्ज़ पर स्वीकृत मार्गदर्शन", "This approved debt guidance matches your Moon placement."],
    [missID, (moonSign % 12) + 1, "दूसरी राशि की व्याख्या", "This reading must not appear for this chart."],
  ] as const) {
    const response = await context.request.post("/api/admin/rules", { headers: { "X-CSRF-Token": admin.csrf }, data: { id, version: 1, planet: "Moon", sign, topic, hindi: "यह परीक्षण के लिए गुरुजी की स्वीकृत व्याख्या है।", english, mantra: "", gemstone: "", priority: 500, conflict_group: "", category: "career", status: "approved", approver: "QA Guruji", demo: false } });
    expect(response.status()).toBe(200);
  }
  await page.getByRole("button", { name: "Video topic links", exact: true }).click();
  await page.getByLabel("Link name (URL)", { exact: true }).fill("debt-" + suffix);
  await page.getByLabel("Video title", { exact: true }).fill("कर्ज़ से परेशान हैं?");
  await expect(page.getByLabel(/English title \(optional\)|Introduction \/ Hindi introduction|English introduction \(optional\)|YouTube video URL \(optional\)/)).toHaveCount(0);
  await page.getByLabel(/कर्ज़ पर स्वीकृत मार्गदर्शन/).check();
  await page.getByLabel(/दूसरी राशि की व्याख्या/).check();
  await page.getByRole("button", { name: "Save video link", exact: true }).click();
  await expect(page.getByRole("status")).toContainText("saved as a draft");
  expect((await context.request.get("/api/video-links/debt-" + suffix)).status()).toBe(404);
  await page.getByLabel("Publish this link", { exact: true }).check();
  await page.getByRole("button", { name: "Save video link", exact: true }).click();
  await expect(page.getByRole("status")).toContainText("published");
  const url = await page.getByLabel("Share link", { exact: true }).inputValue();
  await page.screenshot({ path: "../.local/video-links-admin.png", fullPage: true });
  await page.setViewportSize({ width: 390, height: 844 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  await page.getByRole("button", { name: "New link", exact: true }).click();
  await page.getByLabel("Video topic", { exact: true }).selectOption("zodiac");
  await expect(page.getByLabel("Sign reference", { exact: true })).toHaveValue("moon");
  await page.getByLabel("Sign reference", { exact: true }).selectOption("sun");
  await expect(page.getByLabel("Sign reference", { exact: true })).toHaveValue("sun");
  await page.getByLabel("Sign reference", { exact: true }).selectOption("lagna");
  await expect(page.getByLabel("Sign reference", { exact: true })).toHaveValue("lagna");
  await page.setViewportSize({ width: 1440, height: 1000 });

  await page.goto(url);
  await expect(page.getByRole("heading", { name: "कर्ज़ से परेशान हैं?", exact: true })).toBeVisible();
  await expect(page.locator(".hero,.home-gemstones")).toHaveCount(0);
  await enterBirth(page);
  await expect(page).toHaveURL(new RegExp("/kundli\\?video=debt-" + suffix + "$"));
  await expect(page.locator(".video-chart-card svg")).toBeVisible();
  await expect(page.getByText("This approved debt guidance matches your Moon placement.", { exact: true })).toBeVisible();
  await expect(page.getByText("This reading must not appear for this chart.", { exact: true })).toHaveCount(0);
  await expect(page.locator(".kundli-workspace")).toHaveCount(0);
  await page.screenshot({ path: "../.local/video-reading-desktop.png", fullPage: true });
  await page.reload();
  await expect(page.getByText("This approved debt guidance matches your Moon placement.", { exact: true })).toBeVisible();
  await page.setViewportSize({ width: 390, height: 844 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  await page.screenshot({ path: "../.local/video-reading-mobile.png", fullPage: true });
  await page.locator(".language-selector select").selectOption("hi");
  await expect(page.getByRole("heading", { name: "कर्ज़ से परेशान हैं?", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "पूरी कुंडली देखें", exact: true })).toBeVisible();
  await page.locator(".language-selector select").selectOption("en");
  await page.getByRole("button", { name: "View full Kundli", exact: true }).click();
  await expect(page).toHaveURL(/\/kundli$/);
  await expect(page.locator(".kundli-workspace")).toBeVisible();
  await page.goBack();
  await expect(page.locator(".video-reading")).toBeVisible();
  await page.getByRole("button", { name: "Buy personal remedies PDF", exact: true }).click();
  await expect(page).toHaveURL(/\/login\?next=/);
  expect(new URL(page.url()).searchParams.get("next")).toBe("/kundli?video=debt-" + suffix);
  await page.reload();
  await page.locator("#phone").fill("+9196" + suffix.slice(-8));
  await page.getByRole("button", { name: "Get OTP", exact: true }).click();
  await page.locator("#otp").fill((await page.locator(".auth-card .notice b").textContent())!);
  await page.getByRole("button", { name: "Verify", exact: true }).click();
  await expect(page).toHaveURL(new RegExp("/account/reports/[^?]+\\?video=debt-" + suffix));
  await expect(page.locator("dialog")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.locator(".video-reading")).toBeVisible();
  await page.getByRole("button", { name: "Talk with Guruji", exact: true }).click();
  await expect(page.locator("dialog")).toBeVisible();
  await page.keyboard.press("Escape");

  // The merchant can stop a circulated link without deleting the saved report.
  const current = await (await context.request.get("/api/video-links/debt-" + suffix)).json();
  expect((await context.request.put("/api/admin/video-links/debt-" + suffix, { headers: { "X-CSRF-Token": admin.csrf }, data: { ...current, enabled: false } })).status()).toBe(200);
  await page.goto(url);
  await expect(page.getByRole("heading", { name: "This video link is unavailable.", exact: true })).toBeVisible();
  await expect(page.locator(".birth-form,.video-reading")).toHaveCount(0);
});

test("zodiac mismatch and unknown-time visitors receive no invented topic reading", async ({ page, context }) => {
  await context.addInitScript(() => { localStorage.setItem("m4y-language", "en"); localStorage.setItem("m4y-analytics", "no"); });
  const auth = await (await context.request.post("/api/admin/login", { data: { password: "local-admin-change-me", totp: "" } })).json();
  const suffix = String(Date.now());
  const sample = await (await context.request.post("/api/kundli", { data: { name: "Fixture", date: "1995-08-15", time: "10:30", place_id: "varanasi", unknown_time: false } })).json();
  const moon = sample.chart.planets.find((planet: { name: string }) => planet.name === "Moon").rashi_number;
  const link: VideoLink = { slug: "zodiac-" + suffix, title: "Zodiac video", topic: "zodiac", sign: (moon % 12) + 1, sign_basis: "moon", rule_ids: [], enabled: true, version: 0 };
  expect((await context.request.put("/api/admin/video-links/" + link.slug, { headers: { "X-CSRF-Token": auth.csrf }, data: link })).status()).toBe(200);
  await page.goto("/?video=" + link.slug);
  await enterBirth(page);
  await expect(page.getByRole("status")).toContainText("sign-specific reading is not applied");
  await expect(page.locator(".video-approved-reading")).toHaveCount(0);
  await page.getByRole("button", { name: "Edit birth details", exact: true }).click();
  await page.locator(".birth-form input[type=checkbox]").check();
  await page.locator(".birth-form button[type=submit]").click();
  await expect(page.getByRole("heading", { name: "An accurate birth time is needed", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Buy personal remedies PDF", exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Talk with Guruji", exact: true })).toBeVisible();
});
