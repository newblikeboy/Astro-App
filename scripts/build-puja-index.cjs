// Canonical editorial content stays in puja-guides.json; the list imports only generated metadata.
const fs = require("node:fs");
const path = require("node:path");
const assert = require("node:assert/strict");

/** build validates both language editions, local artwork and related prayer destinations before deriving a lightweight public catalogue. */
function build() {
  const root = path.resolve(__dirname, "..");
  const guides = JSON.parse(fs.readFileSync(path.join(root, "web/src/data/puja-guides.json"), "utf8"));
  const artwork = JSON.parse(fs.readFileSync(path.join(root, "web/src/data/devotional-art.json"), "utf8"));
  const readings = JSON.parse(fs.readFileSync(path.join(root, "web/src/data/devotional-index.json"), "utf8"));
  const seen = new Set();
  assert(guides.length > 0, "The Puja collection must not be empty");
  for (const guide of guides) {
    assert(/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(guide.slug) && !seen.has(guide.slug), `Invalid or duplicate slug: ${guide.slug}`);
    seen.add(guide.slug);
    assert(["festival", "deity", "vrat", "homa"].includes(guide.kind), guide.slug);
    assert(artwork.images[guide.art], `Missing subject art: ${guide.slug}`);
    assert(fs.existsSync(path.join(root, "web/public", artwork.images[guide.art].src)), `Missing local PNG: ${guide.slug}`);
    const [, prefix, category, slug] = guide.related.split("/");
    assert(prefix === "bhakti" && readings.some(item => item.slug === slug && item.categories.includes(category)), `Missing related prayer: ${guide.slug}`);
    assert(guide.sources.length && guide.sources.every(source => source.startsWith("https://www.drikpanchang.com/")), `Missing reviewed reference: ${guide.slug}`);
    for (const language of ["en", "hi"]) {
      assert(guide.title[language] && guide.description[language] && guide.note[language], `Missing translation: ${guide.slug}/${language}`);
      assert(guide.materials[language]?.length >= 3 && guide.steps[language]?.length >= 5, `Incomplete guide: ${guide.slug}/${language}`);
      assert([...guide.materials[language], ...guide.steps[language]].every(value => typeof value === "string" && value.trim()), `Empty text: ${guide.slug}`);
    }
    assert.equal(guide.steps.en.length, guide.steps.hi.length, `Different step counts: ${guide.slug}`);
  }
  const index = guides.map(({ slug, kind, title, description, art, related }) => ({ slug, kind, title, description, art, related }));
  const file = path.join(root, "web/src/data/puja-index.json");
  const expected = JSON.stringify(index, null, 2) + "\n";
  if (process.argv.includes("--check")) assert.equal(fs.readFileSync(file, "utf8"), expected, "Puja index is stale; run node scripts/build-puja-index.cjs");
  else fs.writeFileSync(file, expected);
  console.log(`Verified ${guides.length} bilingual Puja/Homa guides and their local reading/art links.`);
}
build();
