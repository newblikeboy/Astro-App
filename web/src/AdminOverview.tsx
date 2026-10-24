import { useEffect, useState } from "react";
import { ArrowRight, CalendarDays, Check, CircleAlert, Gem, PackageCheck, RefreshCw, Users, Wallet } from "lucide-react";
import { api, localTime, money } from "./api";
import { t } from "./i18n";
import type { AdminData, Config, Order } from "./types";
import type { Gemstone, ShopSettings } from "./shop-types";

type Stock = { products: Gemstone[]; settings: ShopSettings; orders: Order[] };

/** AdminOverview turns protected records into actionable queues; monthly money uses server totals and excludes simulations by default. */
export default function AdminOverview({data, config, month, includeTestFinance, onOpen}: {
  data: AdminData; config: Config; month: string; includeTestFinance: boolean; onOpen: (id: string, shopTab?: string) => void;
}) {
  const [stock, setStock] = useState<Stock | null>(null);
  const [error, setError] = useState(false);
  const [retry, setRetry] = useState(0);
  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    let current = true;
    setStock(null); setError(false);
    api<Stock>("/admin/shop").then(value => { if (current) setStock(value); }).catch(() => { if (current) setError(true); });
    return () => { current = false; };
  }, [data, retry]);
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 60_000);
    return () => clearInterval(timer);
  }, []);
  const actualOrders = data.orders.filter(order => !order.demo);
  const dispatch = actualOrders.filter(order => order.status === "paid" && order.refunded === 0 && order.gemstone && !["dispatched", "delivered"].includes(order.gemstone.fulfilment));
  const upcoming = Object.values(data.bookings).filter(booking => !booking.demo && ["confirmed", "meeting_pending"].includes(booking.status) && Date.parse(booking.end) > now).sort((a, b) => a.start.localeCompare(b.start));
  const jobs = Object.values(data.jobs).filter(job => ["retry", "failed"].includes(job.status));
  const low = stock?.products.filter(product => !product.sample && product.status === "published" && product.available <= 2) || [];
  const pendingStock = stock?.products.filter(product => product.sample).length || 0;
  const recent = actualOrders.slice().sort((a, b) => b.created.localeCompare(a.created)).slice(0, 5);
  return (
    <div className="admin-overview">
      <div className="admin-overview-context"><span>{t("Order and booking summaries exclude test records.")}{includeTestFinance && <> {t("Monthly finance includes test orders.")}</>}</span><span>{t("Finance month")}: <b>{data.finance.month || month}</b></span></div>
      <div className="business-metrics">
        {[
          {label: "Gross collections", value: data.finance.calculation_error ? "—" : money(data.finance.gross), note: "Selected month · before refunds and expenses", icon: Wallet, target: "finance"},
          {label: "Awaiting dispatch", value: String(dispatch.length), note: "Paid stone orders to review", icon: PackageCheck, target: "shop", shopTab: "orders"},
          {label: "Upcoming consultations", value: String(upcoming.length), note: "Confirmed or meeting pending", icon: CalendarDays, target: "calendar"},
          {label: "Customer accounts", value: String(data.users.filter(user => !user.deleted).length), note: "Active accounts · all time", icon: Users, target: "users"},
        ].map(metric => <button className="business-metric" key={metric.label} onClick={() => onOpen(metric.target, metric.shopTab)}><span><metric.icon size={19} aria-hidden="true" />{t(metric.label)}</span><strong>{metric.value}</strong><small>{t(metric.note)}</small><ArrowRight size={16} aria-hidden="true" /></button>)}
      </div>
      <div className="admin-overview-columns">
        <section className="business-panel admin-attention">
          <header><div><span className="admin-kicker">{t("Your next steps")}</span><h2>{t("Needs attention")}</h2></div><CircleAlert size={23} aria-hidden="true" /></header>
          <button className="attention-row" onClick={() => onOpen("shop", "orders")}><span className="attention-count">{dispatch.length}</span><span><strong>{t("Orders to dispatch")}</strong><small>{t("Review paid orders and add courier tracking.")}</small></span><ArrowRight size={16} aria-hidden="true" /></button>
          <button className="attention-row" onClick={() => onOpen("jobs")}><span className="attention-count">{jobs.length}</span><span><strong>{t("Delivery jobs to review")}</strong><small>{t("Check failed reports, meetings or notifications.")}</small></span><ArrowRight size={16} aria-hidden="true" /></button>
          {stock ? <>
            <button className="attention-row" onClick={() => onOpen("shop")}><span className="attention-count">{low.length}</span><span><strong>{t("Low-stock products")}</strong><small>{t("Published inventory with two or fewer pieces available.")}</small></span><ArrowRight size={16} aria-hidden="true" /></button>
            {pendingStock > 0 && <button className="attention-row" onClick={() => onOpen("shop")}><span className="attention-count">{pendingStock}</span><span><strong>{t("Stock confirmation pending")}</strong><small>{t("Confirm actual pieces before making them available to buy.")}</small></span><ArrowRight size={16} aria-hidden="true" /></button>}
          </> : error ? <div className="admin-stock-error" role="alert"><p>{t("Stock information could not be loaded.")}</p><button className="small-link" onClick={() => setRetry(value => value + 1)}><RefreshCw size={14} aria-hidden="true" />{t("Try again")}</button></div> : <p role="status">{t("Loading stock information…")}</p>}
        </section>
        <section className="business-panel">
          <header><div><span className="admin-kicker">{t("Business controls")}</span><h2>{t("Quick actions")}</h2></div><Gem size={23} aria-hidden="true" /></header>
          <div className="admin-quick-actions">
            {[["shop", "Manage products & prices"], ["calendar", "Manage consultations"], ["content", "Edit website content"], ["finance", "Record an expense"]].map(([id, label]) => <button key={id} onClick={() => onOpen(id)}><span>{t(label)}</span><ArrowRight size={16} aria-hidden="true" /></button>)}
          </div>
          <div className="admin-provider-state"><ShieldStatus enabled={!config.simulation} label={config.simulation ? "Payments in test mode" : "Live payment mode configured"} /><ShieldStatus enabled={!!stock?.settings.enabled} label={!stock ? "Shop status unavailable" : stock.settings.enabled ? "Shop checkout enabled" : "Shop checkout paused"} /><p>{t("Payment mode is configured on the server. Use service and shop settings to review selling availability.")}</p><button className="small-link" onClick={() => onOpen("settings")}>{t("Review service settings")}<ArrowRight size={14} aria-hidden="true" /></button></div>
        </section>
      </div>
      <div className="admin-overview-columns">
        <section className="business-panel">
          <header><h2>{t("Recent orders")}</h2><button className="small-link" onClick={() => onOpen("orders")}>{t("View all")}<ArrowRight size={14} aria-hidden="true" /></button></header>
          {recent.length ? <ul className="admin-record-list">{recent.map(order => <li key={order.id}><div><strong>{t(order.title)}</strong><small className="reference">{order.id}</small><small>{localTime(order.created)}</small></div><div><strong>{money(order.amount)}</strong><span className="status">{t(order.status)}</span></div></li>)}</ul> : <div className="admin-empty"><PackageCheck size={25} aria-hidden="true" /><p>{t("No live orders yet. New orders will appear here.")}</p></div>}
        </section>
        <section className="business-panel">
          <header><h2>{t("Next consultations")}</h2><button className="small-link" onClick={() => onOpen("calendar")}>{t("View all")}<ArrowRight size={14} aria-hidden="true" /></button></header>
          {upcoming.length ? <ul className="admin-record-list">{upcoming.slice(0, 4).map(booking => <li key={booking.id}><div><strong>{localTime(booking.start)}</strong><small>{t(booking.status)}</small><small className="reference">{booking.id}</small></div><CalendarDays size={20} aria-hidden="true" /></li>)}</ul> : <div className="admin-empty"><CalendarDays size={25} aria-hidden="true" /><p>{t("No upcoming consultations. Manage availability in service settings.")}</p></div>}
        </section>
      </div>
    </div>
  );
}

/** ShieldStatus pairs a textual configuration state with a redundant status icon. */
function ShieldStatus({enabled, label}: {enabled: boolean; label: string}) {
  return <p className="admin-provider-line">{enabled ? <Check size={15} aria-hidden="true" /> : <CircleAlert size={15} aria-hidden="true" />}<strong>{t(label)}</strong></p>;
}
