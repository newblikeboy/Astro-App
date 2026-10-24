import { useEffect, useId, useRef, useState } from "react";
import type { FormEvent, KeyboardEvent } from "react";
import {
  ArrowLeft,
  ArrowRight,
  Atom,
  Check,
  ChevronDown,
  Compass,
  Flame,
  FileDown,
  Grid2X2,
  Info,
  Leaf,
  LoaderCircle,
  MapPin,
  MessageCircle,
  Moon,
  Orbit,
  Printer,
  Search,
  ShieldCheck,
  SlidersHorizontal,
  Star,
  Sun,
  UserRound,
  UsersRound,
  X,
} from "lucide-react";
import { PortalFooter, PortalHeader } from "./PortalShell";
import ZodiacMark from "./ZodiacMark";
import { clockMinutes } from "./time-display";
import { signs } from "./Chart";
import {
  MatchingMangal,
  MatchingCharts,
  MatchingBirthDetails,
  SavedKundliPicker,
} from "./MatchingDetails";
import {
  factorValue,
  factorExplanation,
  scoreBandLabels,
  mangalStatusLabels,
} from "./matching-display";
import { t, useLanguage } from "./i18n";
import { followRoute } from "./routing";
import type { Birth, Place } from "./types";
import type {
  MatchingFactorCode,
  MatchingPerson,
  MatchingResponse,
  MatchingRole,
  MatchingProfile,
} from "./matching-types";
import "./matching.css";

type Entry = { birth: Birth; place: Place | null };
type MatchError = { message: string; person?: MatchingRole };
const roles: MatchingRole[] = ["bride", "groom"];
const resultTabs = [
  { id: "guna", label: "Guna Milan", Icon: Star },
  { id: "mangal", label: "Mangal analysis", Icon: Flame },
  { id: "charts", label: "Charts", Icon: Grid2X2 },
  { id: "birth", label: "Birth details", Icon: UserRound },
] as const;
type ResultTab = (typeof resultTabs)[number]["id"];
const factorIcons = {
  varna: Sun,
  vashya: Compass,
  tara: Star,
  yoni: Leaf,
  maitri: MessageCircle,
  gana: UsersRound,
  bhakoot: Orbit,
  nadi: Atom,
};
const factorCopy: Record<MatchingFactorCode, [string, string]> = {
  varna: ["Varna", "Moon-sign element categories"],
  vashya: ["Vashya", "Traditional sign-group relationship"],
  tara: ["Tara", "Birth stars counted in both directions"],
  yoni: ["Yoni", "Traditional birth-star animal groups"],
  maitri: ["Graha Maitri", "Relationship between Moon-sign lords"],
  gana: ["Gana", "Traditional birth-star temperament groups"],
  bhakoot: ["Bhakoot", "Relative positions of the Moon signs"],
  nadi: ["Nadi", "Traditional birth-star groups"],
};
const cautionCopy: Record<string, string> = {
  nadi_same:
    "The Nadi groups are the same. This base calculation applies no cancellation adjustments.",
  bhakoot_adverse:
    "The Moon-sign relationship receives no Bhakoot points under this rule set.",
  gana_adverse:
    "The Gana pairing receives few or no points under this rule set.",
  yoni_enemy: "The Yoni pairing is an opposing pair in the traditional table.",
  maitri_adverse:
    "The Moon-sign lord relationship receives few or no Graha Maitri points under this rule set.",
};
/** emptyEntry creates an empty matching participant without inventing a birth time or birthplace. */
const emptyEntry = (): Entry => ({
  birth: { name: "", date: "", time: "", place_id: "", unknown_time: false },
  place: null,
});
/** roleLabel returns the localized bride/groom label used by the matching form. */
const roleLabel = (role: MatchingRole) =>
  t(role === "bride" ? "Bride" : "Groom");
/** placeLabel formats a trusted birthplace label with Hindi text when available. */
const placeLabel = (place: Place, language: string) =>
  language === "hi" && place.hindi
    ? `${place.hindi} · ${place.name}`
    : place.name;

