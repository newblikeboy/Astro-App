import { useEffect, useRef, useState } from "react";
import {
  ArrowDown,
  ChevronDown,
  ChevronRight,
  Clock,
  RefreshCw,
} from "lucide-react";
import AstroSymbol from "./AstroSymbol";
import { hindiPlanets } from "./Chart";
import { t, useLanguage } from "./i18n";
import type { DashaPeriod, DashaStep, Report } from "./types";
import "./dasha.css";

const levels = [
  "Mahadasha",
  "Antardasha",
  "Pratyantar Dasha",
  "Sukshma Dasha",
  "Prana Dasha",
];
type Branch = {
  method: string;
  level: number;
  path: number[];
  parent: DashaPeriod | null;
  ancestors: DashaStep[];
  periods: DashaPeriod[];
  current_path: DashaStep[];
  as_of: string;
  timezone: string;
  year_days: number;
  birth_lord: string;
  balance_years: number;
};
/** keyOf builds the stable cache key for a Dasha branch's ancestor indices. */
const keyOf = (path: number[]) => path.join("/");
/** rowID builds a stable DOM identifier for a Dasha period row. */
const rowID = (path: number[]) => `dasha-row-${path.join("-")}`;
/** childrenID builds the disclosure target identifier for a period's nested subperiods. */
const childrenID = (path: number[]) => `dasha-children-${path.join("-")}`;
/** planetName selects the displayed planet name in the current interface language. */
const planetName = (name: string) => t(hindiPlanets[name] || name);
/** validPeriod rejects unknown planet names and invalid or reversed Dasha intervals. */
const validPeriod = (p: DashaPeriod) =>
  Boolean(hindiPlanets[p?.lord]) &&
  Number.isFinite(Date.parse(p.start)) &&
  Date.parse(p.end) > Date.parse(p.start);

