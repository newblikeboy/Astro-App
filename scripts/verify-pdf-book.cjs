// Inspect synthetic self-contained renderer fixtures only; no application, account or customer-record access.
const { chromium } = require("../web/node_modules/playwright");
const fs = require("node:fs");
const path = require("node:path");
const { pathToFileURL } = require("node:url");
const assert = require("node:assert/strict");
const root = path.resolve(process.env.PDF_BOOK_QA_DIR || ".local/pdf-book-qa");

/** verify checks complete chapter navigation, print-width layout and long-name cover behaviour in both languages. */
async function verify() {
  const browser = await chromium.launch({ executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe", headless: true });
  const results = { passed: false, documents: [], errors: [] };
  try {
    const page = await browser.newPage({ viewport: { width: 643, height: 1000 } });
    page.on("pageerror", (error) => results.errors.push(error.message));
    await page.emulateMedia({ media: "print" });
    for (const file of ["mishra4you-kundli-preview-hi.html", "remedy-language-en.html", "remedy-language-hi.html"]) {
      await page.goto(pathToFileURL(path.join(root, file)).href);
      await page.evaluate(() => document.fonts.ready);
      const review = await page.evaluate(() => {
        const ids = [...document.querySelectorAll("[id]")].map((node) => node.id);
        const links = [...document.querySelectorAll('a[href^="#"]')].map((node) => node.getAttribute("href").slice(1));
        const overflow = [...document.querySelectorAll(".cover,.chapter-heading,.fact-card,.reading-card,.chart-block,table")].filter((node) => {
          const box = node.getBoundingClientRect();
          return box.left < -1 || box.right > innerWidth + 1 || node.scrollWidth > node.clientWidth + 2;
        }).map((node) => ({ tag: node.tagName, class: node.className }));
        return { duplicateIDs: ids.filter((id, i) => ids.indexOf(id) !== i), brokenLinks: links.filter((id) => !document.getElementById(id)), overflow, coverHeight: document.querySelector(".cover").getBoundingClientRect().height, chapters: document.querySelectorAll(".reading-chapter").length, topics: document.querySelectorAll(".topic-group").length };
      });
      assert.deepEqual(review.duplicateIDs, [], file + ": duplicate chapter IDs");
      assert.deepEqual(review.brokenLinks, [], file + ": broken contents link");
      assert.deepEqual(review.overflow, [], file + ": print-width overflow");
      assert(review.coverHeight <= 952.5, file + ": cover spills to another page");
      assert.equal(review.chapters, 5);
      assert(!(await page.locator("body").innerText()).includes("book."), file + ": untranslated copy key");
      await page.locator(".cover-name").evaluate((node) => {
        node.textContent = "LongName".repeat(12) + "Test";
        node.classList.add("cover-name-long");
      });
      const longCover = await page.locator(".cover").evaluate((node) => ({ height: node.getBoundingClientRect().height, width: node.scrollWidth }));
      assert(longCover.height <= 952.5 && longCover.width <= 644, file + ": long name breaks cover");
      results.documents.push({ file, ...review, longNameCoverHeight: longCover.height });
    }
    assert.deepEqual(results.errors, []);
    results.passed = true;
    fs.writeFileSync(path.join(root, "layout-results.json"), JSON.stringify(results, null, 2));
    console.log(JSON.stringify(results));
  } finally { await browser.close(); }
}

verify().catch((error) => { console.error(error); process.exitCode = 1; });
