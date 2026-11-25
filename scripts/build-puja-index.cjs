// Canonical editorial content stays in puja-guides.json; the list imports only generated metadata.
const fs = require("node:fs");
const path = require("node:path");
const assert = require("node:assert/strict");
const crypto = require("node:crypto");

/** build validates bilingual guides, paired mantra meanings and reviewed text hashes before deriving a lightweight catalogue. */
function build() {
  const root = path.resolve(__dirname, "..");
  const guides = JSON.parse(fs.readFileSync(path.join(root, "web/src/data/puja-guides.json"), "utf8"));
  const mantras = JSON.parse(fs.readFileSync(path.join(root, "web/src/data/puja-mantras.json"), "utf8"));
  const audit = JSON.parse(fs.readFileSync(path.join(root, "references/puja/2026-11-24/mantra-audit.json"), "utf8"));
  const referenced = new Set();
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
      assert(guide.mantra_note?.[language]?.trim(), `Missing mantra coverage note: ${guide.slug}/${language}`);
      assert(guide.materials[language]?.length >= 3 && guide.steps[language]?.length >= 5, `Incomplete guide: ${guide.slug}/${language}`);
      assert([...guide.materials[language], ...guide.steps[language]].every(value => typeof value === "string" && value.trim()), `Empty text: ${guide.slug}`);
    }
    assert.equal(guide.steps.en.length, guide.steps.hi.length, `Different step counts: ${guide.slug}`);
    assert(Array.isArray(guide.mantra_readings), `Missing mantra mapping: ${guide.slug}`);
    assert.equal(new Set(guide.mantra_readings).size, guide.mantra_readings.length, `Duplicate mantra group: ${guide.slug}`);
    const coverage = audit.guides.find(item => item.slug === guide.slug);
    assert(coverage, `Missing source review: ${guide.slug}`);
    assert.deepEqual(guide.mantra_readings, coverage.readings, `Mantra coverage differs from review: ${guide.slug}`);
    assert.equal(coverage.status, guide.mantra_readings.length ? "available" : "not-in-reference");
    for (const key of guide.mantra_readings) {
      referenced.add(key);
      const reading = mantras[key];
      assert(reading?.sections.length && reading.title.hi && reading.title.en, `Missing mantra reading: ${guide.slug}/${key}`);
      assert(guide.sources.includes(reading.source), `Uncredited mantra reference: ${guide.slug}/${key}`);
    }
    assert(Array.isArray(guide.step_mantras) && guide.step_mantras.length === guide.steps.hi.length, `Missing step assignments: ${guide.slug}`);
    assert(guide.step_mantras.every(ids => Array.isArray(ids) && ids.every(id => typeof id === "string")), `Invalid step assignments: ${guide.slug}`);
    const expectedIds = guide.mantra_readings.flatMap(key => mantras[key].sections.map(section => section.id));
    const assignedIds = guide.step_mantras.flat();
    assert.equal(new Set(assignedIds).size, assignedIds.length, `Repeated mantra within guide: ${guide.slug}`);
    assert.deepEqual([...assignedIds].sort(), expectedIds.sort(), `Lost or unreferenced mantra in steps: ${guide.slug}`);
  }
  const sectionIds = new Set();
  for (const [key, reading] of Object.entries(mantras)) {
    assert(referenced.has(key), `Unused mantra reading: ${key}`);
    const reference = audit.references.find(item => item.key === key);
    assert.equal(reference?.url, reading.source, `Changed source URL: ${key}`);
    assert.equal(reference.sections.length, reading.sections.length, `Changed source coverage: ${key}`);
    for (const section of reading.sections) {
      assert(section.id && !sectionIds.has(section.id), `Duplicate mantra ID: ${section.id}`);
      sectionIds.add(section.id);
      assert(/[\u0900-\u097f]/.test(section.text) && section.text.trim(), `Missing traditional text: ${section.id}`);
      assert(section.title.hi && section.title.en && section.meaning.hi.trim() && section.meaning.en.trim(), `Missing title or meaning: ${section.id}`);
      assert(/[\u0900-\u097f]/.test(section.title.hi) && /[A-Za-z]/.test(section.title.en), `Wrong title script: ${section.id}`);
      assert(/[\u0900-\u097f]/.test(section.meaning.hi) && /[A-Za-z]/.test(section.meaning.en), `Wrong meaning script: ${section.id}`);
      const reviewed = reference.sections.find(item => item.id === section.id);
      assert.equal(crypto.createHash("sha256").update(section.text).digest("hex"), reviewed?.sha256, `Reviewed mantra changed: ${section.id}`);
      assert.equal(section.origin, reviewed.origin, `Unreviewed text origin: ${section.id}`);
    }
  }
  assert.equal(sectionIds.size, audit.traditional_sections, "Source mantra coverage changed");
  const index = guides.map(({ slug, kind, title, description, art, related }) => ({ slug, kind, title, description, art, related }));
  const file = path.join(root, "web/src/data/puja-index.json");
  const expected = JSON.stringify(index, null, 2) + "\n";
  if (process.argv.includes("--check")) assert.equal(fs.readFileSync(file, "utf8"), expected, "Puja index is stale; run node scripts/build-puja-index.cjs");
  else fs.writeFileSync(file, expected);
  console.log(`Verified ${guides.length} bilingual Puja/Homa guides, ${sectionIds.size} reviewed mantra sections, paired meanings and local reading/art links.`);
}
build();
