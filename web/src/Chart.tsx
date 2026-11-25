import { Fragment, useLayoutEffect, useRef } from "react";
import { t, getLanguage, useLanguage } from "./i18n";
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
export type ChartPlanet = {
  name: string;
  house: number;
  degree_in_rashi?: number;
  nakshatra?: string;
  pada?: number;
  retrograde?: boolean;
  combust?: boolean | null;
  dignity?: string;
  vargottama?: boolean;
};
export type ChartData = {
  lagna: { rashi_number: number };
  planets: ChartPlanet[];
  houses?: { house: number; rashi_number: number }[];
};

/** chartDegree truncates a recorded natal angle to degrees/minutes without rounding into a different sign. */
export function chartDegree(value?: number) {
  if (value === undefined || !Number.isFinite(value)) return "";
  const minutes = Math.floor(value * 60 + 1e-8);
  return `${Math.floor(minutes / 60)}°${String(minutes % 60).padStart(2, "0")}′`;
}

/** chartStatusSymbols exposes supplied condition flags, using a circular return arrow for retrograde motion. */
export function chartStatusSymbols(planet: ChartPlanet) {
  return `${planet.retrograde ? "↺" : ""}${planet.combust ? "†" : ""}${planet.dignity === "Exaltation sign" ? "↑" : planet.dignity === "Debilitation sign" ? "↓" : planet.dignity === "Own sign" ? "स्व" : ""}${planet.vargottama ? "◈" : ""}`;
}

/** chartWithNatalDetails combines displayed house assignments with recorded natal annotations, never deriving divisional longitudes. */
export function chartWithNatalDetails(chart: ChartData, report: Report, natalDignity = true): ChartData {
  return { ...chart, planets: chart.planets.map(planet => {
    const natal = report.chart?.planets.find(entry => entry.name === planet.name);
    const condition = report.details?.analysis.conditions.find(entry => entry.name === planet.name);
    return { ...natal, ...planet, combust: condition?.combust, dignity: natalDignity ? condition?.dignity : undefined };
  }) };
}

/** houseSpan finds the usable horizontal interval inside a chart house at a given SVG ordinate. */
function houseSpan(index: number, y: number) {
  const vertices = houseShapes[index].split(" ").map(point => point.split(",").map(Number));
  const intersections: number[] = [];
  vertices.forEach(([x1,y1],i) => {
    const [x2,y2] = vertices[(i+1)%vertices.length];
    if (y1 !== y2 && y >= Math.min(y1,y2) && y <= Math.max(y1,y2)) intersections.push(x1+(y-y1)*(x2-x1)/(y2-y1));
  });
  return { left: Math.min(...intersections), right: Math.max(...intersections) };
}

/** houseLabelSpan reserves ascender/descender space so rectangular text bounds stay inside diagonal house edges. */
function houseLabelSpan(index: number, y: number, font: number) {
  const top = houseSpan(index,y-font*1.6);
  const bottom = houseSpan(index,y+font*.6);
  return { left: Math.max(top.left,bottom.left)+3, right: Math.min(top.right,bottom.right)-3 };
}

