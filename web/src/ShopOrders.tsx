import { Gem, Package, Truck, ExternalLink } from "lucide-react";
import { t } from "./i18n";
import { localTime, money } from "./api";
import type { Order } from "./types";
import { carats } from "./shop-types";

/** fulfilmentName returns the localized customer-facing name of an order fulfilment status. */
export const fulfilmentName = (status: string) =>
  t(
    (
      {
        awaiting_payment: "Awaiting payment",
        processing: "Preparing your order",
        shipped: "Dispatched",
        delivered: "Delivered",
        needs_resolution: "Seller review required",
        payment_expired: "Reservation expired",
        refunded: "Refunded",
        return_review: "Return under review",
        cancelled: "Cancelled",
      } as Record<string, string>
    )[status] || status,
  );

/** ShopOrders renders owned gemstone purchases with payment, dispatch and tracking information. */
export default function ShopOrders({
  orders,
  onResume,
  onCancel,
}: {
  orders: Order[];
  onResume?: (order: Order) => void;
  onCancel?: (order: Order) => void;
}) {
  const stones = orders.filter((o) => o.gemstone);
  return (
    <div className="shop-orders">
      {stones.length === 0 && (
        <div className="shop-empty">
          <Package size={38} aria-hidden="true" />
          <h2>{t("Your collection starts here")}</h2>
          <p>
            {t("Your gemstone orders and delivery updates will appear here.")}
          </p>
        </div>
      )}
      {stones.map((o) => {
        const g = o.gemstone!;
        const pending = !["paid", "refunded", "cancelled", "expired"].includes(
          o.status,
        );
        return (
          <article className="shop-order" key={o.id}>
            <header>
              <div>
                <small>{o.id}</small>
                <h2>{fulfilmentName(g.fulfilment)}</h2>
                <span className="muted">{localTime(o.created)}</span>
              </div>
              <div>
                <strong>{money(o.amount)}</strong>
                <span className="shop-badge">
                  {o.demo ? t("Test order") : t(o.status)}
                </span>
              </div>
            </header>
            <ul>
              {g.items.map((line) => (
                <li key={line.product_id}>
                  <Gem size={21} aria-hidden="true" />
                  <div>
                    <strong>{line.product.name}</strong>
                    <small>
                      {carats(line.product.weight_milli_carat)} {t("ct")} ·{" "}
                      {t("Quantity")} {line.quantity}
                    </small>
                  </div>
                  <span>{money(line.product.price * line.quantity)}</span>
                </li>
              ))}
            </ul>
            {g.courier && (
              <p className="shop-tracking">
                <Truck size={19} aria-hidden="true" />
                <span>
                  {g.courier} · {g.tracking_number}
                </span>
                {g.tracking_url && (
                  <a href={g.tracking_url} target="_blank" rel="noreferrer">
                    {t("Track delivery")} <ExternalLink size={14} />
                  </a>
                )}
              </p>
            )}
            {o.refunded > 0 && (
              <p>
                {t("Refund recorded")}: {money(o.refunded)}
              </p>
            )}
            {pending && (
              <p className="muted">
                {t("Reservation until")}: {localTime(g.hold_until)}
              </p>
            )}
            <div className="inline-actions">
              {pending &&
                o.provider_id &&
                Date.now() < Date.parse(g.hold_until) &&
                onResume && (
                  <button className="button" onClick={() => onResume(o)}>
                    {t("Continue payment")}
                  </button>
                )}
              {pending && onCancel && (
                <button
                  className="button secondary"
                  onClick={() => onCancel(o)}
                >
                  {t("Cancel reservation")}
                </button>
              )}
            </div>
            <details>
              <summary>{t("Delivery and order details")}</summary>
              <div className="shop-order-detail">
                <p>
                  {g.address.name}
                  <br />
                  {g.address.line1} {g.address.line2}
                  <br />
                  {g.address.city}, {g.address.state} {g.address.pin}
                  <br />
                  {g.address.phone}
                </p>
                <div>
                  <p>
                    {t("Stones")}: {money(g.subtotal)}
                    <br />
                    {t("Delivery")}: {money(g.shipping)}
                  </p>
                  <h3>{t("Delivery policy")}</h3>
                  <p>{g.delivery_policy}</p>
                  <h3>{t("Return policy")}</h3>
                  <p>{g.return_policy}</p>
                </div>
              </div>
            </details>
          </article>
        );
      })}
    </div>
  );
}
