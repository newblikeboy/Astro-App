import type { FaladeshSynthesis, FaladeshSynthesisTopic, GurujiFaladeshSnapshot } from "./guruji-faladesh-types";
import type { Report } from "./types";
import { useLanguage, t } from "./i18n";
import { hindiPlanets } from "./Chart";
import { conclusionForCategory, normalizeReadingCategory, readingParagraphs, readingTopics, savedPeriodContext } from "./faladesh-reading";
import "./guruji-faladesh.css";

/** planetName uses the shared planet vocabulary in the reader's selected language. */
function planetName(name:string){return t(hindiPlanets[name]||name);}

/** readingDate formats saved dates without consulting today's clock or showing invalid dates. */
function readingDate(value:string,language:string,zone:string){
  const at=new Date(value);
  return Number.isNaN(at.getTime())?"—":new Intl.DateTimeFormat(language==="hi"?"hi-IN":"en-IN",{day:"numeric",month:"short",year:"numeric",timeZone:zone}).format(at);
}

/** DashaDates displays saved period ranges only on the Dasha topic, with explicit unavailable states. */
function DashaDates({report}:{report:Report}){
  const {language}=useLanguage(),dasha=report.dasha,context=savedPeriodContext(report);
  if(!dasha?.current_lord)return <p>{language==="hi"?"इस रिपोर्ट में दशा का संदर्भ उपलब्ध नहीं है।":"A dasha reference is unavailable in this report."}</p>;
  return <div className="guruji-period-context"><p className="guruji-reference-date">{language==="hi"?"फलादेश की तारीख":"Report reference"}: {readingDate(dasha.as_of,language,report.place.zone)}</p><dl className="guruji-period-ranges">{[{label:{hi:"महादशा",en:"Mahadasha"},lord:dasha.current_lord,period:context?.major},{label:{hi:"अंतर्दशा",en:"Antardasha"},lord:dasha.current_sub_lord,period:context?.sub}].filter(item=>item.lord).map(item=><div key={item.label.en}><dt>{item.label[language]} · {planetName(item.lord)}</dt><dd>{item.period?<><time dateTime={item.period.start}>{readingDate(item.period.start,language,report.place.zone)}</time> – <time dateTime={item.period.end}>{readingDate(item.period.end,language,report.place.zone)}</time></>:language==="hi"?"तारीखें उपलब्ध नहीं":"Dates unavailable"}</dd></div>)}</dl></div>;
}

/** GurujiFaladesh presents the saved interpretation without technical derivations or source-document controls. */
export default function GurujiFaladesh({data,category,report}:{data?:GurujiFaladeshSnapshot;category:string;report:Report}){
  const {language}=useLanguage();
  if(!data)return null;
  const combined=data.analysis?.topics.find(item=>normalizeReadingCategory(item.category)===category);
  if(combined&&data.analysis)return <CombinedFaladeshReading topic={{...combined,category}} analysis={data.analysis} report={report}/>;
  const conclusion=conclusionForCategory(data,report,category),topic=readingTopics.find(item=>item.id===category);
  const heading=category==="general"?(language==="hi"?"सामान्य फलादेश":"General reading"):category==="career"?(language==="hi"?"करियर का निष्कर्ष":"Career conclusion"):category==="dasha"?(language==="hi"?"दशा का निष्कर्ष":"Dasha conclusion"):topic?.title[language]|| (language==="hi"?"निष्कर्ष":"Conclusion");
  return <section className="guruji-faladesh" aria-label={language==="hi"?"आपका फलादेश":"Your reading"}>
    <article className="guruji-conclusion" data-anchors={conclusion?.anchors.map(item=>item.reading.planet).join(",")}>
      <h3>{heading}</h3>
      {conclusion?<><p className="guruji-conclusion-text">{conclusion.meaning.map(item=>item[language]).join(" ")}</p>{conclusion.caution[0]&&<p className="guruji-conclusion-caution"><strong>{language==="hi"?"ध्यान रखें: ":"Keep in mind: "}</strong>{conclusion.caution[0][language]}</p>}</>:<p>{language==="hi"?"इस विषय का निष्कर्ष देने के लिए सुरक्षित पढ़त उपलब्ध नहीं है।":"A saved reading is not available for a conclusion on this subject."}</p>}
      {category==="dasha"&&<><p className="guruji-fine">{language==="hi"?"यह दशा के ग्रहों की जन्मस्थितियों का सार है; इन तिथियों में घटनाएँ होने का संयुक्त आकलन उपलब्ध नहीं है।":"This summarises the dasha planets' natal placements. A combined forecast of events within these dates is unavailable."}</p><DashaDates report={report}/></>}
    </article>


  </section>;
}

/** CombinedFaladeshReading connects checked retained findings into continuous subject paragraphs with practical priorities and saved period references. */
function CombinedFaladeshReading({topic,analysis,report}:{topic:FaladeshSynthesisTopic;analysis:FaladeshSynthesis;report:Report}){
  const {language}=useLanguage();
  const title=topic.category==="general"?(language==="hi"?"सामान्य फलादेश":"General reading"):topic.category==="dasha"?(language==="hi"?"दशा का फलादेश":"Dasha reading"):readingTopics.find(item=>item.id===topic.category)?.title[language];
  return <section className="guruji-faladesh guruji-combined-reading" aria-label={language==="hi"?"आपका फलादेश":"Your reading"}>
    <article className="guruji-conclusion guruji-unified-reading"><h3>{title}</h3><div className="guruji-reading-prose">{readingParagraphs(topic,language,{analysis,report}).map((paragraph,index,paragraphs)=><p className="guruji-reading-paragraph" key={paragraph.id} data-scope={paragraph.scope} data-evidence={paragraph.evidenceIds.join(",")}>
      {paragraph.scope==="period"&&!paragraphs.slice(0,index).some(item=>item.scope==="period")&&<span className="guruji-reading-date">{language==="hi"?"इस तारीख के अनुसार":"Period reference"}: {readingDate(report.dasha?.as_of||analysis.as_of,language,report.place.zone)}. </span>}{paragraph.leadIn&&<span className="guruji-reading-context">{paragraph.leadIn}: </span>}{paragraph.text}
    </p>)}</div>{topic.category==="dasha"&&<><DashaDates report={report}/>{topic.next_period&&<p className="guruji-next-period"><strong>{language==="hi"?"अगली संबंधित अवधि":"Next related interval"}: {planetName(topic.next_period.lord)}</strong><br/><time dateTime={topic.next_period.start}>{readingDate(topic.next_period.start,language,report.place.zone)}</time> – <time dateTime={topic.next_period.end}>{readingDate(topic.next_period.end,language,report.place.zone)}</time><br/>{topic.next_period.text[language]}</p>}</>}</article>
  </section>;
}
