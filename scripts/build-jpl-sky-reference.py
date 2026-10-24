"""Build offline NASA/JPL sky fixtures without reading application configuration.

Horizons supplies Earth-centred apparent tropical longitudes (quantity 31).
The only Swiss contribution to the reference is the explicitly shared Lahiri
coordinate origin, not planetary longitudes or event times. NASA does not
define an astrological ayanamsa. Default Swiss and Horizons ecliptic-of-date
frames differ slightly; the fixture declares a bounded comparison tolerance.

Annual roots are found from NASA hourly samples, independently of the Go
solver and without seeding searches with application transit times.
Requests are sequential. Raw responses and URLs are cached for provenance.
"""

import argparse
import csv
import datetime as dt
import hashlib
import json
import math
from pathlib import Path
import subprocess
from urllib.parse import urlencode
from urllib.request import urlopen

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / ".local/jpl-planet-audit"
UTC = dt.timezone.utc
IST = dt.timezone(dt.timedelta(hours=5, minutes=30))
BODIES = {"Sun": 10, "Moon": 301, "Mars": 4, "Mercury": 199,
          "Jupiter": 5, "Venus": 299, "Saturn": 6}
# Swiss defaults use planetary-system barycentres (not the outer planet's
# physical centre). The corresponding Horizons 4/5/6 targets match that choice.
CLI_IDS = "0142536"
EPOCHS = ["1995-08-15T05:00:00+00:00", "2000-01-01T12:00:00+00:00",
          "2026-10-18T06:30:00+00:00", "2026-10-19T06:30:00+00:00",
          "2026-10-20T06:30:00+00:00", "2026-10-20T00:54:37+00:00"]
YEAR_START = dt.datetime(2026, 1, 1, tzinfo=IST).astimezone(UTC)
YEAR_END = dt.datetime(2027, 1, 1, tzinfo=IST).astimezone(UTC)
START = YEAR_START - dt.timedelta(hours=2)
STOP = YEAR_END + dt.timedelta(hours=2)


def request(name, body, extra):
    """Fetch and validate a cached NASA Horizons response, retaining its URL, target and content hash."""
    params = {"format": "json", "COMMAND": f"'{body}'", "CENTER": "'500@399'",
              "EPHEM_TYPE": "'OBSERVER'", "QUANTITIES": "'31'",
              "CSV_FORMAT": "'YES'", "TIME_TYPE": "'UT'",
              "CAL_TYPE": "'GREGORIAN'", "CAL_FORMAT": "'BOTH'",
              "TIME_DIGITS": "'FRACSEC'", "ANG_FORMAT": "'DEG'",
              "EXTRA_PREC": "'YES'", "OBJ_DATA": "'NO'", **extra}
    url = "https://ssd.jpl.nasa.gov/api/horizons.api?" + urlencode(params)
    file = CACHE / f"{name}.json"
    if file.exists():
        record = json.loads(file.read_text(encoding="utf-8"))
        if record["url"] != url:
            raise ValueError(f"cached query mismatch: {file}")
    else:
        response = json.loads(urlopen(url, timeout=55).read())
        if "error" in response or "$$SOE" not in response.get("result", ""):
            raise ValueError(response)
        record = {"url": url, "retrieved_utc": dt.datetime.now(UTC).isoformat(),
                  "response": response}
        file.write_text(json.dumps(record, indent=2), encoding="utf-8")
    raw = record["response"]["result"]
    if "GEOCENTRIC" not in raw or "{source: DE441}" not in raw:
        raise ValueError(f"unexpected reference centre/ephemeris: {file}")
    rows = []
    for row in csv.reader(raw.split("$$SOE")[1].split("$$EOE")[0].strip().splitlines()):
        # CAL_FORMAT=BOTH: calendar, JD, solar-presence, lunar-presence, lon, lat.
        stamp = None
        for fmt in ("%Y-%b-%d %H:%M:%S.%f", "%Y-%b-%d %H:%M:%S", "%Y-%b-%d %H:%M"):
            try:
                stamp = dt.datetime.strptime(row[0].strip(), fmt).replace(tzinfo=UTC)
                break
            except ValueError:
                pass
        if stamp is None:
            raise ValueError(f"unrecognized Horizons timestamp: {row[0]}")
        # Horizons may abbreviate the calendar column even with FRACSEC.
        # The JDUT column retains the requested sub-minute instant.
        stamp = dt.datetime.fromtimestamp((float(row[1])-2440587.5)*86400, UTC)
        rows.append((stamp, float(row[4]), float(row[5])))
    return rows, {"url": url, "sha256": hashlib.sha256(raw.encode()).hexdigest(),
                  "signature": record["response"].get("signature"),
                  "target": next(l.strip() for l in raw.splitlines() if l.startswith("Target body name:"))}


