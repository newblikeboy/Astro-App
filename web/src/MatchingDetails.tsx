import { useEffect, useRef, useState } from "react";
import {
  Check,
  ChevronDown,
  Flame,
  Info,
  Layers3,
  Moon,
  Orbit,
  Search,
  UserRound,
  X,
} from "lucide-react";
import ChartViewer from "./ChartViewer";
import { hindiPlanets, signs } from "./Chart";
import { t, useLanguage } from "./i18n";
import { clockMinutes, minuteTimestamp } from "./time-display";
import { followRoute, loginPath } from "./routing";
import {
  levelLabels,
  mangalRuleLabels,
  mangalStatusLabels,
  referenceLabels,
} from "./matching-display";
import type {
  MangalAssessment,
  MatchingPerson,
  MatchingProfile,
  MatchingResponse,
  MatchingRole,
} from "./matching-types";

const roles: MatchingRole[] = ["bride", "groom"];
/** nameOf selects the localized name for matching-report planets, signs or reference labels. */
const nameOf = (role: MatchingRole) => t(role === "bride" ? "Bride" : "Groom");

/** MatchingMangal renders both participants' raw Mangal evidence and separate cancellation assessments. */
export function MatchingMangal({ result }: { result: MatchingResponse }) {
  return (
    <section
      className="matching-mangal"
      aria-labelledby="matching-mangal-title"
    >
      <div className="matching-section-heading">
        <div>
          <h3 id="matching-mangal-title">
            <Flame size={20} aria-hidden="true" />
            {t("Mangal analysis")}
          </h3>
          <p>
            {t(
              "Named exceptions are shown separately. They never add points to the 36-point Ashtakoota total.",
            )}
          </p>
        </div>
      </div>
      <p className="matching-mangal-convention">
        {t(
          "The six-house convention checks 1, 2, 4, 7, 8 and 12 from Lagna, Moon and Venus. One flagged reference is low coverage; two or three are high.",
        )}
      </p>
      <div className="matching-mangal-people">
        {roles.map((role) => (
          <MangalPerson
            key={role}
            assessment={result.mangal[role]}
            role={role}
            person={result[role]}
          />
        ))}
      </div>
      <div className="matching-pair-summary">
        <h4>{t("Pair-level exceptions")}</h4>
        <strong>{t(mangalStatusLabels[result.mangal.status])}</strong>
        <ul>
          {result.mangal.rules
            .filter((rule) => rule.met)
            .map((rule) => (
              <li key={rule.reference + rule.code}>
                {t(referenceLabels[rule.reference])}:{" "}
                {t(mangalRuleLabels[rule.code])}
              </li>
            ))}
        </ul>
      </div>
      <p className="matching-mangal-note">
        {t(
          "Full planetary strength and context-dependent exceptions still require a professional reading. A possible benefic-strength condition is not treated as a confirmed cancellation.",
        )}
      </p>
    </section>
  );
}
/** MangalPerson renders one participant's Lagna/Moon/Venus Mangal reference details. */
function MangalPerson({
  assessment: a,
  role,
  person,
}: {
  assessment: MangalAssessment;
  role: MatchingRole;
  person: MatchingPerson;
}) {
  return (
    <article>
      <h4>
        {person.birth.name}
        <span>{nameOf(role)}</span>
      </h4>
      <dl className="matching-mangal-levels">
        <div>
          <dt>{t("Raw reference level")}</dt>
          <dd>
            {t(levelLabels[a.base_level])}
            <small>{t("{v0} of 3 references", { v0: a.base_count })}</small>
          </dd>
        </div>
        <div>
          <dt>{t("After selected exceptions")}</dt>
          <dd>
            {t(levelLabels[a.level])}
            <small>{t("{v0} of 3 references", { v0: a.active_count })}</small>
          </dd>
        </div>
      </dl>
      {a.references.map((ref) => (
        <div className="matching-mangal-row" key={ref.reference}>
          <span>{t(referenceLabels[ref.reference])}</span>
          <strong>{t("House {v0}", { v0: ref.house })}</strong>
          <span
            className={
              ref.flagged && !ref.cancelled
                ? "matching-position-flagged"
                : "matching-position-clear"
            }
          >
            {t(
              ref.cancelled
                ? "Exception applies"
                : ref.flagged
                  ? "Base placement"
                  : "Outside the checked houses",
            )}
          </span>
        </div>
      ))}
      {a.review_conditions.length > 0 && (
        <p className="matching-strength-note">
          <Info size={14} aria-hidden="true" />
          {t("Benefic strength requires review")}
        </p>
      )}
      <details className="matching-checked-rules">
        <summary>
          {t("Rules and exceptions checked")}
          <ChevronDown size={15} aria-hidden="true" />
        </summary>
        <ul>
          {a.rules.map((rule, i) => (
            <li key={rule.code + rule.reference}>
              <span>
                {t(mangalRuleLabels[rule.code])}
                {rule.reference !== "all" && (
                  <> · {t(referenceLabels[rule.reference])}</>
                )}
              </span>
              <small>
                {rule.met ? (
                  <Check size={13} aria-hidden="true" />
                ) : (
                  <X size={13} aria-hidden="true" />
                )}
                {t(rule.met ? "Condition met" : "Condition not met")}
              </small>
              <span className="sr-only">
                {t("Source PDF page {v0}", { v0: rule.source_pdf_page })}
              </span>
            </li>
          ))}
        </ul>
      </details>
    </article>
  );
}

