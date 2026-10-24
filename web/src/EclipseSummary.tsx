import { useEffect, useState } from "react";
import type { ReactNode } from "react";
import { ArrowRight, Sun, Eye, EyeOff, Moon } from "lucide-react";
import EclipseArt from "./EclipseArt";
import { t, useLanguage } from "./i18n";
import type { Place } from "./types";
import {
  MIN_CALCULATION_YEAR,
  MAX_CALCULATION_YEAR,
} from "./calculation-dates";
import { eclipseNeighbours, eclipseSearchYears } from "./eclipse-summary";
import type { EclipseEvent } from "./eclipse-summary";

/** EclipseSummary shows the previous and next solar/lunar eclipses with selected-place visibility and type. */
export default function EclipseSummary({
  placeID,
  place,
  locationControl,
}: {
  placeID: string;
  place: Place | null;
  locationControl: ReactNode;
}) {
  const { language } = useLanguage();
  const [at, setAt] = useState(Date.now);
  const [events, setEvents] = useState<EclipseEvent[]>([]);
  const [loadedKey, setLoadedKey] = useState("");
  const [error, setError] = useState(false),
    [retry, setRetry] = useState(0);
  const years = eclipseSearchYears(
    at,
    MIN_CALCULATION_YEAR,
    MAX_CALCULATION_YEAR,
  );
  const key = years.join(",") + "/" + placeID;
  useEffect(() => {
    /** refresh updates the reference instant so previous/next cards stay current as time advances. */
    const refresh = () => setAt(Date.now());
    const timer = window.setInterval(refresh, 60_000);
    window.addEventListener("focus", refresh);
    document.addEventListener("visibilitychange", refresh);
    return () => {
      clearInterval(timer);
      window.removeEventListener("focus", refresh);
      document.removeEventListener("visibilitychange", refresh);
    };
  }, []);
  useEffect(() => {
    const abort = new AbortController();
    setError(false);
    setLoadedKey("");
    if (!years.length) {
      setError(true);
      return () => abort.abort();
    }
    Promise.all(
      years.map(async (year) => {
        const response = await fetch(
          `/api/sky/eclipses?year=${year}&place_id=${encodeURIComponent(placeID)}`,
          { signal: abort.signal },
        );
        if (!response.ok) throw new Error("unavailable");
        const data = (await response.json()) as { events: EclipseEvent[] };
        if (!Array.isArray(data.events)) throw new Error("unavailable");
        return data.events;
      }),
    )
      .then((batches) => {
        if (abort.signal.aborted) return;
        setEvents(batches.flat());
        setLoadedKey(key);
      })
      .catch(() => {
        if (!abort.signal.aborted) {
          setError(true);
          abort.abort();
        }
      });
    return () => abort.abort();
  }, [key, retry]);
  const ready = loadedKey === key && !error;
  const locale = language === "hi" ? "hi-IN" : "en-IN";
  /** formatDate formats an eclipse date in the selected language and India timezone. */
  const formatDate = (value: string) =>
    new Intl.DateTimeFormat(locale, {
      timeZone: "Asia/Kolkata",
      day: "numeric",
      month: "long",
      year: "numeric",
    }).format(new Date(value));
  /** formatClock formats the eclipse maximum to hours and minutes in the India timezone. */
  const formatClock = (value: string) =>
    new Intl.DateTimeFormat(locale, {
      timeZone: "Asia/Kolkata",
      hour: "2-digit",
      minute: "2-digit",
      hour12: true,
      timeZoneName: "short",
    }).format(new Date(value));
  const name = place
    ? language === "hi"
      ? place.hindi
      : place.name
    : t("selected location");
  return (
    <div className="sky-tools eclipse-summary">
      <div className="sky-controls">{locationControl}</div>
      <p className="daily-explainer">
        {t(
          "The most recent and next eclipse of each kind, worldwide. Visibility is shown separately for your city.",
        )}
      </p>
      {!ready && !error && (
        <p className="daily-status" role="status">
          {t("Finding the previous and next eclipses…")}
        </p>
      )}
      {error && (
        <div className="daily-status" role="alert">
          <p>{t("Sky calculations unavailable. Please try again.")}</p>
          <button
            className="text-button"
            type="button"
            onClick={() => setRetry((value) => value + 1)}
          >
            {t("Try again")}
          </button>
        </div>
      )}
      {ready && (
        <div className="eclipse-summary-grid">
          {eclipseNeighbours(events, at).map((group) => (
            <section
              className={`eclipse-summary-group eclipse-summary-${group.kind.toLowerCase()}`}
              key={group.kind}
              aria-label={t(
                group.kind === "Solar" ? "Solar eclipse" : "Lunar eclipse",
              )}
            >
              <header>
                <span className="sky-eclipse-symbol" aria-hidden="true">
                  {group.kind === "Solar" ? (
                    <Sun size={26} strokeWidth={1.4} />
                  ) : (
                    <Moon size={26} strokeWidth={1.4} />
                  )}
                </span>
                <div>
                  <h3>
                    {t(
                      group.kind === "Solar" ? "Solar eclipse" : "Lunar eclipse",
                    )}
                  </h3>
                  <p className="eclipse-kind-description">
                    {t(
                      group.kind === "Solar"
                        ? "The Moon passes in front of the Sun."
                        : "The Moon passes through Earth's shadow.",
                    )}
                  </p>
                </div>
              </header>
              <div className="eclipse-neighbours">
                {(["previous", "next"] as const).map((phase) => {
                  const event = group[phase];
                  return (
                    <article
                      key={phase}
                      className={`eclipse-neighbour eclipse-neighbour-${phase}`}
                      data-kind={group.kind}
                      data-phase={phase}
                    >
                      {event && (
                        <EclipseArt kind={group.kind} type={event.type} />
                      )}
                      <h4>
                        {t(
                          phase === "previous"
                            ? "Previous eclipse"
                            : "Next eclipse",
                        )}
                        {phase === "next" && (
                          <ArrowRight size={16} aria-hidden="true" />
                        )}
                      </h4>
                      {event ? (
                        <>
                          <time
                            className="eclipse-summary-date"
                            dateTime={event.maximum}
                          >
                            {formatDate(event.maximum)}
                          </time>
                          <div className="eclipse-summary-type">
                            <span>{t("Worldwide type")}</span>
                            <strong>{t(event.type)}</strong>
                          </div>
                          <p className="eclipse-peak">
                            <span>{t("Peak time")}</span>
                            <time dateTime={event.maximum}>
                              {formatClock(event.maximum)}
                            </time>
                          </p>
                          <p
                            className={`eclipse-summary-visibility ${event.visible ? "is-visible" : ""}`}
                          >
                            {event.visible ? (
                              <Eye size={14} aria-hidden="true" />
                            ) : (
                              <EyeOff size={14} aria-hidden="true" />
                            )}
                            <span>
                              {t(
                                event.visible
                                  ? "Visible from {v0}"
                                  : "Not visible from {v0}",
                                { v0: name },
                              )}
                              {event.visible &&
                                event.local &&
                                event.local.type !== event.type && (
                                  <small>
                                    {t("Local eclipse type")}:{" "}
                                    {t(event.local.type)}
                                  </small>
                                )}
                            </span>
                          </p>
                        </>
                      ) : (
                        <p className="muted">
                          {t(
                            "No event available within the supported date range.",
                          )}
                        </p>
                      )}
                    </article>
                  );
                })}
              </div>
            </section>
          ))}
        </div>
      )}
      {ready && (
        <p className="eclipse-art-note">
          {t("Eclipse illustrations are schematic.")}
        </p>
      )}
    </div>
  );
}
