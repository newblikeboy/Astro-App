// SPDX-License-Identifier: AGPL-3.0-only
import { t } from "./i18n";
import "./source-links.css";

const repository = "https://github.com/newblikeboy/Astro-App";
const revision = __APP_SOURCE_REVISION__;

/** SourceLinks offers the running build's corresponding source, licence and third-party notices. */
export default function SourceLinks() {
  return (
    <span className="source-links">
      <a href={`${repository}/tree/${revision}`}>{t("स्रोत कोड")}</a>
      <span aria-hidden="true">·</span>
      <a href={`${repository}/blob/${revision}/LICENSE`}>AGPL-3.0</a>
      <span aria-hidden="true">·</span>
      <a href={`${repository}/blob/${revision}/THIRD-PARTY-NOTICES.md`}>
        {t("तृतीय-पक्ष सूचनाएँ")}
      </a>
      <span className="source-warranty">{t("सॉफ़्टवेयर बिना वारंटी के उपलब्ध है।")}</span>
    </span>
  );
}
