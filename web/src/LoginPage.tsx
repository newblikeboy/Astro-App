import { useState } from "react";
import type { FormEvent } from "react";
import { ArrowRight, ShieldCheck, BookOpen, CalendarDays, Sun, Eye, EyeOff } from "lucide-react";
import { api } from "./api";
import { t } from "./i18n";
import { Button, ErrorBox } from "./UI";

/** PasswordFields renders labelled credentials with password-manager hints and an accessible visibility control. */
export function PasswordFields({ email, password, confirmation, creating, busy, onEmail, onPassword, onConfirmation }: {
  email: string; password: string; confirmation: string; creating: boolean; busy: boolean;
  onEmail: (value: string) => void; onPassword: (value: string) => void; onConfirmation: (value: string) => void;
}) {
  const [visible, setVisible] = useState(false);
  return <fieldset className="customer-credentials" disabled={busy}>
    <label htmlFor="customer-email">{t("ईमेल पता")}</label>
    <input id="customer-email" name="email" type="email" autoComplete="username" inputMode="email"
      autoCapitalize="none" spellCheck={false} maxLength={254} required value={email} onChange={e => onEmail(e.target.value)} />
    <label htmlFor="customer-password">{t("पासवर्ड")}</label>
    <div className="customer-password-field">
      <input id="customer-password" name="password" type={visible ? "text" : "password"}
        autoComplete={creating ? "new-password" : "current-password"} minLength={creating ? 8 : undefined}
        required value={password} onChange={e => onPassword(e.target.value)} aria-describedby={creating ? "customer-password-note" : undefined} />
      <button type="button" className="icon-button" aria-label={t(visible ? "Hide password" : "Show password")}
        aria-pressed={visible} onClick={() => setVisible(!visible)}>{visible ? <EyeOff size={19} /> : <Eye size={19} />}</button>
    </div>
    {creating && <>
      <p id="customer-password-note" className="field-note">{t("कम से कम 8 अक्षर का पासवर्ड रखें।")}</p>
      <label htmlFor="customer-confirm-password">{t("पासवर्ड दोबारा लिखें")}</label>
      <input id="customer-confirm-password" name="confirm-password" type={visible ? "text" : "password"}
        autoComplete="new-password" required value={confirmation} onChange={e => onConfirmation(e.target.value)} />
    </>}
  </fieldset>;
}

/** LoginPage offers email/password sign-in or registration while preserving the caller's pending action. */
export default function LoginPage({ onSuccess, onPolicy }: {
  onSuccess: () => Promise<void>; onPolicy: (value: string) => void;
}) {
  const [creating, setCreating] = useState(false);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  /** submit creates or verifies credentials before continuing the original login intent. */
  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    if (creating && password !== confirmation) { setError("दोनों पासवर्ड एक जैसे होने चाहिए।"); return; }
    if (new TextEncoder().encode(password).length > 72) { setError("पासवर्ड बहुत लंबा है। कृपया छोटा पासवर्ड रखें।"); return; }
    setBusy(true);
    try {
      await api(creating ? "/auth/register" : "/auth/login", "POST", { email: email.trim(), password });
      setPassword(""); setConfirmation("");
      await onSuccess();
    } catch (cause) { setError((cause as Error).message); }
    finally { setBusy(false); }
  }
  return <div className="auth-layout">
    <aside className="auth-story" aria-labelledby="auth-story-heading">
      <Sun size={40} strokeWidth={1} aria-hidden="true" />
      <h2 id="auth-story-heading">{t("Your Kundli, always with you.")}</h2>
      <p>{t("Keep your reports and consultations in one place.")}</p>
      <ul>
        <li><BookOpen size={19} aria-hidden="true" />{t("Saved Kundlis & reports")}</li>
        <li><CalendarDays size={19} aria-hidden="true" />{t("Your consultation bookings")}</li>
        <li><ShieldCheck size={19} aria-hidden="true" />{t("ईमेल और पासवर्ड से सुरक्षित लॉग इन")}</li>
      </ul>
    </aside>
    <section className="auth-card" aria-labelledby="customer-login-heading">
      <span className="auth-card-icon"><ShieldCheck size={25} aria-hidden="true" /></span>
      <h1 id="customer-login-heading">{t(creating ? "नया खाता बनाएँ" : "Log in to your account")}</h1>
      <form onSubmit={submit} aria-busy={busy}>
        <p className="modal-intro">{t("आपकी कुंडली, रिपोर्ट और परामर्श एक सुरक्षित जगह पर।")}</p>
        <PasswordFields email={email} password={password} confirmation={confirmation} creating={creating} busy={busy}
          onEmail={setEmail} onPassword={setPassword} onConfirmation={setConfirmation} />
        <ErrorBox error={error} />
        <Button className="full" type="submit" busy={busy}>{t(creating ? "खाता बनाएँ" : "लॉग इन")} <ArrowRight size={17} /></Button>
      </form>
      <p className="auth-switch">{t(creating ? "पहले से खाता है?" : "पहली बार आए हैं?")} <button type="button" className="text-button" disabled={busy}
        onClick={() => { setCreating(!creating); setPassword(""); setConfirmation(""); setError(""); }}>
        {t(creating ? "लॉग इन करें" : "नया खाता बनाएँ")}</button></p>
      <div className="auth-legal"><button onClick={() => onPolicy("privacy")}>{t("Privacy")}</button><span aria-hidden="true">·</span>
        <button onClick={() => onPolicy("terms")}>{t("Terms & conditions")}</button></div>
    </section>
  </div>;
}
