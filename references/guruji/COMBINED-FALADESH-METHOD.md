# Combined Faladesh method — 1 November 2026

Historical v1–v4 contract. New reports now use the
[v5 advanced assessment and shared reader contract](ADVANCED-ASSESSMENT-METHOD.md).
The older presentation paths remain for retained snapshots.

Version: `m4y-cross-factor-faladesh-v4`.

The board is the authoritative classification source for this feature, as
explicitly requested by the user. See [role, sign and house rules](ROLE-SIGN-HOUSE-METHOD.md).

New report snapshots contain a deterministic interpretation under
`guruji_faladesh.analysis`. It is an application interpretation using declared
rules, not a Guruji approval record, a calibrated probability model or a
complete implementation of every Jyotish tradition.

## Inputs and decisions

- Read D1 ownership, subject-house occupants, significators and native graha
  aspects on the houses or their owners/significators. Nodes receive neither
  invented ownership nor invented aspects.
- Own/exaltation sign supports relevant owners and significators only when
  their natal board role is Karak. Debilitation qualifies a Karak's favourable
  capacity. Marak, neutral and unlisted roles receive no automatic outcome
  direction from sign strength alone. Checked combustion remains a separately
  calculated qualification. A principal lord in a different
  6th/8th/12th house calls for attention to effort or adjustment; it does not
  establish failure by itself.
- Same-sign associations are labelled whole-sign conjunctions. A node with a
  relevant owner/significator qualifies the reading; it is not automatically
  called an eclipse or a named Dosha.
- Record incoming aspects, dispositor condition and retrogradation. A lord's
  aspect on its own main house supports that house. Other aspects remain
  contextual in v1; every Saturn/Mars aspect is not automatically harmful.
- D9 and the subject division qualify the principal lord using classical
  own/exaltation/debilitation signs. D10 is used for work, D24 for learning,
  D2 for resources and D4 for home. Divisions never acquire physical combustion
  or transit calculations.
- Keep Chalit and D1 houses distinct. Keep actual Sarvashtakavarga bindus as
  context without converting them into percentages. Full Shadbala is not
  computed.
- Retain matched yoga configurations, Kendra–Trikona relationships and checked
  Mangal cancellation. A missing yoga is not an adverse result; cancelled
  Mangal does not trigger a mandatory remedy.
- Keep the board role, independent sign-status flags and D1 whole-sign house
  separately stored. The role/sign combination qualifies capacity without
  changing the role or declaring the final outcome. No alternative role table
  from the earlier doctrinal comparison replaces the board.

Supporting and demanding conditions remain visible together. If both exist,
the reading is mixed rather than resolved by subtracting numerical scores.
`supportive`, `attention`, `mixed` and `context` describe these declared rules,
not measured likelihoods. Every main-point statement retains the IDs of its
actual evidence. Up to two causes in each direction are shown concisely, with
the principal subject lord first;
the complete fact list is in the optional reference section.

Category conclusions now come from the principal subject lord's actual D1
destination, not one of four assessment paragraphs. The original bilingual
library `internal/platform/data/faladesh-findings.json` covers seven subjects
and twelve destination houses (84 readings), plus subject-specific occupant
themes. `scripts/build-faladesh-findings.py` reproduces the library. These are
application interpretations of house relationships, not source quotations or
new Guruji approvals. External traditions never replace the board's roles.

Each topic retains up to five findings: the principal owner, a relevant occupied
house, a distinct secondary owner, a calculated qualification and a valid
period connection. Shared owners are merged. For example, Aquarius wealth and
gains both belong to Jupiter; its placement in house 10 is explained once.
Same-house Moon/Saturn and Sun/Mercury/Jupiter associations receive bounded,
original subject readings without declaring a profession, disease or named
Dosha. A principal owner's role/sign/combustion qualifies its own finding;
relevant divisional conflict or another planet's condition stays visible.

