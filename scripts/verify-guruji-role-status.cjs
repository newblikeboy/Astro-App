// Synthetic display fixtures only; Go tests independently verify the classifications. No customer data or provider writes.
const {chromium}=require('../web/node_modules/playwright');
const fs=require('node:fs');
const path=require('node:path');
const assert=require('node:assert/strict');

/** verifyRoleStatus checks that saved role/sign/house metadata stays intact while the public reading omits technical reference tables. */
async function verifyRoleStatus(){
  const directory=path.resolve(process.env.GURUJI_QA_DIR||'tmp/guruji/qa');
  const base=JSON.parse(fs.readFileSync(path.join(directory,'report.json'),'utf8'));
  const browser=await chromium.launch({executablePath:'C:/Program Files/Google/Chrome/Application/chrome.exe',headless:true});
  const results=[];
  try{
    for(const sample of [
      {lagna:3,sign:6,house:4,role:'karak',primary:'exalted',own:true},
      {lagna:7,sign:6,house:12,role:'marak',primary:'exalted',own:true},
      {lagna:3,sign:12,house:10,role:'karak',primary:'debilitated',own:false},
    ]){
      // Override only the rendered fixture facts; these examples are not astronomical birth-time calculations.
      const report=structuredClone(base);
      report.guruji_faladesh.lagna=sample.lagna;
      const mercury=report.guruji_faladesh.readings.find(item=>item.planet==='Mercury');
      Object.assign(mercury,{role:sample.role,sign_number:sample.sign,house:sample.house,sign_status:{primary:sample.primary,exalted:sample.primary==='exalted',debilitated:sample.primary==='debilitated',own_sign:sample.own}});
      const context=await browser.newContext({viewport:{width:390,height:1000}});
      await context.addInitScript(value=>{localStorage.setItem('m4y-language','en');localStorage.setItem('m4y-analytics','no');sessionStorage.setItem('m4y-guest-report',JSON.stringify(value));},report);
      const page=await context.newPage();
      await page.route('**/api/**',route=>route.request().method()==='GET'?route.continue():route.abort());
      await page.goto(process.env.GURUJI_QA_URL||'http://localhost:5173/kundli');
      await page.locator('.kundli-summary').waitFor();
      await page.locator('.kundli-section-select select').selectOption('reading');
      for(const language of ['en','hi']){
        await page.locator('.language-selector select').selectOption(language);
        for(const width of [320,390,1440]){
          await page.setViewportSize({width,height:1000});
          assert.equal(await page.locator('.guruji-source-reference,.guruji-planet-facts').count(),0,'technical facts exposed in reader view');
          assert((await page.locator('.guruji-unified-reading').innerText()).length>100,'interpretation missing');
          const saved=await page.evaluate(()=>{
            const mercury=JSON.parse(sessionStorage.getItem('m4y-guest-report')).guruji_faladesh.readings.find(r=>r.planet==='Mercury');
            return {role:mercury.role,sign:mercury.sign_number,house:mercury.house,primary:mercury.sign_status.primary,own:mercury.sign_status.own_sign};
          });
          assert.deepEqual(saved,{role:sample.role,sign:sample.sign,house:sample.house,primary:sample.primary,own:sample.own},'presentation changed saved classification');
          assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),'reading overflow');
        }
      }
      results.push(sample);
      await context.close();
    }
    fs.writeFileSync(path.join(directory,'role-sign-browser-results.json'),JSON.stringify({passed:true,examples:results},null,2));
    console.log(JSON.stringify({passed:true,examples:results.length,languages:2,widths:[320,390,1440]}));
  }finally{await browser.close();}
}

verifyRoleStatus().catch(error=>{console.error(error);process.exitCode=1;});
