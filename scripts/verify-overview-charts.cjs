const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const fs = require("node:fs");
const assert = require("node:assert/strict");

/** verify checks simultaneous chart layouts, original natal annotations, house assignments, zoom accessibility and crowded/legacy charts. */
async function verify() {
  const fixture = JSON.parse(fs.readFileSync("tmp/source-only-qa/report.json", "utf8"));
  const files = [["scripts/fixtures/source-only-qa.html","web/.source-only-qa.html"],["scripts/fixtures/source-only-qa.tsx","web/src/.source-only-qa.tsx"]];
  const created = [];
  let browser;
  const results = [];
  try {
    for (const [source,target] of files) {fs.copyFileSync(source,target,fs.constants.COPYFILE_EXCL);created.push(target);}
    browser = await chromium.launch({executablePath:"C:/Program Files/Google/Chrome/Application/chrome.exe",headless:true});
    const context = await browser.newContext({reducedMotion:"reduce"});
    const page = await context.newPage();
    const errors = [];
    page.on("pageerror",error=>errors.push(error.message));
    await page.route("**/api/**",route=>route.abort());
    await page.goto(process.env.SOURCE_QA_URL || "http://localhost:5173/.source-only-qa.html");
    await page.locator(".overview-chart-grid").waitFor();
    for (const language of ["hi","en"]) {
      await page.locator(`#qa-${language}`).click();
      for (const width of [320,390,768,1024,1440]) {
        await page.setViewportSize({width,height:1100});
        await page.evaluate(()=>document.fonts.ready);
        await page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
        await page.waitForFunction(()=>[...document.querySelectorAll(".overview-chart-grid .chart-annotated")].every(chart=>chart.dataset.labelsFitted==="true"));
        assert.equal(await page.locator(".overview-chart-card").count(),3);
        assert.equal(await page.locator(".overview-charts .lagna-graha-roles").count(),1);
        const boxes = await page.locator(".overview-chart-card").evaluateAll(cards=>cards.map(card=>{const rect=card.getBoundingClientRect();return{x:rect.x,y:rect.y,width:rect.width,height:rect.height};}));
        if (width>=768) assert.ok(boxes.every(box=>Math.abs(box.y-boxes[0].y)<1), "three charts should share a desktop row");
        else assert.ok(boxes[1].y>=boxes[0].y+boxes[0].height && boxes[2].y>=boxes[1].y+boxes[1].height, "phone charts should stack without tabs");
        assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+2),false,`page overflow ${width}/${language}`);
        for (const mode of ["d1","moon","d9"]) {
          const chart = page.locator(`[data-chart="${mode}"]`);
          assert.equal(await chart.locator(".chart-detail-position").count(),9);
          assert.equal(await chart.locator("text[textLength]").count(),0,"Glyphs must keep their natural proportions");
          assert.equal(await chart.locator(".chart-detail-nakshatra").count(),9);
          for (const planet of fixture.chart.planets) {
            const position = chart.locator(`.chart-detail-position[data-planet="${planet.name}"]`);
            const text = await position.evaluate(element=>[...element.childNodes].filter(node=>node.nodeName!=="title").map(node=>node.textContent).join(""));
            const match = text.match(/(\d+)°(\d+)′/);
            assert.ok(match,`${mode}/${planet.name}: missing natal degree`);
            const shown = Number(match[1])+Number(match[2])/60;
            assert.ok(planet.degree_in_rashi-shown>=-1e-7 && planet.degree_in_rashi-shown<1/60+1e-7, "annotations must retain natal degrees");
            const expectedHouse = mode==="d1" ? planet.house : mode==="moon" ? (planet.rashi_number-fixture.chart.planets.find(p=>p.name==="Moon").rashi_number+12)%12+1 : fixture.navamsa.planets.find(p=>p.name===planet.name).house;
            assert.equal(await position.locator("..").getAttribute("data-house"),String(expectedHouse));
            if(mode==="d9") assert.doesNotMatch(text,/[↑↓]|स्व/);
          }
        }
        const escapedLabels = await page.locator(".overview-chart-grid .chart-annotated").evaluateAll(charts=>charts.flatMap(chart=>Array.from(chart.querySelectorAll("text")).filter(label=>{
          const bounds=label.getBBox();
          const house=Number(label.parentElement.dataset.house)-1;
          const polygon=chart.querySelectorAll("polygon")[house];
          return [[bounds.x,bounds.y],[bounds.x+bounds.width,bounds.y],[bounds.x,bounds.y+bounds.height],[bounds.x+bounds.width,bounds.y+bounds.height]].some(([x,y])=>!polygon.isPointInFill(new DOMPoint(x,y)));
        }).map(label=>{const b=label.getBBox();return {text:label.textContent,house:label.parentElement.dataset.house,font:label.getAttribute("font-size"),base:label.dataset.baseFont,x:label.getAttribute("x"),y:label.getAttribute("y"),box:{x:b.x,y:b.y,width:b.width,height:b.height}};})));
        if(escapedLabels.length)await page.locator(".overview-charts").screenshot({path:".local/chart-fit-failure.png"});
        assert.deepEqual(escapedLabels,[],`labels outside house ${width}/${language}`);
        const overlappingRows = await page.locator('.overview-chart-grid [data-house]').evaluateAll(houses=>houses.flatMap(house=>{
          const rows=[...house.querySelectorAll('text')].map(text=>text.getBBox());
          return rows.slice(1).flatMap((box,index)=>box.y<rows[index].y+rows[index].height-.05?[house.dataset.house]:[]);
        }));
        assert.deepEqual(overlappingRows,[],`overlapping chart rows ${width}/${language}`);
        results.push({language,width,charts:3,originalNatalAngles:true,houseAssignments:true,noOverflow:true});
      }
    }
    assert.equal(await page.locator(".overview-charts select").count(),0);
    assert.equal(await page.locator(".overview-charts .is-selected").count(),0);
    await page.locator('[data-chart="d9"] .chart-expand').click();
    assert.equal(await page.locator("dialog").evaluate(element=>element.open),true);
    assert.equal(await page.locator(".chart-zoom-table tbody tr").count(),9);
    await page.waitForFunction(()=>document.querySelector("dialog .chart-annotated")?.dataset.labelsFitted==="true");
    await page.keyboard.press("Escape");
    assert.equal(await page.locator("dialog").evaluate(element=>element.open),false);
    assert.equal(await page.locator('[data-chart="d9"] .chart-expand').evaluate(element=>element===document.activeElement),true);
    const scan = await new AxeBuilder({page}).include(".overview-charts").analyze();
    assert.deepEqual(scan.violations.filter(v=>v.impact==="critical"||v.impact==="serious"),[]);
    await page.locator("#qa-hi").click();
    await page.waitForFunction(()=>[...document.querySelectorAll(".overview-chart-grid .chart-annotated")].every(chart=>chart.dataset.labelsFitted==="true"));
    await page.locator(".overview-charts").screenshot({path:"tmp/source-only-qa/three-charts-desktop.png"});
    await page.setViewportSize({width:390,height:1000});
    await page.locator(".overview-charts").screenshot({path:"tmp/source-only-qa/three-charts-mobile.png"});
    await page.locator("#qa-crowded").click();
    assert.equal(await page.locator('[data-chart="d1"] [data-house="2"] .chart-detail-position').count(),9);
    assert.equal(await page.locator('[data-chart="d1"] [data-house="2"] .chart-detail-nakshatra').count(),9);
    await page.waitForFunction(()=>document.querySelector('[data-chart="d1"] .chart-annotated')?.dataset.labelsFitted==="true");
    assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+2),false);
    await page.locator('[data-chart="d1"] .chart-expand').click();
    assert.equal(await page.locator(".chart-zoom-table tbody tr").count(),9);
    await page.keyboard.press("Escape");
    await page.locator("#qa-old-chart").click();
    assert.equal(await page.locator('[data-chart="d9"] .kundli-chart').count(),0);
    assert.equal(await page.locator('[data-chart="d9"] .overview-chart-missing').count(),1);
    await page.locator("#qa-lagna").selectOption("0");
    assert.equal(await page.locator(".overview-charts").count(),0);
    assert.deepEqual(errors,[]);
    fs.writeFileSync("tmp/source-only-qa/three-charts-validation.json",JSON.stringify({results,zoomKeyboard:true,crowdedPlanetsRetained:true,missingD9NoFabrication:true,unknownTimeNoChart:true,seriousAccessibilityViolations:0,pageErrors:errors},null,2));
    console.log(JSON.stringify({views:results.length,zoomKeyboard:true,crowdedPlanetsRetained:true,missingD9NoFabrication:true,seriousAccessibilityViolations:0}));
  } finally {
    if(browser)await browser.close();
    for(const file of created.reverse())fs.unlinkSync(file);
  }
}
verify().catch(error=>{console.error(error);process.exitCode=1;});
