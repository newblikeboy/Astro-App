import { ArrowLeft, ShieldCheck } from "lucide-react";
import LanguageSelector from "./LanguageSelector";
import Brand from "./Brand";
import CartLink from "./CartLink";
import { t } from "./i18n";
import { followRoute } from "./routing";
import "./portal.css";

/** PortalHeader renders standalone-page branding, language selection, customer cart and separate admin navigation. */
export function PortalHeader({ admin = false }: { admin?: boolean }) {
  return (
    <header className={`portal-header ${admin ? "portal-header-admin" : ""}`}>
      <a
        className="brand"
        href="/"
        onClick={(event) => followRoute(event, "/")}
      >
        <Brand subtitle={admin ? t("Administration") : undefined} />
      </a>
      <div className="portal-header-actions">
        {admin && (
          <span className="portal-admin-label">
            <ShieldCheck size={15} aria-hidden="true" />
            {t("Administration")}
          </span>
        )}
        <LanguageSelector />
        {!admin && <CartLink />}
        <a
          className="portal-back"
          href="/"
          onClick={(event) => followRoute(event, "/")}
        >
          <ArrowLeft size={15} aria-hidden="true" />
          <span>{t("Back to website")}</span>
        </a>
      </div>
    </header>
  );
}
/** PortalFooter renders the compact footer used outside the public landing-page layout. */
export function PortalFooter() {
  return (
    <footer className="portal-footer">
      © {new Date().getFullYear()} Mishra 4 You
    </footer>
  );
}