/** civilToday returns today's civil date in the selected birthplace timezone for birth-date validation. */
function civilToday(zone?: string) {
  const parts = new Intl.DateTimeFormat("en", {
    timeZone: zone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(new Date());
  /** part extracts one formatted calendar component while constructing a timezone-local date. */
  const part = (type: string) => parts.find((p) => p.type === type)!.value;
  return `${part("year")}-${part("month")}-${part("day")}`;
}

/** validResponse checks matching totals, factor maxima and required participant data before displaying a result. */
function validResponse(value: MatchingResponse) {
  const maxima: Record<MatchingFactorCode, number> = {
    varna: 1,
    vashya: 2,
    tara: 3,
    yoni: 4,
    maitri: 5,
    gana: 6,
    bhakoot: 7,
    nadi: 8,
  };
  if (
    !value ||
    !Number.isFinite(value.total) ||
    value.max !== 36 ||
    value.total < 0 ||
    value.total > 36 ||
    !Array.isArray(value.factors) ||
    value.factors.length !== 8
  )
    return false;
  const seen = new Set<string>();
  let sum = 0;
  for (const factor of value.factors) {
    if (
      !factor ||
      !maxima[factor.code] ||
      seen.has(factor.code) ||
      factor.max !== maxima[factor.code] ||
      !Number.isFinite(factor.score) ||
      factor.score < 0 ||
      factor.score > factor.max ||
      typeof factor.bride_value !== "string" ||
      typeof factor.groom_value !== "string"
    )
      return false;
    seen.add(factor.code);
    sum += factor.score;
  }
  return (
    Math.abs(sum - value.total) < 1e-8 &&
    roles.every((role) => {
      const person = value[role];
      return (
        person?.birth &&
        person.place &&
        person.mars_checks &&
        person.moon &&
        person.moon.rashi_number >= 1 &&
        person.moon.rashi_number <= 12 &&
        person.moon.pada >= 1 &&
        person.moon.pada <= 4 &&
        Number.isInteger(person.moon.nakshatra_number) &&
        person.moon.nakshatra_number! >= 1 &&
        person.moon.nakshatra_number! <= 27 &&
        person.chart &&
        Array.isArray(person.chart.planets) &&
        person.chart.planets.length === 9 &&
        Number.isFinite(Date.parse(person.chart.utc)) &&
        person.navamsa &&
        person.navamsa.lagna_sign >= 1 &&
        person.navamsa.lagna_sign <= 12 &&
        person.navamsa.planets.length === 9 &&
        person.dasha &&
        Number.isFinite(person.dasha.balance_years) &&
        value.mangal?.[role]?.references?.length === 3 &&
        value.mangal[role].references.every(
          (r) => Number.isInteger(r.house) && r.house >= 1 && r.house <= 12,
        ) &&
        Array.isArray(value.mangal[role].rules)
      );
    }) &&
    Array.isArray(value.cautions) &&
    typeof value.method === "string" &&
    typeof value.mangal?.method === "string" &&
    Number.isFinite(Date.parse(value.calculated_at)) &&
    typeof value.summary?.band === "string" &&
    value.summary.band in scoreBandLabels &&
    value.mangal.status in mangalStatusLabels
  );
}

/** BirthplaceSearch provides cancellable trusted-place search with keyboard selection and stale-response protection. */
function BirthplaceSearch({
  role,
  place,
  onSelect,
}: {
  role: MatchingRole;
  place: Place | null;
  onSelect: (value: Place | null) => void;
}) {
  const { language } = useLanguage();
  const unique = useId();
  const inputId = `matching-${role}-place-${unique}`;
  const listId = `${inputId}-list`;
  const input = useRef<HTMLInputElement>(null);
  const controller = useRef<AbortController | null>(null);
  const epoch = useRef(0);
  const [query, setQuery] = useState("");
  const [options, setOptions] = useState<Place[]>([]);
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [retry, setRetry] = useState(0);

  useEffect(() => {
    if (place) setQuery(placeLabel(place, language));
  }, [place, language]);

  useEffect(() => {
    const current = ++epoch.current;
    const abort = new AbortController();
    controller.current = abort;
    setOptions([]);
    setActive(-1);
    setError("");
    if (place || query.trim().length < 2) {
      setLoading(false);
      return () => abort.abort();
    }
    setLoading(true);
    const timer = window.setTimeout(() => {
      void fetch(`/api/places?q=${encodeURIComponent(query.trim())}`, {
        signal: abort.signal,
        credentials: "same-origin",
      })
        .then(async (response) => {
          if (!response.ok) throw new Error("search unavailable");
          const data: unknown = await response.json();
          if (!Array.isArray(data)) throw new Error("invalid search results");
          if (current === epoch.current && !abort.signal.aborted)
            setOptions(data as Place[]);
        })
        .catch(() => {
          if (current === epoch.current && !abort.signal.aborted)
            setError("Birthplace search is unavailable. Please try again.");
        })
        .finally(() => {
          if (current === epoch.current && !abort.signal.aborted)
            setLoading(false);
        });
    }, 220);
    return () => {
      window.clearTimeout(timer);
      abort.abort();
    };
  }, [query, place, retry]);

  useEffect(() => {
    if (open && active >= 0)
      document
        .getElementById(`${listId}-${active}`)
        ?.scrollIntoView({ block: "nearest" });
  }, [active, open, listId]);

  /** select accepts or clears a trusted place and cancels the previous autocomplete request. */
  function select(value: Place | null) {
    controller.current?.abort();
    epoch.current++;
    onSelect(value);
    setQuery(value ? placeLabel(value, language) : "");
    setOpen(!value);
    setActive(-1);
    input.current?.focus();
  }
  /** keyboard handles arrow, Enter and Escape actions in the matching birthplace suggestions. */
  function keyboard(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === "Escape") {
      setOpen(false);
      setActive(-1);
      return;
    }
    if (event.key === "Tab") {
      setOpen(false);
      return;
    }
    if (place) return;
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      setOpen(true);
      if (options.length)
        setActive((previous) =>
          event.key === "ArrowDown"
            ? (previous + 1) % options.length
            : (previous - 1 + options.length) % options.length,
        );
    } else if (event.key === "Enter" && open) {
      event.preventDefault();
      if (active >= 0 && options[active]) select(options[active]);
    }
  }
  const expanded = open && !place;
  return (
    <div
      className="matching-place"
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget as Node | null))
          setOpen(false);
      }}
    >
      <label htmlFor={inputId}>{t("Birthplace")}</label>
      <div className={`matching-place-input ${place ? "is-selected" : ""}`}>
        <MapPin size={17} aria-hidden="true" />
        <input
          ref={input}
          id={inputId}
          type="text"
          role="combobox"
          autoComplete="off"
          spellCheck={false}
          required
          value={query}
          placeholder={t("Search your city, town or village")}
          aria-autocomplete="list"
          aria-expanded={expanded}
          aria-controls={expanded ? listId : undefined}
          aria-activedescendant={
            expanded && active >= 0 ? `${listId}-${active}` : undefined
          }
          aria-describedby={`${inputId}-help`}
          onFocus={() => setOpen(true)}
          onKeyDown={keyboard}
          onChange={(event) => {
            controller.current?.abort();
            epoch.current++;
            onSelect(null);
            setQuery(event.target.value);
            setOpen(true);
            setActive(-1);
          }}
        />
        {place ? (
          <button
            type="button"
            onClick={() => select(null)}
            aria-label={t("Clear birthplace selection")}
          >
            <X size={16} />
          </button>
        ) : (
          <Search size={16} aria-hidden="true" />
        )}
      </div>
      <p id={`${inputId}-help`} className="matching-field-note">
        {place ? (
          <>
            <Check size={13} aria-hidden="true" />
            {t("Selected birthplace")} · {place.zone}
          </>
        ) : (
          t(
            "Choose a place from the suggestions. A nearby city can change the chart.",
          )
        )}
      </p>
      {expanded && (
        <div className="matching-place-popup">
          <ul
            id={listId}
            role="listbox"
            aria-label={t("Birthplace suggestions")}
            aria-busy={loading}
          >
            {options.map((option, index) => (
              <li
                id={`${listId}-${index}`}
                key={option.id}
                role="option"
                aria-selected={active === index}
                onMouseDown={(event) => event.preventDefault()}
                onMouseEnter={() => setActive(index)}
                onClick={() => select(option)}
              >
                <MapPin size={15} aria-hidden="true" />
                <span>{placeLabel(option, language)}</span>
              </li>
            ))}
          </ul>
          <p role="status">
            {loading
              ? t("Searching places…")
              : error
                ? t(error)
                : !options.length
                  ? t(
                      query.trim().length < 2
                        ? "Type at least two characters to find a birthplace."
                        : "No places found. Try a district or another spelling.",
                    )
                  : t("Use the arrow keys and Enter to select a place.")}
          </p>
          {error && (
            <button
              type="button"
              className="matching-link"
              onClick={() => setRetry((value) => value + 1)}
            >
              {t("Try search again")}
            </button>
          )}
        </div>
      )}
    </div>
  );
}

