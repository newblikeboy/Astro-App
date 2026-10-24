import { useEffect, useRef, useState } from "react";
import { ArrowLeft, ArrowRight, BookOpen, CalendarDays, Flame, Flower2, Search } from "lucide-react";
import { t, useLanguage } from "./i18n";
import { followRoute, navigate, useAppLocation } from "./routing";
import { PortalFooter, PortalHeader } from "./PortalShell";
import DevotionalMotif from "./DevotionalMotif";
import { devotionalSubjectArtwork } from "./devotional-art";
import { devotionalPage } from "./devotional-navigation";
import { pujaKinds } from "./puja-types";
import type { PujaGuide, PujaSummary } from "./puja-types";
import { pujaListPath, pujaPageSize } from "./puja-navigation";
import collection from "./data/puja-index.json";
import guideAsset from "./data/puja-guides.json?url";
import "./devotional.css";
import "./puja.css";

const guides = collection as PujaSummary[];

/** PujaArt shares the local subject illustration and a decorative worship motif without duplicating the guide's accessible name. */
function PujaArt({ subject, compact = false }: { subject: string; compact?: boolean }) {
  const artwork = devotionalSubjectArtwork(subject);
  return <div className={compact ? "puja-reader-art" : "devotional-card-art"} aria-hidden="true">
    {!compact && <DevotionalMotif category="aarti" />}
    {artwork && <span className="devotional-card-photo"><img src={artwork.src} alt="" width={artwork.width} height={artwork.height} loading={compact ? "eager" : "lazy"} decoding="async" /></span>}
  </div>;
}

/** PujaCredits retains source and licence attribution beside every set of reused subject images, deduplicating repeated card subjects. */
function PujaCredits({ subjects, sources = [] }: { subjects: string[]; sources?: string[] }) {
  const { language } = useLanguage();
  return <details className="devotional-image-credits">
    <summary>{t(sources.length ? "References & image credits" : "चित्र स्रोत और श्रेय")}</summary>
    <ul>{sources.map((source, index) => <li key={source}><a href={source} target="_blank" rel="noreferrer">{t("Detailed ritual reference")} · {index + 1} · Drik Panchang</a></li>)}
      {[...new Set(subjects)].map(subject => {
        const artwork = devotionalSubjectArtwork(subject);
        return artwork ? <li key={subject}><a href={artwork.source} target="_blank" rel="noreferrer">{language === "hi" ? artwork.labelHi : artwork.label}</a>{artwork.artist && <> · {artwork.artist}</>} · {artwork.licenseUrl ? <a href={artwork.licenseUrl} target="_blank" rel="noreferrer">{artwork.license}</a> : artwork.license}<br />{artwork.changes}</li> : null;
      })}
    </ul>
  </details>;
}

