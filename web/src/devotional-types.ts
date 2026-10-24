export type DevotionalCategory = "aarti" | "chalisa" | "stotram" | "ashtakam" | "mantra" | "kavacham";

export type DevotionalText = {
  slug: string;
  categories: DevotionalCategory[];
  title: string;
  titleHi: string;
  incipit: string;
  source: string;
  textLanguage?: "hi" | "sa-Latn";
  verses: string[];
};

export const devotionalCategories: { id: DevotionalCategory; label: string }[] = [
  { id: "aarti", label: "आरती" },
  { id: "chalisa", label: "चालीसा" },
  { id: "stotram", label: "स्तोत्रम्" },
  { id: "ashtakam", label: "अष्टकम्" },
  { id: "mantra", label: "वैदिक मंत्र" },
  { id: "kavacham", label: "कवचम्" },
];
