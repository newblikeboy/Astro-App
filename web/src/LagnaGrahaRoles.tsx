import roles from "../../internal/platform/data/guruji-lagna-roles.json";

/** LagnaGrahaRoles prints the handwritten role lists for the natal ascendant, preserving Hindi and empty source lists. */
export default function LagnaGrahaRoles({ natalLagna }: { natalLagna: number }) {
  const entry = roles.records.find((record) => record.lagna === natalLagna);
  if (!entry) return null;
  return (
    <section className="lagna-graha-roles" lang="hi" aria-label={`${entry.title} · कारक, मारक, सम`} data-lagna={entry.lagna}>
      <h3>{entry.title}</h3>
      <dl>
        <div><dt>{entry.karak_label}</dt><dd>{entry.karak.join(", ") || "—"}</dd></div>
        <div><dt>मारक</dt><dd>{entry.marak.join(", ") || "—"}</dd></div>
        <div><dt>सम</dt><dd>{entry.sam.join(", ") || "—"}</dd></div>
      </dl>
    </section>
  );
}