/** PujaLibrary owns public searchable guide URLs, twelve-card paging and a deferred bilingual reader, preserving list context on return. */
export default function PujaLibrary() {
  const { language } = useLanguage();
  const route = useAppLocation();
  const parts = route.pathname.split("/").filter(Boolean);
  const selected = parts.length === 2 ? guides.find(item => item.slug === parts[1]) : undefined;
  const invalid = parts.length > 2 || (parts.length === 2 && !selected);
  const kind = pujaKinds.find(item => item.id === route.searchParams.get("category"));
  const query = route.searchParams.get("q") || "";
  const normalized = query.normalize("NFC").toLocaleLowerCase().trim();
  const filtered = guides.filter(item => (!kind || item.kind === kind.id) &&
    (!normalized || `${item.title.en} ${item.title.hi} ${item.description.en} ${item.description.hi}`.normalize("NFC").toLocaleLowerCase().includes(normalized)));
  const totalPages = Math.max(1, Math.ceil(filtered.length / pujaPageSize));
  const page = devotionalPage(route.searchParams.get("page"), totalPages);
  const offset = (page - 1) * pujaPageSize;
  const visible = filtered.slice(offset, offset + pujaPageSize);
  const listPath = pujaListPath(kind?.id, query, page);
  const suffix = new URL(listPath, location.origin).search;
  const heading = useRef<HTMLHeadingElement>(null);
  const resultsHeading = useRef<HTMLHeadingElement>(null);
  const previousListing = useRef({ page, query, kind: kind?.id });
  const [reading, setReading] = useState<PujaGuide | null>(null);
  const [loadError, setLoadError] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const [fontSize, setFontSize] = useState(18);

  useEffect(() => {
    heading.current?.focus({ preventScroll: true });
    setFontSize(18);
  }, [route.pathname]);
  useEffect(() => {
    const previous = previousListing.current;
    // Pagination moves focus to the new results; typing a search never steals input focus.
    if (!selected && previous.page !== page && previous.query === query && previous.kind === kind?.id) {
      resultsHeading.current?.focus({ preventScroll: true });
      resultsHeading.current?.scrollIntoView({ block: "start" });
    }
    previousListing.current = { page, query, kind: kind?.id };
  }, [page, query, kind, selected]);
  useEffect(() => {
    let active = true;
    const controller = new AbortController();
    setLoadError(false);
    if (selected) fetch(guideAsset, { signal: controller.signal }).then(response => {
      if (!response.ok) throw new Error("Guide asset unavailable");
      return response.json();
    }).then((collection: PujaGuide[]) => {
      const guide = collection.find(item => item.slug === selected.slug);
      if (!guide?.steps.en.length || !guide.steps.hi.length) throw new Error("Guide missing from collection");
      if (active) setReading(guide);
    }).catch(() => { if (active) setLoadError(true); });
    return () => { active = false; controller.abort(); };
  }, [selected, attempt]);
  useEffect(() => {
    document.title = `${invalid ? t("Page not found") : selected ? selected.title[language] : t("Puja Vidhi")} | Mishra 4 You`;
  }, [selected, invalid, language]);

  return <div className="devotional-layout puja-layout">
    <a className="skip" href="#main">{t("मुख्य सामग्री पर जाएँ")}</a>
    <PortalHeader />
    <main id="main" className="devotional-page">
      <a className="devotional-back" href={selected ? listPath : "/"} onClick={event => followRoute(event, selected ? listPath : "/")}>
        <ArrowLeft size={16} aria-hidden="true" />{selected ? t("Back to guides") : t("Back to website")}
      </a>
      {invalid ? <section className="devotional-empty">
        <h1 ref={heading} tabIndex={-1}>{t("Page not found")}</h1>
        <p>{t("This guide is unavailable. Choose another from the collection.")}</p>
        <a href="/puja-vidhi" onClick={event => followRoute(event, "/puja-vidhi")}>{t("All guides")}</a>
      </section> : selected ? <>
        <nav className="devotional-breadcrumb" aria-label={t("पाठ का स्थान")}>
          <a href={listPath} onClick={event => followRoute(event, listPath)}>{t("Puja Vidhi")}</a>
          <span aria-hidden="true">/</span><span>{selected.title[language]}</span>
        </nav>
        <article className="puja-reader" aria-labelledby="puja-title">
          <header className="puja-reader-heading">
            <PujaArt subject={selected.art} compact />
            <div><span className="devotional-kicker"><Flower2 size={16} aria-hidden="true" />{t("Simple worship guide")}</span>
              <h1 id="puja-title" ref={heading} tabIndex={-1}>{selected.title[language]}</h1>
              <p>{selected.description[language]}</p>
            </div>
          </header>
          <div className="devotional-reader-tools" role="group" aria-label={t("पाठ का आकार")}>
            <span>{t("अक्षरों का आकार")}</span>
            <button type="button" aria-label={`A−, ${t("अक्षर छोटे करें")}`} disabled={fontSize <= 16} onClick={() => setFontSize(size => Math.max(16, size - 2))}>A−</button>
            <button type="button" aria-label={`A, ${t("सामान्य अक्षर आकार")}`} onClick={() => setFontSize(18)}>A</button>
            <button type="button" aria-label={`A+, ${t("अक्षर बड़े करें")}`} disabled={fontSize >= 28} onClick={() => setFontSize(size => Math.min(28, size + 2))}>A+</button>
          </div>
          {reading?.slug === selected.slug ? <div className="puja-reading" lang={language} style={{ fontSize: `${fontSize}px` }}>
            <div className="puja-reading-grid">
              <section className="puja-panel puja-materials" aria-labelledby="puja-materials-title">
                <h2 id="puja-materials-title">{t("Materials")}</h2>
                <ul>{reading.materials[language].map(item => <li key={item}>{item}</li>)}</ul>
              </section>
              <section className="puja-panel puja-steps" aria-labelledby="puja-steps-title">
                <h2 id="puja-steps-title">{t("Worship steps")}</h2>
                <ol>{reading.steps[language].map((step, index) => <li key={index}><p>{step}</p></li>)}</ol>
              </section>
            </div>
            <section className="puja-notes" aria-labelledby="puja-notes-title"><h2 id="puja-notes-title">{t("Tradition & details")}</h2><p>{reading.note[language]}</p>
              <p>{t("Rituals vary by family and region. Follow your tradition and your priest's guidance for elaborate ceremonies.")}</p>
            </section>
            <nav className="puja-related" aria-label={t("Related Aarti & readings")}>
              <a href={selected.related} onClick={event => followRoute(event, selected.related)}><BookOpen size={17} aria-hidden="true" />{t("Related Aarti & readings")}<ArrowRight size={16} aria-hidden="true" /></a>
              <a href="/#panchang" onClick={event => followRoute(event, "/#panchang")}><CalendarDays size={17} aria-hidden="true" />{t("View the date and local timings")}<ArrowRight size={16} aria-hidden="true" /></a>
            </nav>
            <PujaCredits subjects={[selected.art]} sources={reading.sources} />
          </div> : loadError ? <div className="devotional-empty" role="alert"><p>{t("The guide could not be opened. Please try again.")}</p>
            <button type="button" className="button" onClick={() => setAttempt(value => value + 1)}>{t("Try again")}</button></div>
            : <p className="puja-loading" role="status">{t("Opening the guide…")}</p>}
        </article>
      </> : <>
        <header className="devotional-intro">
          <div className="devotional-intro-art" aria-hidden="true"><DevotionalMotif category="aarti" /></div>
          <span className="devotional-kicker"><Flower2 size={18} aria-hidden="true" />{t("From preparation to Aarti")}</span>
          <h1 ref={heading} tabIndex={-1}>{t("Puja Vidhi")}</h1>
          <p>{t("Explore materials and simple worship steps, alongside your family's traditions.")}</p>
        </header>
        <nav className="puja-categories" aria-label={t("Puja guide categories")}>
          <a href={pujaListPath(undefined, query)} aria-current={!kind ? "page" : undefined} onClick={event => followRoute(event, pujaListPath(undefined, query))}><Flower2 size={20} aria-hidden="true" /><span>{t("All guides")}</span><small>{guides.length}</small></a>
          {pujaKinds.map(item => <a key={item.id} href={pujaListPath(item.id, query)} aria-current={kind?.id === item.id ? "page" : undefined} onClick={event => followRoute(event, pujaListPath(item.id, query))}>
            {item.id === "homa" ? <Flame size={20} aria-hidden="true" /> : <Flower2 size={20} aria-hidden="true" />}<span>{t(item.label)}</span><small>{guides.filter(guide => guide.kind === item.id).length}</small>
          </a>)}
        </nav>
        <section aria-labelledby="puja-results-title">
          <div className="devotional-collection-heading">
            <div><h2 id="puja-results-title" ref={resultsHeading} tabIndex={-1}>{t(kind?.label || "All guides")}</h2><p role="status">{t("{v1} guides available", { v1: filtered.length })}</p></div>
            <label className="devotional-search"><Search size={19} aria-hidden="true" /><span className="sr-only">{t("Search for a puja or festival")}</span>
              <input type="search" value={query} placeholder={t("Search by name…")} onChange={event => navigate(pujaListPath(kind?.id, event.target.value), true)} />
            </label>
          </div>
          <div className="devotional-grid">
            {visible.map(item => {
              const path = `/puja-vidhi/${item.slug}${suffix}`;
              return <a className="devotional-card puja-card" key={item.slug} href={path} onClick={event => followRoute(event, path)}>
                <PujaArt subject={item.art} />
                <div className="devotional-card-content"><span className="devotional-card-category">{t(pujaKinds.find(kind => kind.id === item.kind)!.label)}</span>
                  <h3>{item.title[language]}</h3><p>{item.description[language]}</p><span className="devotional-read">{t("Read the guide")}<ArrowRight size={16} aria-hidden="true" /></span>
                </div>
              </a>;
            })}
          </div>
          {totalPages > 1 && <nav className="devotional-pagination" aria-label={t("Puja guide pages")}>
            <p role="status">{t("Showing {v1}–{v2} of {v3} guides", { v1: offset + 1, v2: Math.min(offset + pujaPageSize, filtered.length), v3: filtered.length })}</p>
            <div className="devotional-page-links">
              {page > 1 ? <a href={pujaListPath(kind?.id, query, page - 1)} rel="prev" onClick={event => followRoute(event, pujaListPath(kind?.id, query, page - 1))}>{t("पिछला")}</a> : <span aria-disabled="true">{t("पिछला")}</span>}
              {Array.from({ length: totalPages }, (_, index) => index + 1).map(number => <a key={number} href={pujaListPath(kind?.id, query, number)} aria-current={page === number ? "page" : undefined} aria-label={t("पृष्ठ {v1}", { v1: number })} onClick={event => followRoute(event, pujaListPath(kind?.id, query, number))}>{number}</a>)}
              {page < totalPages ? <a href={pujaListPath(kind?.id, query, page + 1)} rel="next" onClick={event => followRoute(event, pujaListPath(kind?.id, query, page + 1))}>{t("अगला")}</a> : <span aria-disabled="true">{t("अगला")}</span>}
            </div>
          </nav>}
          {!filtered.length && <div className="devotional-empty"><BookOpen size={28} aria-hidden="true" /><h3>{t("No guides found")}</h3><p>{t("Try another name or browse all guides.")}</p><button type="button" className="button" onClick={() => navigate(pujaListPath(), true)}>{t("All guides")}</button></div>}
          {visible.length > 0 && <PujaCredits subjects={visible.map(item => item.art)} />}
        </section>
        <a className="puja-library-link" href="/bhakti" onClick={event => followRoute(event, "/bhakti")}><BookOpen size={18} aria-hidden="true" />{t("Open the prayer library")}<ArrowRight size={16} aria-hidden="true" /></a>
      </>}
    </main>
    <PortalFooter />
  </div>;
}
