import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { Gem, Plus, RefreshCw, Save, Upload } from "lucide-react";
import { api, localTime, money } from "./api";
import { parseRupees, rupeeInput } from "./money";
import { t } from "./i18n";
import { Button, ErrorBox } from "./UI";
import GemstoneArt from "./GemstoneArt";
import ShopOrders from "./ShopOrders";
import { carats, parseCarats } from "./shop-types";
import type { Gemstone, ShopSettings } from "./shop-types";
import type { Order } from "./types";
import "./shop.css";

type Data = { products: Gemstone[]; settings: ShopSettings; orders: Order[] };
const blank: Gemstone = {
  id: "",
  slug: "",
  name: "",
  hindi_name: "",
  kind: "precious",
  description: "",
  hindi_description: "",
  price: 0,
  weight_milli_carat: 0,
  stock: 0,
  available: 0,
  origin: "",
  material: "not-specified",
  treatment: "",
  shape: "",
  dimensions: "",
  image: "",
  certificate_lab: "",
  certificate_no: "",
  certificate_url: "",
  status: "draft",
  version: 0,
  updated: "0001-01-01T00:00:00Z",
  preview: false,
  sample: false,
};
/** slugify normalizes a product name into a lowercase, URL-safe catalogue slug. */
const slugify = (s: string) =>
  s
    .toLowerCase()
    .normalize("NFKD")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "");

