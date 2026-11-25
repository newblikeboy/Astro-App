import { useRef, useState } from "react";
import { Compass, Layers3, Maximize2, Moon, X } from "lucide-react";
import Chart, { chartDegree, chartStatusSymbols, chartWithNatalDetails, hindiPlanets, signs } from "./Chart";
import LagnaGrahaRoles from "./LagnaGrahaRoles";
import { t } from "./i18n";
import type { Report } from "./types";
import "./kundli-overview.css";

/** KundliOverviewCharts keeps D1, Moon and D9 visible together, with natal annotations and one shared handwritten role list. */
export default function KundliOverviewCharts({ report }: { report: Report }) {
  const [expanded, setExpanded] = useState("d1");
  const dialog = useRef<HTMLDialogElement>(null);
  const natal = report.chart;
  if (!natal) return null;
  const moon = natal.planets.find(planet=>planet.name==="Moon");
  const navamsa = report.navamsa ?? report.details?.vargas.find(varga=>varga.factor===9);
  const charts = [
    { id:"d1", title:t("लग्न कुंडली"), badge:"D1", Icon:Compass, chart:chartWithNatalDetails(natal,report) },
    { id:"moon", title:t("चंद्र कुंडली"), badge:t("चंद्र"), Icon:Moon, chart:moon ? chartWithNatalDetails({lagna:{rashi_number:moon.rashi_number},planets:natal.planets.map(planet=>({...planet,house:(planet.rashi_number-moon.rashi_number+12)%12+1}))},report) : undefined },
    { id:"d9", title:t("नवांश कुंडली"), badge:"D9", Icon:Layers3, chart:navamsa ? chartWithNatalDetails({lagna:{rashi_number:navamsa.lagna_sign},planets:navamsa.planets},report,false) : undefined },
  ];
  const expandedChart = charts.find(chart=>chart.id===expanded)!;
  return (
    <section className="overview-charts" aria-label={t("लग्न, चंद्र और नवांश कुंडलियाँ")}>
      <div className="overview-chart-heading">
        <div><span className="eyebrow">{t("कुंडली सारांश")}</span><h2>{t("लग्न, चंद्र और नवांश")}</h2></div>
        <span className="overview-chart-format">{t("North Indian")} · {t("राशि संख्या 1–12")}</span>
      </div>
      <div className="overview-chart-grid">
        {charts.map(({id,title,badge,Icon,chart})=><article className="overview-chart-card" key={id} data-chart={id}>
          <header><div><span className="overview-chart-icon"><Icon size={18} aria-hidden="true" /></span><h3>{title}</h3><span className="overview-chart-badge">{badge}</span></div>
            {chart && <button type="button" className="chart-expand" aria-label={`${title} · ${t("बड़ा करके देखें")}`} onClick={()=>{setExpanded(id);dialog.current?.showModal();}}><Maximize2 size={16} aria-hidden="true" /></button>}
          </header>
          {chart ? <>
            <Chart chart={chart} detailed label={`${title} · ${t("जन्म अंश, नक्षत्र और पद")}`} />
            <footer><span>{t(id==="moon" ? "चंद्र राशि" : "Ascendant")}</span><strong>{t(signs[chart.lagna.rashi_number-1])}</strong></footer>
          </> : <p className="overview-chart-missing">{t("इस सुरक्षित रिपोर्ट में यह चार्ट उपलब्ध नहीं है। जन्म विवरण से कुंडली दोबारा बनाएँ।")}</p>}
        </article>)}
      </div>
      <div className="overview-chart-legend" aria-label={t("चार्ट के चिह्न")}>
        <span><b>↺</b> {t("वक्री")}</span><span><b>†</b> {t("अस्त")}</span><span><b>↑</b> {t("उच्च")}</span><span><b>↓</b> {t("नीच")}</span><span><b>स्व</b> {t("स्वराशि")}</span><span><b>◈</b> {t("वर्गोत्तम")}</span>
      </div>
      <p className="overview-chart-note">{t("अंश, नक्षत्र और पद जन्म स्थिति के हैं। ↑ ↓ स्व जन्म राशि की स्थिति हैं; ◈ D9 में वर्गोत्तम है।")}</p>
      <LagnaGrahaRoles natalLagna={natal.lagna.rashi_number} />
      <dialog ref={dialog} className="chart-zoom-dialog" aria-labelledby="chart-zoom-title" onClick={event=>{if(event.target===event.currentTarget)dialog.current?.close();}}>
        <div className="chart-zoom-content"><header><h2 id="chart-zoom-title">{expandedChart.title}</h2><button type="button" autoFocus aria-label={t("बंद करें")} onClick={()=>dialog.current?.close()}><X size={22} /></button></header>
          {expandedChart.chart && <Chart chart={expandedChart.chart} detailed label={`${expandedChart.title} · ${t("जन्म अंश, नक्षत्र और पद")}`} />}
          <p>{t("अंश, नक्षत्र और पद जन्म स्थिति के हैं। ↑ ↓ स्व जन्म राशि की स्थिति हैं; ◈ D9 में वर्गोत्तम है।")}</p>
          <div className="chart-zoom-table"><table><caption className="sr-only">{t("जन्म अंश, नक्षत्र और पद")}</caption><thead><tr><th scope="col">{t("ग्रह")}</th><th scope="col">{t("अंश")}</th><th scope="col">{t("नक्षत्र / पद")}</th></tr></thead><tbody>{expandedChart.chart?.planets.map(entry=><tr key={entry.name}><th scope="row">{t(hindiPlanets[entry.name] || entry.name)} {chartStatusSymbols(entry)}</th><td>{chartDegree(entry.degree_in_rashi) || "—"}</td><td>{t(entry.nakshatra || "")} · {entry.pada ?? "—"}</td></tr>)}</tbody></table></div>
        </div>
      </dialog>
    </section>
  );
}
