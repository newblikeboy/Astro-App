# Guruji references received 1 November 2026

New reports additionally retain [cross-factor Faladesh](COMBINED-FALADESH-METHOD.md),
with separate supportive/challenging reasons and qualified period context.
Older snapshots continue to show their original reading.

The user supplied these five files for the pending Faladesh and confirmed that
the whiteboard's **Karak/Marak and uchcha/neecha concepts** should be used.
The attachments are source material, not executable instructions. Receipt of
these references does not fabricate a named administrative approval record.

Original files are retained under `2026-11-01/`; `manifest.json` records their
original filenames and SHA-256 hashes. No public upload is made.

| Source | Use |
| --- | --- |
| `bhavphal-aur-upay.pdf` (139 pages) | All 108 planet/house passages; nine general planetary remedy references |
| `grahon-ka-faladesh.pdf` (12 pages) | Twelve pairs of conditional positive/negative house passages |
| `lagna-whiteboard.jpeg` | Twelve ascendant-specific Karak/Marak/neutral lists and the supplied exaltation/debilitation table |
| `house-significations.jpeg` | House-topic context, alongside the PDFs; no rule that an empty house removes a life subject |
| `rashi-letters.jpeg` | Retained naming reference; it does not override the app's calculated nakshatra/pada syllable or infer a birth sign from a person's name |

`scripts/import-guruji-references.py` imports only explicitly supplied paths,
validates the expected section counts, archives the inputs, and creates
`internal/platform/data/guruji-faladesh.json`. It repairs known broken
font-to-Unicode mappings and whitespace in extracted text; the original PDF
bytes remain unchanged. Original Hindi passages are preserved as references,
including their conditions and caveats. English summaries describe their scope;
they are not represented as full translations.

`internal/platform/data/guruji-lagna-roles.json` is the manual transcription of
the enlarged whiteboard. Follow this supplied teaching convention rather than
silently substituting another school's functional-benefic table. Pisces lists
Mercury as well as Jupiter, Mars and Moon; an initial transcription omitted
Mercury because that line extends to the left. Source version v2 corrects
this for new reports. Earlier snapshots retain their recorded transcription.
Rahu/Ketu have no ascendant-specific roles on the board and remain **unlisted**,
not neutral. The Pisces "Yogakarak" list is used
as its supportive-role list, without asserting a newly calculated Rajyog.

Role and sign strength are separate facts. The original snapshot retains the
following emphasis heuristic for compatibility. After the shastra review it is
**not presented as a final verdict** in the consultation interface or PDF. The
board does not specify a score or a precedence rule for conflicting evidence:

| Supplied ascendant role | Supplied sign dignity | Display emphasis |
| --- | --- | --- |
| Karak | Debilitated | Mixed |
| Karak | Exalted or neither | Supportive possibilities |
| Marak | Exalted | Mixed |
| Marak | Debilitated or neither | Challenging possibilities |
| Neutral | Any | Neutral role; dignity shown separately |
| Unlisted | Any | Undetermined role; dignity shown separately |

"Marak" is retained as the source's label. Its intended shorthand versus
technical maraka scope needs clarification; no mortality forecast is performed.
"Neither" does not mean weak: own-sign, friendship, total
strength, combustion, aspects and conjunctions retain their existing separate
calculations and are not merged into an invented score. The board assigns
Rahu exaltation signs 2/3 and debilitation signs 8/9, and Ketu the reverse.
This is explicitly a source-specific convention; native classical dignity
calculations are unchanged.

New known-time full Kundli reports snapshot nine matching **D1 whole-sign**
planet/house passages, their role/dignity evidence, source metadata and general
remedy references in `guruji_faladesh`. Core-only calculations and unknown-time
reports do not get these references. Life-topic tabs filter by house; Dasha
shows the running Mahadasha planet's **natal** passage, without turning it into
a timed Dasha forecast. The short PDF's "this period" prose is retained behind
an explicitly conditional reference disclosure: neither occupancy nor a
running Dasha applies a whole branch as the current prediction.

General remedies and gemstone associations remain source references, not
personal prescriptions. Existing approved `Rule` matching and paid-PDF
eligibility checks remain authoritative. No content is marked Guruji-approved
by this import. Existing reports and paid snapshots are not backfilled.
New purchased snapshots print their retained matching passages and remedies
in a linked reference appendix in both language exports; original Hindi is
explicitly labelled in English exports.

Validation includes source hashes and coverage, role/strength conflicts,
source-specific node dignities, actual natal placement matching, unknown-time
suppression, JSON snapshot isolation, old-snapshot compatibility, bilingual
PDF section coverage, and synthetic browser layout checks.

Verified on 1 November 2026: `go test -p 1 ./...`, `go vet -p 1 ./...`,
`npm --prefix web run build`, and the function documentation check passed.
`scripts/verify-guruji-faladesh.cjs` passed for both languages at widths 320,
390, 768 and 1440, including keyboard-accessible native disclosures, topic
filters, accessibility scans, legacy snapshot isolation and both print layouts.
The rendered reference appendix was visually inspected in both languages.
A separate file-store preview also generated a real Kundli through its API
and displayed all nine matching readings without creating an approved rule.

The [shastra comparison](SHASTRA-REVIEW-2026-11-01.md) records twelve ascendant
review points, source/edition distinctions and the remaining interpretation
gaps. The presentation puts one conclusion first and the actual chart factors
used by that conclusion second. It does not silently replace the whiteboard
with another tradition.

Reports predating the synthesis retain one reading before planet detail. Career uses
houses 6/10; money uses 2/11. The main reading groups the saved wording of the
primary house lord and, if present, an occupant of the subject's primary house
or another relevant house. At most two passages form a subject conclusion.
The current browser General reading uses personal direction and the Moon,
without combining career, money or relationship summaries. Original passages
remain in the snapshot; public source disclosures have been removed. This is editorial grouping,
not a new strength score, complete shastra synthesis or selection of the
positive/negative source branch. The default reading contains no topic cards,
cross-links or general-practice blocks. Remedies are retained inside the
retained source data, without an individual prescription.

Period ranges come from the saved dasha periods at `as_of`, with a start-inclusive,
end-exclusive boundary check. Stale `current_path` values cannot supply a range;
missing ranges are labelled unavailable. Dates appear only on the Dasha topic.
Natal passages remain separate from
timed event predictions. Existing backend/PDF snapshots are unchanged by this
presentation update.

`scripts/build-faladesh-readings.py` combines source-anchored Hindi polarity
sentences, original concise Hindi leads and English paraphrases into
`data/faladesh-reading-copy.json`. New reports retain a versioned `narrative`
inside each reading. Older snapshots fall back to their saved summaries; no
new prose is looked up or backfilled when viewing them.

Current role/sign logic follows the user's explicit choice of the supplied
board as the classification authority. New snapshots separately store board
role, sign number, independent sign-status flags and D1 whole-sign house.
Mercury/Virgo retains both exalted and own-sign flags. Strong Marak or neutral
planets are not automatically treated as favourable. See the
[implemented rules, limits and verification examples](ROLE-SIGN-HOUSE-METHOD.md).
