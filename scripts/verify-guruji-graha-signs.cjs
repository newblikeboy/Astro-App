const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const fs = require("node:fs");
const crypto = require("node:crypto");
const zlib = require("node:zlib");
const assert = require("node:assert/strict");

/** verify checks board provenance and natal sign matching for all planets, including the board's two-sign nodal convention. */
async function verify() {
  const data = JSON.parse(fs.readFileSync("internal/platform/data/guruji-graha-signs.json","utf8"));
  assert.equal(crypto.createHash("sha256").update(fs.readFileSync(data.source.file)).digest("hex"),data.source.sha256);
  assert.equal(new Set(data.records.map(record=>record.planet)).size,9);
  for (const record of data.records) {
    assert.ok([...record.exalted_signs,...record.debilitated_signs].every(sign=>Number.isInteger(sign)&&sign>=1&&sign<=12));
    assert.ok(record.exalted_signs.every(sign=>!record.debilitated_signs.includes(sign)));
  }
  const rahu=data.records.find(record=>record.planet==="Rahu"),ketu=data.records.find(record=>record.planet==="Ketu");
  assert.deepEqual(rahu.exalted_signs,[2,3]);assert.deepEqual(rahu.debilitated_signs,[8,9]);
  assert.deepEqual(ketu.exalted_signs,rahu.debilitated_signs);assert.deepEqual(ketu.debilitated_signs,rahu.exalted_signs);
  assert.deepEqual(JSON.parse(zlib.gunzipSync(fs.readFileSync("output/guruji-bhagyaphal/guruji-graha-signs-hi.min.json.gz"))),data);
  const files=[["scripts/fixtures/source-only-qa.html","web/.source-only-qa.html"],["scripts/fixtures/source-only-qa.tsx","web/src/.source-only-qa.tsx"]];
  const created=[];
  let browser;
  const results=[];
  try {
    for(const [source,target] of files){fs.copyFileSync(source,target,fs.constants.COPYFILE_EXCL);created.push(target);}
    browser=await chromium.launch({executablePath:"C:/Program Files/Google/Chrome/Application/chrome.exe",headless:true});
    const context=await browser.newContext({reducedMotion:"reduce"});
    const page=await context.newPage();
    const errors=[];
    page.on("pageerror",error=>errors.push(error.message));
    await page.route("**/api/**",route=>route.abort());
    await page.goto(process.env.SOURCE_QA_URL||"http://localhost:5173/.source-only-qa.html");
    await page.locator(".natal-table-with-status").waitFor();
    for(const language of ["hi","en"]){
      await page.locator(`#qa-${language}`).click();
      for(let sign=1;sign<=12;sign++){
        await page.locator("#qa-sign").selectOption(String(sign));
        for(const record of data.records){
          const expected=record.exalted_signs.includes(sign)?"exalted":record.debilitated_signs.includes(sign)?"debilitated":"unclassified";
          const cell=page.locator(`.natal-table tr[data-planet="${record.planet}"] .graha-sign-status`);
          assert.equal(await cell.getAttribute("data-status"),expected,`${record.planet}/${sign}`);
          assert.equal(await cell.innerText(),expected==="exalted"?(language==="hi"?"उच्च राशि":"Exaltation sign"):expected==="debilitated"?(language==="hi"?"नीच राशि":"Debilitation sign"):"—");
        }
      }
      await page.locator("#qa-sign").selectOption("6");
      for(const width of [320,390,650,700,1440]){
        await page.setViewportSize({width,height:1000});
        assert.equal(await page.locator(".natal-table .graha-sign-status").count(),9);
        assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+2),false,`overflow ${language}/${width}`);
        results.push({language,width,originalBoardLabels:true,overflow:false});
      }
    }
    const scan=await new AxeBuilder({page}).include(".natal-table").include(".graha-table-source").analyze();
    assert.deepEqual(scan.violations.filter(v=>v.impact==="critical"||v.impact==="serious"),[]);
    await page.locator("#qa-hi").click();
    await page.locator(".overview-details-below .planet-panel").screenshot({path:"tmp/source-only-qa/guruji-graha-signs-desktop.png"});
    await page.setViewportSize({width:390,height:1000});
    await page.locator(".overview-details-below .planet-panel").screenshot({path:"tmp/source-only-qa/guruji-graha-signs-mobile.png"});
    await page.locator("#qa-lagna").selectOption("0");
    assert.equal(await page.locator(".natal-table").count(),0);
    assert.deepEqual(errors,[]);
    fs.writeFileSync("tmp/source-only-qa/guruji-graha-signs-validation.json",JSON.stringify({results,planetSignChecks:216,nodalTwoSignConvention:true,sourceImageHash:true,gzipRoundTrip:true,unknownTimeNoClassification:true,seriousAccessibilityViolations:0},null,2));
    console.log(JSON.stringify({planetSignChecks:216,responsiveViews:results.length,nodalTwoSignConvention:true,seriousAccessibilityViolations:0}));
  }finally{if(browser)await browser.close();for(const file of created.reverse())fs.unlinkSync(file);}
}
verify().catch(error=>{console.error(error);process.exitCode=1;});
