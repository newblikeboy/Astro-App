import { useState } from "react";
import {
  CalendarDays,
  Compass,
  Layers,
  Orbit,
  Sparkles,
  Sun,
  Sunset,
} from "lucide-react";
import type { Report } from "./types";
import { t, useLanguage } from "./i18n";
import { hindiPlanets, signs } from "./Chart";
import AstroSymbol from "./AstroSymbol";
import ChartViewer from "./ChartViewer";
import TimeRange from "./TimeRange";
import NatalYogaChecks from "./NatalYogaChecks";
import MangalEvidence from "./MangalEvidence";
import { mangalConclusion, placementConclusion } from "./yoga-conclusions";
import { minuteTimestamp } from "./time-display";
import "./kundli-details.css";

/** planet looks up a localized graha name for supplementary Kundli panels. */
const planet = (name: string) => t(hindiPlanets[name] || name);
/** sign looks up a localized zodiac-sign name for supplementary Kundli panels. */
const sign = (number: number) => t(signs[number - 1]);
const weekdays = [
  "Sunday",
  "Monday",
  "Tuesday",
  "Wednesday",
  "Thursday",
  "Friday",
  "Saturday",
];
/** EarlierDetails renders the core supplementary details available on older report snapshots. */
export function EarlierDetails() {
  return (
    <p className="notice">
      {t(
        "Recreate this Kundli from its birth details to include the new calculations. Your saved report remains unchanged.",
      )}
    </p>
  );
}
/** BirthDetailsPanel renders the birth Panchang, calendar facts and exact boundary labels in birthplace time. */
export function BirthDetailsPanel({ report }: { report: Report }) {
  const { language } = useLanguage();
  const details = report.details;
  if (!details) return <EarlierDetails />;
  const b = details.birth,
    c = b.chakra;
  const moon = report.chart?.planets.find((p) => p.name === "Moon");
  const sun = report.chart?.planets.find((p) => p.name === "Sun");
  /** format formats a birth-detail timestamp with its date and minute-precision local clock. */
  const format = (value: string) =>
    minuteTimestamp(value, report.place.zone, language);
  /** boundaryFormat formats an actual start/end boundary without substituting a sunrise-only date. */
  const boundaryFormat = (value: string) =>
    minuteTimestamp(
      new Date(Math.round(Date.parse(value) / 60000) * 60000).toISOString(),
      report.place.zone,
      language,
    );
  const fields = [
    ["Ascendant", report.chart ? sign(report.chart.lagna.rashi_number) : "—"],
    ["Lagna lord", planet(c.lagna_lord)],
    ["Moon sign", moon ? sign(moon.rashi_number) : "—"],
    ["Moon-sign lord", planet(c.rashi_lord)],
    ["Birth nakshatra", moon ? t(moon.nakshatra) : "—"],
    ["Pada", moon ? String(moon.pada) : "—"],
    ["Nakshatra lord", planet(c.nakshatra_lord)],
    ["Sun sign", sun ? sign(sun.rashi_number) : "—"],
    [
      "Birth Dasha balance",
      report.dasha
        ? `${planet(report.dasha.birth_lord)} · ${report.dasha.balance_years.toFixed(3)} ${t("years remaining")}`
        : "—",
    ],
    ["Varna", t(c.varna)],
    ["Vashya", t(c.vashya)],
    ["Yoni", t(c.yoni)],
    ["Gana", t(c.gana)],
    ["Nadi", t(c.nadi)],
  ];
  return (
    <section className="kundli-details birth-details-panel">
      <div className="report-section-heading">
        <div>
          <h2>{t("Birth details")}</h2>
          <p className="muted">
            {t(
              "Your birth moment, Panchang and traditional birth-star profile in one place.",
            )}
          </p>
        </div>
        <CalendarDays size={23} aria-hidden="true" />
      </div>
      <div className="birth-record-strip">
        <strong>{report.birth.name}</strong>
        <span>{format(b.at)}</span>
        <span>
          {language === "hi" ? report.place.hindi : report.place.name}
        </span>
      </div>
      <div className="birth-day-facts">
        <span>
          <CalendarDays size={17} aria-hidden="true" />
          {t("Civil weekday")}
          <strong>{t(weekdays[b.civil_weekday])}</strong>
        </span>
        <span>
          <Sun size={17} aria-hidden="true" />
          {t("Hindu weekday")}
          <strong>{t(weekdays[b.hindu_weekday])}</strong>
        </span>
        <span>
          <AstroSymbol name="Moon" />
          {t("Paksha")}
          <strong>{t(b.paksha)}</strong>
        </span>
      </div>
      <h3>{t("Panchang at birth")}</h3>
      <p className="muted">
        {t(
          "Calculated at your exact birth moment. Each range shows the full astronomical period.",
        )}
      </p>
      <div className="birth-limb-grid">
        {b.limbs.map((limb, i) => (
          <article key={i}>
            <div className="detail-card-label">
              <AstroSymbol name={["Tithi", "Nakshatra", "Yoga", "Karana"][i]} />
              <span>{t(["Tithi", "Nakshatra", "Yoga", "Karana"][i])}</span>
            </div>
            <h4>{t(limb.name)}</h4>
            <TimeRange
              start={limb.start}
              end={limb.end}
              format={boundaryFormat}
            />
          </article>
        ))}
      </div>
      <div className="birth-solar-strip">
        <div>
          <Sun size={18} aria-hidden="true" />
          <span>
            {t("Sunrise on birth date")}
            <strong>{boundaryFormat(b.sunrise)}</strong>
          </span>
        </div>
        <div>
          <Sunset size={18} aria-hidden="true" />
          <span>
            {t("Sunset on birth date")}
            <strong>{boundaryFormat(b.sunset)}</strong>
          </span>
        </div>
      </div>
      <details className="detail-disclosure">
        <summary>{t("Hindu day containing your birth")}</summary>
        <p>
          {t(
            "The Hindu weekday changes at sunrise. Before sunrise, it can differ from the civil weekday.",
          )}
        </p>
        <TimeRange
          start={b.day_start}
          end={b.day_end}
          format={boundaryFormat}
        />

      </details>
      <div className="detail-section-heading">
        <Compass size={20} aria-hidden="true" />
        <h3>{t("Avkahada Chakra")}</h3>
      </div>
      <dl className="birth-chakra-grid">
        {fields.map(([label, value]) => (
          <div key={label}>
            <dt>{t(label)}</dt>
            <dd>{value}</dd>
          </div>
        ))}
      </dl>

    </section>
  );
}

