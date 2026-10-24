"""Build the local text-only prayer catalogue from reviewed web-reader exports.

Usage: python scripts/build-devotional-library.py .local/devotional-source-v2.json
Only original Devanagari prayer blocks are retained, never descriptions,
translations, media, navigation, comments or provider widgets. Reader exports
are private acquisition evidence, not a runtime dependency.
"""

import argparse
import json
import re
from pathlib import Path
from urllib.parse import urlsplit, urlunsplit

ROOT = Path(__file__).resolve().parent.parent
CATEGORIES = ("aarti", "chalisa", "stotram", "ashtakam", "mantra", "kavacham")
MANTRA_HINDI = {
    "Lord Ganesha": "श्री गणेश मंत्र", "Lord Kubera": "कुबेर मंत्र",
    "Lord Shiva": "शिव मंत्र", "Lord Rama": "श्री राम मंत्र",
    "Lord Krishna": "श्री कृष्ण मंत्र", "Lord Hanuman": "हनुमान मंत्र",
    "Lord Vishnu": "विष्णु मंत्र", "Lord Narasimha": "नृसिंह मंत्र",
    "Lord Narasimha Gayatri Mantra": "नृसिंह गायत्री मंत्र",
    "Lord Parashurama": "परशुराम मंत्र", "Shri Lakshmi": "लक्ष्मी मंत्र",
    "Goddess Gayatri": "गायत्री मंत्र", "Goddess Saraswati": "सरस्वती मंत्र",
    "Goddess Durga": "दुर्गा मंत्र", "Shanti Path": "शान्ति पाठ",
    "Lakshmi Ganesha Mantra": "लक्ष्मी गणेश मंत्र", "Diwali Mantras": "दीपावली मंत्र",
    "Akshaya Tritiya Mantras": "अक्षय तृतीया मंत्र", "Lord Agni": "अग्नि गायत्री मंत्र",
    "Lord Indra": "इन्द्र गायत्री मंत्र", "Surya Mantras": "सूर्य मंत्र",
    "Chandra Mantras": "चन्द्र मंत्र", "Mangal Mantras": "मंगल मंत्र",
    "Budha Mantras": "बुध मंत्र", "Brihaspati Mantras": "बृहस्पति मंत्र",
    "Shukra Mantras": "शुक्र मंत्र", "Shani Mantras": "शनि मंत्र",
    "Rahu Mantras": "राहु मंत्र", "Ketu Mantras": "केतु मंत्र",
    "Kali Mantras": "काली मंत्र", "Tara Mantras": "तारा मंत्र",
    "Shodashi Mantras": "षोडशी मंत्र", "Bhuvaneshvari Mantras": "भुवनेश्वरी मंत्र",
    "Bhairavi Mantras": "भैरवी मंत्र", "Chhinnamasta Mantras": "छिन्नमस्ता मंत्र",
    "Dhumavati Mantras": "धूमावती मंत्र", "Bagalamukhi Mantras": "बगलामुखी मंत्र",
    "Matangi Mantras": "मातंगी मंत्र", "Kamala Mantras": "कमला मंत्र",
}
GROUPS = {"Navdurga Aarti Collection", "Weekday Aarti Collection", "Navagraha", "Dasha Mahavidya"}


def reader_lines(raw):
    """Recover numbered reader lines, retaining empty lines and rejecting duplicate line numbers."""
    return {int(m[1]): m[2].strip() for m in re.finditer(r"L(\d+): ([\s\S]*?)(?=L\d+: |$)", raw)}


def source_url(raw):
    """Return the canonical article URL without provider language or tracking parameters."""
    match = re.search(r"https://[^\s)]+", raw)
    if not match:
        raise ValueError("No source URL in reader export")
    parsed = urlsplit(match[0].rstrip(":"))
    return urlunsplit((parsed.scheme, parsed.netloc, parsed.path, "", ""))


