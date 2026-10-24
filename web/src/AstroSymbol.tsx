import {
  Activity,
  Compass,
  Droplets,
  Flower2,
  HeartPulse,
  Hourglass,
  Layers2,
  MoveRight,
  Orbit,
  ShieldAlert,
  Sparkles,
  TrendingUp,
  Waves,
} from "lucide-react";
import type { ReactNode } from "react";
import "./data-visuals.css";

// Decorative, locally drawn symbols always accompany a visible text label.
const planetPaths: Record<string, ReactNode> = {
  Sun: (
    <>
      <circle cx="16" cy="16" r="9" />
      <circle cx="16" cy="16" r="2" fill="currentColor" stroke="none" />
    </>
  ),
  Moon: <path d="M22 4a12 12 0 1 0 6 20A12 12 0 0 1 22 4Z" />,
  Mars: (
    <>
      <circle cx="13" cy="19" r="8" />
      <path d="m19 13 9-9m-8 0h8v8" />
    </>
  ),
  Mercury: (
    <>
      <circle cx="16" cy="14" r="6.5" />
      <path d="M9.5 3a6.5 6.5 0 0 0 13 0M16 20.5V30m-5-4h10" />
    </>
  ),
  Venus: (
    <>
      <circle cx="16" cy="11" r="8" />
      <path d="M16 19v11m-5-5h10" />
    </>
  ),
  Jupiter: <path d="M7 5c10-3 11 8 0 15h19M21 3v27" />,
  Saturn: <path d="M11 3v26M5 9h12m-6 9c10-11 18-1 12 7-2 3 0 6 3 3" />,
  Rahu: (
    <>
      <path d="M7 23v-7a9 9 0 0 1 18 0v7" />
      <circle cx="7" cy="25" r="3" />
      <circle cx="25" cy="25" r="3" />
    </>
  ),
  Ketu: (
    <>
      <path d="M7 9v7a9 9 0 0 0 18 0V9" />
      <circle cx="7" cy="7" r="3" />
      <circle cx="25" cy="7" r="3" />
    </>
  ),
};
const icons = {
  Amrit: Droplets,
  Shubh: Flower2,
  Labh: TrendingUp,
  Char: MoveRight,
  Udveg: Waves,
  Rog: HeartPulse,
  Kaal: Hourglass,
  "Rahu Kaal": ShieldAlert,
  Yamaganda: Hourglass,
  Gulika: Activity,
  Abhijit: Sparkles,
  Tithi: Orbit,
  Nakshatra: Sparkles,
  Yoga: Compass,
  Karana: Layers2,
};
/** AstroSymbol renders the shared zodiac/graha symbol with an accessible text alternative when required. */
export default function AstroSymbol({
  name,
  className = "",
}: {
  name: string;
  className?: string;
}) {
  const Icon = icons[name as keyof typeof icons] || Orbit;
  return (
    <span
      className={`astro-symbol astro-${name.toLowerCase().replaceAll(" ", "-")} ${className}`}
      aria-hidden="true"
    >
      {planetPaths[name] ? (
        <svg
          viewBox="0 0 32 32"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.7"
          strokeLinecap="round"
          strokeLinejoin="round"
          focusable="false"
        >
          {planetPaths[name]}
        </svg>
      ) : (
        <Icon size={20} strokeWidth={1.65} />
      )}
    </span>
  );
}
