import { useRef, useState } from "react";
import {
  CalendarDays,
  Moon,
  Sunrise,
  Sunset,
  Clock3,
  Compass,
  Sparkles,
  Orbit,
  ShieldCheck,
} from "lucide-react";
import type { DailyPeriod, PanchangDetailsData } from "./panchang-types";
import { t, useLanguage } from "./i18n";
import { signs } from "./Chart";
import TimeRange from "./TimeRange";

const nakshatras = [
  "Ashwini",
  "Bharani",
  "Krittika",
  "Rohini",
  "Mrigashira",
  "Ardra",
  "Punarvasu",
  "Pushya",
  "Ashlesha",
  "Magha",
  "Purva Phalguni",
  "Uttara Phalguni",
  "Hasta",
  "Chitra",
  "Swati",
  "Vishakha",
  "Anuradha",
  "Jyeshtha",
  "Mula",
  "Purva Ashadha",
  "Uttara Ashadha",
  "Shravana",
  "Dhanishtha",
  "Shatabhisha",
  "Purva Bhadrapada",
  "Uttara Bhadrapada",
  "Revati",
];
const groups = [
  { id: "calendar", label: "Calendar & signs", Icon: CalendarDays },
  { id: "muhurats", label: "More timings", Icon: Clock3 },
  { id: "yogas", label: "Yoga & Balam", Icon: Sparkles },
  { id: "lagna", label: "Lagna & Panchaka", Icon: Orbit },
  { id: "directions", label: "Nivas & Shool", Icon: Compass },
];
/** LunarFacts renders lunar-month/year facts and the calculation conventions returned by the native engine. */
export function LunarFacts({
  details: d,
  stamp,
}: {
  details: PanchangDetailsData;
  stamp: (value: string) => string;
}) {
  /** duration formats a returned duration in the daily-details display. */
  const duration = (s: number) =>
    `${Math.floor(s / 3600)} ${t("hr")} ${Math.floor((s % 3600) / 60)} ${t("min")}`;
  return (
    <dl className="daily-lunar-facts">
      {[
        {
          label: "Moonrise",
          value: d.moonrise
            ? stamp(d.moonrise)
            : t("No Moonrise in this sunrise day"),
          Icon: Moon,
        },
        {
          label: "Moonset",
          value: d.moonset
            ? stamp(d.moonset)
            : t("No Moonset in this sunrise day"),
          Icon: Moon,
        },
        { label: "Day length", value: duration(d.day_seconds), Icon: Sunrise },
        {
          label: "Night length",
          value: duration(d.night_seconds),
          Icon: Sunset,
        },
      ].map(({ label, value, Icon }) => (
        <div key={label}>
          <dt>
            <Icon size={15} aria-hidden="true" />
            {t(label)}
          </dt>
          <dd>{value}</dd>
        </div>
      ))}
    </dl>
  );
}
/** PanchangDetails renders the extended native timings, ascendants, directions and traditional daily-rule results. */
export default function PanchangDetails({
  details: d,
  stamp,
}: {
  details: PanchangDetailsData;
  stamp: (value: string) => string;
}) {
  const { language } = useLanguage();
  /** month selects the localized lunar-month label. */
  const month = (name: string) => {
    const names = [
      "Chaitra",
      "Vaishakha",
      "Jyeshtha",
      "Ashadha",
      "Shravana",
      "Bhadrapada",
      "Ashwina",
      "Kartika",
      "Margashirsha",
      "Pausha",
      "Magha",
      "Phalguna",
    ];
    const hindi = [
      "चैत्र",
      "वैशाख",
      "ज्येष्ठ",
      "आषाढ़",
      "श्रावण",
      "भाद्रपद",
      "आश्विन",
      "कार्तिक",
      "मार्गशीर्ष",
      "पौष",
      "माघ",
      "फाल्गुन",
    ];
    return language === "hi" ? hindi[names.indexOf(name)] || name : name;
  };
  const [group, setGroup] = useState("calendar");
  const [birthSign, setBirthSign] = useState(0),
    [birthStar, setBirthStar] = useState(0);
  const nav = useRef<HTMLDivElement>(null);
  /** periods renders start/end intervals for one extended Panchang timing group. */
  function periods(rows: DailyPeriod[], labels = false) {
    return rows?.length ? (
      <ul className="extended-periods">
        {rows.map((p, i) => (
          <li key={i}>
            <div>
              <strong>
                {labels ? t(signs[(p.number || 1) - 1]) : t(p.name)}
              </strong>
              {p.quality && (
                <small className={`daily-quality ${p.quality.toLowerCase()}`}>
                  {t(p.quality)}
                </small>
              )}
            </div>
            <TimeRange start={p.start} end={p.end} format={stamp} />
          </li>
        ))}
      </ul>
    ) : (
      <p className="daily-explainer">
        {t("No periods of this type in the selected Panchang day.")}
      </p>
    );
  }
  /** fact renders one labelled daily-detail value without inventing missing data. */
  const fact = (label: string, value: string | number) => (
    <div key={label}>
      <dt>{t(label)}</dt>
      <dd>{typeof value === "string" ? t(value) : value}</dd>
    </div>
  );
  return (
    <section
      className="panchang-details"
      aria-labelledby="extended-panchang-heading"
    >
      <h3 id="extended-panchang-heading">{t("Explore your Panchang")}</h3>
      <div
        className="panchang-detail-tabs"
        role="tablist"
        aria-label={t("Panchang details")}
        ref={nav}
        onKeyDown={(e) => {
          const i = groups.findIndex((g) => g.id === group);
          const n =
            e.key === "ArrowRight"
              ? (i + 1) % groups.length
              : e.key === "ArrowLeft"
                ? (i + groups.length - 1) % groups.length
                : e.key === "Home"
                  ? 0
                  : e.key === "End"
                    ? groups.length - 1
                    : -1;
          if (n >= 0) {
            e.preventDefault();
            setGroup(groups[n].id);
            nav.current
              ?.querySelectorAll<HTMLButtonElement>("button")
              [n]?.focus();
          }
        }}
      >
        {groups.map(({ id, label, Icon }) => (
          <button
            type="button"
            key={id}
            id={`panchang-detail-${id}`}
            role="tab"
            aria-controls="panchang-detail-panel"
            aria-selected={group === id}
            tabIndex={group === id ? 0 : -1}
            onClick={() => setGroup(id)}
          >
            <Icon size={16} aria-hidden="true" />
            {t(label)}
          </button>
        ))}
      </div>
      <div
        id="panchang-detail-panel"
        role="tabpanel"
        aria-labelledby={`panchang-detail-${group}`}
        tabIndex={0}
      >
        {group === "calendar" && (
          <>
            <dl className="panchang-calendar-facts">
              {fact(
                "Amanta month",
                d.calendar.skipped_month
                  ? `${month(d.calendar.amanta)} / ${month(d.calendar.skipped_month)}`
                  : month(d.calendar.amanta),
              )}
              {fact(
                "Purnimanta month",
                d.calendar.skipped_month
                  ? `${month(d.calendar.amanta)} / ${month(d.calendar.skipped_month)}`
                  : month(d.calendar.purnimanta),
              )}
              {fact("Paksha", d.calendar.paksha)}
              {fact(
                "Month type",
                d.calendar.skipped_month
                  ? "Kshaya combination"
                  : d.calendar.adhika
                    ? "Adhika month"
                    : "Regular month",
              )}
              {d.calendar.skipped_month &&
                fact("Kshaya month", month(d.calendar.skipped_month))}
              {fact("Vikram Samvat", d.calendar.vikram)}
              {fact("Shaka Samvat", d.calendar.shaka)}
              {fact("Astronomical season", d.tropical_ritu)}
              {fact("Lunar season", d.lunar_ritu)}
              {fact("Tropical Ayana", d.tropical_ayana)}
              {fact("Sidereal Ayana", d.sidereal_ayana)}
              {fact("Local midday", stamp(d.midday))}
              {fact("Mean Lahiri ayanamsa", `${d.ayanamsa.toFixed(6)}°`)}
            </dl>
            <p className="daily-explainer">
              {t(
                "Months follow calculated new moons. Samvat years use the Chaitra new year; regional year-start traditions can differ.",
              )}
            </p>
            <div className="extended-split">
              <section>
                <h4>
                  <Moon size={16} aria-hidden="true" />
                  {t("Moon sign changes")}
                </h4>
                {periods(d.moon_signs, true)}
              </section>
              <section>
                <h4>
                  <Sunrise size={16} aria-hidden="true" />
                  {t("Sun sign changes")}
                </h4>
                {periods(d.sun_signs, true)}
              </section>
            </div>
            <section className="panchang-detail-card">
              <h4>{t("Nakshatra Pada changes")}</h4>
              <ul className="extended-periods">
                {d.padas.map((p, i) => (
                  <li key={i}>
                    <strong>
                      {t(p.name)} · {t("Pada")} {(((p.number || 1) - 1) % 4) + 1}
                    </strong>
                    <TimeRange start={p.start} end={p.end} format={stamp} />
                  </li>
                ))}
              </ul>
            </section>
          </>
        )}
        {group === "muhurats" && (
          <>
            <p className="daily-explainer">
              {t(
                "Traditional windows use the actual day, night or Nakshatra duration. Overlaps can still apply.",
              )}
            </p>
            {periods(d.muhurats)}
          </>
        )}
        {group === "yogas" && (
          <>
            <h4>{t("Special Yogas & periods")}</h4>
            {periods(d.special)}
            <h4>
              <ShieldCheck size={16} aria-hidden="true" />
              {t("Chandrabalam & Tarabalam")}
            </h4>
            <p className="daily-explainer">
              {t(
                "Choose your birth Moon sign or birth Nakshatra. These traditional checks are separate from a complete Muhurat assessment.",
              )}
            </p>
            <div className="balam-controls">
              <label>
                {t("Birth Moon sign")}
                <select
                  value={birthSign}
                  onChange={(e) => setBirthSign(Number(e.target.value))}
                >
                  <option value={0}>{t("Choose a sign")}</option>
                  {signs.map((s, i) => (
                    <option value={i + 1} key={s}>
                      {t(s)}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                {t("Birth nakshatra")}
                <select
                  value={birthStar}
                  onChange={(e) => setBirthStar(Number(e.target.value))}
                >
                  <option value={0}>{t("Choose a Nakshatra")}</option>
                  {nakshatras.map((s, i) => (
                    <option value={i + 1} key={s}>
                      {t(s)}
                    </option>
                  ))}
                </select>
              </label>
            </div>
            {!!(birthSign || birthStar) && (
              <ul className="extended-periods">
                {d.balam.map((p, i) => (
                  <li key={i}>
                    <div>
                      {birthSign > 0 && (
                        <span>
                          {t("Chandrabalam")}:{" "}
                          <strong>
                            {t(
                              p.chandra.includes(birthSign)
                                ? "Favourable"
                                : "Caution",
                            )}
                          </strong>
                        </span>
                      )}
                      {birthStar > 0 && (
                        <span>
                          {t("Tarabalam")}:{" "}
                          <strong>
                            {t(
                              p.tara.includes(birthStar)
                                ? "Favourable"
                                : "Caution",
                            )}
                          </strong>
                        </span>
                      )}
                    </div>
                    <TimeRange start={p.start} end={p.end} format={stamp} />
                  </li>
                ))}
              </ul>
            )}
          </>
        )}
        {group === "lagna" && (
          <>
            <div className="extended-split">
              <section>
                <h4>{t("Udaya Lagna")}</h4>
                {periods(d.lagna, true)}
              </section>
              <section>
                <h4>{t("Panchaka Rahita")}</h4>
                {periods(d.panchaka)}
              </section>
            </div>
            <p className="daily-explainer">
              {t(
                "Panchaka uses the Tithi, weekday, Nakshatra and rising sign. A favourable result does not override other cautions.",
              )}
            </p>
          </>
        )}
        {group === "directions" && (
          <>
            <p className="daily-explainer">
              {t(
                "Traditional direction and Nivas rules, split whenever their inputs change.",
              )}
            </p>
            <ul className="extended-periods">
              {d.directions.map((p, i) => (
                <li key={i}>
                  <strong>
                    {t(p.name)} · {t(p.value)}
                  </strong>
                  <TimeRange start={p.start} end={p.end} format={stamp} />
                </li>
              ))}
            </ul>
          </>
        )}
      </div>
      <p className="daily-boundary">
        {t(
          "These calculations do not yet include regional festival dates or fasting and Parana decisions.",
        )}
      </p>
    </section>
  );
}
