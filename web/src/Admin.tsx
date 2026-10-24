import { t } from "./i18n";
import { useEffect, useRef, useState } from "react";
import type { FormEvent } from "react";
import {
  ArrowRight,
  Check,
  Download,
  FileText,
  IndianRupee,
  Info,
  RefreshCw,
  Settings2,
  ShieldCheck,
  Users,
  Video,
  Search,
} from "lucide-react";
import { api, APIError, localTime, money } from "./api";
import TimeRange from "./TimeRange";
import { parseRupees, rupeeInput } from "./money";
import { navigate, useAppLocation } from "./routing";
import type { AdminData, Config, Rule, Settings } from "./types";
import AdminShop from "./AdminShop";
import AdminLogin from "./AdminLogin";
import AdminOverview from "./AdminOverview";
import AdminVideoLinks from "./AdminVideoLinks";
import { adminSections } from "./admin-navigation";
import "./admin-business.css";
/** businessDate formats today's date in the configured business timezone for administrator defaults. */
const businessDate = (zone: string) =>
  new Intl.DateTimeFormat("en-CA", {
    timeZone: zone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date());
const emptyRule: Rule = {
  id: "",
  version: 1,
  planet: "",
  sign: 0,
  topic: "",
  hindi: "",
  english: "",
  mantra: "",
  gemstone: "",
  priority: 10,
  conflict_group: "",
  status: "draft",
  approver: "",
  approved_at: null,
  demo: false,
};
/** Admin renders the administrator's settings, interpretation approvals, bookings, jobs and finance tools. */
export default function Admin({
  config,
  onConfig,
}: {
  config: Config;
  onConfig: () => Promise<unknown>;
}) {
  const route = useAppLocation();
  const isLogin = route.pathname === "/admin/login";
  const [checking, setChecking] = useState(true);
  const sessionEpoch = useRef(0);
  const [data, setData] = useState<AdminData | null>(null),
    [tab, setTab] = useState("overview"),
    [error, setError] = useState(""),
    [notice, setNotice] = useState(""),
    [busy, setBusy] = useState(false),
    [password, setPassword] = useState(""),
    [totp, setTotp] = useState(""),
    [month, setMonth] = useState(() =>
      businessDate(config.settings.zone).slice(0, 7),
    ),
    [demo, setDemo] = useState(false);
  const [settings, setSettings] = useState<Settings>(config.settings),
    [rule, setRule] = useState<Rule>(emptyRule),
    [json, setJson] = useState("");
  const [expense, setExpense] = useState({
      date: businessDate(config.settings.zone),
      category: "infrastructure",
      description: "",
      amount: "",
      receipt: "",
    }),
    [distribution, setDistribution] = useState({
      partner: "A",
      amount: "",
      reference: "",
    }),
    [closing, setClosing] = useState({
      reserve: "0",
      statement_reference: "",
      closed: false,
    });
  const [refundConfirm, setRefundConfirm] = useState(""),
    [refundReason, setRefundReason] = useState("");
  const [sectionSearch, setSectionSearch] = useState("");
  const [recordSearch, setRecordSearch] = useState("");
  const [orderStatus, setOrderStatus] = useState("all");
  const [shopSection, setShopSection] = useState("products");
  /** openSection opens a management tool, selects the intended shop view and clears stale page messages and record filters. */
  function openSection(id: string, shopTab = "products") {
    setTab(id); setShopSection(shopTab); setError(""); setNotice("");
    setRecordSearch(""); setOrderStatus("all"); setSectionSearch("");
  }
  /** load reloads protected administrator overview data and public configuration. */
  async function load() {
    const epoch = sessionEpoch.current;
    const d = await api<AdminData>(`/admin?month=${month}&demo=${demo}`);
    if (epoch === sessionEpoch.current) setData(d);
    return d;
  }
  useEffect(() => {
    if (!data) setChecking(true);
    void load()
      .catch((error) => {
        if (error instanceof APIError && error.status === 401) {
          setData(null);
          navigate("/admin/login", true);
        } else setError(error.message);
      })
      .finally(() => setChecking(false));
  }, [month, demo]);
  useEffect(() => {
    if (!checking && data && isLogin) navigate("/admin", true);
  }, [checking, !!data, isLogin]);
  useEffect(() => {
    if (data) {
      setSettings(data.settings);
      setJson(JSON.stringify(data.settings, null, 2));
      setClosing({
        reserve: rupeeInput(data.month?.reserve || 0),
        statement_reference: data.month?.statement_reference || "",
        closed: data.month?.closed || false,
      });
    }
  }, [data]);
  /** run wraps an administrator operation with busy/error handling and a subsequent data refresh. */
  async function run(
    fn: () => Promise<unknown>,
    message = t("बदलाव सुरक्षित है।"),
  ) {
    setBusy(true);
    setError("");
    setNotice("");
    try {
      await fn();
      await load();
      setNotice(message);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  /** login submits administrator password/TOTP credentials and reloads the protected workspace. */
  async function login(e: FormEvent) {
    e.preventDefault();
    await run(async () => {
      await api("/admin/login", "POST", { password, totp });
      setPassword("");
      setTotp("");
    }, t("सुरक्षित प्रशासन सत्र सक्रिय है।"));
  }
  /** saveSettings submits the edited merchant, scheduling and content settings with administrator CSRF protection. */
  async function saveSettings(next: Settings) {
    if (!data) return;
    await run(async () => {
      await api("/admin/settings", "PUT", next, data.csrf);
      await onConfig();
    });
  }
  if (checking)
    return (
      <div className="portal-status" role="status">
        <ShieldCheck size={28} aria-hidden="true" />
        <p>{t("Checking your session…")}</p>
      </div>
    );
  if (!data && !isLogin)
    return (
      <div className="portal-status">
        <h1>{t("Administration")}</h1>
        {error ? (
          <>
            <p role="alert">{t(error)}</p>
            <button className="button" onClick={() => location.reload()}>
              {t("Try again")}
            </button>
          </>
        ) : (
          <p role="status">{t("Checking your session…")}</p>
        )}
      </div>
    );
  if (!data)
    return (
      <AdminLogin development={config.development} password={password} totp={totp}
        busy={busy} error={error} onPassword={setPassword} onTotp={setTotp} onSubmit={login} />
    );
  /** action dispatches the selected protected administrator action through the shared operation wrapper. */
  const action = (path: string, body: unknown, message?: string) =>
    run(() => api(path, "POST", body, data.csrf), message);
  const activeSection = adminSections.find(section => section.id === tab)!;
  const matchingOrders = data.orders.filter(order => (orderStatus === "all" || order.status === orderStatus) &&
    [order.id, order.provider_id, order.title, t(order.title)].join(" ").toLowerCase().includes(recordSearch.toLowerCase())).sort((a, b) => b.created.localeCompare(a.created));
  const matchingUsers = data.users.filter(user => [user.id, user.phone].join(" ").toLowerCase().includes(recordSearch.toLowerCase()));
  return (
    <div className="container workspace admin-workspace">
      <aside className="admin-sidebar">
        <div className="admin-sidebar-heading"><ShieldCheck size={19} aria-hidden="true" /><span>{t("Business workspace")}</span></div>
        <label className="admin-section-search"><span className="sr-only">{t("Find a management section")}</span><Search size={16} aria-hidden="true" /><input type="search" value={sectionSearch} onChange={event => setSectionSearch(event.target.value)} placeholder={t("Find a section…")} /></label>
        <nav className="tabs admin-tabs" aria-label={t("प्रशासन अनुभाग")}>
          {["Workspace", "Business", "Manage"].map(group => {
            const sections = adminSections.filter(section => section.group === group && t(section.label).toLowerCase().includes(sectionSearch.toLowerCase()));
            return sections.length ? <div className="admin-nav-group" key={group}><p>{t(group)}</p>{sections.map(section => <button key={section.id} className={tab === section.id ? "active" : ""} aria-current={tab === section.id ? "page" : undefined} onClick={() => openSection(section.id)}><section.icon size={18} aria-hidden="true" /><span>{t(section.label)}</span></button>)}</div> : null;
          })}
          {sectionSearch && !adminSections.some(section => t(section.label).toLowerCase().includes(sectionSearch.toLowerCase())) && <p className="admin-nav-empty">{t("No matching sections.")}</p>}
        </nav>
        <p className="admin-sidebar-note"><ShieldCheck size={15} aria-hidden="true" />{t("Changes are recorded in the activity log.")}</p>
      </aside>
      <div className="admin-main">
      <div className="admin-mobile-navigation"><label htmlFor="admin-mobile-section">{t("Management section")}</label><select id="admin-mobile-section" value={tab} onChange={event => openSection(event.target.value)}>{adminSections.map(section => <option value={section.id} key={section.id}>{t(section.label)}</option>)}</select></div>
      <div className="workspace-heading">
        <div>
          <div className="eyebrow">{t("Mishra 4 You \u00B7 संचालन")}</div>
          <h1>{t(activeSection.label)}</h1>
          <p>{t(activeSection.description)}</p>
        </div>
        <div className="inline-actions">
          <button
            className="button secondary"
            disabled={busy}
            onClick={() => void run(load, t("स्थिति ताज़ा है।"))}
          >
            <RefreshCw size={16} />
            {t("ताज़ा करें")}
          </button>
          <button
            className="small-link"
            disabled={busy}
            onClick={() =>
              void (async () => {
                setBusy(true);
                try {
                  await api("/logout?admin=true", "POST", {}, data.csrf);
                  sessionEpoch.current++;
                  setData(null);
                  setError("");
                  navigate("/admin/login", true);
                } catch (error) {
                  setError((error as Error).message);
                } finally {
                  setBusy(false);
                }
              })()
            }
          >
            {t("लॉग आउट")}
          </button>
        </div>
      </div>
      {error && (
        <p className="error" role="alert">
          <Info size={17} />
          {t(error)}
        </p>
      )}
      {notice && (
        <p className="success" role="status">
          <Check size={17} />
          {t(notice)}
        </p>
      )}
      {tab === "shop" && <AdminShop key={shopSection} initialTab={shopSection} csrf={data.csrf} onConfig={onConfig} />}
      {tab === "video-links" && <AdminVideoLinks csrf={data.csrf} />}
      {tab === "overview" && (
        <>
          <div className="admin-month-control"><label htmlFor="overview-month">{t("Finance month")}</label><input id="overview-month" type="month" value={month} onChange={event => { if (event.target.value) setMonth(event.target.value); }} /></div>
          <AdminOverview data={data} config={config} month={month} includeTestFinance={demo} onOpen={openSection} />
          <details className="admin-advanced"><summary>{t("Service readiness & visitor activity")}</summary>
          <p className="field-note">{t("Historical totals below include test records and deleted customer accounts.")}</p>
          <div className="admin-stats">
            <Metric
              label={t("ग्राहक खाते")}
              value={String(data.users.length)}
              icon={<Users size={22} />}
            />
            <Metric
              label={t("भुगतान सत्यापित")}
              value={String(
                data.orders.filter((o) => o.status === "paid").length,
              )}
              icon={<IndianRupee size={22} />}
            />
            <Metric
              label={t("सक्रिय परामर्श")}
              value={String(
                Object.values(data.bookings).filter((b) =>
                  ["confirmed", "meeting_pending"].includes(b.status),
                ).length,
              )}
              icon={<Video size={22} />}
            />
            <Metric
              label={t("पुनः प्रयास वाले कार्य")}
              value={String(
                Object.values(data.jobs).filter((j) => j.status === "retry")
                  .length,
              )}
              icon={<RefreshCw size={22} />}
            />
          </div>
          <div className="admin-two-col">
            <section className="admin-panel">
              <h2>{t("लाइव सेवा की तैयारी")}</h2>
              <ul className="check-list">
                {[
                  [
                    settings.licence_confirmed,
                    "Swiss Ephemeris licence decision",
                  ],
                  [
                    settings.content.approved,
                    t("गुरुजी परिचय, FAQ और कानूनी नीतियों की स्वीकृति"),
                  ],
                  [
                    settings.tax_mode !== "unconfigured",
                    t("कर व्यवस्था की पुष्टि"),
                  ],
                  [settings.duration > 0, t("परामर्श अवधि का निर्णय")],
                  [settings.pdf_enabled, t("PDF बिक्री सक्षम")],
                  [settings.bookings_enabled, t("लाइव बुकिंग सक्षम")],
                ].map(([ok, label]) => (
                  <li key={String(label)}>
                    {ok ? <Check size={16} /> : <Info size={16} />}
                    <span>
                      {String(label)} · {ok ? t("पूर्ण") : t("बाकी")}
                    </span>
                  </li>
                ))}
              </ul>
              <p className="muted">
                {t(
                  "इन सेटिंग के अलावा लागू स्वीकृत व्याख्या और उपाय भी PDF के लिए आवश्यक हैं। सिमुलेशन उत्पादन में सर्वर द्वारा निषिद्ध हैं।",
                )}
              </p>
            </section>
            <section className="admin-panel">
              <h2>{t("फ़नल \u00B7 सहमति वाले आँकड़े")}</h2>
              {[
                "landing_visit",
                "form_started",
                "kundli_generated",
                "offer_viewed",
                "checkout_started",
                "verified_payment",
                "pdf_delivered",
                "consultation_confirmed",
              ].map((name) => (
                <div className="stat-row" key={name}>
                  <span>{name}</span>
                  <b>{data.analytics.filter((e) => e.name === name).length}</b>
                </div>
              ))}
              <p className="muted">
                {t("नाम, फोन और जन्म विवरण शामिल नहीं। PDF और परामर्श भुगतान:")}{" "}
                {
                  data.analytics.filter(
                    (e) => e.name === "verified_payment" && e.product === "pdf",
                  ).length
                }{" "}
                /{" "}
                {
                  data.analytics.filter(
                    (e) =>
                      e.name === "verified_payment" &&
                      e.product === "consultation",
                  ).length
                }
              </p>
              <p className="field-note">
                {t(
                  "गणना घटना-आधारित है; अद्वितीय ग्राहकों का अनुमान नहीं। UTM: टैब की पहली मान्य source / medium / campaign।",
                )}
              </p>
            </section>
          </div>
          </details>
        </>
      )}
      {tab === "orders" && (
        <section>
          <div className="panel-title">
            <h2>{t("भुगतान और धनवापसी")}</h2>
            <button
              className="button secondary"
              disabled={busy}
              onClick={() =>
                void action(
                  "/admin/reconcile",
                  {},
                  t("भुगतान प्रदाता से पुनः जाँच पूरी हुई।"),
                )
              }
            >
              <RefreshCw size={16} />
              {t("प्रदाता से जाँचें")}
            </button>
          </div>
          <div className="admin-record-filters"><label>{t("Find an order")}<input type="search" value={recordSearch} onChange={event => setRecordSearch(event.target.value)} placeholder={t("Order ID, payment reference or product")} /></label><div className="admin-filter-field"><label htmlFor="admin-payment-status">{t("Payment status")}</label><select id="admin-payment-status" value={orderStatus} onChange={event => setOrderStatus(event.target.value)}><option value="all">{t("All statuses")}</option>{[...new Set(data.orders.map(order => order.status))].sort().map(status => <option key={status} value={status}>{t(status)}</option>)}</select></div><span role="status">{matchingOrders.length} {t("Orders found")}</span></div>
          <div className="table-scroll" tabIndex={0} role="region" aria-label={t(activeSection.label)}>
            <table>
              <thead>
                <tr>
                  <th>{t("ऑर्डर / संदर्भ")}</th>
                  <th>{t("विकल्प")}</th>
                  <th>{t("राशि")}</th>
                  <th>{t("स्थिति")}</th>
                  <th>{t("कार्रवाई")}</th>
                </tr>
              </thead>
              <tbody>
                {!matchingOrders.length && <tr><td colSpan={5}>{t("No orders match these filters.")}</td></tr>}
                {matchingOrders.map((o) => (
                  <tr key={o.id}>
                    <td className="reference">
                      {o.id}
                      <small>{o.provider_id}</small>
                    </td>
                    <td>
                      {t(o.title)}{" "}
                      {o.demo && <span className="pill">{t("डेमो")}</span>}
                    </td>
                    <td>
                      {money(o.amount)}
                      <small>
                        {t("वापसी: ")}
                        {money(o.refunded)}
                      </small>
                    </td>
                    <td>
                      <span className="status">{o.status}</span>
                    </td>
                    <td>
                      {o.status === "paid" && o.refunded === 0 && (
                        <button
                          className="small-link danger-text"
                          onClick={() => {
                            setRefundConfirm(o.id);
                            setRefundReason("");
                          }}
                        >
                          {t("धनवापसी अनुरोध")}
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {refundConfirm && (
            <form
              className="admin-panel space-top"
              onSubmit={(e) => {
                e.preventDefault();
                void action(
                  "/admin/refund",
                  { id: refundConfirm, reason: refundReason },
                  t(
                    "धनवापसी कार्य कतार में है; अंतिम पुष्टि प्रदाता से मिलेगी।",
                  ),
                ).then(() => setRefundConfirm(""));
              }}
            >
              <h3>{t("पूर्ण धनवापसी की स्वीकृति")}</h3>
              <p className="reference">{refundConfirm}</p>
              <label htmlFor="refund-reason">{t("कारण (ऑडिट हेतु)")}</label>
              <input
                id="refund-reason"
                value={refundReason}
                onChange={(e) => setRefundReason(e.target.value)}
                minLength={5}
                required
              />
              <div className="inline-actions">
                <button className="button danger" disabled={busy}>
                  {t("धनवापसी स्वीकृत करें")}
                </button>
                <button
                  type="button"
                  className="small-link"
                  onClick={() => setRefundConfirm("")}
                >
                  {t("वापस जाएँ")}
                </button>
              </div>
            </form>
          )}
        </section>
      )}
      {tab === "jobs" && (
        <section>
          <h2>{t("रिपोर्ट, बैठक और सूचनाओं की कतार")}</h2>
          <p className="muted">
            {t(
              "अस्थायी विफलता पर बढ़ते अंतराल से पुनः प्रयास होता है। PDF दुबारा खरीदने की ज़रूरत नहीं।",
            )}
          </p>
          <div className="table-scroll" tabIndex={0} role="region" aria-label={t(activeSection.label)}>
            <table>
              <thead>
                <tr>
                  <th>{t("कार्य")}</th>
                  <th>{t("स्थिति / प्रयास")}</th>
                  <th>{t("त्रुटि / अगला प्रयास")}</th>
                  <th>{t("कार्रवाई")}</th>
                </tr>
              </thead>
              <tbody>
                {Object.values(data.jobs).map((j) => (
                  <tr key={j.id}>
                    <td>
                      {j.kind}
                      <small className="reference">{j.ref}</small>
                    </td>
                    <td>
                      {j.status} · {j.attempts}
                    </td>
                    <td>
                      {j.error || "—"}
                      <small>{localTime(j.next)}</small>
                    </td>
                    <td>
                      <button
                        className="small-link"
                        disabled={busy || j.status === "running"}
                        onClick={() =>
                          void action("/admin/jobs/retry", { id: j.id })
                        }
                      >
                        {t("पुनः चलाएँ")}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}
      {tab === "calendar" && (
        <section>
          <h2>{t("परामर्श कैलेंडर")}</h2>
          <p className="muted">
            {t(
              "ब्राउज़र के समय-क्षेत्र में। निजी attendee लिंक ग्राहक को मिलता है; Zoom host लिंक कभी सहेजा नहीं जाता।",
            )}
          </p>
          {Object.values(data.bookings)
            .sort((a, b) => a.start.localeCompare(b.start))
            .map((b) => (
              <article className="booking-row" key={b.id}>
                <Video size={25} />
                <div className="booking-info">
                  <TimeRange start={b.start} end={b.end} format={localTime} />
                  <p>
                    {b.status} · {b.demo ? t("डेमो") : t("वास्तविक")}
                  </p>
                  <small className="reference">{b.id}</small>
                </div>
                <div className="booking-actions">
                  <button
                    className="small-link"
                    disabled={busy}
                    onClick={() =>
                      void action(`/admin/bookings/${b.id}`, {
                        action: "cancel",
                        slot: "",
                      })
                    }
                  >
                    {t("रद्द करें")}
                  </button>
                  <form
                    onSubmit={(e) => {
                      e.preventDefault();
                      const form = e.currentTarget;
                      const value = (
                        new FormData(form).get("slot") || ""
                      ).toString();
                      void action(`/admin/bookings/${b.id}`, {
                        action: "reschedule",
                        slot: new Date(value).toISOString(),
                      });
                    }}
                  >
                    <label className="sr-only" htmlFor={"slot-" + b.id}>
                      {t("नया स्थानीय समय")}
                    </label>
                    <input
                      id={"slot-" + b.id}
                      name="slot"
                      type="datetime-local"
                      required
                    />
                    <button className="small-link" disabled={busy}>
                      {t("पुनर्निर्धारित करें")}
                    </button>
                  </form>
                </div>
              </article>
            ))}
          {!Object.keys(data.bookings).length && (
            <p>{t("अभी कोई बुकिंग नहीं।")}</p>
          )}
        </section>
      )}
      {tab === "rules" && (
        <div className="admin-two-col">
          <section className="admin-panel">
            <h2>{t("नियम और सामग्री संस्करण")}</h2>
            <p className="muted">
              {t(
                "स्वीकृत संस्करण अपरिवर्तनीय है। बदलाव के लिए अगला संस्करण बनाएँ। समान conflict group में उच्च priority, फिर ID और नवीन संस्करण चुना जाता है।",
              )}
            </p>
            {Object.values(data.rules).map((v) => (
              <button
                className="rule-row"
                key={v.id + v.version}
                onClick={() =>
                  setRule({
                    ...v,
                    version:
                      v.status === "approved" ? v.version + 1 : v.version,
                    status: v.status === "approved" ? "draft" : v.status,
                    approved_at: null,
                  })
                }
              >
                <span>
                  <b>{v.topic || v.id}</b>
                  <small>
                    {v.id} · v{v.version}
                  </small>
                </span>
                <span className="pill">{v.demo ? t("डेमो") : v.status}</span>
              </button>
            ))}
            <button
              className="small-link space-top"
              onClick={() => setRule({ ...emptyRule })}
            >
              {t("नया नियम बनाएँ +")}
            </button>
          </section>
          <form
            className="admin-panel"
            onSubmit={(e) => {
              e.preventDefault();
              void action(
                "/admin/rules",
                rule,
                t(
                  "नियम का संस्करण सुरक्षित है। स्वीकृति का दावा केवल वास्तविक अनुमोदन के बाद करें।",
                ),
              );
            }}
          >
            <h2>{t("व्याख्या / उपाय संपादक")}</h2>
            <div className="form-row">
              <Field label={t("नियम ID")}>
                <input
                  value={rule.id}
                  onChange={(e) => setRule({ ...rule, id: e.target.value })}
                  minLength={3}
                  required
                />
              </Field>
              <Field label={t("संस्करण")}>
                <input
                  type="number"
                  min="1"
                  value={rule.version}
                  onChange={(e) =>
                    setRule({ ...rule, version: +e.target.value })
                  }
                  required
                />
              </Field>
            </div>
            <div className="form-row">
              <Field label={t("ग्रह शर्त")}>
                <select
                  value={rule.planet}
                  onChange={(e) => setRule({ ...rule, planet: e.target.value })}
                >
                  <option value="">{t("सभी कुंडलियाँ")}</option>
                  {[
                    "Sun",
                    "Moon",
                    "Mars",
                    "Mercury",
                    "Jupiter",
                    "Venus",
                    "Saturn",
                    "Rahu",
                    "Ketu",
                  ].map((p) => (
                    <option key={p}>{p}</option>
                  ))}
                </select>
              </Field>
              <Field label={t("राशि (0 = सभी)")}>
                <input
                  type="number"
                  min="0"
                  max="12"
                  value={rule.sign}
                  onChange={(e) => setRule({ ...rule, sign: +e.target.value })}
                />
              </Field>
            </div>
            <Field label={t("विषय / श्रेणी")}>
              <input
                value={rule.topic}
                onChange={(e) => setRule({ ...rule, topic: e.target.value })}
                required
              />
            </Field>
            <Field label={t("Faladesh category")}>
              <select
                value={rule.category || "general"}
                onChange={(e) => setRule({ ...rule, category: e.target.value })}
              >
                {[
                  "General",
                  "Personality",
                  "Education",
                  "Career",
                  "Relationships",
                  "Family",
                  "Wellbeing",
                  "Dasha",
                ].map((name) => (
                  <option key={name} value={name.toLowerCase()}>
                    {t(name)}
                  </option>
                ))}
              </select>
            </Field>
            <div className="form-row">
              <Field label={t("Planet house (0 = any)")}>
                <input
                  type="number"
                  min="0"
                  max="12"
                  value={rule.house || 0}
                  onChange={(e) => setRule({ ...rule, house: +e.target.value })}
                />
              </Field>
              <Field label={t("Ascendant sign (0 = any)")}>
                <input
                  type="number"
                  min="0"
                  max="12"
                  value={rule.lagna_sign || 0}
                  onChange={(e) =>
                    setRule({ ...rule, lagna_sign: +e.target.value })
                  }
                />
              </Field>
            </div>
            <div className="form-row">
              <Field label={t("Moon nakshatra (0 = any)")}>
                <input
                  type="number"
                  min="0"
                  max="27"
                  value={rule.nakshatra_number || 0}
                  onChange={(e) =>
                    setRule({ ...rule, nakshatra_number: +e.target.value })
                  }
                />
              </Field>
              <Field label={t("Mahadasha condition")}>
                <select
                  value={rule.dasha_lord || ""}
                  onChange={(e) =>
                    setRule({ ...rule, dasha_lord: e.target.value })
                  }
                >
                  <option value="">{t("Any dasha")}</option>
                  {[
                    "Sun",
                    "Moon",
                    "Mars",
                    "Mercury",
                    "Jupiter",
                    "Venus",
                    "Saturn",
                    "Rahu",
                    "Ketu",
                  ].map((name) => (
                    <option key={name}>{name}</option>
                  ))}
                </select>
              </Field>
            </div>
            <Field label={t("हिंदी व्याख्या")}>
              <textarea
                rows={4}
                value={rule.hindi}
                onChange={(e) => setRule({ ...rule, hindi: e.target.value })}
                minLength={10}
                required
              />
            </Field>
            <Field label={t("English (वैकल्पिक)")}>
              <textarea
                value={rule.english}
                onChange={(e) => setRule({ ...rule, english: e.target.value })}
              />
            </Field>
            <Field label={t("स्वीकृत मंत्र (वैकल्पिक)")}>
              <textarea
                value={rule.mantra}
                onChange={(e) => setRule({ ...rule, mantra: e.target.value })}
              />
            </Field>
            <Field label={t("स्वीकृत रत्न मार्गदर्शन (वैकल्पिक)")}>
              <textarea
                value={rule.gemstone}
                onChange={(e) => setRule({ ...rule, gemstone: e.target.value })}
              />
            </Field>
            <div className="form-row">
              <Field label="Priority">
                <input
                  type="number"
                  value={rule.priority}
                  onChange={(e) =>
                    setRule({ ...rule, priority: +e.target.value })
                  }
                />
              </Field>
              <Field label="Conflict group">
                <input
                  value={rule.conflict_group}
                  onChange={(e) =>
                    setRule({ ...rule, conflict_group: e.target.value })
                  }
                />
              </Field>
            </div>
            <Field label={t("स्थिति")}>
              <select
                value={rule.status}
                onChange={(e) => setRule({ ...rule, status: e.target.value })}
              >
                <option value="draft">{t("प्रारूप")}</option>
                <option value="approved">{t("गुरुजी द्वारा स्वीकृत")}</option>
                <option value="retired">{t("वापस लिया गया (प्रारूप)")}</option>
              </select>
            </Field>
            <Field label={t("वास्तविक स्वीकृति देने वाले का नाम")}>
              <input
                value={rule.approver}
                onChange={(e) => setRule({ ...rule, approver: e.target.value })}
                required={rule.status === "approved"}
              />
            </Field>
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={rule.demo}
                onChange={(e) => setRule({ ...rule, demo: e.target.checked })}
              />
              {t("केवल डेमो सामग्री (लाइव भुगतान में निषिद्ध)")}
            </label>
            <button className="button full" disabled={busy}>
              {t("संस्करण सुरक्षित करें ")}
              <Check size={17} />
            </button>
          </form>
        </div>
      )}
      {tab === "content" && (
        <form
          className="admin-panel"
          onSubmit={(e) => {
            e.preventDefault();
            void saveSettings(settings);
          }}
        >
          <h2>{t("स्वीकृत वेबसाइट सामग्री")}</h2>
          <Field label={t("गुरुजी का परिचय")}>
            <textarea
              rows={4}
              value={settings.content.intro}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  content: { ...settings.content, intro: e.target.value },
                })
              }
            />
          </Field>
          <Field label={t("सहायता संपर्क")}>
            <input
              value={settings.content.support}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  content: { ...settings.content, support: e.target.value },
                })
              }
            />
          </Field>
          <Field label={t("YouTube वीडियो ID (अधिकतम 10, प्रति पंक्ति एक)")}>
            <textarea
              value={settings.content.videos.join("\n")}
              placeholder={t("स्वीकृत वीडियो के 11 अक्षर वाले ID")}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  content: {
                    ...settings.content,
                    videos: e.target.value
                      .split("\n")
                      .map((id) => id.trim())
                      .filter(Boolean),
                  },
                })
              }
            />
          </Field>
          {Object.entries(settings.content.policies).map(([k, v]) => (
            <Field key={k} label={t("नीति: ") + k}>
              <textarea
                rows={4}
                value={v}
                onChange={(e) =>
                  setSettings({
                    ...settings,
                    content: {
                      ...settings.content,
                      policies: {
                        ...settings.content.policies,
                        [k]: e.target.value,
                      },
                    },
                  })
                }
              />
            </Field>
          ))}
          <h3>FAQ</h3>
          {settings.content.faq.map((v, i) => (
            <div className="faq-editor" key={i}>
              <input
                aria-label={t("सवाल {v0}", { v0: i + 1 })}
                value={v.question}
                onChange={(e) =>
                  setSettings({
                    ...settings,
                    content: {
                      ...settings.content,
                      faq: settings.content.faq.map((f, j) =>
                        i === j ? { ...f, question: e.target.value } : f,
                      ),
                    },
                  })
                }
              />
              <textarea
                aria-label={t("जवाब {v0}", { v0: i + 1 })}
                value={v.answer}
                onChange={(e) =>
                  setSettings({
                    ...settings,
                    content: {
                      ...settings.content,
                      faq: settings.content.faq.map((f, j) =>
                        i === j ? { ...f, answer: e.target.value } : f,
                      ),
                    },
                  })
                }
              />
              <button
                type="button"
                className="small-link"
                onClick={() =>
                  setSettings({
                    ...settings,
                    content: {
                      ...settings.content,
                      faq: settings.content.faq.filter((_, j) => j !== i),
                    },
                  })
                }
              >
                {t("सवाल हटाएँ")}
              </button>
            </div>
          ))}
          <button
            type="button"
            className="small-link"
            onClick={() =>
              setSettings({
                ...settings,
                content: {
                  ...settings.content,
                  faq: [...settings.content.faq, { question: "", answer: "" }],
                },
              })
            }
          >
            {t("सवाल जोड़ें +")}
          </button>
          <label className="checkbox-label space-top">
            <input
              type="checkbox"
              checked={settings.content.approved}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  content: { ...settings.content, approved: e.target.checked },
                })
              }
            />
            {t(
              "परिचय, FAQ, नीतियाँ और संपर्क वास्तविक स्वीकृति के बाद प्रकाशित करने योग्य हैं।",
            )}
          </label>
          <button className="button" disabled={busy}>
            {t("सामग्री सुरक्षित करें ")}
            <Check size={17} />
          </button>
        </form>
      )}
      {tab === "settings" && (
        <form
          className="admin-panel"
          onSubmit={(e) => {
            e.preventDefault();
            void saveSettings(settings);
          }}
        >
          <h2>{t("उत्पाद और उपलब्धता")}</h2>
          <p className="muted">
            {t(
              "कीमतें यहाँ पूर्णांक पैसे में हैं। बदलाव पुराने ऑर्डर की राशि या रिपोर्ट नहीं बदलते।",
            )}
          </p>
          <div className="form-row">
            <NumberField
              label={t("PDF मूल्य (पैसे)")}
              value={settings.pdf_price}
              onChange={(v) => setSettings({ ...settings, pdf_price: v })}
            />
            <NumberField
              label={t("परामर्श मूल्य (पैसे)")}
              value={settings.consultation_price}
              onChange={(v) =>
                setSettings({ ...settings, consultation_price: v })
              }
            />
          </div>
          <div className="form-row">
            <NumberField
              label={t("सत्र अवधि (मिनट, 0 = अनिर्णीत)")}
              value={settings.duration}
              onChange={(v) => setSettings({ ...settings, duration: v })}
            />
            <NumberField
              label={t("बफ़र (मिनट)")}
              value={settings.buffer}
              onChange={(v) => setSettings({ ...settings, buffer: v })}
            />
          </div>
          <div className="form-row">
            <NumberField
              label={t("बुकिंग से पहले न्यूनतम घंटे")}
              value={settings.lead_hours}
              onChange={(v) => setSettings({ ...settings, lead_hours: v })}
            />
            <NumberField
              label={t("बुकिंग क्षितिज (दिन)")}
              value={settings.horizon_days}
              onChange={(v) => setSettings({ ...settings, horizon_days: v })}
            />
          </div>
          <div className="form-row">
            <NumberField
              label={t("रद्द / बदलाव सीमा (घंटे)")}
              value={settings.change_hours}
              onChange={(v) => setSettings({ ...settings, change_hours: v })}
            />
            <Field label={t("उपलब्धता IANA timezone")}>
              <input
                value={settings.zone}
                onChange={(e) =>
                  setSettings({ ...settings, zone: e.target.value })
                }
              />
            </Field>
          </div>
          <h3>{t("साप्ताहिक उपलब्धता")}</h3>
          {settings.availability.map((v, i) => (
            <div className="availability-row" key={i}>
              <select
                aria-label={t("सप्ताह का दिन")}
                value={v.day}
                onChange={(e) =>
                  setSettings({
                    ...settings,
                    availability: settings.availability.map((w, j) =>
                      j === i ? { ...w, day: +e.target.value } : w,
                    ),
                  })
                }
              >
                {[
                  t("रविवार"),
                  t("सोमवार"),
                  t("मंगलवार"),
                  t("बुधवार"),
                  t("गुरुवार"),
                  t("शुक्रवार"),
                  t("शनिवार"),
                ].map((d, n) => (
                  <option key={d} value={n}>
                    {d}
                  </option>
                ))}
              </select>
              <label>
                <span>{t("Start time")}</span>
                <input
                  aria-label={t("शुरू समय")}
                  type="time"
                  value={v.from}
                  onChange={(e) =>
                    setSettings({
                      ...settings,
                      availability: settings.availability.map((w, j) =>
                        j === i ? { ...w, from: e.target.value } : w,
                      ),
                    })
                  }
                />
              </label>
              <label>
                <span>{t("End time")}</span>
                <input
                  aria-label={t("अंत समय")}
                  type="time"
                  value={v.to}
                  onChange={(e) =>
                    setSettings({
                      ...settings,
                      availability: settings.availability.map((w, j) =>
                        j === i ? { ...w, to: e.target.value } : w,
                      ),
                    })
                  }
                />
              </label>
              <button
                type="button"
                className="small-link"
                onClick={() =>
                  setSettings({
                    ...settings,
                    availability: settings.availability.filter(
                      (_, j) => j !== i,
                    ),
                  })
                }
              >
                {t("हटाएँ")}
              </button>
            </div>
          ))}
          <button
            type="button"
            className="small-link"
            onClick={() =>
              setSettings({
                ...settings,
                availability: [
                  ...settings.availability,
                  { day: 1, from: "10:00", to: "13:00" },
                ],
              })
            }
          >
            {t("समय खिड़की जोड़ें +")}
          </button>
          <Field
            label={t("छुट्टी / बंद तारीखें (YYYY-MM-DD, प्रति पंक्ति एक)")}
          >
            <textarea
              value={settings.blocked.join("\n")}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  blocked: e.target.value.split("\n").filter(Boolean),
                })
              }
            />
          </Field>
          <div className="form-row">
            <Field label={t("कर व्यवस्था")}>
              <select
                value={settings.tax_mode}
                onChange={(e) =>
                  setSettings({ ...settings, tax_mode: e.target.value })
                }
              >
                <option value="unconfigured">{t("निर्णय बाकी")}</option>
                <option value="inclusive">{t("कीमत में शामिल")}</option>
                <option value="exclusive">{t("कीमत के अतिरिक्त")}</option>
                <option value="none">{t("लागू नहीं (पुष्टि के बाद)")}</option>
              </select>
            </Field>
            <NumberField
              label={t("कर दर (basis points; 100 = 1%)")}
              value={settings.tax_bps}
              onChange={(v) => setSettings({ ...settings, tax_bps: v })}
            />
          </div>
          {[
            [
              "licence_confirmed",
              t(
                "Swiss Ephemeris licence की शर्तें पूरी होने की वास्तविक पुष्टि",
              ),
            ],
            [
              "pdf_enabled",
              t("लाइव PDF बिक्री सक्षम (स्वीकृत लागू सामग्री भी आवश्यक)"),
            ],
            [
              "bookings_enabled",
              t("लाइव परामर्श सक्षम (अवधि पहले निर्धारित करें)"),
            ],
          ].map(([key, label]) => (
            <label className="checkbox-label" key={key}>
              <input
                type="checkbox"
                checked={Boolean(settings[key as keyof Settings])}
                onChange={(e) =>
                  setSettings({ ...settings, [key]: e.target.checked })
                }
              />
              {label}
            </label>
          ))}
          <button className="button" disabled={busy}>
            {t("सेटिंग सुरक्षित करें ")}
            <Settings2 size={17} />
          </button>
          <details className="space-top">
            <summary>{t("उन्नत पूर्ण कॉन्फ़िगरेशन")}</summary>
            <textarea
              className="code-editor"
              rows={18}
              value={json}
              onChange={(e) => setJson(e.target.value)}
            />
            <button
              className="button secondary"
              type="button"
              disabled={busy}
              onClick={() => {
                try {
                  void saveSettings(JSON.parse(json));
                } catch {
                  setError(t("JSON सही नहीं है।"));
                }
              }}
            >
              {t("JSON सेटिंग लागू करें")}
            </button>
          </details>
        </form>
      )}
      {tab === "finance" && (
        <>
          <div className="panel-title">
            <div>
              <h2>{t("मासिक साझेदारी रिपोर्ट")}</h2>
              <p className="muted">
                {t(
                  "50\u201350 वितरण का अनुमान। किसी खाते में स्वचालित धन हस्तांतरण नहीं।",
                )}
              </p>
            </div>
            <div className="inline-actions">
              <input
                aria-label={t("लेखा महीना")}
                type="month"
                value={month}
                onChange={(e) => setMonth(e.target.value)}
              />
              {config.development && (
                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={demo}
                    onChange={(e) => setDemo(e.target.checked)}
                  />
                  {t("डेमो बिक्री")}
                </label>
              )}
              <a
                className="button secondary"
                href={`/api/admin/finance.csv?month=${month}&demo=${demo}`}
              >
                <Download size={16} />
                CSV
              </a>
            </div>
          </div>
          {(!data.finance.tax_configured || data.finance.unreconciled) && (
            <div className="notice">
              <Info size={18} />
              <span>
                {t("अनुमान मात्र:")}{" "}
                {!data.finance.tax_configured
                  ? t("कर व्यवस्था अनिर्णीत। ")
                  : ""}
                {data.finance.unreconciled
                  ? t("बैंक / प्रदाता विवरण से मासिक मिलान बाकी।")
                  : ""}
              </span>
            </div>
          )}
          {data.finance.calculation_error ? (
            <p role="alert" className="notice">
              {t(data.finance.calculation_error)}
            </p>
          ) : (
            <div className="finance-grid">
              {(
                [
                  [t("PDF संग्रह"), "pdf_sales"],
                  [t("Gemstone sales"), "gemstone_sales"],
                  [t("परामर्श संग्रह"), "consultation_sales"],
                  [t("सकल संग्रह"), "gross"],
                  [t("धनवापसी"), "refunds"],
                  [t("प्रदाता शुल्क"), "fees"],
                  [t("कर देयता"), "tax"],
                  [t("स्वीकृत व्यय"), "expenses"],
                  [t("रक्षित राशि"), "reserves"],
                  [t("अनुमानित वितरण योग्य"), "distributable"],
                  [t("साझेदार A \u00B7 50%"), "partner_a"],
                  [t("साझेदार B \u00B7 50%"), "partner_b"],
                  [t("पैसा rounding शेष"), "rounding_remainder"],
                  [t("वास्तविक दर्ज वितरण"), "distributed"],
                  [t("अवितरित शेष"), "undistributed"],
                ] as const
              ).map(([label, key]) => (
                <div
                  className={key === "distributable" ? "finance-highlight" : ""}
                  key={key}
                >
                  <span>{label}</span>
                  <b>{money(data.finance[key])}</b>
                </div>
              ))}
            </div>
          )}
          <div className="admin-two-col space-top">
            <form
              className="admin-panel"
              onSubmit={(e) => {
                e.preventDefault();
                const amount = parseRupees(expense.amount);
                if (amount === null) {
                  setError(
                    t("Enter a valid amount with at most two decimal places."),
                  );
                  return;
                }
                void action("/admin/expenses", {
                  id: "",
                  ...expense,
                  amount,
                  status: "pending",
                  approver: "",
                });
              }}
            >
              <h3>{t("व्यय दर्ज करें")}</h3>
              <p className="muted">
                {t(
                  "\u20B91,700/माह केवल योजना अनुमान है। WhatsApp विपणन अलग श्रेणी है। प्रदाता शुल्क / कर / वापसी को व्यय में दुबारा न लिखें।",
                )}
              </p>
              <Field label={t("तारीख")}>
                <input
                  type="date"
                  required
                  value={expense.date}
                  onChange={(e) =>
                    setExpense({ ...expense, date: e.target.value })
                  }
                />
              </Field>
              <Field label={t("श्रेणी")}>
                <select
                  value={expense.category}
                  onChange={(e) =>
                    setExpense({ ...expense, category: e.target.value })
                  }
                >
                  <option value="infrastructure">Infrastructure</option>
                  <option value="whatsapp_marketing">WhatsApp marketing</option>
                  <option value="licence">Licence</option>
                  <option value="operations">Operations</option>
                </select>
              </Field>
              <Field label={t("विवरण")}>
                <input
                  required
                  value={expense.description}
                  onChange={(e) =>
                    setExpense({ ...expense, description: e.target.value })
                  }
                />
              </Field>
              <Field label={t("राशि (\u20B9)")}>
                <input
                  type="number"
                  min="0.01"
                  step=".01"
                  value={expense.amount}
                  onChange={(e) =>
                    setExpense({ ...expense, amount: e.target.value })
                  }
                  required
                />
              </Field>
              <Field label={t("निजी रसीद (PDF / JPEG / PNG, अधिकतम 5 MB)")}>
                <input
                  type="file"
                  accept="application/pdf,image/png,image/jpeg"
                  onChange={async (e) => {
                    const file = e.target.files?.[0];
                    if (!file) return;
                    try {
                      const res = await fetch("/api/admin/receipts", {
                        method: "POST",
                        headers: {
                          "X-CSRF-Token": data.csrf,
                          "Content-Type": file.type,
                        },
                        body: file,
                      });
                      const d = await res.json();
                      if (!res.ok) throw new Error(d.error);
                      setExpense({ ...expense, receipt: d.receipt });
                      setNotice(t("रसीद निजी संग्रह में सुरक्षित है।"));
                    } catch (e) {
                      setError((e as Error).message);
                    }
                  }}
                />
              </Field>
              <button className="button" disabled={busy || !expense.receipt}>
                {t("व्यय समीक्षा के लिए दर्ज करें")}
              </button>
            </form>
            <div>
              <form
                className="admin-panel"
                onSubmit={(e) => {
                  e.preventDefault();
                  const reserve = parseRupees(closing.reserve);
                  if (reserve === null) {
                    setError(
                      t(
                        "Enter a valid amount with at most two decimal places.",
                      ),
                    );
                    return;
                  }
                  void run(() =>
                    api(
                      "/admin/month",
                      "PUT",
                      {
                        month,
                        reserve,
                        statement_reference: closing.statement_reference,
                        closed: closing.closed,
                        by: "",
                      },
                      data.csrf,
                    ),
                  );
                }}
              >
                <h3>{t("मासिक मिलान / रक्षित राशि")}</h3>
                <Field label={t("रक्षित राशि (\u20B9)")}>
                  <input
                    type="number"
                    min="0"
                    step=".01"
                    value={closing.reserve}
                    onChange={(e) =>
                      setClosing({ ...closing, reserve: e.target.value })
                    }
                  />
                </Field>
                <Field label={t("बैंक / प्रदाता विवरण संदर्भ")}>
                  <input
                    value={closing.statement_reference}
                    onChange={(e) =>
                      setClosing({
                        ...closing,
                        statement_reference: e.target.value,
                      })
                    }
                  />
                </Field>
                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={closing.closed}
                    onChange={(e) =>
                      setClosing({ ...closing, closed: e.target.checked })
                    }
                  />
                  {t("विवरण और अभिलेखों का मिलान पूरा है")}
                </label>
                <button className="button secondary" disabled={busy}>
                  {t("मिलान सुरक्षित करें")}
                </button>
              </form>
              <form
                className="admin-panel space-top"
                onSubmit={(e) => {
                  e.preventDefault();
                  const amount = parseRupees(distribution.amount);
                  if (amount === null) {
                    setError(
                      t(
                        "Enter a valid amount with at most two decimal places.",
                      ),
                    );
                    return;
                  }
                  void action("/admin/distributions", {
                    id: "",
                    month,
                    ...distribution,
                    amount,
                  });
                }}
              >
                <h3>{t("वास्तविक साझेदार वितरण दर्ज करें")}</h3>
                <Field label={t("साझेदार")}>
                  <select
                    value={distribution.partner}
                    onChange={(e) =>
                      setDistribution({
                        ...distribution,
                        partner: e.target.value,
                      })
                    }
                  >
                    <option>A</option>
                    <option>B</option>
                  </select>
                </Field>
                <Field label={t("राशि (\u20B9)")}>
                  <input
                    type="number"
                    step=".01"
                    min=".01"
                    value={distribution.amount}
                    onChange={(e) =>
                      setDistribution({
                        ...distribution,
                        amount: e.target.value,
                      })
                    }
                    required
                  />
                </Field>
                <Field label={t("वास्तविक हस्तांतरण संदर्भ")}>
                  <input
                    value={distribution.reference}
                    onChange={(e) =>
                      setDistribution({
                        ...distribution,
                        reference: e.target.value,
                      })
                    }
                    required
                  />
                </Field>
                <button className="button secondary" disabled={busy}>
                  {t("वितरण दर्ज करें")}
                </button>
              </form>
            </div>
          </div>
          <h3 className="space-top">{t("व्यय समीक्षा")}</h3>
          <div className="table-scroll" tabIndex={0} role="region" aria-label={t(activeSection.label)}>
            <table>
              <thead>
                <tr>
                  <th>{t("तारीख")}</th>
                  <th>{t("श्रेणी / विवरण")}</th>
                  <th>{t("राशि")}</th>
                  <th>{t("रसीद")}</th>
                  <th>{t("स्थिति")}</th>
                </tr>
              </thead>
              <tbody>
                {Object.values(data.expenses).map((e) => (
                  <tr key={e.id}>
                    <td>{e.date}</td>
                    <td>
                      {e.category}
                      <small>{e.description}</small>
                    </td>
                    <td>{money(e.amount)}</td>
                    <td>
                      <a href={"/api/admin/receipts/" + e.receipt}>
                        {t("निजी रसीद")}
                      </a>
                    </td>
                    <td>
                      {e.status === "pending" ? (
                        <button
                          className="small-link"
                          disabled={busy}
                          onClick={() =>
                            void action(`/admin/expenses/${e.id}/approve`, {})
                          }
                        >
                          {t("स्वीकृत करें")}
                        </button>
                      ) : (
                        t("स्वीकृत")
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <h3>{t("दर्ज वितरण")}</h3>
          {Object.values(data.distributions)
            .filter((d) => d.month === month)
            .map((d) => (
              <p key={d.id}>
                {d.partner} · {money(d.amount)} · {d.reference}
              </p>
            ))}
        </>
      )}
      {tab === "users" && (
        <>
          <h2>{t("ग्राहक और जन्म प्रोफ़ाइल")}</h2>
          <p className="muted">
            {t(
              "सूची में फोन छिपे हैं। संवेदनशील जन्म विवरण प्रशासन सूची से बाहर हैं। केवल व्यवस्थापक भूमिका को यह अनुभाग मिलता है।",
            )}
          </p>
          <div className="admin-record-filters"><label>{t("Find a customer")}<input type="search" value={recordSearch} onChange={event => setRecordSearch(event.target.value)} placeholder={t("Customer ID or last four phone digits")} /></label><span role="status">{matchingUsers.length} {t("Customers found")}</span></div>
          <div className="table-scroll" tabIndex={0} role="region" aria-label={t(activeSection.label)}>
            <table>
              <thead>
                <tr>
                  <th>{t("ग्राहक ID")}</th>
                  <th>{t("मोबाइल")}</th>
                  <th>{t("बना")}</th>
                  <th>{t("प्रोफ़ाइल")}</th>
                  <th>{t("प्रचार सहमति")}</th>
                </tr>
              </thead>
              <tbody>
                {!matchingUsers.length && <tr><td colSpan={5}>{t("No customers match this search.")}</td></tr>}
                {matchingUsers.map((u) => (
                  <tr key={u.id}>
                    <td className="reference">{u.id}</td>
                    <td>{u.phone}</td>
                    <td>{localTime(u.created)}</td>
                    <td>
                      {data.profiles.filter((p) => p.user_id === u.id).length}
                    </td>
                    <td>
                      {u.deleted
                        ? t("खाता मिटाया")
                        : u.marketing
                          ? t("अलग से दी गई")
                          : t("नहीं")}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
      {tab === "audit" && (
        <>
          <h2>{t("संचालन ऑडिट")}</h2>
          <div className="table-scroll" tabIndex={0} role="region" aria-label={t(activeSection.label)}>
            <table>
              <thead>
                <tr>
                  <th>{t("समय")}</th>
                  <th>{t("कर्ता")}</th>
                  <th>{t("कार्रवाई")}</th>
                  <th>{t("संदर्भ")}</th>
                </tr>
              </thead>
              <tbody>
                {data.audit
                  .slice()
                  .reverse()
                  .slice(0, 250)
                  .map((v, i) => (
                    <tr key={i}>
                      <td>{localTime(v.at)}</td>
                      <td>{v.actor}</td>
                      <td>{v.action}</td>
                      <td className="reference">{v.ref}</td>
                    </tr>
                  ))}
              </tbody>
            </table>
          </div>
        </>
      )}
      </div>
    </div>
  );
}
/** Field renders a labelled administrator input and forwards the edited string value. */
function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <label className="admin-field">
      <span>{label}</span>
      {children}
    </label>
  );
}
/** NumberField renders a labelled numeric administrator field with the supplied bounds and change callback. */
function NumberField({
  label,
  value,
  onChange,
}: {
  label: string;
  value: number;
  onChange: (v: number) => void;
}) {
  return (
    <Field label={label}>
      <input
        type="number"
        value={value}
        onChange={(e) => onChange(+e.target.value)}
        required
      />
    </Field>
  );
}
/** Metric renders one labelled operational or financial total in the administrator overview. */
function Metric({
  label,
  value,
  icon,
}: {
  label: string;
  value: string;
  icon: React.ReactNode;
}) {
  return (
    <div className="admin-stat">
      {icon}
      <b>{value}</b>
      <span>{label}</span>
    </div>
  );
}
