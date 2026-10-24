// Local isolated QA. Only the stateless guest calculation may use POST.
const { chromium } = require("../web/node_modules/playwright");
const assert = require("node:assert/strict");
const fs = require("node:fs");

(async () => {
  const base = process.env.PANCHANG_QA_URL || "http://127.0.0.1:18084";
  assert.equal(base, "http://127.0.0.1:18084");
  /** get fetches a read-only historical-range endpoint and requires a successful JSON response. */
  const get = async (path) => {
    const r = await fetch(base + path);
    assert(r.ok, `${path}: ${r.status}`);
    return r.json();
  };
  const result = { passed: false, views: [], errors: [] };
  const january = await get(
    "/api/panchang/calendar?date=1940-01-01&place_id=delhi&view=month",
  );
  const february = await get(
    "/api/panchang/calendar?date=1940-02-29&place_id=delhi&view=month",
  );
  assert.equal(january.days.length, 31);
  assert.equal(january.days[0].date, "1940-01-01");
  assert.equal(february.days.length, 29);
  assert.equal(february.days[28].date, "1940-02-29");
  const midnight = await get(
    "/api/sky/positions?date=1940-01-01&time=00:00&place_id=delhi",
  );
  assert.equal(midnight.planets.length, 9);
  assert.equal(
    (await fetch(base + "/api/panchang?date=1939-12-31&place_id=delhi")).status,
    400,
  );
  const invalid = await fetch(base + "/api/kundli", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      name: "Audit Fixture",
      date: "1939-12-31",
      time: "12:00",
      place_id: "delhi",
      unknown_time: false,
    }),
  });
  assert.equal(invalid.status, 422);
  const transits = await get("/api/sky/transits?year=1940&place_id=delhi");
  const eclipses = await get("/api/sky/eclipses?year=1940&place_id=delhi");
  assert(transits.events.length > 100);
  assert(eclipses.events.length >= 2);
  result.api = {
    januaryDays: january.days.length,
    februaryDays: february.days.length,
    transitEvents: transits.events.length,
    eclipses: eclipses.events.length,
    rejected1939: true,
  };
  const browser = await chromium.launch({
    executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe",
    headless: true,
  });
  try {
    const context = await browser.newContext({
      reducedMotion: "reduce",
      viewport: { width: 339, height: 1000 },
    });
    await context.addInitScript(() => {
      localStorage.setItem("m4y-language", "en");
      localStorage.setItem("m4y-analytics", "no");
      sessionStorage.setItem(
        "m4y-birth",
        JSON.stringify({
          name: "Historical Audit Fixture",
          date: "1940-01-01",
          time: "00:00",
          place_id: "delhi",
          unknown_time: false,
        }),
      );
    });
    const page = await context.newPage();
    page.on("pageerror", (e) => result.errors.push(e.message));
    await page.route("**/api/**", (r) =>
      r.request().method() === "GET" ||
      new URL(r.request().url()).pathname === "/api/kundli"
        ? r.continue()
        : r.abort(),
    );
    /** check checks the selected historical screen at mobile/desktop sizes in the requested language. */
    async function check(label, language) {
      for (const width of [339, 1440]) {
        await page.setViewportSize({ width, height: 1000 });
        await page.evaluate(() => document.fonts.ready);
        await page.evaluate(
          () =>
            new Promise((resolve) =>
              requestAnimationFrame(() => requestAnimationFrame(resolve)),
            ),
        );
        const metrics = await page.evaluate(() => ({
          width: innerWidth,
          scroll: document.documentElement.scrollWidth,
          escaping: [...document.querySelectorAll("body *")]
            .filter((el) => {
              const r = el.getBoundingClientRect();
              return r.width > 0 && (r.right > innerWidth + 1 || r.left < -1);
            })
            .slice(0, 15)
            .map((el) => ({
              tag: el.tagName,
              cls: el.getAttribute("class"),
              text: el.textContent.slice(0, 100),
            })),
        }));
        if (metrics.scroll > width + 1)
          await page.screenshot({
            path: ".local/range-1940-overflow.png",
            fullPage: true,
          });
        assert(
          await page.evaluate(
            () => document.documentElement.scrollWidth <= innerWidth + 1,
          ),
          `${label}/${language}/${width}: overflow ${JSON.stringify(metrics)}`,
        );
        result.views.push({ label, language, width });
      }
    }
    /** chooseDate changes the Panchang date and waits for the matching server response. */
    async function chooseDate(date) {
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
      await page.goto(base);
      await page.locator("#birthplace").waitFor();
      await page.waitForFunction(
        () => document.querySelector("#birthplace")?.value.length > 0,
      );
      await page.locator(".language-selector select").selectOption(language);
      assert.equal(
        await page.locator("#birth-date").getAttribute("min"),
        "1940-01-01",
      );
      await page.locator("#birth-date").fill("1939-12-31");
      assert.equal(
        await page
          .locator("#birth-date")
          .evaluate((el) => el.validity.rangeUnderflow),
        true,
      );
      await page.locator("#birth-date").fill("1940-01-01");
      const generated = page.waitForResponse(
        (r) =>
          new URL(r.url()).pathname === "/api/kundli" &&
          r.request().method() === "POST",
      );
      await page.locator(".birth-form button[type=submit]").click();
      const report = await (await generated).json();
      assert.equal(report.chart.utc, "1939-12-31T18:30:00Z");
      assert.equal(report.chart.planets.length, 9);
      assert(report.navamsa && report.dasha.current_lord);
      await page.locator(".kundli-summary").waitFor();
      await check("1940 Kundli", language);
      await page.locator("#kundli-tab-dasha").click();
      assert((await page.locator(".dasha-timeline details").count()) >= 9);
      await check("1940 Dasha", language);
      await page.goto(base);
      await page.locator(".language-selector select").selectOption(language);
      await page.locator(".daily-limbs").waitFor();
      assert.equal(
        await page.locator("#daily-date").getAttribute("min"),
        "1940-01-01",
      );
      await chooseDate("1940-01-01");
      await check("1940 daily Panchang", language);
      await page.locator(".panchang-view-switch button").nth(1).click();
      await page.waitForFunction(
        () =>
          document.querySelectorAll(".panchang-calendar-grid button").length ===
          6,
      );
      await page.locator(".panchang-view-switch button").nth(2).click();
      await page.waitForFunction(
        () =>
          document.querySelectorAll(".panchang-calendar-grid button").length ===
          31,
      );
      await check("1940 January calendar", language);
      await page.locator("#daily-date").fill("1940-02-29");
      await page.waitForFunction(
        () =>
          document.querySelectorAll(".panchang-calendar-grid button").length ===
          29,
      );
      await check("1940 leap February", language);
      await page.locator(".panchang-view-switch button").nth(0).click();
      await chooseDate("1942-10-18");
      assert.match(
        await page.locator(".daily-date-context").innerText(),
        /UTC\+06:30/,
      );
      await check("historical India offset", language);
      await page.locator("#daily-tab-4").click();
      await page.locator(".sky-positions tbody tr").first().waitFor();
      assert.match(
        await page.locator(".sky-timestamp").innerText(),
        /UTC\+06:30/,
      );
      await page.locator(".sky-switch button").nth(1).click();
      assert.equal(
        await page.locator("#sky-year option").first().textContent(),
        "1940",
      );
      assert.equal(
        await page.locator("#sky-year option").last().textContent(),
        "2099",
      );
      assert.equal(await page.locator("#sky-year option").count(), 160);
      await page.locator("#sky-year").selectOption("1940");
      await page.locator(".sky-transits").waitFor();
      await check("1940 planet transits", language);
      await page.locator("#daily-tab-5").click();
      await page.locator(".eclipse-neighbour").first().waitFor();
      assert.equal(await page.locator("#sky-year").count(), 0);
      // Historical annual coverage is verified via the native API above;
      // the public display now deliberately shows today's previous/next pair.
      await check("previous and next eclipses", language);
    }
    assert.deepEqual(result.errors, []);
    result.passed = true;
    fs.writeFileSync(
      ".local/date-range-1940-browser-results.json",
      JSON.stringify(result, null, 2),
    );
    console.log(
      JSON.stringify({
        passed: true,
        layouts: result.views.length,
        ...result.api,
        errors: result.errors,
      }),
    );
  } finally {
    await browser.close();
  }
})().catch((error) => {
  console.error(error.stack);
  process.exitCode = 1;
});
