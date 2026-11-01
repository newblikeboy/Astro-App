// Read-only guest calculations and local browser snapshots; no customer/order writes.
const assert = require("node:assert/strict");
const fs = require("node:fs");
const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const base = "http://localhost:5173";
const output = ".local/rajyog-qa";
const births = [
  { name: "Raja Babu", date: "1995-11-29", time: "12:35", place_id: "geonames-1273491" },
  { name: "Purnia", date: "1996-11-26", time: "16:00", place_id: "geonames-1259166" },
  { name: "Third chart", date: "1997-08-28", time: "16:00", place_id: "delhi" },
  { name: "Negative control", date: "2001-09-21", time: "22:30", place_id: "delhi" },
];

/** openChecks selects the yoga section using the visible responsive navigation. */
async function openChecks(page) {
  const select = page.locator(".kundli-section-select select");
  if (page.viewportSize().width <= 650) await select.selectOption("checks");
  else await page.locator("#kundli-tab-checks").click();
}

/** verify checks live detection, independent paragraph rendering and legacy guest refresh. */
async function verify() {
  fs.mkdirSync(output, { recursive: true });
  const reports = [];
  for (const birth of births) {
    const response = await fetch(base + "/api/kundli", { method: "POST", headers: { "Content-Type": "application/json", Origin: base }, body: JSON.stringify(birth) });
    assert(response.ok, `guest calculation failed: ${response.status}`);
    reports.push(await response.json());
  }
  assert.deepEqual(reports.map(r => r.details.yoga_checks.rajyog.matches), [true, true, true, false]);
  assert.deepEqual(reports.map(r => r.details.rajyog_reading.paragraphs.length), [2, 2, 1, 1]);
  assert.equal(new Set(reports.slice(0, 3).map(r => r.details.rajyog_reading.paragraphs[0].hi)).size, 3);
  const results = { passed: false, checks: [], errors: [] };
  if (fs.existsSync(".local/rajyog-positions-before.json")) {
    const third = reports[2];
    assert.deepEqual({ chart: third.chart, chalit: third.details.chalit, vargas: third.details.vargas }, JSON.parse(fs.readFileSync(".local/rajyog-positions-before.json", "utf8")));
    results.checks.push("native-positions-unchanged");
  }
  const browser = await chromium.launch({ executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe", headless: true });
  try {
    for (const [index, report] of reports.entries()) {
      const context = await browser.newContext({ reducedMotion: "reduce" });
      await context.addInitScript(r => {
        localStorage.setItem("m4y-language", "hi");
        localStorage.setItem("m4y-analytics", "no");
        if (!sessionStorage.getItem("m4y-guest-report")) {
          sessionStorage.setItem("m4y-guest-report", JSON.stringify(r));
          sessionStorage.setItem("m4y-birth", JSON.stringify(r.birth));
        }
      }, report);
      const page = await context.newPage();
      page.on("pageerror", error => results.errors.push(error.message));
      await page.route("**/api/**", route => {
        assert.equal(route.request().method(), "GET", "unexpected browser mutation");
        return route.continue();
      });
      await page.goto(base + "/kundli");
      for (const language of ["hi", "en"]) {
        await page.locator(".language-selector select").selectOption(language);
        for (const width of [390, 1440]) {
          await page.setViewportSize({ width, height: 1000 });
          await openChecks(page);
          const section = page.locator('[aria-labelledby="rajyog-heading"]');
          const reading = report.details.rajyog_reading;
          assert.equal(await section.locator(".natal-yoga-status").innerText(), reading.summary[language]);
          const text = await section.innerText();
          for (const paragraph of reading.paragraphs) assert(text.includes(paragraph[language]));
          assert(!text.includes("यह योग मौजूद है, लेकिन उससे जुड़े ग्रहों की कमजोरी"));
          assert(!/rajyog\.|\{v\d+\}/u.test(text));
          assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1));
          const audit = await new AxeBuilder({ page }).include('[aria-labelledby="rajyog-heading"]').analyze();
          assert.deepEqual(audit.violations.map(v => v.id), []);
          if (language === "hi" && width === 390) await section.screenshot({ path: `${output}/chart-${index + 1}-hi.png` });
          results.checks.push(`${index + 1}/${language}/${width}`);
        }
      }
      if (index === 2) {
        const old = structuredClone(report);
        old.details.qualifications.method = "m4y-phaladeepika7-neechabhanga-strength-v1";
        delete old.details.rajyog_reading;
        delete old.details.qualifications.rajyog_pairs;
        await page.unroute("**/api/**");
        let refreshes = 0;
        await page.route("**/api/**", route => {
          const request = route.request();
          if (request.method() === "POST" && new URL(request.url()).pathname === "/api/kundli") {
            refreshes++;
            assert.equal(request.postDataJSON().date, report.birth.date);
            return route.continue();
          }
          assert.equal(request.method(), "GET");
          return route.continue();
        });
        await page.evaluate(r => sessionStorage.setItem("m4y-guest-report", JSON.stringify(r)), old);
        await page.reload();
        await openChecks(page);
        await page.waitForFunction(() => JSON.parse(sessionStorage.getItem("m4y-guest-report")).details.rajyog_reading?.method === "m4y-pair-specific-rajyog-reader-v1");
        assert.equal(refreshes, 1);
        await page.locator('[aria-labelledby="rajyog-heading"] .natal-yoga-status').filter({ hasText: report.details.rajyog_reading.summary.hi }).waitFor();
        results.checks.push("legacy-guest-refresh-once");
      }
      await context.close();
    }
    assert.deepEqual(results.errors, []);
    results.passed = true;
    fs.writeFileSync(`${output}/results.json`, JSON.stringify(results, null, 2));
    console.log(JSON.stringify(results));
  } finally { await browser.close(); }
}

verify().catch(error => { console.error(error); process.exitCode = 1; });
