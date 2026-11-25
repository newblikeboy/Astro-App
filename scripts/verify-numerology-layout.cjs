const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const fs = require("node:fs");
const zlib = require("node:zlib");
const crypto = require("node:crypto");
const assert = require("node:assert/strict");

/** verify checks lossless paragraph reconstruction, visible heading hierarchy, real justified line widths and source-change guards. */
async function verify() {
  const catalog=JSON.parse(zlib.gunzipSync(fs.readFileSync("internal/gurujitext/data/catalog.json.gz")));
  const formatting=JSON.parse(fs.readFileSync("web/src/data/guruji-numerology-layout.json","utf8"));
  const source=catalog.sources[0];
  assert.equal(crypto.createHash("sha256").update(fs.readFileSync(`references/guruji/2026-11-24/${source.file}`)).digest("hex"),formatting.source_sha256);
  const fixtures=Object.entries(catalog).filter(([category])=>category==="मूलांक"||category==="भाग्यांक").flatMap(([category,values])=>Object.entries(values).flatMap(([key,articles])=>articles.map(article=>({...article,category,key,file:source.file,sha256:source.sha256}))));
  assert.equal(fixtures.length,17);
  for(const entry of fixtures){
    const layout=formatting.layouts[`${entry.category}:${entry.key}`];
    assert.equal(layout.length,entry.text.length);
    let start=0;
    const reconstruction=layout.blocks.map(([end])=>{assert.ok(end>start&&end<=entry.text.length);const text=entry.text.slice(start,end);start=end;return text;}).join("");
    assert.equal(reconstruction,entry.text);
  }
  fs.writeFileSync("tmp/source-only-qa/numerology-format-fixtures.json",JSON.stringify(fixtures));
  const files=[["scripts/fixtures/source-only-qa.html","web/.source-only-qa.html"],["scripts/fixtures/numerology-reader-qa.tsx","web/src/.source-only-qa.tsx"]];
  const created=[];
  let browser;
  const results=[];
  try{
    for(const [source,target] of files){fs.copyFileSync(source,target,fs.constants.COPYFILE_EXCL);created.push(target);}
    browser=await chromium.launch({executablePath:"C:/Program Files/Google/Chrome/Application/chrome.exe",headless:true});
    const context=await browser.newContext({reducedMotion:"reduce",viewport:{width:1440,height:1150}});
    const page=await context.newPage();
    const errors=[];
    page.on("pageerror",error=>errors.push(error.message));
    await page.route("**/api/**",route=>route.abort());
    await page.goto(process.env.SOURCE_QA_URL||"http://localhost:5173/.source-only-qa.html");
    for(let index=0;index<fixtures.length;index++){
      await page.locator("#qa-source").selectOption(String(index));
      await page.locator(".guruji-source-passage > summary").click();
      const body=page.locator(".guruji-source-text");
      assert.equal(await body.getAttribute("data-layout"),"pdf",`${fixtures[index].category}/${fixtures[index].key}`);
      assert.equal(await body.textContent(),fixtures[index].text);
      assert.equal(await body.locator("h4").count(),1);
      assert.ok(await body.locator("h5").count()>=8);
      assert.ok(await body.locator("p").count()>20);
      if(fixtures[index].category==="मूलांक"&&fixtures[index].key==="3"){
        assert.ok(await body.locator("strong").count()>10);
        assert.doesNotMatch(await body.innerText(),/\*\*|^##/m);
      }
    }
    await page.locator("#qa-source").selectOption("1");
    await page.locator(".guruji-source-passage > summary").click();
    for(const width of [320,390,768,1440]){
      await page.setViewportSize({width,height:1150});
      const body=page.locator(".guruji-formatted-text");
      const metrics=await body.evaluate(element=>{
        const paragraph=element.querySelector("p:not([class])");
        const range=document.createRange();range.selectNodeContents(paragraph);
        const firstLine=Array.from(range.getClientRects()).find(rect=>rect.width>0);
        const title=element.querySelector("h4"),heading=element.querySelector("h5");
        return {align:getComputedStyle(paragraph).textAlign,whiteSpace:getComputedStyle(paragraph).whiteSpace,paragraphWidth:paragraph.getBoundingClientRect().width,firstLineWidth:firstLine.width,titleFont:parseFloat(getComputedStyle(title).fontSize),bodyFont:parseFloat(getComputedStyle(paragraph).fontSize),headingWeight:Number(getComputedStyle(heading).fontWeight),headingBackground:getComputedStyle(heading).backgroundColor};
      });
      assert.equal(metrics.align,"justify");assert.equal(metrics.whiteSpace,"normal");
      assert.ok(metrics.firstLineWidth>=metrics.paragraphWidth*.95,`PDF line breaks must reflow ${width}`);
      assert.ok(metrics.titleFont>metrics.bodyFont*1.25);assert.ok(metrics.headingWeight>=600);
      assert.notEqual(metrics.headingBackground,"rgba(0, 0, 0, 0)");
      assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+2),false);
      results.push({width,...metrics,overflow:false});
      if(width===1440){await page.evaluate(()=>scrollTo(0,0));await page.screenshot({path:"tmp/source-only-qa/numerology-paragraphs-desktop.png"});}
      if(width===390){await body.locator("h5").first().scrollIntoViewIfNeeded();await page.screenshot({path:"tmp/source-only-qa/numerology-paragraphs-mobile.png"});}
    }
    const scan=await new AxeBuilder({page}).include(".guruji-formatted-text").analyze();
    assert.deepEqual(scan.violations.filter(v=>v.impact==="critical"||v.impact==="serious"),[]);
    await page.locator("#qa-modified").click();
    await page.locator(".guruji-source-passage > summary").click();
    assert.equal(await page.locator(".guruji-source-text").getAttribute("data-layout"),"flow");
    assert.equal(await page.locator(".guruji-source-text").textContent(),fixtures[1].text+" ");
    assert.deepEqual(errors,[]);
    fs.writeFileSync("tmp/source-only-qa/numerology-paragraphs-validation.json",JSON.stringify({articles:17,exactOriginalText:true,formattingGuard:true,results,seriousAccessibilityViolations:0},null,2));
    console.log(JSON.stringify({articles:17,exactOriginalText:true,justifiedWidths:results.length,formattingGuard:true,seriousAccessibilityViolations:0}));
  }finally{if(browser)await browser.close();for(const file of created.reverse())fs.unlinkSync(file);}
}
verify().catch(error=>{console.error(error);process.exitCode=1;});
