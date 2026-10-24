import { lazy, Suspense, useEffect, useState } from "react";
import { LoaderCircle } from "lucide-react";
import { api } from "./api";
import { t, useLanguage } from "./i18n";
import type { Config } from "./types";
import { PortalHeader, PortalFooter } from "./PortalShell";
const Admin = lazy(() => import("./Admin"));

/** AdminApp owns the separate administrator route, session state and protected workspace layout. */
export default function AdminApp() {
  useLanguage();
  const [config, setConfig] = useState<Config | null>(null);
  const [error, setError] = useState("");
  /** refresh reloads the administrator session state after login or another authentication transition. */
  const refresh = async () => {
    setError("");
    try {
      setConfig(await api<Config>("/config"));
    } catch (error) {
      setError((error as Error).message);
    }
  };
  useEffect(() => {
    void refresh();
  }, []);
  return (
    <div className="admin-app">
      <a href="#main" className="skip">
        {t("मुख्य सामग्री पर जाएँ")}
      </a>
      <PortalHeader admin />
      <main id="main">
        {error && !config ? (
          <div className="portal-status" role="alert">
            <h1>{t("Administration")}</h1>
            <p>{t(error)}</p>
            <button className="button" onClick={() => void refresh()}>
              {t("Try again")}
            </button>
          </div>
        ) : config ? (
          <Suspense
            fallback={
              <div className="portal-status" role="status">
                <LoaderCircle className="spin" size={24} />
                <p>{t("Opening your workspace…")}</p>
              </div>
            }
          >
            <Admin config={config} onConfig={refresh} />
          </Suspense>
        ) : (
          <div className="portal-status" role="status">
            <LoaderCircle className="spin" size={24} />
            <p>{t("Opening your workspace…")}</p>
          </div>
        )}
      </main>
      <PortalFooter />
    </div>
  );
}
