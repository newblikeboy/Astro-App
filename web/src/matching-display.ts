import { t } from "./i18n";
import type { MatchingFactor, MatchingFactorCode } from "./matching-types";

/** factorValue formats an Ashtakoota factor value in the requested interface language. */
export function factorValue(code: MatchingFactorCode, value: string) {
  if (code === "varna") {
    const elements: Record<string, string> = {
      Brahmin: "Water",
      Kshatriya: "Fire",
      Vaishya: "Earth",
      Shudra: "Air",
    };
    return t(elements[value] || value);
  }
  if (code === "tara") {
    const i = value.indexOf(" (");
    if (i >= 0) return t(value.slice(0, i)) + value.slice(i);
  }
  return t(value);
}
const explanations: Record<MatchingFactorCode, string> = {
  varna:
    "The selected element ordering gives {v0} of {v1} points for {v2} and {v3}.",
  vashya:
    "The {v2} / {v3} pairing receives {v0} of {v1} in the selected Vashya table.",
  tara: "From the groom's star: count {v2}, Tara {v4}. From the bride's star: count {v3}, Tara {v5}. Inclusive counts cycle through 1 to 9; classes 3, 5 and 7 receive 0, and each other direction receives 1.5.",
  yoni: "The traditional {v2} / {v3} animal-group pairing receives {v0} of {v1}.",
  maitri:
    "The natural-friendship table gives {v0} of {v1} for {v2} and {v3}. Temporary planetary friendships are not used.",
  gana: "Bride {v2} and groom {v3} receive {v0} of {v1} in this directional Gana table. Reversing the roles can change this factor.",
  bhakoot:
    "The Moon signs form a {v2}/{v3} relationship. This base method gives 0 for 2/12, 5/9 or 6/8; otherwise 7.",
  nadi: "The birth-star groups are {v2} and {v3}. Same Nadi receives 0; different Nadi receives 8. No Nadi cancellation bonus is added.",
};
/** factorExplanation selects the translated factual explanation for a matching factor. */
export function factorExplanation(f: MatchingFactor) {
  let bride: string | number = factorValue(f.code, f.bride_value),
    groom: string | number = factorValue(f.code, f.groom_value);
  if (f.code === "tara") {
    bride = f.evidence?.bride_count ?? "";
    groom = f.evidence?.groom_count ?? "";
  }
  if (f.code === "bhakoot") {
    bride = f.evidence?.bride_to_groom ?? "";
    groom = f.evidence?.groom_to_bride ?? "";
  }
  return t(explanations[f.code], {
    v0: f.score,
    v1: f.max,
    v2: bride,
    v3: groom,
    v4: f.evidence?.bride_class ?? "",
    v5: f.evidence?.groom_class ?? "",
  });
}
export const mangalRuleLabels: Record<string, string> = {
  mars_own_or_exalted: "Mars in its own or exalted sign",
  mars_debilitated: "Mars in Cancer (selected cancellation tradition)",
  mars_retrograde: "Retrograde Mars",
  mars_combust: "Mars within 17 degrees of the Sun",
  mars_conjunct_jupiter: "Mars and Jupiter in the same sign",
  mars_conjunct_moon: "Mars and Moon in the same sign",
  mars_conjunct_rahu: "Mars and Rahu in the same sign",
  jupiter_aspects_seventh_mars: "Jupiter aspects seventh-house Mars",
  sign_house_exception: "Published sign-and-house exception",
  other_saturn_or_rahu_same_house:
    "Partner Saturn or Rahu in the same house number",
  other_saturn_in_mangal_house: "Partner Saturn in a specified Mangal house",
  other_malefic_in_upachaya: "Partner Mars, Saturn or Rahu in house 3, 6 or 11",
};
export const mangalStatusLabels: Record<string, string> = {
  no_active_flags: "No remaining flags under the selected rules",
  similar_active_levels: "Similar Mangal levels",
  different_active_levels: "Different Mangal levels need review",
  pair_exception_found: "A traditional pair exception was found",
};
export const scoreBandLabels: Record<string, string> = {
  below_18: "Below 18 base points",
  "18_to_24": "18 to below 24 base points",
  "24_to_32": "24 to below 32 base points",
  "32_to_36": "32 to 36 base points",
};
export const levelLabels = {
  none: "No flagged references",
  low: "Low Mangal level",
  high: "High Mangal level",
};
export const referenceLabels: Record<string, string> = {
  lagna: "Lagna",
  moon: "Moon",
  venus: "Venus",
  all: "All references",
  bride: "Bride",
  groom: "Groom",
};
