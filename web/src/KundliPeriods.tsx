import { useEffect, useRef, useState } from "react";
import { Clock, RefreshCw } from "lucide-react";
import type { Report } from "./types";
import type { SaturnTimelineData } from "./kundli-details-types";
import { t, useLanguage } from "./i18n";
import { hindiPlanets, signs } from "./Chart";
import AstroSymbol from "./AstroSymbol";
import TimeRange from "./TimeRange";
import { minuteTimestamp } from "./time-display";
import { EarlierDetails } from "./KundliDetails";
/** planet returns the translated graha name used by supplementary timing panels. */
const planet = (name: string) => t(hindiPlanets[name] || name);

/** YoginiPanel renders Yogini periods and subperiods with their explicit local start/end boundaries. */
export function YoginiPanel({ report }: { report: Report }) {
  const { language } = useLanguage();
  const yogini = report.details?.yogini;
  const activeIndex =
    yogini?.periods.findIndex(
      (p) =>
        Date.parse(p.start) <= Date.parse(yogini.as_of) &&
        Date.parse(yogini.as_of) < Date.parse(p.end),
    ) ?? 0;
  const [cycle, setCycle] = useState(Math.max(0, Math.floor(activeIndex / 8)));
  if (!yogini) return <EarlierDetails />;
  /** format formats a Yogini interval boundary in the recorded birthplace timezone. */
  const format = (value: string) =>
    minuteTimestamp(value, report.place.zone, language);
  /** active tests whether an instant lies within a start-inclusive/end-exclusive period. */
  const active = (start: string, end: string) =>
    Date.parse(start) <= Date.parse(yogini.as_of) &&
    Date.parse(yogini.as_of) < Date.parse(end);
  /** yoginiName selects the translated Yogini period name. */
  const yoginiName = (lord: string) =>
    t(yogini.lords.find((p) => p.lord === lord)?.name || lord);
  /** year extracts a timeline boundary's calendar year in the birthplace timezone. */
  const year = (value: string) =>
    new Intl.DateTimeFormat(language === "hi" ? "hi-IN" : "en-IN", {
      year: "numeric",
      timeZone: report.place.zone,
    }).format(new Date(value));
  return (
    <section className="kundli-details yogini-panel">
      <div className="report-section-heading">
        <div>
          <h2>{t("Yogini Dasha")}</h2>
          <p className="muted">
            {t(
              "Eight Yoginis in a 36-year cycle, with proportional Antardashas.",
            )}
          </p>
        </div>
        <Clock size={23} aria-hidden="true" />
      </div>
      <div className="birth-record-strip">
        <span>
          {t("Running at report creation")}
          <strong>
            {yoginiName(yogini.current_lord)} · {planet(yogini.current_lord)} –{" "}
            {planet(yogini.current_sub_lord)}
          </strong>
        </span>
        <span>{format(yogini.as_of)}</span>
      </div>
      <label className="additional-chart-select">
        {t("Dasha cycle")}
        <select
          value={cycle}
          onChange={(e) => setCycle(Number(e.target.value))}
        >
          {Array.from({ length: yogini.periods.length / 8 }, (_, n) => n).map(
            (n) => (
              <option value={n} key={n}>
                {t("Cycle")} {n + 1} · {year(yogini.periods[n * 8].start)}–
                {year(yogini.periods[n * 8 + 7].end)}
              </option>
            ),
          )}
        </select>
      </label>
      <div className="yogini-periods">
        {yogini.periods.slice(cycle * 8, cycle * 8 + 8).map((p, i) => (
          <details
            key={`${cycle}-${i}`}
            className={active(p.start, p.end) ? "is-current" : ""}
          >
            <summary>
              <AstroSymbol name={p.lord} />
              <span>
                <strong>{yoginiName(p.lord)}</strong>
                <small>
                  {planet(p.lord)}
                  {active(p.start, p.end) && " · " + t("Running")}
                </small>
              </span>
              <span className="yogini-expand-label">{t("Antardasha")}</span>
              <span className="yogini-summary-times">
                <TimeRange start={p.start} end={p.end} format={format} />
              </span>
            </summary>
            <div className="yogini-children">
              {p.antardashas?.map((sub, j) => (
                <article
                  key={j}
                  className={active(sub.start, sub.end) ? "is-current" : ""}
                >
                  <strong>
                    {planet(p.lord)} – {planet(sub.lord)}
                    {active(sub.start, sub.end) && (
                      <small>{t("Running")}</small>
                    )}
                  </strong>
                  <TimeRange start={sub.start} end={sub.end} format={format} />
                </article>
              ))}
            </div>
          </details>
        ))}
      </div>
      <p className="report-caption">
        {t("Birth Dasha balance")}: {yoginiName(yogini.birth_lord)} ·{" "}
        {yogini.balance_years.toFixed(3)} {t("years remaining")}.{" "}
        {yogini.year_days} {t("days per year")}. {t("Birthplace time zone")}:{" "}
        {report.place.zone}.
      </p>
      <details className="detail-disclosure">
        <summary>{t("Dasha calculation reference")}</summary>
        <p>{yogini.method}</p>
      </details>
    </section>
  );
}

