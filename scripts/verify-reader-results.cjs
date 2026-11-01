// Read-only public-page QA. Only stateless matching and Dasha requests are allowed; no account, order or provider changes.
const {chromium}=require('../web/node_modules/playwright');
const fs=require('node:fs');
const assert=require('node:assert/strict');
const report=JSON.parse(fs.readFileSync(process.env.READER_QA_REPORT||'tmp/guruji/qa/report.json','utf8'));
const matchingFixture=process.env.READER_QA_MATCHING?JSON.parse(fs.readFileSync(process.env.READER_QA_MATCHING,'utf8')):undefined;
const output='tmp/reader-results';
const base='http://localhost:5173';
const technical='.guruji-source-reference,.guruji-conclusion-basis,.faladesh-chart-reference,.dasha-method,.matching-method,.matching-print-reference,.matching-checked-rules,.sky-method,.sky-engine,.sky-combustion-evidence,.vivah-method,#daily-convention,#vivah-sunrise,#kundli-tab-settings';

/** inspect verifies that reader pages retain meaningful content without method/source panels, formula headings or horizontal overflow. */
async function inspect(page,label){
 assert.equal(await page.locator(technical).count(),0,`${label}: technical panel remains`);
 const text=await page.locator('main').innerText();
 assert(!/Calculation reference|Calculation details|calculation method|calculation rules|calculation settings|Swiss Ephemeris|Julian day|गणना और मूल|गणना संदर्भ|गणना का आधार|गणना की विधि/i.test(text),`${label}: technical wording remains: ${text.match(/.{0,80}(?:Calculation reference|Calculation details|calculation method|calculation rules|calculation settings|Swiss Ephemeris|Julian day).{0,120}/i)?.[0]}`);
 assert(text.trim().length>100,`${label}: content missing`);
 assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),`${label}: overflow`);
}

/** showSection opens a report result section through the visible mobile or desktop navigation. */
async function showSection(page,id){
 const select=page.locator('.kundli-section-select select');
 if(await select.isVisible())await select.selectOption(id);else await page.locator(`#kundli-tab-${id}`).click();
}