/** ExpandedKundliCharts renders the supported divisional, Moon and Bhav Chalit chart choices. */
export function ExpandedKundliCharts({ report }: { report: Report }) {
  const [mode, setMode] = useState("d1");
  const details = report.details,
    chart = report.chart;
  if (!details || !chart) return <EarlierDetails />;
  const factor = Number(mode.slice(1));
  const varga = details.vargas.find((v) => v.factor === factor);
  const moon = chart.planets.find((p) => p.name === "Moon")!;
  const chalit = mode === "chalit",
    lunar = mode === "moon";
  const selected = chalit
    ? {
        lagna: { rashi_number: chart.lagna.rashi_number },
        planets: details.chalit.planets,
        houses: details.chalit.houses.map((h) => ({
          house: h.house,
          rashi_number: Math.floor(h.middle_longitude / 30) + 1,
        })),
      }
    : lunar
      ? {
          lagna: { rashi_number: moon.rashi_number },
          planets: chart.planets.map((p) => ({
            ...p,
            house: ((p.rashi_number - moon.rashi_number + 12) % 12) + 1,
          })),
        }
      : varga
        ? { lagna: { rashi_number: varga.lagna_sign }, planets: varga.planets }
        : chart;
  const title = chalit
    ? t("Bhav Chalit")
    : lunar
      ? t("Moon chart")
      : mode === "d1"
        ? t("Birth chart · D1")
        : `${t(varga?.name || "")} · D${factor}`;
  return (
    <section className="kundli-details expanded-charts">
      <div className="chart-controls">
        <div
          className="chart-switch"
          role="group"
          aria-label={t("Choose a chart")}
        >
          {[
            ["d1", "Birth chart · D1"],
            ["d9", "Navamsha · D9"],
            ["moon", "Moon chart"],
          ].map(([value, label]) => (
            <button
              key={value}
              aria-pressed={mode === value}
              onClick={() => setMode(value)}
            >
              {t(label)}
            </button>
          ))}
        </div>
        <label className="additional-chart-select">
          {t("All charts")}
          <select
            id="kundli-chart-choice"
            value={mode}
            onChange={(e) => setMode(e.target.value)}
          >
            {details.vargas.map((v) => (
              <option value={`d${v.factor}`} key={v.factor}>
                D{v.factor} · {t(v.name)}
              </option>
            ))}
            <option value="moon">{t("Moon chart")}</option>
            <option value="chalit">{t("Bhav Chalit")}</option>
          </select>
        </label>
      </div>
      <div className="report-layout">
        <section className="report-chart">
          <h2>{title}</h2>
          <ChartViewer
            key={mode}
            chart={selected}
            natalLagna={chart.lagna.rashi_number}
            referenceLabel={
              lunar ? "Moon sign" : chalit ? "Bhava middle" : "Ascendant"
            }
          />
          <p className="muted">
            {chalit
              ? t(
                  "Planets retain their natal signs. Only house assignments change in Bhav Chalit.",
                )
              : t(
                  "Divisional signs are traditional mappings of natal longitudes. They are not new physical positions.",
                )}
          </p>
          {factor >= 40 && (
            <p className="notice">
              {t(
                "Higher divisional charts are especially sensitive to the accuracy of the recorded birth time.",
              )}
            </p>
          )}
        </section>
        <section className="planet-panel">
          <h2>
            {chalit
              ? t("Rashi and Chalit houses")
              : mode === "d9"
                ? t("Navamsha placements")
                : t("Chart placements")}
          </h2>
          <div
            className="table-scroll"
            tabIndex={0}
            role="region"
            aria-label={title + " " + t("Chart placements")}
          >
            <table>
              <thead>
                <tr>
                  <th scope="col">{t("Planet")}</th>
                  <th scope="col">{chalit ? "D1" : t("Sign")}</th>
                  {chalit && <th scope="col">{t("Chalit house")}</th>}
                  <th scope="col">{chalit ? t("Natal house") : t("House")}</th>
                </tr>
              </thead>
              <tbody>
                {selected.planets.map((p) => (
                  <tr key={p.name}>
                    <th scope="row">
                      <span className="planet-identity">
                        <AstroSymbol name={p.name} />
                        {planet(p.name)}
                      </span>
                    </th>
                    <td>
                      {sign(p.rashi_number)}
                      {"vargottama" in p && p.vargottama && (
                        <small>{t("Vargottama")}</small>
                      )}
                    </td>
                    {chalit && <td>{p.house}</td>}
                    <td>
                      {chalit
                        ? chart.planets.find((n) => n.name === p.name)!.house
                        : p.house}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

        </section>
      </div>

    </section>
  );
}

/** PlanetAnalysisPanel renders calculated dignities, aspects, friendships, states and declared strength evidence. */
export function PlanetAnalysisPanel({ report }: { report: Report }) {
  const [friendship, setFriendship] = useState<
    "natural" | "temporary" | "compound"
  >("compound");
  const details = report.details;
  if (!details) return <EarlierDetails />;
  const analysis = details.analysis,
    av = details.ashtakavarga;
  const names = [
    "Sun",
    "Moon",
    "Mars",
    "Mercury",
    "Jupiter",
    "Venus",
    "Saturn",
  ];
  return (
    <section className="kundli-details planet-analysis-panel">
      <div className="report-section-heading">
        <div>
          <h2>{t("Planet analysis")}</h2>
          <p className="muted">
            {t(
              "Sign dignity, planetary states, aspects and traditional point tables.",
            )}
          </p>
        </div>
        <Orbit size={23} aria-hidden="true" />
      </div>
      <div className="planet-condition-grid">
        {analysis.conditions.map((c) => (
          <article key={c.name}>
            <div className="condition-title">
              <AstroSymbol name={c.name} />
              <h3>{planet(c.name)}</h3>
            </div>
            <strong className="dignity-label">{t(c.dignity)}</strong>
            {details.shadbala?.planets.filter(row=>row.planet===c.name).map(row=><p key={row.planet}>{t(row.meets_requirement?"ग्रह की क्षमता को सहारा मिलता है।":"इस ग्रह से जुड़े काम में अधिक तैयारी और नियमित कोशिश जरूरी हो सकती है।")}</p>)}
            {details.qualifications?.neechabhanga.filter(row=>row.planet===c.name&&row.debilitated&&row.cancelled).map(row=><p key={row.planet}>{t("नीच राशि होने के बावजूद कमजोरी को कम करने वाली स्थितियाँ भी हैं। केवल नीच राशि देखकर बुरा फल तय नहीं होता।")}</p>)}
            <dl>
              <div>
                <dt>{t("Sign lord")}</dt>
                <dd>{planet(c.sign_lord)}</dd>
              </div>
              <div>
                <dt>{t("Baladi avastha")}</dt>
                <dd>{t(c.baladi)}</dd>
              </div>
              {c.combust !== null && (
                <div>
                  <dt>{t("Combustion")}</dt>
                  <dd>
                    {t(c.combust ? "Combust" : "Not combust")}
                  </dd>
                </div>
              )}
            </dl>
          </article>
        ))}
      </div>
      <p className="report-caption">
        {t(
          "Dignity describes sign placement, not a complete planetary-strength score. Baladi names describe traditional planetary states.",
        )}
      </p>
      <div className="detail-section-heading">
        <Orbit size={20} aria-hidden="true" />
        <h3>{t("Vimshopaka bala · sixteen vargas")}</h3>
      </div>
      <p className="muted">
        {t(
          "हर ग्रह की अनुकूलता का सार, 20 में से अंक।",
        )}
      </p>
      <div className="varga-strength-list">
        {analysis.varga_strength.map((row) => (
          <article key={row.planet}>
            <span>
              <AstroSymbol name={row.planet} />
              {planet(row.planet)}
            </span>
            <meter
              aria-label={t("Vimshopaka bala") + " · " + planet(row.planet)}
              min={0}
              max={20}
              value={row.total}
            />
            <strong>
              {row.total.toFixed(2)}
              <small> / 20</small>
            </strong>
          </article>
        ))}
      </div>

      <details className="detail-disclosure">
        <summary>{t("Graha drishti · planetary aspects")}</summary>
        <p>
          {t(
            "Full whole-sign aspects of the seven classical planets. Node aspects and angular aspect strength are not included.",
          )}
        </p>
        <div className="aspect-grid">
          {analysis.aspects.map((a) => (
            <article key={a.planet + a.distance}>
              <strong>
                {planet(a.planet)} → {t("House")} {a.house}
              </strong>
              <span>
                {sign(a.sign)} · {t("Aspect")}: {a.distance}
              </span>
              <small>
                {a.occupants.map(planet).join(", ") || t("No natal planets")}
              </small>
            </article>
          ))}
        </div>
      </details>
      <details className="detail-disclosure">
        <summary>{t("Planetary friendship")}</summary>
        <label className="additional-chart-select">
          {t("Relationship type")}
          <select
            value={friendship}
            onChange={(e) => setFriendship(e.target.value as typeof friendship)}
          >
            <option value="natural">{t("Natural")}</option>
            <option value="temporary">{t("Temporary")}</option>
            <option value="compound">{t("Fivefold")}</option>
          </select>
        </label>
        <p>
          {t(
            "Read each row toward its column. Natural relationships can be directional; this is separate from marriage Guna points.",
          )}
        </p>
        <div
          className="table-scroll"
          tabIndex={0}
          role="region"
          aria-label={t("Planetary friendship")}
        >
          <table>
            <thead>
              <tr>
                <th scope="col">{t("Planet")}</th>
                {names.map((n) => (
                  <th key={n} scope="col">
                    {planet(n)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {names.map((from) => (
                <tr key={from}>
                  <th scope="row">{planet(from)}</th>
                  {names.map((to) => (
                    <td key={to}>
                      {from === to
                        ? "—"
                        : t(
                            analysis.friendships.find(
                              (r) => r.from === from && r.to === to,
                            )?.[friendship] || "Not assigned",
                          )}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </details>
      <div className="detail-section-heading">
        <Layers size={20} aria-hidden="true" />
        <h3>{t("Ashtakavarga")}</h3>
      </div>
      <p className="muted">
        {t(
          "हर राशि के लिए अष्टकवर्ग के कुल अंक।",
        )}
      </p>
      <div className="sarva-grid">
        {av.sarva.map((points, i) => (
          <article key={i}>
            <span>{sign(i + 1)}</span>
            <strong>{points}</strong>
            <small>
              {t("House")}{" "}
              {((i + 1 - report.chart!.lagna.rashi_number + 12) % 12) + 1}
            </small>
          </article>
        ))}
      </div>
      <details className="detail-disclosure">
        <summary>{t("Bhinna Ashtakavarga table")}</summary>
        <div
          className="table-scroll"
          tabIndex={0}
          role="region"
          aria-label={t("Bhinna Ashtakavarga table")}
        >
          <table>
            <thead>
              <tr>
                <th scope="col">{t("Planet")}</th>
                {av.sarva.map((_, i) => (
                  <th scope="col" key={i}>
                    {sign(i + 1)}
                  </th>
                ))}
                <th scope="col">{t("Total")}</th>
              </tr>
            </thead>
            <tbody>
              {av.rows.map((r) => (
                <tr key={r.planet}>
                  <th scope="row">{planet(r.planet)}</th>
                  {r.points.map((n, i) => (
                    <td key={i}>{n}</td>
                  ))}
                  <td>{r.total}</td>
                </tr>
              ))}
              <tr>
                <th scope="row">{t("Total")}</th>
                {av.sarva.map((n, i) => (
                  <td key={i}>{n}</td>
                ))}
                <td>{av.total}</td>
              </tr>
            </tbody>
          </table>
        </div>

      </details>

    </section>
  );
}

/** ExpandedYogaPatterns always shows both dosha assessments and filters additional yogas to present patterns. */
export function ExpandedYogaPatterns({ report }: { report: Report }) {
  if (!report.details) return null;
  const mangal = report.details.mangal && mangalConclusion(report.details.mangal);
  const patterns = report.details.analysis.patterns.filter((pattern) => pattern.matches);
  return (
    <section className="kundli-details">
      <NatalYogaChecks report={report} />
      {report.details.mangal && (
        <div className="kundli-mangal-assessment">
          <div className="detail-section-heading">
            <AstroSymbol name="Mars" />
            <h3>{t("Mangal analysis")}</h3>
          </div>
          <div className="yoga-conclusion">
            <span className="detail-card-label">{t("Conclusion")}</span>
            <p className="natal-yoga-status">{mangal?.status}</p>
            <p>{mangal?.meaning}</p>
            {report.details.mangal.base_classification && report.details.mangal.base_classification !== report.details.mangal.classification && <p>{t("mangal.parihar.base")}: {t(`mangal.guruji.${report.details.mangal.base_classification}`)}</p>}
          </div>
          <MangalEvidence rules={report.details.mangal.rules} />
        </div>
      )}
      {patterns.length > 0 && <><div className="detail-section-heading">
        <Sparkles size={20} aria-hidden="true" />
        <h3>{t("Additional yoga placement patterns")}</h3>
      </div>
      <p className="muted">{t("yoga.scope")}</p>
      <div className="yoga-pattern-grid">
        {patterns.map((p) => (
          <article key={p.code}>
            <div>
              <h4>{t(`yoga.name.${p.code}`)}</h4>
              <span
                className={p.matches ? "pattern-matches" : "pattern-absent"}
              >
                {placementConclusion(p.code, p.matches).status}
              </span>
            </div>
            <p>{placementConclusion(p.code, p.matches).meaning}</p>

          </article>
        ))}
      </div>
      </>}
    </section>
  );
}
