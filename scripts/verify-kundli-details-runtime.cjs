// Synthetic session fixture only; no customer, account or provider writes.
const { chromium } = require('../web/node_modules/playwright');
const fs = require('node:fs');
const assert = require('node:assert/strict');
(async () => {
  const report = JSON.parse(fs.readFileSync('.local/kundli-details-fixture.json', 'utf8'));
  const config = await (await fetch('http://127.0.0.1:8080/api/config')).json();
  assert.equal(config.storage, 'MongoDB Atlas');
  const timelineResponse = await fetch('http://127.0.0.1:8080/api/kundli/saturn', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Origin: 'http://localhost:5173' },
    body: JSON.stringify({ birth: report.birth, start_year: 2023, end_year: 2024 }),
  });
  assert(timelineResponse.ok, await timelineResponse.clone().text());
  const timeline = await timelineResponse.json();
  assert(Array.isArray(timeline.periods));
  const browser = await chromium.launch({ executablePath: 'C:/Program Files/Google/Chrome/Application/chrome.exe', headless: true });
  const errors = [], layouts = [];
  try {
    const context = await browser.newContext({ reducedMotion: 'reduce' });
    await context.addInitScript((report) => {
      localStorage.setItem('m4y-language', 'en');
      localStorage.setItem('m4y-analytics', 'no');
      sessionStorage.setItem('m4y-guest-report', JSON.stringify(report));
      sessionStorage.setItem('m4y-birth', JSON.stringify(report.birth));
    }, report);
    const page = await context.newPage();
    page.on('pageerror', e => errors.push(e.message));
    await page.route('**/api/**', route => {
      if (route.request().method() !== 'GET') {
        assert.equal(new URL(route.request().url()).pathname, '/api/kundli/dasha');
      }
      return route.continue();
    });
    await page.goto('http://localhost:5173/kundli');
    for (const language of ['en', 'hi']) {
      await page.locator('.language-selector select').selectOption(language);
      for (const width of [320, 339, 768, 1440]) {
        await page.setViewportSize({ width, height: 900 });
        for (const tab of ['birth', 'charts', 'planets', 'checks', 'dasha', 'transits']) {
          await page.locator('#kundli-tab-' + tab).click();
          if (tab === 'charts') {
            assert.equal(await page.locator('#kundli-chart-choice option').count(), 18);
            await page.locator('#kundli-chart-choice').selectOption('d10');
          }
          if (tab === 'dasha') {
            await page.locator('.dasha-period-list[data-level="1"]').waitFor();
            assert.equal(await page.locator('.dasha-current-chain > li').count(), 5);
            await page.locator('.dasha-system-select select').selectOption('yogini');
            assert.equal(await page.locator('.yogini-periods>details').count(), 8);
            await page.locator('.dasha-system-select select').selectOption('vimshottari');
          }
          assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `${language}/${width}/${tab} overflow`);
          assert(!/\b\d{1,2}:\d{2}:\d{2}\b/.test(await page.locator('#kundli-panel').innerText()), 'Visible seconds');
        }
        layouts.push({ language, width });
      }
    }
    assert.deepEqual(errors, []);
    const results = { passed: true, storage: config.storage, layouts, errors };
    fs.writeFileSync('.local/kundli-details-runtime-results.json', JSON.stringify(results, null, 2));
    console.log(JSON.stringify(results));
  } finally { await browser.close(); }
})().catch(e => { console.error(e); process.exitCode = 1; });
