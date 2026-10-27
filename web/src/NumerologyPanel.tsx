import { Hash } from "lucide-react";
import type { NumerologyNumber, Report } from "./types";
import { hindiPlanets } from "./Chart";
import { t, useLanguage } from "./i18n";
import "./numerology.css";

/** numerologyWorking formats recorded addition steps without recalculating a saved number. */
export function numerologyWorking(number: NumerologyNumber): string[] {
  return number.steps.map((total, i) => {
    const digits = i === 0 ? number.digits : String(number.steps[i - 1]).split("").map(Number);
    return digits.length === 1 ? String(total) : `${digits.join(" + ")} = ${total}`;
  });
}

/** NumberReading explains one saved date number and keeps its arithmetic available below the meaning. */
function NumberReading({ kind, number }: { kind: "mulank" | "bhagyank"; number: NumerologyNumber }) {
  return <article className="numerology-number" data-number-kind={kind}>
    <header>
      <h3>{t(kind === "mulank" ? "Mulank" : "Bhagyank")}</h3>
      <strong className="numerology-value">{number.value}</strong>
    </header>
    <dl><div><dt>{t("Number ruler")}</dt><dd>{t(hindiPlanets[number.planet] || number.planet)}</dd></div></dl>
    <p>{t(`numerology.${kind}.${number.value}`)}</p>
    <details className="detail-disclosure">
      <summary>{t("Calculation details")}</summary>
      <p>{t(`numerology.${kind}-explanation`)}</p>
      <ol className="numerology-working">{numerologyWorking(number).map((step, i) => <li key={i}>{step}</li>)}</ol>
    </details>
  </article>;
}

/** NumerologyPanel renders only saved date-based evidence, including unknown-time reports and an explicit older-report state. */
export default function NumerologyPanel({ report }: { report: Report }) {
  const reading = report.numerology;
  const { language } = useLanguage();
  const extended = reading?.extended;
  const named = extended?.naamank;
  return <section className="kundli-details numerology-panel" aria-labelledby="numerology-heading">
    <div className="report-section-heading">
      <div><h2 id="numerology-heading">{t("Numerology")}</h2><p className="muted">{t("numerology.intro")}</p></div>
      <Hash size={23} aria-hidden="true" />
    </div>
    {!reading ? <p className="notice">{t("numerology.missing")}</p> : <>
      <div className="yoga-conclusion">
        <span className="detail-card-label">{t("Conclusion")}</span>
        <p className="natal-yoga-status">{t("numerology.summary", { v1: reading.mulank.value, v2: reading.bhagyank.value })}{named && <> {t("numerology.name-summary", { v1: named.number.value })}</>}</p>
        <p>{t(extended ? "numerology.extended-intro" : "numerology.summary-meaning")}</p>
      </div>
      {report.birth.unknown_time && <p className="notice">{t("numerology.unknown-time")}</p>}
      <div className="numerology-grid">
        <NumberReading kind="mulank" number={reading.mulank} />
        <NumberReading kind="bhagyank" number={reading.bhagyank} />
        {extended && <article className="numerology-number" data-number-kind="naamank">
          <header><h3>{t("Naamank")}</h3>{named && <strong className="numerology-value">{named.number.value}</strong>}</header>
          {named ? <>
            <dl><div><dt>{t("numerology.name-used")}</dt><dd>{named.spelling}</dd></div><div><dt>{t("Number ruler")}</dt><dd>{t(hindiPlanets[named.number.planet] || named.number.planet)}</dd></div></dl>
            <p>{named.meaning[language]}</p>
            <details className="detail-disclosure"><summary>{t("Calculation details")}</summary>
              <p>{t("numerology.name-method")}</p>
              <p className="numerology-letter-working">{named.letters.map((letter, i) => `${letter} = ${named.number.digits[i]}`).join(" · ")}</p>
              <ol className="numerology-working">{numerologyWorking(named.number).map((step, i) => <li key={i}>{step}</li>)}</ol>
            </details>
          </> : <p className="notice">{t("numerology.name-missing")}</p>}
        </article>}
      </div>
      {extended ? <>
        <section className="numerology-profile" aria-labelledby="numerology-profile-heading">
          <h3 id="numerology-profile-heading">{t("numerology.profile-title")}</h3>
          <p className="muted">{t("numerology.profile-basis", { v1: extended.profile.number })}</p>
          <dl className="numerology-facts">
            <div><dt>{t("numerology.lucky-numbers")}</dt><dd>{extended.lucky_numbers.join(", ")}</dd></div>
            <div><dt>{t("numerology.lucky-dates")}</dt><dd>{extended.lucky_dates.join(", ")}</dd></div>
            {extended.profile.facts.map((fact) => <div key={fact.id}><dt>{fact.label[language]}</dt><dd>{fact.value[language]}</dd></div>)}
          </dl>
          <p className="report-caption">{t("numerology.correspondence-note")}</p>
        </section>
        <div className="numerology-guidance">
          {extended.profile.guidance.map((part) => <article key={part.id} data-numerology-topic={part.id}>
            <h3>{part.title[language]}</h3>
            {part.text[language].split("\n\n").map((paragraph, i) => <p key={i}>{paragraph}</p>)}
            {part.id === "yantra" && <table className="numerology-yantra"><caption>{t("numerology.yantra-caption")}</caption><tbody>
              {extended.profile.yantra.map((row, i) => <tr key={i}>{row.map((value, j) => <td key={j}>{value}</td>)}</tr>)}
            </tbody></table>}
          </article>)}
        </div>
      </> : <p className="notice">{t("numerology.extended-missing")}</p>}
      <p className="report-caption">{t("numerology.scope")}</p>
      <details className="detail-disclosure">
        <summary>{t("Birth calculation reference")}</summary>
        <p>{t("Birth date")}: {reading.birth_date.split("-").reverse().join("-")}</p>
        <p>{t("numerology.method")}</p>
        {extended && <><p>{t("numerology.name-method")}</p><p>{t("numerology.profile-method")}</p><p>{extended.method}</p></>}
      </details>
    </>}
  </section>;
}
