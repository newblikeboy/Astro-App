// Read-only local verification. API writes are blocked; no account or payment state is changed.
const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

/** verifyDevotionalLibrary checks catalogue integrity, public navigation and responsive text-only reading. */
async function verifyDevotionalLibrary() {
  const base = process.env.DESIGN_QA_URL || "http://localhost:5173";
  assert(["localhost", "127.0.0.1"].includes(new URL(base).hostname));
  const metadata = JSON.parse(fs.readFileSync(path.join(__dirname, "../web/src/data/devotional-index.json"), "utf8"));
  const art = JSON.parse(fs.readFileSync(path.join(__dirname, "../web/src/data/devotional-art.json"), "utf8"));
  const refreshed = JSON.parse(fs.readFileSync(path.join(__dirname, "../web/src/data/devotional-art-refreshed.json"), "utf8"));
  Object.assign(art.images, refreshed.images);
  const categories = { aarti: 99, chalisa: 53, stotram: 62, ashtakam: 37, mantra: 39, kavacham: 7 };
  const bodies = Object.fromEntries(Object.keys(categories).map((category) => [category,
    JSON.parse(fs.readFileSync(path.join(__dirname, `../web/src/data/devotional/${category}.json`), "utf8"))]));
  assert.equal(metadata.length, 295);
  assert.equal(new Set(metadata.map((entry) => entry.slug)).size, metadata.length);
  for (const [category, count] of Object.entries(categories))
    assert.equal(metadata.filter((entry) => entry.categories.includes(category)).length, count);
  for (const entry of metadata) {
    assert.match(entry.titleHi, /[\u0900-\u097f]/);
    assert.match(entry.slug, /^[a-z0-9]+(?:-[a-z0-9]+)*$/);
    assert.match(entry.source, /^https:\/\//);
    const verses = bodies[entry.categories[0]][entry.slug];
    const image = art.images[art.readings[entry.slug]];
    assert(image?.source && image?.license && /\.(png|webp)$/.test(image?.src), `Missing credited local artwork: ${entry.slug}`);
    const bytes = fs.readFileSync(path.join(__dirname, "../web/public", image.src));
    if (image.src.endsWith(".png")) assert.equal(bytes.subarray(0, 8).toString("hex"), "89504e470d0a1a0a");
    else {
      assert.equal(bytes.subarray(0, 4).toString(), "RIFF");
      assert.equal(bytes.subarray(8, 12).toString(), "WEBP");
    }
    assert(bytes.length <= 100000, `Oversized card image: ${image.src}`);
    assert(image.width <= 512 && image.height <= 512, `Oversized image dimensions: ${image.src}`);
    assert(verses?.length, `Missing reading: ${entry.slug}`);
    assert(verses.some((verse) => /[।॥]/u.test(verse)), `Missing prayer: ${entry.slug}`);
    assert(!/https?:\/\/|iframe|youtube|\[Button:|A\+A||&(?:amp|nbsp);|Translation of|सबसे प्रसिद्ध आरती|मन्त्र अर्थ|मंत्र अर्थ/u.test(verses.join("\n")), `Unexpected provider content: ${entry.slug}`);
  }
  const browser = await chromium.launch({ executablePath: process.env.CHROME_PATH || "C:/Program Files/Google/Chrome/Application/chrome.exe", headless: true, args: ["--disable-gpu", "--js-flags=--max-old-space-size=128"] });
  const errors = [];
  const result = { passed: false, readings: metadata.length, categories, checks: [], errors };
  try {
    const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, reducedMotion: "reduce" });
    await context.addInitScript(() => {
      if (!localStorage.getItem("m4y-language")) localStorage.setItem("m4y-language", "en");
      localStorage.setItem("m4y-analytics", "no");
    });
    await context.route("**/api/**", (route) => route.request().method() === "GET" ? route.continue() : route.abort());
    const page = await context.newPage();
    page.on("pageerror", (error) => errors.push(error.message));
    await page.goto(base);
    await page.locator(".site-header").waitFor();
    await page.getByRole("button", { name: "Devotional library", exact: true }).click();
    await page.locator(".devotional-card").first().waitFor();
    assert.equal(new URL(page.url()).pathname, "/bhakti");
    assert.equal(await page.locator(".devotional-card").count(), 12);
    assert.equal(await page.locator(".devotional-card-photo img[loading='lazy'][decoding='async']").count(), 12);
    await page.waitForFunction(() => {
      const photo = document.querySelector(".devotional-card-photo img");
      return photo?.complete && photo.naturalWidth > 0;
    });
    assert.equal(await page.locator(".devotional-card-invocation").count(), 0);
    assert.match(await page.locator(".devotional-pagination").innerText(), /Showing 1–12/);
    const secondPageFirst = metadata[12].title;
    await page.locator(".devotional-pagination a[rel='next']").click();
    assert.equal(new URL(page.url()).searchParams.get("page"), "2");
    assert.equal(await page.locator(".devotional-card h3").first().innerText(), secondPageFirst);
    await page.locator(".devotional-card").first().click();
    await page.getByText("End of reading", { exact: true }).waitFor();
    await page.locator(".devotional-back").click();
    assert.equal(new URL(page.url()).pathname, "/bhakti");
    assert.equal(new URL(page.url()).searchParams.get("page"), "2");
    assert.equal(await page.locator(".devotional-card h3").first().innerText(), secondPageFirst);
    await page.reload();
    await page.locator(".devotional-card").first().waitFor();
    assert.equal(await page.locator(".devotional-card").count(), 12);
    await page.locator(".devotional-pagination").getByRole("link", { name: "Page 25", exact: true }).click();
    assert.equal(await page.locator(".devotional-card").count(), 7);
    assert.equal(await page.locator(".devotional-card h3").last().innerText(), metadata.at(-1).title);
    await page.goBack();
    assert.equal(new URL(page.url()).searchParams.get("page"), "2");
    await page.getByRole("searchbox").fill("Shitala");
    assert.equal(new URL(page.url()).searchParams.get("page"), null);
    assert(await page.locator(".devotional-card").count() > 0);
    assert.equal(await page.locator(".devotional-pagination").count(), 0);
    await page.getByRole("searchbox").fill("");
    await page.goto(base + "/bhakti?page=999999");
    await page.locator(".devotional-card").first().waitFor();
    assert.equal(await page.locator(".devotional-card").count(), 7);
    await page.goto(base + "/bhakti?page=-2");
    await page.locator(".devotional-card").first().waitFor();
    assert.equal(await page.locator(".devotional-card").count(), 12);
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    await page.screenshot({ path: ".local/devotional-desktop.png", fullPage: false });
    result.checks.push("All 295 readings are available through 12-card pages; reader return, refresh, history, last page and full-collection search work");
    for (const [category, count] of Object.entries(categories)) {
      await page.locator(`.devotional-categories a[href='/bhakti/${category}']`).click();
      assert.equal(await page.locator(".devotional-card").count(), Math.min(12, count));
    }
    await page.getByRole("searchbox").fill("नारायण");
    assert.equal(await page.locator(".devotional-card").count(), 1);
    await page.getByRole("searchbox").fill("Narayana");
    assert.equal(await page.locator(".devotional-card").count(), 1);
    const kavachamLengths = { "shri-narasimha-kavacham": 34, "shri-narayana-kavacham": 48,
      "shri-bhairava-kavacham": 11, "shri-durga-devi-kavacham": 59, "shri-lakshmi-kavacham": 33,
      "shri-sita-kavacham": 47, "shri-shitala-kavacham": 52 };
    for (const [slug, count] of Object.entries(kavachamLengths)) {
      await page.goto(base + `/bhakti/kavacham/${slug}`);
      await page.getByText("End of reading", { exact: true }).waitFor();
      assert.equal(await page.locator(".devotional-verses p").count(), count);
      assert.equal(await page.locator(".devotional-verses p").first().innerText(), bodies.kavacham[slug][0]);
      assert.equal(await page.locator(".devotional-verses p").last().innerText(), bodies.kavacham[slug].at(-1));
      assert.equal(await page.locator("iframe,video,audio,a[href*='youtu']").count(), 0);
    }
    await page.reload();
    await page.getByText("End of reading", { exact: true }).waitFor();
    await page.getByRole("combobox", { name: "Language / भाषा" }).selectOption("hi");
    await page.getByRole("heading", { name: "श्री शीतला कवचम्", exact: true }).waitFor();
    await page.getByRole("combobox", { name: "Language / भाषा" }).selectOption("en");
    result.checks.push("All seven complete Kavacham readings, Hindi/English category search, direct URLs and language switching");
    await page.goto(base + "/bhakti/chalisa");
    await page.getByRole("searchbox").fill("Hanuman");
    assert.equal(await page.locator(".devotional-card").count(), 1);
    await page.locator(".devotional-card").click();
    await page.getByText("End of reading", { exact: true }).waitFor();
    const hanuman = metadata.find((entry) => entry.slug === "shri-hanuman-chalisa");
    assert.equal(await page.locator(".devotional-verses p").count(), bodies.chalisa[hanuman.slug].length);
    assert.match(await page.locator(".devotional-verses").innerText(), /जय हनुमान ज्ञान गुण सागर/u);
    assert.match(await page.title(), /Hanuman Chalisa/);
    assert.equal(await page.locator("iframe,video,audio,a[href*='youtu']").count(), 0);
    assert.equal(await page.locator(".devotional-verses").getAttribute("lang"), "hi");
    await page.getByRole("button", { name: "Increase text size", exact: true }).click();
    assert.equal(await page.locator(".devotional-verses").evaluate((el) => getComputedStyle(el).fontSize), "24px");
    await page.getByRole("combobox", { name: "Language / भाषा" }).selectOption("hi");
    await page.getByRole("heading", { name: hanuman.titleHi, exact: true }).waitFor();
    assert.equal(await page.locator(".devotional-verses").evaluate((el) => getComputedStyle(el).fontSize), "24px");
    await page.locator(".devotional-back").click();
    assert.equal(new URL(page.url()).searchParams.get("q"), "Hanuman");
    assert.equal(await page.locator(".devotional-card").count(), 1);
    await page.goBack();
    await page.locator(".devotional-reader").waitFor();
    await page.reload();
    await page.getByText("पाठ समाप्त", { exact: true }).waitFor();
    await page.getByRole("combobox", { name: "Language / भाषा" }).selectOption("en");
    result.checks.push("Complete Hanuman Chalisa, text sizing, language changes, search return, Back and refresh");
    for (const slug of ["shree-suktam", "mahalakshmi-ashtakam", "shri-baba-gangaram-chalisa", "shri-parashurama-ashtakam", "lord-ganesha-mantras"]) {
      const entry = metadata.find((item) => item.slug === slug);
      assert(entry, slug);
      const category = entry.categories.at(-1);
      await page.goto(base + `/bhakti/${category}/${slug}`);
      await page.getByText("End of reading", { exact: true }).waitFor();
      const expected = bodies[entry.categories[0]][slug];
      assert.equal(await page.locator(".devotional-verses p").first().innerText(), expected[0]);
      assert.equal(await page.locator(".devotional-verses p").last().innerText(), expected.at(-1));
    }
    await page.goto(base + "/bhakti/invalid/prayer");
    await page.getByRole("heading", { name: "Page not found", exact: true }).waitFor();
    await page.goto(base + "/bhakti/mantra/shri-hanuman-chalisa");
    await page.getByRole("heading", { name: "Page not found", exact: true }).waitFor();
    await page.goto(base + "/bhakti");
    await page.getByRole("searchbox").fill("हनुमान");
    assert(await page.locator(".devotional-card").count() > 0);
    await page.getByRole("searchbox").fill("zzzz-no-such-prayer");
    await page.getByRole("heading", { name: "No readings found", exact: true }).waitFor();
    await page.getByRole("button", { name: "Clear search", exact: true }).click();
    assert.equal(await page.locator(".devotional-card").count(), 12);
    result.checks.push("All category counts, Hindi/English search, empty state, long texts, shared-category and invalid routes");
    const desktopAxe = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa"]).analyze();
    assert.deepEqual(desktopAxe.violations.map((item) => ({ id: item.id, nodes: item.nodes.length })), []);
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto(base);
    await page.locator(".menu-toggle").click();
    await page.getByRole("button", { name: "Devotional library", exact: true }).click();
    await page.locator(".devotional-card").first().waitFor();
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    await page.screenshot({ path: ".local/devotional-mobile.png", fullPage: false });
    await page.locator(".devotional-categories a[href='/bhakti/kavacham']").click();
    assert.equal(await page.locator(".devotional-card").count(), 7);
    await page.screenshot({ path: ".local/kavacham-mobile.png", fullPage: false });
    await page.locator(".devotional-card").first().click();
    await page.getByText("End of reading", { exact: true }).waitFor();
    await page.getByRole("button", { name: "Increase text size", exact: true }).click({ clickCount: 6 });
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    const mobileAxe = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa"]).analyze();
    assert.deepEqual(mobileAxe.violations.map((item) => ({ id: item.id, nodes: item.nodes.length })), []);
    await page.screenshot({ path: ".local/devotional-reader-mobile.png", fullPage: true });
    await page.setViewportSize({ width: 320, height: 740 });
    await page.waitForFunction(() => innerWidth === 320 && document.documentElement.scrollWidth <= innerWidth, null, { timeout: 5000 });
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    result.checks.push("Desktop/mobile layouts and WCAG A/AA accessibility scans pass; no media in the reader");
    assert.deepEqual(errors, []);
    result.passed = true;
  } finally {
    await browser.close();
    fs.writeFileSync(path.join(__dirname, "../.local/devotional-verification.json"), JSON.stringify(result, null, 2));
  }
  process.stdout.write(JSON.stringify(result, null, 2) + "\n");
}

verifyDevotionalLibrary().catch((error) => { process.stderr.write(error.stack + "\n"); process.exitCode = 1; });
