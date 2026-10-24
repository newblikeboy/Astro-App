// Keep PDF calculation labels aligned with the website's Hindi vocabulary.
// Run after changing any of these locale catalogues; authored remedies are never translated here.
const fs = require('node:fs');
const path = require('node:path');
const root = path.resolve(__dirname, '..');
const files = ['en', 'report-en', 'kundli-details-en', 'panchang-en', 'sky-en', 'explorer-en'];
const labels = {};
for (const file of files) {
  const catalogue = JSON.parse(fs.readFileSync(path.join(root, 'web/src/locales', `${file}.json`), 'utf8'));
  for (const [hi, en] of Object.entries(catalogue)) labels[en] = hi;
}
fs.writeFileSync(path.join(root, 'internal/platform/data/pdf-hindi.json'), JSON.stringify(labels, null, 2) + '\n');