def ayanamsas(cli, start, count=1, step_minutes=60):
    """Read only the declared Lahiri coordinate-origin series from the official Swiss CLI."""
    # -ay1 prints swe_get_ayanamsa_ex's Lahiri origin including nutation.
    # Mean-only metadata is not subtracted from apparent longitudes.
    args = [str(cli), "-b" + start.strftime("%d.%m.%Y"),
            "-utc" + start.strftime("%H:%M:%S"), f"-n{count}",
            f"-s{step_minutes/1440:.17g}", "-ay1", "-eswe", "-edirephe",
            "-fJl", "-g,", "-head"]
    result = subprocess.run(args, cwd=ROOT, capture_output=True, text=True, check=True)
    values = [float(row[-1]) for row in csv.reader(result.stdout.splitlines()) if len(row) == 2]
    if len(values) != count or not all(20 < x < 30 for x in values):
        raise ValueError(f"invalid Lahiri origin series: {result.stderr[:300]}")
    return values


def unwrap(values):
    """Make consecutive angular samples continuous across the zero-degree wrap."""
    out = [values[0]]
    for value in values[1:]:
        out.append(out[-1] + (value - out[-1] + 180) % 360 - 180)
    return out


def cubic(values, i):
    """Build a four-sample Lagrange polynomial over the current hourly search segment."""
    # Lagrange polynomial at hourly nodes -1,0,1,2 around segment i..i+1.
    a, b, c, d = values[i-1:i+3]
    return b, (-2*a-3*b+6*c-d)/6, (a-2*b+c)/2, (-a+3*b-3*c+d)/6


def value(c, x):
    """Evaluate the interpolated longitude polynomial at a fractional sample step."""
    return ((c[3]*x+c[2])*x+c[1])*x+c[0]


def derivative(c, x):
    """Evaluate the longitude polynomial's derivative to locate motion reversals."""
    return (3*c[3]*x+2*c[2])*x+c[1]


def bisect(function, lo, hi):
    """Refine a bracketed continuous root without using the production Go search implementation."""
    left = function(lo)
    if left * function(hi) > 0:
        raise ValueError("root not bracketed")
    for _ in range(45):
        mid = (lo+hi)/2
        if (function(mid) < 0) == (left < 0):
            lo = mid
        else:
            hi = mid
    return (lo+hi)/2


def annual_events(planet, rows, origins):
    """Find annual sign ingresses and motion stations from the independent NASA longitude series."""
    sidereal = unwrap([(row[1]-origin) % 360 for row, origin in zip(rows, origins)])
    events = []
    for i in range(1, len(rows)-2):
        c = cubic(sidereal, i)
        points = [0.0, 1.0]
        # Split a segment at every internal stationary point, including a
        # grazing direct/retrograde pair even when endpoint signs are equal.
        aa, bb, cc = 3*c[3], 2*c[2], c[1]
        if abs(aa) < 1e-14:
            roots = [-cc/bb] if abs(bb) > 1e-14 else []
        else:
            disc = bb*bb-4*aa*cc
            roots = [(-bb-math.sqrt(disc))/(2*aa), (-bb+math.sqrt(disc))/(2*aa)] if disc >= 0 else []
        # Rounded Horizons angles can place a station microscopically outside
        # the adjacent segment at an exact sample node. Assign a shared node
        # to the following segment, once, allowing only 0.004 s of x roundoff.
        for x in sorted(set(max(0.0, x) for x in roots if -1e-6 <= x < 1-1e-6)):
            at = rows[i][0] + dt.timedelta(hours=x)
            if YEAR_START <= at < YEAR_END:
                retrograde = derivative(c, min(1, x+0.0001)) < 0
                events.append({"planet": planet, "kind": "Retrograde begins" if retrograde else "Direct motion begins",
                               "at": at.isoformat(), "retrograde": retrograde})
            points.append(x)
        points.sort()
        for lo, hi in zip(points, points[1:]):
            left, right = value(c, lo), value(c, hi)
            sign_left, sign_right = math.floor(left/30), math.floor(right/30)
            if sign_left == sign_right:
                continue
            if abs(sign_left-sign_right) != 1:
                raise ValueError("non-adjacent hourly sign crossing")
            target = max(sign_left, sign_right)*30
            x = bisect(lambda x: value(c, x)-target, lo, hi)
            at = rows[i][0] + dt.timedelta(hours=x)
            if YEAR_START <= at < YEAR_END:
                events.append({"planet": planet, "kind": "Sign ingress", "at": at.isoformat(),
                               "from_sign_number": sign_left % 12 + 1,
                               "to_sign_number": sign_right % 12 + 1,
                               "retrograde": right < left})
    return events


