import { Globe2 } from "lucide-react";
import { useLanguage } from "./i18n";
import type { Language } from "./i18n";

/** LanguageSelector renders the accessible English/Hindi selector backed by the shared language context. */
export default function LanguageSelector() {
  const { language, setLanguage } = useLanguage();
  return (
    <label className="language-selector">
      <Globe2 size={16} aria-hidden="true" />
      <span className="sr-only">Language / भाषा</span>
      <select
        aria-label="Language / भाषा"
        value={language}
        onChange={(event) => setLanguage(event.target.value as Language)}
      >
        <option value="en">English</option>
        <option value="hi">हिन्दी</option>
      </select>
    </label>
  );
}
