// Isolated local QA: only stateless matching POSTs are permitted.
const { chromium } = require("../web/node_modules/playwright");
const {
  default: AxeBuilder,
} = require("../web/node_modules/@axe-core/playwright");
const assert = require("node:assert/strict");
const fs = require("node:fs");
process.chdir(require("node:path").resolve(__dirname, ".."));

(async () => {
  const base = process.env.MATCHING_QA_URL || "http://127.0.0.1:18085";
  assert(["http://127.0.0.1:18085","http://127.0.0.1:18086"].includes(base),"Use the isolated file-store preview");
  const evidence = { passed: false, views: [], errors: [], checks: [] };
  const browser = await chromium.launch({
    executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe",
    headless: true,
  });
  try {
    const context = await browser.newContext({
      reducedMotion: "reduce",
      viewport: { width: 1440, height: 1000 },
    });
    await context.addInitScript(() => {
      localStorage.setItem("m4y-language", "en");
      localStorage.setItem("m4y-analytics", "no");
    });
    const page = await context.newPage();
    const calls = [];
    let fixture,
      mode = "real";
    let profileMode = "real";
    page.on("pageerror", (error) => evidence.errors.push(error.message));
    await page.route("**/api/**", async (route) => {
      const req = route.request(),
        path = new URL(req.url()).pathname;
      calls.push(path);
      if (path === "/api/kundli/match/profiles" && profileMode === "fixture")
        return route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            profiles: [
              {
                id: "saved-qa",
                birth: { ...fixture.bride.birth, name: "Saved QA Fixture" },
                place: fixture.bride.place,
                created: fixture.calculated_at,
              },
            ],
          }),
        });
      if (req.method() === "GET") return route.continue();
      if (path === "/api/kundli/match/pdf" && req.method() === "POST")
        return route.continue();
      if (path !== "/api/kundli/match" || req.method() !== "POST")
        return route.abort();
      if (mode === "real") return route.continue();
      if (mode === "delayed")
        await new Promise((resolve) => setTimeout(resolve, 500));
      if (mode === "error")
        return route.fulfill({
          status: 422,
          contentType: "application/json",
          body: JSON.stringify({
            person: "groom",
            error: "An accurate birth time is required for Kundli matching.",
          }),
        });
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(fixture),
      });
    });
    /** inspect checks matching results across the supported mobile/desktop widths and requested language. */
    async function inspect(label, language) {
      for (const width of [320, 339, 390, 650, 768, 901, 1100, 1440]) {
        await page.setViewportSize({ width, height: 1000 });
        await page.evaluate(() => document.fonts.ready);
        await page.evaluate(
          () =>
            new Promise((resolve) =>
              requestAnimationFrame(() => requestAnimationFrame(resolve)),
            ),
        );
        const metrics = await page.evaluate(() => ({
          viewport: innerWidth,
          width: document.documentElement.scrollWidth,
        }));
        assert(
          metrics.width <= width + 1,
          `${label}/${language}/${width}: ${JSON.stringify(metrics)}`,
        );
        evidence.views.push({ label, language, width });
      }
      const axe = await new AxeBuilder({ page }).analyze();
      assert.deepEqual(
        axe.violations.map((v) => ({
          id: v.id,
          nodes: v.nodes.map((n) => n.target),
        })),
        [],
        `${label}/${language}: accessibility`,
      );
    }
    await page.goto(base + "/kundli-matching");
    await page.locator("#matching-bride-name").waitFor();
    assert(
      !calls.some((path) => ["/api/me", "/api/admin/me"].includes(path)),
      "Matching should mount independently",
    );
    for (const role of ["bride", "groom"]) {
      const date = page.locator(`#matching-${role}-date`);
      assert.equal(await date.getAttribute("min"), "1940-01-01");
      await date.fill("1939-12-31");
      assert(await date.evaluate((el) => el.validity.rangeUnderflow));
      await date.fill(role === "bride" ? "1995-08-15" : "1992-02-29");
      await page
        .locator(`#matching-${role}-name`)
        .fill(role === "bride" ? "Bride QA Fixture" : "Groom QA Fixture");
      await page
        .locator(`#matching-${role}-time`)
        .fill(role === "bride" ? "10:30" : "18:15");
      const card = page
        .locator(".matching-birth-card")
        .nth(role === "bride" ? 0 : 1);
      const search = card.getByRole("combobox");
      await search.fill(role === "bride" ? "New Delhi" : "Mumbai");
      await card.getByRole("option").first().waitFor();
      await search.press("ArrowDown");
      await search.press("Enter");
      await card.locator(".matching-place-input.is-selected").waitFor();
    }
    await inspect("birth forms", "en");
    await page.screenshot({
      path: ".local/matching-form-desktop.png",
      fullPage: true,
    });
    await page.setViewportSize({ width: 339, height: 1000 });
    await page.screenshot({
      path: ".local/matching-form-mobile.png",
      fullPage: true,
    });
    const calculated = page.waitForResponse(
      (r) => new URL(r.url()).pathname === "/api/kundli/match",
    );
    await page.locator(".matching-submit").click();
    const response = await calculated;
    assert(response.ok(), await response.text());
    fixture = await response.json();
    assert.equal(fixture.max, 36);
    assert.equal(fixture.factors.length, 8);
    assert.equal(
      fixture.factors.reduce((sum, f) => sum + f.score, 0),
      fixture.total,
    );
    await page.locator(".matching-results").waitFor();
    assert.equal(await page.locator(".matching-factor").count(), 8);
    assert.equal(await page.locator(".matching-mangal-row").count(), 6);
    assert.equal(fixture.mangal.bride.references.length, 3);
    assert.equal(fixture.bride.navamsa.planets.length, 9);
    assert.equal(
      await page.locator(".matching-score strong").textContent(),
      String(fixture.total),
    );
    assert.equal(
      await page.locator(".matching-person-title h3").first().textContent(),
      "Bride QA Fixture",
    );
    evidence.native = {
      score: fixture.total,
      method: fixture.method,
      factors: fixture.factors.map((f) => ({
        code: f.code,
        score: f.score,
        max: f.max,
      })),
    };
    evidence.checks.push(
      "real native API result, eight exact factor scores, separate Mars checks, trusted keyboard birthplace selection, 1940 date validation",
    );
    await inspect("native result", "en");
    await page.screenshot({
      path: ".local/matching-result-desktop.png",
      fullPage: true,
    });
    // Each report tab is keyboard reachable and retains the same native score.
    await page.locator("#matching-result-tab-guna").focus();
    await page.keyboard.press("ArrowRight");
    assert.equal(
      await page
        .locator("#matching-result-tab-mangal")
        .getAttribute("aria-selected"),
      "true",
    );
    await page.locator(".matching-checked-rules summary").first().click();
    assert.equal(
      await page
        .locator(".matching-checked-rules")
        .first()
        .getAttribute("open"),
      "",
    );
    await inspect("Mangal triad and exception evidence", "en");
    await page.screenshot({
      path: ".local/matching-mangal-desktop.png",
      fullPage: true,
    });
    await page.locator("#matching-result-tab-charts").click();
    for (const [modeIndex, label] of [
      [0, "D1"],
      [1, "D9"],
      [2, "Moon"],
    ]) {
      await page.locator(".matching-chart-modes button").nth(modeIndex).click();
      assert.equal(
        await page.locator(".matching-chart-pair .kundli-chart").count(),
        2,
      );
      const person = fixture.bride;
      await page
        .locator(".matching-chart-pair")
        .first()
        .getByRole("button", { name: /Mars Ma/ })
        .first()
        .click();
      const expectedHouse =
        modeIndex === 0
          ? person.chart.planets.find((p) => p.name === "Mars").house
          : modeIndex === 1
            ? person.navamsa.planets.find((p) => p.name === "Mars").house
            : ((person.chart.planets.find((p) => p.name === "Mars")
                .rashi_number -
                person.moon.rashi_number +
                12) %
                12) +
              1;
      assert(
        (
          await page
            .locator(".matching-chart-pair .chart-house-detail strong")
            .first()
            .innerText()
        ).startsWith("House " + expectedHouse + " "),
      );
      if (modeIndex === 2)
        assert(
          (
            await page
              .locator(".matching-chart-pair .chart-viewer-meta")
              .first()
              .innerText()
          ).includes("Moon sign"),
        );
      await inspect(label + " chart comparison", "en");
    }
    await page.screenshot({
      path: ".local/matching-charts-desktop.png",
      fullPage: true,
    });
    await page.locator("#matching-result-tab-birth").click();
    assert(
      (await page.locator(".matching-metadata-people").innerText()).includes(
        new Intl.DateTimeFormat("en-IN",{timeZone:"UTC",day:"2-digit",month:"short",year:"numeric",hour:"2-digit",minute:"2-digit",timeZoneName:"short"}).format(new Date(fixture.bride.chart.utc)),
      ),
    );
    await inspect("resolved birth metadata", "en");
    await page.locator("#matching-result-tab-guna").click();
    await page.locator(".matching-factor-explanation summary").nth(2).click();
    const tara = fixture.factors.find((f) => f.code === "tara");
    assert(
      (
        await page.locator(".matching-factor-explanation").nth(2).innerText()
      ).includes(String(tara.evidence.bride_count)),
    );
    const englishDownload = page.waitForEvent("download");
    await page.locator(".matching-download").click();
    await (
      await englishDownload
    ).saveAs(".local/matching-browser-export-en.pdf");
    await page.locator(".matching-method summary").click();
    assert(
      (await page.locator(".matching-method").innerText()).includes(
        fixture.method,
      ),
    );
    await page.locator(".language-selector select").selectOption("hi");
    assert.equal(
      await page.locator("#matching-bride-name").inputValue(),
      "Bride QA Fixture",
    );
    assert.equal(await page.locator(".matching-factor").count(), 8);
    assert(
      !/Janma|Sampat|Vipat|Deva|Manushya|Rakshasa|Brahmin/.test(
        await page.locator(".matching-factor-grid").innerText(),
      ),
      "Hindi factor values should be translated",
    );
    await inspect("native result", "hi");
    for (const id of ["mangal", "charts", "birth"]) {
      await page.locator("#matching-result-tab-" + id).click();
      await inspect(id + " comparison", "hi");
      await page.setViewportSize({ width: 339, height: 1000 });
      await page.screenshot({
        path: ".local/matching-" + id + "-hindi-mobile.png",
        fullPage: true,
      });
    }
    await page.locator("#matching-result-tab-guna").click();
    const hindiDownload = page.waitForEvent("download");
    await page.locator(".matching-download").click();
    await (await hindiDownload).saveAs(".local/matching-browser-export-hi.pdf");
    await page.setViewportSize({ width: 390, height: 1000 });
    await page.screenshot({
      path: ".local/matching-result-hindi-mobile.png",
      fullPage: true,
    });
    await page.setViewportSize({ width: 1440, height: 1000 });
    await page.emulateMedia({ media: "print" });
    assert(await page.locator(".matching-results").isVisible());
    assert(!(await page.locator(".matching-form-section").isVisible()));
    assert.equal(await page.locator(".matching-factor:visible").count(), 8);
    await page.pdf({
      path: ".local/matching-print.pdf",
      format: "A4",
      printBackground: true,
    });
    await page.emulateMedia({ media: "screen" });
    const storage = await page.evaluate(() => ({
      local: { ...localStorage },
      session: { ...sessionStorage },
    }));
    assert(
      !JSON.stringify(storage).includes("QA Fixture"),
      "Partner birth details must not persist in browser storage",
    );
    evidence.checks.push(
      "English/Hindi input preservation, readable print view, no partner details in browser storage",
    );
    evidence.checks.push(
      "Four keyboard-accessible result tabs, all three chart frames and exact Mars houses, inclusive Tara trace, real English/Hindi PDF downloads",
    );
    await page.locator(".language-selector select").selectOption("en");
    await page.locator("#matching-bride-name").fill("Edited QA Fixture");
    assert.equal(
      await page.locator(".matching-results").count(),
      0,
      "Editing invalidates previous matching",
    );
    mode = "delayed";
    await page.locator(".matching-submit").click();
    await page.locator("#matching-groom-name").fill("Edited Groom QA");
    await page.waitForTimeout(650);
    assert.equal(
      await page.locator(".matching-results").count(),
      0,
      "Stale matching must not reappear",
    );
    mode = "error";
    await page.locator(".matching-submit").click();
    await page.locator(".matching-error").waitFor();
    assert(
      (await page.locator(".matching-error").innerText()).startsWith("Groom:"),
    );
    mode = "fixture";
    await page.locator(".matching-submit").click();
    await page.locator(".matching-results").waitFor();
    const groomSearch = page
      .locator(".matching-birth-card")
      .nth(1)
      .getByRole("combobox");
    await groomSearch.fill("Unselected City QA");
    assert.equal(await page.locator(".matching-results").count(), 0);
    await page.locator(".matching-submit").click();
    await page.locator(".matching-error").waitFor();
    assert(
      (await page.locator(".matching-error").innerText()).includes(
        "Choose a birthplace",
      ),
    );
    evidence.checks.push(
      "edit invalidation, aborted stale result suppression, role-specific errors, retry, invalid edited birthplace rejection",
    );
    await page.goto(base + "/kundli-matching");
    assert.equal(
      await page.locator("#matching-bride-name").inputValue(),
      "",
      "Refresh clears ephemeral birth data",
    );
    // Saved data is requested only after an explicit action; a guest sees sign-in.
    await page.locator(".matching-saved-button").first().click();
    await page.getByRole("dialog").waitFor();
    await page.getByRole("dialog").getByRole("link").waitFor();
    assert((await page.getByRole("dialog").innerText()).includes("Sign in"));
    assert(
      (
        await page.getByRole("dialog").locator("a").getAttribute("href")
      ).includes("next=%2Fkundli-matching"),
    );
    await page.keyboard.press("Escape");
    profileMode = "fixture";
    await page.locator(".matching-saved-button").first().click();
    await page
      .getByRole("dialog")
      .getByRole("button", { name: /Saved QA Fixture/ })
      .click();
    assert.equal(
      await page.locator("#matching-bride-name").inputValue(),
      "Saved QA Fixture",
    );
    assert.equal(
      await page
        .locator(".matching-birth-card")
        .first()
        .getByRole("combobox")
        .inputValue(),
      fixture.bride.place.name,
    );
    assert.equal(await page.locator("#matching-groom-name").inputValue(), "");
    assert.equal(await page.locator(".matching-results").count(), 0);
    evidence.checks.push(
      "Saved profile picker is explicit opt-in, guest sign-in return is safe, selection populates one role with trusted place and leaves the other role blank",
    );
    await page.locator(".language-selector select").selectOption("hi");
    await inspect("empty forms", "hi");
    await page.goto(base);
    for (const language of ["en", "hi"]) {
      await page.locator(".language-selector select").selectOption(language);
      for (const width of [901, 1100, 1201, 1440]) {
        await page.setViewportSize({ width, height: 1000 });
        assert(
          await page.evaluate(
            () => document.documentElement.scrollWidth <= innerWidth + 1,
          ),
          `homepage header ${language}/${width}`,
        );
      }
    }
    await page.locator(".language-selector select").selectOption("en");
    await page
      .locator(".site-header nav")
      .getByRole("button", { name: "Kundli matching", exact: true })
      .click();
    await page.waitForURL(base + "/kundli-matching");
    await page.locator("#matching-bride-name").waitFor();
    await page.goBack();
    await page.setViewportSize({ width: 339, height: 1000 });
    await page.locator(".menu-toggle").click();
    await page
      .locator(".site-header nav")
      .getByRole("button", { name: "Kundli matching", exact: true })
      .click();
    await page.waitForURL(base + "/kundli-matching");
    evidence.checks.push(
      "desktop/mobile homepage entry and history navigation, blank details after refresh",
    );
    assert.deepEqual(evidence.errors, []);
    evidence.passed = true;
  } finally {
    fs.writeFileSync(
      ".local/kundli-matching-browser-qa.json",
      JSON.stringify(evidence, null, 2),
    );
    await browser.close();
  }
  console.log(JSON.stringify(evidence, null, 2));
})().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
