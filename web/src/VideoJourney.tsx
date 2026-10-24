import { ArrowLeft, ArrowRight, FileText, MessageCircle } from "lucide-react";
import Chart from "./Chart";
import { hindiPlanets, signs } from "./Chart";
import { t, useLanguage } from "./i18n";
import type { Report } from "./types";
import { videoReadings, videoSign, videoTopicHouses } from "./video-links";
import type { VideoLink } from "./video-links";
import "./video-journey.css";

/** VideoIntroduction carries the merchant's single topic title into the existing birth-detail flow. */
export function VideoIntroduction({ link }: { link: VideoLink }) {
  return <section className="video-introduction section-space">
    <div className="container">
      <span className="eyebrow">{t("Guidance from Guruji's video")}</span>
      <h1>{link.title}</h1>
      <p>{t("Enter your birth details below to explore this topic in your own Kundli.")}</p>
    </div>
  </section>;
}

/** VideoReading shows the calculated chart and selected topic readings, with explicit unknown-time and zodiac-mismatch states and existing purchase actions. */
export function VideoReading({ link, report, onFull, onEdit, buy }: {
  link: VideoLink;
  report: Report;
  onFull: () => void;
  onEdit: () => void;
  buy: (product: "pdf" | "consultation") => void;
}) {
  const { language } = useLanguage();
  const readings = videoReadings(report, link);
  const sign = videoSign(report, link);
  const signLabel = link.sign_basis === "moon" ? "Moon sign" : link.sign_basis === "sun" ? "Sun sign" : "Ascendant";
  const mismatch = Boolean(report.chart && link.topic === "zodiac" && sign !== link.sign);
  return <div className="container video-reading workspace">
    <button className="text-button" onClick={onEdit}><ArrowLeft size={17} />{t("Edit birth details")}</button>
    <div className="video-reading-heading"><span className="eyebrow">{report.birth.name} · {t("Your personal chart")}</span>
      <h1>{link.title}</h1>
      <p>{t("Your Kundli, focused on the topic you came to explore.")}</p>
    </div>
    {report.chart ? <div className="video-reading-grid">
      <section className="video-chart-card" aria-labelledby="video-chart-heading">
        <h2 id="video-chart-heading">{t("Your birth chart")}</h2>
        <Chart report={report} />
        <dl className="fact-list"><div><dt>{t(signLabel)}</dt><dd>{sign ? t(signs[sign - 1]) : t("Unavailable")}</dd></div>
          <div><dt>{t("Birthplace")}</dt><dd>{language === "hi" ? report.place.hindi : report.place.name}</dd></div>
        </dl>
      </section>
      <section className="video-topic-facts" aria-labelledby="video-factors-heading">
        <h2 id="video-factors-heading">{t("Chart factors for this topic")}</h2>
        {mismatch && <p className="notice" role="status">{t("This video focuses on {v1}. Your {v2} is {v3}, so its sign-specific reading is not applied to you.", { v1: t(signs[link.sign - 1]), v2: t(signLabel), v3: sign ? t(signs[sign - 1]) : t("Unavailable") })}</p>}
        {(videoTopicHouses[link.topic] || []).length > 0 && <dl className="fact-list">{videoTopicHouses[link.topic].map((house) => <div key={house}>
          <dt>{t("House")} {house} · {t(signs[((report.chart!.lagna.rashi_number + house - 2) % 12)])}</dt>
          <dd>{report.chart!.planets.filter((planet) => planet.house === house).map((planet) => t(hindiPlanets[planet.name] || planet.name)).join(", ") || t("No natal planets")}</dd>
        </div>)}</dl>}
        {report.dasha && <p>{t("Current Mahadasha")}: {t(hindiPlanets[report.dasha.current_lord] || report.dasha.current_lord)}</p>}
        <p className="field-note">{t("Chart placements alone do not establish a personal outcome. Read the matching guidance below or discuss your chart with Guruji.")}</p>
        <h2>{t("Guidance for your chart")}</h2>
        {readings.map((rule) => <article className="video-approved-reading" key={rule.id + ":" + rule.version}>
          <span className="pill">{t("Approved reading")}</span><h3>{rule.topic}</h3>
          <p lang={language === "en" && rule.english ? "en" : "hi"}>{language === "en" && rule.english ? rule.english : rule.hindi}</p>
        </article>)}
        {!readings.length && !mismatch && <p className="notice" role="status">{t("Guruji has not yet approved a matching reading for this topic in your chart. You can explore your full Kundli or book a personal consultation.")}</p>}
      </section>
    </div> : <div className="notice" role="status"><h2>{t("An accurate birth time is needed")}</h2><p>{t("Without a birth time, a personal chart and topic reading cannot be calculated. Add your birth time or discuss it with Guruji.")}</p></div>}
    <section className="video-next-steps" aria-labelledby="video-next-heading">
      <h2 id="video-next-heading">{t("Choose your next step")}</h2>
      <button className="button secondary" onClick={onFull}>{t("View full Kundli")}<ArrowRight size={17} /></button>
      {report.chart && <button className="button" onClick={() => buy("pdf")}><FileText size={18} />{t("Buy personal remedies PDF")}</button>}
      <button className="button secondary" onClick={() => buy("consultation")}><MessageCircle size={18} />{t("Talk with Guruji")}</button>
    </section>
  </div>;
}
