// Synthetic, stateless local QA. No accounts, customer records or provider writes.
const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const assert = require("node:assert/strict");
const fs = require("node:fs");
const base = process.env.DASHA_QA_URL || "http://127.0.0.1:18086";
const result = {
  passed: false,
  views: [],
  checks: [],
  errors: [],
  axeScans: 0,
};

(async () => {
  const response = await fetch(base + "/api/kundli", {
    method: "POST",
    headers: { "Content-Type": "application/json", Origin: base },
    body: JSON.stringify({
      name: "Dasha Layer QA",
      date: "1995-08-15",
      time: "10:30:47",
      place_id: "delhi",
      unknown_time: false,
    }),
  });
  assert(response.ok, await response.clone().text());
  const fixture = await response.json();
  fs.writeFileSync(
    ".local/dasha-five-level-fixture.json",
    JSON.stringify(fixture, null, 2),
  );
  const browser = await chromium.launch({
    executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe",
    headless: true,
  });
  try {
    const context = await browser.newContext({
      reducedMotion: "reduce",
      viewport: { width: 1440, height: 1000 },
    });
    await context.addInitScript((report) => {
      localStorage.setItem("m4y-language", "en");
      localStorage.setItem("m4y-analytics", "no");
      if (!sessionStorage.getItem("m4y-guest-report")) {
        sessionStorage.setItem("m4y-guest-report", JSON.stringify(report));
        sessionStorage.setItem("m4y-birth", JSON.stringify(report.birth));
      }
    }, fixture);
    const page = await context.newPage();
    page.on("pageerror", (error) => result.errors.push(error.message));
    let mode = "real",
      calls = 0;
    const bodies = [],
      responses = new Map();
    await page.route("**/api/**", async (route) => {
      const req = route.request(),
        path = new URL(req.url()).pathname;
      if (req.method() === "GET") return route.continue();
      assert.equal(
        path,
        "/api/kundli/dasha",
        "Only stateless Dasha queries allowed",
      );
      calls++;
      bodies.push(req.postDataJSON());
      if (mode === "error")
        return route.fulfill({
          status: 503,
          json: { error: "Synthetic temporary failure" },
        });
      if (mode === "delayed")
        await new Promise((resolve) => setTimeout(resolve, 650));
      const response = await route.fetch();
      assert(response.ok(), await response.text());
      const json = await response.json();
      responses.set(json.path.join("/"), json);
      return route.fulfill({ response });
    });
    /** list locates the nested Dasha period list for the requested level. */
    const list = (level) =>
      page.locator(`.dasha-period-list[data-level="${level}"]`);
    /** rows locates only the direct period rows at the selected Dasha level. */
    const rows = (level) =>
      list(level).locator(":scope > .dasha-item > .dasha-row");
    /** ready waits for the requested Dasha branch to appear before inspecting it. */
    const ready = (level) => list(level).waitFor();
    /** settle waits for queued browser rendering frames before measuring the layout. */
    const settle = () =>
      page.evaluate(
        () =>
          new Promise((r) =>
            requestAnimationFrame(() => requestAnimationFrame(r)),
          ),
      );
    /** collapse closes expanded Dasha branches through the visible collapse control. */
    const collapse = async () => {
      const button = page.getByRole("button", {
        name: "Collapse all",
        exact: true,
      });
      if (await button.count()) await button.click();
    };
    await page.goto(base + "/kundli");
    await page.locator("#kundli-tab-dasha").click();
    await ready(1);
    assert.equal(await page.locator(".dasha-level-tabs").count(), 0);
    assert.equal(await page.locator(".dasha-current-chain > li").count(), 5);
    const saturn = fixture.dasha.periods.findIndex((p) => p.lord === "Saturn");
    assert(saturn >= 0);
    let path = [];
    for (let level = 1; level <= 5; level++) {
      const expected = responses.get(path.join("/"));
      assert(expected, `No API source for ${path}`);
      assert.equal(await rows(level).count(), expected.periods.length);
      assert.equal(
        await page.locator('.dasha-toggle[aria-expanded="true"]').count(),
        level - 1,
      );
      const actual = await rows(level)
        .locator("time")
        .evaluateAll((nodes) =>
          nodes.map((n) => ({ iso: n.dateTime, text: n.innerText })),
        );
      assert.equal(actual.length, expected.periods.length * 2);
      expected.periods.forEach((period, i) => {
        assert.equal(actual[2 * i].iso, period.start);
        assert.equal(actual[2 * i + 1].iso, period.end);
        if (i > 0)
          assert.equal(
            actual[2 * i - 1].iso,
            actual[2 * i].iso,
            "Adjacent exact boundaries",
          );
      });
      if (level > 1) {
        assert.equal(
          await rows(level).first().locator("strong").innerText(),
          Array(level).fill("Saturn").join(" \u2013 "),
        );
        const parent = expected.ancestors.at(-1);
        assert.equal(actual[0].iso, parent.start);
        assert.equal(actual.at(-1).iso, parent.end);
      }
      for (const language of ["en", "hi"]) {
        await page.locator(".language-selector select").selectOption(language);
        await settle();
        const formatter = new Intl.DateTimeFormat(
          language === "hi" ? "hi-IN" : "en-IN",
          {
            timeZone: fixture.place.zone,
            hour: "2-digit",
            minute: "2-digit",
            hourCycle: "h23",
            timeZoneName: "shortOffset",
          },
        );
        const displayed = await rows(level)
          .locator("time")
          .evaluateAll((nodes) =>
            nodes.map((n) => ({ iso: n.dateTime, text: n.innerText })),
          );
        for (const stamp of displayed)
          assert(stamp.text.includes(formatter.format(new Date(stamp.iso))));
        assert(
          !/\b\d{1,2}:\d{2}:\d{2}\b/.test(
            await page.locator(".dasha-explorer").innerText(),
          ),
          "Visible seconds",
        );
        assert.equal(
          await rows(level)
            .locator("dt")
            .filter({
              hasText:
                language === "en"
                  ? "Start time"
                  : "\u0936\u0941\u0930\u0942 \u0938\u092e\u092f",
            })
            .count(),
          expected.periods.length,
        );
        assert.equal(
          await rows(level)
            .locator("dt")
            .filter({
              hasText:
                language === "en"
                  ? "End time"
                  : "\u0905\u0902\u0924 \u0938\u092e\u092f",
            })
            .count(),
          expected.periods.length,
        );
        if (language === "hi")
          assert(
            !/Mahadasha|Antardasha|Pratyantar|Sukshma|Prana|Collapse all|9 periods/.test(
              await page.locator(".dasha-explorer").innerText(),
            ),
            "Untranslated Hindi labels",
          );
        for (const width of [320, 339, 390, 650, 768, 1100, 1440]) {
          await page.setViewportSize({ width, height: 1000 });
          await page.evaluate(() => document.fonts.ready);
          await settle();
          assert(
            await page.evaluate(
              () => document.documentElement.scrollWidth <= innerWidth + 1,
            ),
            `${language}/${level}/${width}: page overflow`,
          );
          const escaped = await page.locator(".dasha-explorer").evaluate((el) =>
            [
              ...el.querySelectorAll(
                ".dasha-row,.dasha-times,time,.dasha-toggle",
              ),
            ]
              .filter((n) => {
                const box = n.getBoundingClientRect();
                return (
                  box.width &&
                  (box.left < 0 ||
                    box.right > innerWidth + 1 ||
                    n.scrollWidth > n.clientWidth + 1)
                );
              })
              .map((n) => n.className || n.tagName),
          );
          assert.deepEqual(
            escaped,
            [],
            `${language}/${level}/${width}: clipped content`,
          );
          result.views.push({ language, level, width });
          if ([339, 1440].includes(width) && [1, 5].includes(level)) {
            await rows(level).first().scrollIntoViewIfNeeded();
            await page.screenshot({
              path: `.local/dasha-layers-${language}-${level}-${width}.png`,
            });
          }
        }
        const axe = await new AxeBuilder({ page })
          .include(".kundli-workspace")
          .withTags(["wcag2a", "wcag2aa", "wcag21aa"])
          .analyze();
        result.axeScans++;
        assert.deepEqual(
          axe.violations.map((v) => ({
            id: v.id,
            nodes: v.nodes.map((n) => n.target),
          })),
          [],
          `${language}/${level}: Axe`,
        );
      }
      await page.locator(".language-selector select").selectOption("en");
      if (level < 5) {
        const index = level === 1 ? saturn : 0;
        const toggle = rows(level).nth(index).getByRole("button");
        await toggle.focus();
        await page.keyboard.press("Enter");
        path.push(index);
        await ready(level + 1);
        assert.equal(await toggle.getAttribute("aria-expanded"), "true");
        assert(
          await toggle.evaluate((el) => el === document.activeElement),
          "Opening a layer moves keyboard focus",
        );
      }
      console.log(`Dasha layer ${level}: EN/HI and seven widths passed`);
    }
    result.checks.push(
      "Saturn to Saturn-Saturn-Saturn-Saturn-Saturn; every sibling/date matches native API, parent coverage and contiguous exact boundaries; minute-only start/end labels; 70 EN/HI layouts and 10 Axe scans",
    );
    const countBeforeCache = calls;
    const parentToggle = rows(4).first().getByRole("button");
    await parentToggle.focus();
    await page.keyboard.press("Space");
    assert.equal(await list(5).count(), 0);
    await page.keyboard.press("Enter");
    await ready(5);
    assert.equal(calls, countBeforeCache);
    await collapse();
    assert.equal(
      await page.locator('.dasha-toggle[aria-expanded="true"]').count(),
      0,
    );
    await page
      .getByRole("button", { name: "Open running Dasha layers", exact: true })
      .click();
    await ready(5);
    const current = fixture.dasha.current_path[4];
    const currentRow = list(5).locator(":scope > .is-current > .dasha-row");
    assert.equal(
      await currentRow.locator("time").first().getAttribute("datetime"),
      current.start,
    );
    await page.waitForFunction(() =>
      document.activeElement?.classList.contains("dasha-leaf"),
    );
    result.checks.push(
      "Enter/Space disclosure, ancestor collapse/reopen from cache, collapse all; current-Dasha shortcut opens every ancestor and focuses the exact running Prana",
    );
    await collapse();
    // Fail a new sibling, retry in place, then replace it with another sibling.
    const unused = [0, 1, 2, 3, 4, 5, 6, 7, 8].filter(
      (i) => i !== saturn && i !== fixture.dasha.current_path[0].path[0],
    );
    mode = "error";
    await rows(1).nth(unused[0]).getByRole("button").click();
    await page.locator(".dasha-load-error").waitFor();
    assert.equal(await rows(1).count(), fixture.dasha.periods.length);
    mode = "real";
    await page.getByRole("button", { name: "Try again", exact: true }).click();
    await ready(2);
    assert.equal(await page.locator(".dasha-load-error").count(), 0);
    await collapse();
    mode = "delayed";
    await rows(1).nth(unused[1]).getByRole("button").click();
    await page.locator(".dasha-branch-status").waitFor();
    mode = "real";
    await rows(1).nth(saturn).getByRole("button").click();
    await ready(2);
    await page.waitForTimeout(800);
    assert.equal(
      await list(2).getAttribute("data-path"),
      String(saturn),
      "Aborted response replaced selected sibling",
    );
    assert.equal(
      await page.locator('.dasha-toggle[aria-expanded="true"]').count(),
      1,
    );
    result.checks.push(
      "Failed child retries beneath its own parent; sibling selection cancels pending work and does not attach stale children",
    );
    // Legacy snapshots are calculated without mutating saved data.
    await page.evaluate((report) => {
      const old = structuredClone(report);
      delete old.dasha.current_path;
      delete old.dasha.method;
      delete old.dasha.max_depth;
      sessionStorage.setItem("m4y-guest-report", JSON.stringify(old));
    }, fixture);
    await page.reload();
    await page.locator("#kundli-tab-dasha").click();
    await ready(1);
    assert.equal(await page.locator(".dasha-current-chain > li").count(), 5);
    assert.equal(
      await page.evaluate(
        () =>
          JSON.parse(sessionStorage.getItem("m4y-guest-report")).dasha
            .current_path,
      ),
      undefined,
    );
    mode = "delayed";
    await rows(1).nth(4).getByRole("button").click();
    await page.locator("#kundli-tab-charts").click();
    await page.waitForTimeout(800);
    assert.equal(await page.locator(".dasha-explorer").count(), 0);
    mode = "real";
    assert(
      bodies.every((body) => body.birth.time === "10:30:47"),
      "Birth precision lost",
    );
    result.checks.push(
      "Legacy report remains immutable; leaving report tab cancels query; original birth precision retained",
    );
    const nyResponse = await fetch(base + "/api/kundli", {
      method: "POST",
      headers: { "Content-Type": "application/json", Origin: base },
      body: JSON.stringify({
        ...fixture.birth,
        name: "New York Dasha QA",
        date: "1990-07-01",
        time: "10:30:31",
        place_id: "new-york",
      }),
    });
    assert(nyResponse.ok);
    const ny = await nyResponse.json();
    await page.evaluate(
      (report) =>
        sessionStorage.setItem("m4y-guest-report", JSON.stringify(report)),
      ny,
    );
    await page.reload();
    await page.locator("#kundli-tab-dasha").click();
    await ready(1);
    await rows(1).first().getByRole("button").click();
    await ready(2);
    const nyTimes = await rows(2)
      .locator("time")
      .evaluateAll((nodes) =>
        nodes.map((n) => ({ iso: n.dateTime, text: n.innerText })),
      );
    const nyFormatter = new Intl.DateTimeFormat("en-IN", {
      timeZone: "America/New_York",
      hour: "2-digit",
      minute: "2-digit",
      hourCycle: "h23",
      timeZoneName: "shortOffset",
    });
    for (const stamp of nyTimes)
      assert(stamp.text.includes(nyFormatter.format(new Date(stamp.iso))));
    const offsets = new Set(
      nyTimes.map(
        (stamp) =>
          nyFormatter
            .formatToParts(new Date(stamp.iso))
            .find((p) => p.type === "timeZoneName").value,
      ),
    );
    assert(offsets.has("GMT-4") && offsets.has("GMT-5"));
    result.checks.push(
      "Historical New York boundaries use their own GMT-4/GMT-5 offsets",
    );
    assert.deepEqual(result.errors, []);
    result.passed = true;
    console.log(
      JSON.stringify(
        {
          passed: true,
          views: result.views.length,
          axeScans: result.axeScans,
          checks: result.checks,
        },
        null,
        2,
      ),
    );
  } finally {
    fs.writeFileSync(
      ".local/dasha-layers-browser-qa.json",
      JSON.stringify(result, null, 2),
    );
    await browser.close();
  }
})().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
