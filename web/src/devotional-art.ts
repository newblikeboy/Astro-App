import catalogue from "./data/devotional-art.json";
import refreshed from "./data/devotional-art-refreshed.json";

export type DevotionalArtwork = {
  src: string;
  label: string;
  labelHi: string;
  width: number;
  height: number;
  source: string;
  artist: string;
  license: string;
  licenseUrl: string;
  changes: string;
};

// Keep the original source catalogue intact; reviewed replacement artwork wins for both libraries.
const art: { images: Record<string, DevotionalArtwork>; readings: Record<string, string> } = {
  images: { ...catalogue.images, ...refreshed.images },
  readings: catalogue.readings,
};

/** devotionalSubjectArtwork shares a local subject image and its provenance with a guide about the same deity or observance. */
export function devotionalSubjectArtwork(subject: string): DevotionalArtwork | undefined {
  return art.images[subject];
}

/** devotionalArtwork resolves the reviewed subject image and its attribution for an individual reading. */
export function devotionalArtwork(slug: string): DevotionalArtwork | undefined {
  return art.images[art.readings[slug]];
}
