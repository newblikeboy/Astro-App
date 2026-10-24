"""Download credited Wikimedia images and prepare small local PNGs for devotional cards."""

import concurrent.futures
import html
import io
import json
import re
import time
import threading
from pathlib import Path

import requests
from PIL import Image

ROOT = Path(__file__).resolve().parent.parent
HEADERS = {"User-Agent": "Mishra4YouDevotionalLibrary/1.0 (local website image preparation)"}
SOURCE_INFO = {}
REQUEST_LOCK = threading.Lock()
LAST_REQUEST = 0.0
SUBJECTS = {
    "ganesha": ("Ganesha", "गणेश"), "hanuman": ("Hanuman", "हनुमान"),
    "krishna": ("Krishna", "कृष्ण"), "rama": ("Rama", "राम"),
    "shiva": ("Shiva", "शिव"), "vishnu": ("Vishnu", "विष्णु"),
    "lakshmi-narayana": ("Lakshmi Narayana", "लक्ष्मी नारायण"),
    "badrinath": ("Badrinath Temple", "बद्रीनाथ मंदिर"),
    "surya": ("Surya", "सूर्य"), "shani": ("Shani", "शनि"),
    "narasimha": ("Narasimha", "नृसिंह"), "govardhan": ("Govardhan Hill", "गोवर्धन"),
    "chitragupta": ("Chitragupta", "चित्रगुप्त"), "kubera": ("Kubera", "कुबेर"),
    "dhanvantari": ("Dhanvantari", "धन्वन्तरि"), "vishwakarma": ("Vishvakarma", "विश्वकर्मा"),
    "brahma": ("Brahma", "ब्रह्मा"), "bhairava": ("Bhairava", "भैरव"),
    "parashurama": ("Parashurama", "परशुराम"), "durga": ("Durga", "दुर्गा"),
    "lakshmi": ("Lakshmi", "लक्ष्मी"), "santoshi": ("Santoshi Mata", "संतोषी माता"),
    "saraswati": ("Saraswati", "सरस्वती"), "vaishno": ("Vaishno Devi Temple", "वैष्णो देवी मंदिर"),
    "ganga": ("Ganga (goddess)", "गंगा"), "ahoi": ("Ahoi Ashtami", "अहोई अष्टमी"),
    "ekadashi": ("Ekadashi", "एकादशी"), "parvati": ("Parvati", "पार्वती"),
    "lalita": ("Tripura Sundari", "ललिता त्रिपुर सुन्दरी"), "tulasi": ("Tulasi in Hinduism", "तुलसी"),
    "gayatri": ("Gayatri", "गायत्री"), "shitala": ("Shitala", "शीतला"),
    "gau": ("Kamadhenu", "गौ माता"), "vindhyeshwari": ("Vindhyavasini", "विन्ध्यवासिनी"),
    "kali": ("Kali", "काली"), "radha": ("Radha", "राधा"),
    "annapurna": ("Annapurna (goddess)", "अन्नपूर्णा"), "bagalamukhi": ("Bagalamukhi", "बगलामुखी"),
    "narmada": ("Narmada River", "नर्मदा"), "shakambhari": ("Shakambhari", "शाकम्भरी"),
    "sharda": ("Saraswati", "शारदा सरस्वती"), "sita": ("Sita", "सीता"),
    "yamuna": ("Yamuna in Hinduism", "यमुना"), "shyam": ("Khatushyam Temple", "खाटू श्याम"),
    "ramdev": ("Ramdev Pir", "रामदेव पीर"), "sai": ("Sai Baba of Shirdi", "साईं बाबा"),
    "mahavir": ("Mahavira", "महावीर"), "ravidas": ("Ravidas", "रविदास"),
    "gangaram": ("Baba Gangaram", "बाबा गंगाराम"), "pretaraj": ("Mehandipur Balaji Temple", "मेहंदीपुर बालाजी मंदिर"),
    "gorakha": ("Gorakhnath", "गोरखनाथ"), "jaharveer": ("Gogaji", "जाहरवीर गोगाजी"),
    "rani-sati": ("Rani Sati Temple", "रानी सती मंदिर"), "gita": ("Bhagavad Gita", "भगवद्गीता"),
    "ramayana": ("Ramayana", "रामायण"), "ancestors": ("Pitru Paksha", "पितृ पक्ष"),
    "navagraha": ("Navagraha", "नवग्रह"), "mangal": ("Mangala", "मंगल"),
    "kartikeya": ("Kartikeya", "कार्तिकेय"), "nag": ("Shesha", "शेषनाग"),
    "mahavidya": ("Mahavidya", "महाविद्या"), "yogini": ("Yogini", "योगिनी"),
    "yama": ("Yama (Hinduism)", "यम"), "fire": ("Yajna", "यज्ञ"),
    "agni": ("Agni", "अग्नि"), "indra": ("Indra", "इन्द्र"),
    "shailputri": ("Shailaputri", "शैलपुत्री"), "brahmacharini": ("Brahmacharini", "ब्रह्मचारिणी"),
    "chandraghanta": ("Chandraghanta", "चन्द्रघण्टा"), "kushmanda": ("Kushmanda", "कूष्माण्डा"),
    "skandamata": ("Skandamata", "स्कन्दमाता"), "katyayani": ("Katyayani", "कात्यायनी"),
    "kalaratri": ("Kalaratri", "कालरात्रि"), "mahagauri": ("Mahagauri", "महागौरी"),
    "siddhidatri": ("Siddhidatri", "सिद्धिदात्री"), "chandra": ("Chandra", "चन्द्र"),
    "budha": ("Budha", "बुध"), "brihaspati": ("Brihaspati", "बृहस्पति"),
    "shukra": ("Shukra", "शुक्र"), "rahu": ("Rahu", "राहु"), "ketu": ("Ketu (mythology)", "केतु"),
    "tara": ("Tara (Hindu goddess)", "तारा"), "bhuvaneshvari": ("Bhuvaneshvari", "भुवनेश्वरी"),
    "bhairavi": ("Bhairavi", "भैरवी"), "chhinnamasta": ("Chhinnamasta", "छिन्नमस्ता"),
    "dhumavati": ("Dhumavati", "धूमावती"), "matangi": ("Matangi", "मातंगी"),
    "kamala": ("Kamalatmika", "कमला"),
}


