import type { FaladeshSynthesisTopic, GurujiFaladeshSnapshot, GurujiReading } from "./guruji-faladesh-types";
import type { Report } from "./types";
import type { FaladeshSynthesis } from "./guruji-faladesh-types";
import { consultationParagraphs, plainFindingText } from "./consultation-reading.ts";
import { simpleFaladeshHindi } from "./faladesh-hindi.ts";

const signLords = ["Mars", "Venus", "Mercury", "Moon", "Sun", "Mercury", "Venus", "Mars", "Jupiter", "Saturn", "Saturn", "Jupiter"];
export { currentFaladeshVersion } from "./guruji-faladesh-types.ts";

/** readingParagraphs presents connected subject conclusions from checked retained facts, with the original prose fallback for unsupported or incomplete snapshots. */
export function readingParagraphs(topic:FaladeshSynthesisTopic,language:"hi"|"en",context?:{analysis:FaladeshSynthesis;report:Report}) {
  if(context?.analysis.reader_version==="m4y-reader-paragraphs-v1"&&topic.reader?.length){
    const evidence=new Set(topic.evidence.map(item=>item.id));
    if(topic.reader.every(item=>item.text.hi.trim()&&item.text.en.trim()&&item.evidence_ids.every(id=>evidence.has(id))))
      return topic.reader.map(item=>({id:item.id,scope:item.scope,leadIn:"",text:item.text[language],evidenceIds:item.evidence_ids}));
  }
  const consultation=context&&consultationParagraphs(topic,context.analysis,context.report,language);
  if(consultation)return language==="hi"?consultation.map(paragraph=>({...paragraph,text:simpleFaladeshHindi(paragraph.text)})):consultation;
  if(topic.category==="general"){
    // A retained General overview may contain other subjects; keep only its personal-direction finding in the legacy view.
    const personal=topic.findings?.filter(finding=>/^(?:overall-)?(?:lord-personality-1-|occupants-personality-1$)/u.test(finding.id));
    const text=personal?.map(finding=>plainFindingText(finding.text[language])).filter(Boolean).join(" ");
    return [{id:"general-retained-direction",scope:"natal",leadIn:"",text:language==="hi"?simpleFaladeshHindi(text||"इस रिपोर्ट में सामान्य फलादेश की अलग जानकारी नहीं है।"):text||"A separate life-direction reading is unavailable in this report.",evidenceIds:personal?.flatMap(finding=>finding.evidence_ids)||[]}];
  }
  const seen=new Set<string>();
  const sources=topic.findings?.length?topic.findings.map(finding=>({id:finding.id,scope:finding.scope,leadIn:"",text:[plainFindingText(finding.text[language]),plainFindingText(finding.qualification[language])].filter(Boolean).join(" "),evidenceIds:finding.evidence_ids})):[{id:"conclusion",scope:"natal",leadIn:"",text:plainFindingText(topic.conclusion[language]),evidenceIds:[] as string[]}];
  return [...sources,{id:"guidance",scope:"action",leadIn:"",text:topic.potential[language],evidenceIds:[] as string[]}].flatMap(source=>{
    const sentences=(source.text.match(/[^.!?।]+[.!?।]*/gu)||[]).map(sentence=>sentence.trim().replace(/\s+/gu," ")).filter(sentence=>{
      if(!sentence||seen.has(sentence))return false;
      seen.add(sentence);
      return true;
    });
    return sentences.length?[{...source,text:language==="hi"?simpleFaladeshHindi(sentences.join(" ")):sentences.join(" ")}]:[];
  });
}

