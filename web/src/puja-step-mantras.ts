import type { PujaGuide, PujaMantraReading } from "./puja-types";

/** pujaStepReadings resolves explicit per-guide placements without changing text; incomplete or duplicate maps fail the reader load. */
export function pujaStepReadings(guide: PujaGuide, collection: Record<string, PujaMantraReading>): PujaMantraReading[][] {
  if (!Array.isArray(guide.step_mantras) || guide.step_mantras.length !== guide.steps.hi.length || guide.steps.hi.length !== guide.steps.en.length) throw new Error("Puja step map unavailable");
  const available = new Map<string, { reading: PujaMantraReading; section: PujaMantraReading["sections"][number] }>();
  for (const key of guide.mantra_readings) {
    const reading = collection[key];
    if (!reading?.sections.length) throw new Error("Puja mantra reading unavailable");
    for (const section of reading.sections) {
      if (!section.text || !section.meaning?.hi || !section.meaning.en || available.has(section.id)) throw new Error("Puja mantra or meaning unavailable");
      available.set(section.id, { reading, section });
    }
  }
  const seen = new Set<string>();
  const steps = guide.step_mantras.map(ids => {
    const groups: PujaMantraReading[] = [];
    for (const id of ids) {
      const item = available.get(id);
      if (!item || seen.has(id)) throw new Error("Puja mantra placement invalid");
      seen.add(id);
      const previous = groups.at(-1);
      if (previous?.source === item.reading.source) previous.sections.push(item.section);
      else groups.push({ ...item.reading, sections: [item.section] });
    }
    return groups;
  });
  if (seen.size !== available.size) throw new Error("Puja mantra placement incomplete");
  return steps;
}
