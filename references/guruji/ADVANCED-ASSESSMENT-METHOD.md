# Advanced assessment contract — 1 November 2026

This extends the calculation audit without changing planetary positions, the
teacher's lagna-role classification, saved purchases or the Vivah shortlist.
It is a versioned interpretation model, not certification of predicted events
or equivalence to every astrological school's software.

## Native strength and qualifications

`m4y-shadbala-bphs-swiss-means-v1` retains all six groups in unrounded virupas:
Sthana (Uchcha, Saptavargaja, Ojayugma, Kendradi, Drekkana), Dig, Kala
(Nathonnatha, Paksha, Tribhaga, Abda, Masa, Vara, Hora, Ayana, Yuddha),
Cheshta, Naisargika and Drik. Seven classical planets receive strength; nodes
do not receive invented Shadbala scores. Strength is capacity, not a replacement
for Guruji's Karak/Marak/Sam role.

Primary formula reference: BPHS chapters 3, 26 and 27, R. Santhanam volume 1,
printed pp.256–286. The source includes variants and printed/OCR errors, so the
following conventions are explicit rather than silently mixed:

- Saptavargaja uses 45/30/20/15/10/4/2 and natal compound friendship. The
  degree-defined Moolatrikona interval is applied to D1; other vargas use
  ownership/compound friendship. D1 whole signs determine Kendradi; actual
  Sripati house middles determine Dig. A Chalit shift never replaces D1 placement.
- Solar raw Ayana occurs once in Kala and once as solar Cheshta; lunar phase
  Paksha likewise occurs once in Kala and once as lunar Cheshta. Neither is
  doubled in Kala and counted again a third time. Modern declinations are used;
  Ayana's reference obliquity is 23°27′, clamped to the stated 0–60 range.
- Cheshta uses Swiss J2000 osculating orbital mean longitudes, expressed with
  the fixed J2000 Lahiri origin, and circular mean/true midpoints. Mercury and
  Venus use the inner-planet mean/Sun interchange. This is a declared modern
  astronomical adaptation, not a claim to reproduce historical mean-motion
  tables or every JHora/Raman configuration exactly.
- Year/month lords use the source's worked 1 June 1984 example (Ahargana 65295),
  a 360-day year and 30-day month. Its printed year divisor conflicts with its
  worked quotient; 360 reproduces the worked result. Epoch: 31 December 1983
  = 65142. Hora uses equal twenty-four parts of the sunrise day. Tribhaga uses
  actual day/night thirds, including the preceding sunset before civil sunrise.
- BPHS 26's directed piecewise verse rules are used for aspect strength. The
  ordinary printed speculum independently checks the base curve. BPHS 27.19's
  extra Mercury/Jupiter aspect contribution is retained. Mercury's natural
  character for this arithmetic does not replace its board role.
- The five starry planets can enter war within one degree. The selected
  latitude convention gives the northern planet the victory; equal latitude
  stays unresolved. BPHS 27.20's difference in pre-war strengths is transferred
  simultaneously, preserving total strength. This does not simulate every
  visual-brightness or disc-size exception described in other war traditions.

