import { useEffect, useRef, useState } from "react";
import { CalendarDays, ChevronLeft, ChevronRight, Heart, Info, LoaderCircle, MapPin, Search, Sparkles } from "lucide-react";
import { PortalFooter, PortalHeader } from "./PortalShell";
import { t, useLanguage } from "./i18n";
import { followRoute, navigate, useAppLocation } from "./routing";
import { MIN_CALCULATION_DATE, MAX_CALCULATION_DATE, indiaTimeZoneLabel } from "./calculation-dates";
import { minuteTimestamp } from "./time-display";
import TimeRange from "./TimeRange";
import type { Place } from "./types";
import "./vivah.css";

type VivahWindow = { start: string; end: string; tithi: string; nakshatra: string; yoga: string; karana: string; lagna: string };
type VivahDay = { date: string; windows: VivahWindow[]; reasons: string[] };
type VivahMonth = { month: string; zone: string; place: Place; days: VivahDay[]; settings: Record<string, string> };
const minMonth = MIN_CALCULATION_DATE.slice(0, 7);
const maxMonth = MAX_CALCULATION_DATE.slice(0, 7);

/** currentMonth returns the India-local civil month, bounded to the calculation range. */
function currentMonth() {
  const parts = new Intl.DateTimeFormat("en", { timeZone: "Asia/Kolkata", year: "numeric", month: "2-digit" }).formatToParts(new Date());
  const month = `${parts.find((p) => p.type === "year")!.value}-${parts.find((p) => p.type === "month")!.value}`;
  return month < minMonth ? minMonth : month > maxMonth ? maxMonth : month;
}

/** validMonth checks canonical year-month input before it becomes a URL or API argument. */
function validMonth(value: string | null): value is string {
  return !!value && /^\d{4}-(0[1-9]|1[0-2])$/.test(value) && value >= minMonth && value <= maxMonth;
}

/** vivahJSON requests abortable read-only data, revalidates marriage rules after updates and returns a noninternal failure. */
async function vivahJSON<T>(path: string, signal: AbortSignal): Promise<T> {
  const response = await fetch(`/api${path}`, { signal, ...(path.startsWith("/panchang/vivah?") ? { cache: "no-store" as const } : {}) });
  if (!response.ok) throw new Error("Marriage calculations unavailable. Please try again.");
  return response.json() as Promise<T>;
}

/** VivahPlace selects a trusted India city with keyboard and pointer suggestions. */
function VivahPlace({ place, onSelect }: { place: Place | null; onSelect: (place: Place) => void }) {
  const { language } = useLanguage();
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const [places, setPlaces] = useState<Place[]>([]);
  const [active, setActive] = useState(-1);
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState(false);
  const input = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (!open) return;
    const abort = new AbortController();
    setBusy(true);
    setFailed(false);
    setPlaces([]);
    setActive(-1);
    const timer = setTimeout(() => {
      vivahJSON<Place[]>(`/places?q=${encodeURIComponent(query)}`, abort.signal)
        .then((values) => { if (!abort.signal.aborted) setPlaces(values.filter((p) => p.zone === "Asia/Kolkata")); })
        .catch(() => { if (!abort.signal.aborted) setFailed(true); })
        .finally(() => { if (!abort.signal.aborted) setBusy(false); });
    }, 250);
    return () => { clearTimeout(timer); abort.abort(); };
  }, [open, query]);
  useEffect(() => {
    if (active >= 0) document.getElementById(`vivah-place-${active}`)?.scrollIntoView({ block: "nearest" });
  }, [active]);
  /** label selects the shared language's trusted city name. */
  const label = (value: Place) => language === "hi" && value.hindi ? value.hindi : value.name;
  /** select applies a trusted suggestion and restores focus without reopening search. */
  function select(value: Place) {
    onSelect(value);
    setOpen(false);
    setActive(-1);
    input.current?.focus();
  }
  return <div className="vivah-location" onBlur={(event) => { if (!event.currentTarget.contains(event.relatedTarget)) setOpen(false); }}>
    <label htmlFor="vivah-city"><MapPin size={15} aria-hidden="true" />{t("Wedding location in India")}</label>
    <input id="vivah-city" ref={input} role="combobox" autoComplete="off" maxLength={100}
      aria-autocomplete="list" aria-expanded={open} aria-controls={open ? "vivah-places" : undefined}
      aria-activedescendant={open && active >= 0 ? `vivah-place-${active}` : undefined}
      value={open ? query : place ? label(place) : ""} placeholder={t("Search for a city")}
      onFocus={() => { setQuery(""); setOpen(true); }}
      onChange={(event) => { setQuery(event.target.value); setOpen(true); }}
      onKeyDown={(event) => {
        if (event.key === "Escape") { setOpen(false); setActive(-1); }
        if (event.key === "ArrowDown" || event.key === "ArrowUp") {
          event.preventDefault(); setOpen(true);
          setActive((value) => places.length ? value < 0 ? event.key === "ArrowDown" ? 0 : places.length - 1 : (value + (event.key === "ArrowDown" ? 1 : -1) + places.length) % places.length : -1);
        }
        if (event.key === "Enter" && open) { event.preventDefault(); if (places[active]) select(places[active]); }
      }} />
    {open && <div className="vivah-suggestions">
      <ul id="vivah-places" role="listbox" aria-label={t("Location suggestions")}>
        {places.map((value, index) => <li id={`vivah-place-${index}`} key={value.id} role="option" aria-selected={active === index}
          onMouseDown={(event) => event.preventDefault()} onClick={() => select(value)}>{label(value)}</li>)}
      </ul>
      <p role="status">{t(busy ? "Searching locations…" : failed ? "Location search unavailable. Try again." : places.length ? "Use arrow keys and Enter to choose." : "No locations found. Try a nearby town.")}</p>
    </div>}
  </div>;
}

