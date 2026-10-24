# Working on Mishra 4 You

Start with the [codebase map](docs/CODEBASE.md), [function index](docs/FUNCTION_INDEX.md), [routes](docs/ROUTES.md) and [calculation contract](docs/CALCULATIONS.md). Keep changes within the owning module so a future maintainer can find the same feature in one predictable place.

## Function documentation

- Add a purpose comment to every new named function, method and meaningful named local helper. Include tests, scripts and React components.
- Explain what the function accomplishes, its meaningful side effects, and any rule a caller must understand. Describe units, boundaries or error behavior when these affect correctness.
- Keep existing reasoning comments when changing an implementation. Update a comment whenever its behavior changes.
- Use ordinary English. Describe the domain behavior rather than paraphrasing each statement or adding an empty phrase such as "handles this function."
- Inline callbacks belong to their enclosing documented function. Add an inline explanation when a callback contains an independent business rule; do not repeat comments on every trivial `map` or `onClick` expression.
- Do not edit generated `web/dist/`, installed dependencies, reference snapshots or bundled Swiss C code merely to satisfy documentation coverage.

Examples:

```go
// available rejects buffered overlap with active bookings and unexpired holds.
// exclude identifies the existing booking when checking a reschedule.
func available(...) bool {
    // implementation
}
```

```ts
/** checkout sends product IDs and quantities; the server determines prices and stock. */
async function checkout() {
  // implementation
}
```

```python
def unwrap(values):
    """Make consecutive angular samples continuous across the zero-degree wrap."""
    # implementation
```

After changing comments or adding/removing functions, run `node scripts/code-docs.cjs --write`. Run `node scripts/code-docs.cjs --check` before review. The check fails when a named function lacks documentation or the checked-in index is stale. Review the generated index for meaningful descriptions; the check verifies coverage, not the truth of prose.

## Module boundaries

- Put native astronomy and declared astrological conventions in `internal/swisseph/`.
- Put API validation, ownership checks, business transitions, database transactions and provider orchestration in `internal/platform/`.
- Put page interaction and presentation in `web/src/`. Backend prices, stock, entitlements and payment verification stay authoritative.
- Keep the public site, customer login/account, shop and administrator layouts separated by their existing routes.
- Use the shared translation, routing, money and minute-display helpers rather than creating divergent copies.
- Never put secrets in comments, examples, generated indexes or logs. Use the existing environment configuration without committing a real `.env`.

## Rules worth preserving

- Keep money in integer paise and gemstone weight in integer milli-carats. Avoid floating-point conversions for persisted financial totals.
- Preserve precise stored/native instants. Format visitor-facing clocks to hours/minutes using an explicit timezone.
- Resolve birth times in the trusted birthplace's IANA timezone and retain the existing ambiguity/nonexistent-time checks.
- Keep shared Swiss C state behind `engineMu`. Preserve explicit ephemeris flags, data-file checks and failure handling.
- Keep approved interpretation versions and purchased PDF snapshots immutable.
- Preserve transactional stock/slot reservations and idempotent payment/provider/job processing.
- Update independent numerical fixtures deliberately when a documented calculation convention changes. Do not regenerate reference values from the implementation being tested or shift calculations merely to match another website.

## Validation

Run the checks appropriate to the changed module. For a complete local check from the project root:

```powershell
node scripts/code-docs.cjs --check
$env:GOMAXPROCS='2'
go test -p 1 ./...
go vet -p 1 ./...
npm --prefix web run build
```

The Go tests that require a separately configured database or real browser document their prerequisites and may skip without them. Use the relevant read-only browser verification script when changing a user flow or layout. Scripts that create isolated test orders must target their dedicated QA environment, not the running customer database; inspect each script's header and configuration before running it.

For comments-only changes, language-parser/token comparisons can establish that executable code did not change; syntax/build checks still matter. Keep [operations](docs/OPERATIONS.md), [routes](docs/ROUTES.md), and the [codebase map](docs/CODEBASE.md) current when their corresponding behavior or ownership changes.
