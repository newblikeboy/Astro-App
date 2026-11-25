type Wording = { hi: string; en: string };

export type FaladeshSynthesisTopic = {
  reader?:{id:string;scope:"natal"|"period"|"action";text:Wording;evidence_ids:string[]}[];
  resolution?:{factor:string;direction:string;evidence_ids:string[];rule:string}[];
  remedy_recommendation?:{method:string;status:string;planet?:string;approach:string;reason:Wording;practice:Wording;evidence_ids:string[];source?:GurujiLocator};
  category:string;assessment:"supportive"|"attention"|"mixed"|"context";
  conclusion:Wording;present:Wording;potential:Wording;
  period_relevant?:boolean;
  findings?:{id:string;kind:string;scope:"natal"|"period";tone:string;title:Wording;text:Wording;basis:Wording;qualification:Wording;evidence_ids:string[];source_ids:string[]}[];
  main_points:{tone:string;text:Wording;evidence_ids:string[]}[];
  evidence:{id:string;kind:string;planet:string;house?:number;direction:string;text:Wording}[];
  related_planets:string[];
  next_period?:{lord:string;start:string;end:string;text:Wording};
};
export type FaladeshSynthesis = {reader_version?:string;version:string;as_of:string;topics:FaladeshSynthesisTopic[];coverage:string[];limits:Wording[]};
export type GurujiLocator = { source_id: string; page_start: number; page_end: number };
type Passage = { hindi_excerpt: string; source: GurujiLocator };
export type GurujiReading = Passage & {
  narrative?: {version: string; meaning: Wording; support: Wording; caution: Wording};
  planet: string; house: number; title: Wording; summary: Wording;
  role: "karak" | "marak" | "neutral" | "unlisted";
  dignity: "exalted" | "debilitated" | "ordinary";
  sign_number?:number;
  sign_status?:{primary:"exalted"|"debilitated"|"own"|"other";exalted:boolean;debilitated:boolean;own_sign:boolean};
  role_effect?:{direction:"support"|"attention"|"context";explanation:Wording};
  indication: "supportive" | "challenging" | "mixed" | "neutral" | "unclassified";
  remedy: Passage & { planet: string };
};
export type GurujiFaladeshSnapshot = {
  classification_version?:string;
  house_system?:string;
  analysis?:FaladeshSynthesis;
  version: string; lagna: number;
  sources: { id: string; filename: string; archive: string; sha256: string }[];
  readings: GurujiReading[];
  house_references: { house: number; meaning: Wording; positive: Passage; negative: Passage }[];
};
