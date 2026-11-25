"""Index original PDF paragraphs under the existing nine life-area categories."""
from pathlib import Path
import gzip
import hashlib
import importlib.util
import json
import re
import pymupdf

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('number_layout', ROOT / 'scripts/build-numerology-layout.py')
number_layout = importlib.util.module_from_spec(spec)
spec.loader.exec_module(number_layout)
TOPICS = {'general':1,'personality':2,'education':4,'career':8,'finance':16,'relationships':32,'family':64,'wellbeing':128,'dasha':256}
# These are the prior reader's life-area house associations, used only for routing complete original house passages.
HOUSES = {'personality':[1,3,5],'education':[4,5,9],'career':[6,10],'finance':[2,11],'relationships':[5,7],'family':[2,3,4],'wellbeing':[1,6,8,12]}
VOCABULARY = {
    'personality':r'स्वभाव|व्यक्तित्व|गुण|ताकत|कमजोर|कमज़ोर|आत्मविश्वास|आत्मसम्मान|गुप्त पक्ष|छुपा हुआ|छिपा हुआ|व्यवहार|मानसिक प्रवृत्ति|भावनात्मक',
    'education':r'शिक्षा|पढ़ाई|पढाई|अध्ययन|विद्या|बुद्धि|ज्ञान और नेतृत्व',
    'career':r'करियर|व्यवसाय|नौकरी|रोजगार|रोज़गार|कार्यक्षेत्र|कर्मक्षेत्र|पेशा|व्यापार',
    'finance':r'धन|आर्थिक|पैसा|पैसे|संपत्ति|सम्पत्ति|भूमि|कमाई|आय(?:\s|,|$)|कर्ज|ऋण|बचत',
    'relationships':r'प्रेम|विवाह|वैवाहिक|दांपत्य|दाम्पत्य|रिश्त|साझेदारी|दोस्ती|मित्रता',
    'family':r'परिवार|पारिवारिक|संतान|भाई|बहन|माता|पिता|बचपन|घर(?:\s|,|$)',
    'wellbeing':r'स्वास्थ्य|रोग|बीमारी|शरीर|शारीरिक|सेहत|आराम',
    'dasha':r'दशा|अंतर्दशा|अन्तर्दशा',
}


def topic_mask(text):
    """Recognize life-area words in an original section heading without rewriting or interpreting its claims."""
    mask = 0
    for topic, pattern in VOCABULARY.items():
        if re.search(pattern, text):
            mask |= TOPICS[topic]
    return mask or TOPICS['general']


def house_mask(house):
    """Reuse the existing life-area taxonomy for an explicitly supplied house number."""
    return sum(TOPICS[topic] for topic,houses in HOUSES.items() if house in houses) or TOPICS['general']


def heading_kind(text, title):
    """Recognize short source headings, preserving conditional labels and rejecting ordinary body sentences."""
    compact = ' '.join(text.split())
    if not compact:
        return 'space'
    if re.fullmatch(r'[=\-\s]+', compact):
        return 'divider'
    if compact == ' '.join(title.split()):
        return 'title'
    if compact.startswith('##'):
        return 'heading'
    if compact.startswith(('•','“','"','**')) or len(compact)>160 or re.search(r'[।.!…—:;,]$',compact):
        return 'paragraph'
    if re.match(r'^(?:यदि|अगर|लेकिन|क्योंकि|किसी|हालाँकि|हालांकि|वैदिक|पारंपरिक|उदाहरण|महत्वपूर्ण|अर्थ|यही|ऐसे|सामान्यतः|कई बार|कभी|वह |वे |ये |यह |इनके लिए|इसलिए)',compact):
        return 'paragraph'
    if topic_mask(compact)!=TOPICS['general'] or re.search(r'निष्कर्ष|जीवन (?:संदेश|मंत्र|रहस्य|सबक)|भाग्योदय|शुभ|सावधानी|आध्यात्मिक|उपाय|रत्न|भाग्य और|चार चरण|क्या अंतर|अचानक (?:बदलाव|परिवर्तन)',compact):
        return 'heading'
    return 'paragraph'