/** BirthCard collects one participant's known birth details and selected trusted place. */
function BirthCard({
  role,
  entry,
  update,
  selectPlace,
  offsetOpen,
  setOffsetOpen,
  onSaved,
}: {
  role: MatchingRole;
  entry: Entry;
  update: (key: keyof Birth, value: string | number | undefined) => void;
  selectPlace: (place: Place | null) => void;
  offsetOpen: boolean;
  setOffsetOpen: (open: boolean) => void;
  onSaved: () => void;
}) {
  const id = `matching-${role}`;
  return (
    <fieldset className="matching-birth-card">
      <legend>
        <span className="matching-person-icon">
          <UserRound size={21} aria-hidden="true" />
        </span>
        <span>
          {t(
            role === "bride"
              ? "Bride’s birth details"
              : "Groom’s birth details",
          )}
        </span>
      </legend>
      <button type="button" className="matching-saved-button" onClick={onSaved}>
        <UserRound size={14} aria-hidden="true" />
        {t("Use a saved Kundli")}
      </button>
      <div className="matching-field">
        <label htmlFor={`${id}-name`}>{t("Full name")}</label>
        <input
          id={`${id}-name`}
          name={`${role}-name`}
          autoComplete="off"
          required
          minLength={2}
          maxLength={100}
          value={entry.birth.name}
          placeholder={t("Enter full name")}
          onChange={(event) => update("name", event.target.value)}
        />
      </div>
      <div className="matching-date-time">
        <div className="matching-field">
          <label htmlFor={`${id}-date`}>{t("Birth date")}</label>
          <input
            id={`${id}-date`}
            name={`${role}-date`}
            type="date"
            required
            min="1940-01-01"
            max={civilToday(entry.place?.zone)}
            value={entry.birth.date}
            onChange={(event) => update("date", event.target.value)}
          />
        </div>
        <div className="matching-field">
          <label htmlFor={`${id}-time`}>{t("Birth time")}</label>
          <input
            id={`${id}-time`}
            name={`${role}-time`}
            type="time"
            step={60}
            required
            value={clockMinutes(entry.birth.time)}
            onChange={(event) => update("time", event.target.value)}
          />
        </div>
      </div>
      <BirthplaceSearch
        role={role}
        place={entry.place}
        onSelect={selectPlace}
      />
      <details
        className="matching-offset"
        open={offsetOpen}
        onToggle={(event) => setOffsetOpen(event.currentTarget.open)}
      >
        <summary>
          <SlidersHorizontal size={14} aria-hidden="true" />
          {t("Repeated local time options")}
          <ChevronDown size={14} aria-hidden="true" />
        </summary>
        <p>
          {t(
            "Only needed when daylight saving or a historical clock change repeated the same local time. Enter its UTC offset in minutes, such as 330 or 390.",
          )}
        </p>
        <label htmlFor={`${id}-offset`}>{t("UTC offset in minutes")}</label>
        <input
          id={`${id}-offset`}
          type="number"
          step={1}
          min={-1440}
          max={1440}
          inputMode="numeric"
          placeholder="330"
          value={entry.birth.utc_offset_minutes ?? ""}
          onChange={(event) =>
            update(
              "utc_offset_minutes",
              event.target.value === ""
                ? undefined
                : event.target.valueAsNumber,
            )
          }
        />
      </details>
    </fieldset>
  );
}

