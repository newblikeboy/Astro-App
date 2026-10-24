// Display precision only. Original instants remain authoritative for maths,
// boundary comparisons and saved reports.
export const clockMinutes = (value: string) => value.slice(0, 5);
/** minuteTimestamp formats an instant in the requested locale/timezone while omitting displayed clock seconds. */
export function minuteTimestamp(value: string, zone: string, language: string) {
  return new Intl.DateTimeFormat(language === "hi" ? "hi-IN" : "en-IN", {
    timeZone: zone,
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    timeZoneName: "short",
  }).format(new Date(value));
}
