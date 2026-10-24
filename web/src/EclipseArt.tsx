import { useId } from "react";
import type { EclipseEvent } from "./eclipse-summary";

/** EclipseArt draws a decorative schematic of the worldwide eclipse type, not a local observation or calculated geometry. */
export default function EclipseArt({ kind, type }: Pick<EclipseEvent, "kind" | "type">) {
  const id = `eclipse-${useId().replace(/:/g, "")}`;
  const solar = kind === "Solar";
  const total = type === "Total";
  const ring = type === "Annular" || type === "Hybrid";
  const partial = type === "Partial";
  return (
    <div className={`eclipse-art eclipse-art-${solar ? "solar" : "lunar"}`} aria-hidden="true">
      <svg viewBox="0 0 320 148" preserveAspectRatio="xMidYMid slice" focusable="false">
        <defs>
          <radialGradient id={`${id}-halo`}>
            <stop offset="0.36" stopColor={solar ? "#ffd58c" : "#c8c6ff"} stopOpacity="0.65" />
            <stop offset="0.57" stopColor={solar ? "#f6a84d" : "#b6a5f8"} stopOpacity="0.19" />
            <stop offset="1" stopColor={solar ? "#f6a84d" : "#b6a5f8"} stopOpacity="0" />
          </radialGradient>
          <radialGradient id={`${id}-sun`} cx="38%" cy="32%">
            <stop stopColor="#fff9d5" />
            <stop offset="0.7" stopColor="#ffd473" />
            <stop offset="1" stopColor="#ee982d" />
          </radialGradient>
          <radialGradient id={`${id}-moon`} cx="33%" cy="28%">
            <stop stopColor={total ? "#eda574" : "#f6f3ed"} />
            <stop offset="0.63" stopColor={total ? "#b75b3d" : "#c6ccd6"} />
            <stop offset="1" stopColor={total ? "#58283b" : "#747f9c"} />
          </radialGradient>
          <linearGradient id={`${id}-penumbra`} x1="0" x2="1" y1="1" y2="0">
            <stop stopColor="#10142e" stopOpacity="0" />
            <stop offset="0.45" stopColor="#10142e" stopOpacity="0.12" />
            <stop offset="1" stopColor="#10142e" stopOpacity="0.72" />
          </linearGradient>
          <radialGradient id={`${id}-shadow`}>
            <stop offset="0.8" stopColor="#111327" stopOpacity="0.96" />
            <stop offset="0.96" stopColor="#171a32" stopOpacity="0.9" />
            <stop offset="1" stopColor="#262b46" stopOpacity="0.65" />
          </radialGradient>
          <clipPath id={`${id}-disc`}><circle cx="160" cy="74" r="40" /></clipPath>
        </defs>
        <g fill={solar ? "#ffe7bd" : "#d9dcff"} opacity="0.55">
          <circle cx="52" cy="37" r="1" /><circle cx="255" cy="28" r="1.2" />
          <circle cx="274" cy="108" r="0.8" /><circle cx="75" cy="116" r="0.7" />
          <path d="M236 70h6m-3-3v6" stroke="currentColor" strokeWidth="0.6" />
        </g>
        <circle cx="160" cy="74" r={solar && total ? 91 : 78} fill={`url(#${id}-halo)`} />
        {solar ? (
          <>
            {(total || type === "Hybrid") && (
              <g fill="none" stroke="#ffe9c0" strokeLinecap="round">
                <ellipse cx="160" cy="74" rx="62" ry="44" opacity="0.12" />
                <ellipse cx="160" cy="74" rx="45" ry="61" opacity="0.09" />
                {[0, 45, 90, 135, 180, 225, 270, 315].map((angle) => (
                  <path key={angle} d="M160 30V17" transform={`rotate(${angle} 160 74)`} opacity="0.22" />
                ))}
              </g>
            )}
            <circle cx="160" cy="74" r="40" fill={`url(#${id}-sun)`} />
            <circle cx={partial ? 179 : 160} cy={partial ? 61 : 74} r={ring ? 35.5 : total ? 39 : 39.5} fill="#0d1020" />
            {total && <circle cx="160" cy="74" r="40" fill="none" stroke="#fff0c9" strokeWidth="1.5" />}
            {total && <circle cx="125" cy="54" r="2.5" fill="#fff3cd" />}
          </>
        ) : (
          <>
            <circle cx="160" cy="74" r="40" fill={`url(#${id}-moon)`} />
            <g clipPath={`url(#${id}-disc)`}>
              <g fill={total ? "#713627" : "#58647b"} opacity="0.2">
                <path d="M127 54q12-13 23-7t3 17q14-5 18 7t-11 17q-11 0-12-9t-17-7z" />
                <path d="M169 40q16 0 19 12t-8 13q-15-3-11-25M169 91q8-6 16 1t0 16q-17 5-16-17" />
              </g>
              <g fill="none" stroke={total ? "#f2bc91" : "#eef1f7"} strokeWidth="1.2" opacity="0.38">
                <circle cx="145" cy="96" r="8" /><circle cx="177" cy="75" r="5" />
                <circle cx="142" cy="52" r="4" /><circle cx="161" cy="44" r="2.5" />
                <circle cx="133" cy="81" r="3" /><circle cx="163" cy="102" r="3.5" />
              </g>
              {partial ? (
                <circle cx="188" cy="49" r="46" fill={`url(#${id}-shadow)`} />
              ) : !total ? (
                <circle cx="160" cy="74" r="40" fill={`url(#${id}-penumbra)`} />
              ) : null}
            </g>
            <circle cx="160" cy="74" r="40" fill="none" stroke={total ? "#efab82" : "#e0e6fb"} strokeOpacity="0.45" strokeWidth="0.7" />
          </>
        )}
      </svg>
    </div>
  );
}