/** PersonSummary renders one matching participant's calculated Moon sign, star and identity summary. */
function PersonSummary({
  person,
  role,
}: {
  person: MatchingPerson;
  role: MatchingRole;
}) {
  const { language } = useLanguage();
  const date = new Intl.DateTimeFormat(language === "hi" ? "hi-IN" : "en-IN", {
    timeZone: "UTC",
    year: "numeric",
    month: "short",
    day: "numeric",
  }).format(new Date(`${person.birth.date}T12:00:00Z`));
  return (
    <article className="matching-person-summary">
      <div className="matching-person-title">
        <span>{roleLabel(role)}</span>
        <h3>{person.birth.name}</h3>
      </div>
      <p>
        {date} · {clockMinutes(person.birth.time)}
        <br />
        {placeLabel(person.place, language)}
      </p>
      <div className="matching-moon">
        <ZodiacMark sign={person.moon.rashi_number} />
        <div>
          <span>{t("Moon sign")}</span>
          <strong>{t(signs[person.moon.rashi_number - 1])}</strong>
        </div>
        <div>
          <span>{t("Birth star")}</span>
          <strong>{t(person.moon.nakshatra)}</strong>
          <small>{t("Pada {v0}", { v0: person.moon.pada })}</small>
        </div>
      </div>
    </article>
  );
}

