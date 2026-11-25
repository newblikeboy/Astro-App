import type { MangalRule } from "./matching-types";
import { t } from "./i18n";
import "./mangal-evidence.css";

/** MangalEvidence presents matched and unresolved Guruji passages verbatim, keeping review separate from cancellation. */
export default function MangalEvidence({ rules }: { rules: MangalRule[] }) {
  const relevant = rules.filter((rule) => rule.source_rule_number && rule.met);
  const unresolved = rules.filter((rule) => rule.source_rule_number && !rule.met && rule.evaluation === "review");
  if (!relevant.length && !unresolved.length) return null;
  return <div className="guruji-mangal-evidence">
    <h4>{t("mangal.parihar.title")}</h4>
    {relevant.map((rule) => <article key={rule.code} data-effect={rule.evaluation === "review" ? "review" : rule.effect}>
      <div className="mangal-evidence-heading">
        <h5 lang="hi">{rule.heading}</h5>
        <span className="mangal-evidence-state">{t(`mangal.parihar.${rule.evaluation === "review" ? "review" : rule.effect}`)}</span>
      </div>
      {!!rule.matched_clauses?.length && <div className="mangal-evidence-basis"><strong>{t("mangal.parihar.basis")}</strong>{rule.matched_clauses.map((clause) => <p key={clause} lang="hi">{clause}</p>)}</div>}
      {rule.evaluation === "review" && <p className="mangal-evidence-note">{t("mangal.parihar.review-note")}</p>}
      <div className="mangal-evidence-source" lang="hi">{rule.text}</div>
    </article>)}
    {unresolved.length > 0 && <aside className="mangal-evidence-pending">
      <h5>{t("mangal.parihar.review")}</h5>
      <p className="mangal-evidence-note">{t("mangal.parihar.review-note")}</p>
      <ul lang="hi">{unresolved.map((rule) => <li key={rule.code}>{rule.heading}</li>)}</ul>
    </aside>}
  </div>;
}
