import { lazy, Suspense, useEffect, useRef, useState } from "react";
import {
  ChevronLeft,
  ChevronRight,
  MapPin,
  Moon,
  Sunrise,
  Sunset,
  CalendarDays,
  Clock3,
  Grid2X2,
  Sun,
  Orbit,
  Eclipse,
  Check,
  Minus,
  TriangleAlert,
  ShieldCheck,
} from "lucide-react";
import AstroSymbol from "./AstroSymbol";
import type { CSSProperties } from "react";
import TimeRange from "./TimeRange";
import PanchangOverview from "./PanchangOverview";
import type { PanchangDetailsData } from "./panchang-types";
import { t, useLanguage } from "./i18n";
import type { Place } from "./types";
import "./panchang.css";
import "./panchang-refresh.css";
import {
  MIN_CALCULATION_DATE,
  MAX_CALCULATION_DATE,
  indiaTimeZoneLabel,
} from "./calculation-dates";
const SkyTools = lazy(() => import("./SkyTools"));

type Period = {
  number?: number;
  name: string;
  start: string;
  end: string;
  quality?: string;
  cautions?: string[];
};
type Daily = {
  details?: PanchangDetailsData;
  date: string;
  zone: string;
  weekday: number;
  place: Place;
  sunrise: string;
  sunset: string;
  next_sunrise: string;
  tithi: Period[];
  nakshatra: Period[];
  yoga: Period[];
  karana: Period[];
  timings: Period[];
  choghadiya_day: Period[];
  choghadiya_night: Period[];
  hora_day: Period[];
  hora_night: Period[];
  settings: Record<string, string>;
};
const zone = "Asia/Kolkata";
const minDate = MIN_CALCULATION_DATE,
  maxDate = MAX_CALCULATION_DATE;
/** today returns today's India-local date for daily Panchang controls. */
function today() {
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: zone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(new Date());
  return ["year", "month", "day"]
    .map((k) => parts.find((p) => p.type === k)!.value)
    .join("-");
}
/** savedPlace restores the previously selected daily-tool place when its stored data is usable. */
function savedPlace() {
  try {
    return localStorage.getItem("m4y-panchang-place") || "delhi";
  } catch {
    return "delhi";
  }
}
/** readJSON parses a daily-tool response and raises a useful error for unsuccessful requests. */
async function readJSON<T>(path: string, signal: AbortSignal): Promise<T> {
  const res = await fetch("/api" + path, { signal });
  if (!res.ok) throw new Error("Unable to load Panchang. Please try again.");
  return res.json() as Promise<T>;
}
const tabs = [
  "Panchang",
  "Daily timings",
  "Choghadiya",
  "Hora",
  "Planets",
  "Eclipses",
] as const;

