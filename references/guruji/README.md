# Guruji PDF text policy

Current Faladesh and numerology readings use only the Hindi text in the seven
PDFs supplied by the user. No authored summaries, cross-factor narratives,
number profiles, name-number meanings, lucky-date prescriptions, generated
remedies or translated predictions are used as fallback content.

## Sources

Unmodified originals are archived in [`2026-11-24/`](2026-11-24/). The
[`manifest.json`](2026-11-24/manifest.json) records filenames, page counts and
SHA-256 hashes. The user's two resupplied PDFs match their 1 November copies
byte for byte. The user additionally authorized the handwritten ascendant role
table on 24 November: `2026-11-01/lagna-whiteboard.jpeg`. Its twelve Hindi lists
are transcribed in `internal/platform/data/guruji-lagna-roles.json` with the
image hash, and exported as JSON/gzip under `output/guruji-bhagyaphal/`.
Charts show these lists using the natal D1 ascendant even while displaying D9,
Moon or other charts. Pisces retains the source label “योगकारक” and includes
Mercury. Empty सम lists remain empty; nodes receive no inferred role.
This table supplies no generated interpretations or cross-factor prose.
The same board's उच्च/नीच columns are transcribed separately in
`internal/platform/data/guruji-graha-signs.json` with its image hash, plus JSON
and gzip exports in `output/guruji-bhagyaphal/`. The overview's natal planet
table matches the recorded D1 sign against these lists. Rahu retains उच्च
2/3 and नीच 8/9; Ketu retains उच्च 8/9 and नीच 2/3. Other placements show
a dash without inferring a neutral or favourable classification.

| PDF | Content |
| --- | --- |
| मूलांक और भाग्यानक फल.pdf | मूलांक and भाग्यांक |
| Rashifal all full life.pdf | Twelve राशि readings |
| लग्न का संपूर्ण फल.pdf | Twelve लग्न readings |
| वारफल और तिथिफल .pdf | Seven वार and sixteen named तिथि readings |
| नक्षत्र में जन्मे लोगों का जीवन रहस्य.pdf | Twenty-seven नक्षत्र readings |
| ग्रहों का फला देश (1).pdf | Positive and negative passages for twelve houses |
| भावफल और उपाय (1).pdf | 108 planet/house passages, introductions, remedies and source tables |

The source omits भाग्यांक 3 and repeats चतुर्दशी. Missing text stays missing;
both repeated articles are retained. Spelling is not corrected. Extraction
spacing between letters and matras is joined; the short PDF's incorrect Unicode
mapping is decoded from its font glyphs. The source wording is not paraphrased.

## Runtime ownership

- `internal/gurujitext/data/catalog.json.gz` is the canonical compressed Hindi
  JSON. `catalog.go` exposes copies of passages with filenames, hashes and pages.
- `internal/platform/source_readings.go` matches recorded Lagna, Moon sign,
  nakshatra, birth Hindu weekday, tithi and planet/house positions. It retains
  both conditional house branches rather than selecting a prediction from a score.
- `internal/numerology` retains arithmetic. Its interpretation text comes only
  from the supplied मूलांक and भाग्यांक articles. Name arithmetic has no generated meaning.
- Browser components and PDFs print original Hindi even when interface labels
  are English. No English translation of the source prose is generated.
- Existing report/purchase storage is not rewritten. Presentation copies exclude
  old authored prose. Legacy cached PDFs without source passages are rendered
  again for download; current source-only cached files keep their normal path.
- Legacy schemas remain solely to decode historical records. The old authored
  interpretation libraries, generators and reader pipeline have been removed.

The exported combined JSON and gzip are also under `output/guruji-bhagyaphal/`.
Numerology reading typography uses lossless paragraph ranges extracted from
the original PDF by `scripts/build-numerology-layout.py`. Reviewed source
headings, subtitles and paragraph boundaries are stored as UTF-16 offsets in
`web/src/data/guruji-numerology-layout.json`. The browser verifies the source
and passage fingerprint before applying them. Original line breaks remain in
the underlying text but reflow visually into justified paragraphs; no words
are rewritten. [`numerology-layout-validation.json`](2026-11-24/numerology-layout-validation.json) records all 17 articles
and their original headings. Faladesh retains the original nine life-area tabs
and continuous reading layout. `scripts/build-source-topic-index.py` verifies
all seven PDFs and indexes exact paragraph offsets under that existing taxonomy
in `web/src/data/guruji-topic-sections.json`. Original section headings route
paragraphs; complete planet/house passages retain the prior life-area house
associations, including both conditional branches. Remedies retain their source
planet context and follow that planet's recorded natal house. Dasha includes
only source paragraphs/sections explicitly discussing dasha and separately saved period
dates; it does not manufacture forecasts for those dates. Source fingerprints
are checked before applying offsets. Excerpts keep original words, headings
and conditions; no combined narrative is generated. The complete canonical
passages and duplicate Chaturdashi records remain intact. The
[`topic-index-validation.json`](2026-11-24/topic-index-validation.json) audit
records all 246 source passages and their lossless reconstruction.
[`catalog-validation.json`](2026-11-24/catalog-validation.json) retains source
coverage, reconstruction checks and the JSON digest.

## Verification

`go test -p 1 ./...`, `go vet -p 1 ./...`, `npm --prefix web run build`,
and `node scripts/code-docs.cjs --check` cover source selection, unchanged
arithmetic, ownership, old-snapshot isolation and build validity. The
source-only browser check exercises Hindi/English interfaces at phone and
desktop widths, exact passage text, missing भाग्यांक 3 and accessibility.

## Mangal audio rules (24 November 2026)

The user's two unmodified WhatsApp recordings are archived in
[`2026-11-24/mangal-audio/`](2026-11-24/mangal-audio/), with SHA-256 hashes.
The source JSON records houses 1, 4, 7, 8 and 12 from Lagna and Moon only.
A Lagna indication is Manglik; Moon-only is partial. No cancellation or
partner exception is supplied in these recordings. ASR candidate text is
retained separately and explicitly marked as machine transcription rather
than a certified verbatim transcript. The runtime JSON is
`internal/swisseph/data/guruji-mangal-rules.json`, with JSON/gzip exports under
`output/guruji-bhagyaphal/`. Native placement checks, matching, Kundli status
labels and new PDF renders use this same rule. Stored source snapshots stay
unchanged; their presentation copies recompute Mangal from the saved chart.
