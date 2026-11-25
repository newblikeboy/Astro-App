"""Record original PDF paragraph/heading boundaries without changing any source text."""
from pathlib import Path
import gzip
import hashlib
import json
import re
import struct
import pymupdf

ROOT = Path(__file__).resolve().parents[1]
# Reviewed topic-heading paragraph indices in the supplied, hash-verified PDF.
HEADINGS = {
    "मूलांक:1": [6,13,19,25,33,38,45,57,69,79],
    "मूलांक:2": [7,14,27,39,52,57,65,77,86,95,109],
    "मूलांक:3": [17,31,51,69,91,111,127,149,159,179,199,227],
    "मूलांक:4": [10,22,32,46,56,65,75,84,93,103,112,124],
    "मूलांक:5": [13,21,32,41,55,67,77,86,93,102,115,129],
    "मूलांक:6": [12,20,31,41,49,63,68,78,85,91,100,109,122],
    "मूलांक:7": [11,20,32,39,52,60,69,77,82,91,100,109,124],
    "मूलांक:8": [6,15,24,34,40,51,63,76,87,94],
    "मूलांक:9": [10,22,31,41,51,61,70,76,86,99,108],
    "भाग्यांक:1": [13,25,36,49,61,73,80,92,101,112,125],
    "भाग्यांक:2": [12,21,29,40,56,65,83,97,109,118,130],
    "भाग्यांक:4": [12,21,32,44,52,62,69,75,84,92,103,113],
    "भाग्यांक:5": [12,22,30,43,50,60,72,78,88,96,106,119],
    "भाग्यांक:6": [5,13,22,31,40,47,59,68,78,83,95],
    "भाग्यांक:7": [9,18,29,41,50,58,66,72,85,94,105,112],
    "भाग्यांक:8": [11,20,31,38,49,59,67,73,84,93,101,108,117],
    "भाग्यांक:9": [12,24,35,45,53,61,68,82,93,98,106,113,125],
}


def normalize(text):
    """Use the catalog's existing matra-spacing repair solely to align PDF blocks."""
    return re.sub(r'(?<=[\u0900-\u097f]) +(?=[\u0900-\u0903\u093a-\u094f\u0951-\u0957\u0962-\u0963])', '', text)


def fingerprint(text):
    """Fingerprint UTF-16 units so the browser applies formatting only to matching source text."""
    units = text.encode('utf-16-le')
    result = 2166136261
    for (unit,) in struct.iter_unpack('<H', units):
        result = ((result ^ unit) * 16777619) & 0xffffffff
    return f'{result:08x}'


def main():
    """Verify the source PDF and export contiguous, lossless formatting ranges for all 17 number articles."""
    raw_catalog = gzip.decompress((ROOT / 'internal/gurujitext/data/catalog.json.gz').read_bytes())
    catalog = json.loads(raw_catalog)
    source = catalog['sources'][0]
    pdf_path = ROOT / 'references/guruji/2026-11-24' / source['file']
    assert hashlib.sha256(pdf_path.read_bytes()).hexdigest() == source['sha256']
    source_blocks = []
    raw = ''
    with pymupdf.open(pdf_path) as pdf:
        for page_number, page in enumerate(pdf, 1):
            page_text = ''
            for block in page.get_text('blocks'):
                text = normalize(block[4])
                start = len(raw) + len(page_text)
                source_blocks.append({'start': start, 'end': start + len(text), 'page': page_number})
                page_text += text
            assert page_text == normalize(page.get_text()), page_number
            raw += page_text
    layouts = {}
    audit = []
    for category in ['मूलांक', 'भाग्यांक']:
        for key, passages in catalog[category].items():
            assert len(passages) == 1
            article = passages[0]
            text = article['text']
            start = raw.index(text)
            end = start + len(text)
            paragraphs = []
            identity = f'{category}:{key}'
            for block in source_blocks:
                if block['start'] < end and block['end'] > start:
                    a = max(start, block['start']) - start
                    b = min(end, block['end']) - start
                    paragraphs.append({'start': a, 'end': b, 'page': block['page'], 'kind': 'paragraph'})
            assert ''.join(text[p['start']:p['end']] for p in paragraphs) == text
            for index, paragraph in enumerate(paragraphs):
                content = text[paragraph['start']:paragraph['end']].strip()
                if index == 0:
                    paragraph['kind'] = 'title'
                elif category == 'मूलांक' and int(key) <= 7 and index == 1:
                    paragraph['kind'] = 'subtitle'
                elif index in HEADINGS[identity]:
                    paragraph['kind'] = 'heading'
                elif (identity == 'मूलांक:8' and index in [64,67,71]) or (identity == 'मूलांक:9' and index in [87,90,94]):
                    paragraph['kind'] = 'subheading'
                elif not content:
                    paragraph['kind'] = 'space'
                elif re.fullmatch(r'[=\-\s]+', content):
                    paragraph['kind'] = 'divider'
                elif re.match(r'^\*?\*?\d.*[=→]', content):
                    paragraph['kind'] = 'formula'
                elif re.fullmatch(r'“[\s\S]+”', content):
                    paragraph['kind'] = 'quote'
            merged = []
            for paragraph in paragraphs:
                previous = merged[-1] if merged else None
                previous_text = text[previous['start']:previous['end']].strip() if previous else ''
                same_paragraph = previous and previous['kind'] == 'paragraph' and paragraph['kind'] == 'paragraph' and previous['page'] != paragraph['page'] and not re.search(r'[।.!?…—:”]$', previous_text)
                open_emphasis = previous and previous['kind'] == 'paragraph' and paragraph['kind'] in ['paragraph','space'] and previous_text.count('**') % 2 == 1
                if same_paragraph or open_emphasis:
                    previous['end'] = paragraph['end']
                    previous['page'] = paragraph['page']
                else:
                    merged.append(dict(paragraph))
            offsets = [0]
            for character in text:
                offsets.append(offsets[-1] + len(character.encode('utf-16-le')) // 2)
            assert merged[0]['start'] == 0 and merged[-1]['end'] == len(text)
            assert all(a['end'] == b['start'] for a,b in zip(merged,merged[1:]))
            layouts[identity] = {'hash': fingerprint(text), 'length': offsets[-1], 'blocks': [[offsets[p['end']], p['kind']] for p in merged]}
            audit.append({'category': category, 'key': key, 'source_pages': article['pages'], 'paragraphs': len(merged), 'headings': [text[p['start']:p['end']].strip() for p in merged if p['kind'] in ['title','heading','subheading']], 'exact_reconstruction': True})
    assert len(layouts) == len(HEADINGS) == 17
    target = ROOT / 'web/src/data/guruji-numerology-layout.json'
    target.parent.mkdir(exist_ok=True)
    target.write_text(json.dumps({'source_sha256': source['sha256'], 'layouts': layouts}, ensure_ascii=False, separators=(',', ':')) + '\n', encoding='utf-8')
    audit_path = ROOT / 'references/guruji/2026-11-24/numerology-layout-validation.json'
    audit_path.write_text(json.dumps({'source_file': source['file'], 'source_sha256': source['sha256'], 'catalog_sha256': hashlib.sha256(raw_catalog).hexdigest(), 'word_changes': 0, 'articles': audit}, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    print(json.dumps({'articles': len(layouts), 'blocks': sum(len(layout['blocks']) for layout in layouts.values()), 'exact_reconstruction': True, 'word_changes': 0}))


if __name__ == '__main__':
    main()
