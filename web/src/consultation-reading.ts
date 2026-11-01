import type { FaladeshSynthesis, FaladeshSynthesisTopic } from "./guruji-faladesh-types";
import type { Report } from "./types";
import { consultationConclusions, periodActivities, type ConsultationSubject } from "./consultation-copy.ts";
import { generalLifeReading } from "./general-reading.ts";

type Language = "hi" | "en";
type Finding = NonNullable<FaladeshSynthesisTopic["findings"]>[number];
export type ReadingParagraph = {id:string;scope:string;leadIn:string;text:string;evidenceIds:string[]};
type Profile = {subject:ConsultationSubject;topic:FaladeshSynthesisTopic;root:Finding;house:number;planet:string;secondary?:{finding:Finding;house:number}};
const lords = ["Mars","Venus","Mercury","Moon","Sun","Mercury","Venus","Mars","Jupiter","Saturn","Saturn","Jupiter"];
const subjectHouses:Record<ConsultationSubject,number[]> = {personality:[1,3,5],education:[5,4,9],career:[10,6,11],finance:[2,11],relationships:[7,5],family:[4,2,3],wellbeing:[1,6,8,12]};
const planetNames:Record<string,string> = {Sun:"सूर्य",Moon:"चंद्रमा",Mars:"मंगल",Mercury:"बुध",Jupiter:"गुरु",Venus:"शुक्र",Saturn:"शनि",Rahu:"राहु",Ketu:"केतु"};

/** wording selects original bilingual editorial copy, without translating technical evidence into the reading. */
function wording(pair:readonly [string,string],language:Language){return pair[language==="hi"?0:1];}

/** owner uses only the retained D1 ascendant for ordinary sign ownership, leaving node roles and astronomical positions unchanged. */
function owner(report:Report,house:number){return lords[((report.chart?.lagna.rashi_number||0)+house-2)%12];}

/** checkedProfile requires agreement between the retained finding, its factual ledger and the native D1 position before interpreting it. */
function checkedProfile(report:Report,analysis:FaladeshSynthesis,subject:ConsultationSubject):Profile|undefined{
  const topic=analysis.topics.find(item=>item.category===subject);
  const primary=subjectHouses[subject][0],planet=owner(report,primary),position=report.chart?.planets.find(item=>item.name===planet);
  const root=topic?.findings?.find(item=>item.kind==="principal-lord"&&item.id===`lord-${subject}-${primary}-in-${position?.house}`);
  if(!topic||!root||!position||position.house<1||position.house>12)return undefined;
  const fact=topic.evidence.find(item=>item.kind==="placement"&&item.planet===planet&&item.house===position.house&&root.evidence_ids.includes(item.id));
  if(!fact)return undefined;
  const secondary=topic.findings?.flatMap(finding=>{
    if(finding.kind!=="secondary-lord")return [];
    for(const source of subjectHouses[subject].slice(1)){
      const name=owner(report,source),p=report.chart?.planets.find(item=>item.name===name);
      if(p&&finding.id===`lord-${subject}-${source}-in-${p.house}`&&topic.evidence.some(item=>item.kind==="placement"&&item.planet===name&&item.house===p.house&&finding.evidence_ids.includes(item.id)))return [{finding,house:p.house}];
    }
    return [];
  })[0];
  return {subject,topic,root,house:position.house,planet,secondary};
}

/** plainFindingText keeps reader consequences while omitting technical explanation sentences from older retained editorial passages. */
export function plainFindingText(text:string){
  return (text.match(/[^.!?।]+[.!?।]*/gu)||[]).map(sentence=>sentence.trim()).filter(sentence=>!/(?:लग्न|कारक ग्रह|मारक|ग्रह की|ग्रहों|भाव[- ]संबंध|राशि-कुंडली|वर्ग-कुंडली|इस संबंध का फल दूसरे|\bD\d+\b|ascendant|\bKarak\b|\bMarak\b|combust|divisional|natal house|sign placement|needs to be read alongside)/iu.test(sentence)).join(" ");
}

/** afterOpening retains explanatory detail while avoiding a second opening statement of the same theme. */
function afterOpening(text:string){const plain=plainFindingText(text);return plain.replace(/^[^.!?।]+[.!?।]+\s*/u,"").trim()||plain;}

/** subjectConclusion supplies a complete subject-specific result for the checked destination, rather than a good/bad score. */
function subjectConclusion(profile:Profile,language:Language){return wording(consultationConclusions[profile.subject][profile.house-1],language);}