Every finding carries bilingual text, a factual basis, capacity qualifications,
scope (`natal` or `period`), retained evidence IDs and reference IDs. Generic
planet-house paragraphs remain in optional source references. General brings
together personal direction, career, finances and relationships with their
original evidence; its short opening summarises personal direction and work.
Birth details do not establish that an event has happened.
No job loss, illness, relationship incident or past event is fabricated.

## Periods and remedies

Both running major/sub lords must match real start-inclusive/end-exclusive
periods at saved `as_of`. Stale names or `current_path` cannot supply timing.
Period relevance requires ownership or occupancy; a generic significator
alone is insufficient to assert current subject emphasis.

The main reading combines saved findings, qualifications and practical guidance
as connected paragraphs under the category's single title. Intermediate finding
titles and the separate detail/next-step sections are omitted. Exact repeated
sentences are removed while distinct detail and the closing guidance are kept.
The same presentation applies to new PDFs. Technical basis paragraphs are retained in the snapshot but omitted
from the main web/PDF reading. The public website no longer exposes calculation
references, source passages, method versions or the evidence ledger. These remain
in the original files and saved data for internal verification. Reader pages show
the interpretation and useful result dates; original PDF reference handling is separate.
Dasha dates are shown on the Dasha subject; validated subject-specific period
findings can appear alongside natal findings. Dasha also explains each running
lord's actual natal house/ownership themes. Web aliases `marriage`, `health`
and `debt` map to Relationships, Wellbeing and Money, matching PDF grouping.
Only superseded, unsaved guest v1/v2/v3 readings refresh on opening Faladesh; saved
reports and purchased snapshots retain their original interpretation.

The earliest related Antardasha within twelve months is shown when it belongs
to the retained current Mahadasha. It is a review interval, not an event date.
Cross-Mahadasha future interpretation, complete transit Vedha, all cancellation
doctrines and exact event timing remain outside v1. Jupiter/Saturn transits
are dated context, not automatic auspicious/adverse event predictions.

Each category separates the thematic interpretation from its actual chart
cause and any calculated qualification. Challenging factors may suggest individual remedy
assessment and practical adjustments. Supporting factors receive constructive
guidance, never an invented Dosha needing removal. No compulsory ritual,
personal gemstone or guaranteed remedy effect is added.

The approach is informed by [Phaladeepika chapter 15](https://www.wisdomlib.org/hinduism/book/phaladeepika-by-mantreswara-text-and-translation/d/doc1621587.html)
on houses, lords, significators and opposing conditions, and [chapter 19](https://www.wisdomlib.org/hinduism/book/phaladeepika-by-mantreswara-text-and-translation/d/doc1621591.html)
on qualified period interpretation. These transcriptions warn of OCR errors.
The application rules are a bounded adaptation, not an exhaustive translation.
The owner/destination structure also follows the broad house-lord framework of
[Brihat Parashara Hora Shastra, chapter 24](https://vedic-astro.s3.amazonaws.com/books/bhrihat_parasara_hora_shastra.pdf).
The library does not reproduce deterministic promises, adverse medical claims
or literal translations of its OCR text.

## Persistence and verification

Analysis is created after expanded calculations, periods and the transit
snapshot are assembled. Incomplete charts, unknown-time reports and core-only
calculations fail closed. Opening an older report never creates a new analysis.
Purchased PDFs use retained snapshots; new PDFs include the same category
readings and main points without changing original source text.

Scenario tests cover opposing dignity/combustion, source-role isolation,
significator qualification, divisional disagreement, aspects to a displaced
lord, cancellations, provenance, period boundaries, earliest future interval,
incomplete input and immutable reloads. Browser checks cover both languages,
mobile/desktop layouts, keyboard disclosures and retained report generations.
New regression cases exercise every subject/destination combination, differing
readings despite identical assessment labels, Raja Babu's audited D1 house
connections, shared ownership, both running lords, expired timing suppression,
finding provenance and historical snapshot compatibility.