def stanzas(rows):
    """Rejoin word-segmented Sanskrit and preserve complete verse and section boundaries."""
    content = [row for row in rows if row]
    segmented = sum(len(row.split()) <= 1 for row in content) > len(content) * 0.55
    result, pending = [], []
    for row in rows:
        if not row:
            continue
        if row.startswith("#"):
            if pending:
                result.append((" " if segmented else "\n").join(pending))
                pending = []
            label = row.lstrip("# ").strip()
            if re.search(r"[\u0900-\u097f]", label):
                result.append(label)
            continue
        pending.append(row)
        if re.search(r"॥(?:[०-९0-9]+॥)?(?:\s*x[0-9]+)?[)\s]*$", row):
            result.append((" " if segmented else "\n").join(pending))
            pending = []
    if pending:
        result.append((" " if segmented else "\n").join(pending))
    return [re.sub(r" +([।॥])", r"\1", verse) for verse in result]


def extract(entry):
    """Extract original prayer text within its bounded article region; fail on incomplete exports."""
    raw = entry["raw"]
    lines = reader_lines(raw)
    if len(lines) < 100:
        raise ValueError("Source unavailable")
    numbers = sorted(lines)
    if entry["cat"] == "mantra":
        start = next((n for n in numbers if n >= 110 and
                      (lines[n].startswith("### 1.") or
                       (re.match(r"^[\u0900-\u097f]", lines[n]) and "" not in lines[n]))), None)
    else:
        start = next((n for n in numbers if n >= 110 and lines[n].startswith("###")
                      and "" not in lines[n] and re.search(r"[\u0900-\u097f]", lines[n])), None)
    if start is None:
        raise ValueError("Prayer start marker missing")
    end = next((n for n in numbers if n > start and
                ("[Button: A" in lines[n] or lines[n] == "A+A A-" or "†Follow†" in lines[n] or
                 re.search(r"^#+ (?:Translation|Meaning)", lines[n], re.I))), None)
    if end is None:
        raise ValueError("Prayer end marker missing; export may be truncated")
    if any(n not in lines for n in range(start, end)):
        raise ValueError("Missing lines inside prayer")
    rows = []
    skip_meaning = False
    for n in range(start, end):
        row = lines[n]
        if entry["cat"] == "mantra":
            # Sanskrit originals only. Transliterations and translations stay out.
            if row.lstrip("* ").startswith("#"):
                skip_meaning = False
                match = re.search(r"^### (\d+)\.", row)
                if match:
                    rows.append(f"### ॥ {match[1]} ॥")
                continue
            if re.match(r"^(?:मन्त्र|मंत्र) अर्थ", row):
                skip_meaning = True
            if skip_meaning:
                continue
            roman_original = entry["title"] == "Diwali Mantras" and re.match(r"^[A-Za-z].*[।॥]$", row)
            if (not re.match(r"^[\u0900-\u097f]", row) and not roman_original) or "" in row:
                continue
        elif re.search(r"यह (?:प्रसिद्ध )?आरती|सबसे प्रसिद्ध आरती|लोकप्रिय आरती|एक और आरती|गाया जाता है|गायी जाती है", row):
            continue
        elif "" in row or row.startswith("["):
            raise ValueError(f"Unexpected link/widget within prayer at line {n}")
        row = row.replace("\u200b", "").replace("\u200c", "").replace("\u200d", "")
        rows.append(row)
    verses = stanzas(rows)
    if not verses or not any(re.search(r"[।॥]", verse) for verse in verses):
        raise ValueError("No complete prayer verse")
    heading = next((re.sub(r"^#+\s*", "", lines[n]).strip("॥ ") for n in numbers
                    if n >= 110 and re.match(r"^## [\u0900-\u097f]", lines[n])), None)
    title_hi = entry.get("titleHi") or MANTRA_HINDI.get(entry["title"]) or heading
    if not title_hi:
        title_hi = re.sub(r"^#+\s*", "", lines[start]).strip("॥ ")
    title = entry["title"]
    if entry["cat"] == "mantra" and not title.endswith(("Mantra", "Mantras")):
        title += " Mantras"
    slug = re.sub(r"[^a-z0-9]+", "-", title.lower()).strip("-")
    incipit = entry["incipit"]
    if not re.search(r"[\u0900-\u097f]", incipit):
        incipit = next(verse for verse in verses if not verse.startswith("॥"))
        incipit = incipit.split("\n")[0][:110]
    return {"slug": slug, "categories": [entry["cat"]], "title": title,
            "textLanguage": "sa-Latn" if entry["title"] == "Diwali Mantras" else "hi",
            "titleHi": title_hi, "incipit": incipit, "source": source_url(raw), "verses": verses}


