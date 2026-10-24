const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");
(async () => {
  const { formatPaise, parseRupees, rupeeInput } = await import(
    pathToFileURL(path.resolve("web/src/money.ts")).href
  );
  for (const [p, display] of [
    [0, "₹0"],
    [1, "₹0.01"],
    [-1, "-₹0.01"],
    [100, "₹1"],
    [101, "₹1.01"],
    [Number.MAX_SAFE_INTEGER, "₹9,00,71,99,25,47,409.91"],
  ]) {
    assert.equal(formatPaise(p), display);
  }
  for (const p of [0, 1, 101, 9903, Number.MAX_SAFE_INTEGER])
    assert.equal(parseRupees(rupeeInput(p)), p);
  for (const value of [
    "",
    "-1",
    "1.001",
    "Infinity",
    "1e2",
    "90071992547409.92",
  ])
    assert.equal(parseRupees(value), null);
  for (const p of [NaN, Infinity, 0.1, Number.MAX_SAFE_INTEGER + 1])
    assert.throws(() => formatPaise(p));
  console.log(
    "Exact paise formatting, input round trips, and invalid/unsafe amount checks passed.",
  );
})().catch((e) => {
  console.error(e);
  process.exitCode = 1;
});
