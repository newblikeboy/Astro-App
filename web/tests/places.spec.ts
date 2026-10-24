import { test, expect } from "@playwright/test";

test("nationwide birthplace search, saved selection and Hindi aliases", async ({
  page,
}) => {
  await page.goto("/");
  await page
    .getByRole("button", { name: "Essential only", exact: true })
    .click();
  await page.locator("#name").fill("Nationwide Search");
  await page.locator("#birth-date").fill("1995-08-15");
  await page.locator("#birth-time").fill("10:30");
  const input = page.locator("#birthplace");
  for (const [query, match] of [
    ["Kochi Kerala", /Kochi.*Kerala/],
    ["Guwahati", /Guwahati.*Assam/],
    ["Leh Ladakh", /Leh.*Ladakh/],
  ] as const) {
    await input.fill(query);
    await expect(
      page.getByRole("listbox").getByRole("button").first(),
    ).toHaveText(match);
  }
  await input.fill("Gorakhpur Uttar Pradesh");
  await page.getByRole("listbox").getByRole("button").first().click();
  const selected = await input.inputValue();
  expect(selected).toMatch(/Gorakhpur.*Uttar Pradesh/);
  await page.reload();
  await expect(input).toHaveValue(selected);
  const selector = page.getByRole("combobox", { name: "Language / भाषा" });
  await selector.selectOption("hi");
  await expect(input).toHaveValue(/गोरखपुर/);
  await input.fill("गोरखपुर");
  await page.getByRole("listbox").getByRole("button").first().click();
  await selector.selectOption("en");
  await expect(input).toHaveValue(selected);
  await input.fill("zzzz-no-such-place-zzzz");
  await expect(page.getByRole("listbox")).toContainText("No matching location");
  await input.fill("Gorakhpur Uttar Pradesh");
  await page.getByRole("listbox").getByRole("button").first().click();
  await page
    .getByRole("button", { name: "Create my free Kundli", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Nationwide Search, your story." }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Your planetary positions" }),
  ).toBeVisible();
});

test("stale and failed location requests never show an incorrect match", async ({
  page,
}) => {
  await page.goto("/");
  const input = page.locator("#birthplace");
  let release!: () => void;
  const paused = new Promise<void>((resolve) => {
    release = resolve;
  });
  await page.route("**/api/places?q=Kochi", async (route) => {
    await paused;
    await route.fulfill({
      json: [
        {
          id: "stale",
          name: "Stale Kochi",
          hindi: "",
          zone: "Asia/Kolkata",
          lat: 1,
          lon: 1,
        },
      ],
    });
  });
  const pending = page.waitForRequest("**/api/places?q=Kochi");
  await input.fill("Kochi");
  await pending;
  await input.fill("Guwahati");
  await expect(
    page.getByRole("listbox").getByRole("button").first(),
  ).toHaveText(/Guwahati.*Assam/);
  release();
  await expect(page.getByRole("listbox")).not.toContainText("Stale Kochi");
  await page.route("**/api/places?q=Failure", (route) => route.abort());
  await input.fill("Failure");
  await expect(page.getByRole("listbox")).toContainText(
    "Location search is unavailable",
  );
  await input.fill("Guwahati");
  await expect(
    page.getByRole("listbox").getByRole("button").first(),
  ).toHaveText(/Guwahati.*Assam/);
});
