import { t, useLanguage } from "./i18n";
import "./brand-identity.css";

/** Brand renders the shared celestial M emblem and accessible wordmark with an optional page subtitle. */
export default function Brand({
  subtitle,
  inverse = false,
}: {
  subtitle?: string;
  inverse?: boolean;
}) {
  useLanguage();
  return (
    <>
      <img
        className="brand-mark"
        src={
          inverse ? "/images/brand/mark-reverse.svg" : "/images/brand/mark.svg"
        }
        width="48"
        height="48"
        alt=""
        aria-hidden="true"
      />
      <span className="brand-wordmark">
        <strong>
          Mishra <em>4</em> You
        </strong>
        <small>{subtitle ?? t("पं. राजा मिश्रा")}</small>
      </span>
    </>
  );
}
