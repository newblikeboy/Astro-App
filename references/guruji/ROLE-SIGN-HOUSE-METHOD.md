# Board role, sign condition and D1 house

Classification version: `m4y-board-role-sign-v1`; synthesis version:
`m4y-cross-factor-faladesh-v4` (role/sign rules retained from v3).

The user's supplied whiteboard is authoritative for this feature's twelve
ascendant role lists and exaltation/debilitation sign numbers. The existing
[transcription](../../internal/platform/data/guruji-lagna-roles.json) is retained
unchanged. No alternate classification from the historical shastra comparison
is substituted. Unlisted planets remain unlisted, not implicitly neutral.

Each newly calculated planet reading retains these distinct values:

- `role`: board Karak, Marak, neutral, or unlisted.
- `sign_number`: the native D1 sidereal sign, 1 through 12.
- `sign_status`: primary exalted/debilitated/own/other label and independent
  `exalted`, `debilitated`, `own_sign` booleans. Mercury in Virgo has both
  exalted and own-sign flags, with exalted as its primary display status.
- `house`: the existing native D1 whole-sign house, never the Chalit house.
- `role_effect`: the bounded capacity interpretation and its direction.

Classical sign ownership is Sun/Leo, Moon/Cancer, Mars/Aries and Scorpio,
Mercury/Gemini and Virgo, Jupiter/Sagittarius and Pisces, Venus/Taurus and
Libra, Saturn/Capricorn and Aquarius. Nodes acquire no inferred own sign or
ascendant role; their exaltation/debilitation follows the board's supplied
two-sign convention.

## Interpretation boundary

Karak plus exalted strengthens favourable capacity; Karak plus own sign
supports that capacity; Karak plus debilitated qualifies it as potentially
weak or obstructed. Neither role nor sign alone promises an event.

Marak remains Marak in every sign. Exaltation and ownership do not make it
Karak or automatically support a favourable conclusion. Marak plus debilitation
does not by itself prove a bad outcome either. Neutral remains neutral.
These combinations receive contextual explanations rather than invented
positive/negative scores. The same natal board-role qualification applies to
existing divisional sign evidence so a D9 sign cannot bypass it.

This feature does not calculate or claim conjunction, aspect, Neechabhanga,
Dasha or event timing. Existing independently calculated factors elsewhere
in the synthesis remain separately identified; their presence is not inferred
from this classification. Astronomical positions and native house calculations
are unchanged.

## Independent verification examples

| Lagna | Planet/sign | D1 house | Board role | Primary status | Own sign |
| --- | --- | --- | --- | --- | --- |
| Gemini (3) | Mercury/Virgo (6) | 4 | Karak | Exalted | true |
| Libra (7) | Mercury/Virgo (6) | 12 | Marak | Exalted | true |
| Gemini (3) | Mercury/Gemini (3) | 1 | Karak | Own sign | true |
| Gemini (3) | Mercury/Pisces (12) | 10 | Karak | Debilitated | false |

Tests additionally move Pisces/Virgo Mercury through all twelve ascendants,
exercise actual synthesis and divisional evidence, check Chalit separation,
and compare native chart serialization before and after classification.

Only newly calculated reports receive these fields. Historical saved reports
and purchased snapshots remain unchanged; no render-time reconstruction fills
missing flags. Superseded unsaved guest v1/v2 readings can be recalculated
through the existing public computation endpoint without saving a report.
