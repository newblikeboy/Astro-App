import { useEffect, useState } from "react";
import type { ReactNode } from "react";
import { ArrowRight, ChevronLeft, ChevronRight, RotateCcw, Flame, Sunrise } from "lucide-react";
import AstroSymbol from "./AstroSymbol";
import EclipseSummary from "./EclipseSummary";
import ZodiacMark from "./ZodiacMark";
import { t, useLanguage } from "./i18n";
import type { Place } from "./types";
import "./sky.css";
import {
  MIN_CALCULATION_YEAR,
  MAX_CALCULATION_YEAR,
  indiaTimeZoneLabel,
} from "./calculation-dates";

type SkyPlanet = {
  name: string;
  rashi: string;
  degree_dms: string;
  nakshatra: string;
  pada: number;
  longitude_speed_deg_per_day: number;
  retrograde: boolean;
};
type Transition = {
  planet: string;
  kind: string;
  at: string;
  from_sign: string;
  to_sign: string;
  retrograde: boolean;
  sun_separation?: number;
  combust_limit?: number;
  boundary?: string;
};
type SkyData = {
  at?: string;
  year?: number;
  planets?: SkyPlanet[];
  events?: Transition[];
  settings: Record<string, string>;
};
const bodies = [
  "Sun",
  "Moon",
  "Mars",
  "Mercury",
  "Jupiter",
  "Venus",
  "Saturn",
  "Rahu",
  "Ketu",
];
const signs: Record<string, string> = {
  Mesh: "Aries",
  Vrishabh: "Taurus",
  Mithun: "Gemini",
  Kark: "Cancer",
  Simha: "Leo",
  Kanya: "Virgo",
  Tula: "Libra",
  Vrishchik: "Scorpio",
  Dhanu: "Sagittarius",
  Makar: "Capricorn",
  Kumbh: "Aquarius",
  Meen: "Pisces",
};
/** currentClock returns the current India-local clock to minute precision for planet-position inputs. */
function currentClock() {
  return new Intl.DateTimeFormat("en-GB", {
    timeZone: "Asia/Kolkata",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
  }).format(new Date());
}
/** currentDate returns today's India-local civil date for the sky tools. */
function currentDate() {
  const d = new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Kolkata",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(new Date());
  return ["year", "month", "day"]
    .map((k) => d.find((p) => p.type === k)!.value)
    .join("-");
}

type SkyToolsProps = {
  mode: "planets" | "eclipses";
  date: string;
  placeID: string;
  place: Place | null;
  onDateChange: (value: string) => void;
  locationControl: ReactNode;
  dateControls: ReactNode;
};

