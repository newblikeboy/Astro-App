// Build first-party bilingual copy for the canonical Go reading contract.
// Legacy TypeScript readers remain for old snapshots; new reports store final
// paragraphs once so the API, browser and PDF cannot pick different prose.
const fs=require('node:fs');
const vm=require('node:vm');
const {stripTypeScriptTypes}=require('node:module');
const parser=require('../web/node_modules/@babel/parser');

/** moduleCopy evaluates our own data-only TypeScript module in an isolated
 * context after removing type syntax; it never loads third-party chart code. */
function moduleCopy(path,names){
 const source=stripTypeScriptTypes(fs.readFileSync(path,'utf8')).replace(/export\s+(?=const|function)/gu,'');
 const sandbox={module:{exports:{}}};
 vm.runInNewContext(source+`\nmodule.exports={${names.join(',')}};`,sandbox,{filename:path});
 return sandbox.module.exports;
}

/** pair converts an editorial [Hindi,English] pair into the backend wording DTO. */
function pair(value){return {hi:value[0],en:value[1]};}

/** capacityCopy extracts the existing subject-specific care/support literals
 * rather than introducing a second manually maintained Hindi dictionary. */
function capacityCopy(){
 const source=fs.readFileSync('web/src/consultation-reading.ts','utf8');
 const ast=parser.parse(source,{sourceType:'module',plugins:['typescript']});
 const fn=ast.program.body.find(node=>node.type==='FunctionDeclaration'&&node.id.name==='capacityParagraph');
 const result={};
 for(const statement of fn.body.body){
  if(statement.type!=='VariableDeclaration')continue;
  for(const decl of statement.declarations){
   if(!['care','strength'].includes(decl.id.name))continue;
   const literal=vm.runInNewContext('('+source.slice(decl.init.start,decl.init.end)+')');
   result[decl.id.name]=Object.fromEntries(Object.entries(literal).map(([key,value])=>[key,pair(value)]));
  }
 }
 return result;
}

/** generalCase executes only our existing General editorial grouping against
 * synthetic placements to retain its exact separate scope and simple language. */
function generalCase(runtime,house,moonHouse,saturn,care){
 const positions=[{name:'Mars',house},{name:'Moon',house:moonHouse},{name:'Saturn',house:saturn?house:(house%12)+1}];
 const evidence=positions.map(p=>({id:`placement:${p.name}:${p.house}`,kind:'placement',planet:p.name,house:p.house,direction:'context'}));
 if(care)evidence.push({id:'combustion:Mars:'+house,kind:'combustion',planet:'Mars',house,direction:'attention'});
 const report={chart:{lagna:{rashi_number:1},planets:positions}};
 const topic={evidence};
 const root={evidence_ids:[`placement:Mars:${house}`]};
 return ['hi','en'].map(language=>runtime.generalLifeReading(report,topic,root,language));
}

/** buildReaderCopy generates and optionally checks the source-controlled copy
 * artifact consumed at calculation time; it creates no customer reports. */
function buildReaderCopy(){
 const general=moduleCopy('web/src/general-reading.ts',['direction','axes','needs','strength','priority','generalLifeReading']);
 const consultation=moduleCopy('web/src/consultation-copy.ts',['consultationConclusions','periodActivities']);
 const balances={};
 for(let house=1;house<=12;house++)for(let moon=1;moon<=12;moon++){
  const rows=generalCase(general,house,moon,false,false);
  balances[general.axes[house-1]+'/'+general.axes[moon-1]]={hi:rows[0][2].text,en:rows[1][2].text};
 }
 const saturn=generalCase(general,1,1,true,false),care=generalCase(general,1,1,false,true);
 const output={
  version:'m4y-reader-copy-v1',
  headlines:Object.fromEntries(Object.entries(consultation.consultationConclusions).map(([key,values])=>[key,values.map(pair)])),
  activities:consultation.periodActivities.map(pair),
  capacity:capacityCopy(),
  general:{direction:general.direction.map(pair),axes:general.axes,needs:Object.fromEntries(Object.entries(general.needs).map(([key,value])=>[key,pair(value)])),strength:Object.fromEntries(Object.entries(general.strength).map(([key,value])=>[key,pair(value)])),priority:Object.fromEntries(Object.entries(general.priority).map(([key,value])=>[key,pair(value)])),balances,saturn_balance:{hi:saturn[0][2].text,en:saturn[1][2].text},saturn_priority:{hi:saturn[0][3].text,en:saturn[1][3].text},capacity_care:{hi:care[0][1].text.slice(general.strength.initiative[0].length),en:care[1][1].text.slice(general.strength.initiative[1].length)}},
  simple_hindi:JSON.parse(fs.readFileSync('web/src/locales/faladesh-simple-hi.json','utf8')),
 };
 const path='internal/platform/data/reader-copy.json',text=JSON.stringify(output,null,2)+'\n';
 if(process.argv.includes('--check')){
  if(fs.readFileSync(path,'utf8')!==text)throw new Error('Reader copy is stale; run node scripts/build-reader-copy.cjs');
 }else fs.writeFileSync(path,text);
 console.log('Canonical reader copy verified.');
}
buildReaderCopy();
