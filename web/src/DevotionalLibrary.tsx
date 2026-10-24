import { useEffect, useRef, useState } from "react";
import { ArrowLeft, ArrowRight, BookOpen, Flower2, Search } from "lucide-react";
import { t, useLanguage } from "./i18n";
import { followRoute, navigate, useAppLocation } from "./routing";
import { PortalFooter, PortalHeader } from "./PortalShell";
import { devotionalCategories } from "./devotional-types";
import type { DevotionalCategory, DevotionalText } from "./devotional-types";
import collection from "./data/devotional-index.json";
import "./devotional.css";
import DevotionalMotif from "./DevotionalMotif";
import { devotionalListPath, devotionalPage, devotionalPageSize } from "./devotional-navigation";
import { devotionalArtwork } from "./devotional-art";
import type { DevotionalArtwork } from "./devotional-art";

const texts = collection as Omit<DevotionalText, "verses">[];
const readingLoaders: Record<DevotionalCategory, () => Promise<{ default: Record<string, string[]> }>> = {
  /** aarti loads the bundled Aarti verses only when a visitor opens a reading. */
  aarti: () => import("./data/devotional/aarti.json"),
  /** chalisa loads the bundled Chalisa verses only when a visitor opens a reading. */
  chalisa: () => import("./data/devotional/chalisa.json"),
  /** stotram loads the bundled Stotram verses only when a visitor opens a reading. */
  stotram: () => import("./data/devotional/stotram.json"),
  /** ashtakam loads the bundled Ashtakam verses only when a visitor opens a reading. */
  ashtakam: () => import("./data/devotional/ashtakam.json"),
  /** mantra loads the bundled mantra groups only when a visitor opens a reading. */
  mantra: () => import("./data/devotional/mantra.json"),
  /** kavacham loads the bundled Kavacham texts only when a visitor opens a reading. */
  kavacham: () => import("./data/devotional/kavacham.json"),
};

