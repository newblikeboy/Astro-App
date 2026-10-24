import { useRef, useState } from "react";
import {
  ArrowLeft,
  ArrowRight,
  ChevronDown,
  CalendarDays,
  Clock,
  FileText,
  Info,
  LockKeyhole,
  Sparkles,
  UserRound,
  GraduationCap,
  BriefcaseBusiness,
  Heart,
  UsersRound,
  HeartPulse,
  LayoutDashboard,
  Grid2X2,
  Orbit,
  Settings2,
  Compass,
  Moon,
} from "lucide-react";
import { hindiPlanets, signs } from "./Chart";
import ChartViewer from "./ChartViewer";
import DashaPanel from "./DashaPanel";
import {
  BirthDetailsPanel,
  ExpandedKundliCharts,
  PlanetAnalysisPanel,
  ExpandedYogaPatterns,
} from "./KundliDetails";
import { SaturnTimeline, YoginiPanel } from "./KundliPeriods";
import { clockMinutes, minuteTimestamp } from "./time-display";
import ZodiacMark from "./ZodiacMark";
import AstroSymbol from "./AstroSymbol";
import { t, useLanguage } from "./i18n";
import type { DashaPeriod, Report, Rule } from "./types";

/** planetName returns the localized name for a report graha. */
const planetName = (name: string) => t(hindiPlanets[name] || name);
/** signName returns the localized zodiac-sign name for a report placement. */
const signName = (sign: number) => t(signs[sign - 1]);
const categories = [
  { name: "General", Icon: Sparkles },
  { name: "Personality", Icon: UserRound },
  { name: "Education", Icon: GraduationCap },
  { name: "Career", Icon: BriefcaseBusiness },
  { name: "Relationships", Icon: Heart },
  { name: "Family", Icon: UsersRound },
  { name: "Wellbeing", Icon: HeartPulse },
  { name: "Dasha", Icon: Clock },
];
const tabs = [
  ["overview", "Overview"],
  ["birth", "Birth details"],
  ["charts", "Charts"],
  ["reading", "Faladesh"],
  ["dasha", "Dasha"],
  ["planets", "Planet analysis"],
  ["checks", "Yogas & Doshas"],
  ["transits", "Transits"],
  ["settings", "Calculation reference"],
];
const tabIcons = [
  LayoutDashboard,
  CalendarDays,
  Grid2X2,
  Sparkles,
  Clock,
  Orbit,
  Compass,
  Orbit,
  Settings2,
];
const housesByCategory: Record<string, number[]> = {
  personality: [1, 5],
  education: [4, 5, 9],
  career: [2, 6, 10, 11],
  relationships: [5, 7],
  family: [2, 4],
  wellbeing: [1, 6, 8, 12],
};
const checkCopy: Record<string, [string, string]> = {
  "mangal-lagna": [
    "Mangal placement · Lagna",
    "Mars in houses 1, 2, 4, 7, 8 or 12 from the ascendant. Cancellation rules are not applied.",
  ],
  "mangal-moon": [
    "Mangal placement · Moon",
    "The same six-house check, counted from the natal Moon sign. Traditions use different reference points.",
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

/** RuleReading renders one applicable approved interpretation with its supplied language content. */
function RuleReading({ rule }: { rule: Rule }) {
  const { language } = useLanguage();
  return (
    <article className="faladesh-reading">
      <span className="pill">
        {rule.demo ? t("Demo reading") : t("Approved reading")} · {t("Version")}{" "}
        {rule.version}
      </span>
      <h3>{t(rule.topic)}</h3>
      <p
        lang={
          language === "en" && rule.english ? "en" : rule.demo ? language : "hi"
        }
      >
        {language === "en" && rule.english ? rule.english : t(rule.hindi)}
      </p>
      {rule.mantra && (
        <p>
          <strong>{t("मंत्र: ")}</strong>
          {rule.mantra}
        </p>
      )}
      {rule.gemstone && (
        <p>
          <strong>{t("रत्न संबंधी मार्गदर्शन: ")}</strong>
          {rule.gemstone}
          {t(" · रत्न खरीद में शामिल नहीं।")}
        </p>
      )}
    </article>
  );
}

/** KundliReport organizes the calculated Kundli, Faladesh, charts, planetary details and timing explorers. */
export default function KundliReport({
  report,
  onBack,
  backLabel,
  onEditBirth,
  onSave,
  saving,
  buy,
}: {
  report: Report;
  onBack: () => void;
  backLabel?: string;
  onEditBirth?: () => void;
  onSave: () => void;
  saving: boolean;
  buy: (p: "pdf" | "consultation") => void;
}) {
  const { language } = useLanguage();
  const [tab, setTab] = useState("overview");
  const [chartMode, setChartMode] = useState("d1");
  const [dashaSystem, setDashaSystem] = useState("vimshottari");
  const [category, setCategory] = useState("general");
  const nav = useRef<HTMLDivElement>(null);
  const topicNav = useRef<HTMLDivElement>(null);
  const chart = report.chart;
  const moon = chart?.planets.find((p) => p.name === "Moon");
  const sun = chart?.planets.find((p) => p.name === "Sun");
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
  /** isActive checks whether the report instant belongs to the displayed Dasha interval. */
  const isActive = (period: DashaPeriod) =>
    new Date(asOf) >= new Date(period.start) &&
    new Date(asOf) < new Date(period.end);
  const activePeriod = report.dasha?.periods.find(isActive);
  const rules = report.rules.filter(
    (r) => (r.category || "general") === category,
  );
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

  /** natalTable renders the natal graha table from the supplied calculated placements. */
  function natalTable() {
    return (
      <section className="planet-panel">
        <h2>{t("आपकी ग्रह-सारणी")}</h2>
        <p className="muted">
          {t("लाहिड़ी अयनांश · राशि आधारित भाव · औसत राहु")}
        </p>
        <div
          className="table-scroll"
          tabIndex={0}
          role="region"
          aria-label={t("आपकी ग्रह-सारणी")}
        >
          <table className="natal-table">
            <caption className="sr-only">
              {t("ग्रह, राशि, अंश, नक्षत्र, पद और भाव")}
            </caption>
            <thead>
              <tr>
                <th scope="col">{t("ग्रह")}</th>
                <th scope="col">{t("राशि / अंश")}</th>
                <th scope="col">{t("नक्षत्र / पद")}</th>
                <th scope="col">{t("भाव")}</th>
              </tr>
            </thead>
            <tbody>
              {chart?.planets.map((p) => (
                <tr key={p.name}>
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
              "ग्रह स्थिति Swiss Ephemeris से गणना की गई है। डेमो व्याख्या व्यक्तिगत सलाह नहीं है; स्वीकृत सामग्री आने तक लाइव बिक्री बंद है।",
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
        {!chart ? (
          <div className="empty-state">
            <Clock size={36} />
            <h2>{t("सही जन्म समय के बिना गणना सीमित है।")}</h2>
            <p>
              {t(
                "An accurate birth time is needed for charts, Faladesh inputs, Dasha and transit comparisons. No assumed time has been used.",
              )}
            </p>
            <button className="button" onClick={onEditBirth || onBack}>
              {t("जन्म विवरण बदलें")}
            </button>
          </div>
        ) : (
          <>
            {tab === "overview" && (
              <>
                <div className="report-layout">
                  <section className="report-chart">
                    <h2>{t("जन्म कुंडली")} · D1</h2>
                    <ChartViewer chart={chart} />
                    <p className="muted">
                      {t(
                        "खानों में राशि संख्या (1–12) है। भाव लग्न से वामावर्त चलते हैं।",
                      )}
                    </p>
                    <button
                      className="text-button"
                      onClick={() => setTab("charts")}
                    >
                      {t("Explore D9 and Moon charts")}
                      <ArrowRight size={16} />
                    </button>
                  </section>
                  <div className="overview-details">
                    {summary}
                    <p className="report-caption">
                      {t("Timing and transits calculated for")}{" "}
                      {date(asOf, true)}.
                    </p>
                    {natalTable()}
                  </div>
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
                    <p>
                      {t(
                        "Explore the chart factors behind each area of life and read the interpretations available for your chart.",
                      )}
                    </p>
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
                  {categories.map(({ name, Icon }, index) => {
                    const value = name.toLowerCase();
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
                          setCategory(categories[next].name.toLowerCase());
                          topicNav.current
                            ?.querySelectorAll<HTMLButtonElement>("[role=tab]")
                            [next]?.focus();
                        }}
                      >
                        <span className="faladesh-topic-icon">
                          <Icon size={20} aria-hidden="true" />
                        </span>
                        <span>{t(name)}</span>
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
                  <div className="chart-facts">
                    <h3>{t("Chart factors")}</h3>
                    {category === "general" ? (
                      <dl className="fact-list">
                        <div>
                          <dt>{t("Ascendant")}</dt>
                          <dd>{signName(chart.lagna.rashi_number)}</dd>
                        </div>
                        <div>
                          <dt>{t("चंद्र राशि")}</dt>
                          <dd>{moon && signName(moon.rashi_number)}</dd>
                        </div>
                        <div>
                          <dt>{t("Sun sign")}</dt>
                          <dd>{sun && signName(sun.rashi_number)}</dd>
                        </div>
                      </dl>
                    ) : category === "dasha" ? (
                      <p>
                        {activePeriod ? (
                          <>
                            {planetName(activePeriod.lord)} {t("Mahadasha")} ·{" "}
                            {date(activePeriod.start)} –{" "}
                            {date(activePeriod.end)}
                          </>
                        ) : (
                          t(
                            "Dasha calculations are not available in this saved report.",
                          )
                        )}
                      </p>
                    ) : (
                      <dl className="fact-list">
                        {(housesByCategory[category] || []).map((house) => (
                          <div key={house}>
                            <dt>
                              {t("House")} {house} ·{" "}
                              {signName(
                                ((chart.lagna.rashi_number + house - 2) % 12) +
                                  1,
                              )}
                            </dt>
                            <dd>
                              {chart.planets
                                .filter((p) => p.house === house)
                                .map((p) => planetName(p.name))
                                .join(", ") || t("No natal planets")}
                            </dd>
                          </div>
                        ))}
                      </dl>
                    )}
                    <p className="report-caption">
                      {t(
                        "These are calculated chart factors, not predictions. An unoccupied house does not mean an area of life is absent.",
                      )}
                    </p>
                  </div>
                  {rules.map((rule) => (
                    <RuleReading key={rule.id + rule.version} rule={rule} />
                  ))}
                  {!rules.some((r) => !r.demo) && (
                    <div className="reading-status">
                      <Info size={19} />
                      <p>
                        {t(
                          "Guru Ji-approved Faladesh for this topic is not yet available. Personal outcomes and remedies will appear only when an applicable reading is approved.",
                        )}
                      </p>
                    </div>
                  )}
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
                <p className="muted">
                  {t(
                    "Transparent placement checks. These are not final assessments of yoga strength, dosha cancellation or life outcomes.",
                  )}
                </p>
                {report.checks?.length ? (
                  <div className="placement-list">
                    {report.checks.map((check) => {
                      const copy = checkCopy[check.code];
                      return (
                        copy && (
                          <article key={check.code}>
                            <div>
                              <h3>{t(copy[0])}</h3>
                              <span className="check-result">
                                {t(
                                  check.matches
                                    ? "Placement matches"
                                    : "Placement does not match",
                                )}
                              </span>
                            </div>
                            <p>{t(copy[1])}</p>
                            {check.house && (
                              <small>
                                {t("Calculated house")} {check.house}
                              </small>
                            )}
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
                <ExpandedYogaPatterns report={report} />
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
            {tab === "settings" && (
              <section className="reading-panel">
                <h2>{t("पारदर्शी गणना संदर्भ")}</h2>
                <p>
                  {report.place.lat}, {report.place.lon} · {report.place.zone}
                </p>
                <p>
                  UTC: {minuteTimestamp(chart.utc, "UTC", language)} · Swiss
                  Ephemeris {chart.swiss_version}
                </p>
                <dl className="settings-list">
                  {Object.entries(chart.settings).map(([key, value]) => (
                    <div key={key}>
                      <dt>{key}</dt>
                      <dd>{value}</dd>
                    </div>
                  ))}
                </dl>
                <p>
                  {t("इंजन ")}
                  {report.engine} · {t("रिपोर्ट ")}
                  {report.id}
                </p>
              </section>
            )}
          </>
        )}
      </div>
      <div className="limitations">
        {report.limitations.map((text, index) => (
          <p key={index}>
            <Info size={15} />
            {t(text)}
          </p>
        ))}
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
