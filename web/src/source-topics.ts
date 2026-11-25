import type { Report, SourcePassage } from "./types";
import index from "./data/guruji-topic-sections.json";
import { sourceTextFingerprint } from "./SourcePassageText";

export const readingTopics = [
  { id: "general", name: "General", title: { hi: "सामान्य फलादेश", en: "General reading" } },
  { id: "personality", name: "Personality", title: { hi: "आपका स्वभाव", en: "Your nature" } },
  { id: "education", name: "Education", title: { hi: "पढ़ाई और सीखना", en: "Learning & direction" } },
  { id: "career", name: "Career", title: { hi: "काम और करियर", en: "Work & career" } },
  { id: "finance", name: "Finance", title: { hi: "पैसे और बचत", en: "Money & resources" } },
  { id: "relationships", name: "Relationships", title: { hi: "प्यार और साझेदारी", en: "Love & partnership" } },
  { id: "family", name: "Family", title: { hi: "घर और परिवार", en: "Home & family" } },
  { id: "wellbeing", name: "Wellbeing", title: { hi: "रोज की आदतें और आराम", en: "Balance & routine" } },
  { id: "dasha", name: "Dasha", title: { hi: "दशा का फलादेश", en: "Dasha reading" } },
];
type Part = { start: number; end: number; kind: string; text: string };
type Layout = { source: number; length: number; remedy: boolean; parts: [number, string, number][] };
export type TopicExcerpt = { entry: SourcePassage; identity: string; parts: Part[] };
const records = index.records as unknown as Record<string, Layout>;
const topicBits = index.topic_bits as Record<string, number>;
const houseTopics = index.houses as Record<string, number[]>;
const planetNames: Record<string, string> = { सूर्य: "Sun", चंद्रमा: "Moon", मंगल: "Mars", बुध: "Mercury", गुरु: "Jupiter", शुक्र: "Venus", शनि: "Saturn", राहु: "Rahu", केतु: "Ketu" };

/** sourceTopicExcerpts selects lossless original paragraphs under the prior life-area taxonomy; it never composes interpretation prose. */
export function sourceTopicExcerpts(report: Report, category: string): TopicExcerpt[] {
  const bit = topicBits[category];
  if (!bit) return [];
  return [...(report.source_readings || []), ...(report.numerology?.source_readings || [])].flatMap(entry => {
    const identity = `${entry.category}:${entry.key}:${entry.pages[0]}:${entry.pages[1]}:${sourceTextFingerprint(entry.text)}`;
    const layout = records[identity];
    const verified = layout && layout.source === entry.source && index.source_hashes[layout.source] === entry.sha256 && layout.length === entry.text.length;
    if (!verified) return category === "general" ? [{ entry, identity, parts: [{ start: 0, end: entry.text.length, kind: "paragraph", text: entry.text }] }] : [];
    const house = report.chart?.planets.find(planet => planet.name === planetNames[entry.key])?.house;
    const remedyMatches = layout.remedy && (house ? !!houseTopics[category]?.includes(house) : category === "general");
    let start = 0;
    const parts = layout.parts.map(([end, kind, mask]) => {
      const part = { start, end, kind, text: entry.text.slice(start, end), selected: !!(mask & bit) || remedyMatches };
      start = end;
      return part;
    });
    const selected = new Set(parts.flatMap((part, position) => part.selected && !["title", "subtitle", "space", "divider"].includes(part.kind) ? [position] : []));
    if (!selected.size) return [];
    // Keep original article and section labels beside excerpts, including both conditional house branches.
    let heading = -1;
    parts.forEach((part, position) => {
      if (["title", "subtitle"].includes(part.kind)) selected.add(position);
      if (["heading", "subheading"].includes(part.kind)) heading = position;
      if (selected.has(position) && heading >= 0) selected.add(heading);
    });
    return [{ entry, identity, parts: parts.filter((_, position) => selected.has(position)) }];
  });
}
