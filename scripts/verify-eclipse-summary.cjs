// Read-only: native eclipse GETs only; no customer/provider mutations.
const {chromium}=require('../web/node_modules/playwright');
const AxeBuilder=require('../web/node_modules/@axe-core/playwright').default;
const {stripTypeScriptTypes}=require('node:module');
const vm=require('node:vm');
const fs=require('node:fs');
const assert=require('node:assert/strict');
const base=process.env.PANCHANG_QA_URL||'http://localhost:5173';
const result={passed:false,layouts:[],audits:0,checks:[],errors:[]};
const moduleExports={};
vm.runInNewContext(stripTypeScriptTypes(fs.readFileSync('web/src/eclipse-summary.ts','utf8')).replace(/^export /gm,'')+'\nObject.assign(exports,{eclipseNeighbours,eclipseSearchYears});',{exports:moduleExports,Date,Intl,Number});
const fixture=[
  {kind:'Solar',maximum:'2025-12-01T00:00:00Z'}, {kind:'Solar',maximum:'2026-02-01T00:00:00Z'},
  {kind:'Solar',maximum:'2027-03-01T00:00:00Z'}, {kind:'Lunar',maximum:'2025-11-01T00:00:00Z'},
  {kind:'Lunar',maximum:'2026-04-01T00:00:00Z'}, {kind:'Lunar',maximum:'2027-01-01T00:00:00Z'},
];
const at=Date.parse('2026-02-01T00:00:00Z');
const picks=moduleExports.eclipseNeighbours([...fixture].reverse(),at);
assert.equal(picks[0].previous.maximum,'2026-02-01T00:00:00Z');
assert.equal(picks[0].next.maximum,'2027-03-01T00:00:00Z');
assert.equal(picks[1].previous.maximum,'2025-11-01T00:00:00Z');
assert.equal(picks[1].next.maximum,'2026-04-01T00:00:00Z');
assert.equal(fixture[0].maximum,'2025-12-01T00:00:00Z');
assert.deepEqual(Array.from(moduleExports.eclipseSearchYears(Date.parse('2025-12-31T19:00:00Z'),1940,2099)),[2025,2026,2027]);
assert.deepEqual(Array.from(moduleExports.eclipseSearchYears(Date.parse('1940-01-01T00:00:00Z'),1940,2099)),[1940,1941]);
assert.deepEqual(Array.from(moduleExports.eclipseSearchYears(Date.parse('2099-12-01T00:00:00Z'),1940,2099)),[2098,2099]);
result.checks.push('Independent boundary fixtures: exact maximum, mixed/unsorted kinds, adjacent years, India New Year and supported range edges');
(async()=>{
  const native=[];
  for(const year of [2025,2026,2027]){
    const response=await fetch(base+`/api/sky/eclipses?year=${year}&place_id=delhi`);
    assert(response.ok,await response.clone().text());native.push(...(await response.json()).events);
  }
  const browser=await chromium.launch({executablePath:'C:/Program Files/Google/Chrome/Application/chrome.exe',headless:true});
  try {
    const context=await browser.newContext({viewport:{width:1440,height:1000},reducedMotion:'reduce'});
    await context.addInitScript(()=>{localStorage.setItem('m4y-language','en');localStorage.setItem('m4y-analytics','no');localStorage.removeItem('m4y-panchang-place')});
    const page=await context.newPage();page.setDefaultTimeout(20000);page.on('pageerror',e=>result.errors.push(e.message));
    await page.route('**/api/**',route=>{assert.equal(route.request().method(),'GET');return route.continue()});
    /** open fixes the browser reference clock and opens the four previous/next eclipse cards. */
    const open=async(reference)=>{
      await page.clock.setFixedTime(new Date(reference));
      await page.goto(base+'/?eclipse-reference='+encodeURIComponent(reference)+'#panchang');await page.locator('.daily-tabs [role=tab]').nth(5).click();await page.locator('.eclipse-neighbour').first().waitFor();
      assert.equal(await page.locator('.eclipse-neighbour').count(),4);
      assert.equal(await page.locator('.eclipse-summary #sky-year,.eclipse-summary .sky-switch,.eclipse-summary details').count(),0);
      assert.equal(await page.locator('.eclipse-summary #daily-city').count(),1);
      for(const kind of ['Solar','Lunar']){
        const source=native.filter(e=>e.kind===kind);
        const past=source.filter(e=>Date.parse(e.maximum)<=Date.parse(reference)).reduce((a,b)=>Date.parse(a.maximum)>Date.parse(b.maximum)?a:b);
        const future=source.filter(e=>Date.parse(e.maximum)>Date.parse(reference)).reduce((a,b)=>Date.parse(a.maximum)<Date.parse(b.maximum)?a:b);
        for(const [phase,e] of [['previous',past],['next',future]]){
          const card=page.locator(`.eclipse-neighbour[data-kind="${kind}"][data-phase="${phase}"]`);
          assert.equal(await card.locator('.eclipse-summary-date').getAttribute('datetime'),e.maximum,reference+'/'+kind+'/'+phase);
          assert.equal(await card.locator('.eclipse-summary-type strong').innerText(),e.type);
          assert.equal(await card.locator('.is-visible').count(),e.visible?1:0);
        }
      }
    };
    for(const reference of ['2026-01-01T09:00:00Z','2026-12-31T09:00:00Z','2026-10-20T09:00:00Z'])await open(reference);
    result.checks.push('All four rendered selections match native annual API values at January, October and December references, including next-year events');
    for(const language of ['en','hi']){
      await page.locator('.language-selector select').selectOption(language);
      for(const width of [320,339,390,650,768,1100,1440]){
        await page.bringToFront();await page.setViewportSize({width,height:1000});await page.evaluate(()=>document.fonts.ready);
        await page.evaluate(()=>new Promise(r=>{setTimeout(r,400);requestAnimationFrame(()=>requestAnimationFrame(r))}));
        const state=await page.locator('#panchang').evaluate(el=>({overflow:document.documentElement.scrollWidth-innerWidth,clipped:[...el.querySelectorAll('.eclipse-neighbour,.eclipse-summary-date,.eclipse-peak,input')].filter(n=>{const r=n.getBoundingClientRect();return r.width&&(r.left < -1||r.right>innerWidth+1||n.scrollWidth>n.clientWidth+1)}).map(n=>n.className)}));
        assert(state.overflow<=1&&!state.clipped.length,JSON.stringify({language,width,state}));
        assert(!/\b\d{1,2}:\d{2}:\d{2}\b/.test(await page.locator('.eclipse-summary').innerText()),'Visible seconds');
        if([339,1440].includes(width)){
          const scan=await new AxeBuilder({page}).setLegacyMode().include('#panchang').analyze();assert.deepEqual(scan.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>n.target)})),[]);result.audits++;
          await page.locator('#panchang').screenshot({path:`.local/eclipse-summary-${language}-${width}.png`});
        }
        result.layouts.push({language,width});
      }
    }
    await page.locator('.language-selector select').selectOption('en');
    let fail=true;
    await page.route('**/api/sky/eclipses?**',route=>fail?route.fulfill({status:503,json:{error:'Synthetic temporary failure'}}):route.fallback());
    await page.reload();await page.locator('.daily-tabs [role=tab]').nth(5).click();await page.locator('.eclipse-summary [role=alert]').waitFor();assert.equal(await page.locator('.eclipse-neighbour').count(),0);
    fail=false;await page.locator('.eclipse-summary').getByRole('button',{name:'Try again',exact:true}).click();await page.locator('.eclipse-neighbour').first().waitFor();
    result.checks.push('Error/retry hides stale results; city control remains inside the panel; minute-only clocks');
    assert.deepEqual(result.errors,[]);result.passed=true;console.log(JSON.stringify({passed:true,layouts:result.layouts.length,audits:result.audits,checks:result.checks}));
  }finally{fs.writeFileSync('.local/eclipse-summary-browser-results.json',JSON.stringify(result,null,2));await browser.close()}
})().catch(e=>{console.error(e);process.exitCode=1});