/** SkyTools renders native positions, transits, combustion boundaries and retrograde stations with relevant controls. */
export default function SkyTools({
  mode,
  date,
  placeID,
  place,
  onDateChange,
  locationControl,
  dateControls,
}: SkyToolsProps) {
  const { language } = useLanguage();
  const [planetView, setPlanetView] = useState("positions");
  const [clock, setClock] = useState(currentClock);
  const [year, setYear] = useState(Number(date.slice(0, 4)));
  const [planet, setPlanet] = useState("Sun");
  const [combustionPlanet, setCombustionPlanet] = useState("All planets");
  const [retrogradePlanet, setRetrogradePlanet] = useState("All planets");
  const [limit, setLimit] = useState(20);
  const [data, setData] = useState<SkyData | null>(null);
  const [busy, setBusy] = useState(true);
  const [error, setError] = useState(false);
  const [retry, setRetry] = useState(0);
  const isPositions = planetView === "positions";
  const isCombustion = planetView === "combustion";
  const isRetrograde = planetView === "retrograde";
  // Retrograde stations reuse the native transit response, rather than a second solver.
  const tool = isRetrograde ? "transits" : planetView;
  const selectedPlanet = isCombustion ? combustionPlanet : isRetrograde ? retrogradePlanet : planet;
  const selectableBodies = isCombustion ? bodies.slice(1, 7) : isRetrograde ? bodies.slice(2, 7) : bodies;
  const key = isPositions
    ? `${tool}?date=${date}&time=${clock}&place_id=${encodeURIComponent(placeID)}`
    : `${tool}?year=${year}&place_id=${encodeURIComponent(placeID)}`;
  const [loadedKey, setLoadedKey] = useState("");
  const ready = data && loadedKey === key;
  const locale = language === "hi" ? "hi-IN" : "en-IN";
  /** sign returns the localized zodiac-sign label for a sky result. */
  const sign = (value: string) => t(signs[value] || value);
  useEffect(() => setYear(Number(date.slice(0, 4))), [date.slice(0, 4)]);
  useEffect(() => {
    setLimit(20);
  }, [selectedPlanet, year, planetView]);
  useEffect(() => {
    if (mode === "eclipses") {
      setData(null);
      setLoadedKey("");
      return;
    }
    const abort = new AbortController();
    setBusy(true);
    setError(false);
    setData(null);
    fetch("/api/sky/" + key, { signal: abort.signal })
      .then(async (res) => {
        if (!res.ok) throw new Error("unavailable");
        return (await res.json()) as SkyData;
      })
      .then((v) => {
        setData(v);
        setLoadedKey(key);
      })
      .catch(() => {
        if (!abort.signal.aborted) setError(true);
      })
      .finally(() => {
        if (!abort.signal.aborted) setBusy(false);
      });
    return () => abort.abort();
  }, [mode, key, retry]);
  /** when formats a sky event's date and minute-precision clock in the India timezone. */
  const when = (value: string) =>
    new Intl.DateTimeFormat(locale, {
      timeZone: "Asia/Kolkata",
      day: "numeric",
      month: "short",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      hour12: true,
    }).format(new Date(value));
  const events =
    ready && data.events
      ? (data.events as Transition[]).filter(
          (e) => (selectedPlanet === "All planets" || e.planet === selectedPlanet) &&
            (!isRetrograde || e.kind === "Retrograde begins" || e.kind === "Direct motion begins"),
        )
      : [];
  if (mode === "eclipses")
    return (
      <EclipseSummary
        placeID={placeID}
        place={place}
        locationControl={locationControl}
      />
    );
  return (
    <div className="sky-tools">
      <div className="sky-switch" role="group" aria-label={t("Planet tools")}>
        {[
          ["positions", "Planet positions"], ["transits", "Planet transits"],
          ["combustion", "Planet combustion"], ["retrograde", "Planet retrograde"],
        ].map(([view, label]) => (
          <button key={view} type="button" aria-pressed={planetView === view} onClick={() => setPlanetView(view)}>
            {t(label)}
          </button>
        ))}
      </div>
      <div className="sky-controls">
        {isPositions ? (
          <>
            {dateControls}
            <label htmlFor="sky-clock">
              {t("Local time in India")}
              <input
                id="sky-clock"
                type="time"
                value={clock}
                onChange={(e) => {
                  if (/^\d{2}:\d{2}$/.test(e.target.value))
                    setClock(e.target.value);
                }}
              />
            </label>
            <button
              className="sky-current-time"
              type="button"
              onClick={() => {
                setClock(currentClock());
                onDateChange(currentDate());
              }}
            >
              {t("Use current time")}
            </button>
          </>
        ) : (
          <>
            <div className="sky-year">
              <button
                type="button"
                aria-label={t("Previous year")}
                disabled={year <= MIN_CALCULATION_YEAR}
                onClick={() => setYear((v) => v - 1)}
              >
                <ChevronLeft size={17} />
              </button>
              <label htmlFor="sky-year">
                <span className="sr-only">{t("Year")}</span>
                <select
                  id="sky-year"
                  value={year}
                  onChange={(e) => setYear(Number(e.target.value))}
                >
                  {Array.from(
                    { length: MAX_CALCULATION_YEAR - MIN_CALCULATION_YEAR + 1 },
                    (_, i) => MIN_CALCULATION_YEAR + i,
                  ).map((v) => (
                    <option key={v}>{v}</option>
                  ))}
                </select>
              </label>
              <button
                type="button"
                aria-label={t("Next year")}
                disabled={year >= MAX_CALCULATION_YEAR}
                onClick={() => setYear((v) => v + 1)}
              >
                <ChevronRight size={17} />
              </button>
            </div>
            <label htmlFor="sky-planet">
              {t("Planet")}
              <select
                id="sky-planet"
                value={selectedPlanet}
                onChange={(e) => (isCombustion ? setCombustionPlanet : isRetrograde ? setRetrogradePlanet : setPlanet)(e.target.value)}
              >
                {["All planets", ...selectableBodies].map((v) => (
                  <option key={v} value={v}>
                    {t(v)}
                  </option>
                ))}
              </select>
            </label>
          </>
        )}
      </div>
      <p className="daily-explainer">
        {isPositions
          ? t(
              "Positions for the selected date and time, using the Lahiri sidereal zodiac. These are sky positions, independent of a birth chart.",
            )
          : isCombustion ? t("Dates when planets enter or leave combustion under the declared Sun-separation limits. This traditional angular rule is not a forecast of naked-eye visibility.")
          : isRetrograde ? t("Retrograde and direct station dates in the selected year. A motion phase may begin in one year and end in the next.")
          : t(
              "Sign changes and motion changes throughout the selected year, including retrograde returns. Times use India's local clock, including historical timezone changes.",
            )}
      </p>
      {isRetrograde && <p className="daily-boundary">{t("Sun and Moon have no retrograde stations. Mean Rahu and Ketu remain retrograde and have no stations in this model.")}</p>}
      {isCombustion && <details className="sky-method">
        <summary>{t("Combustion calculation rules")}</summary>
        <p>{t("Shortest geocentric longitude separation from the Sun, with inclusive limits: Moon 12°, Mars 17°, Mercury 14° (12° retrograde), Jupiter 11°, Venus 10° (8° retrograde), Saturn 15°. Sun, Rahu and Ketu are excluded.")}</p>
        <p>{t("The limits match the Kundli report. A motion-dependent limit change is labelled separately from an angular crossing.")}</p>
      </details>}
      {busy && (
        <p role="status" className="daily-status">
          {t("Calculating sky events…")}
        </p>
      )}
      {error && (
        <div role="alert" className="daily-status">
          <p>{t("Sky calculations unavailable. Please try again.")}</p>
          <button
            type="button"
            className="text-button"
            onClick={() => setRetry((v) => v + 1)}
          >
            {t("Try again")}
          </button>
        </div>
      )}
      {ready && isPositions && (
        <>
          <p className="sky-timestamp">
            <time dateTime={data.at}>{when(data.at!)}</time> ·{" "}
            {t(indiaTimeZoneLabel(data.at!))}
          </p>
          <table className="sky-positions">
            <caption className="sr-only">{t("Planet positions")}</caption>
            <thead>
              <tr>
                {[
                  "Planet",
                  "Zodiac sign",
                  "Degree in sign",
                  "Nakshatra",
                  "Pada",
                  "Motion",
                ].map((v) => (
                  <th key={v} scope="col">
                    {t(v)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data.planets?.map((p) => (
                <tr key={p.name}>
                  <th scope="row">
                    <span className="planet-identity">
                      <AstroSymbol name={p.name} />
                      <span>{t(p.name)}</span>
                    </span>
                  </th>
                  <td data-label={t("Zodiac sign")}>
                    <span className="zodiac-identity">
                      <ZodiacMark
                        sign={Object.keys(signs).indexOf(p.rashi) + 1}
                      />
                      {sign(p.rashi)}
                    </span>
                  </td>
                  <td data-label={t("Degree in sign")}>{p.degree_dms}</td>
                  <td data-label={t("Nakshatra")}>{t(p.nakshatra)}</td>
                  <td data-label={t("Pada")}>{p.pada}</td>
                  <td data-label={t("Motion")}>
                    <span
                      className={
                        "motion-label " +
                        (p.retrograde ? "sky-retro" : "sky-direct")
                      }
                    >
                      {p.retrograde ? (
                        <RotateCcw size={12} aria-hidden="true" />
                      ) : (
                        <ArrowRight size={12} aria-hidden="true" />
                      )}
                      {t(p.retrograde ? "Retrograde" : "Direct")}
                    </span>
                    <small>
                      {p.longitude_speed_deg_per_day.toFixed(4)} {t("deg/day")}
                    </small>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <p className="daily-boundary">
            {t(
              "Rahu is the mean lunar node; Ketu is opposite. The lunar nodes are calculated points, not physical planets.",
            )}
          </p>
        </>
      )}
      {ready && !isPositions && (
        <>
          <p className="sky-count">
            {events.length} {t("events in this year")}
          </p>
          {events.length ? (
            <>
              <ol className="sky-transits">
                {events.slice(0, limit).map((e, i) => (
                  <li key={i}>
                    <AstroSymbol name={e.planet} />
                    <time dateTime={e.at}>{when(e.at)}</time>
                    <div className="sky-transit-copy">
                      <strong>{t(e.planet)}</strong>
                      <span>{t(e.kind)}</span>
                      {e.kind === "Sign ingress" ? (
                        <p>
                          <span className="sky-sign">{sign(e.from_sign)}</span>{" "}
                          <ArrowRight size={13} aria-label={t("to")} />{" "}
                          <span className="sky-sign">{sign(e.to_sign)}</span>
                        </p>
                      ) : (
                        <p>{sign(e.to_sign)}</p>
                      )}
                      {isCombustion && <p className="sky-combustion-evidence">
                        {t("Sun separation")}: {e.sun_separation?.toLocaleString(locale, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}°
                        <span>{t("Combustion limit")}: {e.combust_limit}°</span>
                        {e.boundary === "Motion-dependent limit" && <span>{t("Motion-dependent limit change")}</span>}
                      </p>}
                    </div>
                    <span
                      className={
                        "motion-label " +
                        (e.retrograde ? "sky-retro" : "sky-direct")
                      }
                    >
                      {isCombustion ? (e.kind === "Combustion begins" ? <Flame size={12} aria-hidden="true" /> : <Sunrise size={12} aria-hidden="true" />) : e.retrograde ? (
                        <RotateCcw size={12} aria-hidden="true" />
                      ) : (
                        <ArrowRight size={12} aria-hidden="true" />
                      )}
                      {isCombustion ? t(e.kind === "Combustion begins" ? "Combust" : "Not combust") : t(e.retrograde ? "Retrograde" : "Direct")}
                    </span>
                  </li>
                ))}
              </ol>
              {events.length > limit && (
                <button
                  type="button"
                  className="sky-more"
                  onClick={() => setLimit((v) => v + 20)}
                >
                  {t("Show more events")}
                </button>
              )}
            </>
          ) : (
            <p className="sky-empty">
              {t(isCombustion ? "No combustion changes for this planet in this year." : isRetrograde ? "No retrograde or direct stations for this planet in this year." : "No sign or motion changes for this planet in this year.")}
            </p>
          )}
        </>
      )}
      {ready && (
        <small className="sky-engine">
          Swiss Ephemeris {data.settings.swiss_version} · {data.settings.engine}
        </small>
      )}
    </div>
  );
}
