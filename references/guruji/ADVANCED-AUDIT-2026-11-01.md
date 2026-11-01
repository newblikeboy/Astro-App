# Calculation and Faladesh follow-up — 1 November 2026

The requested follow-up addresses the material interpretation, strength,
transit, matching and remedy gaps from the original audit. Personalised Vivah
Muhurat remains excluded. Existing planetary-position calculations and Guruji's
board classification were preserved.

## Implemented

| Area | Result |
| --- | --- |
| Planetary strength | All six Shadbala groups and their subcomponents, seven classical planets, explicit units and requirements; no invented node strength |
| Debility and yogas | Four Phaladeepika cancellation conditions, original debility retained, participant-strength/combustion/war qualifications for the supported yoga catalog |
| Supporting planets | Aspectors and dispositors use their own strength and obstacles; own/exalted Marak is not automatic support |
| Personal transit | Seven-planet lunar Gochar/Vedha table, actual blockers and reciprocal exemptions; Tara and from-Lagna positions kept separately |
| Matching exceptions | Five Nadi and four unambiguous Bhakoot conditions, with raw dosha, active assessment and unchanged base Guna scores |
| Mangal | Strong Jupiter/Venus exception now requires actual Shadbala and obstacle checks; pair-level conclusions use the qualified individual assessments |
| Subject findings | Primary and distinct secondary owners, occupants, qualified aspects/conjunctions, dispositor, D9/subject varga, BAV, yogas and valid linked periods affect the reading |
| General | Four separate personal-direction paragraphs; other subjects' forecasts remain in their own tabs |
| Remedies | Conditional selection from actual subject-linked causes, source-page locators, optional simple practices; no automatic gemstones or fabricated personal approval |
| Shared presentation | New v5 reports retain exact Hindi/English paragraphs consumed by API, browser and main PDF chapters |
| Existing reports | Historical fallbacks remain; only superseded unsaved guest readings refresh. The v4-only acceptance bug in guest refresh is corrected |

Calculation methods, primary references and deliberate convention differences
are specified in [the assessment contract](ADVANCED-ASSESSMENT-METHOD.md).

## Verification

The final combined suite, including real Chrome PDFs, is recorded in
`.local/advanced-complete-tests.jsonl`: **646 tests/subtests passed, zero
failures** (449 native, 191 platform, six numerology). This includes all five
optional Chrome PDF integration tests. Only the unrelated external-database
integration test was skipped. `go vet` and the backend build passed.

Seven real PDF artifacts were rendered in Hindi/English, including long-content
stress cases. The final reader pages and strength tables were visually inspected;
the text-boundary scan found zero spans outside their page. The final strict
frontend build also passed.

Additional checks passed:

- Independent published Shadbala geometry: seven Uchcha values, seven Dig
  values, Ojayugma and Kendradi compared with AstroSage's public example.
  Full totals were not forced to match different formula conventions.
- Every lunar-pada matching pair (108 by 108), all declared exception branches,
  complete favourable-place/Vedha table, reciprocal exemptions and actual
  blocking. Raw Guna scores remain unchanged.
- Isolated debility-cancellation rules, actual full-strength day/night samples,
  war boundaries/ties/conservation, weak/combust/unresolved strong-benefic cases.
- Two requested birth examples, all nine categories in both languages,
  valid retained evidence IDs and exact browser/PDF paragraph consumption.
- Changing strength, divisional context, supporting-planet capacity and yoga
  qualification changes findings; a birthplace-local reference-date regression
  guards midnight formatting.
- Responsive reading at 320/390/768/1440 pixels, keyboard navigation,
  accessibility and guest refresh; no page errors.
- Nine Kundli sections, four matching sections, daily/sky views and the existing
  Vivah view at 390/1440 pixels in Hindi/English; no horizontal overflow.
- Strict TypeScript build, original language/legacy-reader checks, generated
  copy checks and documentation coverage for 1,150 named functions.

Browser records are `tmp/advanced-qa/purnia/browser-results.json` and
`tmp/reader-results/results.json`. Pure shared-reader checks cover 290 retained
paragraphs across the two examples and languages. These are traceability and
presentation checks, not an accuracy percentage for predicted life events.

## Cleanup and compatibility

The earlier audit removed compiler-reported unused frontend symbols, obsolete
Faladesh CSS and the unused `pdfPlacementChart`/`pdfNames` Go wrappers. Strict
unused-symbol checking remains enabled. The reader version is now shared by
the frontend's rendering and guest-refresh paths. Live legacy readers and
snapshot renderers remain because existing reports still require them.

## Boundaries retained explicitly

The implementation follows a finite, documented rule catalog. Contradictory
Bhakoot passages, unspecified high-Guna/age thresholds, ambiguous distribution
rules and every possible cancellation doctrine are not guessed into additional
results. No exact personal event date or verified life-event history is claimed.
These limits do not change Guruji's board roles or original sign/house data.

The local API was subsequently restarted with
`.local/server-advanced-assessment.exe`. The live health endpoint returned
`status: ok`, and a fresh Raja Babu guest report returned analysis version
`m4y-cross-factor-faladesh-v5`. The frontend on port 5173 also returned HTTP 200.
The earlier browser checks used fresh native report/matching fixtures;
daily-tool requests at that time used the previous local service.

## Rajyog follow-up — 2 November 2026

Three reported charts genuinely match the implemented base relationship rules:
Raja Babu has Sun–Mercury and Mars–Venus conjunctions; the Purnia example has
Sun–Mars exchange and Moon–Sun mutual aspect; 28 August 1997, 16:00, Delhi has
one Mercury–Sun conjunction covering both seventh/ninth and tenth/ninth lordship.
The control of 21 September 2001, 22:30, Delhi does not match this Rajyog check.

The follow-up found genuine assessment and presentation bugs: strength included
non-matching candidate lords, synthesis could attach non-matching relationships
to subjects, and the same general caution described independent matched pairs.
Qualification method v2 filters matched participants and assesses each pair
separately. Repeated lordship combinations for the same two planets are shown
once. Web and PDF chapters use the same saved bilingual pair-specific reading.
Superseded unsaved guest charts refresh when opening the reading or yoga tab;
historical saved snapshots are not rewritten. The base detection convention,
Guruji's role table and planetary-position calculations remain unchanged.
The local API now runs `.local/server-rajyog-pair-reading.exe`; its health
endpoint returned `status: ok` and live guest responses include the new reading.
