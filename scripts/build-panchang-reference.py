"""Offline QA only. Requires the official Swiss swetest executable.

Independent reference: sample CLI geocentric Lahiri Sun/Moon every minute,
unwrap cyclic angles, linearly interpolate crossings. This does not call the
Go Panchang solver. Solar times are swetest -rise output. Output timestamps
are UT1 labelled Z for fixture parsing; comparison allows 1 second for UTC/UT1
and the CLI's 0.1-second solar output precision. No runtime data scraping.
"""
import datetime as dt
import json
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parents[1]
CLI = ROOT / ".local/swetest.exe"

def run(args):
    """Run the official Swiss CLI against the bundled ephemeris when building offline fixtures."""
    return subprocess.check_output([str(CLI), *args, "-edir" + str(ROOT / "ephe")], text=True, cwd=ROOT)

cases = [
    ("2026-10-18", 28.6139, 77.209, "upper-limb"),
    ("2026-10-18", 28.6139, 77.209, "hindu"),
    ("2026-11-08", 19.076, 72.8777, "upper-limb"),
    ("2026-03-19", 25.3176, 82.9739, "upper-limb"),
    ("1972-01-02", 13.0827, 80.2707, "upper-limb"),
    ("2099-12-30", 27.5, 95, "upper-limb"),
]
fixtures = []
for date, lat, lon, convention in cases:
    start = dt.datetime.fromisoformat(date).replace(tzinfo=dt.timezone.utc)
    european = start.strftime("%d.%m.%Y")
    args = ["-b" + european, "-ut0", "-p0", "-rise", f"-geopos{lon},{lat},0", "-at1013.25,15", "-n3"]
    if convention == "hindu": args += ["-hindu"]
    solar = run(args)
    events = re.findall(r"(?:rise|set)\s+(\d+\.\d+\.\d+)\s+(\d+:\d+:\d+\.\d+)", solar)
    assert len(events) >= 3, solar
    instants = [dt.datetime.strptime(d + " " + t, "%d.%m.%Y %H:%M:%S.%f").replace(tzinfo=dt.timezone.utc) for d,t in events]
    rise, sunset, next_rise = instants[:3]
    # 72 hours ensure an actual end even for a limb spanning next sunrise.
    output = run(["-b" + european, "-ut0", "-p01", "-sid1", "-fl", "-head", "-n4321", "-s0.0006944444444444444"])
    values = [float(line.strip()) for line in output.splitlines() if re.fullmatch(r"\s*[0-9]+\.[0-9]+\s*",line)]
    assert len(values) == 8642, len(values)
    samples = []
    for i in range(0,len(values),2):
        sun,moon = values[i:i+2]
        samples.append([(moon-sun)%360, moon, (moon+sun)%360, (moon-sun)%360])
    result = {"date":date,"latitude":lat,"longitude":lon,"convention":convention,"sunrise":rise.isoformat(),"sunset":sunset.isoformat(),"next_sunrise":next_rise.isoformat(),"limbs":[]}
    for kind,width in enumerate([12,360/27,360/27,6]):
        transitions = []
        for i in range(1,len(samples)):
            a,b = samples[i-1][kind], samples[i][kind]
            b = a + (b-a)%360
            boundary = (int(a/width)+1)*width
            if b >= boundary:
                instant = start + dt.timedelta(minutes=i-1+(boundary-a)/(b-a))
                if instant <= rise: continue
                transitions.append({"number":int(a/width)+1,"end":instant.isoformat()})
                if instant >= next_rise: break
        assert transitions
        result["limbs"].append(transitions)
    fixtures.append(result)
target = ROOT / "internal/swisseph/testdata/panchang-reference.json"
target.write_text(json.dumps({"source":"Swiss Ephemeris 2.10.03 official swetest; one-minute longitude samples with linear crossing interpolation; UT1 timestamps", "cases":fixtures},indent=2)+"\n",encoding="utf-8")
print(f"Wrote {len(fixtures)} independent CLI reference cases to {target.relative_to(ROOT)}")
