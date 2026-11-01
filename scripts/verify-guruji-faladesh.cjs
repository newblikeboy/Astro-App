// Synthetic browser snapshots only; no customer, account, payment or provider writes.
const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const fs = require("node:fs");
const path = require("node:path");
const assert = require("node:assert/strict");

/** verify checks complete category readings, absence of technical source panels, period boundaries and bilingual responsive access. */
async function verify() {
  const directory=path.resolve(process.env.GURUJI_QA_DIR||"tmp/guruji/qa");
  const report=JSON.parse(fs.readFileSync(path.join(directory,"report.json"),"utf8"));
  const browser=await chromium.launch({executablePath:"C:/Program Files/Google/Chrome/Application/chrome.exe",headless:true});
  const results={passed:false,layouts:[],errors:[]};
  try {
    const context=await browser.newContext({reducedMotion:"reduce"});
    await context.addInitScript(value=>{
      localStorage.setItem("m4y-language","en");
      localStorage.setItem("m4y-analytics","no");
      sessionStorage.setItem("m4y-birth",JSON.stringify(value.birth));
      sessionStorage.setItem("m4y-guest-report",JSON.stringify(value));
    },report);
    const page=await context.newPage();
    page.on("pageerror",error=>results.errors.push(error.message));
    await page.route("**/api/**",route=>route.request().method()==="GET"?route.continue():route.abort());
    await page.goto(process.env.GURUJI_QA_URL||"http://localhost:5173/kundli");
    await page.locator(".kundli-summary").waitFor();
    for(const language of ["en","hi"]){
      await page.locator(".language-selector select").selectOption(language);
      for(const width of [320,390,768,1440]){
        await page.setViewportSize({width,height:1000});
        const selector=page.locator(".kundli-section-select select");
        if(await selector.isVisible())await selector.selectOption("reading");
        else await page.locator("#kundli-tab-reading").click();
        for(const category of ["general","personality","education","career","finance","relationships","family","wellbeing","dasha"]){
          await page.locator(`#faladesh-topic-${category}`).click();
          const reading=page.locator(".guruji-faladesh");
          assert.equal(await reading.locator("button,a").count(),0,"unnecessary reading navigation");
          assert.equal(await reading.locator(".guruji-conclusion").count(),1);
          const combined=report.guruji_faladesh.analysis?.topics.find(item=>item.category===category);
          if(combined){
            const unified=!!combined.reader?.length||!!combined.findings?.length;
            if(unified){
              const main=reading.locator('.guruji-unified-reading');
              const paragraphs=main.locator('.guruji-reading-paragraph');
              assert.equal(await main.locator('h3').count(),1);
              assert.equal(await main.locator('h4,h5').count(),0,'category split into subheadings');
              assert.equal(await reading.locator('.guruji-main-points,.guruji-synthesis-potential,.guruji-finding-basis').count(),0,'reading still split into separate sections');
              const expected=await page.evaluate(async({topic,language,report})=>{
                const {readingParagraphs}=await import('/src/faladesh-reading.ts');
                return readingParagraphs(topic,language,{analysis:report.guruji_faladesh.analysis,report});
              },{topic:combined,language,report});
              assert.equal(await paragraphs.count(),expected.length);
              const displayed=[];
              for(let i=0;i<expected.length;i++){
                const text=await paragraphs.nth(i).evaluate(node=>{const copy=node.cloneNode(true);copy.querySelectorAll('.guruji-reading-date,.guruji-reading-context').forEach(item=>item.remove());return copy.textContent.trim();});
                assert.equal(text,expected[i].text);
                displayed.push(text);
                assert.equal(await paragraphs.nth(i).getAttribute('data-scope'),expected[i].scope);
              }
              const sentences=displayed.flatMap(text=>(text.match(/[^.!?\u0964]+[.!?\u0964]*/gu)||[]).map(value=>value.trim().replace(/\s+/gu,' ')));
              assert(!/(?:लग्न|मारक|कारक ग्रह|भाव-संबंध|वर्ग-कुंडली|\bKarak\b|\bMarak\b|ascendant|combust|divisional)/iu.test(displayed.join(' ')),'technical qualification leaked into reader prose');
              for(const source of expected.map(paragraph=>paragraph.text)){
                for(const sentence of source.match(/[^.!?\u0964]+[.!?\u0964]*/gu)||[]){
                  const normalized=sentence.trim().replace(/\s+/gu,' ');
                  if(normalized)assert.equal(sentences.filter(value=>value===normalized).length,1,'detail omitted or sentence repeated');
                }
              }
              assert((await main.innerText()).length>combined.conclusion[language].length,'expanded content missing from category');
              assert.equal(await paragraphs.last().getAttribute('data-scope'),'action','guidance is not part of category reading');
              const periodIndex=expected.findIndex(p=>p.scope==='period');
              assert.equal(await main.locator('.guruji-reading-date').count(),periodIndex>=0?1:0,'period reference lost or repeated');
            }else{
              assert.equal(await reading.locator('.guruji-conclusion-text').innerText(),combined.conclusion[language]);
              assert.equal(await reading.locator('.guruji-synthesis-potential > p').first().innerText(),combined.potential[language]);
              const points=reading.locator('.guruji-main-points li');
              assert.equal(await points.count(),combined.main_points.length);
              for(let index=0;index<combined.main_points.length;index++)assert.equal(await points.nth(index).innerText(),combined.main_points[index].text[language]);
            }
            assert.equal(await reading.locator('.guruji-source-reference,.guruji-planet-facts,.guruji-conclusion-basis').count(),0,'technical reference panel in reader view');
            const conclusion=combined.conclusion[language];
            assert(conclusion.length<750,`${category}: conclusion too long`);
            assert(!report.guruji_faladesh.readings.some(item=>item.narrative&&conclusion.includes(item.narrative.meaning[language])),`${category}: unrelated natal source text leaked`);
            if(category==="education")assert(!/daily work|working style|कार्यशैली/i.test(conclusion),"career content in Education");
            if(category==="relationships")assert(!/writing|research|लेखन|शोध/i.test(conclusion),"education content in Relationships");
          } else {
          assert.equal(await reading.locator('.guruji-conclusion-basis,.guruji-source-reference').count(),0);
          const anchors=(await reading.locator('.guruji-conclusion').getAttribute('data-anchors')).split(',');
          const conclusion=await reading.locator(".guruji-conclusion-text").innerText();
          for(const planet of anchors){
            const source=report.guruji_faladesh.readings.find(item=>item.planet===planet);
            assert(conclusion.includes((source.narrative?.meaning||source.summary)[language]),"ungrounded conclusion");
          }
          assert(!/\?{3,}|undefined|Invalid Date/.test(await reading.innerText()),`${language}/${category}: broken copy`);
          assert.equal(await reading.locator(".guruji-period-context").count(),category==="dasha"?1:0,"repeated dasha block");
          if(category==="dasha")assert.equal(await reading.locator("time").count(),4);
          if(category==="career"&&report.guruji_faladesh.lagna===6)assert.deepEqual(anchors,["Mercury","Saturn"],"empty career house lost its ruler");
          }
          assert(!/\?{3,}|undefined|Invalid Date/.test(await reading.innerText()),`${language}/${category}: broken copy`);
          assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),`${language}/${width}/${category}: overflow`);
          if(["general","education","career","relationships"].includes(category)&&[390,1440].includes(width)){
            // Fixed headers are omitted only from component capture, avoiding a screenshot overlay across the text.
            await reading.screenshot({path:path.join(directory,`combined-${category}-${language}-${width}.png`),style:"header { visibility: hidden !important; }"});
          }
        }
        await page.locator("#faladesh-topic-general").click();
        assert.equal(await page.locator('.guruji-source-reference').count(),0);
        // Main tab navigation remains the single way to change subjects, including by keyboard.
        await page.locator("#faladesh-topic-general").focus();
        await page.keyboard.press("ArrowRight");
        assert.equal(await page.locator("#faladesh-topic-personality").getAttribute("aria-selected"),"true");
        results.layouts.push({language,width});
      }
      for(const category of ["general","career"]){
        await page.locator(`#faladesh-topic-${category}`).click();
        const audit=await new AxeBuilder({page}).include(".guruji-faladesh").analyze();
        assert.deepEqual(audit.violations.map(item=>({id:item.id,impact:item.impact})),[],`${language}/${category} accessibility`);
      }
    }
    const prose=await page.evaluate(async()=>{
      const {readingParagraphs}=await import('/src/faladesh-reading.ts');
      const topic={potential:{hi:'अगला कदम अपनाएँ।',en:'Apply the next step.'},findings:[
        {id:'first',scope:'natal',text:{hi:'पहली बात। अगली बात।',en:'First point. Further detail.'},qualification:{hi:'सहायक बात।',en:'Supporting point.'},evidence_ids:[]},
        {id:'second',scope:'natal',text:{hi:'पहली बात। दूसरी व्याख्या।',en:'First point. Another explanation.'},qualification:{hi:'सहायक बात।',en:'Supporting point.'},evidence_ids:[]},
      ]};
      return ['hi','en'].map(language=>readingParagraphs(topic,language).map(p=>p.text));
    });
    assert.deepEqual(prose,[['पहली बात। अगली बात। सहायक बात।','दूसरी व्याख्या।','अगला कदम अपनाएँ।'],['First point. Further detail. Supporting point.','Another explanation.','Apply the next step.']]);
    const boundaries=await page.evaluate(async()=>{
      const {savedPeriodContext,conclusionForCategory}=await import('/src/faladesh-reading.ts');
      const value=JSON.parse(sessionStorage.getItem('m4y-guest-report'));
      value.dasha={as_of:'2026-01-01T00:00:00Z',current_lord:'Saturn',current_sub_lord:'Moon',current_path:[{lord:'Wrong',start:'1900-01-01',end:'1901-01-01'}],periods:[{lord:'Saturn',start:'2025-01-01T00:00:00Z',end:'2027-01-01T00:00:00Z',antardashas:[{lord:'Sun',start:'2025-01-01T00:00:00Z',end:'2026-01-01T00:00:00Z'},{lord:'Moon',start:'2026-01-01T00:00:00Z',end:'2027-01-01T00:00:00Z'}]}]};
      const inside=savedPeriodContext(value);
      value.dasha.as_of='2027-01-01T00:00:00Z';
      const boundary=savedPeriodContext(value);
      value.dasha.as_of='invalid';
      const invalid=savedPeriodContext(value);
      value.guruji_faladesh.readings.forEach(item=>delete item.narrative);
      const legacy=conclusionForCategory(value.guruji_faladesh,value,'general');
      const fallback=legacy.meaning.every(item=>value.guruji_faladesh.readings.some(reading=>reading.summary.en===item.en));
      return {major:inside.major.lord,sub:inside.sub.lord,boundary:!!boundary.major,invalid:invalid===undefined,legacySupport:legacy.support.length,fallback};
    });
    assert.deepEqual(boundaries,{major:'Saturn',sub:'Moon',boundary:false,invalid:true,legacySupport:0,fallback:true});
    const aliases=await page.evaluate(async()=>{
      const {normalizeReadingCategory}=await import('/src/faladesh-reading.ts');
      return ['marriage','health','debt',' Career ','',undefined].map(normalizeReadingCategory);
    });
    assert.deepEqual(aliases,['relationships','wellbeing','finance','career','general','general']);
    // Reopen a superseded guest snapshot: exactly one compute request, with its original birth details and no persistence.
    const superseded=structuredClone(report);
    superseded.guruji_faladesh.analysis.version='m4y-cross-factor-faladesh-v1';
    const upgradeContext=await browser.newContext({viewport:{width:1440,height:1000}});
    await upgradeContext.addInitScript(value=>{localStorage.setItem('m4y-language','en');localStorage.setItem('m4y-analytics','no');sessionStorage.setItem('m4y-guest-report',JSON.stringify(value));},superseded);
    const upgradePage=await upgradeContext.newPage();
    let computations=0;
    await upgradePage.route('**/api/**',async route=>{
      if(route.request().method()==='GET')return route.continue();
      assert.equal(new URL(route.request().url()).pathname,'/api/kundli','unexpected persistence request');
      assert.deepEqual(route.request().postDataJSON(),report.birth);
      computations++;
      return route.fulfill({status:200,contentType:'application/json',body:JSON.stringify(report)});
    });
    await upgradePage.goto(process.env.GURUJI_QA_URL||'http://localhost:5173/kundli');
    await upgradePage.locator('.kundli-summary').waitFor();
    await upgradePage.locator('#kundli-tab-reading').click();
    await upgradePage.waitForFunction(version=>JSON.parse(sessionStorage.getItem('m4y-guest-report')).guruji_faladesh.analysis.version===version,report.guruji_faladesh.analysis.version);
    await upgradePage.locator('#faladesh-topic-education').click();
    assert((await upgradePage.locator('.guruji-unified-reading').innerText()).length>300,'refreshed subject detail missing');
    assert(await upgradePage.locator('.guruji-reading-paragraph[data-evidence*="placement:"]').count()>0,'refreshed reading lost its causes');
    await upgradePage.locator('#faladesh-topic-career').click();
    assert.equal(computations,1,'guest refresh repeated');
    await upgradeContext.close();
    const legacy=structuredClone(report);
    delete legacy.guruji_faladesh;
    const legacyContext=await browser.newContext({viewport:{width:1440,height:1000}});
    await legacyContext.addInitScript(value=>{
      localStorage.setItem("m4y-analytics","no");
      sessionStorage.setItem("m4y-guest-report",JSON.stringify(value));
    },legacy);
    const legacyPage=await legacyContext.newPage();
    await legacyPage.route("**/api/**",route=>route.request().method()==="GET"?route.continue():route.abort());
    await legacyPage.goto(process.env.GURUJI_QA_URL||"http://localhost:5173/kundli");
    await legacyPage.locator(".kundli-summary").waitFor();
    await legacyPage.locator("#kundli-tab-reading").click();
    assert.equal(await legacyPage.locator(".guruji-faladesh").count(),0,"legacy snapshot got a new reading");
    await legacyContext.close();
    const retained=structuredClone(report);
    delete retained.guruji_faladesh.analysis;
    const retainedContext=await browser.newContext({viewport:{width:1440,height:1000}});
    await retainedContext.addInitScript(value=>{localStorage.setItem("m4y-analytics","no");sessionStorage.setItem("m4y-guest-report",JSON.stringify(value));},retained);
    const retainedPage=await retainedContext.newPage();
    await retainedPage.route("**/api/**",route=>route.request().method()==="GET"?route.continue():route.abort());
    await retainedPage.goto(process.env.GURUJI_QA_URL||"http://localhost:5173/kundli");
    await retainedPage.locator(".kundli-summary").waitFor();
    await retainedPage.locator("#kundli-tab-reading").click();
    assert.equal(await retainedPage.locator(".guruji-combined-reading").count(),0,"retained report got live analysis");
    assert.equal(await retainedPage.locator(".guruji-conclusion-basis").count(),0);
    await retainedContext.close();
    assert.deepEqual(results.errors,[]);
    results.passed=true;
    fs.writeFileSync(path.join(directory,"browser-results.json"),JSON.stringify(results,null,2));
    console.log(JSON.stringify(results));
  }finally{await browser.close();}
}

verify().catch(error=>{console.error(error);process.exitCode=1;});
