import { useState } from "react";
import type { FormEvent } from "react";
import { ArrowRight, CalendarDays, Eye, EyeOff, Gem, LockKeyhole, ShieldCheck, Wallet } from "lucide-react";
import { t } from "./i18n";

/** AdminLogin presents the separate password/TOTP entry without exposing protected business data. */
export default function AdminLogin({ development, password, totp, busy, error, onPassword, onTotp, onSubmit }: {
  development: boolean; password: string; totp: string; busy: boolean; error: string;
  onPassword: (value: string) => void; onTotp: (value: string) => void;
  onSubmit: (event: FormEvent) => Promise<void>;
}) {
  const [visible, setVisible] = useState(false);
  return (
    <div className="admin-entry">
      <div className="admin-login">
        <section className="admin-login-story" aria-labelledby="admin-story-title">
          <span className="admin-entry-label"><ShieldCheck size={16} aria-hidden="true" />{t("Business workspace")}</span>
          <h2 id="admin-story-title">{t("Your business.\nOne clear view.")}</h2>
          <p>{t("Everything you need to look after your customers and grow Mishra 4 You.")}</p>
          <ul>
            {[
              [Gem, "Gemstones & orders", "Manage prices, stock and deliveries."],
              [CalendarDays, "Consultations & reports", "Plan bookings and follow up on delivery."],
              [Wallet, "Payments & accounts", "Review collections, refunds and expenses."],
            ].map(([Icon, title, description]) => {
              const Symbol = Icon as typeof Gem;
              return <li key={String(title)}><Symbol size={21} aria-hidden="true" /><div><strong>{t(String(title))}</strong><span>{t(String(description))}</span></div></li>;
            })}
          </ul>
          <span className="admin-entry-foot">Mishra 4 You <span aria-hidden="true">·</span> {t("Administration")}</span>
        </section>
        <section className="admin-login-form" aria-labelledby="admin-login-title">
          <span className="admin-login-lock"><LockKeyhole size={24} aria-hidden="true" /></span>
          <div className="eyebrow">{t("Welcome back")}</div>
          <h1 id="admin-login-title">{t("संचालक लॉग इन")}</h1>
          <p>{t("Sign in to manage your business.")}</p>
          <form onSubmit={onSubmit} aria-busy={busy}>
            <label htmlFor="admin-password">{t("व्यवस्थापक पासवर्ड")}</label>
            <div className="admin-password-field">
              <input id="admin-password" type={visible ? "text" : "password"} autoComplete="current-password" required value={password} onChange={e => onPassword(e.target.value)} disabled={busy} />
              <button type="button" aria-label={t(visible ? "Hide password" : "Show password")} aria-pressed={visible} onClick={() => setVisible(!visible)}>
                {visible ? <EyeOff size={19} aria-hidden="true" /> : <Eye size={19} aria-hidden="true" />}
              </button>
            </div>
            <label htmlFor="admin-totp">{t("Authenticator code")}{development && <span className="admin-optional">{t("Optional locally")}</span>}</label>
            <input id="admin-totp" type="text" inputMode="numeric" autoComplete="one-time-code" pattern="[0-9]{6}" maxLength={6} value={totp} onChange={e => onTotp(e.target.value)} required={!development} disabled={busy} aria-describedby="admin-code-help" />
            <p id="admin-code-help" className="field-note">{t("Enter the six-digit code from your authenticator app.")}</p>
            {error && <p className="error" role="alert">{t(error)}</p>}
            <button className="button full" type="submit" disabled={busy}>{busy ? t("Signing in…") : t("व्यवस्थापन खोलें ")}<ArrowRight size={17} aria-hidden="true" /></button>
          </form>
          <p className="admin-login-security"><ShieldCheck size={16} aria-hidden="true" />{t("ग्राहक खाते से अलग, सीमित अवधि का प्रशासन सत्र।")}</p>
          {development && <details className="admin-login-help"><summary>{t("Local sign-in help")}</summary><p>{t("स्थानीय डेमो का पासवर्ड README में है। उत्पादन में अलग, मज़बूत पासवर्ड और TOTP अनिवार्य हैं।")}</p></details>}
        </section>
      </div>
    </div>
  );
}
