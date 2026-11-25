const fs = require("node:fs");
const path = require("node:path");
const assert = require("node:assert/strict");

// Each row follows the guide's five existing instructions. Tuples name a source
// reading and inclusive one-based section positions; an omitted range keeps the
// complete recitation group intact. This is an editorial map, not keyword routing.
const placements = {
  diwali: [
    [["diwali-atma-shodhan"]],
    [["diwali-sankalpa"], ["diwali-shanti-patha"], ["diwali-mangal-patha"], ["diwali-kalash-sthapana"], ["diwali-ganapati"], ["diwali-navagraha"], ["diwali-matrika"], ["ganesha", 1, 4]],
    [["ganesha", 5, 10], ["lakshmi", 1, 26]],
    [["kali"], ["diwali-saraswati-accounts"], ["kubera"]],
    [["ganesha", 11, 19], ["lakshmi", 27, 34], ["diwali-deepmalika"], ["diwali-visarjan"]],
  ],
  lakshmi: [[], [["lakshmi", 1, 4]], [["lakshmi", 5, 20]], [["lakshmi", 21, 27]], [["lakshmi", 28, 34]]],
  ganesha: [[], [["ganesha", 1, 4]], [["ganesha", 5, 8]], [["ganesha", 9, 14]], [["ganesha", 15, 19]]],
  shiva: [[], [["shiva", 1, 5]], [["shiva", 6, 11]], [["shiva", 12, 19]], [["shiva", 20, 27]]],
  "maha-shivaratri": [
    [["maha-shivaratri-mantras", 2, 2]],
    [["maha-shivaratri-mantras", 1, 1], ["maha-shivaratri-mantras", 3, 4]],
    [["maha-shivaratri-mantras", 5, 6], ["maha-shivaratri-mantras", 11, 18]],
    [["maha-shivaratri-mantras", 7, 10], ["maha-shivaratri-mantras", 27, 27]],
    [["maha-shivaratri-mantras", 19, 26]],
  ],
  kubera: [[], [], [["kubera", 1, 3]], [["kubera", 4, 4]], [["kubera", 5, 5]]],
  "rama-navami": [[], [["rama-navami", 1, 3]], [["rama-navami", 4, 15]], [["rama-navami", 16, 20]], [["rama-navami", 21, 24]]],
  "krishna-janmashtami": [[], [["krishna-janmashtami", 1, 3]], [["krishna-janmashtami", 4, 16]], [["krishna-janmashtami", 17, 19]], [["krishna-janmashtami", 20, 23]]],
  holi: [[["holi", 1, 1]], [["holi", 2, 3], ["holi", 5, 5], ["holi", 7, 7], ["holi", 9, 9]], [["holi", 4, 4], ["holi", 6, 6], ["holi", 8, 8], ["holi", 10, 10]], [["holi", 11, 11]], []],
  saraswati: [[], [], [["saraswati", 1, 14]], [["saraswati", 15, 21]], [["saraswati", 22, 25]]],
  "karwa-chauth": [[["karwa-chauth-mantras", 1, 1]], [], [["karwa-chauth-mantras", 2, 2]], [["karwa-chauth-mantras", 3, 3]], []],
  ghatasthapana: [
    [],
    [["diwali-kalash-sthapana", 1, 2], ["diwali-kalash-sthapana", 10, 10]],
    [["diwali-kalash-sthapana", 3, 9], ["diwali-kalash-sthapana", 11, 11]],
    [["diwali-kalash-sthapana", 12, 15]],
    [["diwali-kalash-sthapana", 16, 18]],
  ],
  "ganesha-chaturthi": [[], [["ganesha-chaturthi", 1, 3]], [["ganesha-chaturthi", 4, 24]], [["ganesha-chaturthi", 25, 34]], [["ganesha-chaturthi", 35, 35]]],
  "raksha-bandhan": [[], [], [], [["raksha-bandhan"]], []],
  kali: [[], [["kali", 1, 2]], [["kali", 3, 3]], [["kali", 4, 4]], [["kali", 5, 5]]],
  durga: [[], [["durga", 1, 2]], [["durga", 3, 17]], [["durga", 18, 28]], [["durga", 29, 31]]],
  hanuman: [[["hanuman", 1, 1]], [["hanuman", 2, 3]], [["hanuman", 4, 18]], [["hanuman", 19, 27]], [["hanuman", 28, 36]]],
  hartalika: [[], [], [["hartalika-mantras"]], [], []],
  "ahoi-ashtami": [[], [], [], [], []],
  varalakshmi: [[], [["varalakshmi", 1, 2]], [["varalakshmi", 3, 15]], [["varalakshmi", 16, 20]], [["varalakshmi", 21, 32]]],
  dhanvantari: [[["dhanvantari", 1, 1], ["dhanvantari", 3, 3]], [["dhanvantari", 2, 2]], [["dhanvantari", 4, 9]], [["dhanvantari", 10, 17]], [["dhanvantari", 18, 21]]],
  govardhan: [[], [], [["govardhan-mantras", 2, 2]], [["govardhan-mantras", 1, 1]], []],
  "naraka-chaturdashi": [[], [["naraka-chaturdashi", 1, 1]], [["naraka-chaturdashi", 2, 2]], [], []],
  "vat-savitri": [[], [["vat-savitri", 1, 2]], [["vat-savitri", 3, 23]], [], [["vat-savitri", 24, 25]]],
  satyanarayana: [[], [], [["satyanarayana-mantras", 1, 14]], [["satyanarayana-mantras", 15, 17]], [["satyanarayana-mantras", 18, 20]]],
  "ganapati-homa": [[], [], [["ganapati-homa", 1, 2]], [["ganapati-homa", 3, 3]], [["ganapati-homa", 4, 6]]],
  "durga-homa": [[], [], [["durga-homa", 1, 2]], [["durga-homa", 3, 3]], [["durga-homa", 4, 6]]],
};