def subject_for(slug):
    """Resolve specific deity names before broader family names; report unknown readings for review."""
    special = {
        "shree-suktam": "lakshmi", "kanakadhara-stotram": "lakshmi",
        "parameshwara-stuti-stotram": "vishnu", "shri-purusha-suktam": "vishnu",
        "sunday-aarti": "surya", "monday-aarti": "shiva", "tuesday-aarti": "hanuman",
        "wednesday-aarti": "krishna", "thursday-aarti": "brihaspati", "friday-aarti": "santoshi",
        "saturday-aarti": "shani", "shanti-path-mantras": "fire", "diwali-mantras": "lakshmi",
        "akshaya-tritiya-mantras": "lakshmi", "lakshmi-ganesha-mantra": "ganesha",
    }
    if slug in special:
        return special[slug]
    rules = [
        (r"narasimha|narsingh", "narasimha"), (r"lakshminarayana|lakshmi-narayana", "lakshmi-narayana"),
        (r"navagraha", "navagraha"), (r"shailputri", "shailputri"), (r"brahmacharini", "brahmacharini"),
        (r"chandraghanta", "chandraghanta"), (r"kushmanda", "kushmanda"), (r"skandamata", "skandamata"),
        (r"katyayani", "katyayani"), (r"kalaratri", "kalaratri"), (r"mahagauri", "mahagauri"),
        (r"siddhidatri", "siddhidatri"), (r"bhagavad-gita", "gita"), (r"ramayan", "ramayana"),
        (r"shiva|shankar|rudra|gaurisha|vishvanath|linga|pashupati|mahadev|gangadhara|kailasavasi|mrityunjaya", "shiva"),
        (r"ganesh|ganapat|vinayaka", "ganesha"), (r"hanuman|hanumat|bajrang|balaji", "hanuman"),
        (r"krishna|gopal|govind|bihari|natavara|giridhari|yashodalala|madhura|nandakumar|achyuta", "krishna"),
        (r"lakshmi", "lakshmi"), (r"saraswati", "saraswati"), (r"sharda", "sharda"),
        (r"bagalamukhi", "bagalamukhi"), (r"bhairava", "bhairava"), (r"bhairavi", "bhairavi"),
        (r"mahavidya", "mahavidya"), (r"yogini", "yogini"), (r"chhinnamasta", "chhinnamasta"),
        (r"dhumavati", "dhumavati"), (r"matangi", "matangi"), (r"kamala-mantras", "kamala"),
        (r"bhuvaneshvari", "bhuvaneshvari"), (r"shodashi|lalita", "lalita"), (r"tara-mantras", "tara"),
        (r"kalika|mahakali|kali-", "kali"), (r"durga|navadurga|mahishasura|bhagavati|ambe-", "durga"),
        (r"aparadha-kshamapana", "durga"), (r"bhavani|parvati", "parvati"),
        (r"santoshi", "santoshi"), (r"vaishno", "vaishno"), (r"ganga-mata|ganga-ashtakam|ganga-dussehra", "ganga"),
        (r"ahoi", "ahoi"), (r"ekadashi", "ekadashi"), (r"tulasi", "tulasi"), (r"gayatri-mata|goddess-gayatri", "gayatri"),
        (r"shitala", "shitala"), (r"gau-mata", "gau"), (r"vindhyeshwari", "vindhyeshwari"),
        (r"radha", "radha"), (r"annapurna", "annapurna"), (r"narmada", "narmada"), (r"shakambhari", "shakambhari"),
        (r"sita|janaki", "sita"), (r"yamuna", "yamuna"), (r"shyam", "shyam"), (r"ramdev", "ramdev"),
        (r"sai", "sai"), (r"mahavir", "mahavir"), (r"ravidas", "ravidas"), (r"gangaram", "fire"),
        (r"pretaraj|pretraj", "pretaraj"), (r"gorakha", "gorakha"), (r"jaharveer", "jaharveer"), (r"rani-sati", "rani-sati"),
        (r"pitar|pitru", "ancestors"), (r"parashurama", "parashurama"), (r"surya|aditya", "surya"),
        (r"shani", "shani"), (r"govardhan|giriraj", "govardhan"), (r"chitragupta", "chitragupta"),
        (r"kubera", "kubera"), (r"dhanvantari", "dhanvantari"), (r"vishwakarma", "vishwakarma"), (r"brahma-", "brahma"),
        (r"mangal", "mangal"), (r"kartikeya", "kartikeya"), (r"^nag-", "nag"), (r"^yama-", "yama"),
        (r"agni", "agni"), (r"indra", "indra"), (r"chandra-mantras", "chandra"), (r"budha", "budha"),
        (r"brihaspati", "brihaspati"), (r"shukra", "shukra"), (r"rahu", "rahu"), (r"ketu", "ketu"),
        (r"badarinatha", "badrinath"), (r"rama|raghu|sitarama", "rama"),
        (r"vishnu|jagdish|narayana|purushottam|satyanarayana|hari-|dinabandhu|kamalapati|venkatesha", "vishnu"),
    ]
    for pattern, subject in rules:
        if re.search(pattern, slug):
            return subject
    raise ValueError(f"No reviewed image subject for {slug}")


