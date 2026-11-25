import { useState } from "react";
import type { FormEvent } from "react";
import { api } from "./api";
import { t } from "./i18n";
import { PasswordFields } from "./LoginPage";
import { Button, ErrorBox } from "./UI";

/** AccountCredentials lets an authenticated phone-only customer retain their account when setting email/password login. */
export default function AccountCredentials({ csrf, refresh }: { csrf: string; refresh: () => Promise<unknown> }) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  /** submit attaches credentials through the guarded legacy-account endpoint without creating another account. */
  async function submit(event: FormEvent) {
    event.preventDefault(); setError("");
    if (password !== confirmation) { setError("दोनों पासवर्ड एक जैसे होने चाहिए।"); return; }
    setBusy(true);
    try {
      await api("/auth/credentials", "PUT", { email: email.trim(), password }, csrf);
      setPassword(""); setConfirmation(""); await refresh();
    } catch (cause) { setError((cause as Error).message); }
    finally { setBusy(false); }
  }
  return <section className="legacy-account-credentials auth-card" aria-labelledby="legacy-credentials-heading">
    <h2 id="legacy-credentials-heading">{t("अपने खाते का ईमेल और पासवर्ड सेट करें")}</h2>
    <p>{t("अगली बार इसी ईमेल और पासवर्ड से लॉग इन करें। आपकी सहेजी गई कुंडलियाँ और orders इसी खाते में रहेंगे।")}</p>
    <form onSubmit={submit} aria-busy={busy}>
      <PasswordFields email={email} password={password} confirmation={confirmation} creating busy={busy}
        onEmail={setEmail} onPassword={setPassword} onConfirmation={setConfirmation} />
      <ErrorBox error={error} /><Button type="submit" busy={busy}>{t("ईमेल और पासवर्ड सहेजें")}</Button>
    </form>
  </section>;
}
