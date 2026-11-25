import type { NumerologyNumber, Report } from "./types";
import { t, useLanguage } from "./i18n";
import SourcePassages from "./SourcePassages";
import "./numerology.css";

/** numerologyWorking formats the recorded arithmetic without recalculating saved evidence. */
export function numerologyWorking(number: NumerologyNumber): string[] {
  return number.steps.map((total, i) => {
    const digits = i === 0 ? number.digits : String(number.steps[i - 1]).split("").map(Number);
    return digits.length === 1 ? String(total) : `${digits.join(" + ")} = ${total}`;
  });
}

/** NumberSummary presents the recorded number and its full arithmetic in a compact overview card. */
function NumberSummary({ kind, number }: { kind: "मूलांक" | "भाग्यांक"; number: NumerologyNumber }) {
  return <article className="numerology-summary-card">
    <div className="numerology-summary-main">
      <div><h3>{t(kind === "मूलांक" ? "Mulank" : "Bhagyank")}</h3><p>{t(kind === "मूलांक" ? "जन्म के दिन से" : "पूरी जन्मतिथि से")}</p></div>
      <strong className="numerology-value">{number.value}</strong>
    </div>
    <div className="numerology-calculation"><span>{t("Calculation details")}</span><ol className="numerology-working">{numerologyWorking(number).map((line,index)=><li key={index}>{line}</li>)}</ol></div>
  </article>;
}

/** NumberReading offers the complete original Hindi passage in a single, full-width disclosure. */
function NumberReading({ kind, number, report }: { kind: "मूलांक" | "भाग्यांक"; number: NumerologyNumber; report: Report }) {
  const passages = (report.numerology?.source_readings || []).filter(entry => entry.category === kind && entry.key === String(number.value));
  return <article className="numerology-number" data-number-kind={kind === "मूलांक" ? "mulank" : "bhagyank"}>
    {passages.length ? <SourcePassages passages={passages} numberMarker={number.value} /> : <div className="numerology-unavailable"><span className="source-number-marker" aria-hidden="true">{number.value}</span><div><h4>{kind} {number.value}</h4><p>इस अंक का फल दी गई PDF में उपलब्ध नहीं है।</p></div></div>}
  </article>;
}

/** NumerologyPanel presents date arithmetic and Guruji's original Hindi, without authored number profiles. */
export default function NumerologyPanel({ report }: { report: Report }) {
  const { language } = useLanguage();
  const reading = report.numerology;
  const recordedDate = reading?.birth_date || report.birth.date;
  const birthDate = reading ? new Intl.DateTimeFormat(language === "hi" ? "hi-IN" : "en-IN", { day:"numeric", month:"long", year:"numeric", timeZone:"UTC" }).format(new Date(`${recordedDate}T12:00:00Z`)) : "";
  return <section className="kundli-details numerology-panel" aria-labelledby="numerology-heading">
    <header className="numerology-heading"><div><span className="numerology-eyebrow">{t("जन्मतिथि पर आधारित")}</span><h2 id="numerology-heading">{t("Numerology")}</h2></div>{reading && <div className="numerology-birth-date"><span>{t("जन्मतिथि")}</span><time dateTime={recordedDate}>{birthDate}</time></div>}</header>
    {!reading ? <p className="numerology-empty">{t("numerology.missing")}</p> : <>
      <div className="numerology-grid"><NumberSummary kind="मूलांक" number={reading.mulank} /><NumberSummary kind="भाग्यांक" number={reading.bhagyank} /></div>
      <div className="numerology-reading-heading"><h3>{t("गुरु जी का फलादेश")}</h3><span>{t("मूल हिंदी पाठ")}</span></div>
      <div className="numerology-readings"><NumberReading kind="मूलांक" number={reading.mulank} report={report} /><NumberReading kind="भाग्यांक" number={reading.bhagyank} report={report} /></div>
    </>}
  </section>;
}
