// Read-only display review of Raja Babu's anonymous, unsaved public calculation; no account or provider writes.
const {chromium}=require('../web/node_modules/playwright');
const fs=require('node:fs');
const path=require('node:path');
const assert=require('node:assert/strict');

/** verifyRaja checks the audited person's distinct career, finance and personal findings in both languages without saving or ordering a report. */
async function verifyRaja(){
  const directory=path.resolve('tmp/astrosage-faladesh-audit');
  const report=JSON.parse(fs.readFileSync(path.join(directory,'darbhanga-detailed-v4.json'),'utf8'));
  const analysis=report.guruji_faladesh.analysis;
  assert.equal(analysis.version,'m4y-cross-factor-faladesh-v4');
  const career=analysis.topics.find(t=>t.category==='career');
  const finance=analysis.topics.find(t=>t.category==='finance');
  assert.equal(career.findings[0].id,'lord-career-10-in-11');
  assert.equal(finance.findings[0].id,'lord-finance-2-in-10');
  assert(finance.findings[0].basis.en.includes('2, 11'));
  assert.equal(analysis.topics[0].findings[0].id,'overall-occupants-personality-1');
  const browser=await chromium.launch({executablePath:'C:/Program Files/Google/Chrome/Application/chrome.exe',headless:true});
  const errors=[];
  try{
    const context=await browser.newContext({viewport:{width:390,height:1000},reducedMotion:'reduce'});
    await context.addInitScript(value=>{localStorage.setItem('m4y-language','hi');localStorage.setItem('m4y-analytics','no');sessionStorage.setItem('m4y-guest-report',JSON.stringify(value));sessionStorage.setItem('m4y-birth',JSON.stringify(value.birth));},report);
    const page=await context.newPage();
    page.on('pageerror',e=>errors.push(e.message));
    await page.route('**/api/**',route=>route.request().method()==='GET'?route.continue():route.abort());
    await page.goto('http://localhost:5173/kundli');
    await page.locator('.kundli-summary').waitFor();
    await page.locator('.kundli-section-select select').selectOption('reading');
    for(const language of ['hi','en']){
      await page.locator('.language-selector select').selectOption(language);
      for(const width of [390,1440]){
        await page.setViewportSize({width,height:1000});
        for(const category of ['general','personality','career','finance','relationships','dasha']){
          await page.locator(`#faladesh-topic-${category}`).click();
          const topic=analysis.topics.find(t=>t.category===category),reading=page.locator('.guruji-faladesh');
          const main=reading.locator('.guruji-unified-reading');
          assert.equal(await main.locator('h3').count(),1);
          assert.equal(await main.locator('h4,h5').count(),0);
          assert.equal(await reading.locator('.guruji-main-points,.guruji-synthesis-potential,.guruji-finding-basis').count(),0);
          // Plain Hindi can use fewer characters; require a complete next-step sentence rather than padding its length.
          assert((await main.locator('.guruji-reading-paragraph').last().innerText()).trim().split(/\s+/u).length>=8,`${language}/${category}: specific next step missing`);
          assert(!/(?:लग्न|मारक|कारक ग्रह|भाव-संबंध|\bKarak\b|\bMarak\b|ascendant|combust|divisional)/iu.test(await main.innerText()),'technical prose leaked');
          assert((await main.innerText()).length>topic.conclusion[language].length);
          assert.equal(await main.locator('.guruji-reading-paragraph').last().getAttribute('data-scope'),'action');
          assert.equal(await main.locator('.guruji-reading-date').count(),category!=='general'&&topic.findings.some(f=>f.scope==='period')?1:0);
          if(category==='general'){
            assert.equal(await main.locator('h3').innerText(),language==='hi'?'सामान्य फलादेश':'General reading');
            assert.equal(await main.locator('.guruji-reading-paragraph').count(),4);
            assert(!/(?:करियर|कमाई|बचत|भुगतान|ग्राहक|जीवनसाथी|महादशा|अंतर्दशा|career|earnings|savings|payment|clients|spouse|Mahadasha|Antardasha)/iu.test(await main.innerText()),'General contains a dedicated subject forecast');
          }
          assert.equal(await reading.locator('button,a').count(),0);
          assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),'reading overflow');
          if(['general','career','finance'].includes(category))await reading.screenshot({path:path.join(directory,`detailed-${category}-${language}-${width}.png`),style:'header {visibility:hidden !important;}'});
        }
      }
    }
    assert.deepEqual(errors,[]);
    fs.writeFileSync(path.join(directory,'detailed-browser-results.json'),JSON.stringify({passed:true,languages:2,widths:[390,1440],subjects:6,errors},null,2));
    console.log(JSON.stringify({passed:true,person:'Raja Babu',languages:2,widths:[390,1440],subjects:6}));
  }finally{await browser.close();}
}

verifyRaja().catch(error=>{console.error(error);process.exitCode=1;});