/** chartData adapts a matching participant's calculated placements to the shared chart renderer. */
function chartData(person: MatchingPerson, mode: string) {
  if (mode === "d9")
    return {
      lagna: { rashi_number: person.navamsa.lagna_sign },
      planets: person.navamsa.planets,
    };
  if (mode === "moon")
    return {
      lagna: { rashi_number: person.moon.rashi_number },
      planets: person.chart.planets.map((p) => ({
        ...p,
        house: ((p.rashi_number - person.moon.rashi_number + 12) % 12) + 1,
      })),
    };
  return person.chart;
}
const chartModes = [
  { id: "d1", name: "D1 birth chart", Icon: Orbit },
  { id: "d9", name: "Navamsha D9", Icon: Layers3 },
  { id: "moon", name: "Moon chart", Icon: Moon },
];
/** MatchingCharts renders the pair's D1, D9 and Moon chart comparisons. */
export function MatchingCharts({ result }: { result: MatchingResponse }) {
  const [mode, setMode] = useState("d1");
  return (
    <section
      className="matching-charts"
      aria-labelledby="matching-charts-heading"
    >
      <div className="matching-section-heading">
        <div>
          <h3 id="matching-charts-heading">{t("Compare both charts")}</h3>
          <p>
            {t(
              "These are calculated chart placements. A D9 strength or complete marriage assessment is not inferred from the chart view.",
            )}
          </p>
        </div>
      </div>
      <div
        className="matching-chart-modes"
        role="group"
        aria-label={t("Chart comparison mode")}
      >
        {chartModes.map(({ id, name, Icon }) => (
          <button
            key={id}
            type="button"
            aria-pressed={mode === id}
            onClick={() => setMode(id)}
          >
            <Icon size={16} aria-hidden="true" />
            {t(name)}
          </button>
        ))}
      </div>
      <div className="matching-chart-pair">
        {roles.map((role) => (
          <article key={role}>
            <h4>
              <UserRound size={18} aria-hidden="true" />
              {nameOf(role)}
              <span>{result[role].birth.name}</span>
            </h4>
            <ChartViewer
              key={role + mode}
              chart={chartData(result[role], mode)}
              referenceLabel={mode === "moon" ? "Moon sign" : "Ascendant"}
              label={`${nameOf(role)} · ${t(chartModes.find((m) => m.id === mode)!.name)}`}
            />
          </article>
        ))}
      </div>
    </section>
  );
}

/** MatchingBirthDetails renders the recorded birth inputs and calculation conventions for both participants. */
export function MatchingBirthDetails({ result }: { result: MatchingResponse }) {
  const { language } = useLanguage();
  /** number formats matching birth-detail numeric values in the selected language. */
  const number = (n: number, max = 6) =>
    new Intl.NumberFormat(language === "hi" ? "hi-IN" : "en-IN", {
      maximumFractionDigits: max,
    }).format(n);
  const signLords = [
    "Mars",
    "Venus",
    "Mercury",
    "Moon",
    "Sun",
    "Mercury",
    "Venus",
    "Mars",
    "Jupiter",
    "Saturn",
    "Saturn",
    "Jupiter",
  ];
  const starLords = [
    "Ketu",
    "Venus",
    "Sun",
    "Moon",
    "Mars",
    "Rahu",
    "Jupiter",
    "Saturn",
    "Mercury",
  ];
  return (
    <section
      className="matching-birth-metadata"
      aria-labelledby="matching-birth-metadata-heading"
    >
      <div className="matching-section-heading">
        <div>
          <h3 id="matching-birth-metadata-heading">
            {t("Verified birth inputs")}
          </h3>
        </div>
      </div>
      <div className="matching-metadata-people">
        {roles.map((role) => {
          const p = result[role];
          const rows: [string, string][] = [
            ["Birth date", p.birth.date],
            ["Birth time", clockMinutes(p.birth.time)],
            [
              "Birthplace",
              language === "hi" && p.place.hindi ? p.place.hindi : p.place.name,
            ],
            ["IANA time zone", p.place.zone],
            [
              "Latitude / longitude",
              `${number(p.place.lat)} / ${number(p.place.lon)}`,
            ],
            [
              "Resolved local birth time",
              minuteTimestamp(p.chart.birth_time, p.place.zone, language),
            ],
            [
              "Resolved UTC instant",
              minuteTimestamp(p.chart.utc, "UTC", language),
            ],
            ["Lagna", t(signs[p.lagna.rashi_number - 1])],
            [
              "Lagna lord",
              t(hindiPlanets[signLords[p.lagna.rashi_number - 1]]),
            ],
            ["Moon sign", t(signs[p.moon.rashi_number - 1])],
            ["Birth star", t(p.moon.nakshatra)],
            ["Pada", String(p.moon.pada)],
            [
              "Nakshatra lord",
              t(hindiPlanets[starLords[(p.moon.nakshatra_number! - 1) % 9]]),
            ],
            [
              "Birth Dasha balance",
              `${t(hindiPlanets[p.dasha.birth_lord])} · ${t("{v0} years ({v1}-day year)", { v0: number(p.dasha.balance_years, 4), v1: p.dasha.year_days })}`,
            ],
          ];
          return (
            <article key={role}>
              <h4>
                {nameOf(role)}
                <span>{p.birth.name}</span>
              </h4>
              <dl>
                {rows.map(([label, value]) => (
                  <div key={label}>
                    <dt>{t(label)}</dt>
                    <dd>{value}</dd>
                  </div>
                ))}
              </dl>
            </article>
          );
        })}
      </div>
    </section>
  );
}

