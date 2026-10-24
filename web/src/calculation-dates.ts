export const MIN_CALCULATION_YEAR = 1940;
export const MAX_CALCULATION_YEAR = 2099;
export const MIN_CALCULATION_DATE = `${MIN_CALCULATION_YEAR}-01-01`;
export const MAX_CALCULATION_DATE = `${MAX_CALCULATION_YEAR}-12-30`;

/** indiaTimeZoneLabel returns the localized India timezone label used by daily calculation controls. */
export function indiaTimeZoneLabel(at: string): string {
  const offset = new Intl.DateTimeFormat("en", {
    timeZone: "Asia/Kolkata",
    timeZoneName: "longOffset",
  })
    .formatToParts(new Date(at))
    .find((part) => part.type === "timeZoneName")!
    .value.replace("GMT", "UTC");
  return offset === "UTC+05:30" ? "IST (UTC+05:30)" : offset;
}
