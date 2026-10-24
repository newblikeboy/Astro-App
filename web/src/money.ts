// Preserve paise before formatting/decoding decimal rupees. Dividing a large
// safe integer by 100 as a floating-point number can lose its last paisa.
export function formatPaise(paise: number): string {
  if (!Number.isSafeInteger(paise)) throw new RangeError("Invalid paise");
  const absolute = BigInt(paise < 0 ? -paise : paise);
  const whole = absolute / 100n;
  const cents = absolute % 100n;
  const signed = paise < 0 ? (whole === 0n ? -0 : -whole) : whole;
  return new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency: "INR",
    minimumFractionDigits: cents ? 2 : 0,
    maximumFractionDigits: cents ? 2 : 0,
  })
    .formatToParts(signed)
    .map((part) =>
      part.type === "fraction" ? cents.toString().padStart(2, "0") : part.value,
    )
    .join("");
}

/** parseRupees parses a bounded rupee input string into exact integer paise without floating-point multiplication. */
export function parseRupees(value: string): number | null {
  const match = /^(\d+)(?:\.(\d{1,2}))?$/.exec(value.trim());
  if (!match) return null;
  const paise =
    BigInt(match[1]) * 100n + BigInt((match[2] || "").padEnd(2, "0"));
  return paise <= BigInt(Number.MAX_SAFE_INTEGER) ? Number(paise) : null;
}

/** rupeeInput converts stored integer paise to an exact rupee editor string. */
export function rupeeInput(paise: number): string {
  if (!Number.isSafeInteger(paise) || paise < 0) return "";
  const amount = BigInt(paise);
  return `${amount / 100n}.${(amount % 100n).toString().padStart(2, "0")}`;
}
