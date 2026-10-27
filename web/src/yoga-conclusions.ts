import type { MangalAssessment } from "./matching-types";
import { t } from "./i18n";

/** placementConclusion explains a recorded base configuration without implying strength or cancellation checks. */
export function placementConclusion(code: string, matches: boolean) {
  const base = code.endsWith("-base") || code.startsWith("mangal-");
  return {
    status: t(`placement.${base ? "base-" : ""}${matches ? "present" : "absent"}`, { v1: t(`yoga.name.${code}`) }),
    meaning: t(matches ? `yoga.meaning.${code}` : "placement.absent-meaning"),
  };
}

/** mangalConclusion distinguishes an absent base placement from cancellation and a remaining indication. */
export function mangalConclusion(assessment: MangalAssessment) {
  const state = assessment.active_count > 0 ? "present" : assessment.base_count > 0 ? "cancelled" : "absent";
  return { status: t(`mangal.${state}`), meaning: t(state === "present" ? "mangal.meaning" : `mangal.${state}-meaning`) };
}