/** detailedHouseRows separates planet/nakshatra pairs and reserves polygon space without stretching letter shapes. */
function detailedHouseRows(index: number, sign: number, planets: ChartPlanet[]) {
  const english = getLanguage() === "en";
  const rows = [{ text: String(sign), kind: "sign", planet: undefined as ChartPlanet | undefined, font: 7.5 }];
  planets.forEach(planet => {
    const name = english ? shortNames[planet.name] : hindiShortNames[planet.name];
    rows.push({text:`${name || planet.name} ${chartDegree(planet.degree_in_rashi)}${chartStatusSymbols(planet)}`,kind:"position",planet,font:11});
    if (planet.nakshatra) rows.push({text:`${t(planet.nakshatra)}${planet.pada ? ` · ${planet.pada}` : ""}`,kind:"nakshatra",planet,font:8});
  });
  const ordinates = houseShapes[index].split(" ").map(point=>Number(point.split(",")[1]));
  const minY = Math.min(...ordinates), maxY = Math.max(...ordinates);
  const offsets = [0];
  rows.slice(1).forEach((row,i) => offsets.push(offsets[i] + (row.kind === "nakshatra" ? 12.5 : 16)));
  const scale = Math.min(1,(maxY-minY-18)/(offsets[offsets.length-1]+14));
  const height = offsets[offsets.length-1]*scale;
  let bestStart = (minY+maxY-height)/2, bestScore = -Infinity;
  for (let start=minY+7.5*scale*1.6+3;start+height<maxY-8*scale*.6-3;start+=1) {
    const score = rows.reduce((sum,row,i)=> {
      const span = houseLabelSpan(index,start+offsets[i]*scale,row.font*scale);
      return sum + Math.log(Math.max(1,span.right-span.left) / Math.max(12,row.text.length*row.font*.55));
    },0);
    if(score>bestScore){bestScore=score;bestStart=start;}
  }
  return rows.map((row,i)=> {
    const y=bestStart+offsets[i]*scale;
    const span=houseLabelSpan(index,y,row.font*scale);
    const font=row.font*scale;
    return {...row,x:(span.left+span.right)/2,y,font};
  });
}
/** Chart draws a chart's signs and grahas in the supported visual layout using calculated placements. */
export default function Chart({
  report,
  sample = false,
  chart: suppliedChart,
  label,
  activeHouse,
  detailed = false,
}: {
  report?: Report;
  sample?: boolean;
  chart?: ChartData;
  label?: string;
  activeHouse?: number;
  detailed?: boolean;
}) {
  const chart = suppliedChart ?? report?.chart;
  const svg = useRef<SVGSVGElement>(null);
  const { language } = useLanguage();
  useLayoutEffect(() => {
    if (!detailed || sample) return;
    let active = true;
    let frame = 0;
    /** scheduleFit waits until the browser has laid out the current SVG glyphs before measuring them. */
    function scheduleFit() {
      if (!active) return;
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(() => { frame = requestAnimationFrame(fitChartLabels); });
    }
    /** fitChartLabels reads one settled layout, then shrinks only overflowing labels and rechecks on a later frame. */
    function fitChartLabels() {
      if (!active || !svg.current?.getClientRects().length) return;
      let changed = false;
      svg.current.querySelectorAll<SVGTextElement>("text[data-base-font]").forEach(label => {
        const font = Number(label.getAttribute("font-size"));
        const index = Number(label.parentElement?.dataset.house)-1;
        const box = label.getBBox();
        const top = houseSpan(index,box.y-1.5), bottom = houseSpan(index,box.y+box.height+1.5);
        const left = Math.max(top.left,bottom.left)+4, right = Math.min(top.right,bottom.right)-4;
        const available = right-left;
        if (!Number.isFinite(available) || available <= 0) {
          label.setAttribute("font-size",String(font*.75));
          changed = true;
          return;
        }
        const center = (left+right)/2;
        if (Math.abs(Number(label.getAttribute("x"))-center) > .05) {
          label.setAttribute("x",String(center));
          changed = true;
        }
        const naturalWidth = Math.max(box.width,label.getComputedTextLength());
        if (naturalWidth > available) {
          label.setAttribute("font-size",String(font*available/naturalWidth*.96));
          changed = true;
        }
        const badge = label.previousElementSibling;
        if (badge?.classList.contains("chart-sign-badge")) {
          badge.setAttribute("x",String(box.x-3));
          badge.setAttribute("y",String(box.y-1));
          badge.setAttribute("width",String(box.width+6));
          badge.setAttribute("height",String(box.height+2));
        }
      });
      svg.current.dataset.labelsFitted = String(!changed);
      if (changed) scheduleFit();
    }
    /** resetFit restores preferred type sizes after content or font changes, keeping reads separate from writes. */
    function resetFit() {
      if (!active || !svg.current) return;
      svg.current.dataset.labelsFitted = "false";
      svg.current.querySelectorAll<SVGTextElement>("text[data-base-font]").forEach(label => label.setAttribute("font-size",label.dataset.baseFont!));
      scheduleFit();
    }
    resetFit();
    void document.fonts.ready.then(resetFit);
    document.fonts.addEventListener("loadingdone",resetFit);
    const observer = new ResizeObserver(scheduleFit);
    if (svg.current) observer.observe(svg.current);
    return () => { active = false; cancelAnimationFrame(frame); observer.disconnect(); document.fonts.removeEventListener("loadingdone",resetFit); };
  }, [chart,detailed,sample,language]);
  return (
    <svg
      ref={svg}
      className={`kundli-chart${detailed ? " chart-annotated" : ""}`}
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
        className="chart-frame"
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
        className="chart-lines"
        d="M0 0L300 300M300 0L0 300M150 0L300 150L150 300L0 150Z"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.1"
      />
      {centers.map(([x, y], i) => {
        const planets = chart?.planets.filter((p) => p.house === i + 1) || [];
        const sign = chart ? (chart.houses?.find(h=>h.house===i+1)?.rashi_number ?? ((chart.lagna.rashi_number+i-1)%12)+1) : ((6+i-1)%12)+1;
        if (detailed && !sample) return <g key={i} textAnchor="middle" data-house={i+1}>
          {detailedHouseRows(i,sign,planets).map((row,n)=><Fragment key={n}>
            {row.kind === "sign" && <rect className="chart-sign-badge" rx="2.5" x={row.x-6} y={row.y-8} width="12" height="10" />}
            <text x={row.x} y={row.y} fontSize={row.font} data-base-font={row.font} className={`chart-detail-${row.kind}`} data-planet={row.planet?.name}>
            {row.planet && <title>{`${t(hindiPlanets[row.planet.name] || row.planet.name)} · ${chartDegree(row.planet.degree_in_rashi)} · ${t(row.planet.nakshatra || "")} · ${row.planet.pada || ""} ${chartStatusSymbols(row.planet)}`}</title>}
            {row.kind === "position" && row.planet ? <><tspan className="chart-planet-name">{(language === "en" ? shortNames[row.planet.name] : hindiShortNames[row.planet.name]) || row.planet.name}</tspan><tspan className="chart-planet-degree"> {chartDegree(row.planet.degree_in_rashi)}</tspan><tspan className="chart-planet-status" fontSize=".8em">{chartStatusSymbols(row.planet)}</tspan></> : row.text}
          </text></Fragment>)}
        </g>;
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
              {sign}
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
