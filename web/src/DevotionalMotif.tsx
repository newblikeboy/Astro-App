import type { DevotionalCategory } from "./devotional-types";

/** DevotionalMotif draws a decorative worship symbol for each collection without adding media or accessible-name noise. */
export default function DevotionalMotif({ category, className = "" }: {
  category: DevotionalCategory | "all";
  className?: string;
}) {
  return <svg className={`devotional-motif ${className}`} viewBox="0 0 80 80" fill="none"
    stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"
    aria-hidden="true" focusable="false">
    <circle cx="40" cy="40" r="35" opacity=".2" />
    <circle cx="40" cy="40" r="31" strokeDasharray="1 6" opacity=".45" />
    {category === "aarti" ? <>
      <path d="M19 46h42c-3 11-11 17-21 17S22 57 19 46Z" fill="currentColor" fillOpacity=".12" />
      <path d="M25 66h30M40 45v-9M40 35c-12-6-6-16 1-22-1 8 11 13 5 19-1 2-3 3-6 3Z" fill="currentColor" fillOpacity=".15" />
      <path d="m16 31 5 3m43-3-5 3M25 19l3 5m27-5-3 5" />
    </> : category === "chalisa" ? <>
      <path d="M39 17c-4-1-6 3-6 7l-2 17-8 10 10 12 7-9V23c0-3 0-5-1-6Z" fill="currentColor" fillOpacity=".1" />
      <path d="M41 17c4-1 6 3 6 7l2 17 8 10-10 12-7-9V23c0-3 0-5 1-6ZM19 54l12 13 5-5-12-13-5 5Zm42 0L49 67l-5-5 12-13 5 5Z" />
      <path d="M22 28h-6m48 0h-6M27 19l-4-5m30 5 4-5M40 9V5" opacity=".65" />
    </> : category === "stotram" ? <>
      <path d="M40 57C22 45 31 27 40 16c9 11 18 29 0 41Z" fill="currentColor" fillOpacity=".12" />
      <path d="M40 57C19 56 15 41 16 29c13 2 23 12 24 28Zm0 0c21-1 25-16 24-28-13 2-23 12-24 28Z" fill="currentColor" fillOpacity=".08" />
      <path d="M40 57C25 64 14 55 10 43c12-1 22 4 30 14Zm0 0c15 7 26-2 30-14-12-1-22 4-30 14ZM27 66h26" />
    </> : category === "ashtakam" ? <>
      {Array.from({ length: 8 }, (_, index) => <ellipse key={index} cx="40" cy="25" rx="6" ry="13"
        transform={`rotate(${index * 45} 40 40)`} fill="currentColor" fillOpacity=".08" />)}
      <circle cx="40" cy="40" r="8" fill="currentColor" fillOpacity=".2" />
      <circle cx="40" cy="40" r="3" fill="currentColor" stroke="none" />
    </> : category === "kavacham" ? <>
      <path d="M27 43h26l4 12c2 10-5 14-17 14s-19-4-17-14l4-12Z" fill="currentColor" fillOpacity=".12" />
      <path d="M25 39h30v5H25zM27 55h26M40 39c-12-8-10-18 0-26 10 8 12 18 0 26Z" fill="currentColor" fillOpacity=".12" />
      <path d="M40 39C26 36 18 31 17 24c12 0 20 5 23 15Zm0 0c14-3 22-8 23-15-12 0-20 5-23 15ZM40 48v13m-5-6h10" />
    </> : <>
      <text x="40" y="54" textAnchor="middle" fontFamily="Noto, 'Nirmala UI', sans-serif"
        fontSize="43" fill="currentColor" stroke="none">ॐ</text>
      <path d="M26 65h28" opacity=".5" />
    </>}
  </svg>;
}
