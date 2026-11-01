"""Import the supplied Guruji PDFs without turning their prose into executable instructions."""

import argparse
import hashlib
import json
import pathlib
import re
import shutil

import pymupdf


PLANETS = [
    ("Sun", "सूर्य", 1, 20, "confidence, leadership, recognition and authority"),
    ("Moon", "चंद्रमा", 20, 36, "emotions, sensitivity, adaptability and care"),
    ("Mars", "मंगल", 37, 55, "initiative, energy, courage and conflict"),
    ("Mercury", "बुध", 55, 65, "communication, learning, analysis and trade"),
    ("Jupiter", "गुरु", 65, 76, "knowledge, guidance, values and judgement"),
    ("Venus", "शुक्र", 76, 86, "relationships, creativity, comfort and enjoyment"),
    ("Saturn", "शनि", 86, 96, "discipline, responsibility, patience and sustained effort"),
    ("Rahu", "राहु", 96, 111, "ambition, unfamiliar experiences, technology and uncertainty"),
    ("Ketu", "केतु", 111, 125, "detachment, introspection, research and spiritual interests"),
]
HOUSES = [
    ("व्यक्तित्व और जीवन-दृष्टि", "personality and outlook"),
    ("परिवार, वाणी और संचित धन", "family, speech and savings"),
    ("साहस, संवाद और भाई-बहन", "initiative, communication and siblings"),
    ("घर, माता और घरेलू सुख", "home, mother and domestic comfort"),
    ("विद्या, रचनात्मकता और संतान", "learning, creativity and children"),
    ("दैनिक कार्य, सेवा और प्रतिस्पर्धा", "daily work, service and competition"),
    ("विवाह और साझेदारी", "marriage and partnerships"),
    ("परिवर्तन, गहन अध्ययन और साझा संसाधन", "change, research and shared resources"),
    ("धर्म, गुरु और उच्च शिक्षा", "beliefs, mentors and higher learning"),
    ("कर्म, करियर और सार्वजनिक प्रतिष्ठा", "work, career and public standing"),
    ("आय, मित्र और दीर्घकालीन लक्ष्य", "income, networks and long-term goals"),
    ("व्यय, विदेश और एकांत", "expenses, foreign connections and solitude"),
]


def clean_text(text):
    """Repair known PDF font-spacing artifacts while retaining the supplied wording and conditions."""
    for old, new in {"क ुंडली": "कुंडली", "क ुंडल": "कुंडल", "च िं": "चिं", "स िं": "सिं", "आत्मच िं": "आत्मचिं", " ाँ": "ाँ", " ां": "ां", "म ूँ": "मूँ", "गेह ूँ": "गेहूँ"}.items():
        text = text.replace(old, new)
    text = re.sub(r"={3,}", "", text)
    # Preserve bullet/paragraph boundaries; ordinary PDF line wraps become spaces.
    text = re.sub(r" *\n(?=•|\d+\. |शुभ फल|अशुभ|विशेष|निष्कर्ष|व्यवहारिक सीख|करियर:|पिता और|महत्वपूर्ण)", "\n\n", text)
    text = re.sub(r"\n{3,}", "\n\n", text)
    return re.sub(r"(?<!\n)\n(?!\n)", " ", text).strip()


def document_text(path):
    """Return document text and one-based page offsets for exact source locators."""
    doc = pymupdf.open(path)
    parts, offsets = [], []
    length = 0
    for page in doc:
        offsets.append(length)
        value = page.get_text() + "\n"
        parts.append(value)
        length += len(value)
    return "".join(parts), offsets, doc