/** activePeriods validates both saved period boundaries and names; browser time never supplies or advances a forecast. */
function activePeriods(report:Report){
  const data=report.dasha,at=Date.parse(data?.as_of||"");
  if(!data||!Number.isFinite(at))return undefined;
  const major=data.periods.find(item=>item.lord===data.current_lord&&Date.parse(item.start)<=at&&at<Date.parse(item.end));
  const sub=major?.antardashas?.find(item=>item.lord===data.current_sub_lord&&Date.parse(item.start)<=at&&at<Date.parse(item.end));
  return major&&sub?{major,sub}:undefined;
}

/** periodHouse selects a subject's actual owned house before occupancy, never treating a broad General connection as a connection to every category. */
function periodHouse(report:Report,name:string,subject?:ConsultationSubject){
  const houses=subject?subjectHouses[subject]:[10,9,6,7,2,11,5,3,4,1,8,12];
  const position=report.chart?.planets.find(item=>item.name===name);
  return houses.find(house=>owner(report,house)===name)||(position&&houses.includes(position.house)?position.house:undefined);
}

/** periodParagraph states concrete themes for a verified current interval without asserting an event, its date, or a favourable outcome. */
function periodParagraph(report:Report,topic:FaladeshSynthesisTopic,language:Language,subject?:ConsultationSubject):ReadingParagraph|undefined{
  const active=activePeriods(report);
  if(!active)return undefined;
  const linked=[...new Set([active.major.lord,active.sub.lord])].flatMap(name=>{
    const house=periodHouse(report,name,subject);
    const fact=topic.evidence.find(item=>item.kind==="placement"&&item.planet===name);
    return house&&fact?[{name,house,id:fact.id}]:[];
  });
  if(!linked.length)return undefined;
  const themes=[...new Set(linked.map(item=>wording(periodActivities[item.house-1],language)))];
  const joined=themes.join(language==="hi"?" और ":" together with ");
  const text=language==="hi"
    ?`इस समय के फलादेश में ${joined} से जुड़े फैसले अहम हैं। ${subject==="career"?"काम के बारे में फैसला लें तो देखें कि उसमें क्या करना होगा और आप कितने तैयार हैं।":subject==="finance"?"कमाई का मौका और खर्च के बाद बचे पैसे अलग देखकर फैसला लें।":subject==="relationships"?"फैसला लेने से पहले बात करें कि दोनों क्या चाहते हैं और कौन क्या जिम्मेदारी लेगा।":"पहले तय करें कि इनमें क्या सबसे जरूरी है, फिर इन्हें साथ सँभालने का तरीका देखें।"} इससे समझ आता है कि इस समय किन बातों पर ध्यान दें। किसी खास घटना का पक्का वादा नहीं किया गया है।`
    :`The current period highlights decisions about ${joined}. ${subject==="career"?"Consider the actual demands of a role alongside your preparation when reviewing work direction.":subject==="finance"?"Distinguish an earning opportunity from the money actually available after expenses.":subject==="relationships"?"Discuss both people's expectations and shared responsibilities before deciding.":"Set priorities and consider these needs together rather than postponing them separately."} This describes the period's focus, rather than promising a particular event.`;
  return {id:`reader-period-${topic.category}`,scope:"period",leadIn:"",text,evidenceIds:linked.map(item=>item.id)};
}