/** SavedKundliPicker loads only the signed-in customer's eligible saved Kundlis for explicit participant selection. */
export function SavedKundliPicker({
  role,
  onSelect,
  onClose,
}: {
  role: MatchingRole;
  onSelect: (p: MatchingProfile) => void;
  onClose: () => void;
}) {
  const { language } = useLanguage();
  const dialog = useRef<HTMLDialogElement>(null);
  const [profiles, setProfiles] = useState<MatchingProfile[]>([]),
    [loading, setLoading] = useState(true),
    [error, setError] = useState(""),
    [needsLogin, setNeedsLogin] = useState(false),
    [retry, setRetry] = useState(0),
    [query, setQuery] = useState("");
  useEffect(() => {
    const d = dialog.current!;
    d.showModal();
    return () => d.close();
  }, []);
  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError("");
    setNeedsLogin(false);
    void fetch("/api/kundli/match/profiles", {
      credentials: "same-origin",
      signal: controller.signal,
    })
      .then(async (response) => {
        if (response.status === 401) {
          setNeedsLogin(true);
          return;
        }
        if (!response.ok) throw new Error("unavailable");
        const data = await response.json();
        if (!Array.isArray(data.profiles)) throw new Error("invalid profiles");
        if (!controller.signal.aborted) setProfiles(data.profiles);
      })
      .catch(() => {
        if (!controller.signal.aborted)
          setError("Saved Kundlis are unavailable right now.");
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [retry]);
  return (
    <dialog
      ref={dialog}
      className="matching-profile-dialog"
      aria-labelledby="matching-profile-picker-title"
      onCancel={onClose}
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <header>
        <h2 id="matching-profile-picker-title">
          {t("Choose a saved Kundli")} · {nameOf(role)}
        </h2>
        <button type="button" aria-label={t("Close")} onClick={onClose}>
          <X size={20} aria-hidden="true" />
        </button>
      </header>
      {loading ? (
        <p role="status">{t("Loading your saved Kundlis…")}</p>
      ) : needsLogin ? (
        <div>
          <p>
            {t(
              "Sign in to choose your saved Kundlis. Leaving this page clears this unsaved comparison.",
            )}
          </p>
          <a
            href={loginPath("/kundli-matching")}
            onClick={(e) => followRoute(e, loginPath("/kundli-matching"))}
          >
            {t("Log in")}
          </a>
        </div>
      ) : error ? (
        <div role="alert">
          <p>{t(error)}</p>
          <button
            type="button"
            className="matching-link"
            onClick={() => setRetry((n) => n + 1)}
          >
            {t("Retry saved Kundlis")}
          </button>
        </div>
      ) : profiles.length === 0 ? (
        <p>
          {t(
            "No eligible saved Kundlis were found. A known birth time and a birth date from 1940 are required.",
          )}
        </p>
      ) : (
        <>
          <label className="matching-profile-search">
            <Search size={17} aria-hidden="true" />
            <span className="sr-only">{t("Full name")}</span>
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              autoComplete="off"
              placeholder={t("Full name")}
            />
          </label>
          <ul>
            {profiles
              .filter((p) =>
                (p.birth.name + " " + p.place.name)
                  .toLocaleLowerCase()
                  .includes(query.toLocaleLowerCase()),
              )
              .map((p) => (
                <li key={p.id}>
                  <button type="button" onClick={() => onSelect(p)}>
                    <strong>{p.birth.name}</strong>
                    <span>
                      {p.birth.date} · {clockMinutes(p.birth.time)}
                    </span>
                    <small>
                      {language === "hi" && p.place.hindi
                        ? p.place.hindi
                        : p.place.name}
                    </small>
                    <span>{t("Use these birth details")}</span>
                  </button>
                </li>
              ))}
          </ul>
        </>
      )}
    </dialog>
  );
}
