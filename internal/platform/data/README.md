# India birthplace catalog

Location data © [GeoNames](https://www.geonames.org/), licensed under
[Creative Commons Attribution 4.0](https://creativecommons.org/licenses/by/4.0/).
Source: [India gazetteer and administrative names](https://download.geonames.org/export/dump/).
GeoNames provides the data without a guarantee of accuracy or completeness.

`india.tsv.gz` is an adapted extract: populated settlements in India, excluding
abandoned, destroyed and historical features; recognized state/UT codes; validated
coordinate ranges and source timezone `Asia/Kolkata`. No missing coordinate or
timezone is guessed. Source district/state labels and alternate names are retained.
Devanagari display names are used where available; they are not machine translations.
The catalog covers 548,985 records across all 36 states and union territories.
Coverage varies by region and does not guarantee every village or street address.

`india-source.json` records input URLs, download date, SHA-256 hashes and coverage.
Rebuild with `python scripts/import_places.py`; remove or replace cached source
files under `.local/geonames` first when intentionally updating the snapshot.
The importer uses the Python standard library only. The compressed output is
embedded in the Go executable; location searches need no external service/API key.

TSV columns: GeoNames ID, ASCII place name, available Devanagari name, state/UT,
district (when present), latitude, longitude, population, pipe-separated aliases.
Results rank exact names, then prefixes, then other partial matches, with source
population as tiebreaker. Queries can include district/state to narrow results.
The original 20 IDs/coordinates remain valid; corresponding GeoNames entries are
deduplicated into their existing IDs and gain source alternate names.