/** capacityParagraph translates only a retained, checked capacity qualification into its everyday subject consequence. */
function capacityParagraph(profile:Profile,report:Report,language:Language):ReadingParagraph|undefined{
  const capacity=profile.topic.findings?.find(item=>item.kind==="capacity");
  const direct=profile.topic.evidence.find(item=>item.kind==="combustion"&&item.planet===profile.planet&&item.direction==="attention");
  const facts=capacity?profile.topic.evidence.filter(item=>capacity.evidence_ids.includes(item.id)):direct?[direct]:[];
  const attention=capacity?.tone==="attention"||facts.some(item=>item.direction==="attention");
  const support=capacity?.tone==="support"||facts.some(item=>item.direction==="support");
  if(!attention&&!support)return undefined;
  // A role label alone is not a capacity finding and cannot establish success, failure or illness.
  const relevant=facts.filter(item=>["combustion","dignity","varga-9","varga-2","varga-4","varga-10","varga-24"].includes(item.kind));
  if(!relevant.length)return undefined;
  const care:Record<ConsultationSubject,readonly [string,string]>={
    personality:["अपनी बात समझना और उसे दूसरे को समझा पाना अलग बातें हैं। दबाव में बात अधूरी रह जाए तो अपनी जरूरत किसी उदाहरण से समझाकर देखें। इसे हमेशा की कमजोरी न मानें।","Understanding your own view and communicating it are separate tasks. Under pressure, explaining your needs with an example may help; this does not establish a permanent personal weakness."],
    education:["विषय अच्छा लगने पर भी लगातार अच्छा करने के लिए ज्यादा अभ्यास लग सकता है। छोटे लक्ष्य बनाएँ, लिखकर अभ्यास करें और बीच-बीच में तैयारी जाँचें। बात समझ आ गई है, इसका मतलब तैयारी पूरी हो गई है, ऐसा न मानें।","Interest may still need extra practice before it becomes consistent performance. Use small goals, written practice and regular review rather than treating understanding alone as complete preparation."],
    career:["काम समझना और अपनी मेहनत की पहचान मिलना अलग बातें हैं। आपकी मेहनत दूसरों को साफ न दिख रही हो तो किए गए काम का रिकॉर्ड रखें। पहले तय करें कि आपको क्या करना है और उसका क्या नतीजा चाहिए।","Understanding a task and having your contribution recognised are separate stages. If your work is not being communicated clearly, document your contribution and agree responsibilities and expected outcomes."],
    finance:["कमाई का मौका मिलना और उसमें से नियमित बचत करना अलग बातें हैं। जो पैसा अभी मिलना बाकी है, उसे मिल चुका मानकर योजना न बनाएँ। पहले साफ करें कि कितनी रकम आई है और कितना खर्च हुआ है।","Turning earning potential into steady savings needs separate discipline. Keep expected income distinct from money already received and clarify actual receipts and expenses."],
    relationships:["पास रहना चाहने पर भी अपनी बात और उम्मीदें समझाने में ज्यादा कोशिश लग सकती है। दूसरे व्यक्ति के अपने-आप समझ लेने का इंतजार न करें। एक बात पर पूरी बातचीत करें; इस संकेत का मतलब रिश्ता टूटना नहीं है।","Wanting closeness may still require extra effort to express needs and expectations. Discuss one issue fully rather than expecting your partner to infer it; this does not establish a separation."],
    family:["घर की जिम्मेदारी निभाना और अपनी जरूरत समझाना अलग बातें हैं। सब कुछ चुपचाप सँभाल रहे हों तो बात करें कि काम कैसे बाँटेंगे और आप कितना समय दे सकते हैं। इससे मदद की जरूरत अनकही नहीं रहेगी।","Handling home duties and expressing your own needs are separate tasks. If too much is being carried silently, clarify responsibilities and availability so the need for support is not left unspoken."],
    wellbeing:["रोज की आदतें सुधारने की इच्छा को निभाने के लिए छोटे बदलाव जरूरी होंगे। काम के कारण अपना समय कम पड़ता हो तो आराम का समय पहले तय करें। आराम को सारे कामों के बाद बचा हुआ समय न मानें।","Good intentions about routine need small, practical changes. If responsibilities keep displacing personal time, give rest a planned place rather than relying on whatever time remains."],
  };
  const strength:Record<ConsultationSubject,readonly [string,string]>={
    personality:["अपने विचार पर आगे बढ़ने में मदद करने वाले संकेत भी हैं। किसी एक चुने हुए काम को पूरा करने पर ध्यान दें। एक साथ बहुत-सी बातों में बँटने से इस मौके का पूरा फायदा नहीं मिल पाएगा।","There is also support for giving direction to your ideas. Use the room for initiative by completing a chosen task rather than splitting attention across too many priorities."],
    education:["समझ बढ़ाने और सही सलाह का इस्तेमाल करने में मदद करने वाले संकेत भी हैं। सीखी हुई बात को अभ्यास, किसी काम या पढ़ाई के साफ लक्ष्य में इस्तेमाल करें। इससे पढ़ाई का फायदा ज्यादा साफ दिखेगा।","There is support for developing understanding and using guidance. Turn that support into practice, a project or a clear learning goal."],
    career:["जिम्मेदारी सँभालने में मदद करने वाले संकेत भी हैं। इसका मतलब अपने-आप पद या सफलता मिलना नहीं है। सही काम चुनना और अपनी मेहनत का साफ सबूत दिखाना भी जरूरी होगा।","There is also support for handling responsibility. This does not automatically confer a position or success; a suitable role and clear evidence of your work matter."],
    finance:["पैसे और साधन सँभालने में मदद करने वाले संकेत भी हैं। कमाई बढ़ाने के साथ मिले हुए पैसों में से नियमित बचत अलग रखें। इससे इस क्षमता का बेहतर इस्तेमाल होगा।","There is also support for managing resources. Use it by setting aside regular savings from actual receipts as well as considering earning opportunities."],
    relationships:["साथ निभाने और एक-दूसरे को समझने में मदद करने वाले संकेत भी हैं। इससे दूसरे व्यक्ति के व्यवहार की गारंटी नहीं मिलती। भरोसा बढ़ाने के लिए समय, बातचीत और दोनों की कोशिश जरूरी होगी।","There is also support for cooperation and shared understanding. This cannot guarantee another person's behaviour; trust still needs time, communication and participation from both people."],
    family:["घर सँभालने और मदद करने में सहारा देने वाले संकेत भी हैं। जिम्मेदारियाँ बाँटकर इसका अच्छा इस्तेमाल करें। इससे परिवार का पूरा बोझ किसी एक व्यक्ति पर नहीं रहेगा।","There is also support for organisation and contribution at home. Use it to share duties so family support does not become one person's entire burden."],
    wellbeing:["रोज की अच्छी आदतें बनाने में मदद करने वाले संकेत भी हैं। अपनी ऊर्जा के हिसाब से छोटा और निभ सकने वाला क्रम चुनें। इससे किसी बीमारी के होने या न होने की पुष्टि नहीं होती।","There is support for establishing a routine. Choose small, sustainable steps suited to your energy; this is not evidence of a health condition."],
  };
  // Retained conditions are used as qualifications, not converted into invented facts about the reader's life.
  if(!report.details)return undefined;
  return {id:`reader-capacity-${profile.subject}`,scope:"natal",leadIn:"",text:wording(attention?care[profile.subject]:strength[profile.subject],language),evidenceIds:relevant.map(item=>item.id)};
}