/** MatchingApp coordinates two-person inputs, stateless matching results, saved-profile selection and PDF export. */
export default function MatchingApp() {
  const { language } = useLanguage();
  const [entries, setEntries] = useState<Record<MatchingRole, Entry>>({
    bride: emptyEntry(),
    groom: emptyEntry(),
  });
  const [offsets, setOffsets] = useState<Record<MatchingRole, boolean>>({
    bride: false,
    groom: false,
  });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<MatchError | null>(null);
  const [result, setResult] = useState<MatchingResponse | null>(null);
  const [tab, setTab] = useState<ResultTab>("guna");
  const [picker, setPicker] = useState<MatchingRole | null>(null);
  const [pdfBusy, setPDFBusy] = useState(false);
  const [pdfError, setPDFError] = useState("");
  const pdfRequest = useRef<AbortController | null>(null);
  const requestEpoch = useRef(0);
  const activeRequest = useRef<AbortController | null>(null);
  const resultHeading = useRef<HTMLHeadingElement>(null);
  const errorBox = useRef<HTMLDivElement>(null);
  const form = useRef<HTMLFormElement>(null);
  /** number formats a matching score in the active language without excessive decimal places. */
  const number = (value: number) =>
    new Intl.NumberFormat(language === "hi" ? "hi-IN" : "en-IN", {
      maximumFractionDigits: 2,
    }).format(value);
  const calculatedAt = result
    ? new Intl.DateTimeFormat(language === "hi" ? "hi-IN" : "en-IN", {
        year: "numeric",
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
        timeZoneName: "short",
      }).format(new Date(result.calculated_at))
    : "";
  useEffect(
    () => () => {
      requestEpoch.current++;
      activeRequest.current?.abort();
      pdfRequest.current?.abort();
    },
    [],
  );
  useEffect(() => {
    if (error) errorBox.current?.focus();
  }, [error]);
  useEffect(() => {
    if (result) {
      resultHeading.current?.focus({ preventScroll: true });
      resultHeading.current?.scrollIntoView({ block: "start" });
    }
  }, [result]);
  useEffect(() => {
    pdfRequest.current?.abort();
    pdfRequest.current = null;
    setPDFBusy(false);
    setPDFError("");
  }, [language]);

  /** invalidate cancels pending matching/PDF requests when their source birth inputs change. */
  function invalidate() {
    requestEpoch.current++;
    activeRequest.current?.abort();
    activeRequest.current = null;
    setBusy(false);
    setResult(null);
    setError(null);
    pdfRequest.current?.abort();
    pdfRequest.current = null;
    setPDFBusy(false);
    setPDFError("");
  }
  /** update changes one participant field and invalidates results based on the earlier inputs. */
  function update(
    role: MatchingRole,
    key: keyof Birth,
    value: string | number | undefined,
  ) {
    invalidate();
    setEntries((previous) => ({
      ...previous,
      [role]: {
        ...previous[role],
        birth: { ...previous[role].birth, [key]: value },
      },
    }));
  }
  /** selectPlace records a participant's trusted birthplace and clears obsolete matching results. */
  function selectPlace(role: MatchingRole, place: Place | null) {
    invalidate();
    setEntries((previous) => ({
      ...previous,
      [role]: {
        place,
        birth: {
          ...previous[role].birth,
          place_id: place?.id || "",
          utc_offset_minutes: undefined,
        },
      },
    }));
  }
  /** submit validates both participants and requests a new stateless Kundali comparison. */
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (activeRequest.current) return;
    for (const role of roles) {
      const { birth, place } = entries[role];
      if (!place || birth.place_id !== place.id) {
        setError({
          person: role,
          message:
            "Choose a birthplace from the suggestions before calculating.",
        });
        return;
      }
      if (!birth.time) {
        setError({
          person: role,
          message: "An accurate birth time is required for Kundli matching.",
        });
        return;
      }
      if (birth.date < "1940-01-01" || birth.date > civilToday(place.zone)) {
        setError({
          person: role,
          message:
            "Choose a birth date between 1940 and today at the selected birthplace.",
        });
        return;
      }
    }
    const controller = new AbortController();
    const current = ++requestEpoch.current;
    activeRequest.current = controller;
    setBusy(true);
    setError(null);
    setResult(null);
    try {
      const response = await fetch("/api/kundli/match", {
        method: "POST",
        credentials: "same-origin",
        signal: controller.signal,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          bride: entries.bride.birth,
          groom: entries.groom.birth,
        }),
      });
      const data = await response.json();
      if (current !== requestEpoch.current || controller.signal.aborted) return;
      if (!response.ok) {
        const person: MatchingRole | undefined = roles.includes(data.person)
          ? data.person
          : undefined;
        const message =
          typeof data.error === "string"
            ? data.error
            : "Matching calculations are unavailable right now. Please try again.";
        setError({ message, person });
        if (person && /UTC|offset|daylight|दो UTC|दोहराए/i.test(message))
          setOffsets((previous) => ({ ...previous, [person]: true }));
        return;
      }
      if (!validResponse(data)) throw new Error("invalid matching response");
      setTab("guna");
      setResult(data);
    } catch {
      if (current === requestEpoch.current && !controller.signal.aborted)
        setError({
          message:
            "Matching calculations are unavailable right now. Please try again.",
        });
    } finally {
      if (current === requestEpoch.current && !controller.signal.aborted) {
        activeRequest.current = null;
        setBusy(false);
      }
    }
  }

  /** useSavedProfile copies an explicitly selected owned Kundli into the chosen participant's form. */
  function useSavedProfile(profile: MatchingProfile) {
    if (!picker) return;
    invalidate();
    setEntries((previous) => ({
      ...previous,
      [picker]: { birth: { ...profile.birth }, place: profile.place },
    }));
    setPicker(null);
  }
  /** downloadPDF requests a server-recalculated matching PDF and downloads it only if the inputs remain current. */
  async function downloadPDF() {
    if (!result || pdfRequest.current) return;
    const controller = new AbortController(),
      epoch = requestEpoch.current;
    pdfRequest.current = controller;
    setPDFBusy(true);
    setPDFError("");
    try {
      const response = await fetch("/api/kundli/match/pdf", {
        method: "POST",
        credentials: "same-origin",
        signal: controller.signal,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          bride: entries.bride.birth,
          groom: entries.groom.birth,
          language,
        }),
      });
      if (!response.ok) {
        const data = await response.json();
        throw new Error(
          typeof data.error === "string"
            ? data.error
            : "The comparison PDF could not be prepared. Please try again.",
        );
      }
      if (!response.headers.get("Content-Type")?.startsWith("application/pdf"))
        throw new Error(
          "The comparison PDF could not be prepared. Please try again.",
        );
      const blob = await response.blob();
      if (controller.signal.aborted || epoch !== requestEpoch.current) return;
      const url = URL.createObjectURL(blob),
        link = document.createElement("a");
      link.href = url;
      link.download = `mishra4you-kundli-matching-${language}.pdf`;
      link.click();
      window.setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (error) {
      if (!controller.signal.aborted && epoch === requestEpoch.current)
        setPDFError(
          error instanceof Error
            ? error.message
            : "The comparison PDF could not be prepared. Please try again.",
        );
    } finally {
      if (pdfRequest.current === controller) {
        pdfRequest.current = null;
        setPDFBusy(false);
      }
    }
  }

  return (
    <div className="matching-page">
      <PortalHeader />
      <main className="matching-main">
        <div className="matching-hero">
          <div>
            <a
              href="/#birth"
              onClick={(event) => followRoute(event, "/#birth")}
              className="matching-back"
            >
              <ArrowLeft size={15} aria-hidden="true" />
              {t("Free Kundli")}
            </a>
            <h1>{t("Kundli matching")}</h1>
            <p>
              {t(
                "Bring two birth charts into one clear view. Explore a traditional Ashtakoota comparison, with every point explained.",
              )}
            </p>
          </div>
          <div className="matching-hero-note">
            <span className="matching-linked-icons" aria-hidden="true">
              <Moon size={23} />
              <Star size={22} />
            </span>
            <div>
              <strong>{t("Eight factors. 36 points.")}</strong>
              <span>
                {t("A considered starting point for a conversation.")}
              </span>
            </div>
          </div>
        </div>

        <section
          className="matching-form-section"
          aria-labelledby="matching-form-title"
        >
          <div className="matching-section-heading">
            <div>
              <h2 id="matching-form-title">
                {t("Start with accurate birth details")}
              </h2>
              <p>
                {t(
                  "Both birth times are required. Use the local time recorded at each birthplace.",
                )}
              </p>
            </div>
            <span className="matching-free">{t("Free calculation")}</span>
          </div>
          <form ref={form} onSubmit={submit}>
            <div className="matching-forms">
              {roles.map((role) => (
                <BirthCard
                  key={role}
                  role={role}
                  entry={entries[role]}
                  update={(key, value) => update(role, key, value)}
                  selectPlace={(place) => selectPlace(role, place)}
                  offsetOpen={offsets[role]}
                  setOffsetOpen={(open) =>
                    setOffsets((previous) => ({ ...previous, [role]: open }))
                  }
                  onSaved={() => setPicker(role)}
                />
              ))}
            </div>
            {error && (
              <div
                className="matching-error"
                ref={errorBox}
                role="alert"
                tabIndex={-1}
              >
                <Info size={19} aria-hidden="true" />
                <p>
                  {error.person && <strong>{roleLabel(error.person)}: </strong>}
                  {t(error.message)}
                </p>
              </div>
            )}
            <div className="matching-submit-row">
              <p>
                <ShieldCheck size={17} aria-hidden="true" />
                <span>
                  {t(
                    "No account needed. These details stay in this page for this visit and are not saved by matching.",
                  )}
                </span>
              </p>
              <button className="matching-submit" type="submit" disabled={busy}>
                {busy ? (
                  <LoaderCircle
                    size={18}
                    className="matching-spin"
                    aria-hidden="true"
                  />
                ) : (
                  <UsersRound size={19} aria-hidden="true" />
                )}
                {t(busy ? "Calculating both charts…" : "Compare birth charts")}
                {!busy && <ArrowRight size={18} aria-hidden="true" />}
              </button>
            </div>
            <p className="matching-live" role="status">
              {busy
                ? t(
                    "Calculating the two birth charts and all eight factors. Please wait.",
                  )
                : ""}
            </p>
          </form>
        </section>

        {result && (
          <section
            className="matching-results"
            aria-labelledby="matching-result-title"
          >
            <div className="matching-result-heading">
              <div>
                <h2
                  id="matching-result-title"
                  ref={resultHeading}
                  tabIndex={-1}
                >
                  {t("Your Ashtakoota comparison")}
                </h2>
                <p>{t("The full breakdown, without hidden adjustments.")}</p>
              </div>
              <div className="matching-export-actions">
                <button
                  type="button"
                  className="matching-download"
                  onClick={downloadPDF}
                  disabled={pdfBusy}
                >
                  {pdfBusy ? (
                    <LoaderCircle
                      size={17}
                      className="matching-spin"
                      aria-hidden="true"
                    />
                  ) : (
                    <FileDown size={17} aria-hidden="true" />
                  )}
                  {t(
                    pdfBusy
                      ? "Preparing comparison PDF…"
                      : "Download matching PDF",
                  )}
                </button>
                <button
                  type="button"
                  className="matching-print"
                  onClick={() => window.print()}
                >
                  <Printer size={16} aria-hidden="true" />
                  {t("Print comparison")}
                </button>
              </div>
            </div>
            {pdfError && (
              <p className="matching-pdf-error" role="alert">
                {t(pdfError)}
              </p>
            )}
            <p className="matching-export-note">
              {t(
                "The PDF is recalculated from these birth details. Downloading does not save a partner profile or report to your account.",
              )}
            </p>
            <div className="matching-overview">
              <div className="matching-score">
                <span>{t("Ashtakoota points")}</span>
                <div>
                  <strong>{number(result.total)}</strong>
                  <span>/ {number(result.max)}</span>
                </div>
                <p>{t("Traditional base score")}</p>
                <div className="matching-score-scale" aria-hidden="true">
                  <span
                    style={{ width: `${(result.total / result.max) * 100}%` }}
                  />
                </div>
                <small>
                  {t("Read the individual factors alongside this total.")}
                </small>
              </div>
              <div className="matching-people">
                {roles.map((role) => (
                  <PersonSummary key={role} role={role} person={result[role]} />
                ))}
              </div>
            </div>

            <div className="matching-result-summary">
              <div>
                <span>{t("Traditional score band")}</span>
                <strong>{t(scoreBandLabels[result.summary.band])}</strong>
              </div>
              <div>
                <span>{t("Mangal findings")}</span>
                <strong>{t(mangalStatusLabels[result.mangal.status])}</strong>
              </div>
              <p>
                {t(
                  "A score band is a traditional reference, not a marriage verdict. Read the individual factors and Mangal findings together.",
                )}
              </p>
            </div>
            <div
              className="matching-result-tabs"
              role="tablist"
              aria-label={t("Comparison sections")}
              onKeyDown={(event) => {
                const keys = ["ArrowRight", "ArrowLeft", "Home", "End"];
                if (!keys.includes(event.key)) return;
                event.preventDefault();
                const i = resultTabs.findIndex((t) => t.id === tab),
                  next =
                    event.key === "Home"
                      ? 0
                      : event.key === "End"
                        ? resultTabs.length - 1
                        : (i +
                            (event.key === "ArrowRight" ? 1 : -1) +
                            resultTabs.length) %
                          resultTabs.length;
                setTab(resultTabs[next].id);
                const buttons =
                  event.currentTarget.querySelectorAll<HTMLButtonElement>(
                    "[role=tab]",
                  );
                buttons[next]?.focus();
                buttons[next]?.scrollIntoView({
                  block: "nearest",
                  inline: "nearest",
                });
              }}
            >
              {resultTabs.map(({ id, label, Icon }) => (
                <button
                  key={id}
                  id={`matching-result-tab-${id}`}
                  role="tab"
                  type="button"
                  aria-selected={tab === id}
                  aria-controls={`matching-result-panel-${id}`}
                  tabIndex={tab === id ? 0 : -1}
                  onClick={() => setTab(id)}
                >
                  <Icon size={18} aria-hidden="true" />
                  {t(label)}
                </button>
              ))}
            </div>
            <section
              id="matching-result-panel-guna"
              className="matching-tab-panel"
              role="tabpanel"
              aria-labelledby="matching-result-tab-guna"
              hidden={tab !== "guna"}
            >
              <div className="matching-section-heading matching-breakdown-heading">
                <div>
                  <h3>{t("Where the points come from")}</h3>
                  <p>
                    {t(
                      "Each factor uses its own traditional scale. The eight maximum scores add up to 36.",
                    )}
                  </p>
                </div>
              </div>
              <div className="matching-factor-grid">
                {result.factors.map((factor) => {
                  const Icon = factorIcons[factor.code];
                  return (
                    <article className="matching-factor" key={factor.code}>
                      <div className="matching-factor-heading">
                        <span
                          className={`matching-factor-icon matching-icon-${factor.code}`}
                        >
                          <Icon size={21} aria-hidden="true" />
                        </span>
                        <div>
                          <h4>{t(factorCopy[factor.code][0])}</h4>
                          <p>{t(factorCopy[factor.code][1])}</p>
                        </div>
                        <span className="matching-factor-score">
                          <strong>{number(factor.score)}</strong>
                          <span>/ {number(factor.max)}</span>
                        </span>
                      </div>
                      <div className="matching-factor-values">
                        <div>
                          <span>{roleLabel("bride")}</span>
                          <strong>
                            {factorValue(factor.code, factor.bride_value)}
                          </strong>
                        </div>
                        <div>
                          <span>{roleLabel("groom")}</span>
                          <strong>
                            {factorValue(factor.code, factor.groom_value)}
                          </strong>
                        </div>
                      </div>
                      <div className="matching-factor-bar" aria-hidden="true">
                        <span
                          style={{
                            width: `${(factor.score / factor.max) * 100}%`,
                          }}
                        />
                      </div>
                      {factor.code === "tara" && (
                        <p className="matching-factor-note">
                          {t(
                            "Bride’s Tara is counted from the groom’s star; groom’s Tara is counted from the bride’s star.",
                          )}
                        </p>
                      )}
                      <details className="matching-factor-explanation">
                        <summary>
                          {t("Why these points")}
                          <ChevronDown size={14} aria-hidden="true" />
                        </summary>
                        <p>{factorExplanation(factor)}</p>
                      </details>
                    </article>
                  );
                })}
              </div>

              {result.cautions.length > 0 && (
                <aside className="matching-cautions">
                  <Info size={20} aria-hidden="true" />
                  <div>
                    <h3>{t("Points to review in context")}</h3>
                    <ul>
                      {result.cautions.map((code) => (
                        <li key={code}>
                          {t(
                            cautionCopy[code] ||
                              "A traditional factor needs contextual review. No cancellation adjustments are included.",
                          )}
                        </li>
                      ))}
                    </ul>
                  </div>
                </aside>
              )}
            </section>

            <section
              id="matching-result-panel-mangal"
              className="matching-tab-panel"
              role="tabpanel"
              aria-labelledby="matching-result-tab-mangal"
              hidden={tab !== "mangal"}
            >
              <MatchingMangal result={result} />
            </section>
            <section
              id="matching-result-panel-charts"
              className="matching-tab-panel"
              role="tabpanel"
              aria-labelledby="matching-result-tab-charts"
              hidden={tab !== "charts"}
            >
              <MatchingCharts result={result} />
            </section>
            <section
              id="matching-result-panel-birth"
              className="matching-tab-panel"
              role="tabpanel"
              aria-labelledby="matching-result-tab-birth"
              hidden={tab !== "birth"}
            >
              <MatchingBirthDetails result={result} />
            </section>

            <div className="matching-context">
              <Info size={21} aria-hidden="true" />
              <div>
                <h3>{t("Give the score its proper context")}</h3>
                <p>
                  {t(
                    "Ashtakoota is a traditional Moon-based point system. Use it alongside a personal conversation, shared values and a full professional reading. A point total cannot establish the outcome of a relationship.",
                  )}
                </p>
                <a
                  className="matching-review-link"
                  href="/#guidance"
                  onClick={(e) => followRoute(e, "/#guidance")}
                >
                  {t("Review with an astrologer")}
                  <ArrowRight size={14} aria-hidden="true" />
                </a>
              </div>
            </div>
            <details className="matching-method">
              <summary>
                <Compass size={18} aria-hidden="true" />
                {t("Calculation method and limits")}
                <ChevronDown size={17} aria-hidden="true" />
              </summary>
              <dl>
                <div>
                  <dt>{t("Zodiac")}</dt>
                  <dd>{t("Lahiri sidereal zodiac")}</dd>
                </div>
                <div>
                  <dt>{t("Rule set")}</dt>
                  <dd>{t("Eight-factor Ashtakoota base calculation")}</dd>
                </div>
                <div>
                  <dt>{t("Adjustments")}</dt>
                  <dd>
                    {t("Raw base points; no cancellation or exception bonuses")}
                  </dd>
                </div>
                <div>
                  <dt>{t("Calculation time")}</dt>
                  <dd>{calculatedAt}</dd>
                </div>
                <div>
                  <dt>{t("Method version")}</dt>
                  <dd>
                    <code>{result.method}</code>
                  </dd>
                </div>
              </dl>
              <p>
                {t(
                  "This method uses the AIFAS Horoscope Matching (2021) base tables. The Cow–Tiger Yoni pairing receives 0 in both directions, following the reciprocal enemy table and correcting an inconsistent printed cell.",
                )}
              </p>
              <div className="matching-sources">
                <a
                  href="https://www.futuresamachar.com/download/horoscope-matching-325.pdf"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  {t("AIFAS reference tables (PDF)")}
                </a>
                <a
                  href="https://saravali.github.io/astrology/koota_yoni.html"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  {t("Maitreya Yoni table reference")}
                </a>
              </div>
              <p>
                {t(
                  "Bride and groom identify the traditional calculation roles. Some factors depend on that order; changing the roles can change the points. Different regional tables and cancellation rules can produce different scores.",
                )}
              </p>
            </details>
            <aside className="matching-print-reference">
              <h3>{t("Calculation method and limits")}</h3>
              <p>
                <strong>{t("Method version")}: </strong>
                <code>{result.method}</code>
              </p>
              <p>
                {t(
                  "AIFAS Horoscope Matching (2021) base tables. Lahiri sidereal zodiac. Raw base points; no cancellation or exception bonuses.",
                )}
              </p>
              <p>
                {t(
                  "The Cow–Tiger Yoni pairing is scored 0 in both directions in this version.",
                )}
              </p>
              <p>
                <strong>{t("Calculation time")}: </strong>
                {calculatedAt}
              </p>
            </aside>
          </section>
        )}
        {!result && (
          <p className="matching-initial-note">
            <Info size={16} aria-hidden="true" />
            {t(
              "Ashtakoota points describe a traditional comparison. They do not determine a relationship’s outcome.",
            )}
          </p>
        )}
      </main>
      {picker && (
        <SavedKundliPicker
          role={picker}
          onSelect={useSavedProfile}
          onClose={() => setPicker(null)}
        />
      )}
      <PortalFooter />
    </div>
  );
}
