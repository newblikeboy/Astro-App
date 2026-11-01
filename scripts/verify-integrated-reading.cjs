// Pure canonical-contract checks; fixtures come from opt-in native tests, never
// customer persistence or a renderer's own regenerated interpretation.
const fs=require('node:fs');
const path=require('node:path');
const assert=require('node:assert/strict');

/** verifyIntegratedReading compares website consumption with retained bilingual
 * API prose, checks General's scope and rejects unattributed public paragraphs. */
async function verifyIntegratedReading(){
  const {readingParagraphs,currentFaladeshVersion}=await import('../web/src/faladesh-reading.ts');
  const directory=process.env.ADVANCED_QA_DIR||'tmp/advanced-qa';
  let paragraphs=0;
  for(const person of ['purnia','raja']){
    const report=JSON.parse(fs.readFileSync(path.join(directory,person,'report.json'),'utf8'));
    const before=JSON.stringify(report),analysis=report.guruji_faladesh.analysis;
    assert.equal(analysis.version,'m4y-cross-factor-faladesh-v5');
    assert.equal(analysis.version,currentFaladeshVersion,'native and browser reader versions diverged');
    for(const language of ['hi','en'])for(const topic of analysis.topics){
      const rendered=readingParagraphs(topic,language,{analysis,report});
      assert.equal(rendered.length,topic.reader.length);
      for(let i=0;i<rendered.length;i++){
        assert.equal(rendered[i].text,topic.reader[i].text[language],'website rewrote retained prose');
        assert.deepEqual(rendered[i].evidenceIds,topic.reader[i].evidence_ids);
        assert(rendered[i].evidenceIds.length,'unattributed paragraph');
        assert(!/undefined|Invalid Date|\?{3,}/u.test(rendered[i].text));
        assert(!/(?:लग्न|मारक|कारक ग्रह|भाव-संबंध|वर्ग-कुंडली|\bKarak\b|\bMarak\b|ascendant|combust|divisional)/iu.test(rendered[i].text));
      }
      if(topic.category==='general'){
        assert.equal(rendered.length,4);
        assert(rendered.every(item=>item.scope!=='period'));
        assert(!/(?:करियर|कमाई|बचत|भुगतान|ग्राहक|जीवनसाथी|महादशा|अंतर्दशा|career|earnings|savings|payment|clients|spouse|Mahadasha|Antardasha)/iu.test(rendered.map(item=>item.text).join(' ')));
      }
      paragraphs+=rendered.length;
    }
    assert.equal(JSON.stringify(report),before,'renderer changed immutable snapshot');
  }
  console.log(JSON.stringify({passed:true,people:2,languages:2,topics:9,paragraphs,sharedSnapshot:true}));
}
verifyIntegratedReading().catch(error=>{console.error(error);process.exitCode=1;});
