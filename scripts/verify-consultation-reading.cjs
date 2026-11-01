// Pure interpretation checks use retained reports; no server, account, booking or order writes.
const fs=require('node:fs');
const assert=require('node:assert/strict');
let consultationParagraphs,consultationConclusions;
const output='tmp/consultation-reading';
const reports=['tmp/consultation-reading/purnia-report.json','tmp/astrosage-faladesh-audit/darbhanga-detailed-v4.json'].map(file=>JSON.parse(fs.readFileSync(file,'utf8')));

/** paragraphs computes a reader view from one retained subject without altering the fixture. */
function paragraphs(report,category,language='hi'){
 const analysis=report.guruji_faladesh.analysis;
 return consultationParagraphs(analysis.topics.find(topic=>topic.category===category),analysis,report,language);
}

/** verifyConsultationReading checks General's separate scope, detailed subject findings, provenance, period isolation and fail-closed behavior. */
async function verifyConsultationReading(){
 ({consultationParagraphs}=await import('../web/src/consultation-reading.ts'));
 ({consultationConclusions}=await import('../web/src/consultation-copy.ts'));
 const {readingParagraphs,conclusionForCategory}=await import('../web/src/faladesh-reading.ts');
 const texts=[];
 for(const report of reports){
  const before=JSON.stringify(report);
  for(const language of ['hi','en'])for(const topic of report.guruji_faladesh.analysis.topics){
   const result=paragraphs(report,topic.category,language);
   assert(result&&result.length>=3,`${report.birth.date}/${language}/${topic.category}: missing detailed reading`);
   assert.equal(result.at(-1).scope,'action');
   assert(result.every(paragraph=>!/(?:लग्न|मारक|कारक ग्रह|भाव-संबंध|वर्ग-कुंडली|राशि-कुंडली|\bD\d+\b|ascendant|\bKarak\b|\bMarak\b|combust|divisional)/iu.test(paragraph.text)),`${language}/${topic.category}: technical explanation leaked`);
   assert(!result.some(paragraph=>paragraph.text.includes('एक समय में अपनी सबसे जरूरी समस्या चुनकर')));
   assert(result[0].text.length>60,'incomplete conclusion');
   const ids=new Set(topic.evidence.map(item=>item.id));
   for(const paragraph of result)for(const id of paragraph.evidenceIds)assert(ids.has(id),`${topic.category}: fabricated cause ${id}`);
   texts.push({birth:report.birth.date,category:topic.category,language,paragraphs:result});
  }
  assert.equal(JSON.stringify(report),before,'retained calculations or historical data changed');
 }
 const purnia=reports[0],raja=reports[1];
 const general=paragraphs(purnia,'general');
 for(const report of reports)for(const language of ['hi','en']){
  const reading=paragraphs(report,'general',language),text=reading.map(item=>item.text).join(' ');
  assert.equal(reading.length,4,'General must have its own four connected paragraphs');
  assert(!reading.some(item=>item.scope==='period'),'General contains the Dasha forecast');
  assert(!/(?:करियर|कमाई|बचत|भुगतान|ग्राहक|जीवनसाथी|महादशा|अंतर्दशा|career|earnings|savings|payment|clients|spouse|Mahadasha|Antardasha)/iu.test(text),'General repeats dedicated subject forecasts');
  for(const category of ['personality','career','finance','relationships','dasha']){
   const detail=paragraphs(report,category,language).map(item=>item.text).join(' ');
   for(const paragraph of reading)assert(!detail.includes(paragraph.text),'General copied a dedicated subject paragraph');
  }
 }
 assert(general[0].text.includes('अपनी सोच'),'Purnia personal direction missing');
 assert(general[2].text.includes('भरोसा'),'personal direction and Moon need not combined');
 assert.notEqual(general[0].text,paragraphs(raja,'general')[0].text,'different personal directions collapsed into one introduction');
 assert(paragraphs(raja,'general')[2].text.includes('जिम्मेदार'),'retained Moon/Saturn balance missing');
 assert(paragraphs(purnia,'finance').some(item=>item.text.includes('खर्च')),'detailed expense pattern removed from Finance');
 assert.equal(paragraphs(purnia,'finance').filter(item=>item.scope==='period').length,0,'unrelated running lords received finance timing');
 assert(paragraphs(purnia,'dasha').some(item=>item.text.includes('सीखने')),'current period lost concrete life subjects');
 const expired=structuredClone(purnia);
 expired.dasha.as_of=expired.dasha.periods.find(item=>item.lord===expired.dasha.current_lord).end;
 assert(!paragraphs(expired,'career').some(item=>item.scope==='period'),'expired current period was presented as active');
 assert.deepEqual(paragraphs(expired,'general'),general,'period boundary rewrote the General reading');
 const missing=structuredClone(purnia);
 missing.guruji_faladesh.analysis.topics.find(item=>item.category==='finance').evidence=[];
 assert.equal(paragraphs(missing,'finance'),undefined,'unattributed theme was interpreted');
 const mismatch=structuredClone(purnia);
 mismatch.chart.planets.find(item=>item.name==='Saturn').house=10;
 assert.equal(paragraphs(mismatch,'general'),undefined,'Chalit or contradictory house accepted as D1');
 const unknown=structuredClone(purnia);unknown.birth.unknown_time=true;
 assert.equal(paragraphs(unknown,'general'),undefined,'unknown time received a new house narrative');
 const historical=structuredClone(purnia);historical.guruji_faladesh.analysis.version='m4y-cross-factor-faladesh-v3';
 assert.equal(paragraphs(historical,'general'),undefined,'unsupported snapshot acquired newly generated interpretation');
 const retainedGeneral=historical.guruji_faladesh.analysis.topics.find(item=>item.category==='general');
 const fallback=readingParagraphs(retainedGeneral,'hi',{analysis:historical.guruji_faladesh.analysis,report:historical});
 assert(!fallback.some(item=>item.scope==='period'||/(?:करियर|कमाई|बचत|ग्राहक|महादशा|अंतर्दशा)/u.test(item.text)),'legacy General exposes dedicated subjects');
 assert.deepEqual(conclusionForCategory(purnia.guruji_faladesh,purnia,'general').anchors.map(item=>item.reading.planet),['Mars','Moon'],'legacy General includes work or wealth anchors');
 const focused=structuredClone(purnia);
 focused.guruji_faladesh.analysis.topics=focused.guruji_faladesh.analysis.topics.filter(item=>['general','personality'].includes(item.category));
 assert.deepEqual(paragraphs(focused,'general'),general,'General still depends on work, money or relationship forecasts');
 const missingMoon=structuredClone(purnia);
 missingMoon.guruji_faladesh.analysis.topics.find(item=>item.category==='general').evidence=missingMoon.guruji_faladesh.analysis.topics.find(item=>item.category==='general').evidence.filter(item=>item.planet!=='Moon');
 assert.equal(paragraphs(missingMoon,'general'),undefined,'an unattributed underlying need was invented');
 for(const language of ['hi','en']){
  const openings=new Set();
  for(let house=1;house<=12;house++){
   // Synthetic D1 destination scenarios isolate editorial selection; these are not astronomical fixtures.
   const scenario=structuredClone(purnia);
   const anchor=scenario.chart.planets.find(item=>item.name==='Mars');
   anchor.house=house;anchor.rashi_number=house;
   const personal=scenario.guruji_faladesh.analysis.topics.find(item=>item.category==='personality');
   const root=personal.findings.find(item=>item.kind==='principal-lord');
   root.id=`lord-personality-1-in-${house}`;
   root.evidence_ids=root.evidence_ids.map(id=>id==='placement:Mars:5'?`placement:Mars:${house}`:id);
   for(const topic of scenario.guruji_faladesh.analysis.topics)for(const fact of topic.evidence){
    if(fact.kind==='placement'&&fact.planet==='Mars'){fact.house=house;fact.id=`placement:Mars:${house}`;}
   }
   const reading=paragraphs(scenario,'general',language);
   assert(reading&&reading.length===4,`${language}/${house}: life-direction coverage missing`);
   openings.add(reading[0].text);
  }
  assert.equal(openings.size,12,'different D1 destinations collapsed into one direction');
 }
 const baselinePath='tmp/general-scope/subject-baseline.json';
 if(fs.existsSync(baselinePath))assert.deepEqual(texts.filter(item=>item.category!=='general'&&item.language==='en'),JSON.parse(fs.readFileSync(baselinePath,'utf8')).filter(item=>item.language==='en'),'a dedicated English category changed during Hindi editing');
 for(const [subject,rows] of Object.entries(consultationConclusions))assert.equal(rows.length,12,`${subject}: incomplete D1 destination coverage`);
 fs.writeFileSync(`${output}/readings.json`,JSON.stringify(texts,null,2));
 console.log(JSON.stringify({passed:true,people:2,languages:2,categories:9,checks:['separate General scope','distinct life directions','detailed subjects preserved','retained evidence','no technical prose','period boundaries','no unrelated timing','unknown time','D1/Chalit isolation','immutable native data']}));
}
verifyConsultationReading().catch(error=>{console.error(error);process.exitCode=1;});
