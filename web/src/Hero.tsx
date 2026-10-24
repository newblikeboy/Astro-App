import {
  ArrowRight,
  ArrowUpRight,
  BookOpen,
  CalendarDays,
  CheckCircle2,
  Eclipse,
  Orbit,
  Sun,
} from "lucide-react";
import Chart from "./Chart";
import { t } from "./i18n";

/** Hero introduces the free chart, offers consultation as a secondary path, and groups daily-tool shortcuts. */
export default function Hero({
  onKundli,
  onTools,
  onConsultation,
}: {
  onKundli: () => void;
  onTools: (index: number) => void;
  onConsultation: () => void;
}) {
  return (
    <section className="hero" aria-labelledby="hero-heading">
      <div className="container hero-grid">
        <div className="hero-copy">
          <div className="hero-signature">
            <Sun size={17} aria-hidden="true" />
            {t("Vedic astrology. A fresh perspective.")}
          </div>
          <h1 id="hero-heading">
            {t("Your Kundli.")}
            <br />
            <span>{t("A new perspective.")}</span>
          </h1>
          <p className="hero-description">
            {t(
              "Create your birth chart, follow the daily Panchang, and explore the sky. One place to make sense of it all.",
            )}
          </p>
          <div className="hero-actions">
            <button className="button" onClick={onKundli}>
              {t("निःशुल्क कुंडली बनाएँ ")}
              <ArrowRight size={17} />
            </button>
            <button className="text-button" onClick={onConsultation}>
              {t("Explore consultations")}
              <ArrowUpRight size={16} />
            </button>
          </div>
          <div className="hero-points">
            <span>
              <CheckCircle2 size={14} />
              {t("Free birth chart")}
            </span>
            <span>
              <BookOpen size={14} />
              {t("English & Hindi")}
            </span>
            <span>
              <Sun size={14} />
              {t("Personal guidance")}
            </span>
          </div>
        </div>
        <div className="hero-art">
          <div className="sky-atlas">
            <svg
              className="atlas-orbits"
              viewBox="0 0 640 560"
              fill="none"
              aria-hidden="true"
            >
              <defs>
                <radialGradient id="atlasSun">
                  <stop stopColor="#fff5ee" />
                  <stop offset=".6" stopColor="#fcae79" />
                  <stop offset="1" stopColor="#8a4c23" />
                </radialGradient>
                <radialGradient id="atlasMoon" cx=".3" cy=".3">
                  <stop stopColor="#fff1e7" />
                  <stop offset="1" stopColor="#a56439" />
                </radialGradient>
              </defs>
              <g stroke="#c68459" strokeWidth=".75">
                <ellipse
                  cx="320"
                  cy="280"
                  rx="270"
                  ry="176"
                  transform="rotate(-28 320 280)"
                />
                <ellipse
                  cx="320"
                  cy="280"
                  rx="220"
                  ry="224"
                  transform="rotate(32 320 280)"
                />
                <ellipse
                  cx="320"
                  cy="280"
                  rx="145"
                  ry="247"
                  transform="rotate(44 320 280)"
                />
              </g>
              <circle
                cx="320"
                cy="280"
                r="205"
                stroke="#e79d6c"
                strokeDasharray="1 10"
                strokeWidth="1.5"
              />
              <path
                d="M65 280H575M320 45V515"
                stroke="#af7d5b"
                strokeDasharray="3 10"
                strokeWidth=".6"
              />
              <g fill="#ebb28d">
                {[
                  [72, 190],
                  [152, 100],
                  [222, 66],
                  [410, 67],
                  [526, 177],
                  [567, 318],
                  [480, 455],
                  [293, 499],
                  [109, 396],
                ].map(([x, y], i) => (
                  <g key={i}>
                    <circle cx={x} cy={y} r={i % 3 === 0 ? 3 : 1.8} />
                    {i % 3 === 0 && (
                      <circle
                        cx={x}
                        cy={y}
                        r="8"
                        stroke="#ebb28d"
                        strokeWidth=".6"
                      />
                    )}
                  </g>
                ))}
              </g>
              <circle cx="481" cy="112" r="31" fill="url(#atlasSun)" />
              <circle
                cx="481"
                cy="112"
                r="40"
                stroke="#fcb07e"
                strokeWidth=".7"
                opacity=".5"
              />
              <circle cx="123" cy="348" r="25" fill="url(#atlasMoon)" />
              <ellipse
                cx="123"
                cy="348"
                rx="47"
                ry="10"
                stroke="#f9be96"
                strokeWidth="3"
                transform="rotate(-28 123 348)"
              />
              <circle cx="442" cy="431" r="14" fill="url(#atlasMoon)" />
              <path
                d="M562 205v14m-7-7h14M177 436v14m-7-7h14M190 155v10m-5-5h10"
                stroke="#eeb48d"
                strokeWidth="1"
              />
            </svg>
            <div className="atlas-chart">
              <Chart sample />
            </div>
            <p className="atlas-note">
              {t("D1, Navamsha & Moon charts")}
              <br />
              {t("Made from your birth details.")}
            </p>
          </div>
          <div className="atlas-caption">
            {t("Tradition")}
            <span aria-hidden="true" />
            {t("Astronomy")}
            <span aria-hidden="true" />
            {t("Understanding")}
          </div>
        </div>
      </div>
      <div className="container">
        <div
          className="hero-shortcuts"
          aria-label={t("Explore astrology tools")}
        >
          <button onClick={() => onTools(0)}>
            <CalendarDays aria-hidden="true" />
            <span>
              <strong>{t("Daily Panchang")}</strong>
              <small>{t("Tithi, Nakshatra & local timings")}</small>
            </span>
            <ArrowUpRight aria-hidden="true" />
          </button>
          <button onClick={() => onTools(4)}>
            <Orbit aria-hidden="true" />
            <span>
              <strong>{t("Planet positions")}</strong>
              <small>{t("Positions, transits & retrogrades")}</small>
            </span>
            <ArrowUpRight aria-hidden="true" />
          </button>
          <button onClick={() => onTools(5)}>
            <Eclipse aria-hidden="true" />
            <span>
              <strong>{t("Eclipse dates")}</strong>
              <small>{t("Solar, lunar & local visibility")}</small>
            </span>
            <ArrowUpRight aria-hidden="true" />
          </button>
        </div>
      </div>
    </section>
  );
}
