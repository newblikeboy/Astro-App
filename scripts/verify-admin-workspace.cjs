// Read-only against the running app: protected responses and every mutation are fulfilled in the browser with fictional QA records.
const { chromium } = require('../web/node_modules/playwright');
const AxeBuilder = require('../web/node_modules/@axe-core/playwright').default;
const assert = require('node:assert/strict');
const fs = require('node:fs');
const base = process.env.ADMIN_QA_URL || 'http://localhost:5173';
const evidence = {passed: false, layouts: [], audits: 0, checks: [], errors: []};

/** verify checks the administrator entry, task navigation, truthful summaries and responsive tools without touching business records. */
async function verify() {
  const config = await (await fetch(base + '/api/config')).json();
  const product = {id:'qa_ruby', slug:'qa-ruby', name:'QA Ruby', hindi_name:'परीक्षण माणिक', kind:'precious', description:'Fictional QA stone for browser verification.', hindi_description:'परीक्षण विवरण', price:125000, weight_milli_carat:2500, stock:1, available:1, origin:'', material:'natural', treatment:'Disclosed for QA', shape:'Oval', dimensions:'', image:'/images/gemstones/ruby.webp', certificate_lab:'', certificate_no:'', certificate_url:'', status:'published', version:1, updated:'2026-10-20T08:00:00Z', sample:false, preview:false};
  const settings = {...config.settings, availability: config.settings.availability || [], blocked:config.settings.blocked || []};
  const stock = {products:[product, {...product, id:'qa_emerald', name:'QA Emerald', slug:'qa-emerald', sample:true, available:0}], settings:{enabled:false, shipping_fee:0, tax_mode:'unconfigured', tax_bps:0, delivery_policy:'QA delivery policy', return_policy:'QA return policy'}, orders:[]};
  const order = {id:'qa_order_1', product:'gemstone', title:'QA Ruby', amount:125000, status:'paid', provider_id:'qa_payment_1', report_id:'', booking_id:'', created:'2026-10-20T08:00:00Z', demo:false, tax_mode:'none', refunded:0, gemstone:{items:[{product_id:product.id, quantity:1, product}], address:{name:'QA Customer',phone:'+919800000000',line1:'Fictional QA address',line2:'',city:'Delhi',state:'Delhi',pin:'110001',country:'IN'},subtotal:125000,shipping:0,delivery_policy:'QA only',return_policy:'QA only',hold_until:'2026-10-20T09:00:00Z',fulfilment:'processing',inventory_taken:true,restocked:false,courier:'',tracking_number:'',tracking_url:'',shipment_version:0}};
  stock.orders = [order];
  const data = {csrf:'browser-only-qa',settings,rules:{},users:[{id:'qa_customer_1',phone:'••••••1234',created:'2026-10-20T08:00:00Z',marketing:false,deleted:false},{id:'qa_deleted',phone:'deleted',created:'2026-10-20T08:00:00Z',marketing:false,deleted:true}],profiles:[],orders:[order,{...order,id:'qa_test',demo:true},{...order,id:'qa_pending',status:'pending',gemstone:undefined}],bookings:{qa_booking:{id:'qa_booking',start:'2090-01-01T08:00:00Z',end:'2090-01-01T08:45:00Z',status:'confirmed',demo:false,order_id:'qa_order_booking'}},jobs:{qa_job:{id:'qa_job',kind:'pdf',status:'retry',attempts:2,error:'Fictional transient QA error',ref:'qa_order_1',next:'2026-10-20T09:00:00Z'}},expenses:{},distributions:{},month:null,finance:{month:'2026-10',gross:125000,pdf_sales:0,gemstone_sales:125000,consultation_sales:0,refunds:0,fees:0,tax:0,expenses:0,reserves:0,distributable:125000,partner_a:62500,partner_b:62500,distributed:0,undistributed:125000,tax_configured:false,unreconciled:true,rounding_remainder:0},audit:[],analytics:[],pdf_blockers:[],booking_blockers:[]};
  const browser = await chromium.launch({executablePath:'C:/Program Files/Google/Chrome/Application/chrome.exe',headless:true});
  try {
    const context = await browser.newContext({viewport:{width:1440,height:1000},reducedMotion:'reduce'});
    await context.addInitScript(() => {localStorage.setItem('m4y-language','en');localStorage.setItem('m4y-analytics','no');});
    const page = await context.newPage();
    page.on('pageerror', error => evidence.errors.push(error.message));
    let signedIn = false, failStock = false, capturedProduct;
    await page.route('**/api/**', async route => {
      const request = route.request(), url = new URL(request.url());
      /** json fulfils a browser-only fixture response without contacting the business API. */
      const json = value => route.fulfill({json:value});
      if (url.pathname === '/api/admin/login') {assert.equal(request.method(),'POST');signedIn=true;return json({ok:true});}
      if (url.pathname === '/api/logout') {signedIn=false;return json({ok:true});}
      if (url.pathname === '/api/admin/shop/products' && request.method() === 'POST') {capturedProduct=request.postDataJSON();return json(capturedProduct);}
      assert.equal(request.method(),'GET','No business mutation may reach the real app');
      if (url.pathname === '/api/admin') return signedIn ? json(data) : route.fulfill({status:401,json:{error:'Sign in required'}});
      if (url.pathname === '/api/admin/shop') return failStock ? route.fulfill({status:503,json:{error:'Fictional stock outage'}}) : json(stock);
      if (url.pathname === '/api/config') return json({...config, development:true, simulation:true, settings});
      return route.continue();
    });
    /** inspect checks wrapping and accessibility at phone/tablet/desktop widths and saves representative screenshots. */
    async function inspect(name, audits = true) {
      if (process.env.ADMIN_QA_VIEWS && !process.env.ADMIN_QA_VIEWS.split(',').includes(name)) return;
      for (const language of ['en','hi']) {
        await page.locator('.language-selector select').selectOption(language);
        for (const width of [320,390,650,768,901,1100,1440]) {
          await page.setViewportSize({width,height:1000});
          await page.evaluate(async () => {await document.fonts.ready;await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));});
          const state = await page.evaluate(() => ({overflow:document.documentElement.scrollWidth-innerWidth,h1:document.querySelectorAll('h1').length,clipped:[...document.querySelectorAll('input,select,button')].filter(node => {const rect=node.getBoundingClientRect();return rect.width && !node.closest('.table-scroll') && (rect.left < -1 || rect.right > innerWidth+1);}).map(node=>node.outerHTML.slice(0,120))}));
          if (state.overflow > 1 || state.clipped.length) {
            await page.screenshot({path:'.local/admin-layout-failure.png',fullPage:true});
            state.escaped = await page.evaluate(() => [...document.querySelectorAll('.admin-main *')].filter(node => !node.closest('.table-scroll') && node.getBoundingClientRect().right > innerWidth + 1).map(node => ({tag:node.tagName,class:node.className,text:node.textContent.slice(0,80)})).slice(0,15));
          }
          assert(state.overflow <= 1 && state.h1 === 1 && !state.clipped.length,JSON.stringify({name,language,width,state}));
          if (audits && [320,1440].includes(width)) {
            const scan=await new AxeBuilder({page}).analyze();
            assert.deepEqual(scan.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>n.target)})),[],JSON.stringify({name,language,width}));
            evidence.audits++;
            await page.screenshot({path:`.local/admin-${name}-${language}-${width}.png`,fullPage:true});
          }
          evidence.layouts.push({name,language,width});
        }
      }
      await page.locator('.language-selector select').selectOption('en');
      await page.setViewportSize({width:1440,height:1000});
    }
    /** open selects a management section using the desktop navigation. */
    async function open(id) {
      const ids = ['overview','shop','orders','calendar','users','finance','content','rules','jobs','settings','audit'];
      await page.locator('.admin-tabs button').nth(ids.indexOf(id)).click();
    }
    await page.goto(base+'/admin');await page.locator('#admin-password').waitFor();
    assert.equal(new URL(page.url()).pathname,'/admin/login');
    await inspect('login');
    await page.locator('#admin-password').fill('fictional-qa-password');
    await page.getByRole('button',{name:'Show password',exact:true}).click();assert.equal(await page.locator('#admin-password').getAttribute('type'),'text');
    await page.getByRole('button',{name:'Hide password',exact:true}).click();assert.equal(await page.locator('#admin-password').getAttribute('type'),'password');
    await page.locator('#admin-totp').fill('123');await page.getByRole('button',{name:'Open admin',exact:true}).click();assert.equal(signedIn,false);
    await page.locator('#admin-totp').fill('');await page.getByRole('button',{name:'Open admin',exact:true}).click();await page.locator('.business-metrics').waitFor();
    await page.locator('.attention-row').filter({hasText:'Low-stock products'}).waitFor();
    assert.equal(await page.locator('.business-metric strong').allTextContents().then(values=>values.slice(1).join(',')),'1,1,1');
    assert.match(await page.locator('.business-metric').first().innerText(),/1,250/);
    assert.equal(await page.locator('.admin-attention .attention-count').allTextContents().then(values=>values.join(',')),'1,1,1,1');
    await inspect('overview');
    await page.getByRole('button',{name:/Orders to dispatch/}).click();await page.locator('.shop-order').waitFor();assert.equal(await page.locator('.shop-admin-tabs [aria-pressed=true]').innerText(),'Stone orders');
    await inspect('shipments');
    await open('shop');await page.getByLabel('Find a product',{exact:true}).fill('emerald');assert.equal(await page.locator('.shop-product-list button').count(),2);
    await page.getByLabel('Find a product',{exact:true}).fill('absent');await page.getByText('No products match this search.',{exact:true}).waitFor();
    await page.getByLabel('Find a product',{exact:true}).fill('ruby');await page.locator('.shop-product-list button').filter({hasText:'QA Ruby'}).click();
    await page.getByLabel('Price per piece (INR)',{exact:true}).fill('1234.56');await page.getByRole('button',{name:'Save stone',exact:true}).click();await page.getByText('Saved successfully',{exact:true}).waitFor();assert.equal(capturedProduct.price,123456);assert.equal(capturedProduct.sample,false);
    await inspect('products');
    await open('orders');await page.getByLabel('Find an order',{exact:true}).fill('qa_pending');assert.equal(await page.locator('tbody tr').count(),1);
    await page.getByLabel('Payment status',{exact:true}).selectOption('paid');await page.getByText('No orders match these filters.',{exact:true}).waitFor();await page.getByLabel('Find an order',{exact:true}).fill('');
    await inspect('payments');
    await open('users');await page.getByLabel('Find a customer',{exact:true}).fill('1234');assert.equal(await page.locator('tbody tr').count(),1);await inspect('customers');
    for (const id of ['calendar','jobs','rules','content','settings','finance','audit']) {await open(id);await inspect(id,['calendar','settings','finance'].includes(id));}
    await open('overview');failStock=true;await page.getByRole('button',{name:'Refresh',exact:true}).click();await page.getByText('Stock information could not be loaded.',{exact:true}).waitFor();failStock=false;await page.locator('.admin-stock-error button').click();await page.getByRole('button',{name:/Low-stock products/}).waitFor();
    data.orders=[];data.bookings={};data.users=[];data.jobs={};stock.products=[];stock.orders=[];
    await page.getByRole('button',{name:'Refresh',exact:true}).click();await page.getByText('No live orders yet. New orders will appear here.',{exact:true}).waitFor();await inspect('empty-overview',false);
    await page.getByRole('button',{name:'Log out',exact:true}).click();await page.locator('#admin-password').waitFor();assert.equal(await page.locator('#admin-password').inputValue(),'');assert.equal(await page.locator('.business-metrics').count(),0);
    assert.deepEqual(evidence.errors,[]);evidence.passed=true;
    evidence.checks=['Separate login, password visibility, six-digit validation, browser-only login/logout; no real mutations','Correct server money and live order/booking/account counts; sample products excluded from low-stock count','Dispatch shortcut selects Stone orders; product/order/customer search and payment-status filters','Exact integer-paise product edit fulfilled only in browser; all management sections reachable','Stock error/retry, truthful empty states, cleared credentials and protected content on logout'];
    console.log(JSON.stringify({passed:true,layouts:evidence.layouts.length,audits:evidence.audits,checks:evidence.checks}));
  } finally {fs.writeFileSync('.local/admin-workspace-verification.json',JSON.stringify(evidence,null,2));await browser.close();}
}
verify().catch(error=>{console.error(error);process.exitCode=1;});
