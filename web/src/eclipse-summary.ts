export type EclipseEvent = {
  kind: "Solar" | "Lunar";
  type: string;
  maximum: string;
  visible: boolean;
  local?: { type: string; maximum: string };
};

// Previous/next refer to the native global maximum, with a shared cutoff:
// maxima at or before the reference instant are past, later maxima are next.
export function eclipseNeighbours(events: EclipseEvent[], at: number) {
  return (["Solar", "Lunar"] as const).map((kind) => {
    const sorted = events
      .filter(
        (event) =>
          event.kind === kind && Number.isFinite(Date.parse(event.maximum)),
      )
      .sort((a, b) => Date.parse(a.maximum) - Date.parse(b.maximum));
    return {
      kind,
      previous: sorted
        .filter((event) => Date.parse(event.maximum) <= at)
        .at(-1),
      next: sorted.find((event) => Date.parse(event.maximum) > at),
    };
  });
}

/** eclipseSearchYears chooses bounded adjacent years needed to find previous/next eclipses around an instant. */
export function eclipseSearchYears(
  at: number,
  minimum: number,
  maximum: number,
) {
  if (!Number.isFinite(at)) return [];
  const year = Number(
    new Intl.DateTimeFormat("en", {
      year: "numeric",
      timeZone: "Asia/Kolkata",
    }).format(new Date(at)),
  );
  if (year < minimum || year > maximum) return [];
  return [year - 1, year, year + 1].filter(
    (value) => value >= minimum && value <= maximum,
  );
}
