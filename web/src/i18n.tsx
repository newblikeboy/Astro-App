import { createContext, useContext, useEffect, useState } from "react";
import type { ReactNode } from "react";
import english from "./locales/en.json";
import content from "./locales/content-en.json";
import report from "./locales/report-en.json";
import panchang from "./locales/panchang-en.json";
import sky from "./locales/sky-en.json";
import design from "./locales/design-en.json";
import portal from "./locales/portal-en.json";
import explorer from "./locales/explorer-en.json";
import kundliDetails from "./locales/kundli-details-en.json";
import shop from "./locales/shop-en.json";
import shopDetail from "./locales/shop-detail-en.json";
import checkout from "./locales/checkout-en.json";
import panchangDetails from "./locales/panchang-details-en.json";
import matching from "../../internal/platform/data/matching-en.json";
import ux from "./locales/ux-en.json";
import admin from "./locales/admin-en.json";
import devotional from "./locales/devotional-en.json";
import puja from "./locales/puja-en.json";
import vivah from "./locales/vivah-en.json";
import video from "./locales/video-en.json";
import kundliWording from "../../internal/platform/data/kundli-copy.json";
import yogaConclusions from "../../internal/platform/data/yoga-conclusions.json";
import numerologyWording from "../../internal/platform/data/numerology-copy.json";

export type Language = "en" | "hi";
const key = "m4y-language";
/** initialLanguage restores an explicitly saved Hindi preference and otherwise defaults the site to English. */
function initialLanguage(): Language {
  try {
    return localStorage.getItem(key) === "hi" ? "hi" : "en";
  } catch {
    return "en";
  }
}
let currentLanguage: Language = initialLanguage();
const catalog: Record<string, string> = {
  ...english,
  ...content,
  ...report,
  ...panchang,
  ...sky,
  ...design,
  ...portal,
  ...explorer,
  ...kundliDetails,
  ...shop,
  ...shopDetail,
  ...checkout,
  ...panchangDetails,
  ...matching,
  ...ux,
  ...admin,
  ...devotional,
  ...puja,
  ...vivah,
  ...video,
};
const hindi = Object.fromEntries(
  Object.entries(catalog).map(([hi, en]) => [en, hi]),
);
const kundliCopy: Record<string, { hi: string; en: string }> = { ...kundliWording, ...yogaConclusions, ...numerologyWording };
/** getLanguage returns the language used by non-component formatting and request helpers. */
export const getLanguage = () => currentLanguage;
/** t looks up a translated string and interpolates its named variables. */
export function t(
  value: string,
  variables?: Record<string, string | number>,
): string {
  const source = value.trim();
  const wording = kundliCopy[source] || kundliCopy[catalog[source]];
  const translated =
    wording ? wording[currentLanguage] : currentLanguage === "en"
      ? (catalog[source] ?? source)
      : (hindi[source] ?? source);
  const result =
    value.slice(0, value.indexOf(source)) +
    translated +
    value.slice(value.indexOf(source) + source.length);
  return result.replace(/\{(v\d+)\}/g, (match, name) =>
    variables?.[name] === undefined ? match : String(variables[name]),
  );
}
const LanguageContext = createContext<{
  language: Language;
  setLanguage: (language: Language) => void;
}>({
  language: currentLanguage,
  /** setLanguage provides the inert default setter before a component is wrapped by LanguageProvider. */
  setLanguage: () => {},
});
/** LanguageProvider owns the English/Hindi preference and keeps React, browser storage and document language synchronized. */
export function LanguageProvider({ children }: { children: ReactNode }) {
  const [language, updateLanguage] = useState<Language>(currentLanguage);
  /** setLanguage changes the active language, persists the choice and notifies the subscribed interface. */
  const setLanguage = (next: Language) => {
    currentLanguage = next;
    updateLanguage(next);
    try {
      localStorage.setItem(key, next);
    } catch {
      /* Preference still works for this visit. */
    }
  };
  useEffect(() => {
    document.documentElement.lang = language;
    document.title =
      language === "en"
        ? "Mishra 4 You · Understand your Kundli"
        : "Mishra 4 You · अपनी कुंडली समझें";
    document
      .querySelector('meta[name="description"]')
      ?.setAttribute(
        "content",
        language === "en"
          ? "Explore your birth chart with Mishra 4 You and Pt. Raja Mishra. Create your free Kundli and choose personal guidance."
          : "Mishra 4 You — अपनी जन्म कुंडली समझें। पं. राजा मिश्रा के साथ ज्योतिषीय मार्गदर्शन।",
      );
    /** sync updates this browser tab when another tab changes the stored language preference. */
    const sync = (event: StorageEvent) => {
      if (event.key === key) {
        const next = event.newValue === "hi" ? "hi" : "en";
        currentLanguage = next;
        updateLanguage(next);
      }
    };
    window.addEventListener("storage", sync);
    return () => window.removeEventListener("storage", sync);
  }, [language]);
  return (
    <LanguageContext.Provider value={{ language, setLanguage }}>
      {children}
    </LanguageContext.Provider>
  );
}
/** useLanguage returns the current language and setter from the shared React context. */
export const useLanguage = () => useContext(LanguageContext);
