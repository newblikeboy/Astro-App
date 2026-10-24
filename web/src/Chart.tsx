import { t, getLanguage } from "./i18n";
import type { Report } from "./types";
const centers = [
  [150, 82],
  [77, 30],
  [28, 82],
  [75, 152],
  [28, 225],
  [77, 275],
  [150, 229],
  [223, 275],
  [272, 225],
  [225, 152],
  [272, 82],
  [223, 30],
];
export const shortNames: Record<string, string> = {
  Sun: "Su",
  Moon: "Mo",
  Mars: "Ma",
  Mercury: "Me",
  Jupiter: "Ju",
  Venus: "Ve",
  Saturn: "Sa",
  Rahu: "Ra",
  Ketu: "Ke",
};
const hindiShortNames: Record<string, string> = {
  Sun: "सू",
  Moon: "चं",
  Mars: "मं",
  Mercury: "बु",
  Jupiter: "गु",
  Venus: "शु",
  Saturn: "श",
  Rahu: "रा",
  Ketu: "के",
};
const houseShapes = [
  "150,0 225,75 150,150 75,75",
  "0,0 150,0 75,75",
  "0,0 75,75 0,150",
  "0,150 75,75 150,150 75,225",
  "0,150 75,225 0,300",
  "0,300 75,225 150,300",
  "150,150 225,225 150,300 75,225",
  "150,300 225,225 300,300",
  "225,225 300,150 300,300",
  "150,150 225,75 300,150 225,225",
  "225,75 300,0 300,150",
  "150,0 300,0 225,75",
];
export const hindiPlanets: Record<string, string> = {
  Sun: "सूर्य",
  Moon: "चंद्र",
  Mars: "मंगल",
  Mercury: "बुध",
  Jupiter: "गुरु",
  Venus: "शुक्र",
  Saturn: "शनि",
  Rahu: "राहु",
  Ketu: "केतु",
};
export const signs = [
  "मेष",
  "वृषभ",
  "मिथुन",
  "कर्क",
  "सिंह",
  "कन्या",
  "तुला",
  "वृश्चिक",
  "धनु",
  "मकर",
  "कुंभ",
  "मीन",
];
/** Chart draws a chart's signs and grahas in the supported visual layout using calculated placements. */
export default function Chart({
  report,
  sample = false,
  chart: suppliedChart,
  label,
  activeHouse,
}: {
  report?: Report;
  sample?: boolean;
  chart?: {
    lagna: { rashi_number: number };
    planets: { name: string; house: number }[];
    houses?: { house: number; rashi_number: number }[];
  };
  label?: string;
  activeHouse?: number;
}) {
  const chart = suppliedChart ?? report?.chart;
  return (
    <svg
      className="kundli-chart"
      viewBox="-8 -8 316 316"
      role="img"
      aria-label={
        label ??
        (sample
          ? t("उदाहरण उत्तर भारतीय कुंडली, आपकी व्यक्तिगत रिपोर्ट नहीं")
          : t(
              "उत्तर भारतीय जन्म कुंडली। ग्रहों और भावों की जानकारी नीचे सारणी में उपलब्ध है।",
            ))
      }
    >
      <rect
        x="0"
        y="0"
        width="300"
        height="300"
        rx="3"
        fill="var(--chart-fill, #fffdf7)"
        stroke="currentColor"
        strokeWidth="1.3"
      />
      {!sample &&
        houseShapes.map((points, i) => (
          <polygon
            key={i}
            points={points}
            className={
              activeHouse === i + 1
                ? "chart-house-selected"
                : i === 0
                  ? "chart-house-ascendant"
                  : "chart-house"
            }
          />
        ))}
      <path
        d="M0 0L300 300M300 0L0 300M150 0L300 150L150 300L0 150Z"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.1"
      />
      {centers.map(([x, y], i) => {
        const planets = chart?.planets.filter((p) => p.house === i + 1) || [];
        const names = planets.map((p) =>
          getLanguage() === "en" ? shortNames[p.name] : hindiShortNames[p.name],
        );
        const rows = Array.from(
          { length: Math.ceil(names.length / 2) },
          (_, n) => names.slice(n * 2, n * 2 + 2).join(" · "),
        );
        const firstY = Math.min(
          y + 11 - Math.max(0, rows.length - 1) * 5.5,
          295 - Math.max(0, rows.length - 1) * 11,
        );
        const sampleNames = [
          t("बु \u00B7 शु"),
          t("सू"),
          "",
          t("मं"),
          "",
          t("रा"),
          t("चं"),
          "",
          t("गु"),
          t("श"),
          t("के"),
          "",
        ];
        return (
          <g key={i} textAnchor="middle">
            <text className="chart-sign" x={x} y={sample ? y - 7 : firstY - 18}>
              {chart
                ? (chart.houses?.find((h) => h.house === i + 1)?.rashi_number ??
                  ((chart.lagna.rashi_number + i - 1) % 12) + 1)
                : ((6 + i - 1) % 12) + 1}
            </text>
            <text className="chart-planets" x={x} y={sample ? y + 11 : firstY}>
              {sample
                ? sampleNames[i]
                : rows.map((row, n) => (
                    <tspan key={n} x={x} dy={n ? 11 : 0}>
                      {row}
                    </tspan>
                  ))}
            </text>
          </g>
        );
      })}
    </svg>
  );
}
