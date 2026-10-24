"""Offline wrapper reference for the 1940 date-range extension.

Uses the official Swiss swetest CLI, not the application's Go entry points.
Before 1972 both use Swiss's documented UT1 clock convention. This verifies
the wrapper/settings, not an independent astronomical model.
"""
import datetime as dt
import json
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[1]
CASES = [
    ("1939-12-31T18:30:00+00:00", 28.6139, 77.209),
    ("1940-01-01T00:00:00+00:00", 28.6139, 77.209),
    ("1940-02-29T06:00:00+00:00", 22.5726, 88.3639),
    ("1960-07-15T12:34:00+00:00", 19.076, 72.8777),
]
results = []
for stamp, lat, lon in CASES:
    instant = dt.datetime.fromisoformat(stamp)
    args = [str(ROOT / ".local/swetest.exe"), "-b" + instant.strftime("%d.%m.%Y"),
            "-ut" + instant.strftime("%H:%M:%S"), "-p0123456m", "-sid1", "-eswe",
            "-edir" + str(ROOT / "ephe"), "-fPls", "-g,", "-head",
            f"-house{lon},{lat},W"]
    raw = subprocess.check_output(args, text=True, cwd=ROOT)
    rows = {}
    for line in raw.splitlines():
        cells = [c.strip() for c in line.split(",")]
        if len(cells) == 3:
            try:
                rows[cells[0]] = (float(cells[1]), float(cells[2]))
            except ValueError:
                pass
    names = ["Sun", "Moon", "Mars", "Mercury", "Jupiter", "Venus", "Saturn", "mean Node"]
    assert all(name in rows for name in names) and "Ascendant" in rows, raw
    results.append({"at": stamp, "latitude": lat, "longitude": lon,
                    "lagna": rows["Ascendant"][0],
                    "planets": [{"name": "Rahu" if name == "mean Node" else name,
                                 "longitude": rows[name][0], "speed": rows[name][1]} for name in names]})
target = ROOT / "internal/swisseph/testdata/historical-reference.json"
target.write_text(json.dumps({"source": "Official Swiss 2.10.03 swetest CLI, UT1 clock input, geocentric Lahiri, mean node, whole-sign houses", "tolerance_degrees": 1e-6, "cases": results}, indent=2) + "\n", encoding="utf-8")
print(f"Wrote {len(results)} historical CLI fixtures.")
