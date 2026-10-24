// Upright zodiac line symbols; decorative wherever the sign name is also shown.
const shapes = [
  "M12 21V9C12-1 0 3 5 10M12 9C12-1 24 3 19 10",
  "M4 3C4 10 20 10 20 3M18 15a6 6 0 1 1-12 0 6 6 0 0 1 12 0",
  "M5 3Q12 6 19 3M5 21Q12 18 19 21M8 5V19M16 5V19",
  "M20 6H7a4 4 0 1 0 4 4c0-2-2-4-4-4M4 18h13a4 4 0 1 0-4-4c0 2 2 4 4 4",
  "M7 14a3 3 0 1 0 0 6 3 3 0 0 0 0-6M9 15C12 10 4 4 10 3c8-2 3 12 7 15q2 2 4-1",
  "M3 5v14M3 8q4-7 6 0v11M9 8q4-7 6 0v9q0 5 6 1M15 10q8-6 5 1l-6 9",
  "M3 20h18M3 13h5a5 5 0 1 1 8 0h5",
  "M3 5v14M3 8q4-7 6 0v11M9 8q4-7 6 0v7q0 5 6 3M18 15l3 3-3 3",
  "M4 20 20 4M11 4h9v9M6 10l8 8",
  "M3 5q4 0 4 9M7 8q5-8 6-1v10q0 6 5 4a4 4 0 1 0-5-4",
  "m2 9 4-4 4 4 4-4 4 4 4-4M2 19l4-4 4 4 4-4 4 4 4-4",
  "M5 3q9 9 0 18M19 3q-9 9 0 18M3 12h18",
];
/** ZodiacMark renders the shared zodiac glyph/mark for the requested sign. */
export default function ZodiacMark({ sign }: { sign: number }) {
  return (
    <svg
      className="zodiac-mark"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d={shapes[sign - 1]} />
    </svg>
  );
}