/** Panchang coordinates daily Panchang, timing tabs, planet tools and their relevant selectors. */
export default function Panchang({
  selection,
}: {
  selection?: { index: number; id: number };
}) {
  const { language } = useLanguage();
  const [placeID, setPlaceID] = useState(savedPlace);
  const [place, setPlace] = useState<Place | null>(null);
  const [query, setQuery] = useState("");
  const [editing, setEditing] = useState(false);
  const [suggestions, setSuggestions] = useState<Place[]>([]);
  const [searching, setSearching] = useState(false);
  const [searchError, setSearchError] = useState(false);
  const [active, setActive] = useState(-1);
  const [date, setDate] = useState(today);
  const [convention, setConvention] = useState("upper-limb");
  const [data, setData] = useState<Daily | null>(null);
  const [busy, setBusy] = useState(true);
  const [error, setError] = useState("");
  const [retry, setRetry] = useState(0);
  const [tab, setTab] = useState(0);
  useEffect(() => {
    if (selection && selection.index >= 0 && selection.index < tabs.length)
      setTab(selection.index);
  }, [selection]);
  const isDaily = tab < 4;
  const [now, setNow] = useState(Date.now());
  const input = useRef<HTMLInputElement>(null);
  const listRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const locale = language === "hi" ? "hi-IN" : "en-IN";
  /** placeName selects a readable place label in the active language. */
  const placeName = (p: Place) => (language === "hi" ? p.hindi : p.name);

  useEffect(() => {
    if (active >= 0)
      document
        .getElementById(`daily-place-${active}`)
        ?.scrollIntoView({ block: "nearest" });
  }, [active]);

  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 30000);
    return () => clearInterval(timer);
  }, []);
  useEffect(() => {
    const abort = new AbortController();
    readJSON<Place>("/places/" + encodeURIComponent(placeID), abort.signal)
      .then((p) => {
        if (p.zone === zone) setPlace(p);
        else setPlaceID("delhi");
      })
      .catch(() => {
        if (!abort.signal.aborted && placeID !== "delhi") setPlaceID("delhi");
      });
    return () => abort.abort();
  }, [placeID]);
  useEffect(() => {
    if (!isDaily) return;
    const abort = new AbortController();
    setBusy(true);
    setError("");
    setData(null);
    readJSON<Daily>(
      `/panchang?date=${date}&place_id=${encodeURIComponent(placeID)}&sunrise=${convention}`,
      abort.signal,
    )
      .then((v) => {
        setData(v);
        setPlace(v.place);
      })
      .catch(() => {
        if (!abort.signal.aborted)
          setError("Unable to load Panchang. Please try again.");
      })
      .finally(() => {
        if (!abort.signal.aborted) setBusy(false);
      });
    return () => abort.abort();
  }, [date, placeID, convention, retry, isDaily]);
  useEffect(() => {
    if (!editing) return;
    const abort = new AbortController();
    setSearching(true);
    setSearchError(false);
    setSuggestions([]);
    setActive(-1);
    const timer = setTimeout(() => {
      readJSON<Place[]>("/places?q=" + encodeURIComponent(query), abort.signal)
        .then((places) => setSuggestions(places.filter((p) => p.zone === zone)))
        .catch(() => {
          if (!abort.signal.aborted) setSearchError(true);
        })
        .finally(() => {
          if (!abort.signal.aborted) setSearching(false);
        });
    }, 250);
    return () => {
      clearTimeout(timer);
      abort.abort();
    };
  }, [query, editing]);

  /** selectPlace records a trusted daily-tool place and updates the calculation inputs. */
  function selectPlace(p: Place) {
    setPlaceID(p.id);
    setPlace(p);
    setEditing(false);
    setQuery("");
    setActive(-1);
    try {
      localStorage.setItem("m4y-panchang-place", p.id);
    } catch {
      /* Visit-only preference. */
    }
    input.current?.focus();
  }
  /** moveDate moves the selected daily date within the supported calculation range. */
  function moveDate(offset: number) {
    const value = new Date(date + "T12:00:00Z");
    value.setUTCDate(value.getUTCDate() + offset);
    const next = value.toISOString().slice(0, 10);
    setDate(next < minDate ? minDate : next > maxDate ? maxDate : next);
  }
  /** time formats a daily-tool instant as a minute-precision India-local clock. */
  function time(value: string) {
    // Round display consistently, preserving precise API instants for comparisons.
    return new Intl.DateTimeFormat(locale, {
      timeZone: zone,
      hour: "2-digit",
      minute: "2-digit",
      hour12: true,
    }).format(new Date(Math.round(Date.parse(value) / 60000) * 60000));
  }
  /** stamp adds the local calendar date when a timing extends beyond the selected Panchang date. */
  function stamp(value: string) {
    const d = new Date(Math.round(Date.parse(value) / 60000) * 60000);
    const day = new Intl.DateTimeFormat("en-CA", {
      timeZone: zone,
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    }).formatToParts(d);
    const localDate = ["year", "month", "day"]
      .map((k) => day.find((p) => p.type === k)!.value)
      .join("-");
    return (
      time(value) +
      (localDate !== date
        ? " · " +
          new Intl.DateTimeFormat(locale, {
            timeZone: zone,
            day: "numeric",
            month: "short",
            ...(localDate.slice(0, 4) !== date.slice(0, 4)
              ? { year: "numeric" as const }
              : {}),
          }).format(d)
        : "")
    );
  }
  /** isCurrent tests whether the current instant belongs to a displayed timing interval. */
  const isCurrent = (p: Period) =>
    !!data &&
    now >= Date.parse(data.sunrise) &&
    now < Date.parse(data.next_sunrise) &&
    now >= Date.parse(p.start) &&
    now < Date.parse(p.end);
  /** timingRows renders labelled timing intervals with quality, current-state and overlap information. */
  function timingRows(periods: Period[], showTrack = false) {
    return periods.map((p, i) => (
      <li
        key={i}
        className={isCurrent(p) ? "panchang-current" : ""}
        aria-current={isCurrent(p) ? "time" : undefined}
      >
        <AstroSymbol name={p.name} />
        <div className="daily-period-copy">
          <div className="daily-period-name">
            <strong>{t(p.name)}</strong>
            {isCurrent(p) && <span className="daily-now">{t("Now")}</span>}
          </div>
          {p.quality && (
            <small className={"daily-quality " + p.quality.toLowerCase()}>
              {p.quality === "Favourable" ? (
                <Check size={11} aria-hidden="true" />
              ) : p.quality === "Caution" ? (
                <TriangleAlert size={11} aria-hidden="true" />
              ) : (
                <Minus size={11} aria-hidden="true" />
              )}
              {t(p.quality)}
            </small>
          )}
        </div>
        <TimeRange start={p.start} end={p.end} format={stamp} />
        {!!p.cautions?.length && (
          <small className="daily-cautions">
            <TriangleAlert size={12} aria-hidden="true" />
            <span>
              <span className="sr-only">{t("Overlaps / exceptions")}: </span>
              {p.cautions.map((name) => t(name)).join(" \u00b7 ")}
            </span>
          </small>
        )}
        {showTrack && data && (
          <div className="daily-period-track" aria-hidden="true">
            <span
              style={{
                left: `${Math.max(0, Math.min(100, ((Date.parse(p.start) - Date.parse(data.sunrise)) / (Date.parse(data.sunset) - Date.parse(data.sunrise))) * 100))}%`,
                width: `${(Math.max(0, Math.min(Date.parse(p.end), Date.parse(data.sunset)) - Math.max(Date.parse(p.start), Date.parse(data.sunrise))) / (Date.parse(data.sunset) - Date.parse(data.sunrise))) * 100}%`,
              }}
            />
          </div>
        )}
      </li>
    ));
  }
  /** splitPeriods separates day/night at sunset and shares row tracks so paired intervals align despite warnings or wrapped dates. */
  function splitPeriods(day: Period[], night: Period[]) {
    return (
      <div
        className="daily-split"
        style={{ "--period-rows": Math.max(day.length, night.length) + 1 } as CSSProperties}
      >
        <div className="daily-period-group">
          <div className="daily-group-heading">
            <h3>
              <Sunrise size={17} aria-hidden="true" />
              {t("Daytime")}
            </h3>
            <TimeRange
              start={day[0].start}
              end={day[day.length - 1].end}
              format={stamp}
            />
          </div>
          <ol className="daily-periods">{timingRows(day)}</ol>
        </div>
        <div className="daily-period-group daily-night">
          <div className="daily-group-heading">
            <h3>
              <Moon size={17} aria-hidden="true" />
              {t("Night · until next sunrise")}
            </h3>
            <TimeRange
              start={night[0].start}
              end={night[night.length - 1].end}
              format={stamp}
            />
          </div>
          <ol className="daily-periods">{timingRows(night)}</ol>
        </div>
      </div>
    );
  }
  const locationControl = (
    <div
      className="daily-location"
      onBlur={(e) => {
        if (!e.currentTarget.contains(e.relatedTarget)) {
          setEditing(false);
          setActive(-1);
        }
      }}
    >
      <label htmlFor="daily-city">
        <MapPin size={15} aria-hidden="true" />
        {t("Location in India")}
      </label>
      <input
        id="daily-city"
        ref={input}
        role="combobox"
        aria-autocomplete="list"
        aria-expanded={editing}
        aria-controls={editing ? "daily-places" : undefined}
        aria-activedescendant={
          active >= 0 ? `daily-place-${active}` : undefined
        }
        autoComplete="off"
        maxLength={100}
        value={
          editing
            ? query
            : place
              ? placeName(place)
              : t("New Delhi, Delhi, India")
        }
        onFocus={() => {
          setEditing(true);
          setQuery("");
        }}
        onChange={(e) => {
          setQuery(e.target.value);
          setEditing(true);
        }}
        onKeyDown={(e) => {
          if (e.key === "Escape") {
            setEditing(false);
            setActive(-1);
          }
          if (e.key === "ArrowDown" || e.key === "ArrowUp") {
            e.preventDefault();
            setEditing(true);
            setActive((v) =>
              suggestions.length
                ? v < 0
                  ? e.key === "ArrowDown"
                    ? 0
                    : suggestions.length - 1
                  : (v +
                      (e.key === "ArrowDown" ? 1 : -1) +
                      suggestions.length) %
                    suggestions.length
                : -1,
            );
          }
          if (e.key === "Enter") {
            e.preventDefault();
            if (active >= 0 && suggestions[active])
              selectPlace(suggestions[active]);
          }
        }}
      />
      {editing && (
        <div className="daily-place-popover">
          <ul
            id="daily-places"
            role="listbox"
            aria-label={t("Location suggestions")}
          >
            {suggestions.map((p, i) => (
              <li
                id={`daily-place-${i}`}
                key={p.id}
                role="option"
                aria-selected={i === active}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => selectPlace(p)}
              >
                {placeName(p)}
              </li>
            ))}
          </ul>
          <p role="status">
            {searching
              ? t("Searching locations…")
              : searchError
                ? t("Location search unavailable. Try again.")
                : suggestions.length
                  ? t("Use arrow keys and Enter to choose.")
                  : t("No locations found. Try a nearby town.")}
          </p>
        </div>
      )}
    </div>
  );
  const dateControls = (
    <>
      <div className="daily-date">
        <label htmlFor="daily-date">{t("Date")}</label>
        <input
          id="daily-date"
          type="date"
          min={minDate}
          max={maxDate}
          value={date}
          onChange={(e) => {
            const v = e.target.value;
            if (/^\d{4}-\d{2}-\d{2}$/.test(v) && v >= minDate && v <= maxDate)
              setDate(v);
          }}
        />
      </div>
      <div className="daily-date-nav">
        <button
          type="button"
          aria-label={t("Previous day")}
          disabled={date <= minDate}
          onClick={() => moveDate(-1)}
        >
          <ChevronLeft size={18} />
        </button>
        <button type="button" onClick={() => setDate(today())}>
          {t("Today")}
        </button>
        <button
          type="button"
          aria-label={t("Next day")}
          disabled={date >= maxDate}
          onClick={() => moveDate(1)}
        >
          <ChevronRight size={18} />
        </button>
      </div>
    </>
  );
  return (
    <section
      className="panchang-section section-space"
      id="panchang"
      aria-labelledby="panchang-heading"
    >
      <div className="container">
        <div className="panchang-heading">
          <div>
            <span className="panchang-kicker"><Sun size={14} aria-hidden="true" />{t("Daily almanac")}</span>
            <h2 id="panchang-heading">{t("Panchang & sky tools")}</h2>
            <p>{t("The essentials first. Explore more when you need it.")}</p>
          </div>
          <span className="daily-free"><ShieldCheck size={14} aria-hidden="true" />{t("Free · no sign-in")}</span>
        </div>
        <div className="daily-sheet" aria-busy={isDaily && busy}>
          <div
            className="daily-tabs"
            role="tablist"
            aria-label={t("Daily tools")}
            aria-orientation="horizontal"
          >
            {tabs.map((name, i) => (
              <button
                key={name}
                ref={(el) => {
                  listRefs.current[i] = el;
                }}
                type="button"
                id={`daily-tab-${i}`}
                role="tab"
                tabIndex={i === tab ? 0 : -1}
                aria-selected={i === tab}
                aria-controls="daily-panel"
                onClick={() => setTab(i)}
                onKeyDown={(e) => {
                  let next = tab;
                  if (e.key === "ArrowRight" || e.key === "ArrowDown")
                    next = (tab + 1) % tabs.length;
                  else if (e.key === "ArrowLeft" || e.key === "ArrowUp")
                    next = (tab + tabs.length - 1) % tabs.length;
                  else if (e.key === "Home") next = 0;
                  else if (e.key === "End") next = tabs.length - 1;
                  else return;
                  e.preventDefault();
                  setTab(next);
                  listRefs.current[next]?.focus();
                }}
              >
                {(() => {
                  const Icon = [
                    CalendarDays,
                    Clock3,
                    Grid2X2,
                    Sun,
                    Orbit,
                    Eclipse,
                  ][i];
                  return <Icon size={17} aria-hidden="true" />;
                })()}
                <span>{t(name)}</span>
              </button>
            ))}
          </div>
          <div
            id="daily-panel"
            role="tabpanel"
            aria-labelledby={`daily-tab-${tab}`}
            tabIndex={0}
          >
            {isDaily && (
              <div className="daily-controls">
                {locationControl}
                {dateControls}
              </div>
            )}
            {isDaily && !data && (
              <div className="daily-panel-content">
                {busy && (
                  <p role="status" className="daily-status">
                    {t("Calculating your local Panchang…")}
                  </p>
                )}
                {error && (
                  <div className="daily-status" role="alert">
                    <p>{t(error)}</p>
                    <button
                      type="button"
                      className="text-button"
                      onClick={() => setRetry((v) => v + 1)}
                    >
                      {t("Try again")}
                    </button>
                  </div>
                )}
              </div>
            )}
            {isDaily && data && (
              <>
                <div className="daily-solar">
                  <div className="daily-date-context">
                    <strong>
                      <time dateTime={data.date}>{new Intl.DateTimeFormat(locale, {
                        weekday: "long",
                        day: "numeric",
                        month: "long",
                        year: "numeric",
                        timeZone: zone,
                      }).format(new Date(data.date + "T12:00:00Z"))}</time>
                    </strong>
                    <small>
                      <MapPin size={12} aria-hidden="true" />
                      {placeName(data.place)} ·{" "}
                      {t(indiaTimeZoneLabel(data.sunrise))}
                    </small>
                  </div>
                  <span>
                    <Sunrise size={17} aria-hidden="true" />
                    {t("Sunrise")} <b>{time(data.sunrise)}</b>
                  </span>
                  <span>
                    <Sunset size={17} aria-hidden="true" />
                    {t("Sunset")} <b>{time(data.sunset)}</b>
                  </span>
                </div>
                <div className="daily-results">
                  {tab === 0 && <PanchangOverview data={data} stamp={stamp} isCurrent={isCurrent} />}
                  {tab === 1 && (
                    <>
                      <p className="daily-explainer">
                        {t(
                          "Daytime periods change with sunrise and sunset; they are not fixed clock times.",
                        )}
                      </p>
                      <ul className="daily-periods daily-timings">
                        {timingRows(data.timings, true)}
                      </ul>
                      <div className="daily-track-key">
                        <span>
                          <Sunrise size={14} aria-hidden="true" />
                          {t("Sunrise")} {time(data.sunrise)}
                        </span>
                        <span>
                          <Sunset size={14} aria-hidden="true" />
                          {t("Sunset")} {time(data.sunset)}
                        </span>
                      </div>
                      <p className="daily-boundary">
                        {t(
                          "Each bar shows the period within local daylight. Overlapping periods remain separate.",
                        )}
                      </p>
                    </>
                  )}
                  {tab === 2 && (
                    <>
                      <p className="daily-explainer">
                        {t(
                          "Eight daytime and eight nighttime periods. Check the overlaps before choosing a time.",
                        )}
                      </p>
                      {splitPeriods(data.choghadiya_day, data.choghadiya_night)}
                    </>
                  )}
                  {tab === 3 && (
                    <>
                      <p className="daily-explainer">
                        {t(
                          "Twelve daytime and twelve nighttime planetary hours, each scaled to local daylight or night.",
                        )}
                      </p>
                      {splitPeriods(data.hora_day, data.hora_night)}
                    </>
                  )}
                </div>
              </>
            )}
            {!isDaily && (
              <div className="daily-panel-content">
                <Suspense
                  fallback={
                    <p className="daily-status" role="status">
                      {t("Opening your workspace…")}
                    </p>
                  }
                >
                  <SkyTools
                    mode={tab === 4 ? "planets" : "eclipses"}
                    date={date}
                    placeID={placeID}
                    place={place}
                    onDateChange={setDate}
                    locationControl={locationControl}
                    dateControls={dateControls}
                  />
                </Suspense>
              </div>
            )}
          </div>
        </div>
        <div className="daily-foot">
          <p>
            {t(
              isDaily
                ? "All times are local to the selected place. Night timings show the next date after midnight. These general periods do not replace an event-specific Muhurat."
                : "Sky event times use India's local clock, including historical timezone changes.",
            )}
          </p>
          {isDaily && (
            <details>
              <summary>{t("Calculation settings")}</summary>
              <div className="daily-settings">
                <label htmlFor="daily-convention">{t("Sunrise method")}</label>
                <select
                  id="daily-convention"
                  value={convention}
                  onChange={(e) => setConvention(e.target.value)}
                >
                  <option value="upper-limb">
                    {t("Upper limb · with refraction")}
                  </option>
                  <option value="hindu">
                    {t("Hindu · centre, without refraction")}
                  </option>
                </select>
                <p>
                  {t(
                    "Lahiri sidereal positions. Sea-level horizon; terrain and actual weather are not modelled. Display times are rounded to the nearest minute.",
                  )}
                </p>
                <p>
                  {t(
                    "Moonrise and Moonset use the upper limb with refraction at sea level. Cross-provider Tithi and Karana timing differences remain under review.",
                  )}
                </p>
                {data && (
                  <small>
                    Swiss Ephemeris {data.settings.swiss_version} ·{" "}
                    {data.settings.engine} · {data.zone}
                  </small>
                )}
              </div>
            </details>
          )}
        </div>
      </div>
    </section>
  );
}
