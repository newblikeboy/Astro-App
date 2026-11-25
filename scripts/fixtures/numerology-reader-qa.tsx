import { useState } from "react";
import { createRoot } from "react-dom/client";
import { LanguageProvider } from "./i18n";
import SourcePassages from "./SourcePassages";
import type { SourcePassage } from "./types";
import fixtures from "../../tmp/source-only-qa/numerology-format-fixtures.json";
import "./styles.css";
import "./kundli-details.css";
import "./numerology.css";

/** NumerologyReaderQA exercises every supplied number article and the formatting fingerprint guard without customer data. */
function NumerologyReaderQA() {
  const [index,setIndex]=useState(1);
  const [modified,setModified]=useState(false);
  const original=(fixtures as unknown as SourcePassage[])[index];
  const entry=modified?{...original,text:original.text+" "}:original;
  return <main className="container workspace"><label>QA source<select id="qa-source" value={index} onChange={event=>{setIndex(Number(event.target.value));setModified(false);}}>{fixtures.map((fixture,i)=><option value={i} key={i}>{fixture.category} {fixture.key}</option>)}</select></label><button id="qa-modified" onClick={()=>setModified(true)}>Modified snapshot QA</button><div className="kundli-panel"><section className="kundli-details numerology-panel"><SourcePassages key={`${index}-${modified}`} passages={[entry]} numberMarker={Number(entry.key)} /></section></div></main>;
}
createRoot(document.getElementById("root")!).render(<LanguageProvider><NumerologyReaderQA /></LanguageProvider>);