/** DevotionalLibrary provides public, shareable category/search pages and a text-only reader of original prayers. */
export default function DevotionalLibrary() {
  const { language } = useLanguage();
  const route = useAppLocation();
  const parts = route.pathname.split("/").filter(Boolean);
  const category = devotionalCategories.find((item) => item.id === parts[1]);
  const selected = parts.length === 3 && category
    ? texts.find((item) => item.slug === parts[2] && item.categories.includes(category.id))
    : undefined;
  const invalid = parts.length > 3 || (parts.length > 1 && !category) || (parts.length === 3 && !selected);
  const query = route.searchParams.get("q") || "";
  const normalized = query.normalize("NFC").toLocaleLowerCase().trim();
  const listingCategory = selected && route.searchParams.get("collection") === "all" ? undefined : category;
  const filtered = texts.filter((item) => (!listingCategory || item.categories.includes(listingCategory.id)) &&
    (!normalized || `${item.title} ${item.titleHi} ${item.incipit}`.normalize("NFC").toLocaleLowerCase().includes(normalized)));
  const totalPages = Math.max(1, Math.ceil(filtered.length / devotionalPageSize));
  const currentPage = devotionalPage(route.searchParams.get("page"), totalPages);
  const offset = (currentPage - 1) * devotionalPageSize;
  const visibleTexts = filtered.slice(offset, offset + devotionalPageSize);
  const visibleArtwork = [...new Map(visibleTexts.map((item) => {
    const artwork = devotionalArtwork(item.slug);
    return [artwork?.src, artwork] as const;
  })).values()].filter((artwork): artwork is DevotionalArtwork => Boolean(artwork));
  const pageNumbers = [...new Set([1, currentPage - 1, currentPage, currentPage + 1, totalPages])]
    .filter((page) => page > 0 && page <= totalPages).sort((a, b) => a - b);
  const [fontSize, setFontSize] = useState(22);
  const [reading, setReading] = useState<{ slug: string; verses: string[] } | null>(null);
  const [loadError, setLoadError] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const heading = useRef<HTMLHeadingElement>(null);
  const resultsHeading = useRef<HTMLHeadingElement>(null);
  const previousPage = useRef(currentPage);
  const listPath = devotionalListPath(listingCategory?.id, query, currentPage);

  useEffect(() => {
    heading.current?.focus({ preventScroll: true });
    setFontSize(22);
  }, [route.pathname]);
  useEffect(() => {
    if (!selected && previousPage.current !== currentPage) {
      resultsHeading.current?.focus({ preventScroll: true });
      resultsHeading.current?.scrollIntoView({ block: "start" });
    }
    previousPage.current = currentPage;
  }, [currentPage, selected]);
  useEffect(() => {
    let active = true;
    setLoadError(false);
    if (selected) readingLoaders[selected.categories[0]]().then((result) => {
      const verses = result.default[selected.slug];
      if (!verses?.length) throw new Error("Reading missing from bundled collection");
      if (active) setReading({ slug: selected.slug, verses });
    }).catch(() => { if (active) setLoadError(true); });
    return () => { active = false; };
  }, [selected, attempt]);
  useEffect(() => {
    const label = invalid ? t("Page not found") : selected
      ? (language === "hi" ? selected.titleHi : selected.title)
      : `${t("भक्ति संग्रह")}${category ? " · " + t(category.label) : ""}`;
    document.title = `${label} | Mishra 4 You`;
  }, [route.pathname, selected, category, invalid, language]);

  return <div className="devotional-layout">
    <a className="skip" href="#main">{t("मुख्य सामग्री पर जाएँ")}</a>
    <PortalHeader />
    <main id="main" className="devotional-page">
      <a className="devotional-back" href={selected ? listPath : "/"}
        onClick={(event) => followRoute(event, selected ? listPath : "/")}>
        <ArrowLeft size={16} aria-hidden="true" />{selected ? t("संग्रह पर वापस जाएँ") : t("Back to website")}
      </a>
      {invalid ? <section className="devotional-empty">
        <h1 ref={heading} tabIndex={-1}>{t("Page not found")}</h1>
        <p>{t("यह पाठ उपलब्ध नहीं है। संग्रह में दूसरा पाठ चुनें।")}</p>
        <a href="/bhakti" onClick={(event) => followRoute(event, "/bhakti")}>{t("भक्ति संग्रह")}</a>
      </section> : selected ? <>
        <nav className="devotional-breadcrumb" aria-label={t("पाठ का स्थान")}>
          <a href="/bhakti" onClick={(event) => followRoute(event, "/bhakti")}>{t("भक्ति संग्रह")}</a>
          <span aria-hidden="true">/</span>
          <a href={devotionalListPath(category!.id)} onClick={(event) => followRoute(event, devotionalListPath(category!.id))}>{t(category!.label)}</a>
        </nav>
        <article className="devotional-reader" aria-labelledby="devotional-title">
          <header>
            <DevotionalMotif category={category!.id} className="devotional-reader-symbol" />
            <span className="devotional-kicker"><BookOpen size={16} aria-hidden="true" />{t(category!.label)}</span>
            <h1 id="devotional-title" ref={heading} tabIndex={-1}>{language === "hi" ? selected.titleHi : selected.title}</h1>
            {language === "en" && <p lang="hi" className="devotional-hindi-title">{selected.titleHi}</p>}
            {selected.textLanguage === "sa-Latn" && <p className="devotional-script-note">{t("इस समूह के मूल मंत्र रोमन लिपि में हैं।")}</p>}
            <div className="devotional-reader-tools" role="group" aria-label={t("पाठ का आकार")}>
              <span>{t("अक्षरों का आकार")}</span>
              <button type="button" aria-label={t("अक्षर छोटे करें")} disabled={fontSize <= 18} onClick={() => setFontSize((size) => Math.max(18, size - 2))}>A−</button>
              <button type="button" aria-label={t("सामान्य अक्षर आकार")} onClick={() => setFontSize(22)}>A</button>
              <button type="button" aria-label={t("अक्षर बड़े करें")} disabled={fontSize >= 34} onClick={() => setFontSize((size) => Math.min(34, size + 2))}>A+</button>
            </div>
          </header>
          <div className="devotional-verses" lang={selected.textLanguage || "hi"} style={{ fontSize: `${fontSize}px` }}>
            {reading?.slug === selected.slug ? reading.verses.map((verse, index) => <p key={index}>{verse}</p>)
              : loadError ? <div role="alert"><p>{t("पाठ नहीं खुल सका। फिर कोशिश करें।")}</p>
                <button type="button" className="button" onClick={() => setAttempt((value) => value + 1)}>{t("फिर कोशिश करें")}</button></div>
                : <p role="status">{t("पाठ खुल रहा है…")}</p>}
          </div>
          {reading?.slug === selected.slug && <footer><Flower2 size={18} aria-hidden="true" /><span>{t("पाठ समाप्त")}</span></footer>}
        </article>
      </> : <>
        <header className="devotional-intro">
          <div className="devotional-intro-art" aria-hidden="true"><DevotionalMotif category="mantra" /></div>
          <span className="devotional-kicker"><Flower2 size={18} aria-hidden="true" />{t("नित्य पाठ और प्रार्थना")}</span>
          <h1 ref={heading} tabIndex={-1}>{t("भक्ति संग्रह")}</h1>
          <p>{t("आरती, चालीसा, स्तोत्रम्, अष्टकम्, वैदिक मंत्र और कवचम् — अपने समय पर श्रद्धा से पढ़ें।")}</p>
        </header>
        <a className="puja-library-link" href="/puja-vidhi" onClick={(event) => followRoute(event, "/puja-vidhi")}>
          <Flower2 size={18} aria-hidden="true" />{t("Explore puja and havan guides")}<ArrowRight size={16} aria-hidden="true" />
        </a>
        <nav className="devotional-categories" aria-label={t("भक्ति संग्रह की श्रेणियाँ")}>
          <a href="/bhakti" aria-current={!category ? "page" : undefined} onClick={(event) => followRoute(event, "/bhakti")}>
            <DevotionalMotif category="all" /><span>{t("सभी पाठ")}</span><small>{texts.length}</small>
          </a>
          {devotionalCategories.map((item) => <a key={item.id} href={devotionalListPath(item.id, query)}
            data-devotion={item.id}
            aria-current={category?.id === item.id ? "page" : undefined}
            onClick={(event) => followRoute(event, devotionalListPath(item.id, query))}>
            <DevotionalMotif category={item.id} /><span>{t(item.label)}</span>
            <small>{texts.filter((text) => text.categories.includes(item.id)).length}</small>
          </a>)}
        </nav>
        <section className="devotional-collection" aria-labelledby="devotional-results-title">
          <div className="devotional-collection-heading">
            <div><h2 id="devotional-results-title" ref={resultsHeading} tabIndex={-1}>{category ? t(category.label) : t("सभी पाठ")}</h2>
              <p role="status">{t("{v1} पाठ उपलब्ध", { v1: filtered.length })}</p></div>
            <label className="devotional-search"><Search size={19} aria-hidden="true" />
              <span className="sr-only">{t("पाठ खोजें")}</span>
              <input type="search" value={query} placeholder={t("नाम या पहली पंक्ति से खोजें…")}
                onChange={(event) => navigate(devotionalListPath(category?.id, event.target.value), true)} />
            </label>
          </div>
          <div className="devotional-grid">
            {visibleTexts.map((item) => {
              const artwork = devotionalArtwork(item.slug);
              const readingParams = new URLSearchParams(new URL(listPath, location.origin).search);
              if (!category) readingParams.set("collection", "all");
              const suffix = readingParams.toString();
              const path = `/bhakti/${category?.id || item.categories[0]}/${item.slug}${suffix ? "?" + suffix : ""}`;
              return <a className="devotional-card" data-devotion={category?.id || item.categories[0]} key={item.slug} href={path} onClick={(event) => followRoute(event, path)}>
                <div className="devotional-card-art" aria-hidden="true">
                  <DevotionalMotif category={category?.id || item.categories[0]} />
                  {artwork && <span className="devotional-card-photo"><img src={artwork.src} alt=""
                    width={artwork.width} height={artwork.height} loading="lazy" decoding="async" /></span>}
                </div>
                <div className="devotional-card-content">
                <span className="devotional-card-category">{t(devotionalCategories.find((entry) => entry.id === (category?.id || item.categories[0]))!.label)}</span>
                <h3>{language === "hi" ? item.titleHi : item.title}</h3>
                <p lang={item.textLanguage || "hi"}>{item.incipit}</p>
                <span className="devotional-read">{t("पाठ पढ़ें")}<ArrowRight size={16} aria-hidden="true" /></span>
                </div>
              </a>;
            })}
          </div>
          {filtered.length > devotionalPageSize && <nav className="devotional-pagination" aria-label={t("संग्रह के पृष्ठ")}>
            <p role="status">{t("{v1}–{v2} दिखा रहे हैं · कुल {v3} पाठ", { v1: offset + 1, v2: Math.min(offset + devotionalPageSize, filtered.length), v3: filtered.length })}</p>
            <div className="devotional-page-links">
              {currentPage > 1 ? <a href={devotionalListPath(category?.id, query, currentPage - 1)} rel="prev"
                onClick={(event) => followRoute(event, devotionalListPath(category?.id, query, currentPage - 1))}>{t("पिछला")}</a>
                : <span aria-disabled="true">{t("पिछला")}</span>}
              {pageNumbers.map((page, index) => <span className="devotional-page-number" key={page}>
                {index > 0 && page - pageNumbers[index - 1] > 1 && <span className="devotional-page-gap" aria-hidden="true">…</span>}
                <a href={devotionalListPath(category?.id, query, page)} aria-label={t("पृष्ठ {v1}", { v1: page })}
                  aria-current={currentPage === page ? "page" : undefined}
                  onClick={(event) => followRoute(event, devotionalListPath(category?.id, query, page))}>{page}</a>
              </span>)}
              {currentPage < totalPages ? <a href={devotionalListPath(category?.id, query, currentPage + 1)} rel="next"
                onClick={(event) => followRoute(event, devotionalListPath(category?.id, query, currentPage + 1))}>{t("अगला")}</a>
                : <span aria-disabled="true">{t("अगला")}</span>}
            </div>
          </nav>}
          {!filtered.length && <div className="devotional-empty"><BookOpen size={28} aria-hidden="true" />
            <h3>{t("कोई पाठ नहीं मिला")}</h3><p>{t("दूसरा नाम या पहली पंक्ति लिखकर खोजें।")}</p>
            <button type="button" className="button" onClick={() => navigate(devotionalListPath(category?.id), true)}>{t("खोज हटाएँ")}</button>
          </div>}
          {visibleArtwork.length > 0 && <details className="devotional-image-credits">
            <summary>{t("चित्र स्रोत और श्रेय")}</summary>
            <p>{t("तस्वीरें छोटे आकार की PNG प्रतियों में दिखाई गई हैं। इनके मूल लाइसेंस लागू हैं।")}</p>
            <ul>{visibleArtwork.map((artwork) => <li key={artwork.src}>
              <a href={artwork.source} target="_blank" rel="noreferrer">{language === "hi" ? artwork.labelHi : artwork.label}</a>
              {artwork.artist && <> · {artwork.artist}</>}
              {" · "}{artwork.licenseUrl ? <a href={artwork.licenseUrl} target="_blank" rel="noreferrer">{artwork.license}</a> : artwork.license}
            </li>)}</ul>
          </details>}
        </section>
      </>}
    </main>
    <PortalFooter />
  </div>;
}
