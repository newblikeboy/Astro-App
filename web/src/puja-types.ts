export type PujaKind = "festival" | "deity" | "vrat" | "homa";
export type PujaText = { en: string; hi: string };
export type PujaSummary = {
  slug: string;
  kind: PujaKind;
  title: PujaText;
  description: PujaText;
  art: string;
  related: string;
};
export type PujaGuide = PujaSummary & {
  materials: { en: string[]; hi: string[] };
  steps: { en: string[]; hi: string[] };
  note: PujaText;
  sources: string[];
};
export const pujaKinds: { id: PujaKind; label: string }[] = [
  { id: "festival", label: "Festival pujas" },
  { id: "deity", label: "Deity worship" },
  { id: "vrat", label: "Vrat & observances" },
  { id: "homa", label: "Homa & havan" },
];
