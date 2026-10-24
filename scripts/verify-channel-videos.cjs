const { chromium } = require("../web/node_modules/playwright");
const AxeBuilder = require("../web/node_modules/@axe-core/playwright").default;
const assert = require("node:assert/strict");
const fs = require("node:fs");
const catalog = require("../internal/platform/data/channel-videos.json");
(async () => {
  const browser = await chromium.launch({
    executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe",
    headless: true,
  });
  const results = [],
    errors = [];
  try {
    const context = await browser.newContext({
      reducedMotion: "reduce",
      isMobile: true,
      hasTouch: true,
    });
    await context.addInitScript(() => {
      localStorage.setItem("m4y-analytics", "no");
      localStorage.setItem("m4y-language", "en");
    });
    const page = await context.newPage();
    page.on("pageerror", (e) => errors.push(e.message));
    await page.route("**/api/**", (route) =>
      route.request().method() === "GET" ? route.continue() : route.abort(),
    );
    await page.goto("http://localhost:5173");
    await page.locator(".video-card").first().waitFor();
    assert.deepEqual(
      await page
        .locator(".video-card")
        .evaluateAll((nodes) => nodes.map((el) => el.dataset.videoId)),
      catalog.videos.map((v) => v.id),
    );
    assert.equal(
      await page.locator("#videos iframe").count(),
      0,
      "Players must not load before clicking",
    );
    // Load all lazy thumbnails and verify the actual selected assets exist.
    for (const image of await page.locator("#videos img").all()) {
      await image.scrollIntoViewIfNeeded();
      await image.evaluate((img) =>
        img.complete
          ? undefined
          : new Promise((resolve, reject) => {
              img.addEventListener("load", resolve, { once: true });
              img.addEventListener(
                "error",
                () => reject(new Error("thumbnail failed")),
                { once: true },
              );
            }),
      );
      assert(await image.evaluate((img) => img.naturalWidth > 0));
    }
    for (const language of ["en", "hi"]) {
      await page.locator(".language-selector select").selectOption(language);
      for (const width of [320, 339, 390, 650, 768, 1440]) {
        await page.setViewportSize({ width, height: 900 });
        await page.evaluate(() => document.fonts.ready);
        assert.equal(await page.locator(".video-card").count(), 10);
        const metrics = await page.locator("#videos").evaluate((section) => ({
          pageOverflow: document.documentElement.scrollWidth > innerWidth,
          rail: (() => {
            const rail = section.querySelector(".videos-grid");
            const r = rail.getBoundingClientRect();
            return {
              left: r.left,
              right: r.right,
              scrollable: rail.scrollWidth > rail.clientWidth,
            };
          })(),
          cards: [...section.querySelectorAll(".video-card")].map((el) => {
            const r = el.getBoundingClientRect();
            return {
              left: r.left,
              right: r.right,
              width: r.width,
              overflow: el.scrollWidth > el.clientWidth,
            };
          }),
        }));
        assert(
          !metrics.pageOverflow && metrics.cards.every((c) => !c.overflow),
          `${language}/${width}: video layout overflow`,
        );
        assert(
          metrics.rail.left >= 0 &&
            metrics.rail.right <= width &&
            metrics.rail.scrollable,
          "Video rail must contain horizontal browsing within the page",
        );
        for (const [index, video] of catalog.videos.entries()) {
          const card = page.locator(".video-card").nth(index);
          assert.equal(
            await card.locator("h3").textContent(),
            language === "en" ? video.english_title : video.title,
          );
          assert.equal(
            await card.locator("a").getAttribute("href"),
            `https://www.youtube.com/watch?v=${video.id}`,
          );
          assert(
            (await card.locator(".video-meta").innerText()).includes(
              String(new Date(video.published_at).getFullYear()),
            ),
          );
        }
        if ([339, 1440].includes(width)) {
          const audit = await new AxeBuilder({ page })
            .include("#videos")
            .withTags(["wcag2a", "wcag2aa", "wcag21aa"])
            .analyze();
          assert.deepEqual(
            audit.violations
              .filter((v) => ["serious", "critical"].includes(v.impact))
              .map((v) => ({ id: v.id, nodes: v.nodes.map((n) => n.target) })),
            [],
          );
          // Capture only the section; keep the sticky header outside this QA image.
          const header = page.locator(".site-header");
          const oldPosition = await header.evaluate((el) => el.style.position);
          await header.evaluate((el) => (el.style.position = "relative"));
          await page.locator("#videos").screenshot({
            path: `.local/channel-videos-${language}-${width}.png`,
          });
          await header.evaluate(
            (el, position) => (el.style.position = position),
            oldPosition,
          );
        }
        results.push({ language, width, passed: true });
      }
    }
    // Verify click-to-play wiring without loading a third-party video during QA.
    await page.route("https://www.youtube-nocookie.com/embed/**", (route) =>
      route.fulfill({
        contentType: "text/html",
        body: "<p>Embed request verified</p>",
      }),
    );
    await page.locator(".video-card").first().locator("button").click();
    assert.equal(
      await page.locator(".video-card iframe").first().getAttribute("src"),
      `https://www.youtube-nocookie.com/embed/${catalog.videos[0].id}?autoplay=1`,
    );
    assert.equal(
      await page
        .locator(".video-card iframe")
        .first()
        .getAttribute("referrerpolicy"),
      "strict-origin-when-cross-origin",
    );
    assert.deepEqual(errors, []);
    fs.writeFileSync(
      ".local/channel-videos-verification.json",
      JSON.stringify(
        {
          results,
          thumbnailCount: 10,
          axeAudits: 4,
          clickToPlay: true,
          errors,
        },
        null,
        2,
      ),
    );
    console.log(
      JSON.stringify({
        layouts: results.length,
        videos: 10,
        thumbnails: 10,
        axeAudits: 4,
        clickToPlay: true,
        errors,
        passed: true,
      }),
    );
  } finally {
    await browser.close();
  }
})().catch((error) => {
  console.error(error.stack);
  process.exitCode = 1;
});
