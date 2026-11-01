// Public layout checks use a saved anonymous fixture and never submit account, booking or order changes.
const {chromium}=require('../web/node_modules/playwright');
const fs=require('node:fs');
const assert=require('node:assert/strict');
const output='tmp/public-layout';
const widths=[320,390,768,1440,1920];
const report=JSON.parse(fs.readFileSync('tmp/astrosage-faladesh-audit/darbhanga-detailed-v4.json','utf8'));

/** inspectLayout checks shared header/content edges and viewport containment after fonts settle. */
async function inspectLayout(page,selector,label){
 await page.locator(selector).first().waitFor();
 await page.evaluate(()=>document.fonts.ready);
 const geometry=await page.evaluate(selector=>{
  const element=document.querySelector(selector),style=getComputedStyle(element),box=element.getBoundingClientRect();
  const header=document.querySelector('.nav-wrap,.portal-header,.shop-header'),headerBox=header.getBoundingClientRect(),headerStyle=getComputedStyle(header);
  return {left:box.left+parseFloat(style.paddingLeft),right:box.right-parseFloat(style.paddingRight),headerLeft:headerBox.left+parseFloat(headerStyle.paddingLeft),headerRight:headerBox.right-parseFloat(headerStyle.paddingRight),overflow:document.documentElement.scrollWidth-innerWidth};
 },selector);
 assert(geometry.overflow<=1,`${label}: horizontal overflow ${geometry.overflow}`);
 assert(Math.abs(geometry.left-geometry.headerLeft)<=1,`${label}: left edges differ ${JSON.stringify(geometry)}`);
 assert(Math.abs(geometry.right-geometry.headerRight)<=1,`${label}: right edges differ ${JSON.stringify(geometry)}`);
 return {label,...geometry};
}

/** openSection uses the visible report navigation at each responsive breakpoint. */
async function openSection(page,id){
 const select=page.locator('.kundli-section-select select');
 if(await select.isVisible())await select.selectOption(id);else await page.locator(`#kundli-tab-${id}`).click();
}

/** verifyPublicLayout exercises every report section/category and the shared shells across public routes. */
async function verifyPublicLayout(){
 fs.mkdirSync(output,{recursive:true});
 const browser=await chromium.launch({executablePath:'C:/Program Files/Google/Chrome/Application/chrome.exe',headless:true});
 const results={passed:false,checks:[],errors:[]};
 try{
  const context=await browser.newContext({reducedMotion:'reduce'});
  await context.addInitScript(value=>{localStorage.setItem('m4y-language','hi');localStorage.setItem('m4y-analytics','no');sessionStorage.setItem('m4y-guest-report',JSON.stringify(value));sessionStorage.setItem('m4y-birth',JSON.stringify(value.birth));},report);
  const page=await context.newPage();
  page.on('pageerror',error=>results.errors.push(error.message));
  await page.route('**/api/**',route=>route.request().method()==='GET'||['/api/kundli/dasha','/api/dasha/children'].includes(new URL(route.request().url()).pathname)?route.continue():route.abort());
  for(const language of ['hi','en']){
   for(const width of widths){
    await page.setViewportSize({width,height:1000});
    await page.goto('http://localhost:5173/kundli');
    await page.locator('.kundli-summary').waitFor();
    await page.locator('.language-selector select').selectOption(language);
    for(const section of ['overview','birth','numerology','charts','reading','dasha','planets','checks','transits']){
     await openSection(page,section);
     results.checks.push(await inspectLayout(page,'.kundli-workspace',`${language}/${width}/kundli/${section}`));
     const geometry=await page.locator('.report-next').evaluate(element=>({top:element.getBoundingClientRect().top,previousBottom:element.previousElementSibling.getBoundingClientRect().bottom}));
     assert(geometry.top-geometry.previousBottom>=19,`${section}: action touches result`);
     if(section==='reading'){
      const topics=await page.locator('.faladesh-topic-tabs button').evaluateAll(elements=>elements.map(element=>element.id));
      assert.equal(topics.length,9);
      for(const topicId of topics){
       const topic=topicId.replace('faladesh-topic-','');
       await page.locator(`#${topicId}`).click();
       const reading=await page.locator('.guruji-faladesh').boundingBox(),action=await page.locator('.report-next').boundingBox();
       assert(Math.abs(reading.x-action.x)<1&&Math.abs(reading.width-action.width)<1,`${language}/${width}/${topic}: reading and action widths differ`);
       const prose=await page.locator('.guruji-reading-prose').boundingBox();
       assert(reading.width-prose.width<65,`${topic}: unused space inside reading`);
       assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),`${topic}: overflow`);
       if(topic==='general'&&[390,1920].includes(width))await page.locator('.kundli-workspace').screenshot({path:`${output}/kundli-${language}-${width}.png`,style:'header{visibility:hidden!important}'});
      }
     }
    }
    for(const [route,selector] of [['/','.hero .container'],['/kundli-matching','.matching-main'],['/vivah-muhurat','.vivah-main'],['/bhakti','.devotional-page'],['/puja-vidhi','.devotional-page'],['/shop','.shop-collection'],['/shop/bag','.shop-bag'],['/login','.auth-page']]){
     await page.goto(`http://localhost:5173${route}`);
     if(route==='/login'){
      await page.locator('.auth-layout').waitFor();
      assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),`${language}/${width}/login: overflow`);
     }else results.checks.push(await inspectLayout(page,selector,`${language}/${width}${route}`));
     if(language==='hi'&&[390,1920].includes(width)&&['/kundli-matching','/vivah-muhurat','/bhakti','/shop'].includes(route))await page.screenshot({path:`${output}/${route.slice(1)}-${width}.png`});
    }
   }
  }
  assert.deepEqual(results.errors,[]);
  results.passed=true;
  console.log(JSON.stringify({passed:true,languages:2,widths,reportSections:9,readingCategories:9,publicRoutes:8,checks:results.checks.length}));
 }finally{
  fs.writeFileSync(`${output}/results.json`,JSON.stringify(results,null,2));
  await browser.close();
 }
}
verifyPublicLayout().catch(error=>{console.error(error);process.exitCode=1;});
