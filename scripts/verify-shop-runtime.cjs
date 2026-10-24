// Main-site smoke test: GETs only; never changes Atlas or sends a payment.
const {chromium}=require('../web/node_modules/playwright');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const baseURL=process.env.SHOP_URL||'http://localhost:5173';
const target=new URL(baseURL);
if(!['localhost','127.0.0.1'].includes(target.hostname)||target.protocol!=='http:')throw new Error('Shop runtime checks require a localhost URL');
(async()=>{
  const browser=await chromium.launch({executablePath:'C:/Program Files/Google/Chrome/Application/chrome.exe',headless:true,args:['--disable-gpu','--renderer-process-limit=1']});
  const result={passed:false,layouts:[],errors:[]};
  try{
    const context=await browser.newContext({viewport:{width:1440,height:950},reducedMotion:'reduce'});
    await context.addInitScript(()=>{localStorage.setItem('m4y-analytics','no');if(!localStorage.getItem('m4y-language'))localStorage.setItem('m4y-language','en')});
    const page=await context.newPage();page.setDefaultTimeout(20000);page.on('pageerror',e=>result.errors.push(e.message));
    await page.route('**/api/**',route=>{assert.equal(route.request().method(),'GET','Unexpected main-site mutation');return route.continue()});
    for(const path of ['/','/shop','/shop/bag','/shop/orders']){
      await page.goto(baseURL+path);await page.locator('.language-selector select').waitFor();
      if(path==='/shop')await page.locator('.gem-card').first().waitFor();
      for(const language of ['en','hi']){
        await page.locator('.language-selector select').selectOption(language);
        for(const width of [320,339,768,1024,1440]){
          await page.setViewportSize({width,height:950});await page.evaluate(()=>document.fonts.ready);
          assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),`${path}/${language}/${width} overflow`);
          result.layouts.push({path,language,width});
        }
      }
    }
    await page.goto(baseURL+'/shop');await page.locator('.gem-card').first().waitFor();
    assert.equal(await page.locator('.gem-card').count(),6);
    await page.locator('.language-selector select').selectOption('en');
    await page.setViewportSize({width:339,height:950});await page.locator('.shop-menu').click();assert(await page.locator('.shop-header nav').isVisible());
    await page.locator('.shop-header nav a[href="/shop/orders"]').click();await page.waitForURL('**/shop/orders');
    const next=await page.locator('.shop-empty a').getAttribute('href');assert.equal(new URL(next,baseURL).searchParams.get('next'),'/shop/orders');
    for(const width of [1440,339]){
      await page.goto(baseURL);await page.setViewportSize({width,height:950});await page.locator('.language-selector select').selectOption('en');
      if(width<1201)await page.locator('.menu-toggle').click();
      await page.locator('#primary-navigation').getByRole('button',{name:'Gemstones',exact:true}).click();await page.waitForURL('**/#gemstones');
      await page.locator('#gemstones .home-stone').first().waitFor();assert.equal(await page.locator('#gemstones .home-stone').count(),6);
      if(width<1201)assert.equal(await page.locator('#primary-navigation.nav-open').count(),0);
      assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),`Home collection/${width} overflow`);
      await page.locator('#gemstones .home-stone').first().click();await page.locator('.shop-product-copy').waitFor();assert.equal(await page.getByRole('button',{name:'Add to bag',exact:true}).count(),0);
    }
    assert.equal(await page.locator('a[href^="/admin"]').count(),0);
    assert.deepEqual(result.errors,[]);result.passed=true;console.log(JSON.stringify({passed:true,layouts:result.layouts.length,checks:'Embedded landing collection, desktop/mobile section navigation, product links, protected order return, unpurchasable previews and no API writes'}));
  }finally{fs.writeFileSync('.local/shop-runtime-results.json',JSON.stringify(result,null,2));await browser.close()}
})().catch(e=>{console.error(e);process.exitCode=1});
