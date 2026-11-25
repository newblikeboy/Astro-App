import { useEffect, useRef, useState } from "react";
import {
  ArrowLeft,
  ArrowRight,
  CalendarDays,
  Clock,
  FileText,
  Info,
  LockKeyhole,
  Sparkles,
  LayoutDashboard,
  Grid2X2,
  Orbit,
  Compass,
  Moon,
  Hash,
} from "lucide-react";
import { hindiPlanets, signs } from "./Chart";
import ChartViewer from "./ChartViewer";
import KundliOverviewCharts from "./KundliOverviewCharts";
import GurujiGrahaStatus from "./GurujiGrahaStatus";
import DashaPanel from "./DashaPanel";
import {
  BirthDetailsPanel,
  ExpandedKundliCharts,
  PlanetAnalysisPanel,
  ExpandedYogaPatterns,
} from "./KundliDetails";
import { SaturnTimeline, YoginiPanel } from "./KundliPeriods";
import { clockMinutes } from "./time-display";
import ZodiacMark from "./ZodiacMark";
import AstroSymbol from "./AstroSymbol";
import { t, useLanguage } from "./i18n";
import type { Report } from "./types";
import { placementConclusion } from "./yoga-conclusions";
import NumerologyPanel from "./NumerologyPanel";
import GurujiFaladesh from "./GurujiFaladesh";
import { readingTopics } from "./source-topics";

/** planetName returns the localized name for a report graha. */
const planetName = (name: string) => t(hindiPlanets[name] || name);
/** signName returns the localized zodiac-sign name for a report placement. */
const signName = (sign: number) => t(signs[sign - 1]);
const categories = readingTopics;
const tabs = [
  ["overview", "Overview"],
  ["birth", "Birth details"],
  ["numerology", "Numerology"],
  ["charts", "Charts"],
  ["reading", "Faladesh"],
  ["dasha", "Dasha"],
  ["planets", "Planet analysis"],
  ["checks", "Yogas & Doshas"],
  ["transits", "Transits"],
];
const tabIcons = [
  LayoutDashboard,
  CalendarDays,
  Hash,
  Grid2X2,
  Sparkles,
  Clock,
  Orbit,
  Compass,
  Orbit,
];
const checkCopy: Record<string, [string, string]> = {
  "mangal-lagna": [
    "Mangal placement · Lagna",
    "Mars in houses 1, 4, 7, 8 or 12 from Lagna, according to Guruji's audio.",
  ],
  "mangal-moon": [
    "Mangal placement · Moon",
    "Guruji's Moon check uses houses 1, 4, 7, 8 or 12; a Moon-only indication is partial Manglik.",
  ],
  "gajakesari-base": [
    "Gajakesari · base placement",
    "Jupiter in the 1st, 4th, 7th or 10th sign from the Moon. Strength, aspects and cancellation conditions need a full reading.",
  ],
  "budha-aditya-base": [
    "Budha Aditya · base placement",
    "Sun and Mercury in the same natal sign. Combustion, separation and strength are not evaluated by this placement check.",
  ],
};

