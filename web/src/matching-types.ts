import type { Birth, Place, Position, Report } from "./types";

export type MatchingRole = "bride" | "groom";
export type MatchingFactorCode =
  "varna" | "vashya" | "tara" | "yoni" | "maitri" | "gana" | "bhakoot" | "nadi";

export type MatchingFactor = {
  code: MatchingFactorCode;
  name: string;
  score: number;
  max: number;
  bride_value: string;
  groom_value: string;
  detail_code?: string;
  evidence?: Record<string, number>;
};

export type MatchingPerson = {
  birth: Birth;
  place: Place;
  moon: Position & { nakshatra_number?: number };
  lagna: Position & { nakshatra_number?: number };
  chart: NonNullable<Report["chart"]> & { birth_time: string };
  navamsa: NonNullable<Report["navamsa"]>;
  dasha: NonNullable<Report["dasha"]>;
  mars_checks: {
    from_lagna: number;
    from_moon: number;
    lagna_flagged: boolean;
    moon_flagged: boolean;
  };
};

export type MangalRule = {
  code: string;
  met: boolean;
  reference: string;
  source_pdf_page: number;
  source_rule_number?: number;
  heading?: string;
  text?: string;
  effect?: "cancellation" | "mitigation" | "review" | "balance";
  evaluation?: "met" | "not_met" | "review";
  matched_clauses?: string[];
};
export type MangalAssessment = {
  classification?: "mangal" | "partial" | "none" | "cancelled";
  base_classification?: "mangal" | "partial" | "none";
  strength?:{method:string;conditions:{planet:string;eligible:boolean;ratio:number;confirmed:boolean;limitations:string[]}[]};
  references: {
    reference: "lagna" | "moon" | "venus";
    house: number;
    flagged: boolean;
    cancelled: boolean;
    exceptions: string[];
  }[];
  base_count: number;
  active_count: number;
  base_level: "none" | "low" | "high";
  level: "none" | "low" | "high";
  sun_separation_degrees: number;
  rules: MangalRule[];
  review_conditions: string[];
  not_evaluated: string[];
};
export type MatchingProfile = {
  id: string;
  birth: Birth;
  place: Place;
  created: string;
};

export type MatchingResponse = {
  exceptions?:{method:string;nadi:MatchingDosha;bhakoot:MatchingDosha};
  method: string;
  calculated_at: string;
  total: number;
  max: number;
  factors: MatchingFactor[];
  cautions: string[];
  bride: MatchingPerson;
  groom: MatchingPerson;
  settings: Record<string, string>;
  mangal: {
    method: string;
    bride: MangalAssessment;
    groom: MangalAssessment;
    status: string;
    rules: MangalRule[];
  };
  summary: { band: string; review_codes: string[] };
};

export type MatchingDosha={code:string;status:"absent"|"present"|"cancelled";base_present:boolean;active:boolean;rules:{code:string;met:boolean;source_pdf_page:number}[]};
