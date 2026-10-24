import type { ReactNode } from "react";
import { t } from "./i18n";
import "./time-range.css";

/** TimeRange renders an interval with clear start/end labels and localized minute-precision timestamps. */
export default function TimeRange({
  start,
  end,
  format,
}: {
  start: string;
  end: string;
  format: (value: string) => ReactNode;
}) {
  return (
    <span className="time-range">
      <span className="time-range-boundary">
        <small>{t("Start time")}</small>
        <time dateTime={start}>{format(start)}</time>
      </span>
      <span className="time-range-boundary">
        <small>{t("End time")}</small>
        <time dateTime={end}>{format(end)}</time>
      </span>
    </span>
  );
}
