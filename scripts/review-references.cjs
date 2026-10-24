const { chromium } = require("../web/node_modules/playwright");
(async () => {
  const browser = await chromium.launch({
    executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe",
    headless: true,
  });
  const page = await browser.newPage({
    viewport: { width: 1440, height: 1000 },
  });
  for (const [name, url] of [
    [
      "drik-kundali",
      "https://www.drikpanchang.com/jyotisha/kundali/kundali.html",
    ],
    ["drik-daily", "https://www.drikpanchang.com/panchang/day-panchang.html"],
    ["astrosage", "https://www.astrosage.com/freechart/"],
    ["astrotalk", "https://astrotalk.com/freekundli"],
  ]) {
    try {
      await page.goto(url, { waitUntil: "domcontentloaded", timeout: 30000 });
      await page.screenshot({ path: ".local/reference-" + name + ".png" });
      console.log(name, await page.title());
    } catch (e) {
      console.log(name, e.message);
    }
  }
  await browser.close();
})();