/** subjectAction gives the priority revealed by a specific destination or conflict, instead of a universal instruction to choose a problem. */
function subjectAction(profile:Profile,language:Language){
  if(profile.subject==="finance"&&profile.secondary?.house===12&&profile.house!==12)return wording([
    "आपके लिए पहले यह देखना जरूरी है कि कमाई में से बचत कितनी हो रही है। ग्राहक या साझेदारी से मिले पैसे, उससे जुड़े खर्च और अपने पास बची रकम का अलग हिसाब रखें। कमाई आने पर भी बचत न हो तो गुरुजी से इसी बात और इससे जुड़े समय पर बात करके उपाय चुनें।",
    "The first priority is the gap between earnings and savings. Separate receipts from clients or partnerships, related expenses and the amount retained. If savings remain limited despite income, use that specific gap and its timing as the focus of an individual consultation.",
  ],language);
  return profile.topic.potential[language];
}

/** deduplicatedReading retains distinct details once when a combined conclusion already states a subject's opening sentence. */
function deduplicatedReading(paragraphs:ReadingParagraph[]){
  const seen=new Set<string>();
  return paragraphs.flatMap(paragraph=>{
    const sentences=(paragraph.text.match(/[^.!?।]+[.!?।]*/gu)||[]).map(sentence=>sentence.trim().replace(/\s+/gu," ")).filter(sentence=>{
      if(!sentence||seen.has(sentence))return false;
      seen.add(sentence);
      return true;
    });
    return sentences.length?[{...paragraph,text:sentences.join(" ")}]:[];
  });
}

