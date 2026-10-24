// Isolated development store ONLY. This test never connects to the user's Atlas.
const { chromium } = require('../web/node_modules/playwright');
const AxeBuilder = require('../web/node_modules/@axe-core/playwright').default;
const assert = require('node:assert/strict');
const fs = require('node:fs');
const base='http://127.0.0.1:18087';
const results={passed:false,layouts:[],audits:0,checks:[],errors:[]};
(async()=>{
  const browser=await chromium.launch({executablePath:'C:/Program Files/Google/Chrome/Application/chrome.exe',headless:true});
  try {
    const context=await browser.newContext({viewport:{width:1440,height:1000},reducedMotion:'reduce'});
    await context.addInitScript(()=>{if(!localStorage.getItem('m4y-language'))localStorage.setItem('m4y-language','en');localStorage.setItem('m4y-analytics','no')});
    const page=await context.newPage();page.setDefaultTimeout(20000);page.on('pageerror',e=>results.errors.push(e.message));
    const conf=await (await context.request.get(base+'/api/config')).json();assert.equal(conf.storage,'local development file');assert(conf.simulation&&conf.otp_simulation);
    /** call sends an authenticated request to the isolated shop QA server and requires successful JSON. */
    const call=async(path,method='GET',body,csrf)=>{const res=await context.request.fetch(base+'/api'+path,{method,headers:{Origin:base,...(csrf?{'X-CSRF-Token':csrf}:{})},data:body});assert(res.ok(),path+': '+await res.text());return res.json()};
    await call('/admin/login','POST',{password:'shop-qa-local-only',totp:''});
    const initialAdmin=await call('/admin');
    const existing=await call('/admin/shop');
    for(const p of existing.products.filter(p=>p.status==='published'))await call('/admin/shop/products','POST',{...p,status:'archived'},initialAdmin.csrf);
    await page.goto(base+'/shop');await page.locator('.gem-card').first().waitFor();
    await page.screenshot({path:'.local/shop-preview-1440.png',fullPage:true});
    assert.equal(await page.locator('.gem-card').count(),6);
    assert.equal(await page.getByText('Collection preview',{exact:true}).count(),6);
    // All setup mutations below target the isolated file store after that assertion.
    await call('/admin/login','POST',{password:'shop-qa-local-only',totp:''});
    const admin=await call('/admin');
    const prefix='qa-'+Date.now();
    const productTemplate={id:'',slug:'',name:'',hindi_name:'',kind:'precious',description:'Isolated QA product description. This is not real merchandise.',hindi_description:'',price:123456,weight_milli_carat:4250,stock:3,available:0,origin:'Seller supplied',material:'not-specified',treatment:'Not assessed',shape:'Oval',dimensions:'10 x 8 mm',image:'',certificate_lab:'',certificate_no:'',certificate_url:'',status:'published',version:0,updated:'0001-01-01T00:00:00Z',preview:false};
    const stones=[];
    for(const [i,name] of ['Ruby','Emerald','Yellow sapphire','Blue sapphire'].entries())stones.push(await call('/admin/shop/products','POST',{...productTemplate,name,slug:prefix+'-'+i,stock:i===3?0:3,price:123456+i*100000},admin.csrf));
    await call('/admin/shop/products','POST',{...productTemplate,name:'Hidden draft',slug:prefix+'-draft',status:'draft'},admin.csrf);
    await call('/admin/shop/settings','PUT',{enabled:true,shipping_fee:12500,tax_mode:'exclusive',tax_bps:300,delivery_policy:'QA only: dispatch after payment confirmation. Shipping within India.',return_policy:'QA only: contact the seller for return instructions before returning a stone.'},admin.csrf);
    const otp=await call('/auth/request','POST',{phone:'+919876543210'});
    await call('/auth/verify','POST',{challenge:otp.challenge,code:otp.demo_code});
    const buyer=await call('/me');
    /** inspect checks the current shop view across both languages, responsive widths and optional accessibility audits. */
    const inspect=async(name,{axe=true}={})=>{
      for(const language of ['en','hi']){
        await page.locator('.language-selector select').selectOption(language);
        for(const width of [320,339,390,650,768,1440]){
          await page.bringToFront();await page.setViewportSize({width,height:1000});await page.evaluate(()=>document.fonts.ready);
          await page.evaluate(()=>new Promise(r=>{setTimeout(r,500);requestAnimationFrame(()=>requestAnimationFrame(r))}));
          const overflow=await page.evaluate(()=>({width:innerWidth,scroll:document.documentElement.scrollWidth,clipped:[...document.querySelectorAll('main button,main input,main textarea,main select,.shop-header a,.shop-header button')].filter(n=>{const r=n.getBoundingClientRect();return r.width&&(r.left < -1||r.right>innerWidth+1)}).map(n=>n.outerHTML.slice(0,140))}));
          assert(overflow.scroll<=width+1&&!overflow.clipped.length,JSON.stringify({name,language,width,overflow}));
          if([339,1440].includes(width)){
            await page.screenshot({path:`.local/shop-${name}-${language}-${width}.png`,fullPage:true});
            if(axe && width===339){console.log('Axe '+name+'/'+language);const scan=await new AxeBuilder({page}).setLegacyMode().analyze();assert.deepEqual(scan.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>n.target)})),[],`${name}/${language}/${width}`);results.audits++}
          }
          results.layouts.push({name,language,width});
          if(width===1440)console.log(name+"/"+language+" passed");
        }
      }
      await page.locator('.language-selector select').selectOption('en');
    };
    await page.reload();await page.locator('.gem-card').first().waitFor();assert.equal(await page.locator('.gem-card').count(),4);
    await inspect('catalog');
    await page.getByPlaceholder('Search stones').fill('Ruby');assert.equal(await page.locator('.gem-card').count(),1);await page.getByPlaceholder('Search stones').fill('');
    await page.getByLabel('In stock only',{exact:true}).check();assert.equal(await page.locator('.gem-card').count(),3);await page.getByLabel('In stock only',{exact:true}).uncheck();
    await page.goto(base+'/shop/stones/'+stones[0].slug);await page.locator('.shop-product-copy').waitFor();await inspect('product');
    assert(await page.getByText('No laboratory report has been supplied for this stone.').isVisible());
    await page.getByRole('button',{name:'Add to bag',exact:true}).click();await page.goto(base+'/shop/bag');await page.locator('.shop-delivery-form').waitFor();await inspect('bag');
    for(const [label,value] of [['Recipient name','QA Customer'],['Delivery mobile number','9876543210'],['Address line 1','12 Test Street'],['City','Delhi'],['State','Delhi'],['PIN code','110001']])await page.getByLabel(label,{exact:true}).fill(value);
    await page.getByLabel('I have reviewed the delivery and return policies below.').check();await page.getByRole('button',{name:'Review order & payment',exact:true}).click();await page.locator('.shop-checkout-review').waitFor();
    await inspect('payment');await page.getByRole('button',{name:/Simulate successful payment|डेमो भुगतान सफल/}).click();
    await page.waitForURL('**/shop/orders');await page.locator('.shop-order').first().waitFor();await inspect('orders');
    const me=await call('/me');const paid=me.orders.find(o=>o.product==='gemstone'&&o.status==='paid');assert(paid);assert.equal(paid.amount,140035);assert.equal(paid.gemstone.fulfilment,'processing');
    results.checks.push('Real API catalogue, draft visibility, filters, cart, address, server totals, explicit simulated payment and order history');
    await page.goto(base+'/admin');await page.locator('.admin-tabs').waitFor();await page.getByRole('button',{name:'Gemstone shop',exact:true}).click();await page.locator('.shop-admin-form').waitFor();await inspect('admin-products');
    await page.getByLabel('Name (English)',{exact:true}).fill('QA draft from admin');
    await page.getByLabel('Product photograph',{exact:true}).setInputFiles({name:'qa-stone.png',mimeType:'image/png',buffer:Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+yK1sAAAAASUVORK5CYII=','base64')});
    await page.getByText('Photo uploaded. Save the product to apply it.',{exact:true}).waitFor();
    await page.getByRole('button',{name:'Save stone',exact:true}).click();
    await page.getByText('Saved successfully',{exact:true}).waitFor();
    const savedDraft=(await call('/admin/shop')).products.find(p=>p.name==='QA draft from admin');assert(savedDraft&&savedDraft.status==='draft'&&savedDraft.image);
    assert.equal((await call('/shop')).products.length,4);
    results.checks.push('Admin creates a draft through the actual form and uploads a local photograph; draft remains outside the public catalogue');

    await page.locator('.shop-admin-tabs').getByRole('button',{name:'Stone orders',exact:true}).click();await page.locator('.shop-order').first().waitFor();await inspect('admin-orders');
    const shipment=page.locator('form[data-shop-order="'+paid.id+'"]');await shipment.getByLabel('Courier',{exact:true}).fill('QA Courier');await shipment.getByLabel('Tracking number',{exact:true}).fill('QA12345');await shipment.getByRole('button',{name:'Mark as dispatched',exact:true}).click();await page.getByRole('button',{name:'Mark as delivered',exact:true}).waitFor();
    await page.locator('.shop-admin-tabs').getByRole('button',{name:'Shop settings',exact:true}).click();await page.getByRole('button',{name:'Save shop settings'}).waitFor();await inspect('admin-settings');
    results.checks.push('Admin product form, order dispatch with tracking, shop policies and Hindi layouts');
    assert.deepEqual(results.errors,[]);results.passed=true;
    console.log(JSON.stringify({passed:true,layouts:results.layouts.length,audits:results.audits,checks:results.checks}));
  }finally{fs.writeFileSync('.local/shop-browser-results.json',JSON.stringify(results,null,2));await browser.close()}
})().catch(e=>{console.error(e);process.exitCode=1});