/** AdminShop manages gemstone products, merchant settings, stock and order fulfilment in the admin workspace. */
export default function AdminShop({
  csrf,
  onConfig,
  initialTab = "products",
}: {
  csrf: string;
  onConfig: () => Promise<unknown>;
  initialTab?: string;
}) {
  const [data, setData] = useState<Data | null>(null),
    [tab, setTab] = useState(initialTab),
    [error, setError] = useState(""),
    [notice, setNotice] = useState(""),
    [busy, setBusy] = useState(false);
  const [search, setSearch] = useState("");
  const [product, setProduct] = useState<Gemstone>({ ...blank }),
    [price, setPrice] = useState("0.00"),
    [weight, setWeight] = useState("0");
  const [settings, setSettings] = useState<ShopSettings | null>(null),
    [shipping, setShipping] = useState("0.00"),
    [tax, setTax] = useState("0.00");
  /** load reloads the protected gemstone catalogue and order-management data. */
  async function load() {
    const next = await api<Data>("/admin/shop");
    setData(next);
    return next;
  }
  useEffect(() => {
    void load()
      .then((d) => {
        setSettings(d.settings);
        setShipping(rupeeInput(d.settings.shipping_fee));
        setTax(rupeeInput(d.settings.tax_bps));
      })
      .catch((e) => setError(e.message));
  }, []);
  /** edit copies a product into the editor and converts stored paise/weight into exact input strings. */
  function edit(p: Gemstone) {
    setProduct({ ...p });
    setPrice(rupeeInput(p.price));
    setWeight(carats(p.weight_milli_carat));
    setNotice("");
    setError("");
  }
  /** run runs a catalogue operation with shared busy, success and validation-error handling. */
  async function run(
    fn: () => Promise<unknown>,
    message = "Saved successfully",
  ) {
    setBusy(true);
    setError("");
    setNotice("");
    try {
      await fn();
      await load();
      setNotice(t(message));
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  /** saveProduct parses exact price/weight inputs and saves the validated product version. */
  async function saveProduct(e: FormEvent) {
    e.preventDefault();
    await run(async () => {
      const paise = parseRupees(price);
      if (paise === null)
        throw new Error(
          t("Enter a valid price with up to two decimal places."),
        );
      const p = await api<Gemstone>(
        "/admin/shop/products",
        "POST",
        { ...product, price: paise, weight_milli_carat: parseCarats(weight) },
        csrf,
      );
      edit(p);
    });
  }
  /** upload uploads a selected gemstone photo and stores its returned image reference in the editor. */
  async function upload(file?: File) {
    if (!file) return;
    await run(async () => {
      const form = new FormData();
      form.append("image", file);
      const response = await fetch("/api/admin/shop/images", {
        method: "POST",
        credentials: "same-origin",
        headers: { "X-CSRF-Token": csrf },
        body: form,
      });
      const out = await response.json();
      if (!response.ok) throw new Error(out.error || "Photo upload failed.");
      setProduct((p) => ({ ...p, image: out.image }));
    }, "Photo uploaded. Save the product to apply it.");
  }
  /** saveSettings submits gemstone shipping, tax and policy settings with administrator CSRF protection. */
  async function saveSettings(e: FormEvent) {
    e.preventDefault();
    if (!settings) return;
    await run(async () => {
      const fee = parseRupees(shipping),
        bps = parseRupees(tax);
      if (fee === null || bps === null)
        throw new Error(t("Enter valid shipping and tax amounts."));
      await api(
        "/admin/shop/settings",
        "PUT",
        { ...settings, shipping_fee: fee, tax_bps: bps },
        csrf,
      );
      await onConfig();
    });
  }
  /** field renders a labelled gemstone editor input bound to the requested product property. */
  const field = (
    key: keyof Gemstone,
    label: string,
    options: React.InputHTMLAttributes<HTMLInputElement> = {},
  ) => (
    <label>
      {t(label)}
      <input
        value={String(product[key])}
        onChange={(e) => setProduct({ ...product, [key]: e.target.value })}
        maxLength={200}
        {...options}
      />
    </label>
  );
  return (
    <section className="shop-admin">
      <div className="shop-admin-toolbar">
        <h2>
          <Gem size={24} /> {t("Gemstone shop")}
        </h2>
        <div className="inline-actions">
          <a
            className="button secondary"
            href="/shop"
            target="_blank"
            rel="noreferrer"
          >
            {t("Open shop")}
          </a>
          <Button
            className="secondary"
            busy={busy}
            onClick={() => void run(load, "Collection refreshed")}
          >
            <RefreshCw size={16} />
            {t("Refresh")}
          </Button>
        </div>
      </div>
      <div className="shop-admin-tabs">
        {[
          ["products", "Products"],
          ["orders", "Stone orders"],
          ["settings", "Shop settings"],
        ].map(([id, label]) => (
          <button
            key={id}
            aria-pressed={tab === id}
            onClick={() => {
              setTab(id);
              setError("");
              setNotice("");
            }}
          >
            {t(label)}
          </button>
        ))}
      </div>
      <ErrorBox error={error} />
      {notice && (
        <p className="success" role="status">
          {notice}
        </p>
      )}
      {!data && !error && <p role="status">{t("Loading shop...")}</p>}
      {data && tab === "products" && (
        <>
          <p className="shop-admin-note">
            {t(
              "Add actual stones, set their price per piece and publish when ready. Preview illustrations are not inventory or certificates.",
            )}
          </p>
          <div className="shop-admin-layout">
            <div className="shop-product-list">
              <label className="admin-product-search">{t("Find a product")}<input type="search" value={search} onChange={event => setSearch(event.target.value)} placeholder={t("Name, stone type or product ID")} /></label>
              <button
                onClick={() => edit({ ...blank })}
                className={!product.id ? "active" : ""}
              >
                <span>
                  <strong>{t("Add a stone")}</strong>
                  <small>{t("Start with a draft")}</small>
                </span>
                <Plus size={19} />
              </button>
              {data.products.filter(p => [p.name, p.hindi_name, p.kind, p.id].join(" ").toLowerCase().includes(search.toLowerCase())).map((p) => (
                <button
                  key={p.id}
                  className={product.id === p.id ? "active" : ""}
                  onClick={() => edit(p)}
                >
                  <span>
                    <strong>{p.name}</strong>
                    <small>
                      {p.sample ? (
                        t("Availability pending")
                      ) : (
                        <>
                          {t(p.status)} · {p.stock} {t("stock")} / {p.available}{" "}
                          {t("available")}
                        </>
                      )}
                    </small>
                  </span>
                  <span>{money(p.price)}</span>
                </button>
              ))}
              {!data.products.length && (
                <p>{t("Your catalogue is empty. Add the first stone here.")}</p>
              )}
              {!!data.products.length && !data.products.some(p => [p.name, p.hindi_name, p.kind, p.id].join(" ").toLowerCase().includes(search.toLowerCase())) && <p>{t("No products match this search.")}</p>}
            </div>
            <form className="shop-admin-form" onSubmit={saveProduct}>
              <h3>{t(product.id ? "Edit stone" : "New stone")}</h3>
              <label className="shop-stock-toggle">
                <input
                  type="checkbox"
                  checked={!!product.sample}
                  onChange={(event) =>
                    setProduct({ ...product, sample: event.target.checked })
                  }
                />
                {t("Require stock confirmation before checkout")}
              </label>
              <div className="shop-admin-photo">
                <GemstoneArt key={product.image} product={product} />
                <div>
                  <label>
                    <Upload size={17} />
                    {t("Product photograph")}
                    <input
                      aria-label={t("Product photograph")}
                      type="file"
                      accept="image/png,image/jpeg"
                      disabled={busy}
                      onChange={(e) => void upload(e.target.files?.[0])}
                    />
                  </label>
                  <small className="muted">
                    {t(
                      "JPEG or PNG, up to 5 MB. Use a photo of the actual stone.",
                    )}
                  </small>
                  {product.image && (
                    <button
                      className="small-link"
                      type="button"
                      onClick={() => setProduct({ ...product, image: "" })}
                    >
                      {t("Remove photo")}
                    </button>
                  )}
                </div>
              </div>
              <div className="shop-form-grid">
                <label>
                  {t("Name (English)")}
                  <input
                    required
                    minLength={2}
                    maxLength={150}
                    value={product.name}
                    onChange={(e) =>
                      setProduct({
                        ...product,
                        name: e.target.value,
                        slug:
                          !product.id &&
                          (!product.slug ||
                            product.slug === slugify(product.name))
                            ? slugify(e.target.value)
                            : product.slug,
                      })
                    }
                  />
                </label>
                {field("hindi_name", "Name (Hindi)", { maxLength: 250 })}
                {field("slug", "Product URL slug", {
                  required: true,
                  pattern: "[a-z0-9]+(?:-[a-z0-9]+)*",
                  maxLength: 100,
                })}
                <label>
                  {t("Category")}
                  <select
                    value={product.kind}
                    onChange={(e) =>
                      setProduct({ ...product, kind: e.target.value })
                    }
                  >
                    <option value="precious">{t("Precious")}</option>
                    <option value="semi-precious">{t("Semi-precious")}</option>
                    <option value="spiritual">{t("Spiritual stones")}</option>
                  </select>
                </label>
                <label>
                  {t("Price per piece (INR)")}
                  <input
                    required
                    inputMode="decimal"
                    value={price}
                    onChange={(e) => setPrice(e.target.value)}
                  />
                </label>
                <label>
                  {t("Weight (carats)")}
                  <input
                    required
                    inputMode="decimal"
                    value={weight}
                    onChange={(e) => setWeight(e.target.value)}
                  />
                </label>
                <label>
                  {t("Unsold stock (pieces)")}
                  <input
                    type="number"
                    required
                    min="0"
                    max="10000"
                    value={product.stock}
                    onChange={(e) =>
                      setProduct({ ...product, stock: Number(e.target.value) })
                    }
                  />
                </label>
                <label>
                  {t("Product visibility")}
                  <select
                    value={product.status}
                    onChange={(e) =>
                      setProduct({ ...product, status: e.target.value })
                    }
                  >
                    <option value="draft">{t("Draft")}</option>
                    <option value="published">{t("Published")}</option>
                    <option value="archived">{t("Archived")}</option>
                  </select>
                </label>
              </div>
              <label>
                {t("Description (English)")}
                <textarea
                  rows={4}
                  maxLength={6000}
                  value={product.description}
                  onChange={(e) =>
                    setProduct({ ...product, description: e.target.value })
                  }
                />
              </label>
              <label>
                {t("Description (Hindi)")}
                <textarea
                  rows={4}
                  maxLength={10000}
                  value={product.hindi_description}
                  onChange={(e) =>
                    setProduct({
                      ...product,
                      hindi_description: e.target.value,
                    })
                  }
                />
              </label>
              <details className="shop-guide-editor">
                <summary>{t("Stone guide (English & Hindi)")}</summary>
                {[
                  ["group", "Collection group"],
                  ["planet", "Traditional association"],
                  ["mineral", "Mineral"],
                  ["hardness", "Mohs hardness"],
                  ["color", "Colour"],
                  ["tradition", "Astrological tradition"],
                  ["wearing", "Choosing how to wear it"],
                  ["care", "Everyday care"],
                  ["buying", "Before you buy"],
                ].map(([key, label]) => (
                  <div className="shop-form-grid" key={key}>
                    {(["en", "hi"] as const).map((lang) => (
                      <label key={lang}>
                        {t(label)} ({lang === "en" ? "English" : "हिन्दी"})
                        <textarea
                          rows={
                            key === "care" ||
                            key === "buying" ||
                            key === "tradition"
                              ? 4
                              : 2
                          }
                          maxLength={lang === "en" ? 3000 : 6000}
                          value={product.details?.[key]?.[lang] || ""}
                          onChange={(event) =>
                            setProduct({
                              ...product,
                              details: {
                                ...product.details,
                                [key]: {
                                  en: product.details?.[key]?.en || "",
                                  hi: product.details?.[key]?.hi || "",
                                  [lang]: event.target.value,
                                },
                              },
                            })
                          }
                        />
                      </label>
                    ))}
                  </div>
                ))}
              </details>
              <h3>{t("Stone disclosures")}</h3>
              <div className="shop-form-grid">
                <label>
                  {t("Material")}
                  <select
                    value={product.material}
                    onChange={(e) =>
                      setProduct({ ...product, material: e.target.value })
                    }
                  >
                    <option value="not-specified">{t("Not specified")}</option>
                    <option value="natural">{t("Natural")}</option>
                    <option value="lab-grown">{t("Lab-grown")}</option>
                  </select>
                </label>
                {field("treatment", "Treatment disclosure")}
                {field("origin", "Origin")}
                {field("shape", "Shape")}
                {field("dimensions", "Dimensions")}
              </div>
              <h3>{t("Laboratory report (optional)")}</h3>
              <div className="shop-form-grid">
                {field("certificate_lab", "Laboratory name")}
                {field("certificate_no", "Report number")}
                {field("certificate_url", "Certificate report URL", {
                  type: "url",
                  maxLength: 1000,
                })}
              </div>
              <p className="shop-admin-note">
                {t(
                  "Only enter verifiable details for this stone. Leave certificate fields empty when no report is available.",
                )}
              </p>
              <Button type="submit" busy={busy}>
                <Save size={17} />
                {t("Save stone")}
              </Button>
            </form>
          </div>
        </>
      )}
      {data && tab === "orders" && (
        <>
          <p className="shop-admin-note">
            {t(
              "Confirm dispatch only after the payment is verified. Test orders are labelled separately.",
            )}
          </p>
          {data.orders.length === 0 && (
            <div className="shop-empty">{t("No gemstone orders yet.")}</div>
          )}
          {data.orders.map((o) => (
            <div key={o.id}>
              <ShopOrders orders={[o]} />
              {o.status === "paid" && o.refunded === 0 && (
                <Shipment
                  order={o}
                  busy={busy}
                  save={(body) =>
                    run(
                      () =>
                        api(
                          `/admin/shop/orders/${o.id}/fulfilment`,
                          "POST",
                          body,
                          csrf,
                        ),
                      "Shipment updated",
                    )
                  }
                />
              )}
            </div>
          ))}
        </>
      )}
      {data && settings && tab === "settings" && (
        <form
          className="shop-admin-form shop-admin-settings"
          onSubmit={saveSettings}
        >
          <h3>{t("Selling & delivery")}</h3>
          <label className="shop-policy-consent">
            <input
              type="checkbox"
              checked={settings.enabled}
              onChange={(e) =>
                setSettings({ ...settings, enabled: e.target.checked })
              }
            />
            {t("Enable online gemstone checkout")}
          </label>
          <p className="shop-admin-note">
            {t(
              "Payments use the existing Razorpay configuration. Checkout requires delivery and return policies plus a gemstone tax treatment.",
            )}
          </p>
          <div className="shop-form-grid">
            <label>
              {t("Delivery charge per order (INR)")}
              <input
                required
                inputMode="decimal"
                value={shipping}
                onChange={(e) => setShipping(e.target.value)}
              />
            </label>
            <label>
              {t("Gemstone tax treatment")}
              <select
                value={settings.tax_mode || "unconfigured"}
                onChange={(e) =>
                  setSettings({ ...settings, tax_mode: e.target.value })
                }
              >
                <option value="unconfigured">{t("Not configured")}</option>
                <option value="inclusive">{t("Tax included in prices")}</option>
                <option value="exclusive">{t("Add tax at checkout")}</option>
                <option value="none">{t("No tax")}</option>
              </select>
            </label>
            <label>
              {t("Tax rate (%)")}
              <input
                required
                inputMode="decimal"
                value={tax}
                onChange={(e) => setTax(e.target.value)}
              />
            </label>
          </div>
          <label>
            {t("Delivery policy")}
            <textarea
              rows={4}
              maxLength={6000}
              value={settings.delivery_policy}
              onChange={(e) =>
                setSettings({ ...settings, delivery_policy: e.target.value })
              }
            />
          </label>
          <label>
            {t("Return policy")}
            <textarea
              rows={4}
              maxLength={6000}
              value={settings.return_policy}
              onChange={(e) =>
                setSettings({ ...settings, return_policy: e.target.value })
              }
            />
          </label>
          <p className="shop-admin-note">
            {t(
              "Shipping is currently for Indian addresses. The configured rate applies to stones and delivery; confirm the correct treatment before enabling sales.",
            )}
          </p>
          <Button busy={busy}>
            <Save size={17} />
            {t("Save shop settings")}
          </Button>
        </form>
      )}
    </section>
  );
}

/** Shipment collects an order's fulfilment status and validated carrier/tracking information. */
function Shipment({
  order,
  busy,
  save,
}: {
  order: Order;
  busy: boolean;
  save: (body: unknown) => Promise<unknown>;
}) {
  const [courier, setCourier] = useState(""),
    [tracking, setTracking] = useState(""),
    [url, setURL] = useState("");
  const g = order.gemstone!;
  if (!["processing", "shipped", "needs_resolution"].includes(g.fulfilment))
    return null;
  return (
    <form
      className="shop-admin-form"
      data-shop-order={order.id}
      onSubmit={(e) => {
        e.preventDefault();
        void save({
          version: g.shipment_version,
          status:
            g.fulfilment === "needs_resolution"
              ? "processing"
              : g.fulfilment === "processing"
                ? "shipped"
                : "delivered",
          courier,
          tracking_number: tracking,
          tracking_url: url,
        });
      }}
    >
      <h3>{t("Update fulfilment")}</h3>
      {g.fulfilment === "processing" && (
        <div className="shop-form-grid">
          <label>
            {t("Courier")}
            <input
              required
              maxLength={120}
              value={courier}
              onChange={(e) => setCourier(e.target.value)}
            />
          </label>
          <label>
            {t("Tracking number")}
            <input
              required
              maxLength={150}
              value={tracking}
              onChange={(e) => setTracking(e.target.value)}
            />
          </label>
          <label>
            {t("Tracking URL (optional)")}
            <input
              type="url"
              maxLength={1000}
              value={url}
              onChange={(e) => setURL(e.target.value)}
            />
          </label>
        </div>
      )}
      {g.fulfilment === "needs_resolution" && (
        <p className="shop-admin-note">
          {t(
            "Restore sufficient stock before resolving this paid order, or use the existing refund controls.",
          )}
        </p>
      )}
      <Button busy={busy}>
        {t(
          g.fulfilment === "needs_resolution"
            ? "Resolve stock & prepare order"
            : g.fulfilment === "processing"
              ? "Mark as dispatched"
              : "Mark as delivered",
        )}
      </Button>
    </form>
  );
}