def repair_house_font(text):
    """Repair the short PDF's broken font-to-Unicode mappings; original PDF bytes remain archived unchanged."""
    replacements = {
        "अद्धधकता":"अधिकता", "अद्धधकार":"अधिकार", "अद्धभलार्ा":"अभिलाषा", "अनुभूद्धत":"अनुभूति", "आद्धथषक":"आर्थिक", "आद्धशर्":"आशीष", "इसद्धलए":"इसलिए", "उपलद्धधध":"उपलब्धि", "कद्धठनाई":"कठिनाई", "कोद्धशश":"कोशिश", "क्योंद्धक":"क्योंकि", "दैद्धनक":"दैनिक",
        "द्धकताब":"किताब", "द्धकया":"किया", "द्धकसी":"किसी", "द्धचंता":"चिंता", "द्धदखेगी":"दिखेगी", "द्धनकटवती":"निकटवर्ती", "द्धनद्धित":"निश्चित", "द्धपता":"पिता", "द्धबगाड़":"बिगाड़", "द्धबगड़":"बिगड़", "द्धमल":"मिल", "द्धलए":"लिए", "द्धलखने":"लिखने", "द्धवदेश":"विदेश", "द्धवद्या":"विद्या", "द्धवनम्रता":"विनम्रता", "द्धववेक":"विवेक", "द्धवशेर्":"विशेष", "द्धशक्षा":"शिक्षा", "द्धस्थद्धत":"स्थिति", "द्धहरदय":"हृदय", "द्धिस":"जिस", "नद्धनहाल":"ननिहाल", "पूद्धतष":"पूर्ति", "प्राद्धप्त":"प्राप्ति", "बद्धिया":"बढ़िया", "बाबाद्धसर":"बवासीर", "भूद्धम":"भूमि", "मद्धस्तष्क":"मस्तिष्क", "मद्धहला":"महिला", "मानद्धसक":"मानसिक", "याद्धन":"यानी", "लेद्धकन":"लेकिन", "वैवाद्धहक":"वैवाहिक", "संपद्धि":"संपत्ति", "संबंद्धधत":"संबंधित", "हाद्धन":"हानि", "िूद्धनयर":"जूनियर", "िमा":"जमा", "िो":"जो", "िन्म":"जन्म", "िुड़ा":"जुड़ा", "िाने":"जाने", "िाए":"जाए", "िाती":"जाती", "िाता":"जाता", "िरुर":"जरूर", "गवष":"गर्व", "धमष":"धर्म", "कायष":"कार्य", "ददष":"दर्द", "संघर्ष":"संघर्ष"
    }
    for old, new in replacements.items():
        text = text.replace(old, new)
    return text


def locator(offsets, start, end, source):
    """Locate the inclusive source pages containing a passage, excluding trailing whitespace."""
    return {"source_id": source, "page_start": max(i + 1 for i, x in enumerate(offsets) if x <= start), "page_end": max(i + 1 for i, x in enumerate(offsets) if x < end)}


