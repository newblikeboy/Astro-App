"""Build the bundled India catalog from GeoNames (CC BY 4.0).

Run: python scripts/import_places.py
Downloaded inputs are cached under .local/geonames; runtime needs no API key.
"""

import csv
import datetime
import gzip
import hashlib
import io
import json
from pathlib import Path
import urllib.request
import zipfile

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / ".local/geonames"
OUT = ROOT / "internal/platform/data"
BASE = "https://download.geonames.org/export/dump/"
CACHE.mkdir(parents=True, exist_ok=True)
OUT.mkdir(parents=True, exist_ok=True)
FILES = ("IN.zip", "admin1CodesASCII.txt", "admin2Codes.txt")
for name in FILES:
    if not (CACHE / name).exists():
        urllib.request.urlretrieve(BASE + name, CACHE / name)


def admin_names(filename):
    """Load GeoNames administrative-code labels for enriching trusted birthplace records."""
    return {
        row[0]: row[2]
        for row in csv.reader((CACHE / filename).open(encoding="utf-8"), delimiter="\t")
        if row[0].startswith("IN.")
    }


states = admin_names("admin1CodesASCII.txt")
districts = admin_names("admin2Codes.txt")
rows = []
coverage = {}
with zipfile.ZipFile(CACHE / "IN.zip") as archive:
    with io.TextIOWrapper(archive.open("IN.txt"), encoding="utf-8") as source:
        for r in csv.reader(source, delimiter="\t", quoting=csv.QUOTE_NONE):
            # Exclude abandoned/destroyed/historical settlements and ambiguous
            # administrative or timezone records. Never invent coordinates/zones.
            state = states.get("IN." + r[10])
            if (r[6] != "P" or r[7] in {"PPLQ", "PPLW", "PPLH"}
                    or r[8] != "IN" or r[17] != "Asia/Kolkata" or not state):
                continue
            latitude, longitude = float(r[4]), float(r[5])
            if not (-66 <= latitude <= 66 and -180 <= longitude <= 180):
                continue
            names = list(dict.fromkeys([r[1], r[2], *r[3].split(",")]))
            local = next((n for n in names if any("\u0900" <= c <= "\u097f" for c in n)), "")
            district = districts.get("IN." + r[10] + "." + r[11], "")
            rows.append([r[0], r[2] or r[1], local, state, district,
                         r[4], r[5], r[14], "|".join(n for n in names if n)])
            coverage[state] = coverage.get(state, 0) + 1

# Population ordering is also the deterministic tiebreaker for search results.
rows.sort(key=lambda r: (-int(r[7]), r[1], int(r[0])))
with (OUT / "india.tsv.gz").open("wb") as raw:
    with gzip.GzipFile(fileobj=raw, mode="wb", filename="", mtime=0) as compressed:
        with io.TextIOWrapper(compressed, encoding="utf-8", newline="") as text:
            csv.writer(text, delimiter="\t", lineterminator="\n").writerows(rows)
metadata = {
    "source": "GeoNames", "license": "CC BY 4.0",
    "license_url": "https://creativecommons.org/licenses/by/4.0/",
    "retrieved_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
    "places": len(rows), "state_and_union_territory_counts": dict(sorted(coverage.items())),
    "inputs": {name: {"url": BASE + name, "sha256": hashlib.sha256((CACHE / name).read_bytes()).hexdigest()}
               for name in FILES},
    "output_sha256": hashlib.sha256((OUT / "india.tsv.gz").read_bytes()).hexdigest(),
}
(OUT / "india-source.json").write_text(json.dumps(metadata, indent=2) + "\n", encoding="utf-8")
print(f"Built {len(rows):,} places across {len(coverage)} states/union territories.")
