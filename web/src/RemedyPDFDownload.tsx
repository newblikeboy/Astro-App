import { useState } from "react";
import { Download, LoaderCircle } from "lucide-react";
import { getLanguage, t } from "./i18n";
import type { Language } from "./i18n";
import "./remedy-pdf.css";

/** RemedyPDFDownload lets a customer choose a PDF language independently of the website and reports preparation failures inline. */
export default function RemedyPDFDownload({ orderId }: { orderId: string }) {
  const [language, setLanguage] = useState<Language>(getLanguage());
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  /** download requests the owned purchased snapshot in the selected language and saves only a valid PDF response. */
  async function download() {
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      const response = await fetch(`/api/orders/${encodeURIComponent(orderId)}/pdf?language=${language}`, { cache: "no-store" });
      if (!response.ok) {
        const data = await response.json().catch(() => ({}));
        throw new Error(data.error || "The remedy PDF could not be downloaded. Please try again.");
      }
      if (!response.headers.get("Content-Type")?.startsWith("application/pdf")) {
        throw new Error("The remedy PDF could not be downloaded. Please try again.");
      }
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `Mishra4You-remedies-${language}.pdf`;
      document.body.appendChild(link);
      link.click();
      link.remove();
      setTimeout(() => URL.revokeObjectURL(url), 60_000);
    } catch (failure) {
      setError((failure as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return <div className="remedy-pdf-download">
    <label>
      <span>{t("PDF language")}</span>
      <select value={language} disabled={busy} onChange={(event) => { setLanguage(event.target.value as Language); setError(""); }}>
        <option value="en">English</option>
        <option value="hi">हिन्दी</option>
      </select>
    </label>
    <button className="button secondary" disabled={busy} onClick={() => void download()}>
      {busy ? <LoaderCircle size={17} className="spin" aria-hidden="true" /> : <Download size={17} aria-hidden="true" />}
      {t(busy ? "Preparing PDF…" : "PDF डाउनलोड")}
    </button>
    {error && <p className="remedy-pdf-error" role="alert">{t(error)}</p>}
  </div>;
}
