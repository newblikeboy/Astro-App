import { t, useLanguage } from "./i18n";
import type { PujaMantraReading } from "./puja-types";

/** PujaMantras presents exact Sanskrit and meanings as always-visible paragraphs directly after their worship instruction. */
export default function PujaMantras({ readings }: { readings: PujaMantraReading[] }) {
  const { language } = useLanguage();
  const count = readings.reduce((total, reading) => total + reading.sections.length, 0);
  if (!count) return null;
  return <div className="puja-mantras puja-step-mantras">
    {readings.map(reading => <div className="puja-mantra-group" key={`${reading.source}:${reading.sections[0].id}`}>
        {reading.sections.map(section => <div className="puja-mantra-section" role="group" key={section.id} data-mantra-id={section.id} aria-labelledby={`puja-${section.id}`}>
          <h3 id={`puja-${section.id}`}>{section.title[language]}</h3>
          <p className="puja-mantra-text" lang="sa">{section.text}</p>
          {section.note && <p className="puja-mantra-note">{section.note[language]}</p>}
          <div className="puja-mantra-meaning"><h4>{t("सरल भावार्थ")}</h4><p>{section.meaning[language]}</p></div>
        </div>)}
    </div>)}
  </div>;
}