/** verify checks end-result views across Kundli, numerology, matching, daily sky tools and marriage windows in both languages. */
async function verify(){
 fs.mkdirSync(output,{recursive:true});
 const browser=await chromium.launch({executablePath:'C:/Program Files/Google/Chrome/Application/chrome.exe',headless:true});
 const results={passed:false,checks:[],errors:[]};
 try{
  const context=await browser.newContext({viewport:{width:390,height:1000},reducedMotion:'reduce'});
  await context.addInitScript(value=>{localStorage.setItem('m4y-language','en');localStorage.setItem('m4y-analytics','no');sessionStorage.setItem('m4y-guest-report',JSON.stringify(value));sessionStorage.setItem('m4y-birth',JSON.stringify(value.birth));},report);
  const page=await context.newPage();
  page.on('pageerror',e=>results.errors.push(e.message));
  await page.route('**/api/**',route=>{
   const request=route.request(),url=new URL(request.url());
   if(matchingFixture&&request.method()==='POST'&&url.pathname==='/api/kundli/match'){
    for(const role of ['bride','groom']){
     const supplied=request.postDataJSON()[role],expected=matchingFixture[role].birth;
     for(const field of ['date','place_id'])assert.equal(supplied[field],expected[field],'mock matching birth differs from native fixture');
     assert.equal(supplied.time.replace(/:00$/u,''),expected.time.replace(/:00$/u,''));
    }
    return route.fulfill({status:200,contentType:'application/json',body:JSON.stringify(matchingFixture)});
   }
   if(request.method()==='GET'||(request.method()==='POST'&&['/api/kundli/match','/api/kundli/dasha','/api/dasha/children'].includes(url.pathname)))return route.continue();
   return route.abort();
  });
  for(const language of ['en','hi']){
   for(const width of [390,1440]){
    await page.setViewportSize({width,height:1000});
    await page.goto(base+'/kundli');
    await page.locator('.kundli-summary').waitFor();
    await page.locator('.language-selector select').selectOption(language);
    assert(!await page.locator('.kundli-section-select select').locator('option[value=settings]').count());
    for(const id of ['overview','birth','numerology','charts','reading','dasha','planets','checks','transits']){
     await showSection(page,id);
     await inspect(page,`${language}/${width}/kundli/${id}`);
     if(id==='numerology')assert.equal(await page.locator('.numerology-value').count(),report.numerology.extended?.naamank?3:2,'number results missing');
     if(id==='reading'){
      assert((await page.locator('.guruji-reading-prose').innerText()).length>200,'reading removed with references');
      if(width===390)await page.locator('.guruji-faladesh').screenshot({path:`${output}/reading-${language}.png`,style:'header{visibility:hidden!important}'});
     }
    }
    await page.goto(base+'/kundli-matching');
    await page.locator('#matching-bride-name').waitFor();
    for(const role of ['bride','groom']){
     await page.locator(`#matching-${role}-name`).fill(matchingFixture?.[role].birth.name||(role==='bride'?'Reader One':'Reader Two'));
     await page.locator(`#matching-${role}-date`).fill(matchingFixture?.[role].birth.date||(role==='bride'?'1995-08-15':'1995-11-29'));
     await page.locator(`#matching-${role}-time`).fill(matchingFixture?.[role].birth.time||(role==='bride'?'10:30':'12:35'));
     const card=page.locator('.matching-birth-card').nth(role==='bride'?0:1),search=card.getByRole('combobox');
     await search.fill(matchingFixture?.[role].place.name||(role==='bride'?'New Delhi':'Darbhanga'));
     await card.getByRole('option').first().waitFor();
     await search.press('ArrowDown');await search.press('Enter');
     await card.locator('.matching-place-input.is-selected').waitFor();
    }
    await page.locator('.matching-submit').click();
    await page.locator('.matching-results').waitFor({timeout:60000});
    for(const id of ['guna','mangal','charts','birth']){
     await page.locator(`#matching-result-tab-${id}`).click();
     await inspect(page,`${language}/${width}/matching/${id}`);
    }
    await page.goto(base+'/');
    await page.locator('#daily-tab-0').waitFor();
    await page.locator('.daily-solar').waitFor({timeout:60000});
    for(const index of [0,1,2,3,4]){
     await page.locator(`#daily-tab-${index}`).click();
     if(index<4)await page.locator('.daily-solar').waitFor();
     else{
      await page.locator('.sky-tools').waitFor();
      for(let mode=0;mode<4;mode++){
       await page.locator('.sky-switch button').nth(mode).click();
       await page.locator('.sky-tools .daily-status[role=status]').waitFor({state:'hidden',timeout:60000});
       assert.equal(await page.locator('.sky-tools [role=alert]').count(),0,'sky results unavailable');
       await inspect(page,`${language}/${width}/sky/${mode}`);
      }
     }
     await inspect(page,`${language}/${width}/daily/${index}`);
    }
    await page.goto(base+'/vivah-muhurat?month=2026-11&place_id=delhi');
    await page.locator('.vivah-day').first().waitFor({timeout:60000});
    await inspect(page,`${language}/${width}/vivah`);
    assert(await page.locator('.vivah-windows time').count()>0,'marriage windows missing');
    if(width===390)await page.locator('.vivah-hero').screenshot({path:`${output}/vivah-${language}.png`});
    results.checks.push({language,width,reportSections:9,matchingSections:4,dailyViews:5,skyViews:4,vivah:true});
   }
  }
  assert.deepEqual(results.errors,[]);
  results.passed=true;fs.writeFileSync(`${output}/results.json`,JSON.stringify(results,null,2));console.log(JSON.stringify(results));
 }finally{await browser.close();}
}
verify().catch(e=>{console.error(e);process.exitCode=1;});
