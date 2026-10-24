import type { Report, Rule } from "./types";

export type VideoLink = {
  slug: string;
  title: string;
  topic: string;
  sign: number;
  sign_basis: "moon" | "lagna" | "sun";
  rule_ids: string[];
  enabled: boolean;
  version: number;
  created?: string;
  updated?: string;
};

export const videoTopics = [
  ["debt", "Debt & finances"], ["promotion", "Promotion & career growth"],
  ["zodiac", "Zodiac sign"], ["general", "General"], ["personality", "Personality"],
  ["education", "Education"], ["career", "Career"], ["relationships", "Relationships"],
  ["family", "Family"], ["wellbeing", "Wellbeing"], ["dasha", "Dasha"],
] as const;

export const videoTopicHouses: Record<string, number[]> = {
  debt: [2, 6, 8, 11, 12], promotion: [6, 10, 11], personality: [1, 5],
  education: [4, 5, 9], career: [2, 6, 10, 11], relationships: [5, 7],
  family: [2, 4], wellbeing: [1, 6, 8, 12],
};

/** videoSharePath builds a public topic entry with bounded campaign attribution and no birth details in the URL. */
export function videoSharePath(slug: string) {
  return "/?" + new URLSearchParams({ video: slug, utm_source: "youtube", utm_medium: "video", utm_campaign: slug });
}

/** videoSign reads the campaign's chosen sign reference from the calculated report; unknown-time reports remain unavailable. */
export function videoSign(report: Report, link: VideoLink) {
  if (link.sign_basis === "lagna") return report.chart?.lagna.rashi_number;
  return report.chart?.planets.find((planet) => planet.name === (link.sign_basis === "sun" ? "Sun" : "Moon"))?.rashi_number;
}

/** videoReadings selects explicitly linked approved readings already matched by the engine, suppressing a mismatched zodiac audience. */
export function videoReadings(report: Report, link: VideoLink): Rule[] {
  if (!report.chart || (link.topic === "zodiac" && videoSign(report, link) !== link.sign)) return [];
  return report.rules.filter((rule) => link.rule_ids.includes(rule.id) && !rule.demo && rule.status === "approved" && Boolean(rule.approver && rule.approved_at));
}

