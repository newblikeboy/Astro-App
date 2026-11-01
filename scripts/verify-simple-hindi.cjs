// Language-only checks compare retained output before this edit with the same reports after editing.
const fs=require('node:fs');
const assert=require('node:assert/strict');
const before=JSON.parse(fs.readFileSync('tmp/simple-hindi/before.json','utf8'));
const files=['tmp/consultation-reading/purnia-report.json','tmp/astrosage-faladesh-audit/darbhanga-detailed-v4.json'];

/** metadata omits only display wording so a language edit cannot silently alter finding identities, timing scopes or causes. */
function metadata(paragraphs){return paragraphs.map(({text,...rest})=>rest);}

/** sentenceLengths measures sentence length for the editorial review without using readability as a substitute for checking meaning. */
function sentenceLengths(rows){return rows.flatMap(row=>row.paragraphs.flatMap(p=>(p.text.match(/[^.!?।]+[.!?।]*/gu)||[]).map(sentence=>sentence.trim().split(/\s+/u).length)));}

/** verifySimpleHindi checks preserved English, finding metadata and input reports, then records shorter Hindi for all nine topics. */
async function verifySimpleHindi(){
 const {readingParagraphs}=await import('../web/src/faladesh-reading.ts');
 const after=[];
 for(const file of files){
  const report=JSON.parse(fs.readFileSync(file,'utf8')),unchanged=JSON.stringify(report),analysis=report.guruji_faladesh.analysis;
  for(const language of ['hi','en'])for(const topic of analysis.topics){
   const paragraphs=readingParagraphs(topic,language,{analysis,report});
   const prior=before.find(row=>row.birth===report.birth.date&&row.category===topic.category&&row.language===language);
   assert(prior,'missing before/after comparison');
   assert.deepEqual(metadata(paragraphs),metadata(prior.paragraphs),`${report.birth.date}/${language}/${topic.category}: findings changed`);
   if(language==='en')assert.deepEqual(paragraphs,prior.paragraphs,'English reading changed');
   else{
    const text=paragraphs.map(p=>p.text).join(' ');
    assert.notEqual(text,prior.paragraphs.map(p=>p.text).join(' '),'Hindi was not reviewed');
    assert(!/(?:जीवन-दिशा|निरंतरता|अभिव्यक्ति|प्रतिफल|संसाधन|परिप्रेक्ष्य|गुंजाइश|प्राथमिकता|व्यक्तिगत परामर्श)/u.test(text),`${topic.category}: difficult editorial wording remains`);
    assert(!/undefined|\?{3,}/u.test(text),'broken Hindi');
   }
   after.push({birth:report.birth.date,category:topic.category,language,paragraphs});
  }
  assert.equal(JSON.stringify(report),unchanged,'native or saved report changed');
 }
 const oldLengths=sentenceLengths(before.filter(row=>row.language==='hi')),newLengths=sentenceLengths(after.filter(row=>row.language==='hi'));
 const oldAverage=oldLengths.reduce((a,b)=>a+b,0)/oldLengths.length,newAverage=newLengths.reduce((a,b)=>a+b,0)/newLengths.length;
 assert(newAverage<oldAverage,'sentences were not made shorter');
 fs.writeFileSync('tmp/simple-hindi/after.json',JSON.stringify(after,null,2));
 for(const row of after.filter(row=>row.birth==='1996-11-26'&&row.language==='hi'&&['general','career','finance','dasha'].includes(row.category)))fs.writeFileSync(`tmp/simple-hindi/${row.category}.txt`,row.paragraphs.map(p=>p.text).join('\n\n'));
 console.log(JSON.stringify({passed:true,people:2,categories:9,hindiReadings:18,englishUnchanged:true,findingMetadataUnchanged:true,averageSentenceWords:{before:Number(oldAverage.toFixed(1)),after:Number(newAverage.toFixed(1))}}));
}
verifySimpleHindi().catch(error=>{console.error(error);process.exitCode=1;});
