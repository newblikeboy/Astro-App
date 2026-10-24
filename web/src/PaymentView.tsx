import { useState } from "react";
import { CheckCircle2, FileText, Gem, Info, LockKeyhole } from "lucide-react";
import { api, money } from "./api";
import { t } from "./i18n";
import { Button, ErrorBox } from "./UI";
import type { Order, Config } from "./types";
declare global {
  interface Window {
    Razorpay: new (options: Record<string, unknown>) => {
      open: () => void;
      on: (name: string, cb: () => void) => void;
    };
  }
}
/** PaymentView renders the order payment controls and keeps final confirmation on the server. */
export default function PaymentView({
  order,
  config,
  csrf,
  onPaid,
}: {
  order: Order;
  config: Config;
  csrf: string;
  onPaid: () => Promise<void>;
}) {
  const [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  /** simulate explicitly confirms a successful/failed development payment through the simulation-only endpoint. */
  async function simulate(success: boolean) {
    setBusy(true);
    try {
      await api(`/orders/${order.id}/simulate`, "POST", { success }, csrf);
      if (success) await onPaid();
      else {
        setError(
          t(
            "डेमो भुगतान असफल दर्ज हुआ। आप इसी ऑर्डर पर फिर प्रयास कर सकते हैं।",
          ),
        );
        setBusy(false);
      }
    } catch (e) {
      setError((e as Error).message);
      setBusy(false);
    }
  }
  /** real loads Razorpay checkout and wires signed provider results to server verification. */
  async function real() {
    setBusy(true);
    setError("");
    try {
      if (!window.Razorpay)
        await new Promise<void>((resolve, reject) => {
          const s = document.createElement("script");
          s.src = "https://checkout.razorpay.com/v1/checkout.js";
          s.onload = () => resolve();
          s.onerror = () =>
            reject(new Error(t("भुगतान सेवा लोड नहीं हो पाई।")));
          document.body.appendChild(s);
        });
      const rzp = new window.Razorpay({
        key: config.payment_key,
        order_id: order.provider_id,
        amount: order.amount,
        currency: "INR",
        name: "Mishra 4 You",
        description: t(order.title),
        theme: { color: "#B44B00" },
        /** handler submits the Razorpay payment ID/signature for server verification before reporting success. */
        handler: async (v: {
          razorpay_payment_id: string;
          razorpay_signature: string;
        }) => {
          try {
            await api(
              `/orders/${order.id}/verify`,
              "POST",
              {
                payment_id: v.razorpay_payment_id,
                signature: v.razorpay_signature,
              },
              csrf,
            );
            await onPaid();
          } catch (e) {
            setError((e as Error).message);
            setBusy(false);
          }
        },
        modal: {
          /** ondismiss clears the payment busy state when the provider checkout dialog is dismissed. */
          ondismiss: () => setBusy(false),
        },
      });
      rzp.on("payment.failed", () => {
        setBusy(false);
        setError(
          t("भुगतान पूरा नहीं हुआ। स्थिति खाते में देखें और फिर प्रयास करें।"),
        );
      });
      rzp.open();
    } catch (e) {
      setError((e as Error).message);
      setBusy(false);
    }
  }
  return (
    <>
      <div className="checkout-summary">
        {order.product === "gemstone" ? (
          <Gem size={26} />
        ) : (
          <FileText size={26} />
        )}
        <div>
          <h3>{t(order.title)}</h3>
          <small className="reference">{order.id}</small>
        </div>
        <strong>{money(order.amount)}</strong>
      </div>
      {config.simulation ? (
        <>
          <div className="notice">
            <Info size={18} />
            <span>
              {t(
                "यह पूरी तरह स्थानीय डेमो है। कोई बैंक, UPI या कार्ड लेनदेन नहीं होगा।",
              )}
            </span>
          </div>
          <Button
            className="full"
            busy={busy}
            onClick={() => void simulate(true)}
          >
            {t("डेमो भुगतान सफल करें ")}
            <CheckCircle2 size={18} />
          </Button>
          <button
            className="small-link space-top"
            disabled={busy}
            onClick={() => void simulate(false)}
          >
            {t("असफल भुगतान का परीक्षण करें")}
          </button>
        </>
      ) : (
        <Button className="full" busy={busy} onClick={() => void real()}>
          {t("Razorpay में भुगतान करें ")}
          <LockKeyhole size={17} />
        </Button>
      )}
      <ErrorBox error={t(error)} />
      <p className="muted">
        {order.product === "gemstone"
          ? t(
              "Your payment is confirmed on the server before your stone order is processed.",
            )
          : t(
              "केवल सर्वर द्वारा सत्यापित भुगतान के बाद रिपोर्ट या बैठक तैयार होगी।",
            )}
      </p>
    </>
  );
}