/** DashaPanel renders the five-level expandable Vimshottari sequence with explicit start/end times. */
export default function DashaPanel({ report }: { report: Report }) {
  const { language } = useLanguage();
  const [branches, setBranches] = useState(new Map<string, Branch>());
  const [openPath, setOpenPath] = useState<number[]>([]);
  const [loadingPath, setLoadingPath] = useState<number[] | null>(null);
  const [failedPath, setFailedPath] = useState<number[] | null>(null);
  const request = useRef<AbortController | null>(null);
  const cache = useRef(new Map<string, Branch>());
  const reportRef = useRef(report);
  const root = branches.get("");

  /** date formats a Dasha boundary date or minute-precision clock in the recorded birthplace timezone. */
  const date = (value: string, clock = false) =>
    new Intl.DateTimeFormat(language === "hi" ? "hi-IN" : "en-IN", {
      timeZone: report.place.zone,
      ...(clock
        ? {
            hour: "2-digit",
            minute: "2-digit",
            hourCycle: "h23" as const,
            timeZoneName: "shortOffset" as const,
          }
        : { day: "2-digit", month: "short" as const, year: "numeric" }),
    }).format(new Date(value));
  /** stamp renders a semantic timestamp with separate local date and clock labels. */
  const stamp = (value: string) => (
    <time dateTime={value}>
      <span>{date(value)}</span>
      <small>{date(value, true)}</small>
    </time>
  );
  /** times renders explicitly labelled start and end boundaries for one Dasha period. */
  const times = (p: DashaPeriod) => (
    <dl className="dasha-times">
      <div>
        <dt>{t("Start time")}</dt>
        <dd>{stamp(p.start)}</dd>
      </div>
      <div>
        <dt>{t("End time")}</dt>
        <dd>{stamp(p.end)}</dd>
      </div>
    </dl>
  );

  // Keep the source report immutable. Only requested sibling lists are retained.
  async function fetchBranch(
    path: number[],
    controller: AbortController,
  ): Promise<Branch> {
    const cached = cache.current.get(keyOf(path));
    if (cached) return cached;
    const response = await fetch("/api/kundli/dasha", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        birth: reportRef.current.birth,
        path,
        as_of: reportRef.current.dasha?.as_of || reportRef.current.created,
      }),
      signal: controller.signal,
    });
    if (!response.ok) throw new Error("Dasha request failed");
    const out: Branch = await response.json();
    if (
      !Array.isArray(out.path) ||
      out.level !== path.length + 1 ||
      keyOf(out.path) !== keyOf(path) ||
      out.timezone !== reportRef.current.place.zone ||
      !hindiPlanets[out.birth_lord] ||
      !Number.isFinite(out.balance_years) ||
      out.balance_years < 0 ||
      out.balance_years > 20 ||
      !Array.isArray(out.ancestors) ||
      out.ancestors.length !== path.length ||
      !out.ancestors.every(validPeriod) ||
      !Array.isArray(out.periods) ||
      out.periods.length < 9 ||
      out.periods.length > 18 ||
      !out.periods.every(validPeriod) ||
      !Array.isArray(out.current_path) ||
      out.current_path.length !== 5 ||
      !out.current_path.every(validPeriod)
    )
      throw new Error("Invalid Dasha response");
    if (controller.signal.aborted) throw new Error("Aborted");
    // Retain ancestors of the visible branch when evicting old explorations.
    if (cache.current.size >= 64) {
      const oldest = [...cache.current.keys()].find(
        (key) =>
          key !== "" &&
          key !== keyOf(path) &&
          !keyOf(path).startsWith(key + "/"),
      );
      if (oldest !== undefined) cache.current.delete(oldest);
    }
    cache.current.set(keyOf(path), out);
    setBranches(new Map(cache.current));
    return out;
  }
  /** cancelRequest aborts an obsolete branch fetch and clears its transient loading/error state. */
  function cancelRequest() {
    request.current?.abort();
    setFailedPath(null);
    setLoadingPath(null);
  }
  /** reveal loads and validates a Dasha branch, optionally restoring ancestors and keyboard focus. */
  async function reveal(path: number[], ancestors = false, focusID?: string) {
    cancelRequest();
    const controller = new AbortController();
    request.current = controller;
    if (path.length) setOpenPath(path);
    let pending = path;
    try {
      // The running shortcut opens every ancestor, so no missing layer is skipped.
      const paths = ancestors
        ? path.map((_, i) => path.slice(0, i + 1))
        : [path];
      for (pending of paths) {
        if (controller.signal.aborted) return;
        setLoadingPath(pending);
        await fetchBranch(pending, controller);
      }
      if (controller.signal.aborted) return;
      setLoadingPath(null);
      if (focusID)
        requestAnimationFrame(() => {
          if (controller.signal.aborted) return;
          const target = document.getElementById(focusID);
          target?.focus({ preventScroll: true });
          target?.scrollIntoView({ block: "center", behavior: "instant" });
        });
    } catch {
      if (!controller.signal.aborted) {
        setLoadingPath(null);
        setFailedPath(pending);
      }
    }
  }
  /** toggle opens or closes a nested Dasha period while cancelling superseded requests. */
  function toggle(path: number[], expanded: boolean) {
    cancelRequest();
    if (expanded) setOpenPath(path.slice(0, -1));
    else void reveal(path);
  }
  useEffect(() => {
    if (reportRef.current.chart && !reportRef.current.birth.unknown_time)
      void reveal([]);
    return () => request.current?.abort();
  }, []);

  /** status renders the requested branch's loading or retryable failure state. */
  const status = (path: number[]) => (
    <div className="dasha-branch-status">
      {failedPath !== null && keyOf(failedPath) === keyOf(path) ? (
        <div className="dasha-load-error" role="alert">
          <p>{t("Dasha periods could not be loaded. Please try again.")}</p>
          <button
            type="button"
            onClick={() =>
              void reveal(
                openPath,
                openPath.length > 0,
                openPath.length ? rowID(openPath) : "dasha-root-heading",
              )
            }
          >
            <RefreshCw size={15} aria-hidden="true" />
            {t("Try again")}
          </button>
        </div>
      ) : (
        <p role="status">
          <RefreshCw size={15} aria-hidden="true" />
          {t("Loading Dasha periods…")}
        </p>
      )}
    </div>
  );
  /** renderBranch recursively renders the loaded period rows while retaining the complete planetary sequence. */
  function renderBranch(path: number[], lords: string[]) {
    const branch = branches.get(keyOf(path));
    if (!branch) return status(path);
    return (
      <ol
        className="dasha-period-list"
        data-level={branch.level}
        data-path={keyOf(path)}
      >
        {branch.periods.map((p, index) => {
          const childPath = [...path, index];
          const expanded =
            path.length < 4 &&
            openPath.length >= childPath.length &&
            keyOf(openPath.slice(0, childPath.length)) === keyOf(childPath);
          const running = root?.current_path.some(
            (step) =>
              step.level === branch.level &&
              step.start === p.start &&
              step.end === p.end,
          );
          const sequence = [...lords, p.lord].map(planetName).join(" – ");
          const identity = (
            <>
              <span className="dasha-planet-mark">
                <AstroSymbol name={p.lord} />
              </span>
              <span className="dasha-planet-label">
                <small>
                  {t(levels[branch.level - 1])}
                  {running && (
                    <span className="dasha-running">{t("Running")}</span>
                  )}
                </small>
                <strong>{sequence}</strong>
              </span>
            </>
          );
          return (
            <li
              key={p.start}
              className={`dasha-item${expanded ? " is-open" : ""}${running ? " is-current" : ""}`}
            >
              <div className="dasha-row">
                {path.length < 4 ? (
                  <button
                    id={rowID(childPath)}
                    className="dasha-toggle"
                    type="button"
                    aria-expanded={expanded}
                    aria-controls={childrenID(childPath)}
                    onClick={() => toggle(childPath, expanded)}
                  >
                    {identity}
                    <ChevronDown
                      className="dasha-disclosure-icon"
                      size={18}
                      aria-hidden="true"
                    />
                  </button>
                ) : (
                  <div
                    id={rowID(childPath)}
                    className="dasha-leaf"
                    tabIndex={-1}
                  >
                    {identity}
                  </div>
                )}
                {times(p)}
              </div>
              {path.length < 4 && (
                <div
                  id={childrenID(childPath)}
                  className="dasha-children"
                  hidden={!expanded}
                >
                  {expanded && (
                    <>
                      <div className="dasha-child-heading">
                        <span>{t(levels[branch.level])}</span>
                        <small>{t("9 periods")}</small>
                      </div>
                      {renderBranch(childPath, [...lords, p.lord])}
                    </>
                  )}
                </div>
              )}
            </li>
          );
        })}
      </ol>
    );
  }

  if (!report.chart || report.birth.unknown_time)
    return (
      <section className="dasha-panel">
        <h2>{t("Vimshottari Dasha")}</h2>
        <p className="notice">
          {t("A known birth time is required for Dasha calculations.")}
        </p>
      </section>
    );
  return (
    <section className="dasha-panel dasha-explorer">
      <div className="report-section-heading">
        <div>
          <h2>{t("Vimshottari Dasha")}</h2>
          <p className="muted">
            {t(
              "Open a planet to explore its next Dasha layer, all the way to Prana.",
            )}
          </p>
        </div>
        <Clock size={22} aria-hidden="true" />
      </div>
      {root && (
        <div className="dasha-running-card">
          <div className="dasha-live-heading">
            <h3>
              <span className="dasha-live-dot" aria-hidden="true" />
              {t("Running at report creation")}
            </h3>
            <span>
              {date(root.as_of)} · {date(root.as_of, true)}
            </span>
          </div>
          <ol
            className="dasha-current-chain"
            aria-label={t("Running Dasha sequence")}
          >
            {root.current_path.map((p, i) => (
              <li key={p.level}>
                <small>{t(levels[i])}</small>
                <span>
                  <AstroSymbol name={p.lord} />
                  <strong>{planetName(p.lord)}</strong>
                </span>
                {i < 4 && <ChevronRight size={14} aria-hidden="true" />}
              </li>
            ))}
          </ol>
          <button
            className="dasha-jump-current"
            type="button"
            disabled={loadingPath !== null}
            onClick={() =>
              void reveal(
                root.current_path[4].path.slice(0, 4),
                true,
                rowID(root.current_path[4].path),
              )
            }
          >
            {t("Open running Dasha layers")}
            <ArrowDown size={15} aria-hidden="true" />
          </button>
        </div>
      )}
      <div className="dasha-clock-note">
        <Clock size={15} aria-hidden="true" />
        <div>
          <strong>
            {t("Birthplace time zone")}: {report.place.zone}
          </strong>
          <p>
            {t(
              "Start and end times are shown to the minute in the birthplace time zone. A period ends exactly when the next begins.",
            )}
          </p>
        </div>
      </div>
      <div className="dasha-explore-heading">
        <h3 id="dasha-root-heading" tabIndex={-1}>
          {t("Mahadasha")}
        </h3>
        {openPath.length > 0 && (
          <button
            type="button"
            onClick={() => {
              cancelRequest();
              setOpenPath([]);
              document
                .getElementById("dasha-root-heading")
                ?.focus({ preventScroll: true });
            }}
          >
            {t("Collapse all")}
          </button>
        )}
      </div>
      <div className="dasha-tree">{renderBranch([], [])}</div>
      {loadingPath !== null && root && (
        <span className="sr-only" role="status">
          {t("Loading Dasha periods…")}
        </span>
      )}
      {root && (
        <details className="dasha-method">
          <summary>{t("Dasha calculation reference")}</summary>
          <p>
            {t(
              "Subperiod duration = full parent duration × the subperiod lord’s Vimshottari years ÷ 120. Each sequence starts with the parent lord.",
            )}
          </p>
          <p>
            {t(
              "The first period begins before birth. Subperiods use its full duration, not the remaining birth balance.",
            )}
          </p>
          <p>
            {root.year_days} {t("days per year")} ·{" "}
            {t("Start inclusive; end exclusive")}
          </p>
          <p>
            {t("At birth")}: {planetName(root.birth_lord)} ·{" "}
            {root.balance_years.toFixed(3)} {t("years remaining")}
          </p>
          <code>{root.method}</code>
        </details>
      )}
    </section>
  );
}
