import source from "../../internal/platform/data/guruji-graha-signs.json";
import { t } from "./i18n";

/** GurujiGrahaStatus matches only the supplied board's exalted/debilitated sign numbers; other or unlisted placements stay unclassified. */
export default function GurujiGrahaStatus({ planet, sign }: { planet: string; sign: number }) {
  const record = source.records.find(entry => entry.planet === planet);
  const status = record?.exalted_signs.includes(sign) ? "exalted" : record?.debilitated_signs.includes(sign) ? "debilitated" : "unclassified";
  return <span className="graha-sign-status" data-status={status}>
    {status === "exalted" ? t("उच्च राशि") : status === "debilitated" ? t("नीच राशि") : "—"}
  </span>;
}
