import { useState } from "react";
import { Compass } from "lucide-react";
import Chart, { hindiPlanets, signs, shortNames } from "./Chart";
import AstroSymbol from "./AstroSymbol";
import { t, useLanguage } from "./i18n";

type ChartData = {
  lagna: { rashi_number: number };
  planets: { name: string; house: number }[];
  houses?: { house: number; rashi_number: number }[];
};

/** ChartViewer provides the chart-style controls and delegates the selected layout to the shared chart renderer. */
export default function ChartViewer({
  chart,
  label,
  referenceLabel = "Ascendant",
}: {
  chart: ChartData;
  label?: string;
  referenceLabel?: string;
}) {
  const { language } = useLanguage();
  const [selected, setSelected] = useState<string | null>(null);
  const planet = chart.planets.find((p) => p.name === selected);
  const house = planet?.house ?? 1;
  const sign =
    chart.houses?.find((h) => h.house === house)?.rashi_number ??
    ((chart.lagna.rashi_number + house - 2) % 12) + 1;
  const occupants = chart.planets.filter((p) => p.house === house);
  return (
    <div className="chart-viewer">
      <div className="chart-viewer-meta">
        <span>
          <Compass size={14} aria-hidden="true" />
          {t("North Indian")}
        </span>
        <span>
          {t(referenceLabel)} · {t(signs[chart.lagna.rashi_number - 1])}
        </span>
      </div>
      <Chart chart={chart} label={label} activeHouse={house} />
      <p className="chart-legend-label">
        {t("Select a planet to locate its house")}
      </p>
      <div
        className="chart-legend"
        role="group"
        aria-label={t("Chart planet legend")}
      >
        {chart.planets.map((p) => (
          <button
            key={p.name}
            type="button"
            aria-pressed={selected === p.name}
            onClick={() => setSelected(selected === p.name ? null : p.name)}
          >
            <AstroSymbol name={p.name} />
            <span>{t(hindiPlanets[p.name] || p.name)}</span>
            {language === "en" && <small>{shortNames[p.name]}</small>}
          </button>
        ))}
      </div>
      <div className="chart-house-detail" role="status">
        <strong>
          {t("House")} {house} <span>· {t(signs[sign - 1])}</span>
        </strong>
        <span>
          {occupants.length
            ? occupants
                .map((p) => t(hindiPlanets[p.name] || p.name))
                .join(" · ")
            : t("No planets in this house")}
        </span>
      </div>
    </div>
  );
}
