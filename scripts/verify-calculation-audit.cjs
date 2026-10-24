// Focused regression checks against an isolated, read-only audit preview.
const assert = require("node:assert/strict");
const fs = require("node:fs");
const { chromium } = require("../web/node_modules/playwright");

(async () => {
  const base = process.env.PANCHANG_QA_URL || "http://127.0.0.1:18083";
  assert.equal(new URL(base).hostname, "127.0.0.1");
  assert.equal(new URL(base).port, "18083");
  /** get fetches a read-only audit endpoint and rejects an unsuccessful response. */
  const get = async (path) => {
    const response = await fetch(base + path);
    assert(response.ok, `${path}: ${response.status}`);
    return response.json();
  };
  /** daily requests the fixed New Delhi Panchang for a calculation-audit date. */
  const daily = (date) => get(`/api/panchang?date=${date}&place_id=delhi`);
  const reference = JSON.parse(
    fs.readFileSync(
      "internal/swisseph/testdata/jpl-panchang-reference.json",
      "utf8",
    ),
  );
  const result = { passed: false, threeDays: [], browserViews: [], errors: [] };
  for (const date of ["2026-10-18", "2026-10-19", "2026-10-20"]) {
    const p = await daily(date);
    assert.equal(p.settings.engine, "m4y-panchang-v2");
    assert.equal(p.settings.details_rules, "m4y-panchang-rules-v2");
    const want = reference.cases.find((c) => c.date === date).tithi;
    const residual = Math.min(
      ...want
        .filter((r) => r.number === p.tithi[0].number)
        .map(
          (r) =>
            Math.abs(Date.parse(r.end) - Date.parse(p.tithi[0].end)) / 1000,
        ),
    );
    assert(residual < 2);
    for (const event of [p.details.moonrise, p.details.moonset].filter(
      Boolean,
    )) {
      assert(
        Date.parse(event) >= Date.parse(p.sunrise) &&
          Date.parse(event) < Date.parse(p.next_sunrise),
      );
    }
    result.threeDays.push({
      date,
      firstTithiEnd: p.tithi[0].end,
      jplResidualSeconds: residual,
      moonset: p.details.moonset,
    });
  }
  assert.match(result.threeDays[2].moonset, /^2026-10-21T01:25:/);
  const winter = await daily("2024-01-01");
  assert.equal(winter.details.calendar.vikram, 2080);
  assert.equal(winter.details.calendar.shaka, 1945);
  let missing;
  for (let d = 1; d <= 31; d++) {
    const p = await daily(`2026-10-${String(d).padStart(2, "0")}`);
    if (!p.details.moonrise || !p.details.moonset) {
      missing = p;
      break;
    }
  }
  assert(missing, "No missing lunar event case was exercised");
  result.noMoonEventDate = missing.date;
  const config = await get("/api/config");
  const browser = await chromium.launch({
    executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe",
    headless: true,
  });
  try {
    const context = await browser.newContext({
      viewport: { width: 1440, height: 1000 },
      timezoneId: "America/Los_Angeles",
      reducedMotion: "reduce",
    });
    await context.addInitScript(() => {
      localStorage.setItem("m4y-language", "en");
      localStorage.setItem("m4y-analytics", "no");
      sessionStorage.setItem(
        "m4y-birth",
        JSON.stringify({
          name: "Audit Fixture",
          date: "1995-08-15",
          time: "10:30",
          place_id: "delhi",
          unknown_time: false,
        }),
      );
    });
    const page = await context.newPage();
    page.on("pageerror", (error) => result.errors.push(error.message));
    await page.route("**/api/**", (r) =>
      r.request().method() === "GET" ? r.continue() : r.abort(),
    );
    await page.clock.setFixedTime(new Date("2026-10-19T00:30:00Z"));
    await page.goto(base);
    await page.waitForFunction(
      () => document.querySelector("#birth-date")?.max === "2026-10-19",
    );
    /** selectDate changes the daily control and waits for the corresponding calculation response. */
    async function selectDate(date) {
      if ((await page.locator("#daily-date").inputValue()) === date) return;
      const response = page.waitForResponse(
        (r) =>
          new URL(r.url()).pathname === "/api/panchang" &&
          new URL(r.url()).searchParams.get("date") === date,
      );
      await page.locator("#daily-date").fill(date);
      assert((await response).ok());
      await page.waitForFunction(
        () =>
          document.querySelector(".daily-sheet")?.getAttribute("aria-busy") ===
          "false",
      );
    }
    for (const language of ["en", "hi"]) {
      await page.locator(".language-selector select").selectOption(language);
      for (const width of [339, 1440]) {
        await page.setViewportSize({ width, height: 1000 });
        await selectDate("2024-01-01");
        await page.locator("#panchang-detail-calendar").click();
        const facts = await page
          .locator(".panchang-calendar-facts")
          .innerText();
        assert.match(facts, /2080/);
        assert.match(facts, /1945/);
        if (language === "en") assert.match(facts, /Mean Lahiri ayanamsa/);
        await selectDate(missing.date);
        const text = await page.locator(".daily-lunar-facts").innerText();
        if (language === "en")
          assert.match(text, /No Moon(rise|set) in this sunrise day/);
        else assert.match(text, /इस सूर्योदय से अगले सूर्योदय तक/);
        await page.evaluate(() => document.fonts.ready);
        const overflow = await page.locator("#panchang").evaluate((el) => ({
          page: document.documentElement.scrollWidth - innerWidth,
          section: el.scrollWidth - el.clientWidth,
          lunar: [...el.querySelectorAll(".daily-lunar-facts dd")]
            .filter((n) => n.scrollWidth > n.clientWidth + 1)
            .map((n) => n.textContent),
        }));
        assert(
          overflow.page <= 1 && overflow.section <= 1 && !overflow.lunar.length,
          JSON.stringify(overflow),
        );
        await page.locator("#panchang").screenshot({
          path: `.local/calculation-audit-${language}-${width}.png`,
          style: "header,.dev-strip {visibility:hidden!important}",
        });
        result.browserViews.push({
          language,
          width,
          noMoonEvent: true,
          samvat: true,
        });
      }
    }
    const financeError =
      "Financial totals exceed the supported exact range. Review the ledger.";
    await page.route("**/api/admin?**", (r) =>
      r.fulfill({
        json: {
          csrf: "isolated-browser-fixture",
          settings: config.settings,
          rules: {},
          users: [],
          profiles: [],
          orders: [],
          bookings: {},
          jobs: {},
          expenses: {},
          distributions: {},
          month: {
            month: "2026-10",
            reserve: Number.MAX_SAFE_INTEGER + 1,
            statement_reference: "fixture",
            closed: false,
          },
          finance: {
            month: "2026-10",
            calculation_error: financeError,
            unreconciled: true,
          },
          audit: [],
          analytics: [],
          pdf_blockers: [],
          booking_blockers: [],
        },
      }),
    );
    await page.clock.setFixedTime(new Date("2026-09-30T20:00:00Z"));
    await page.goto(base + "/admin");
    await page.locator(".admin-tabs").waitFor();
    await page.locator(".language-selector select").selectOption("en");
    await page
      .locator(".admin-tabs")
      .getByRole("button", { name: "Partnership accounts", exact: true })
      .click();
    assert.equal(
      await page.locator('input[type="month"]').inputValue(),
      "2026-10",
    );
    assert.equal(
      await page.locator('input[type="date"]').inputValue(),
      "2026-10-01",
    );
    result.businessDateDefaults = true;
    await page.getByRole("alert").filter({ hasText: financeError }).waitFor();
    assert.equal(await page.locator(".finance-grid").count(), 0);
    result.unsafeFinanceHidden = true;
    assert.deepEqual(result.errors, []);
    result.passed = true;
    fs.writeFileSync(
      ".local/critical-audit-browser-results.json",
      JSON.stringify(result, null, 2),
    );
    console.log(JSON.stringify(result));
  } finally {
    await browser.close();
  }
})().catch((error) => {
  console.error(error.stack);
  process.exitCode = 1;
});
