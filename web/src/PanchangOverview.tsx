import { useState } from "react";
import type { ReactNode } from "react";
import { ChevronDown, Clock3, Compass } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import AstroSymbol from "./AstroSymbol";
import PanchangDetails, { LunarFacts } from "./PanchangDetails";
import TimeRange from "./TimeRange";
import type { DailyPeriod, PanchangDetailsData } from "./panchang-types";
import { t } from "./i18n";

type OverviewDay = {
  sunrise: string;
  next_sunrise: string;
  tithi: DailyPeriod[];
  nakshatra: DailyPeriod[];
  yoga: DailyPeriod[];
  karana: DailyPeriod[];
  details?: PanchangDetailsData;
};

/** PanchangDisclosure keeps detailed results closed initially and mounts them only after the first deliberate expansion, preserving later selections. */
function PanchangDisclosure({ title, hint, Icon, children, className }: {
  title: string;
  hint: string;
  Icon: LucideIcon;
  children: ReactNode;
  className: string;
}) {
  const [visited, setVisited] = useState(false);
  return (
    <details className={"panchang-disclosure " + className} onToggle={(event) => {
      if (event.currentTarget.open) setVisited(true);
    }}>
      <summary>
        <span className="panchang-disclosure-icon"><Icon size={20} aria-hidden="true" /></span>
        <span className="panchang-disclosure-copy"><strong>{t(title)}</strong><small>{t(hint)}</small></span>
        <ChevronDown size={18} className="panchang-disclosure-chevron" aria-hidden="true" />
      </summary>
      {visited && <div className="panchang-disclosure-body">{children}</div>}
    </details>
  );
}

/** PanchangOverview shows one relevant value per core limb and retains every precise interval and extended result behind optional disclosures. */
export default function PanchangOverview({ data, stamp, isCurrent }: {
  data: OverviewDay;
  stamp: (value: string) => string;
  isCurrent: (period: DailyPeriod) => boolean;
}) {
  const limbs: [string, DailyPeriod[]][] = [
    ["Tithi", data.tithi], ["Nakshatra", data.nakshatra],
    ["Yoga", data.yoga], ["Karana", data.karana],
  ];
  const current = limbs.some(([, periods]) => periods.some(isCurrent));
  return (
    <div className="panchang-overview">
      <div className="panchang-overview-heading">
        <div><span className="panchang-eyebrow">{t("The day's essentials")}</span><h3>{t("Your day, at a glance")}</h3></div>
        <span className="panchang-snapshot-label">{t(current ? "Current period" : "At sunrise")}</span>
      </div>
      <div className="daily-limbs panchang-essentials">
        {limbs.map(([label, periods], index) => {
          const period = periods.find(isCurrent) || periods[0];
          if (!period) return null;
          return (
            <article key={label} className={"panchang-essential" + (index === 0 ? " panchang-essential-featured" : "")}>
              <div className="panchang-essential-label"><AstroSymbol name={label} /><span>{t(label)}</span></div>
              <h4>{t(period.name)}</h4>
              <p>{t("Until")} <time dateTime={period.end}>{stamp(period.end)}</time></p>
            </article>
          );
        })}
      </div>
      <div className="panchang-expansions">
        <PanchangDisclosure className="panchang-timeline-disclosure" Icon={Clock3} title="All changes in the day" hint="Every Tithi, Nakshatra, Yoga and Karana interval">
          <p className="daily-explainer">{t("Start and end times are shown for each interval. The first interval in this day view starts at sunrise; subsequent intervals start at each change.")}</p>
          <div className="panchang-timeline-grid">
            {limbs.map(([label, periods]) => <div className="daily-element" key={label}>
              <div className="daily-element-heading"><AstroSymbol name={label} /><h3>{t(label)}</h3></div>
              {periods.map((period, index) => <div className="daily-limb" key={index}>
                <strong>{t(period.name)}</strong>
                {isCurrent(period) && <small className="daily-now">{t("Now")}</small>}
                <TimeRange start={period.start} end={period.end} format={stamp} />
              </div>)}
            </div>)}
          </div>
          <div className="daily-boundary"><TimeRange start={data.sunrise} end={data.next_sunrise} format={stamp} /></div>
        </PanchangDisclosure>
        {data.details && <PanchangDisclosure className="panchang-extended-disclosure" Icon={Compass} title="Explore the detailed Panchang" hint="Moon, calendar, Yogas, Balam, Lagna and directions">
          <LunarFacts details={data.details} stamp={stamp} />
          <PanchangDetails details={data.details} stamp={stamp} />
        </PanchangDisclosure>}
      </div>
    </div>
  );
}