def extract_supplement(entry):
    """Create a prayer from manually bounded original-script rows at a separately reviewed source."""
    title = entry["title"]
    labels = {"Shri Ravidas Chalisa": "श्री रविदास चालीसा",
              "Shri Baba Gangaram Chalisa": "श्री बाबा गंगाराम चालीसा",
              "Parameshwara Stuti Stotram": "परमेश्वर स्तुति स्तोत्रम्",
              "Shri Parashurama Ashtakam": "श्री परशुरामाष्टकम्"}
    rows = entry["originalRows"]
    return {"slug": re.sub(r"[^a-z0-9]+", "-", title.lower()).strip("-"),
            "categories": [entry["cat"]], "title": title, "titleHi": labels[title],
            "incipit": entry["incipit"], "source": entry["alternateSource"],
            "verses": stanzas(rows)}


def main():
    """Generate the bundled catalogue and an explicit source-coverage report without network access."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", type=Path)
    parser.add_argument("--additional-source", type=Path, action="append", default=[])
    parser.add_argument("--supplement", type=Path)
    args = parser.parse_args()
    entries = json.loads(args.source.read_text(encoding="utf-8"))
    for additional_source in args.additional_source:
        entries.extend(json.loads(additional_source.read_text(encoding="utf-8")))
    supplements = {entry["title"]: entry for entry in json.loads(args.supplement.read_text(encoding="utf-8"))} if args.supplement else {}
    records, issues, sources = [], [], {}
    for entry in entries:
        if entry["title"] in GROUPS:
            continue
        try:
            record = extract_supplement(supplements[entry["title"]]) if entry["title"] in supplements else extract(entry)
            # A prayer listed under two categories remains one local reading.
            key = record["source"]
            if key in sources:
                prior = sources[key]
                prior["categories"] = list(dict.fromkeys(prior["categories"] + record["categories"]))
            else:
                if any(prior["slug"] == record["slug"] for prior in records):
                    record["slug"] += "-" + entry["cat"]
                records.append(record)
                sources[key] = record
        except (ValueError, StopIteration) as error:
            issues.append({"title": entry["title"], "category": entry["cat"], "reason": str(error),
                           "source": source_url(entry["raw"])})
    data_dir = ROOT / "web/src/data"
    destination = data_dir / "devotional-index.json"
    metadata = [{key: value for key, value in record.items() if key != "verses"} for record in records]
    destination.write_text(json.dumps(metadata, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    readings_dir = data_dir / "devotional"
    readings_dir.mkdir(exist_ok=True)
    for category in CATEGORIES:
        payload = {record["slug"]: record["verses"] for record in records if record["categories"][0] == category}
        (readings_dir / f"{category}.json").write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    report = {"reviewed": "2026-10-21", "readings": len(records),
              "categories": {cat: sum(cat in x["categories"] for x in records)
                             for cat in CATEGORIES},
              "unresolved": issues}
    (ROOT / ".local/devotional-coverage.json").write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(report, ensure_ascii=True, indent=2))


if __name__ == "__main__":
    main()