/** KundliReport organizes the calculated Kundli, Faladesh, charts, planetary details and timing explorers. */
export default function KundliReport({
  report,
  onBack,
  backLabel,
  onEditBirth,
  onSave,
  onRefreshReading,
  saving,
  buy,
}: {
  report: Report;
  onBack: () => void;
  backLabel?: string;
  onEditBirth?: () => void;
  onSave: () => void;
  onRefreshReading?: () => Promise<void>;
  saving: boolean;
  buy: (p: "pdf" | "consultation") => void;
}) {
  const { language } = useLanguage();
  const [tab, setTab] = useState("overview");
  const [chartMode, setChartMode] = useState("d1");
  const [dashaSystem, setDashaSystem] = useState("vimshottari");
  const [category, setCategory] = useState("general");
  const [readingRefreshing, setReadingRefreshing] = useState(false);
  const [readingError, setReadingError] = useState("");
  const refreshedReadings = useRef(new Set<string>());
  const version = report.source_readings?.length;
  const staleGuest = !report.user_id && !version && !!onRefreshReading;
  const staleYogaGuest = !report.user_id && !!onRefreshReading && report.details?.qualifications?.method === "m4y-phaladeepika7-neechabhanga-strength-v1";
  // Refresh only this superseded unsaved guest reading. Saved reports and historical PDF snapshots remain immutable.
  useEffect(() => {
    if (!onRefreshReading || !(tab === "reading" && (staleGuest || staleYogaGuest) || tab === "checks" && staleYogaGuest)) return;
    const birthKey = JSON.stringify(report.birth);
    if (refreshedReadings.current.has(birthKey)) return;
    refreshedReadings.current.add(birthKey);
    setReadingRefreshing(true);
    setReadingError("");
    void onRefreshReading().catch(() => {
      setReadingError(language === "hi" ? "फलादेश अपडेट नहीं हो सका। पेज दोबारा खोलकर प्रयास करें।" : "The reading could not be updated. Reopen the page to try again.");
    }).finally(() => setReadingRefreshing(false));
  }, [tab, staleGuest, staleYogaGuest, onRefreshReading, version, report.birth, language]);
  const nav = useRef<HTMLDivElement>(null);
  const topicNav = useRef<HTMLDivElement>(null);
  const chart = report.chart;
  const moon = chart?.planets.find((p) => p.name === "Moon");
  const asOf = report.dasha?.as_of || report.created;
  /** date formats a report date/time in the birthplace timezone to minute precision. */
  const date = (value: string, withTime = false) =>
    new Intl.DateTimeFormat(language === "hi" ? "hi-IN" : "en-IN", {
      timeZone: report.place.zone,
      day: "numeric",
      month: "short",
      year: "numeric",
      ...(withTime
        ? ({
            hour: "2-digit",
            minute: "2-digit",
            timeZoneName: "short",
          } as const)
        : {}),
    }).format(new Date(value));
  const displayChart =
    chartMode === "d9" && report.navamsa
      ? {
          lagna: { rashi_number: report.navamsa.lagna_sign },
          planets: report.navamsa.planets,
        }
      : chartMode === "moon" && chart && moon
        ? {
            lagna: { rashi_number: moon.rashi_number },
            planets: chart.planets.map((p) => ({
              ...p,
              house: ((p.rashi_number - moon.rashi_number + 12) % 12) + 1,
            })),
          }
        : chart;

  /** natalTable renders calculated natal placements and the handwritten board's sign-based exaltation/debilitation labels. */
  function natalTable() {
    return (
      <section className="planet-panel">
        <h2>{t("आपकी ग्रह-सारणी")}</h2>
        <p className="graha-table-source">{t("उच्च/नीच स्थिति गुरु जी की हस्तलिखित सूची के अनुसार है।")}</p>

        <div
          className="table-scroll"
          tabIndex={0}
          role="region"
          aria-label={t("आपकी ग्रह-सारणी")}
        >
          <table className="natal-table natal-table-with-status">
            <caption className="sr-only">
              {t("ग्रह, राशि, अंश, नक्षत्र, पद, उच्च/नीच स्थिति और भाव")}
            </caption>
            <thead>
              <tr>
                <th scope="col">{t("ग्रह")}</th>
                <th scope="col">{t("राशि / अंश")}</th>
                <th scope="col">{t("नक्षत्र / पद")}</th>
                <th scope="col">{t("उच्च / नीच")}</th>
                <th scope="col">{t("भाव")}</th>
              </tr>
            </thead>
            <tbody>
              {chart?.planets.map((p) => (
                <tr key={p.name} data-planet={p.name}>
                  <th scope="row">
                    <span className="planet-identity">
                      <AstroSymbol name={p.name} />
                      <span>
                        {planetName(p.name)}
                        {p.retrograde && (
                          <small className="retro">{t("वक्री")}</small>
                        )}
                      </span>
                    </span>
                  </th>
                  <td data-label={t("राशि / अंश")}>
                    <span className="zodiac-identity">
                      <ZodiacMark sign={p.rashi_number} />
                      {signName(p.rashi_number)}
                    </span>
                    <small>{p.degree_in_rashi.toFixed(4)}°</small>
                  </td>
                  <td data-label={t("नक्षत्र / पद")}>
                    {t(p.nakshatra)}
                    <small>
                      {t("पद ")}
                      {p.pada}
                    </small>
                  </td>
                  <td data-label={t("उच्च / नीच")}>
                    <GurujiGrahaStatus planet={p.name} sign={p.rashi_number} />
                  </td>
                  <td data-label={t("भाव")}>
                    <span className="house-badge">{p.house}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    );
  }
  const summary = (
    <dl className="kundli-summary">
      <div>
        <dt>
          <Compass size={16} aria-hidden="true" />
          {t("Ascendant")}
        </dt>
        <dd>{chart ? signName(chart.lagna.rashi_number) : "—"}</dd>
      </div>
      <div>
        <dt>
          <Moon size={16} aria-hidden="true" />
          {t("चंद्र राशि")}
        </dt>
        <dd>{moon ? signName(moon.rashi_number) : "—"}</dd>
      </div>
      <div>
        <dt>
          <Sparkles size={16} aria-hidden="true" />
          {t("Birth nakshatra")}
        </dt>
        <dd>
          {moon ? t(moon.nakshatra) : "—"}
          {moon && (
            <small>
              {t("पद ")}
              {moon.pada}
            </small>
          )}
        </dd>
      </div>
      <div>
        <dt>
          <Clock size={16} aria-hidden="true" />
          {t("Running dasha")}
        </dt>
        <dd>
          {report.dasha?.current_lord
            ? planetName(report.dasha.current_lord)
            : "—"}
          {report.dasha?.current_sub_lord && (
            <small>
              {planetName(report.dasha.current_sub_lord)} {t("Antardasha")}
            </small>
          )}
        </dd>
      </div>
    </dl>
  );

  return (
    <div className="container workspace kundli-workspace">
      <button className="small-link" onClick={onBack}>
        <ArrowLeft size={15} />
        {backLabel || t("जन्म विवरण पर वापस जाएँ")}
      </button>
      <div className="workspace-heading">
        <div>
          <h1>
            {report.birth.name}
            {t(", आपकी कहानी।")}
          </h1>
          <p>
            {report.birth.date} ·{" "}
            {report.birth.unknown_time
              ? t("जन्म समय अज्ञात")
              : clockMinutes(report.birth.time)}{" "}
            · {language === "hi" ? report.place.hindi : report.place.name}
          </p>
        </div>
        <button
          className="button secondary"
          disabled={saving || !!report.user_id}
          onClick={onSave}
        >
          <LockKeyhole size={16} />
          {saving
            ? t("Saving…")
            : report.user_id
              ? t("खाते में सहेजी गई")
              : t("खाते में सहेजें")}
        </button>
      </div>
      {report.demo && (
        <div className="notice">
          <Info size={18} />
          <span>
            {t(
              "यह प्रारंभिक पढ़त है; व्यक्तिगत सलाह के लिए गुरुजी से परामर्श लें। अभी ऑनलाइन खरीद उपलब्ध नहीं है।",
            )}
          </span>
        </div>
      )}
      <label className="kundli-section-select">
        {t("Report section")}
        <select value={tab} onChange={(event) => setTab(event.target.value)}>
          {tabs.map(([id, label]) => (
            <option key={id} value={id}>
              {t(label)}
            </option>
          ))}
        </select>
      </label>
      <div
        className="tabs kundli-tabs"
        role="tablist"
        aria-label={t("Kundli report sections")}
        ref={nav}
        onKeyDown={(event) => {
          const current = tabs.findIndex(([id]) => id === tab);
          const next =
            event.key === "ArrowRight"
              ? (current + 1) % tabs.length
              : event.key === "ArrowLeft"
                ? (current + tabs.length - 1) % tabs.length
                : event.key === "Home"
                  ? 0
                  : event.key === "End"
                    ? tabs.length - 1
                    : -1;
          if (next >= 0) {
            event.preventDefault();
            setTab(tabs[next][0]);
            nav.current
              ?.querySelectorAll<HTMLButtonElement>('[role="tab"]')
              [next]?.focus();
          }
        }}
      >
        {tabs.map(([id, label], index) => (
          <button
            key={id}
            id={`kundli-tab-${id}`}
            role="tab"
            aria-selected={tab === id}
            aria-controls="kundli-panel"
            tabIndex={tab === id ? 0 : -1}
            onClick={() => setTab(id)}
          >
            {(() => {
              const Icon = tabIcons[index];
              return <Icon size={16} aria-hidden="true" />;
            })()}
            {t(label)}
          </button>
        ))}
      </div>
      <div
        id="kundli-panel"
        className="kundli-panel"
        role="tabpanel"
        aria-labelledby={`kundli-tab-${tab}`}
        tabIndex={0}
      >
        {tab === "numerology" ? <NumerologyPanel report={report} /> : !chart ? (
          <div className="empty-state">
            <Clock size={36} />
            <h2>{t("सही जन्म समय के बिना गणना सीमित है।")}</h2>
            <p>
              {t(
                "An accurate birth time is needed for charts, Faladesh inputs, Dasha and transit comparisons. No assumed time has been used.",
              )}
            </p>
            {report.numerology && <>
              <p>{t("numerology.unknown-time")}</p>
              <button className="button secondary" onClick={() => setTab("numerology")}>{t("numerology.open")}</button>
            </>}
            <button className="button" onClick={onEditBirth || onBack}>
              {t("जन्म विवरण बदलें")}
            </button>
          </div>
        ) : (
          <>
            {tab === "overview" && (
              <>
                <KundliOverviewCharts report={report} />
                <div className="overview-details overview-details-below">
                    {summary}
                    <p className="report-caption">
                      {t("Timing and transits calculated for")}{" "}
                      {date(asOf, true)}.
                    </p>
                    {natalTable()}
                </div>
              </>
            )}
            {tab === "birth" && <BirthDetailsPanel report={report} />}
            {tab === "planets" && <PlanetAnalysisPanel report={report} />}
            {tab === "charts" && report.details && (
              <ExpandedKundliCharts report={report} />
            )}
            {tab === "charts" && !report.details && (
              <>
                <div
                  className="chart-switch"
                  role="group"
                  aria-label={t("Choose a chart")}
                >
                  {[
                    ["d1", "Birth chart · D1"],
                    ["d9", "Navamsha · D9"],
                    ["moon", "Moon chart"],
                  ].map(([id, label]) => (
                    <button
                      key={id}
                      aria-pressed={chartMode === id}
                      onClick={() => setChartMode(id)}
                    >
                      {t(label)}
                    </button>
                  ))}
                </div>
                {chartMode === "d9" && !report.navamsa ? (
                  <p className="notice">
                    {t(
                      "This saved report uses an earlier calculation version. Recreate it from the birth details to include the new calculations.",
                    )}
                  </p>
                ) : (
                  <div className="report-layout">
                    <section className="report-chart">
                      <h2>
                        {t(
                          chartMode === "d9"
                            ? "Navamsha · D9"
                            : chartMode === "moon"
                              ? "Moon chart"
                              : "Birth chart · D1",
                        )}
                      </h2>
                      {displayChart && (
                        <ChartViewer
                          key={chartMode}
                          chart={displayChart}
                          natalLagna={chart.lagna.rashi_number}
                          referenceLabel={
                            chartMode === "moon" ? "Moon sign" : "Ascendant"
                          }
                          label={t(
                            chartMode === "d9"
                              ? "North Indian Navamsha chart. Sign and house placements are provided in the table."
                              : "North Indian chart. Planet and house information is provided in the table.",
                          )}
                        />
                      )}
                      <p className="muted">
                        {t(
                          chartMode === "d9"
                            ? "Each natal sign is divided into nine parts of 3°20′. D9 houses are counted from the D9 ascendant. Small birth-time changes can change this chart."
                            : chartMode === "moon"
                              ? "The natal Moon sign is treated as the first house; planetary positions remain the same as D1."
                              : "Whole-sign houses are counted from the natal ascendant.",
                        )}
                      </p>
                    </section>
                    {chartMode === "d9" ? (
                      <section className="planet-panel">
                        <h2>{t("Navamsha placements")}</h2>
                        <p className="muted">
                          {t(
                            "Vargottama means a planet occupies the same sign in D1 and D9.",
                          )}
                        </p>
                        <div
                          className="table-scroll"
                          tabIndex={0}
                          role="region"
                          aria-label={t("Navamsha placements")}
                        >
                          <table>
                            <thead>
                              <tr>
                                <th scope="col">{t("ग्रह")}</th>
                                <th scope="col">D1</th>
                                <th scope="col">D9</th>
                                <th scope="col">{t("भाव")}</th>
                              </tr>
                            </thead>
                            <tbody>
                              {report.navamsa?.planets.map((p) => (
                                <tr key={p.name}>
                                  <th scope="row">
                                    <span className="planet-identity">
                                      <AstroSymbol name={p.name} />
                                      <span>{planetName(p.name)}</span>
                                    </span>
                                  </th>
                                  <td>
                                    {signName(
                                      chart.planets.find(
                                        (n) => n.name === p.name,
                                      )!.rashi_number,
                                    )}
                                  </td>
                                  <td>
                                    {signName(p.rashi_number)}
                                    {p.vargottama && (
                                      <small>{t("Vargottama")}</small>
                                    )}
                                  </td>
                                  <td>{p.house}</td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      </section>
                    ) : chartMode === "moon" ? (
                      <section className="planet-panel">
                        <h2>{t("Moon-relative houses")}</h2>
                        <div
                          className="table-scroll"
                          tabIndex={0}
                          role="region"
                          aria-label={t("Moon-relative houses")}
                        >
                          <table>
                            <thead>
                              <tr>
                                <th scope="col">{t("ग्रह")}</th>
                                <th scope="col">{t("राशि")}</th>
                                <th scope="col">{t("भाव")}</th>
                              </tr>
                            </thead>
                            <tbody>
                              {displayChart?.planets.map((p) => (
                                <tr key={p.name}>
                                  <th scope="row">
                                    <span className="planet-identity">
                                      <AstroSymbol name={p.name} />
                                      <span>{planetName(p.name)}</span>
                                    </span>
                                  </th>
                                  <td>
                                    {signName(
                                      chart.planets.find(
                                        (n) => n.name === p.name,
                                      )!.rashi_number,
                                    )}
                                  </td>
                                  <td>{p.house}</td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      </section>
                    ) : (
                      natalTable()
                    )}
                  </div>
                )}
              </>
            )}
            {tab === "reading" && (
              <section className="faladesh-panel">
                <div className="report-section-heading">
                  <div>
                    <h2>{t("Faladesh")}</h2>
                  </div>
                </div>
                <p className="faladesh-topic-label" id="faladesh-topic-label">
                  {t("Area of life")}
                </p>
                <div
                  className="faladesh-topic-tabs"
                  role="tablist"
                  aria-labelledby="faladesh-topic-label"
                  ref={topicNav}
                >
                  {categories.map(({ name, id }, index) => {
                    const value = id;
                    return (
                      <button
                        key={value}
                        id={`faladesh-topic-${value}`}
                        type="button"
                        role="tab"
                        aria-selected={category === value}
                        aria-controls="faladesh-topic-panel"
                        tabIndex={category === value ? 0 : -1}
                        onClick={() => setCategory(value)}
                        onKeyDown={(event) => {
                          const columns = topicNav.current
                            ? getComputedStyle(
                                topicNav.current,
                              ).gridTemplateColumns.split(" ").length
                            : 1;
                          let next = index;
                          if (event.key === "ArrowRight")
                            next = (index + 1) % categories.length;
                          else if (event.key === "ArrowLeft")
                            next =
                              (index + categories.length - 1) %
                              categories.length;
                          else if (event.key === "ArrowDown")
                            next = (index + columns) % categories.length;
                          else if (event.key === "ArrowUp")
                            next =
                              (index + categories.length - columns) %
                              categories.length;
                          else if (event.key === "Home") next = 0;
                          else if (event.key === "End")
                            next = categories.length - 1;
                          else return;
                          event.preventDefault();
                          setCategory(categories[next].id);
                          topicNav.current
                            ?.querySelectorAll<HTMLButtonElement>("[role=tab]")
                            [next]?.focus();
                        }}
                      >
                        <span>{name === "General" ? (language === "hi" ? "सामान्य" : "General") : name === "Finance" ? (language === "hi" ? "धन" : "Money") : t(name)}</span>
                      </button>
                    );
                  })}
                </div>
                <div
                  id="faladesh-topic-panel"
                  className="faladesh-topic-panel"
                  role="tabpanel"
                  aria-labelledby={`faladesh-topic-${category}`}
                  tabIndex={0}
                >
                  {readingRefreshing ? <p role="status">{language === "hi" ? "फलादेश अपडेट हो रहा है…" : "Updating your reading…"}</p> : readingError ? <p role="alert">{readingError}</p> : <GurujiFaladesh data={report.guruji_faladesh} category={category} report={report} />}

                </div>
              </section>
            )}
            {tab === "dasha" && (
              <>
                <label className="additional-chart-select dasha-system-select">
                  {t("Dasha system")}
                  <select
                    value={dashaSystem}
                    onChange={(e) => setDashaSystem(e.target.value)}
                  >
                    <option value="vimshottari">
                      {t("Vimshottari Dasha")}
                    </option>
                    <option value="yogini">{t("Yogini Dasha")}</option>
                  </select>
                </label>
                {dashaSystem === "vimshottari" ? (
                  <DashaPanel key={report.id} report={report} />
                ) : (
                  <YoginiPanel report={report} />
                )}
              </>
            )}
            {tab === "checks" && (
              <section className="placement-panel">
                <h2>{t("Yogas & Doshas")}</h2>
                {readingRefreshing && <p role="status">{language === "hi" ? "फलादेश अपडेट हो रहा है…" : "Updating your reading…"}</p>}
                {readingError && <p role="alert">{readingError}</p>}
                <p className="muted">
                  {t("आपकी कुंडली में मिलने वाले योग-दोष और उनका सामान्य अर्थ।")}
                </p>
                <ExpandedYogaPatterns report={report} />
                {report.checks?.length ? (
                  <div className="placement-list">
                    {report.checks.filter((check) => !report.details?.mangal || !check.code.startsWith("mangal-")).map((check) => {
                      const copy = checkCopy[check.code];
                      return (
                        copy && (
                          <article key={check.code}>
                            <div>
                              <h3>{t(copy[0])}</h3>
                              <span className="check-result">
                                {placementConclusion(check.code, check.matches).status}
                              </span>
                            </div>
                            <p>{placementConclusion(check.code, check.matches).meaning}</p>

                          </article>
                        )
                      );
                    })}
                  </div>
                ) : (
                  <p className="notice">
                    {t(
                      "This saved report uses an earlier calculation version. Recreate it from the birth details to include the new calculations.",
                    )}
                  </p>
                )}
              </section>
            )}
            {tab === "transits" && (
              <section className="transit-panel">
                <h2>{t("Planetary transits")}</h2>
                {report.transits ? (
                  <>
                    <p className="muted">
                      {t("Snapshot for")} {date(report.transits.as_of, true)}.{" "}
                      {t(
                        "Houses are counted from your natal ascendant and Moon sign.",
                      )}
                    </p>
                    <div className="sade-sati">
                      <h3>{t("Sade Sati")}</h3>
                      <strong>
                        {t(
                          {
                            none: "Not active in this snapshot",
                            rising: "Rising phase",
                            peak: "Middle phase",
                            setting: "Setting phase",
                          }[report.transits.sade_sati],
                        )}
                      </strong>
                      <p>
                        {t(
                          "This snapshot uses Saturn in the 12th, 1st or 2nd sign from the natal Moon. Explore start/end dates in the timeline below.",
                        )}
                      </p>
                    </div>
                    <div
                      className="table-scroll"
                      tabIndex={0}
                      role="region"
                      aria-label={t("Planetary transits")}
                    >
                      <table>
                        <thead>
                          <tr>
                            <th scope="col">{t("ग्रह")}</th>
                            <th scope="col">{t("राशि / अंश")}</th>
                            <th scope="col">{t("From Lagna")}</th>
                            <th scope="col">{t("From Moon")}</th>
                          </tr>
                        </thead>
                        <tbody>
                          {report.transits.planets.map((p) => (
                            <tr key={p.name}>
                              <th scope="row">
                                <span className="planet-identity">
                                  <AstroSymbol name={p.name} />
                                  <span>
                                    {planetName(p.name)}
                                    {p.retrograde && (
                                      <small>{t("वक्री")}</small>
                                    )}
                                  </span>
                                </span>
                              </th>
                              <td>
                                {signName(p.rashi_number)}
                                <small>{p.degree_in_rashi.toFixed(4)}°</small>
                              </td>
                              <td>{p.house}</td>
                              <td>{p.from_moon}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                    <p className="report-caption">
                      {t(
                        "Saved reports keep this snapshot. Create a new report to calculate transits for a later date.",
                      )}
                    </p>
                    <SaturnTimeline key={report.id} report={report} />
                  </>
                ) : (
                  <p className="notice">
                    {t(
                      "A transit snapshot is unavailable for this report. Your natal calculations remain available.",
                    )}
                  </p>
                )}
              </section>
            )}

          </>
        )}
      </div>

      <div className="report-next">
        <div>
          <h2>{t("अपने अगले कदम को समझें।")}</h2>
          <p>{t("रिपोर्ट और परामर्श दो स्वतंत्र विकल्प हैं।")}</p>
        </div>
        <div>
          <button
            className="button secondary"
            onClick={() => buy("pdf")}
            disabled={!chart}
          >
            <FileText size={17} />
            {t("उपाय PDF")}
          </button>
          <button className="button" onClick={() => buy("consultation")}>
            {t("गुरुजी से परामर्श ")}
            <ArrowRight size={16} />
          </button>
        </div>
      </div>
    </div>
  );
}