// Bounded, in-memory only. Changing the birth details/window changes the key.
const saturnCache = new Map<string, SaturnTimelineData>();
/** SaturnTimeline loads and displays Sade Sati/Dhaiya intervals for the chosen supported year range. */
export function SaturnTimeline({ report }: { report: Report }) {
  const { language } = useLanguage();
  const currentYear = Number(
    new Intl.DateTimeFormat("en", {
      year: "numeric",
      timeZone: report.place.zone,
    }).format(new Date(report.created)),
  );
  const [startYear, setStartYear] = useState(Math.max(1940, currentYear - 10));
  const [endYear, setEndYear] = useState(Math.min(2100, currentYear + 20));
  const [data, setData] = useState<SaturnTimelineData | null>(null);
  const [busy, setBusy] = useState(false),
    [failed, setFailed] = useState(false);
  const controller = useRef<AbortController | null>(null);
  const [loadedWindow, setLoadedWindow] = useState("");
  const key = JSON.stringify([report.birth, startYear, endYear]);
  const valid =
    Number.isInteger(startYear) &&
    Number.isInteger(endYear) &&
    startYear >= 1940 &&
    endYear <= 2100 &&
    endYear > startYear &&
    endYear - startYear <= 30;
  useEffect(() => () => controller.current?.abort(), []);
  /** invalidate aborts and clears a Saturn timeline response when its input range changes. */
  function invalidate() {
    controller.current?.abort();
    setBusy(false);
    setFailed(false);
    setData(null);
    setLoadedWindow("");
  }
  /** calculate requests the validated Saturn range and ignores cancelled or obsolete responses. */
  async function calculate() {
    if (!valid) return;
    invalidate();
    const cached = saturnCache.get(key);
    if (cached) {
      setData(cached);
      setLoadedWindow(key);
      return;
    }
    const pending = new AbortController();
    controller.current = pending;
    setBusy(true);
    try {
      const response = await fetch("/api/kundli/saturn", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          birth: report.birth,
          start_year: startYear,
          end_year: endYear,
        }),
        signal: pending.signal,
      });
      if (!response.ok) throw new Error("Timeline unavailable");
      const out: SaturnTimelineData = await response.json();
      if (
        !Array.isArray(out.periods) ||
        !Number.isFinite(Date.parse(out.start)) ||
        Date.parse(out.end) <= Date.parse(out.start) ||
        out.moon_sign < 1 ||
        out.moon_sign > 12 ||
        !out.periods.every(
          (p) =>
            Date.parse(p.end) > Date.parse(p.start) &&
            p.sign >= 1 &&
            p.sign <= 12,
        )
      )
        throw new Error("Invalid timeline");
      if (pending.signal.aborted) return;
      if (saturnCache.size >= 24)
        saturnCache.delete(saturnCache.keys().next().value!);
      saturnCache.set(key, out);
      setData(out);
      setLoadedWindow(key);
      setBusy(false);
    } catch {
      if (!pending.signal.aborted) {
        setBusy(false);
        setFailed(true);
      }
    }
  }
  /** format formats a Saturn interval boundary in the birthplace timezone without displaying clock seconds. */
  const format = (value: string) =>
    minuteTimestamp(value, report.place.zone, language);
  return (
    <section className="kundli-details saturn-timeline">
      <div className="detail-section-heading">
        <AstroSymbol name="Saturn" />
        <h3>{t("Sade Sati & Dhaiya timeline")}</h3>
      </div>
      <p className="muted">
        {t(
          "Explore any 1–30 year window. Retrograde exits and re-entries are shown separately.",
        )}
      </p>
      <form
        className="saturn-window"
        onSubmit={(e) => {
          e.preventDefault();
          void calculate();
        }}
      >
        <label>
          {t("From year")}
          <input
            type="number"
            min="1940"
            max="2099"
            required
            value={startYear}
            onChange={(e) => {
              invalidate();
              setStartYear(Number(e.target.value));
            }}
          />
        </label>
        <label>
          {t("Until year (exclusive)")}
          <input
            type="number"
            min={startYear + 1}
            max={Math.min(2100, startYear + 30)}
            required
            value={endYear}
            onChange={(e) => {
              invalidate();
              setEndYear(Number(e.target.value));
            }}
          />
        </label>
        <button className="button" disabled={!valid || busy}>
          {busy ? t("Calculating…") : t("Show timeline")}
        </button>
      </form>
      <p className="report-caption">
        {t(
          "Times use your birthplace time zone. A boundary at the edge of the selected window is labelled as clipped.",
        )}
      </p>
      {busy && <p role="status">{t("Calculating Saturn periods…")}</p>}
      {failed && (
        <div className="notice" role="alert">
          {t("The timeline could not be loaded. Please try again.")}
          <button className="text-button" onClick={() => void calculate()}>
            <RefreshCw size={15} aria-hidden="true" />
            {t("Try again")}
          </button>
        </div>
      )}
      {data && loadedWindow === key && (
        <>
          <p className="saturn-window-caption">
            {t("Moon sign")}: {t(signs[data.moon_sign - 1])} · {startYear}–
            {endYear} · {report.place.zone}
          </p>
          {data.periods.length === 0 ? (
            <p className="notice">
              {t("No Sade Sati or Dhaiya periods occur in this window.")}
            </p>
          ) : (
            <div className="saturn-period-list">
              {data.periods.map((p, i) => (
                <article key={i}>
                  <div>
                    <span
                      className={
                        p.kind === "Sade Sati" ? "saturn-kind" : "dhaiya-kind"
                      }
                    >
                      {t(p.kind)}
                    </span>
                    <h4>{t(p.phase)}</h4>
                    <small>
                      {t(signs[p.sign - 1])} · {t("From Moon")} {p.from_moon}
                    </small>
                  </div>
                  <div>
                    <TimeRange start={p.start} end={p.end} format={format} />
                    {(p.start_clipped || p.end_clipped) && (
                      <p className="report-caption">
                        {t(
                          p.start_clipped && p.end_clipped
                            ? "Both boundaries are clipped to the selected window."
                            : p.start_clipped
                              ? "Start is clipped to the selected window."
                              : "End is clipped to the selected window.",
                        )}
                      </p>
                    )}
                  </div>
                </article>
              ))}
            </div>
          )}
        </>
      )}
    </section>
  );
}