/** build expands the reviewed step assignments, rejecting missing or duplicated mantras before updating only the guide maps. */
function build() {
  const root = path.resolve(__dirname, "..");
  const file = path.join(root, "web/src/data/puja-guides.json");
  const guides = JSON.parse(fs.readFileSync(file, "utf8"));
  const mantras = JSON.parse(fs.readFileSync(path.join(root, "web/src/data/puja-mantras.json"), "utf8"));
  assert.deepEqual(Object.keys(placements).sort(), guides.map(guide => guide.slug).sort());
  let count = 0;
  for (const guide of guides) {
    assert.equal(placements[guide.slug].length, guide.steps.hi.length, guide.slug);
    const mapped = placements[guide.slug].map(ranges => ranges.flatMap(([key, start = 1, end = mantras[key]?.sections.length]) => {
      assert(guide.mantra_readings.includes(key), `Unreferenced reading: ${guide.slug}/${key}`);
      assert(Number.isInteger(start) && Number.isInteger(end) && start >= 1 && end >= start && end <= mantras[key].sections.length, `Invalid section range: ${guide.slug}/${key}`);
      return mantras[key].sections.slice(start - 1, end).map(section => section.id);
    }));
    const expected = guide.mantra_readings.flatMap(key => mantras[key].sections.map(section => section.id)).sort();
    assert.deepEqual(mapped.flat().sort(), expected, `Dropped or duplicated mantra: ${guide.slug}`);
    if (process.argv.includes("--check")) assert.deepEqual(guide.step_mantras, mapped, `Stale step map: ${guide.slug}; run node scripts/build-puja-step-map.cjs`);
    else guide.step_mantras = mapped;
    count += expected.length;
  }
  if (!process.argv.includes("--check")) fs.writeFileSync(file, JSON.stringify(guides, null, 2) + "\n");
  console.log(`Verified ${guides.length} original five-step sequences and ${count} mantra placements without text changes.`);
}
build();
