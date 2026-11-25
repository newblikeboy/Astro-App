import type { SourcePassage } from "./types";
import { ChevronDown } from "lucide-react";
import SourcePassageText from "./SourcePassageText";
import "./guruji-faladesh.css";

/** SourcePassages displays complete Hindi passages, with optional number markers for the numerology reader. */
export default function SourcePassages({ passages, numberMarker }: { passages: SourcePassage[]; numberMarker?: number }) {
  return <div className="guruji-source-passages">{passages.map((entry, index) => <details className="guruji-source-passage" key={`${entry.source}-${entry.pages[0]}-${index}`} data-source-category={entry.category}>
    <summary lang="hi">{numberMarker === undefined ? entry.title : <><span className="source-number-marker" aria-hidden="true">{numberMarker}</span><span className="source-passage-title">{entry.title}</span><ChevronDown className="source-passage-chevron" size={18} aria-hidden="true" /></>}</summary>
    <SourcePassageText entry={entry} structured={numberMarker !== undefined} />
    <p className="guruji-source-location" lang="hi">{entry.file} · पृष्ठ {entry.pages[0]}{entry.pages[1] !== entry.pages[0] ? `–${entry.pages[1]}` : ""}</p>
  </details>)}</div>;
}
