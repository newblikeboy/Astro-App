const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const fs = require("node:fs");
const crypto = require("node:crypto");
const zlib = require("node:zlib");
const assert = require("node:assert/strict");

/** verify checks handwritten-source provenance, complete role partitions and actual chart displays without accessing customer APIs. */
async function verify() {
  const bytes = fs.readFileSync("internal/platform/data/guruji-lagna-roles.json");
  const data = JSON.parse(bytes);
  assert.equal(crypto.createHash("sha256").update(fs.readFileSync(data.source.file)).digest("hex"), data.source.sha256);
  assert.deepEqual(data.records.map(row => row.lagna), Array.from({length:12}, (_,i) => i+1));
  const planets = ["सूर्य", "चन्द्र", "मंगल", "बुध", "गुरु", "शुक्र", "शनि"].sort();
  for (const row of data.records) assert.deepEqual([...row.karak, ...row.marak, ...row.sam].sort(), planets);
  assert.equal(data.records[11].karak_label, "योगकारक");
  assert.ok(data.records[11].karak.includes("बुध"));
  for (const lagna of [3,6,9,12]) assert.deepEqual(data.records[lagna-1].sam, []);
  assert.deepEqual(JSON.parse(zlib.gunzipSync(fs.readFileSync("output/guruji-bhagyaphal/guruji-lagna-roles-hi.min.json.gz"))), data);
  const files = [["scripts/fixtures/source-only-qa.html","web/.source-only-qa.html"],["scripts/fixtures/source-only-qa.tsx","web/src/.source-only-qa.tsx"]];
  const created = [];
  let browser;
  const results = [];
  try {
    for (const [source,target] of files) {fs.copyFileSync(source,target,fs.constants.COPYFILE_EXCL);created.push(target);}
    browser = await chromium.launch({executablePath:"C:/Program Files/Google/Chrome/Application/chrome.exe",headless:true});
    const context = await browser.newContext({reducedMotion:"reduce"});
    const page = await context.newPage();
    const errors = [];
    page.on("pageerror",error=>errors.push(error.message));
    await page.route("**/api/**",route=>route.abort());
    await page.goto(process.env.SOURCE_QA_URL || "http://127.0.0.1:5197/.source-only-qa.html");
    await page.locator(".lagna-graha-roles").waitFor();
    for (const language of ["hi","en"]) {
      await page.locator(`#qa-${language}`).click();
      for (const width of [320,390,1440]) {
        await page.setViewportSize({width,height:1000});
        for (const row of data.records) {
          await page.locator("#qa-lagna").selectOption(String(row.lagna));
          const panel = page.locator(".lagna-graha-roles");
          await panel.locator(`h3:text-is("${row.title}")`).waitFor();
          assert.equal(await panel.getAttribute("data-lagna"), String(row.lagna));
          assert.deepEqual(await panel.locator("dt").allTextContents(), [row.karak_label,"मारक","सम"]);
          assert.deepEqual(await panel.locator("dd").allTextContents(), [row.karak,row.marak,row.sam].map(values=>values.join(", ")||"—"));
          assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+2), false, `overflow ${row.lagna}/${width}/${language}`);
          const sourceBox = await panel.boundingBox();
          const chartBox = await page.locator(".overview-chart-grid .kundli-chart").first().boundingBox();
          assert.ok(sourceBox.y >= chartBox.y + chartBox.height, "role list must appear below the chart");
        }
        results.push({language,width,lagnas:12,belowChart:true,overflow:false});
      }
    }
    await page.locator("#qa-lagna").selectOption("7");
    await page.locator("#kundli-tab-charts").click();
    for (const mode of ["d9","moon","chalit"]) {
      await page.locator("#kundli-chart-choice").selectOption(mode);
      assert.equal(await page.locator(".lagna-graha-roles").getAttribute("data-lagna"), "7");
      assert.equal(await page.locator(".lagna-graha-roles h3").innerText(), "तुला लग्न");
    }
    const scan = await new AxeBuilder({page}).include(".lagna-graha-roles").analyze();
    assert.deepEqual(scan.violations.filter(v=>v.impact==="critical"||v.impact==="serious"), []);
    await page.locator("#kundli-tab-overview").click();
    await page.locator("#qa-lagna").selectOption("12");
    await page.screenshot({path:"tmp/source-only-qa/lagna-roles-desktop.png",fullPage:true});
    await page.setViewportSize({width:390,height:1000});
    await page.screenshot({path:"tmp/source-only-qa/lagna-roles-mobile.png",fullPage:true});
    await page.locator("#qa-lagna").selectOption("0");
    assert.equal(await page.locator(".lagna-graha-roles").count(), 0);
    assert.deepEqual(errors, []);
    fs.writeFileSync("tmp/source-only-qa/lagna-roles-validation.json", JSON.stringify({results,sourceHash:true,natalLagnaPreserved:true,unknownTimeNoRoles:true,pageErrors:errors,seriousAccessibilityViolations:0},null,2));
    console.log(JSON.stringify({views:72,sourceHash:true,natalLagnaPreserved:true,unknownTimeNoRoles:true,seriousAccessibilityViolations:0}));
  } finally {
    if (browser) await browser.close();
    for (const file of created.reverse()) fs.unlinkSync(file);
  }
}
verify().catch(error=>{console.error(error);process.exitCode=1;});
