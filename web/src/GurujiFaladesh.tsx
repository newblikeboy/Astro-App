import type { GurujiFaladeshSnapshot } from "./guruji-faladesh-types";
import type { Report } from "./types";
import { useLanguage, t } from "./i18n";
import { hindiPlanets } from "./Chart";
import { SourceInline } from "./SourcePassageText";
import { readingTopics, sourceTopicExcerpts } from "./source-topics";
import "./guruji-faladesh.css";

/** readingDate formats a saved period boundary in the birth place's timezone without deriving a prediction. */
function readingDate(value: string, language: string, zone: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "—" : new Intl.DateTimeFormat(language === "hi" ? "hi-IN" : "en-IN", { day: "numeric", month: "short", year: "numeric", timeZone: zone }).format(date);
}

/** DashaDates retains the old reader's saved period context separately from original PDF prose. */
function DashaDates({ report }: { report: Report }) {
  const { language } = useLanguage();
  const dasha = report.dasha;
  if (!dasha?.current_lord) return null;
  const asOf = Date.parse(dasha.as_of);
  const major = dasha.periods?.find(period => period.lord === dasha.current_lord && Date.parse(period.start) <= asOf && Date.parse(period.end) > asOf);
  const sub = major?.antardashas?.find(period => period.lord === dasha.current_sub_lord && Date.parse(period.start) <= asOf && Date.parse(period.end) > asOf);
  return <div className="guruji-period-context"><p className="guruji-reference-date">{language === "hi" ? "फलादेश की तारीख" : "Report reference"}: {readingDate(dasha.as_of, language, report.place.zone)}</p><dl className="guruji-period-ranges">{[{ label: { hi: "महादशा", en: "Mahadasha" }, lord: dasha.current_lord, period: major }, { label: { hi: "अंतर्दशा", en: "Antardasha" }, lord: dasha.current_sub_lord, period: sub }].filter(item => item.lord).map(item => <div key={item.label.en}><dt>{item.label[language]} · {t(hindiPlanets[item.lord] || item.lord)}</dt><dd>{item.period ? <><time dateTime={item.period.start}>{readingDate(item.period.start, language, report.place.zone)}</time> – <time dateTime={item.period.end}>{readingDate(item.period.end, language, report.place.zone)}</time></> : language === "hi" ? "तारीखें उपलब्ध नहीं" : "Dates unavailable"}</dd></div>)}</dl></div>;
}

/** GurujiFaladesh preserves the original subject tabs and continuous reading design using only verbatim PDF excerpts. */
export default function GurujiFaladesh({ category, report }: { data?: GurujiFaladeshSnapshot; category: string; report: Report }) {
  const { language } = useLanguage();
  const topic = readingTopics.find(item => item.id === category) || readingTopics[0];
  const excerpts = sourceTopicExcerpts(report, topic.id);
  return <section className="guruji-faladesh guruji-combined-reading" aria-label={language === "hi" ? "आपका फलादेश" : "Your reading"}>
    <article className="guruji-conclusion guruji-unified-reading"><h3>{topic.title[language]}</h3><div className="guruji-reading-prose" lang="hi">
      {excerpts.length ? excerpts.map(({ entry, identity, parts }) => <div className="guruji-topic-excerpt" key={identity} data-source-id={identity} data-source-category={entry.category} data-source-key={entry.key}>
        {parts.map(part => ["title", "heading", "subheading", "subtitle"].includes(part.kind) ? <h4 className="guruji-reading-source-heading" key={part.start} data-source-start={part.start} data-source-end={part.end}><SourceInline text={part.text} heading /></h4> : <p className="guruji-reading-paragraph" key={part.start} data-source-start={part.start} data-source-end={part.end}><SourceInline text={part.text} /></p>)}
      </div>) : <p>{language === "hi" ? "इस विषय का मूल PDF फलादेश इस रिपोर्ट में उपलब्ध नहीं है।" : "Original PDF text for this subject is unavailable in this report."}</p>}
    </div>{category === "dasha" && <DashaDates report={report} />}</article>
  </section>;
}