/** VivahMuhurat presents URL-addressable monthly marriage shortlists with cancellable requests and English/Hindi evidence. */
export default function VivahMuhurat() {
  const { language } = useLanguage();
  const route = useAppLocation();
  const requestedMonth = route.searchParams.get("month");
  const month = validMonth(requestedMonth) ? requestedMonth : currentMonth();
  const placeID = route.searchParams.get("place_id") || "delhi";
  const convention = route.searchParams.get("sunrise") === "hindu" ? "hindu" : "upper-limb";
  const [data, setData] = useState<VivahMonth | null>(null);
  const [place, setPlace] = useState<Place | null>(null);
  const [busy, setBusy] = useState(true);
  const [error, setError] = useState(false);
  const [retry, setRetry] = useState(0);
  const [showExcluded, setShowExcluded] = useState(false);
  const resultKey = `${month}/${placeID}/${convention}`;
  const [loadedKey, setLoadedKey] = useState("");
  const locale = language === "hi" ? "hi-IN" : "en-IN";
  const currentData = loadedKey === resultKey ? data : null;
  useEffect(() => {
    // Apply the page title after the shared language provider's default title effect.
    const frame = requestAnimationFrame(() => { document.title = `${t("Vivah Muhurat")} | Mishra 4 You`; });
    return () => cancelAnimationFrame(frame);
  }, [language]);
  useEffect(() => {
    const abort = new AbortController();
    setPlace(null);
    vivahJSON<Place>(`/places/${encodeURIComponent(placeID)}`, abort.signal)
      .then((value) => { if (!abort.signal.aborted) setPlace(value.zone === "Asia/Kolkata" ? value : null); })
      .catch(() => {});
    return () => abort.abort();
  }, [placeID]);
  useEffect(() => {
    const abort = new AbortController();
    setBusy(true); setError(false); setData(null); setShowExcluded(false);
    const params = new URLSearchParams({ month, place_id: placeID, sunrise: convention });
    vivahJSON<VivahMonth>(`/panchang/vivah?${params}`, abort.signal)
      .then((value) => { if (!abort.signal.aborted) { setData(value); setPlace(value.place); setLoadedKey(resultKey); } })
      .catch(() => { if (!abort.signal.aborted) setError(true); })
      .finally(() => { if (!abort.signal.aborted) setBusy(false); });
    return () => abort.abort();
  }, [month, placeID, convention, retry, resultKey]);
  /** update makes filters shareable and lets Back/Forward restore the corresponding search. */
  function update(next: { month?: string; place_id?: string; sunrise?: string }) {
    const params = new URLSearchParams({ month, place_id: placeID, sunrise: convention, ...next });
    navigate(`/vivah-muhurat?${params}`);
  }
  /** moveMonth advances a supported calendar month using UTC arithmetic. */
  function moveMonth(offset: number) {
    const day = new Date(`${month}-01T12:00:00Z`);
    day.setUTCMonth(day.getUTCMonth() + offset);
    const value = day.toISOString().slice(0, 7);
    if (validMonth(value)) update({ month: value });
  }
  /** dateLabel formats a civil date without depending on the visitor's browser timezone. */
  function dateLabel(value: string, options: Intl.DateTimeFormatOptions) {
    return new Intl.DateTimeFormat(locale, { timeZone: "Asia/Kolkata", ...options }).format(new Date(`${value}T12:00:00+05:30`));
  }
  /** stamp uses the shared minute-only formatter, preserving the next date after midnight. */
  const stamp = (value: string) => minuteTimestamp(value, "Asia/Kolkata", language);
  const days = currentData?.days.filter((day) => day.windows.length > 0) || [];
  const excluded = currentData?.days.filter((day) => day.windows.length === 0) || [];
  const placeName = place ? (language === "hi" && place.hindi ? place.hindi : place.name) : t("Choose a location");
  return <div className="vivah-page">
    <a className="skip" href="#main">{t("Skip to main content")}</a>
    <PortalHeader />
    <main id="main" className="vivah-main">
      <header className="vivah-hero">
        <div>
          <span className="vivah-eyebrow"><Heart size={15} aria-hidden="true" />{t("A new beginning")}</span>
          <h1>{t("Vivah Muhurat")}</h1>
          <p>{t("Find marriage timing windows for your wedding city and month, with the Panchang details behind each date.")}</p>
        </div>
        <div className="vivah-motif" aria-hidden="true"><span /><Heart size={40} strokeWidth={1} /><span /></div>
      </header>
      <div className="vivah-controls" role="group" aria-label={t("Marriage search filters")}>
        <VivahPlace place={place} onSelect={(value) => { setPlace(value); update({ place_id: value.id }); }} />
        <div><label htmlFor="vivah-month"><CalendarDays size={15} aria-hidden="true" />{t("Wedding month")}</label>
          <input id="vivah-month" type="month" min={minMonth} max={maxMonth} value={month}
            onChange={(event) => { if (validMonth(event.target.value)) update({ month: event.target.value }); }} /></div>
        <div><label htmlFor="vivah-sunrise">{t("Sunrise method")}</label>
          <select id="vivah-sunrise" value={convention} onChange={(event) => update({ sunrise: event.target.value })}>
            <option value="upper-limb">{t("Upper limb · with refraction")}</option><option value="hindu">{t("Hindu · centre, without refraction")}</option>
          </select></div>
      </div>
      <aside className="vivah-note"><Info size={19} aria-hidden="true" /><p>{t("These are preliminary windows. Confirm the final ceremony time with your priest using both birth charts, a suitable Lagna and your family tradition.")}</p></aside>
      <section className="vivah-results" aria-labelledby="vivah-results-heading" aria-busy={busy || (!currentData && !error)}>
        <div className="vivah-results-top"><div><h2 id="vivah-results-heading">{dateLabel(`${month}-01`, { month: "long", year: "numeric" })}</h2>
          <p>{placeName} · {indiaTimeZoneLabel(`${month}-01T12:00:00+05:30`)}</p></div>
          <div className="vivah-month-nav">
            <button type="button" aria-label={t("Previous month")} disabled={month <= minMonth} onClick={() => moveMonth(-1)}><ChevronLeft size={18} /></button>
            <button type="button" onClick={() => update({ month: currentMonth() })}>{t("This month")}</button>
            <button type="button" aria-label={t("Next month")} disabled={month >= maxMonth} onClick={() => moveMonth(1)}><ChevronRight size={18} /></button>
          </div>
        </div>
        {busy || (!currentData && !error) ? <div className="vivah-status" role="status"><LoaderCircle size={25} className="spin" aria-hidden="true" /><p>{t("Calculating marriage windows…")}</p></div>
          : error ? <div className="vivah-status" role="alert"><p>{t("Marriage calculations unavailable. Please try again.")}</p><button className="button" type="button" onClick={() => setRetry((value) => value + 1)}>{t("Try again")}</button></div>
          : <>
            <p className="vivah-count" role="status"><Sparkles size={16} aria-hidden="true" />{t("{v1} dates with preliminary windows", { v1: days.length })}</p>
            {!days.length && <div className="vivah-status"><Search size={30} aria-hidden="true" /><h3>{t("No windows meet the selected rules this month")}</h3><p>{t("Explore another month, or open the excluded dates to see the checks behind this result.")}</p><button className="button" type="button" disabled={month >= maxMonth} onClick={() => moveMonth(1)}>{t("Explore next month")}</button></div>}
            <div className="vivah-date-list">{days.map((day) => <article className="vivah-day" key={day.date}>
              <header><div className="vivah-date-tile"><strong>{dateLabel(day.date, { day: "2-digit" })}</strong><span>{dateLabel(day.date, { month: "short" })}</span></div>
                <div><h3>{dateLabel(day.date, { weekday: "long" })}</h3><span>{t("Preliminary marriage windows")}</span></div></header>
              <ol className="vivah-windows">{day.windows.map((window) => <li key={window.start}>
                <TimeRange start={window.start} end={window.end} format={stamp} />
                <dl><div><dt>{t("Nakshatra")}</dt><dd>{t(window.nakshatra)}</dd></div><div><dt>{t("Tithi")}</dt><dd>{t(window.tithi)}</dd></div>
                  <div><dt>{t("Yoga")}</dt><dd>{t(window.yoga)}</dd></div><div><dt>{t("Karana")}</dt><dd>{t(window.karana)}</dd></div>
                  <div className="vivah-lagna"><dt>{t("Ascendant")}</dt><dd>{window.lagna ? t(window.lagna) : t("Unavailable")}</dd></div></dl>
              </li>)}</ol>
            </article>)}</div>
            <details className="vivah-excluded" open={showExcluded} onToggle={(event) => setShowExcluded(event.currentTarget.open)}>
              <summary>{t("Excluded dates and reasons")} <span>({excluded.length})</span></summary>
              <p>{t("Reasons can apply to different parts of the day. No interval of at least five minutes passes every selected check.")}</p>
              <div>{excluded.map((day) => <article key={day.date}><strong>{dateLabel(day.date, { day: "numeric", month: "short", weekday: "short" })}</strong><p>{day.reasons.map((reason) => t(reason)).join(" · ")}</p></article>)}</div>
            </details>
            {month === maxMonth && <p className="vivah-range-note">{t("Calculations are available through 30 December 2099. The final day of this month is outside the supported range.")}</p>}
          </>}
      </section>
      <section className="vivah-method" aria-labelledby="vivah-method-heading"><h2 id="vivah-method-heading">{t("How these dates are shortlisted")}</h2>
        <div className="vivah-method-grid"><div><h3>{t("Guruji's supplied rules")}</h3><p>{t("Uses the fourteen Nakshatras in Guruji's supplied list. Saturdays are excluded. Each window stays within Taurus, Gemini, Virgo, Libra, Sagittarius or Pisces Lagna.")}</p></div>
          <div><h3>{t("Additional Panchang safeguards")}</h3><p>{t("Existing safeguards remain: Magha/Mula first quarters, Rikta Tithis, Amavasya, adverse Yogas and prohibited Karanas; selected solar signs, Chaturmas, Adhika/Kshaya Masa, Holashtak, Rahu Kaal, Dur Muhurtam and Varjyam.")}</p></div>
          <div><h3>{t("Guru and Shukra")}</h3><p>{t("Uses angular combustion: Jupiter within 11°, Venus within 10° direct or 8° retrograde. Heliacal Asta and the days before and after it need separate review.")}</p></div></div>
        <p className="vivah-method-foot">{t("Guruji's complete method is still under review: Tuesday, Tithis, month conventions, Panchashalaka Vedha, planetary house conditions, exceptions and Trijyeshtha need confirmation. Other weekdays retain their current behavior.")}</p>
        <p className="vivah-method-foot">{t("Each day runs from sunrise to the next sunrise. Times after midnight carry the next date. Lagna signs are checked; full Lagna/Navamsha Shuddhi, personal Tara/Chandra bala, Lattadi and eclipse or Sankranti buffers require separate review.")}</p>
        <div className="vivah-method-links"><a href="https://www.drikpanchang.com/shubh-dates/info/choosing-auspicious-marriage-date.html" target="_blank" rel="noreferrer">{t("Marriage rule reference")}</a>
          <a href="/kundli-matching" onClick={(event) => followRoute(event, "/kundli-matching")}>{t("Kundli matching")} →</a></div>
      </section>
    </main><PortalFooter />
  </div>;
}
