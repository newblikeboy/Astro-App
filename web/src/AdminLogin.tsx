import { useState } from "react";
import type { FormEvent } from "react";
import { ArrowRight, CalendarDays, Eye, EyeOff, Gem, LockKeyhole, ShieldCheck, Wallet } from "lucide-react";
import { t } from "./i18n";

/** AdminLogin presents the separate email/password entry without exposing protected business data. */
export default function AdminLogin({ development, email, password, busy, error, onEmail, onPassword, onSubmit }: {
  development: boolean; email: string; password: string; busy: boolean; error: string;
  onEmail: (value: string) => void; onPassword: (value: string) => void;
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
            <label htmlFor="admin-email">{t("ईमेल पता")}</label>
            <input id="admin-email" name="email" type="email" autoComplete="username" autoCapitalize="none" spellCheck={false} required value={email} onChange={e => onEmail(e.target.value)} disabled={busy} />
            <label htmlFor="admin-password">{t("व्यवस्थापक पासवर्ड")}</label>
            <div className="admin-password-field">
              <input id="admin-password" name="password" type={visible ? "text" : "password"} autoComplete="current-password" required value={password} onChange={e => onPassword(e.target.value)} disabled={busy} />
              <button type="button" aria-label={t(visible ? "Hide password" : "Show password")} aria-pressed={visible} onClick={() => setVisible(!visible)}>
                {visible ? <EyeOff size={19} aria-hidden="true" /> : <Eye size={19} aria-hidden="true" />}
              </button>
            </div>
            {error && <p className="error" role="alert">{t(error)}</p>}
            <button className="button full" type="submit" disabled={busy}>{busy ? t("Signing in…") : t("व्यवस्थापन खोलें ")}<ArrowRight size={17} aria-hidden="true" /></button>
          </form>
          <p className="admin-login-security"><ShieldCheck size={16} aria-hidden="true" />{t("ग्राहक खाते से अलग, सीमित अवधि का प्रशासन सत्र।")}</p>
          {development && <details className="admin-login-help"><summary>{t("Local sign-in help")}</summary><p>{t("व्यवस्थापक ईमेल और पासवर्ड सर्वर की निजी सेटिंग में हैं। स्थानीय डेमो की जानकारी README में है।")}</p></details>}
        </section>
      </div>
    </div>
  );
}
