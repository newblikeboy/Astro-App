# Guruji reference and Faladesh review — 1 November 2026

Historical review: the user subsequently designated the board as authoritative
for this feature. No comparison below replaces its classifications. Software
gap observations below predate the current
[role/sign/house implementation](ROLE-SIGN-HOUSE-METHOD.md).

## Finding

The supplied PDFs are a useful teaching/reference library, but do not specify a
complete personal prediction engine. The whiteboard needs clarification before
its categories can be described as a verified, universal Parashari table. This
reviews the **supplied notes and our implementation**, not the extent of
Guruji's knowledge. A short classroom board may depend on explanations absent
from the photograph. No traditional consistency check establishes scientific
predictive validity.

## Primary texts inspected

- [Brihat Parashara Hora Shastra, R. Santhanam, volume 1](https://storage.yandexcloud.net/j108/library/hr6fkynz/Maharishi_Parashara_-_Brihat_Parasara_Hora_Sastra_%28Vol._1%29.pdf): chapter 34, verses 19–44, printed pp. 347–356 (scan pages 346–355). The Sanskrit and printed translation for Gemini, Virgo/Libra and Pisces were visually checked. Translator commentary is distinguished from the root verses.
- [BPHS chapter 3, Sanskrit with translation](https://parashara.net/index.php?page=chapter3): verses 49–58, dignity and relationships.
- [BPHS transcription](https://vedic-astro.s3.amazonaws.com/books/bhrihat_parasara_hora_shastra.pdf): chapter 34 (PDF pp. 52–53), chapter 44 (pp. 67–68), and chapter 47, verses 34–39 (p. 88). Useful for navigation; the scan above is preferred for checking chapter 34.
- [Phaladeepika, chapter 15](https://www.wisdomlib.org/hinduism/book/phaladeepika-by-mantreswara-text-and-translation/d/doc1621587.html): verses 1–11; the house, its lord, its significator, strength and aspects belong in a combined assessment. The online transcription warns of OCR errors; it is not a critical edition.
- [Brihat Jataka, chapter II, Aiyar translation](https://chestofbooks.com/new-age/astrology/Brihat-Jataka/Definitions-And-Elementary-Planetary-Principles-Continued.html): verses 19–21 and accompanying notes, multiple kinds of strength.
- [Brihat Jataka, chapter XX](https://chestofbooks.com/new-age/astrology/Brihat-Jataka/Chapter-XX-On-The-Planets-In-The-Bhavas.html): verses 10–11, qualification of house-placement readings. This edition itself records variant readings.
- [Phaladeepika, chapter XIX](https://www.wisdomlib.org/hinduism/book/phaladeepika-by-mantreswara-text-and-translation/d/doc1621591.html) and [chapter XXVI](https://www.wisdomlib.org/hinduism/book/phaladeepika-by-mantreswara-text-and-translation/d/doc1621598.html): period interpretation and separate transit rules.

## Comparison of all twelve whiteboard rows

These are **review points**, not automatic substitutions. The compared categories
are not perfectly equivalent: the board's broad Karak/Marak lists must not be
mistaken for separate classical benefic, yogakaraka and maraka classifications.
Verse numbers below refer to BPHS chapter 34 in the inspected edition.

| Lagna | Supplied note requiring review | Compared text, in brief | Verses |
| --- | --- | --- | --- |
| Aries | Saturn neutral | Saturn adverse | 19–22 |
| Taurus | Venus supportive; Sun neutral | Venus adverse; Sun auspicious | 23–24 |
| Gemini | Jupiter supportive | Jupiter adverse | 25–26 |
| Cancer | Venus neutral | Venus adverse | 27–28 |
| Leo | Mercury supportive; Venus neutral | Both adverse | 29–30 |
| Virgo | Jupiter supportive | Jupiter adverse | 31–32 |
| Libra | Mercury Marak; Venus supportive | Mercury auspicious; Venus neutral | 33–34 |
| Scorpio | Saturn neutral; Mars supportive | Saturn adverse; Mars neutral | 35–36 |
| Sagittarius | Jupiter supportive | Jupiter neutral | 37–38 |
| Capricorn | Mars neutral; Sun Marak | Mars adverse; Sun neutral | 39–40 |
| Aquarius | Jupiter supportive; Mars neutral | Both adverse | 41–42 |
| Pisces | Mercury supportive | Mercury adverse and maraka | 43–44 |

This is not an exhaustive list of qualifications within each row. In particular,
an auspicious planet can also possess a maraka qualification. A one-category
lookup cannot represent that overlap. One translation's Taurus-Mercury wording
also differs from the Sanskrit's limited-benefit wording; that entry needs
edition-level discussion, not a confident automated correction.

**Transcription correction:** the first app import missed Mercury in the Pisces
supportive list because the handwritten line extends left of the main Pisces
column. A wider enlarged inspection confirms it is present. Source version v2
corrects our transcription for new reports. This omission was our error, not
missing knowledge in the supplied board. The doctrinal comparison above uses
the corrected reading.

### What aligns

The seven visible classical-planet exaltation/debilitation sign pairs on the
board align with BPHS 3.49–50. Basic house subjects and the longer PDF's repeated
instruction to consider other factors are compatible with using the material
as conditional guidance. The longer PDF also sensibly distinguishes general
planetary remedies from individual gemstone recommendations.

### Rahu/Ketu require a named convention

The board gives Rahu exaltation in signs 2/3 and Ketu in 8/9. The compared BPHS
Dasha passage separates Taurus/Scorpio exaltation from Gemini/Sagittarius
moolatrikona. Therefore the board combines labels that this edition separates.
Retain the teacher's convention as a named source convention until clarified;
do not describe both signs as an uncontested classical exaltation rule.

### Naming image

The image is a short rashi-letter reference, not a nakshatra/pada calculation.
Some rows are visibly incomplete or compressed (for example, the Gemini row
does not list nine clearly separate syllables). It should not replace the
existing birth-star/pada-derived syllable calculation.

## Gaps in our software — separate from the teacher's material

1. **Category conflation.** `gurujiIndication` reduces the source role and sign
   dignity to one emphasis label. That is our heuristic, not a cited shloka.
   Its previous prominent green/amber badges looked like final personal
   conclusions. Stop using them as a verdict in the consultation presentation.
   Keep the source-role and dignity facts separate in the explanation.
2. **Occupancy-only topic selection.** A career or relationship tab previously
   selected only planets occupying a short list of houses. Empty houses still
   have lords. Include factual house-lord connections in the reading selection,
   label why each planet is shown, and do not invent a combined judgement.
3. **Strength is calculated in parts, but not synthesised.** Sign dignity,
   combustion, aspects, friendship and varga-related evidence exist elsewhere
   in the report. The previous Faladesh engine did not combine them. Full
   Shadbala/Bhavbala is still absent; sign dignity is not a substitute.
4. **Missing rule priority.** There is no supplied rule for balancing lordship,
   dignity, combustion, aspects, associations, dispositor condition and
   cancellation. A simple positive/negative score would hide this gap.
5. **Periods are not interpreted.** Dasha dates are calculated, but selecting
   the current Mahadasha planet's natal paragraph is not a Dasha/Antardasha
   forecast. Transit placement also needs its own reference frame and rules.
6. **Varga confirmation is absent from this synthesis.** The app calculates
   vargas; it does not yet apply an approved cross-varga decision tree to each
   life-topic conclusion. No claim of such confirmation should be made.
7. **Generic polarity passages lack application conditions.** The short PDF's
   "this period" passages cannot be assigned from occupancy or a running planet
   alone. Keep both branches as original references until their trigger is
   specified.
8. **Remedy prescription is incomplete.** Lists of worship, donation, mantra
   and gemstone associations do not determine whether a particular native
   should strengthen, propitiate or avoid a planet. Do not attach nine personal
   remedies automatically or drive gemstone purchases from these categories.
9. **No grantha-to-sentence traceability.** The supplied 108 modern explanatory
   passages have PDF page provenance, not individual shloka provenance. New
   internet findings must stay separately attributed; do not rebrand them as
   Guruji's approved words.
10. **Presentation was a source browser.** Repeated technical preambles, long
    planet lists, nested full-text disclosures and mixed-language paragraphs
    obscured what the person should understand. Actual concise readings need
    to come first; source prose belongs behind "Why this reading".

## Consultation presentation

Use a single coherent reading desk within Faladesh:

1. **Personal saar:** the Lagna-lord reading, Moon pattern and a dated Dasha
   context, using the actual saved chart. Avoid a fake overall life score.
2. **Life subjects:** personality, learning, work, relationships, family and
   wellbeing. Each has a short lead and a clear next step to explore it.
3. **A balanced reading:** concise source-based meaning, helpful possibilities
   and points to watch, shown together. Do not equate a teacher's Marak label
   with all difficulties in that subject.
4. **Why this reading:** actual planet/sign/house, ownership of the topic's
   house, recorded dignity and combustion when available, then source pages.
   A missing assessment remains missing; no numerical confidence is invented.
5. **Practical reflection, then remedies:** one relevant general practice,
   followed by optional original mantra/remedy references. Approved personal
   instructions retain their existing separate status.
6. **Original sources:** available on demand, without reproducing the entire
   reference library in the default screen. A dated period context is distinct
   from lifelong natal tendencies.

New source-based reading summaries must be snapshotted with their source locator
and version. Old purchased snapshots must not be rewritten. This UI improvement
does not close the interpretation-engine gaps listed above.

## Questions to resolve with Guruji

- Are the board's Marak entries shorthand for adverse planets, formal marakas,
  or a period-specific teaching classification?
- Which text/edition or parampara explains the differing Jupiter/Mercury rows?
- Are the second node signs intended as moolatrikona, rather than exaltation?
- Which factor takes priority when role, dignity, combustion and aspects disagree?
- What exact conditions select each short-PDF positive/negative passage?
- What additional conditions approve a personal mantra or gemstone instruction?

These questions are recorded for review; no message has been sent to Guruji.

## Implemented and verified

The local app now follows the consultation presentation described above. New
reports retain all 108 source-based bilingual summaries as a versioned optional
`narrative` for each matched planet/house; original Hindi helpful/caution
sentences retain their conditions. Older reports keep their saved prose.
Prominent heuristic outcome badges have been removed, while the teacher's
original role labels remain available as source notes.

Verification: Go suite and targeted narrative/source tests, Go vet, production
frontend build, documentation checks, English/Hindi layouts at 320/390/768/1440,
native keyboard disclosures and accessibility scans passed. The empty tenth
house fixture includes its Mercury lord in house twelve. The main Atlas-backed
local API generated nine v2 readings with concise narratives; it did not create
an approval. Updated bilingual PDF reference pages were rendered and inspected.
