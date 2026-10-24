const { chromium } = require("../web/node_modules/playwright");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

/** exportBrand refreshes transparent 3x logo PNGs and the 180px touch icon from local SVG masters without network access. */
async function exportBrand() {
  const directory = path.resolve(__dirname, "../web/public/images/brand");
  const browser = await chromium.launch({
    executablePath:
      process.env.CHROME_PATH ||
      "C:/Program Files/Google/Chrome/Application/chrome.exe",
    headless: true,
  });
  try {
    const page = await browser.newPage({
      viewport: { width: 352, height: 96 },
      deviceScaleFactor: 3,
    });
    for (const name of ["logo", "logo-reverse"]) {
      await page.goto(pathToFileURL(path.join(directory, name + ".svg")).href);
      await page.screenshot({
        path: path.join(directory, name + ".png"),
        omitBackground: true,
      });
    }
    const icon = await browser.newPage({
      viewport: { width: 180, height: 180 },
      deviceScaleFactor: 1,
    });
    await icon.goto(pathToFileURL(path.join(directory, "favicon.svg")).href);
    await icon.screenshot({
      path: path.join(directory, "apple-touch-icon.png"),
      omitBackground: true,
    });
    console.log(
      "Exported light/dark 1056 × 288 logos and the 180 × 180 touch icon.",
    );
  } finally {
    await browser.close();
  }
}
exportBrand().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