/** consultationParagraphs builds a connected reader narrative from validated retained findings, with no recalculation or mutation of the saved report. */
export function consultationParagraphs(topic:FaladeshSynthesisTopic,analysis:FaladeshSynthesis,report:Report,language:Language):ReadingParagraph[]|undefined{
  if(analysis.version!=="m4y-cross-factor-faladesh-v4"||!report.chart||report.birth.unknown_time||!report.details)return undefined;
  const lagna=report.chart.lagna.rashi_number;
  if(lagna<1||lagna>12||report.chart.planets.length!==9||new Set(report.chart.planets.map(item=>item.name)).size!==9||report.chart.planets.some(item=>item.house!==(item.rashi_number-lagna+12)%12+1))return undefined;
  const paragraphs:ReadingParagraph[]=[];
  /** add preserves a paragraph's retained causes and prevents exact duplicate statements within this reading. */
  function add(id:string,text:string,evidenceIds:string[],scope="natal"){
    if(text.trim()&&!paragraphs.some(item=>item.text===text.trim()))paragraphs.push({id,scope,leadIn:"",text:text.trim(),evidenceIds:[...new Set(evidenceIds)]});
  }
  if(topic.category==="dasha"){
    const active=activePeriods(report);
    if(!active)return undefined;
    const names=language==="hi"?[planetNames[active.major.lord],planetNames[active.sub.lord]]:[active.major.lord,active.sub.lord];
    const timing=topic.evidence.filter(item=>item.kind==="period-window"&&item.planet===active.major.lord).map(item=>item.id);
    add("reader-dasha-conclusion",language==="hi"?`इस समय ${names[0]} महादशा में ${names[1]} अंतर्दशा चल रही है। नीचे बताया है कि इस समय जीवन की किन बातों पर ध्यान देना जरूरी है।`:`The running period is ${names[0]} Mahadasha with ${names[1]} Antardasha. The reading below brings together the life subjects connected with both lords.`,timing,"period");
    for(const [index,name] of [...new Set([active.major.lord,active.sub.lord])].entries()){
      const candidates=(["career","education","finance","relationships","family","personality","wellbeing"] as ConsultationSubject[]).flatMap(subject=>{
        const profile=checkedProfile(report,analysis,subject);
        const houses=subjectHouses[subject];
        const connection=periodHouse(report,name,subject);
        const owned=houses.some(house=>owner(report,house)===name);
        const score=profile?.planet===name?3:owned?2:connection===houses[0]?1:0;
        return profile&&score?[{profile,score}]:[];
      }).sort((a,b)=>b.score-a.score).slice(0,1).map(item=>item.profile);
      const house=periodHouse(report,name);
      const fact=topic.evidence.find(item=>item.kind==="placement"&&item.planet===name);
      if(!house||!fact)continue;
      const periodName=language==="hi"?(index===0?"इस लंबे दौर में":"फिलहाल"):(index===0?"Across the wider period":"Within the shorter interval");
      const text=language==="hi"?`${periodName} ${wording(periodActivities[house-1],language)} पर ध्यान देना जरूरी है। `:`${periodName}, the focus falls on ${wording(periodActivities[house-1],language)}. `;
      const retained=[fact.id,...candidates.flatMap(item=>item.root.evidence_ids)].filter(id=>topic.evidence.some(item=>item.id===id));
      add(`reader-dasha-${name}`,text+candidates.map(profile=>subjectConclusion(profile,language)+" "+afterOpening(profile.root.text[language])+" "+subjectAction(profile,language)).join(" "),retained,"period");
    }
    add("reader-dasha-action",wording(["इस समय को हर बात के लिए अच्छा या बुरा न मानें। ऊपर बताई बातों में देखें कि आपकी असली स्थिति क्या है। उपाय उसी परेशानी और उसकी जरूरत के हिसाब से चुनें। केवल दशा का नाम देखकर कोई रत्न या पूजा तय न करें।","Review your actual circumstances in the subjects identified rather than applying one good or bad verdict to the whole interval. Choose an individual remedy around a specific concern, rather than a period name alone."],language),[],"action");
    return deduplicatedReading(paragraphs);
  }
  if(topic.category==="general"){
    const personal=checkedProfile(report,analysis,"personality");
    return personal?generalLifeReading(report,topic,personal.root,language):undefined;
  }
  if(!(topic.category in subjectHouses))return undefined;
  const profile=checkedProfile(report,analysis,topic.category as ConsultationSubject);
  if(!profile)return undefined;
  add("reader-conclusion-"+profile.subject,subjectConclusion(profile,language)+" "+afterOpening(profile.root.text[language]),profile.root.evidence_ids);
  const occupants=topic.findings?.find(item=>item.kind==="occupants");
  if(occupants)add("reader-occupants-"+profile.subject,plainFindingText(occupants.text[language]),occupants.evidence_ids);
  if(profile.secondary&&profile.secondary.house!==profile.house)add("reader-secondary-"+profile.subject,wording(["एक और बात पर ध्यान दें। ","There is a second side to this pattern. "],language)+subjectConclusion({...profile,house:profile.secondary.house},language)+" "+afterOpening(profile.secondary.finding.text[language]),profile.secondary.finding.evidence_ids);
  const capacity=capacityParagraph(profile,report,language);
  if(capacity)paragraphs.push(capacity);
  const period=periodParagraph(report,topic,language,profile.subject);
  if(period)paragraphs.push(period);
  add("reader-action-"+profile.subject,subjectAction(profile,language),[...profile.root.evidence_ids,...(profile.secondary?.finding.evidence_ids||[])],"action");
  return deduplicatedReading(paragraphs);
}