def get_json(url, params):
    """Read a public Wikimedia endpoint with bounded retries for transient network errors."""
    global LAST_REQUEST
    for attempt in range(3):
        try:
            with REQUEST_LOCK:
                time.sleep(max(0, 2 - (time.monotonic() - LAST_REQUEST)))
                LAST_REQUEST = time.monotonic()
            response = requests.get(url, params=params, headers=HEADERS, timeout=35)
            if response.status_code == 429:
                delay = min(120, max(30, int(response.headers.get("Retry-After", "60"))))
                print(json.dumps({"status": "Provider rate limit; respecting cooldown", "seconds": delay}), flush=True)
                time.sleep(delay)
            response.raise_for_status()
            return response.json()
        except (requests.RequestException, ValueError):
            if attempt == 2:
                raise
            time.sleep(attempt + 1)


def prepare_sources(subjects):
    """Batch article and Commons metadata lookups so each collection requires only a few API requests."""
    cache_path = ROOT / ".local/devotional-image-sources.json"
    sources = json.loads(cache_path.read_text(encoding="utf-8")) if cache_path.exists() else {}
    pending = [subject for subject in subjects if subject not in sources]
    pages_by_subject = {}
    for offset in range(0, len(pending), 40):
        group = pending[offset:offset + 40]
        data = get_json("https://en.wikipedia.org/w/api.php", {
            "action": "query", "format": "json", "redirects": 1,
            "titles": "|".join(SUBJECTS[subject][0] for subject in group),
            "prop": "pageimages", "pithumbsize": 256, "piprop": "thumbnail|name"})
        query = data.get("query", {})
        aliases = {item["from"]: item["to"] for item in query.get("normalized", []) + query.get("redirects", [])}
        pages = {page["title"]: page for page in query.get("pages", {}).values()}
        for subject in group:
            title = SUBJECTS[subject][0]
            for _ in range(4):
                title = aliases.get(title, title)
            pages_by_subject[subject] = pages.get(title, {})
    filenames = sorted(set(page["pageimage"] for page in pages_by_subject.values() if page.get("pageimage")))
    file_infos = {}
    for offset in range(0, len(filenames), 40):
        data = get_json("https://commons.wikimedia.org/w/api.php", {
            "action": "query", "format": "json", "titles": "|".join("File:" + filename for filename in filenames[offset:offset + 40]),
            "prop": "imageinfo", "iiprop": "url|extmetadata", "iiurlwidth": 256})
        for page in data.get("query", {}).get("pages", {}).values():
            file_infos[page["title"].removeprefix("File:")] = page.get("imageinfo", [{}])[0]
    for subject, page in pages_by_subject.items():
        filename = page.get("pageimage", "")
        sources[subject] = {"page": page, "info": file_infos.get(filename.replace("_", " "), file_infos.get(filename, {}))}
    cache_path.write_text(json.dumps(sources, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    return sources


def plain(value):
    """Convert provider attribution HTML into plain text before it reaches the application."""
    return html.unescape(re.sub(r"<[^>]+>", "", value or "")).strip()


def obtain_subject(subject):
    """Download a credited subject photograph or painting and transcode it to a bounded-size PNG."""
    title, label_hi = SUBJECTS[subject]
    target = ROOT / "web/public/images/devotional" / f"{subject}.png"
    cache = ROOT / ".local/devotional-image-metadata" / f"{subject}.json"
    if target.exists() and cache.exists():
        return subject, json.loads(cache.read_text(encoding="utf-8"))
    page = SOURCE_INFO.get(subject, {}).get("page", {})
    filename = page.get("pageimage")
    if not filename:
        return subject, {"error": "No article image", "title": title}
    info = SOURCE_INFO.get(subject, {}).get("info", {})
    meta = info.get("extmetadata", {})
    license_name = plain(meta.get("LicenseShortName", {}).get("value"))
    if not info.get("url") or not re.search(r"public domain|^pd|cc0|cc by|cc-by", license_name, re.I):
        return subject, {"error": "Image requires alternate source", "title": title, "file": filename, "license": license_name}
    url = info.get("thumburl") or page["thumbnail"]["source"]
    response = requests.get(url, headers=HEADERS, timeout=45)
    response.raise_for_status()
    image = Image.open(io.BytesIO(response.content))
    image.thumbnail((256, 256), Image.Resampling.LANCZOS)
    image = image.convert("RGBA").quantize(colors=192)
    target.parent.mkdir(parents=True, exist_ok=True)
    image.save(target, format="PNG", optimize=True)
    record = {"src": f"/images/devotional/{subject}.png", "label": title,
              "labelHi": label_hi, "width": image.width, "height": image.height,
              "source": info["descriptionurl"], "artist": plain(meta.get("Artist", {}).get("value")),
              "license": license_name, "licenseUrl": plain(meta.get("LicenseUrl", {}).get("value")),
              "changes": "Resized to fit 256 pixels and converted to PNG; no compositional edits."}
    cache.parent.mkdir(parents=True, exist_ok=True)
    cache.write_text(json.dumps(record, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    return subject, record


def main():
    """Prepare every resolved card subject and report unresolved images without publishing incomplete mappings."""
    global SOURCE_INFO
    entries = json.loads((ROOT / "web/src/data/devotional-index.json").read_text(encoding="utf-8"))
    readings = {entry["slug"]: subject_for(entry["slug"]) for entry in entries}
    images = {}
    SOURCE_INFO = prepare_sources(sorted(set(readings.values())))
    with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
        futures = {pool.submit(obtain_subject, subject): subject for subject in sorted(set(readings.values()))}
        for future in concurrent.futures.as_completed(futures):
            subject = futures[future]
            try:
                _, record = future.result()
            except Exception as error:
                record = {"error": str(error), "title": SUBJECTS[subject][0]}
            images[subject] = record
            print(json.dumps({"subject": subject, "status": record.get("error", "saved")}, ensure_ascii=True), flush=True)
    result = {"images": images, "readings": readings}
    (ROOT / ".local/devotional-art-preparation.json").write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    issues = {key: value for key, value in images.items() if "error" in value}
    print(json.dumps({"subjects": len(images), "unresolved": issues}, ensure_ascii=True, indent=2))
    if issues:
        raise SystemExit(1)
    (ROOT / "web/src/data/devotional-art.json").write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    credits = ["# Devotional card image credits", "",
               "These PNG copies retain the license of each original image. They were resized to fit 256 pixels, color-optimized and converted to PNG without compositional edits.", ""]
    for subject, image in sorted(images.items()):
        credits.append(f"- **{subject}.png**: [{image['label']}]({image['source']}) · {image['artist'] or 'Artist as listed at source'} · {image['license']}. "
                       + (f"[License]({image['licenseUrl']})." if image['licenseUrl'] else ""))
    (ROOT / "web/public/images/devotional/CREDITS.md").write_text("\n".join(credits) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