Independent numeric comparison uses the shared geometric components in
[AstroSage's published example, PDF p.108](https://www.astrosage.com/pdf/vedic-report-en.pdf):
seven Uchcha values, seven Dig values, Ojayugma and Kendradi. Its full totals
are not forced onto this model: for example, its Sun minimum is five rupas,
whereas the selected BPHS 27.32–33 explicitly requires 390 virupas (6.5 rupas).
The independent comparison therefore does not certify identical full totals.

`m4y-phaladeepika7-neechabhanga-strength-v2` tests four explicit Phaladeepika
7.26–29 conditions: dispositor in a Kendra from Lagna/Moon; exaltation-sign
lord in a Kendra from Lagna/Moon; their mutual Kendras; dispositor's full
classical aspect on the debilitated planet. Cancellation is a separate flag:
the original debility and the board role remain unchanged. No automatic
retrograde cancellation is invented. The eleven existing base patterns,
Gajakesari/Budha-Aditya base checks and Kendra–Trikona relationships receive
participant-strength, combustion, uncancelled-debility and war qualifications.
Rajyog strength uses only pairs with an actual conjunction, mutual aspect or
exchange. Each matched pair retains an independent assessment; non-matching
tested lords and separate Yogakaraka ownership cannot supply a limitation.
The reader groups repeated ownership combinations involving the same two
planets into one relationship and uses the same bilingual text in web and PDF.
An absent yoga is not interpreted as harm. This catalog is finite; “every yoga
or cancellation in every text” is not a supported claim.

## Personal transits and matching

`m4y-lunar-gochar-vedha-v1` uses the exact seven-planet favourable-place/Vedha
pairs in P.V.R. Narasimha Rao, *Vedic Astrology: An Integrated Approach*, table
63, printed p.347. Sun/Saturn and Moon/Mercury are reciprocal exemptions.
All nine actual transiting planets can otherwise obstruct; nodes have no
invented favourable-place table. From-Lagna position and lunar-star Tara are
retained separately. No sign-Vedha result is substituted for Tara, a full
Sarvatobhadra model, or an exact event date.

`m4y-ashtakoota-aifas2021-exceptions-v1` tests five Nadi rules and four
unambiguous Bhakoot rules from AIFAS *Horoscope Matching* (2021), PDF pp.72–73
and 80. Each condition retains its page. Base factors and total out of 36 stay
unchanged; `base_present`, `active`, and `cancelled` are separate. The source's
contradictory 5/9-direction paragraph and ambiguous lord-1/7 sentence are not
used to invent an additional cancellation. Both UI and matching PDF show the
separate assessment. The model is not a complete couple-level marriage verdict.

`m4y-mangal-aifas2021-shadbala-v2` additionally resolves AIFAS PDF p.87's
previously pending strong Jupiter/Venus condition in both Kundli and matching.
Own/exalted sign and house 1/7 must coincide with the planet's six-component
strength meeting its requirement, no combustion, and no defeated/unresolved
planetary war. An unresolved war remains a review condition. The original Mars
reference houses, raw flags and base coverage are preserved; pair-level rules
are rebuilt after the individual cancellation. Matching computes only these
extra native inputs, leaving its core report unexpanded and unpersisted.
Unspecified high-Guna/age thresholds and ambiguous all-benefic distributions
are not guessed into additional cancellations.

## What actually changes a reading

New reports use `m4y-cross-factor-faladesh-v5` and freeze bilingual paragraphs
under `m4y-reader-paragraphs-v1`. Each paragraph retains actual evidence IDs.
The browser and PDF consume those exact strings. Purchased/old snapshots keep
their historical presentation paths; they are not overwritten by a live model.

The subject's primary owner is resolved before secondary owners. Independent
capacity, environment (aspects/conjunctions), foundation (dispositor, D9/subject
varga, own BAV) and qualified-yoga causes are kept separately. Supporting and
opposing causes produce a mixed qualification; they are not counted into an
invented accuracy or outcome percentage. BAV above/below four follows the
textbook's chapter 12, and is an additional context rather than a verdict.
Chalit differences stay context only. Actual running major/sub periods must be
valid at the retained reference instant; only directly linked subjects receive
period/transit wording. General stays four personal-direction paragraphs and
does not contain the other tabs' forecasts.

Aspectors and dispositors are qualified by their own computed strength,
combustion, uncancelled debility and war status. Their raw strength records are
retained as context, without double-counting them as another directional vote.
A weak/unknown Karak actor is not counted as fully supportive. An own/exalted
Marak dispositor does not become automatic support because of sign alone.

`m4y-subject-source-remedy-v1` ranks actual subject-linked obstacles. Primary
ownership and a valid running period affect review priority, not probability.
Without an obstacle no astrological practice is invented. A weak Karak can
receive a supportive approach; Marak/unlisted roles receive a balancing approach,
never automatic strengthening. Optional simple practices are paraphrases from
the supplied remedy pages and retain their exact page locator. Gemstones,
medical diagnoses, compulsory rituals, guaranteed relief and fabricated Guruji
approval are excluded. Genuinely approved personal rules retain their original
approval records separately.

## Primary references

- [BPHS, Santhanam volume 1](https://storage.yandexcloud.net/j108/library/hr6fkynz/Maharishi_Parashara_-_Brihat_Parasara_Hora_Sastra_%28Vol._1%29.pdf)
- [PVR textbook](https://www.vedicastrologer.org/articles/vedic_astro_textbook.pdf)
- [Phaladeepika 7.26–29, Sastri translation](https://www.wisdomlib.org/hinduism/book/phaladeepika-by-mantreswara-text-and-translation/d/doc1621579.html)
- [Brihat Samhita 17.9–10, war conventions](https://www.siva.sh/brihat-samhita/17/6-10)
- [AIFAS Horoscope Matching](https://www.futuresamachar.com/download/horoscope-matching-325.pdf)
- [Swiss Ephemeris API: elements, reference frames and apparent solar time](https://www.astro.com/swisseph/swephprg.htm)

Source PDFs are reference material, not instructions authorizing code changes,
messages, medical claims or replacement of Guruji's role classification.