def main():
    """Collect NASA samples, derive independent events and write the offline planetary regression fixture."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--swetest", default=".local/swetest.exe")
    args = parser.parse_args()
    cli = (ROOT / args.swetest).resolve()
    CACHE.mkdir(parents=True, exist_ok=True)
    count = int((STOP-START).total_seconds()/3600)+1
    origins = ayanamsas(cli, START, count)
    epochs = [dt.datetime.fromisoformat(x) for x in EPOCHS]
    epoch_origins = [ayanamsas(cli, at)[0] for at in epochs]
    positions, events, provenance = [], [], []
    for planet, body in BODIES.items():
        rows, source = request(f"annual-2026-{planet}", body,
                               {"START_TIME": f"'{START:%Y-%m-%d %H:%M}'",
                                "STOP_TIME": f"'{STOP:%Y-%m-%d %H:%M}'", "STEP_SIZE": "'1 h'"})
        if len(rows) != count or any(abs((row[0]-(START+dt.timedelta(hours=i))).total_seconds()) > 0.001 for i, row in enumerate(rows)):
            raise ValueError(f"unexpected time grid for {planet}")
        events.extend(annual_events(planet, rows, origins))
        source.update({"planet": planet, "purpose": "annual search", "rows": len(rows)})
        provenance.append(source)
        times = ",".join(f"{2440587.5+at.timestamp()/86400:.12f}" for at in epochs)
        samples, source = request(f"epochs-{planet}", body, {"TLIST": f"'{times}'", "TLIST_TYPE": "'JD'"})
        if len(samples) != len(epochs):
            raise ValueError("missing position samples")
        for i, epoch in enumerate(epochs):
            # Horizons sorts a TLIST chronologically rather than preserving
            # the caller's order. Match the returned JD, never zip by index.
            at, lon, lat = min(samples, key=lambda row: abs((row[0]-epoch).total_seconds()))
            if abs((at-epoch).total_seconds()) > 0.002:
                raise ValueError("position sample clock mismatch")
            positions.append({"planet": planet, "at": epochs[i].isoformat(),
                              "jpl_apparent_tropical_longitude": lon,
                              "jpl_apparent_latitude": lat, "lahiri_origin": epoch_origins[i],
                              "sidereal_longitude": (lon-epoch_origins[i]) % 360})
        source.update({"planet": planet, "purpose": "matched positions", "rows": len(samples)})
        provenance.append(source)
        print(f"NASA/JPL collected {planet}: {count} annual samples and {len(samples)} exact epochs", flush=True)
    events.sort(key=lambda e: (e["at"], e["planet"], e["kind"]))
    # Independently confirm the three provider-disputed ingresses on a much
    # denser NASA time grid. No application result supplies these values.
    dense = []
    for planet, start in (("Jupiter", "2026-10-31T06:00:00+00:00"),
                          ("Mercury", "2026-07-07T05:00:00+00:00"),
                          ("Venus", "2026-11-05T19:00:00+00:00")):
        a = dt.datetime.fromisoformat(start)
        b = a + dt.timedelta(hours=1)
        rows, source = request("dense-ingress-" + planet, BODIES[planet],
                               {"START_TIME": f"'{a:%Y-%m-%d %H:%M}'",
                                "STOP_TIME": f"'{b:%Y-%m-%d %H:%M}'", "STEP_SIZE": "'1 m'"})
        offsets = ayanamsas(cli, a, 61, step_minutes=1)
        if len(rows) != len(offsets):
            raise ValueError("incomplete dense reference")
        values = unwrap([(row[1]-origin) % 360 for row, origin in zip(rows, offsets)])
        before = len(dense)
        for i in range(len(rows)-1):
            left, right = values[i:i+2]
            if math.floor(left/30) == math.floor(right/30):
                continue
            target = max(math.floor(left/30), math.floor(right/30))*30
            fraction = (target-left)/(right-left)
            at = rows[i][0] + (rows[i+1][0]-rows[i][0])*fraction
            dense.append({"planet": planet, "kind": "Sign ingress", "at": at.isoformat()})
        if len(dense) != before+1:
            raise ValueError(f"expected one disputed ingress for {planet}")
        source.update({"planet": planet, "purpose": "dense one-minute ingress", "rows": len(rows)})
        provenance.append(source)
    out = {"source": "NASA/JPL Horizons DE441; Earth geocentre; apparent observer quantity 31",
           "api": "https://ssd-api.jpl.nasa.gov/doc/horizons.html",
           "manual": "https://ssd.jpl.nasa.gov/horizons/manual.html#obsquan",
           "method": "Independent hourly NASA search; four-point Lagrange cubic; shared nutation-inclusive Lahiri origin from Swiss CLI. Not an independent certification of Lahiri or nodes.",
           "position_tolerance_arcsec": 0.5, "ingress_tolerance_seconds": 60,
           "station_tolerance_seconds": 120, "year": 2026,
           "positions": positions, "events": events, "dense_events": dense,
           "provenance": provenance}
    target = ROOT / "internal/swisseph/testdata/jpl-sky-reference.json"
    target.write_text(json.dumps(out, indent=2)+"\n", encoding="utf-8")
    print(f"Wrote {len(positions)} positions and {len(events)} independent annual events", flush=True)


if __name__ == "__main__":
    main()
