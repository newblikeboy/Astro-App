import type { Report } from "./types";
import type { RajyogPair } from "./kundli-details-types";
import { t } from "./i18n";
import { hindiPlanets, signs } from "./Chart";

const sideLabels: Record<string, string> = {
  rahu_to_ketu: "Rahu to Ketu",
  ketu_to_rahu: "Ketu to Rahu",
  node_boundary: "On the nodal boundary",
  none: "Planets on both sides",
};
const linkLabels: Record<string, string> = {
  conjunction: "Same-sign conjunction",
  mutual_aspect: "Mutual full aspect",
  sign_exchange: "Exchange of signs",
};

/** graha formats a localized planet name using the report's shared vocabulary. */
const graha = (name: string) => t(hindiPlanets[name] || name);

/** RajyogEvidence shows the checked house lords, their occupied signs and all matching links. */
function RajyogEvidence({ pairs }: { pairs: RajyogPair[] }) {
  return <div className="yoga-pattern-grid">
    {pairs.map((pair) => <article key={`${pair.kendra_house}-${pair.trikona_house}`}>
      <h4>{t("Kendra lord")} {pair.kendra_house}: {graha(pair.kendra_lord)} · {t("Trikona lord")} {pair.trikona_house}: {graha(pair.trikona_lord)}</h4>
      <p>{graha(pair.kendra_lord)}: {t(signs[pair.kendra_lord_sign - 1])} · {graha(pair.trikona_lord)}: {t(signs[pair.trikona_lord_sign - 1])}</p>
      <span className={pair.links.length ? "pattern-matches" : "pattern-absent"}>
        {pair.links.length ? pair.links.map((link) => t(linkLabels[link] || link)).join(" · ") : t("No checked relationship")}
      </span>
    </article>)}
  </div>;
}

/** NatalYogaChecks renders saved server evidence for Kaal Sarp and basic Rajyog, with an explicit older-snapshot state. */
export default function NatalYogaChecks({ report }: { report: Report }) {
  const checks = report.details?.yoga_checks;
  if (!checks) return <section className="natal-yoga-checks">
    <h3>{t("Kaal Sarp Dosha & Rajyog")}</h3>
    <p className="notice">{t("Recreate this Kundli from its birth details to include Kaal Sarp Dosha and Rajyog checks. This saved report has not been assessed for them.")}</p>
  </section>;
  const kaal = checks.kaal_sarp;
  const raj = checks.rajyog;
  const matches = raj.pairs.filter((pair) => pair.links.length > 0);
  // A stored present assessment can be named from its recorded houses even on older snapshots.
  const typed = kaal.status === "present" && Number.isInteger(kaal.rahu_house) && kaal.rahu_house >= 1 && kaal.rahu_house <= 12 && kaal.ketu_house === (kaal.rahu_house + 5) % 12 + 1;
  const kaalStatus = kaal.status === "present"
    ? t(typed ? "kaal-sarp.present" : "kaal-sarp.present-unnamed", { v1: t(`kaal-sarp.type.${kaal.rahu_house}`) })
    : t(`kaal-sarp.${kaal.status}`);
  return <div className="natal-yoga-checks">
    <section aria-labelledby="kaal-sarp-heading">
      <h3 id="kaal-sarp-heading">{t("Kaal Sarp Dosha")}</h3>
      <div className="yoga-conclusion" data-status={kaal.status}>
        <span className="detail-card-label">{t("Conclusion")}</span>
        <p className="natal-yoga-status">{kaalStatus}</p>
        <p>{t(kaal.status === "present" ? "kaal-sarp.meaning" : `kaal-sarp.${kaal.status}-meaning`)}</p>
        {typed && <p>{t(`kaal-sarp.theme.${kaal.rahu_house}`)}</p>}
      </div>
      <h4>{t("What this means")}</h4>
      <p>{t("kaal-sarp.definition")}</p>
      <details className="detail-disclosure">
        <summary>{t("Calculation details")}</summary>
        <p>{t("All seven classical planets must lie strictly within the same half of the Rahu–Ketu axis, using exact longitudes. Both halves are checked; a planet on a node is flagged for review.")}</p>
        <p>{graha("Rahu")}: {t("House")} {kaal.rahu_house} · {graha("Ketu")}: {t("House")} {kaal.ketu_house} · {t(sideLabels[kaal.arc])}</p>
        <p>{graha("Rahu")}: {kaal.rahu_longitude.toFixed(6)}° · {graha("Ketu")}: {kaal.ketu_longitude.toFixed(6)}°</p>
        <div className="yoga-pattern-grid">
          {kaal.planets.map((p) => <article key={p.planet}>
            <h4>{graha(p.planet)}</h4>
            <p>{t("Sidereal longitude")}: {p.longitude.toFixed(6)}°</p>
            <p>{t("Forward angle from Rahu")}: {p.from_rahu_degrees.toFixed(6)}°</p>
            <small>{t(sideLabels[p.side])}</small>
          </article>)}
        </div>
      </details>
      <details className="detail-disclosure">
        <summary>{t("Twelve Kaal Sarp types")}</summary>
        <div className="table-scroll" tabIndex={0} role="region" aria-label={t("Twelve Kaal Sarp types")}>
          <table><thead><tr><th scope="col">{t("Kaal Sarp type")}</th><th scope="col">{graha("Rahu")} · {t("House")}</th><th scope="col">{graha("Ketu")} · {t("House")}</th></tr></thead>
            <tbody>{Array.from({ length: 12 }, (_, i) => <tr key={i} aria-current={typed && kaal.rahu_house === i + 1 ? "true" : undefined}><th scope="row">{t(`kaal-sarp.type.${i + 1}`)}</th><td>{i + 1}</td><td>{(i + 6) % 12 + 1}</td></tr>)}</tbody>
          </table>
        </div>
      </details>
    </section>
    <section aria-labelledby="rajyog-heading">
      <h3 id="rajyog-heading">{t("Rajyog · Kendra–Trikona relationships")}</h3>
      <div className="yoga-conclusion" data-status={raj.matches ? "present" : "absent"}>
        <span className="detail-card-label">{t("Conclusion")}</span>
        <p className="natal-yoga-status">{t(raj.matches ? "rajyog.present" : "rajyog.absent")}</p>
        <p>{t(raj.matches ? "rajyog.meaning" : "rajyog.absent-meaning")}</p>
      </div>
      <details className="detail-disclosure">
        <summary>{t("Calculation details")}</summary>
        <p>{t("Checks distinct lords of Kendra houses 1, 4, 7, 10 and Trikona houses 1, 5, 9 for same-sign conjunction, mutual full graha aspect or sign exchange in D1. A one-way aspect does not qualify.")}</p>
        {matches.length > 0 && <RajyogEvidence pairs={matches} />}
        <h4>{t("All checked lord pairs")}</h4>
        <RajyogEvidence pairs={raj.pairs} />
      </details>
      <h4>{t("Yogakaraka lordship")}</h4>
      {raj.yogakarakas.length ? <>
        {raj.yogakarakas.map((lord) => <p key={lord.planet}>{t("yogakaraka.present", { v1: graha(lord.planet) })}</p>)}
        <p>{t("yogakaraka.meaning")}</p>
      </> : <p>{t("yogakaraka.absent")}</p>}
      <p className="report-caption">{t("yoga.scope")}</p>
    </section>
  </div>;
}
