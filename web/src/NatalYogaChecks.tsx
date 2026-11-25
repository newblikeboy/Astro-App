import type { Report } from "./types";
import { t } from "./i18n";
import { hindiPlanets } from "./Chart";

/** NatalYogaChecks always shows Kaal Sarp status, while hiding absent Rajyog and missing Yogakarakas. */
export default function NatalYogaChecks({ report }: { report: Report }) {
  const checks = report.details?.yoga_checks;
  if (!checks) return <section className="natal-yoga-checks">
    <h3>{t("Kaal Sarp Dosha & Rajyog")}</h3>
    <p className="notice">{t("Recreate this Kundli from its birth details to include Kaal Sarp Dosha and Rajyog checks. This saved report has not been assessed for them.")}</p>
  </section>;
  const kaal = checks.kaal_sarp;
  const raj = checks.rajyog;
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
      </div>
      <h4>{t("What this means")}</h4>
      <p>{t("kaal-sarp.definition")}</p>


    </section>
    {(raj.matches || raj.yogakarakas.length > 0) && <section aria-labelledby="rajyog-heading">
      <h3 id="rajyog-heading">{t("Rajyog · Kendra–Trikona relationships")}</h3>
      {raj.matches && <div className="yoga-conclusion" data-status="present">
        <span className="detail-card-label">{t("Conclusion")}</span>
        <p className="natal-yoga-status">{t(raj.matches ? "rajyog.present" : "rajyog.absent")}</p>
      </div>}

      {raj.yogakarakas.length > 0 && <>
        <h4>{t("Yogakaraka lordship")}</h4>
        {raj.yogakarakas.map((lord) => <p key={lord.planet}>{t("yogakaraka.present", { v1: t(hindiPlanets[lord.planet] || lord.planet) })}</p>)}
      </>}
      <p className="report-caption">{t("yoga.scope")}</p>
    </section>}
  </div>;
}