def import_catalog(bhav_path, grah_path, image_paths, root):
    """Archive the five supplied files and build all 108 placement passages and conditional references."""
    archive = root / "references/guruji/2026-11-01"
    archive.mkdir(parents=True, exist_ok=True)
    files = [("bhav-upay", bhav_path, "bhavphal-aur-upay.pdf"), ("house-outcomes", grah_path, "grahon-ka-faladesh.pdf")]
    files += [(kind, path, name) for (kind, name), path in zip([
        ("rashi-letters", "rashi-letters.jpeg"), ("house-significations", "house-significations.jpeg"), ("lagna-whiteboard", "lagna-whiteboard.jpeg")
    ], image_paths)]
    sources = []
    for source_id, path, filename in files:
        dest = archive / filename
        shutil.copyfile(path, dest)
        sources.append({"id": source_id, "filename": path.name, "archive": dest.relative_to(root).as_posix(), "sha256": hashlib.sha256(path.read_bytes()).hexdigest()})
    text, offsets, doc = document_text(bhav_path)
    entries = []
    for planet, hindi, page_start, page_end, themes in PLANETS:
        lo = offsets[page_start - 1]
        hi = offsets[page_end] if page_end < len(offsets) else len(text)
        segment = text[lo:hi]
        if planet == "Moon":
            headings = list(re.finditer(r"^(?:पहला|दूसरा|तीसरा|चौथा|प\s*ाँचव\s*ाँ|छठा|सातव\s*ाँ|आठव\s*ाँ|नौव\s*ाँ|दसव\s*ाँ|ग्यारहव\s*ाँ|बारहव\s*ाँ) भाव —.*$", segment, re.M))
        else:
            headings = list(re.finditer(r"^(\d{1,2})\. [^\n]*भाव में " + hindi + r"[^\n]*$", segment, re.M))
        headings = headings[:12]
        if len(headings) != 12:
            raise ValueError(f"{planet}: expected 12 headings, found {len(headings)}")
        for i, heading in enumerate(headings):
            if planet != "Moon" and int(heading.group(1)) != i + 1:
                raise ValueError(f"{planet}: non-sequential house heading")
            start = lo + heading.start()
            end = lo + headings[i + 1].start() if i < 11 else hi
            if i == 11:
                # The next planet introduction/summary must not leak into house twelve.
                stops = [m.start() for m in re.finditer(r"^13\. |^={3,}|^क ुंडली के सभी|^राहु का 12|^केतु का 12|^मंगल का क|^1\. नवग्रह", text[start:end], re.M)]
                if stops:
                    end = start + min(stops)
            raw = text[start:end].rstrip()
            end = start + len(raw)
            entries.append({"planet": planet, "house": i + 1, "title": {"hi": f"भाव {i+1} में {hindi}", "en": f"{planet} in house {i+1}"}, "summary": {"hi": f"यह भाव {HOUSES[i][0]} से जुड़ा है। नीचे गुरुजी के संदर्भ में इस स्थान के संभावित फल और उनकी शर्तें पढ़ें। केवल इस स्थान से शुभ या अशुभ परिणाम निश्चित नहीं होता।", "en": f"This house concerns {HOUSES[i][1]}. The supplied passage considers {themes} in this area, including possible benefits and challenges. Placement alone does not select a favourable or adverse outcome."}, "hindi_excerpt": clean_text(raw), "source": locator(offsets, start, end, "bhav-upay")})
    remedies = []
    remedy_heads = list(re.finditer(r"^(?:[2-9]|10)\. (सूर्य|चंद्रमा|मंगल|बुध|गुरु|शुक्र|शनि|राहु|केतु) ग्रह के उपाय *$", text, re.M))
    if len(remedy_heads) != 9:
        raise ValueError(f"expected 9 remedy headings, found {len(remedy_heads)}")
    end_remedies = re.search(r"^11\. सभी नौ ग्रहों के उपाय", text, re.M).start()
    for i, head in enumerate(remedy_heads):
        start, end = head.start(), remedy_heads[i+1].start() if i < 8 else end_remedies
        raw = text[start:end].rstrip()
        remedies.append({"planet": PLANETS[i][0], "hindi_excerpt": clean_text(raw), "source": locator(offsets, start, start+len(raw), "bhav-upay")})
    house_text, house_offsets, _ = document_text(grah_path)
    house_heads = list(re.finditer(r"^ *(?:प्रथम|\d+(?:nd|ed|rd|th))\s+भाव का (?:सकारात्मक|नाकारात्मक|नकारात्मक) फला ?देश *$", house_text, re.M))
    if len(house_heads) != 24:
        raise ValueError(f"expected 24 house-outcome headings, found {len(house_heads)}")
    house_outcomes = []
    for i in range(12):
        passages = []
        for head_index in [2*i,2*i+1]:
            start = house_heads[head_index].start()
            end = house_heads[head_index+1].start() if head_index < 23 else len(house_text)
            raw = house_text[start:end].rstrip()
            passages.append({"hindi_excerpt": repair_house_font(clean_text(raw)), "source": locator(house_offsets, start, start+len(raw), "house-outcomes")})
        house_outcomes.append({"house": i+1, "meaning": {"hi": HOUSES[i][0], "en": HOUSES[i][1]}, "positive": passages[0], "negative": passages[1]})
    result = {"version": "guruji-supplied-2026-11-01-v2", "sources": sources, "placements": entries, "remedies": remedies, "house_references": house_outcomes}
    dest = root / "internal/platform/data/guruji-faladesh.json"
    dest.write_text(json.dumps(result, ensure_ascii=False, indent=2)+"\n", encoding="utf-8")
    (archive / "manifest.json").write_text(json.dumps({"received": "2026-11-01", "sources": sources}, ensure_ascii=False, indent=2)+"\n", encoding="utf-8")
    print(f"Imported {len(entries)} placements, {len(remedies)} remedy references, 12 conditional house pairs; archived {len(sources)} sources.")


def main():
    """Accept explicit input paths so imports never consume unrelated documents in Downloads."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--bhav", type=pathlib.Path, required=True)
    parser.add_argument("--grah", type=pathlib.Path, required=True)
    parser.add_argument("--images", type=pathlib.Path, nargs=3, required=True)
    args = parser.parse_args()
    import_catalog(args.bhav, args.grah, args.images, pathlib.Path(__file__).resolve().parents[1])


if __name__ == "__main__":
    main()