def main():
    """Verify every PDF and export source-only paragraph offsets and subject routing for all 246 passages."""
    raw_catalog = gzip.decompress((ROOT/'internal/gurujitext/data/catalog.json.gz').read_bytes())
    catalog = json.loads(raw_catalog)
    numerology = json.loads((ROOT/'web/src/data/guruji-numerology-layout.json').read_text(encoding='utf-8'))
    originals = {}
    for index,source in enumerate(catalog['sources']):
        path = ROOT/'references/guruji/2026-11-24'/source['file']
        assert hashlib.sha256(path.read_bytes()).hexdigest()==source['sha256']
        if index==5:
            continue  # The decoded short PDF has one original paragraph per conditional house branch.
        raw=''
        blocks=[]
        with pymupdf.open(path) as pdf:
            for page_number,page in enumerate(pdf,1):
                page_text=''
                for block in page.get_text('blocks'):
                    text=number_layout.normalize(block[4]);start=len(raw)+len(page_text)
                    blocks.append((start,start+len(text),page_number));page_text+=text
                assert page_text==number_layout.normalize(page.get_text())
                raw+=page_text
        originals[index]=(raw,blocks)
    records={}
    audit=[]
    for category,values in catalog.items():
        if category=='sources':
            continue
        for key,articles in values.items():
            for article in articles:
                text=article['text'];source_index=article['source'];paragraphs=[]
                if source_index==0:
                    layout=numerology['layouts'][f'{category}:{key}'];start=0
                    for end,kind in layout['blocks']:
                        # Convert browser offsets back to Python character positions only for the lossless source slice.
                        end_chars=len(text.encode('utf-16-le')[:end*2].decode('utf-16-le'))
                        paragraphs.append({'start':start,'end':end_chars,'kind':kind,'page':0});start=end_chars
                elif source_index==5:
                    end=text.find(article['title'])+len(article['title'])
                    assert end>0
                    paragraphs=[{'start':0,'end':end,'kind':'title','page':article['pages'][0]},{'start':end,'end':len(text),'kind':'paragraph','page':article['pages'][0]}]
                else:
                    raw,blocks=originals[source_index];start=raw.index(text);end=start+len(text)
                    for a,b,page in blocks:
                        if a<end and b>start:
                            left=max(a,start)-start;right=min(b,end)-start
                            paragraphs.append({'start':left,'end':right,'kind':heading_kind(text[left:right],article['title']),'page':page})
                assert ''.join(text[p['start']:p['end']] for p in paragraphs)==text
                base=house_mask(article['house']) if article.get('house') else house_mask(int(key)) if source_index==5 else TOPICS['general']
                remedy=category=='उपाय'
                whole_house=category=='भावफल' and article.get('house') or source_index==5
                current=0 if remedy else base
                merged=[]
                for paragraph in paragraphs:
                    content=text[paragraph['start']:paragraph['end']]
                    if paragraph['kind'] in ['heading','subheading'] and not remedy:
                        current=(base|topic_mask(content)) if whole_house else topic_mask(content)
                    mask=current
                    if re.search(VOCABULARY['dasha'],content):
                        mask|=TOPICS['dasha']
                    item={**paragraph,'mask':mask}
                    previous=merged[-1] if merged else None
                    if previous and previous['kind']=='paragraph' and item['kind']=='paragraph' and previous['page']!=item['page'] and previous['mask']==item['mask'] and not re.search(r'[।.!?…—:”]$',text[previous['start']:previous['end']].strip()):
                        previous['end']=item['end'];previous['page']=item['page']
                    else:
                        merged.append(item)
                offsets=[0]
                for character in text:
                    offsets.append(offsets[-1]+len(character.encode('utf-16-le'))//2)
                fingerprint=number_layout.fingerprint(text)
                identity=f"{category}:{key}:{article['pages'][0]}:{article['pages'][1]}:{fingerprint}"
                assert identity not in records
                records[identity]={'source':source_index,'length':offsets[-1],'remedy':remedy,'parts':[[offsets[p['end']],p['kind'],p['mask']] for p in merged]}
                assert merged[0]['start']==0 and merged[-1]['end']==len(text)
                assert all(a['end']==b['start'] for a,b in zip(merged,merged[1:]))
                audit.append({'category':category,'key':key,'pages':article['pages'],'fingerprint':fingerprint,'parts':len(merged),'topics':[name for name,bit in TOPICS.items() if any(p['mask']&bit for p in merged)],'exact_reconstruction':True})
    assert len(records)==246
    target=ROOT/'web/src/data/guruji-topic-sections.json'
    target.write_text(json.dumps({'source_hashes':[s['sha256'] for s in catalog['sources']],'topic_bits':TOPICS,'houses':HOUSES,'records':records},ensure_ascii=False,separators=(',',':'))+'\n',encoding='utf-8')
    (ROOT/'references/guruji/2026-11-24/topic-index-validation.json').write_text(json.dumps({'catalog_sha256':hashlib.sha256(raw_catalog).hexdigest(),'source_count':7,'passages':len(records),'word_changes':0,'records':audit},ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
    print(json.dumps({'sources':7,'passages':len(records),'topics':list(TOPICS),'word_changes':0,'bytes':target.stat().st_size}))


if __name__=='__main__':
    main()
