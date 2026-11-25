import React from "react";
import { createRoot } from "react-dom/client";
import { useState } from "react";
import { LanguageProvider, useLanguage } from "./i18n";
import KundliReport from "./KundliReport";
import type { Report } from "./types";
import fixture from "../../tmp/source-only-qa/report.json";
import "./styles.css";

/** QA exercises actual report components with isolated synthetic snapshots and no API access. */
function QA() {
  const {setLanguage}=useLanguage();
  const [report,setReport]=useState(fixture as unknown as Report);
  return <>
    <div>
      <button id="qa-hi" onClick={()=>setLanguage("hi")}>Hindi QA</button>
      <button id="qa-en" onClick={()=>setLanguage("en")}>English QA</button>
      <button id="qa-reset" onClick={()=>setReport(structuredClone(fixture) as unknown as Report)}>Reset QA</button>
      <button id="qa-missing" onClick={()=>{const copy=structuredClone(fixture) as unknown as Report;copy.numerology!.bhagyank.value=3;setReport(copy);}}>Missing QA</button>
      <button id="qa-old-chart" onClick={()=>{const copy=structuredClone(fixture) as unknown as Report;copy.navamsa=undefined;copy.details=undefined;setReport(copy);}}>Old chart QA</button>
      <button id="qa-crowded" onClick={()=>{
        const copy=structuredClone(fixture) as unknown as Report;
        copy.chart!.planets.forEach(planet=>{planet.house=2;planet.rashi_number=7;});
        copy.navamsa!.planets.forEach(planet=>{planet.house=2;planet.rashi_number=7;});
        setReport(copy);
      }}>Crowded QA</button>
      <label>QA Lagna<select id="qa-lagna" value={report.chart?.lagna.rashi_number ?? 0} onChange={event=>{const copy=structuredClone(fixture) as unknown as Report;const lagna=Number(event.target.value);if(lagna)copy.chart!.lagna.rashi_number=lagna;else{copy.chart=null;copy.birth.unknown_time=true;}setReport(copy);}}>{Array.from({length:13},(_,lagna)=><option key={lagna} value={lagna}>{lagna}</option>)}</select></label>
      <label>QA planet sign<select id="qa-sign" value={report.chart?.planets[0].rashi_number ?? 1} onChange={event=>{const copy=structuredClone(fixture) as unknown as Report;const sign=Number(event.target.value);copy.chart!.planets.forEach(planet=>{planet.rashi_number=sign;planet.house=(sign-copy.chart!.lagna.rashi_number+12)%12+1;});setReport(copy);}}>{Array.from({length:12},(_,index)=><option key={index} value={index+1}>{index+1}</option>)}</select></label>
    </div>
    <KundliReport report={report} onBack={()=>{}} onSave={()=>{}} saving={false} buy={()=>{}} />
  </>;
}
createRoot(document.getElementById("root")!).render(<React.StrictMode><LanguageProvider><QA /></LanguageProvider></React.StrictMode>);
