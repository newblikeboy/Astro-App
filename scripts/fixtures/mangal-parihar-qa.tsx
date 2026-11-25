import { useState } from "react";
import { createRoot } from "react-dom/client";
import { LanguageProvider, useLanguage } from "./i18n";
import { MatchingMangal } from "./MatchingDetails";
import { ExpandedYogaPatterns } from "./KundliDetails";
import type { MangalAssessment, MatchingResponse } from "./matching-types";
import type { Report } from "./types";
import source from "../../internal/swisseph/data/guruji-mangal-parihar.json";
import fixture from "../../.local/mangal-parihar-live-fixture.json";
import "./styles.css";
import "./matching.css";

/** QA renders hypothetical display states; native tests independently verify the natal conditions that produce each state. */
function QA() {
  const { setLanguage } = useLanguage();
  const [state, setState] = useState<NonNullable<MangalAssessment["classification"]>>("mangal");
  const [natalPresent,setNatalPresent]=useState(false);
  const copy = structuredClone(fixture) as unknown as MatchingResponse;
  const rule = source.rules[state === "cancelled" ? 23 : 2];
  const assessment: MangalAssessment = {
    ...copy.mangal.bride, classification: state, base_classification: state === "cancelled" ? "mangal" : state,
    active_count: state === "none" || state === "cancelled" ? 0 : state === "partial" ? 1 : 2,
    rules: state === "none" ? [] : [
      { code: `guruji-parihar-${rule.number}`, source_pdf_page: 0, reference: "all", source_rule_number: rule.number, heading: rule.heading, text: rule.body, met: true, effect: state === "cancelled" ? "cancellation" : "mitigation", evaluation: "met", matched_clauses: state === "cancelled" ? [source.rules[23].body.split("\n").map((line) => line.trim()).find((line) => line.startsWith("2. "))!] : [] },
      { code: "guruji-parihar-15", source_pdf_page: 0, reference: "natal", source_rule_number: 15, heading: source.rules[14].heading, text: source.rules[14].body, met: true, effect: "review", evaluation: "review" },
      { code: "guruji-parihar-19", source_pdf_page: 0, reference: "natal", source_rule_number: 19, heading: source.rules[18].heading, text: source.rules[18].body, met: false, effect: "review", evaluation: "review" },
    ],
  };
  copy.mangal.bride = assessment;
  copy.mangal.groom = assessment;
  const report = { details: { mangal: assessment, analysis: { patterns: [{code:"gajakesari-base",matches:true},{code:"budha-aditya-base",matches:false}] }, yoga_checks: {kaal_sarp:{status:natalPresent?"present":"absent",rahu_house:1,ketu_house:7},rajyog:{matches:natalPresent,yogakarakas:natalPresent?[{planet:"Mars"}]:[]}} } } as unknown as Report;
  return <main id="parihar-qa" style={{ maxWidth: 1160, margin: "20px auto", padding: 16 }}>
    <h1>Mangal parihar QA</h1>
    <nav aria-label="QA controls">
      <button id="qa-hi" onClick={() => setLanguage("hi")}>Hindi QA</button>
      <button id="qa-en" onClick={() => setLanguage("en")}>English QA</button>
      <button id="qa-natal-present" onClick={()=>setNatalPresent(true)}>Present natal QA</button>
      <button id="qa-natal-absent" onClick={()=>setNatalPresent(false)}>Absent natal QA</button>
      {(["mangal", "partial", "cancelled", "none"] as const).map((value) => <button id={`qa-state-${value}`} key={value} onClick={() => setState(value)}>{value}</button>)}
    </nav>
    <MatchingMangal result={copy} />
    <ExpandedYogaPatterns report={report} />
  </main>;
}

createRoot(document.getElementById("root")!).render(<LanguageProvider><QA /></LanguageProvider>);
