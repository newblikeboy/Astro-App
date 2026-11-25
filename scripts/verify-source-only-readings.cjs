const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const fs = require("node:fs");
const assert = require("node:assert/strict");

/** fingerprint distinguishes source branches that share the same category, key and PDF page range. */
function fingerprint(text){
  let hash=2166136261;
  for(let index=0;index<text.length;index++)hash=Math.imul(hash^text.charCodeAt(index),16777619);
  return (hash>>>0).toString(16).padStart(8,"0");
}

/** verify checks original passages, the numerology layout, keyboard disclosures, missing-number behavior and responsive overflow. */
async function verify() {
  const report = JSON.parse(fs.readFileSync("tmp/source-only-qa/report.json", "utf8"));
  const files=[["scripts/fixtures/source-only-qa.html","web/.source-only-qa.html"],["scripts/fixtures/source-only-qa.tsx","web/src/.source-only-qa.tsx"]];
  const created=[];
  let browser;
  const results=[];
  try {
    for(const [source,target] of files){fs.copyFileSync(source,target,fs.constants.COPYFILE_EXCL);created.push(target);}
    browser = await chromium.launch({executablePath:"C:/Program Files/Google/Chrome/Application/chrome.exe",headless:true});
    const context=await browser.newContext({reducedMotion:"reduce"});
    const page=await context.newPage();
    const errors=[];
    page.on("pageerror",error=>errors.push(error.message));
    await page.route("**/api/**",route=>route.abort());
    await page.goto(process.env.SOURCE_QA_URL||"http://127.0.0.1:5197/.source-only-qa.html");
    await page.locator(".kundli-summary").waitFor();
    for(const language of ["hi","en"]){
      await page.locator(`#qa-${language}`).click();
      for(const width of [320,390,1440]){
        await page.setViewportSize({width,height:900});
        const select=page.locator(".kundli-section-select select");
        if(await select.isVisible())await select.selectOption("reading");else await page.locator("#kundli-tab-reading").click();
        const topics=["general","personality","education","career","finance","relationships","family","wellbeing","dasha"];
        assert.deepEqual(await page.locator('.faladesh-topic-tabs [role="tab"]').evaluateAll(tabs=>tabs.map(tab=>tab.id.replace("faladesh-topic-",""))),topics);
        for(const topic of topics){
          await page.locator(`#faladesh-topic-${topic}`).click();
          assert.equal(await page.locator("#faladesh-topic-panel .guruji-unified-reading").count(),1);
          assert.equal(await page.locator("#faladesh-topic-panel details").count(),0);
          const excerpts=await page.locator(".guruji-topic-excerpt").evaluateAll(nodes=>nodes.map(node=>({category:node.dataset.sourceCategory,key:node.dataset.sourceKey,identity:node.dataset.sourceId,parts:[...node.querySelectorAll("[data-source-start]")].map(part=>({start:Number(part.dataset.sourceStart),end:Number(part.dataset.sourceEnd),text:part.textContent}))})));
          for(const excerpt of excerpts){
            const entry=[...report.source_readings,...report.numerology.source_readings].find(p=>p.category===excerpt.category&&p.key===excerpt.key&&excerpt.identity===`${p.category}:${p.key}:${p.pages[0]}:${p.pages[1]}:${fingerprint(p.text)}`);
            assert.ok(entry,`original source for ${topic}`);
            for(const part of excerpt.parts)assert.equal(part.text,entry.text.slice(part.start,part.end),`${topic}: exact PDF words`);
          }
          assert.ok(excerpts.length,`${topic} original excerpts`);
          assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+2),false,`reading overflow ${width}/${language}/${topic}`);
        }
        await page.locator("#faladesh-topic-dasha").focus();
        await page.keyboard.press("Home");
        assert.equal(await page.locator("#faladesh-topic-general").getAttribute("aria-selected"),"true");
        if(language==="hi"&&(width===390||width===1440)){
          await page.locator(".faladesh-topic-tabs").scrollIntoViewIfNeeded();
          await page.screenshot({path:`tmp/source-only-qa/faladesh-restored-${width}.png`});
        }
        const overflow=await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+2);
        assert.equal(overflow,false,`overflow ${width}/${language}`);
        if(await select.isVisible())await select.selectOption("numerology");else await page.locator("#kundli-tab-numerology").click();
        assert.deepEqual(await page.locator(".numerology-panel .guruji-source-text").allTextContents(),report.numerology.source_readings.map(p=>p.text));
        assert.equal(await page.locator(".numerology-profile,.numerology-guidance,.numerology-yantra").count(),0);
        assert.deepEqual(await page.locator(".numerology-summary-card .numerology-value").allTextContents(),[String(report.numerology.mulank.value),String(report.numerology.bhagyank.value)]);
        assert.equal(await page.locator(".numerology-panel details details").count(),0);
        const article=page.locator(".numerology-panel .guruji-source-passage").first();
        await article.locator("summary").focus();
        await page.keyboard.press("Enter");
        assert.equal(await article.evaluate(element=>element.open),true);
        assert.equal(await article.locator(".guruji-source-text").isVisible(),true);
        assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+2),false,`numerology overflow ${width}/${language}`);
        await page.keyboard.press("Space");
        assert.equal(await article.evaluate(element=>element.open),false);
        if(language==="hi"&&(width===390||width===1440))await page.locator(".numerology-panel").screenshot({path:`tmp/source-only-qa/numerology-redesign-${width}.png`});
        results.push({language,width,originalText:true,noAuthoredProfiles:true,overflow:false});
      }
    }
    const normalScan=await new AxeBuilder({page}).include(".numerology-panel").analyze();
    assert.deepEqual(normalScan.violations.filter(v=>v.impact==="critical"||v.impact==="serious"),[]);
    await page.locator("#qa-missing").click();
    const missing=page.locator('[data-number-kind="bhagyank"]');
    assert.equal(await missing.locator(".guruji-source-text").count(),0);
    assert.match(await missing.innerText(),/इस अंक का फल दी गई PDF में उपलब्ध नहीं है/);
    const scan=await new AxeBuilder({page}).include(".numerology-panel").analyze();
    assert.deepEqual(scan.violations.filter(v=>v.impact==="critical"||v.impact==="serious"),[]);
    await page.screenshot({path:"tmp/source-only-qa/numerology-desktop.png",fullPage:true});
    assert.deepEqual(errors,[]);
    fs.writeFileSync("tmp/source-only-qa/browser-validation.json",JSON.stringify({results,missingNumberNoFallback:true,seriousAccessibilityViolations:0,pageErrors:errors},null,2));
    console.log(JSON.stringify({views:results.length,missingNumberNoFallback:true,originalText:true,seriousAccessibilityViolations:0}));
  } finally {if(browser)await browser.close();for(const file of created.reverse())fs.unlinkSync(file);}
}
verify().catch(error=>{console.error(error);process.exitCode=1;});
