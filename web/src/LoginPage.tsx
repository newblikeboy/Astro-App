import { useState } from "react";
import type { FormEvent } from "react";
import {
  ArrowRight,
  Info,
  ShieldCheck,
  BookOpen,
  CalendarDays,
  Sun,
} from "lucide-react";
import { api } from "./api";
import { t } from "./i18n";
import type { Config } from "./types";
import { Button, ErrorBox } from "./UI";

/** LoginPage renders the standalone customer authentication page with a safe local return destination. */
export default function LoginPage({
  config,
  onSuccess,
  onPolicy,
}: {
  config: Config;
  onSuccess: () => Promise<void>;
  onPolicy: (value: string) => void;
}) {
  return (
    <div className="auth-layout">
      <aside className="auth-story" aria-labelledby="auth-story-heading">
        <Sun size={40} strokeWidth={1} aria-hidden="true" />
        <h2 id="auth-story-heading">{t("Your Kundli, always with you.")}</h2>
        <p>{t("Keep your reports and consultations in one place.")}</p>
        <ul>
          <li>
            <BookOpen size={19} aria-hidden="true" />
            {t("Saved Kundlis & reports")}
          </li>
          <li>
            <CalendarDays size={19} aria-hidden="true" />
            {t("Your consultation bookings")}
          </li>
          <li>
            <ShieldCheck size={19} aria-hidden="true" />
            {t("Private access with mobile verification")}
          </li>
        </ul>
      </aside>
      <section className="auth-card" aria-labelledby="customer-login-heading">
        <span className="auth-card-icon">
          <ShieldCheck size={25} aria-hidden="true" />
        </span>
        <h1 id="customer-login-heading">{t("Log in to your account")}</h1>
        <LoginForm config={config} onSuccess={onSuccess} />
        <div className="auth-legal">
          <button onClick={() => onPolicy("privacy")}>{t("Privacy")}</button>
          <span aria-hidden="true">·</span>
          <button onClick={() => onPolicy("terms")}>
            {t("Terms & conditions")}
          </button>
        </div>
      </section>
    </div>
  );
}
/** LoginForm manages phone/OTP entry, challenge expiry and the successful-login callback. */
function LoginForm({
  config,
  onSuccess,
}: {
  config: Config;
  onSuccess: () => Promise<void>;
}) {
  const [phone, setPhone] = useState(""),
    [challenge, setChallenge] = useState(""),
    [code, setCode] = useState(""),
    [demo, setDemo] = useState(""),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  /** submit requests or verifies the OTP according to the current login step. */
  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      if (!challenge) {
        const v = await api<{
          challenge: string;
          demo_code?: string;
        }>("/auth/request", "POST", { phone });
        setChallenge(v.challenge);
        setDemo(v.demo_code || "");
      } else {
        await api("/auth/verify", "POST", { challenge, code });
        await onSuccess();
      }
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <form onSubmit={submit}>
      <p className="modal-intro">
        {t("आपकी कुंडली, रिपोर्ट और परामर्श एक सुरक्षित जगह पर।")}
      </p>
      {!challenge ? (
        <>
          <label htmlFor="phone">{t("मोबाइल नंबर")}</label>
          <input
            id="phone"
            type="tel"
            autoComplete="tel"
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            placeholder="+91 98765 43210"
            required
            autoFocus
          />
          <p className="field-note">
            {t("भारत के लिए 10 अंक, या देश कोड सहित नंबर लिखें।")}
          </p>
        </>
      ) : (
        <>
          <p>
            {phone}
            {t(" के लिए सत्यापन कोड")}{" "}
            <button
              className="small-link"
              type="button"
              onClick={() => {
                setChallenge("");
                setDemo("");
                setCode("");
              }}
            >
              {t("नंबर बदलें")}
            </button>
          </p>
          <label htmlFor="otp">{t("6 अंकों का OTP")}</label>
          <input
            id="otp"
            inputMode="numeric"
            pattern="[0-9]{6}"
            maxLength={6}
            autoComplete="one-time-code"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            required
            autoFocus
          />
          {demo && (
            <div className="notice">
              <Info size={18} />
              <span>
                {t("केवल स्थानीय डेमो \u00B7 OTP: ")}
                <b>{demo}</b> <br />
                {t("यह SMS नहीं भेजता। कोड 5 मिनट में समाप्त होगा।")}
              </span>
            </div>
          )}
        </>
      )}
      <ErrorBox error={t(error)} />
      <Button className="full" busy={busy}>
        {challenge ? t("सत्यापित करें") : t("OTP पाएँ")}{" "}
        <ArrowRight size={17} />
      </Button>
      <p className="field-note">
        {t("मोबाइल सत्यापन आपको WhatsApp प्रचार में शामिल नहीं करता।")}
      </p>
      {config.otp_simulation && !challenge && (
        <p className="muted">{t("अभी OTP का स्थानीय सिमुलेशन चालू है।")}</p>
      )}
    </form>
  );
}
