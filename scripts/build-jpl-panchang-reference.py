"""Offline audit only: independent NASA/JPL apparent-geocentric phase fixtures.

Horizons observer quantity 31 is apparent IAU76/80 ecliptic-of-date longitude.
Taking Moon minus Sun cancels the common equinox origin/ayanamsa. Linear
interpolation of two-minute samples is independent of the Go boundary solver.
This script never loads application config or contacts a customer database.
Reference API/method: https://ssd-api.jpl.nasa.gov/doc/horizons.html
https://ssd.jpl.nasa.gov/horizons/manual.html#obsquan
"""
import csv
import datetime as dt
import json
from pathlib import Path
from urllib.parse import urlencode
from urllib.request import urlopen

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / ".local/jpl-critical-audit"
CACHE.mkdir(parents=True, exist_ok=True)
CASES = [
    ("2026-10-18", 28.6139, 77.209),
    ("2026-10-19", 28.6139, 77.209),
    ("2026-10-20", 28.6139, 77.209),
    ("2026-11-08", 19.076, 72.8777),
    ("2026-03-19", 25.3176, 82.9739),
    ("2023-07-20", 28.6139, 77.209),
    ("1972-01-02", 13.0827, 80.2707),
]


def positions(date, body):
    """Fetch cached NASA apparent Sun/Moon samples for an independent Panchang phase reference."""
    start = dt.datetime.fromisoformat(date)
    params = {
        "format": "json", "COMMAND": f"'{body}'", "CENTER": "'500@399'",
        "EPHEM_TYPE": "'OBSERVER'", "START_TIME": f"'{date} 00:00'",
        "STOP_TIME": f"'{start + dt.timedelta(days=3):%Y-%m-%d} 00:00'",
        "STEP_SIZE": "'2 m'", "QUANTITIES": "'31'", "CSV_FORMAT": "'YES'",
        "TIME_TYPE": "'UT'", "CAL_TYPE": "'GREGORIAN'", "ANG_FORMAT": "'DEG'",
        "EXTRA_PREC": "'YES'", "OBJ_DATA": "'NO'",
    }
    url = "https://ssd.jpl.nasa.gov/api/horizons.api?" + urlencode(params)
    file = CACHE / f"{date}-{body}.json"
    if file.exists():
        record = json.loads(file.read_text(encoding="utf-8"))
        if record["url"] != url:
            raise ValueError("cached reference query mismatch")
    else:
        response = json.loads(urlopen(url, timeout=60).read())
        if "error" in response or "$$SOE" not in response.get("result", ""):
            raise ValueError(response)
        record = {"url": url, "response": response}
        file.write_text(json.dumps(record, indent=2), encoding="utf-8")
    raw = record["response"]["result"]
    rows = []
    for row in csv.reader(raw.split("$$SOE")[1].split("$$EOE")[0].strip().splitlines()):
        stamp = dt.datetime.strptime(row[0].strip(), "%Y-%b-%d %H:%M").replace(tzinfo=dt.timezone.utc)
        rows.append((stamp, float(row[3])))
    assert len(rows) == 2161, len(rows)
    return rows


fixtures = []
for date, lat, lon in CASES:
    # Requests are sequential, in accordance with the public API usage policy.
    sun, moon = positions(date, 10), positions(date, 301)
    phases = [(m[1] - s[1]) % 360 for s, m in zip(sun, moon)]
    boundaries = {"tithi": [], "karana": []}
    for name, width in (("tithi", 12), ("karana", 6)):
        for i in range(1, len(phases)):
            a = phases[i - 1]
            travel = (phases[i] - a) % 360
            assert 0 < travel < 1, travel
            target = (int(a / width) + 1) * width
            if a + travel >= target:
                at = sun[i - 1][0] + (sun[i][0] - sun[i - 1][0]) * ((target - a) / travel)
                boundaries[name].append({"number": int(a / width) + 1, "end": at.isoformat()})
    fixtures.append({"date": date, "latitude": lat, "longitude": lon, **boundaries})
    print(f"JPL reference collected: {date}", flush=True)

out = {
    "source": "NASA/JPL Horizons DE441, observer quantity 31, Earth geocentre; apparent Moon-Sun longitude; UTC output; two-minute interpolation",
    "api": "https://ssd-api.jpl.nasa.gov/doc/horizons.html",
    "method": "https://ssd.jpl.nasa.gov/horizons/manual.html#obsquan",
    "tolerance_seconds": 2,
    "cases": fixtures,
}
target = ROOT / "internal/swisseph/testdata/jpl-panchang-reference.json"
target.write_text(json.dumps(out, indent=2) + "\n", encoding="utf-8")
print(f"Wrote {len(fixtures)} independent JPL cases", flush=True)
