# Copyright, licensing and source notices

The original Mishra4You application code is licensed under GNU AGPL version 3
only (`AGPL-3.0-only`); see [LICENSE](LICENSE). Existing copyright ownership and
third-party notices remain intact. This declaration does not replace the
licences of independently licensed dependencies, fonts, data or media.

The software is provided without warranty, subject to the terms in LICENSE.
Recipients may copy, modify and redistribute the covered application under
those terms, including for commercial purposes.

## Swiss Ephemeris

Copyright (C) 1997–2021 Astrodienst AG, Switzerland. The original notice is
preserved verbatim in [internal/swisseph/LICENSE](internal/swisseph/LICENSE)
and in the bundled source files. This application uses Swiss Ephemeris under
its AGPL option. No Professional licence purchase is asserted.

The Go integration and application-specific calculation modules are additions
to the upstream Swiss Ephemeris C sources; their changes and dates are recorded
in this repository's Git history. This licensing/source-access update is dated
4 November 2026. Upstream C source notices have not been replaced.

Upstream source: <https://github.com/aloistr/swisseph>.
Licensing terms: <https://www.astro.com/swisseph/swephinfo_e.htm>.

## Fonts, location data and media

- Noto Sans Devanagari: retain [assets/OFL-NotoSansDevanagari.txt](assets/OFL-NotoSansDevanagari.txt).
- DM Sans: retain [web/public/fonts/DMSans-LICENSE.txt](web/public/fonts/DMSans-LICENSE.txt).
- Instrument Serif: retain [web/public/fonts/InstrumentSerif-LICENSE.txt](web/public/fonts/InstrumentSerif-LICENSE.txt).
- The bundled India location catalogue is derived from GeoNames, copyright its
  contributors, under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).
  Attribution and provenance are in
  [internal/platform/data/README.md](internal/platform/data/README.md) and
  [india-source.json](internal/platform/data/india-source.json).
- Devotional image attribution and adaptation records are preserved in
  [CREDITS.md](web/public/images/devotional/CREDITS.md) and
  [ARTWORK.md](web/public/images/devotional/ARTWORK.md).
- Existing media and reference attribution/source records remain applicable.
  Inclusion in this repository does not transfer a third party's copyright or
  create permissions beyond its existing terms.

## Dependencies and corresponding source

Go dependencies and exact versions are recorded in go.mod/go.sum; frontend
dependencies and exact versions are recorded in web/package.json and
web/package-lock.json. Preserve their upstream copyright and licence files
when redistributing them. This notice is not a replacement for those files.

The public source repository is <https://github.com/newblikeboy/Astro-App>.
Each frontend build's footer links to the Git commit used for that build,
including this notice and LICENSE at that revision. The source offer is
available without an account or charge. [DEPLOYMENT.md](DEPLOYMENT.md) describes
building, installing and running it, including provider request contracts.
Operators must publish changes before deploying and keep the corresponding
commit available. Customer data, credentials and provider accounts are not
part of this source offer.