/** normalizeReadingCategory routes historical subject aliases without changing the saved interpretation itself. */
export function normalizeReadingCategory(category?: string): string {
  const key = category?.trim().toLowerCase() || "general";
  return ({marriage:"relationships",health:"wellbeing",debt:"finance"} as Record<string,string>)[key] || key;
}
export const readingTopics = [
  { id: "personality", title: {hi:"आपका स्वभाव",en:"Your nature"}, subtitle:{hi:"सोच, अपनी बात कहना और कोशिश करना",en:"Outlook, expression and initiative"}, houses:[1,3,5], primary:1 },
  { id: "education", title: {hi:"पढ़ाई और सीखना",en:"Learning & direction"}, subtitle:{hi:"पढ़ाई, हुनर और सही सलाह",en:"Study, skills and guidance"}, houses:[4,5,9], primary:5 },
  { id: "career", title: {hi:"काम और करियर",en:"Work & career"}, subtitle:{hi:"काम, जिम्मेदारी और काम करने का तरीका",en:"Responsibility, direction and working style"}, houses:[6,10], primary:10 },
  { id: "finance", title: {hi:"पैसे और बचत",en:"Money & resources"}, subtitle:{hi:"कमाई, बचत और आगे के लक्ष्य",en:"Income, saving and long-term aims"}, houses:[2,11], primary:2 },
  { id: "relationships", title: {hi:"प्यार और साझेदारी",en:"Love & partnership"}, subtitle:{hi:"साथ, बातचीत और मिलकर लिए गए फैसले",en:"Closeness, communication and shared decisions"}, houses:[5,7], primary:7 },
  { id: "family", title: {hi:"घर और परिवार",en:"Home & family"}, subtitle:{hi:"अपनापन, बातचीत और घर की जिम्मेदारी",en:"Belonging and family responsibilities"}, houses:[2,3,4], primary:4 },
  { id: "wellbeing", title: {hi:"रोज की आदतें और आराम",en:"Balance & routine"}, subtitle:{hi:"अपना ध्यान, आराम और मन का सुकून",en:"Rest, reflection and everyday balance"}, houses:[1,6,8,12], primary:1 },
];
export type ReadingConnection = { reading: GurujiReading; ownedHouses: number[]; occupiesTopic: boolean; primaryLord: boolean };
/** consultationForTopic groups at most two saved natal passages into one subject reading, without scoring strength or selecting a shubh/ashubh branch. */
export function consultationForTopic(data:GurujiFaladeshSnapshot,report:Report,category:string){
  const connections=connectedReadings(data,report,category),topic=readingTopics.find(item=>item.id===category);
  if(!topic||!connections.length)return undefined;
  const lead=connections.find(item=>item.primaryLord)||connections[0];
  const companion=connections.find(item=>item!==lead&&item.reading.house===topic.primary)
    ||connections.find(item=>item!==lead&&item.occupiesTopic);
  const anchors=[lead,...(companion?[companion]:[])];
  return {anchors,connections,
    meaning:anchors.map(item=>item.reading.narrative?.meaning||item.reading.summary),
    support:anchors.flatMap(item=>item.reading.narrative?[item.reading.narrative.support]:[]),
    caution:anchors.flatMap(item=>item.reading.narrative?[item.reading.narrative.caution]:[])};
}

/** conclusionForCategory selects saved natal meanings for an overall or subject conclusion; it does not estimate probabilities or infer outcomes from role/dignity labels. */
export function conclusionForCategory(data:GurujiFaladeshSnapshot,report:Report,category:string){
  if(category!=="general"&&category!=="dasha")return consultationForTopic(data,report,category);
  const connections=connectedReadings(data,report,category);
  // General keeps personal direction and the mind; dedicated subjects retain work, resources and timing.
  const planets=category==="general"?[lordOfHouse(data.lagna,1),"Moon"]:[report.dasha?.current_lord,report.dasha?.current_sub_lord];
  const anchors=[...new Set(planets)].flatMap(planet=>{const item=connections.find(connection=>connection.reading.planet===planet);return item?[item]:[];});
  if(!anchors.length)return undefined;
  return {anchors,connections,meaning:anchors.map(item=>item.reading.narrative?.meaning||item.reading.summary),
    support:anchors.flatMap(item=>item.reading.narrative?[item.reading.narrative.support]:[]),
    caution:anchors.flatMap(item=>item.reading.narrative?[item.reading.narrative.caution]:[])};
}

/** savedPeriodContext finds dated major/sub periods at the report's reference instant; it never substitutes today's date or trusts a stale current_path. */
export function savedPeriodContext(report:Report){
  const dasha=report.dasha,at=Date.parse(dasha?.as_of||"");
  if(!dasha||!Number.isFinite(at))return undefined;
  const major=dasha.periods.find(item=>item.lord===dasha.current_lord&&Date.parse(item.start)<=at&&at<Date.parse(item.end));
  const sub=major?.antardashas?.find(item=>item.lord===dasha.current_sub_lord&&Date.parse(item.start)<=at&&at<Date.parse(item.end));
  return {major,sub};
}

/** lordOfHouse resolves ordinary D1 sign ownership without assigning signs to the nodes. */
export function lordOfHouse(lagna: number, house: number): string { return signLords[(lagna + house - 2) % 12]; }

/** ownedHouses lists factual D1 ownership separately from favourable/adverse classifications. */
export function ownedHouses(lagna: number, planet: string): number[] {
  return Array.from({length:12},(_,index)=>index+1).filter(house=>lordOfHouse(lagna,house)===planet);
}

/** connectedReadings includes house owners and occupants, ordered by relevance rather than an outcome score. */
export function connectedReadings(data: GurujiFaladeshSnapshot, report: Report, category: string): ReadingConnection[] {
  const topic=readingTopics.find(item=>item.id===category), houses=topic?.houses||[];
  return data.readings.map(reading=>({reading,ownedHouses:ownedHouses(data.lagna,reading.planet).filter(house=>houses.includes(house)),occupiesTopic:houses.includes(reading.house),primaryLord:!!topic&&reading.planet===lordOfHouse(data.lagna,topic.primary)}))
    .filter(item=>category==="general"||(category==="dasha"?[report.dasha?.current_lord,report.dasha?.current_sub_lord].includes(item.reading.planet):item.occupiesTopic||item.ownedHouses.length>0))
    .sort((a,b)=>Number(b.primaryLord)-Number(a.primaryLord)||Number(b.occupiesTopic)-Number(a.occupiesTopic));
}
