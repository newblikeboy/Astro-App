import { useEffect, useRef, useState } from "react";
import {
  ArrowLeft,
  ArrowRight,
  Check,
  Gem,
  Menu,
  Package,
  Search,
  ShoppingBag,
  CreditCard,
  Minus,
  Plus,
  Trash2,
  ArrowUpRight,
  SlidersHorizontal,
  Truck,
  UserRound,
  X,
} from "lucide-react";
import type { FormEvent, ReactNode } from "react";
import { api, APIError, money } from "./api";
import { t, useLanguage } from "./i18n";
import LanguageSelector from "./LanguageSelector";
import Brand from "./Brand";
import PaymentView from "./PaymentView";
import ShopOrders from "./ShopOrders";
import GemstoneArt from "./GemstoneArt";
import GemstoneDetail from "./GemstoneDetail";
import AddToCartButton from "./AddToCartButton";
import CartLink from "./CartLink";
import { useShopBag } from "./ShopBag";
import { Button, ErrorBox } from "./UI";
import { followRoute, loginPath, navigate, useAppLocation } from "./routing";
import { carats } from "./shop-types";
import type { DeliveryAddress, Gemstone, ShopCatalog } from "./shop-types";
import type { Account, Config, Order } from "./types";
import "./shop.css";
import "./checkout.css";

/** Link renders an application link that preserves normal browser navigation gestures. */
function Link({
  to,
  children,
  className = "",
}: {
  to: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <a className={className} href={to} onClick={(e) => followRoute(e, to)}>
      {children}
    </a>
  );
}
/** stonePath builds a product-detail route from its validated catalogue slug. */
const stonePath = (p: Gemstone) => "/shop/stones/" + p.slug;
const emptyAddress: DeliveryAddress = {
  name: "",
  phone: "",
  line1: "",
  line2: "",
  city: "",
  state: "",
  pin: "",
  country: "IN",
};
/** ShopApp coordinates gemstone browsing, product details, cart quantities, delivery, checkout and customer orders. */
export default function ShopApp() {
  const { language } = useLanguage();
  const route = useAppLocation();
  const [catalog, setCatalog] = useState<ShopCatalog | null>(null),
    [config, setConfig] = useState<Config | null>(null),
    [account, setAccount] = useState<Account | null>(null),
    [resolved, setResolved] = useState(false);
  const [error, setError] = useState(""),
    [notice, setNotice] = useState(""),
    [busy, setBusy] = useState(false),
    [menu, setMenu] = useState(false);
  const { bag, setBag } = useShopBag();
  const [search, setSearch] = useState(""),
    [kind, setKind] = useState("all"),
    [collection, setCollection] = useState("all"),
    [sort, setSort] = useState("name"),
    [inStock, setInStock] = useState(false);
  const [address, setAddress] = useState<DeliveryAddress>(emptyAddress),
    [order, setOrder] = useState<Order | null>(null);
  const checkoutKey = useRef(
    sessionStorage.getItem("m4y-shop-checkout-key") || crypto.randomUUID(),
  );
  /** name selects the product's Hindi name when available, otherwise its catalogue name. */
  const name = (p: Gemstone) =>
    language === "hi" && p.hindi_name ? p.hindi_name : p.name;
  /** refreshCatalog reloads server catalogue prices and currently available quantities. */
  async function refreshCatalog() {
    setCatalog(await api<ShopCatalog>("/shop"));
  }
  /** refreshAccount reloads the customer session/purchases and handles an unauthenticated shop visitor. */
  async function refreshAccount() {
    try {
      const a = await api<Account>("/me");
      setAccount(a);
      return a;
    } catch (e) {
      if (e instanceof APIError && e.status === 401) {
        setAccount(null);
        return null;
      }
      throw e;
    } finally {
      setResolved(true);
    }
  }
  /** load loads the catalogue and account data needed by the current shop route. */
  async function load() {
    setError("");
    try {
      await Promise.all([
        refreshCatalog(),
        api<Config>("/config").then(setConfig),
        refreshAccount(),
      ]);
    } catch (e) {
      setError((e as Error).message);
    }
  }
  useEffect(() => {
    void load();
  }, []);
  useEffect(() => {
    setMenu(false);
    setError("");
  }, [route.pathname]);
  const isBag = route.pathname === "/shop/bag",
    isOrders = route.pathname === "/shop/orders";
  const slug = route.pathname.startsWith("/shop/stones/")
    ? route.pathname.slice(13)
    : "";
  const product = catalog?.products.find((p) => p.slug === slug);
  const products = (catalog?.products || [])
    .filter(
      (p) =>
        (kind === "all" || p.kind === kind) &&
        (collection === "all" ||
          (p.details?.group?.en === "Navratna"
            ? "navratna"
            : "alternatives") === collection) &&
        (!inStock || (!p.sample && p.available > 0)) &&
        [
          p.name,
          p.hindi_name,
          p.description,
          p.details?.planet?.en,
          p.details?.planet?.hi,
        ]
          .join(" ")
          .toLowerCase()
          .includes(search.toLowerCase()),
    )
    .sort((a, b) =>
      sort === "low"
        ? a.price - b.price
        : sort === "high"
          ? b.price - a.price
          : a.name.localeCompare(b.name),
    );
  const bagRows = bag.map((line) => ({
    line,
    product: catalog?.products.find((p) => p.id === line.product_id),
  }));
  const hasSamples = bagRows.some((row) => row.product?.sample);
  const itemCount = bag.reduce((count, line) => count + line.quantity, 0);
  const checkoutStep = order
    ? 2
    : !hasSamples && catalog?.checkout_available && account
      ? 1
      : 0;
  /** changeBagQuantity keeps edited cart quantities within stock and the existing ten-per-product limit. */
  function changeBagQuantity(
    productID: string,
    quantity: number,
    available: number,
  ) {
    if (
      !Number.isInteger(quantity) ||
      quantity < 1 ||
      quantity > Math.min(available, 10)
    )
      return;
    setBag((previous) =>
      previous.map((line) =>
        line.product_id === productID ? { ...line, quantity } : line,
      ),
    );
  }
  const subtotal = bagRows.reduce(
    (sum, row) => sum + (row.product?.price || 0) * row.line.quantity,
    0,
  );
  const shipping = catalog?.settings.shipping_fee || 0;
  // This is an estimate only; the saved server order is reviewed before payment.
  const exclusive = catalog?.settings.tax_mode === "exclusive";
  const estimatedTax = exclusive
    ? Math.floor(
        ((subtotal + shipping) * (catalog?.settings.tax_bps || 0) + 5000) /
          10000,
      )
    : 0;
  /** checkout submits cart product IDs/quantities and delivery details for server-priced order creation. */
  async function checkout(e: FormEvent) {
    e.preventDefault();
    if (hasSamples)
      return setError(
        t(
          "Sample listings are for cart estimates. The seller must confirm stock and prices before checkout.",
        ),
      );
    if (!account) return navigate(loginPath("/shop/bag"));
    setBusy(true);
    setError("");
    try {
      sessionStorage.setItem("m4y-shop-checkout-key", checkoutKey.current);
      setOrder(
        await api<Order>(
          "/shop/checkout",
          "POST",
          { items: bag, address, idempotency_key: checkoutKey.current },
          account.csrf,
        ),
      );
      await refreshAccount();
    } catch (e) {
      setError((e as Error).message);
      await refreshCatalog().catch(() => {});
    } finally {
      setBusy(false);
    }
  }
  /** cancel cancels an owned unpaid order reservation and reloads account/catalogue availability. */
  async function cancel(o: Order) {
    if (!account) return;
    setBusy(true);
    setError("");
    try {
      await api(`/shop/orders/${o.id}/cancel`, "POST", {}, account.csrf);
      setOrder(null);
      checkoutKey.current = crypto.randomUUID();
      sessionStorage.removeItem("m4y-shop-checkout-key");
      await Promise.all([refreshAccount(), refreshCatalog()]);
      setNotice(t("Reservation cancelled"));
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  /** paid refreshes confirmed purchases and stock, then clears the completed browser cart. */
  async function paid() {
    await refreshAccount();
    await refreshCatalog();
    setBag([]);
    setOrder(null);
    sessionStorage.removeItem("m4y-shop-checkout-key");
    checkoutKey.current = crypto.randomUUID();
    setNotice(t("Payment verified. View your order for delivery updates."));
    navigate("/shop/orders");
  }
  /** field renders a labelled delivery-address input bound to the selected address property. */
  const field = (
    key: keyof DeliveryAddress,
    label: string,
    extra: React.InputHTMLAttributes<HTMLInputElement> = {},
  ) => (
    <label>
      {t(label)}
      <input
        value={address[key]}
        onChange={(e) => setAddress({ ...address, [key]: e.target.value })}
        required={key !== "line2"}
        maxLength={key === "line1" || key === "line2" ? 250 : 120}
        {...extra}
      />
    </label>
  );
  return (
    <div className="shop-app">
      <a className="skip" href="#shop-main">
        {t("Skip to collection")}
      </a>
      <header className="shop-header">
        <Link to="/" className="brand">
          <Brand subtitle={t("Ratna collection")} />
        </Link>
        <nav
          className={menu ? "is-open" : ""}
          aria-label={t("Shop navigation")}
        >
          <Link to="/">{t("Home")}</Link>
          <Link to="/shop">{t("The collection")}</Link>
          <Link to="/shop/orders">{t("My stone orders")}</Link>
        </nav>
        <div className="shop-header-actions">
          <LanguageSelector />
          <Link
            to={account ? "/account" : loginPath(route.pathname)}
            className="icon-button"
          >
            <UserRound size={20} />
            <span className="sr-only">{t("My account")}</span>
          </Link>
          <CartLink />
          <button
            className="icon-button shop-menu"
            aria-expanded={menu}
            aria-label={t("Shop navigation")}
            onClick={() => setMenu(!menu)}
          >
            {menu ? <X /> : <Menu />}
          </button>
        </div>
      </header>
      <main id="shop-main">
        {notice && (
          <div className="shop-notice" role="status">
            <Check size={18} />
            <span>{notice}</span>
            <button
              className="icon-button"
              onClick={() => setNotice("")}
              aria-label={t("Dismiss notice")}
            >
              <X size={16} />
            </button>
          </div>
        )}
        {error && (
          <div className="shop-container">
            <ErrorBox error={error} />
            {!catalog && (
              <Button onClick={() => void load()}>{t("Try again")}</Button>
            )}
          </div>
        )}
        {!catalog && !error && (
          <div className="shop-empty" role="status">
            {t("Opening the collection...")}
          </div>
        )}
        {catalog && route.pathname === "/shop" && (
          <>
            <section className="shop-hero">
              <div className="shop-container shop-hero-inner">
                <div>
                  <span className="shop-kicker">
                    <Gem size={16} aria-hidden="true" />
                    {t("The Ratna collection")}
                  </span>
                  <h1>{t("Choose your gemstone")}</h1>
                  <p>
                    {t(
                      "Explore gemstones and meaningful stones. Compare their details, choose your favourites, and order directly.",
                    )}
                  </p>
                </div>
                <div className="shop-hero-art" aria-hidden="true">
                  <span className="shop-orbit shop-orbit-one" />
                  <span className="shop-orbit shop-orbit-two" />
                  <GemstoneArt
                    product={
                      catalog.products.find((p) => p.name === "Ruby") ||
                      catalog.products[0] ||
                      ({ name: "Ruby", image: "" } as Gemstone)
                    }
                  />
                  <span className="shop-art-caption">
                    {t("Colour. Character. Connection.")}
                  </span>
                </div>
              </div>
            </section>
            <section className="shop-container shop-collection" id="collection">
              <div className="shop-section-heading">
                <div>
                  <span className="shop-kicker">
                    {t("Chosen with intention")}
                  </span>
                  <h2>{t("Find your stone")}</h2>
                </div>
                <span className="muted">
                  {products.length} {t("stones to explore")}
                </span>
              </div>
              {catalog.preview && (
                <p className="shop-preview">
                  <Gem size={17} />
                  {t(
                    "The collection is being prepared. Product details will appear when the seller publishes their stones.",
                  )}
                </p>
              )}
              <div className="shop-filters">
                <label className="shop-collection-select">
                  {t("Stone collection")}
                  <select
                    value={collection}
                    onChange={(event) => setCollection(event.target.value)}
                  >
                    <option value="all">{t("All stones")}</option>
                    <option value="navratna">{t("Navratna")}</option>
                    <option value="alternatives">{t("Alternatives")}</option>
                  </select>
                </label>
                <div className="shop-filter-tools">
                  <label className="shop-search">
                    <Search size={18} />
                    <span className="sr-only">{t("Search stones")}</span>
                    <input
                      type="search"
                      value={search}
                      placeholder={t("Search stones")}
                      onChange={(e) => setSearch(e.target.value)}
                    />
                  </label>
                  <label className="shop-sort">
                    <SlidersHorizontal size={16} />
                    <span className="sr-only">{t("Sort by")}</span>
                    <select
                      value={sort}
                      onChange={(e) => setSort(e.target.value)}
                    >
                      <option value="name">{t("Name A–Z")}</option>
                      <option value="low">{t("Price: low to high")}</option>
                      <option value="high">{t("Price: high to low")}</option>
                    </select>
                  </label>
                </div>
                <details className="shop-extra-filters">
                  <summary>
                    <SlidersHorizontal size={15} />
                    {t("More filters")}
                    {(kind !== "all" || inStock) && <span>{t("Active")}</span>}
                  </summary>
                  <div className="shop-extra-filter-content">
                    <div
                      className="shop-category-tabs"
                      aria-label={t("Stone category")}
                    >
                      {[
                        ["all", "All stones"],
                        ["precious", "Precious"],
                        ["semi-precious", "Semi-precious"],
                        ["spiritual", "Spiritual stones"],
                      ].map(([id, label]) => (
                        <button
                          key={id}
                          aria-pressed={kind === id}
                          onClick={() => setKind(id)}
                        >
                          {t(label)}
                        </button>
                      ))}
                    </div>
                    <label className="shop-stock-toggle">
                      <input
                        type="checkbox"
                        checked={inStock}
                        onChange={(e) => setInStock(e.target.checked)}
                      />
                      {t("In stock only")}
                    </label>{" "}
                  </div>
                </details>
                {(search ||
                  kind !== "all" ||
                  collection !== "all" ||
                  inStock) && (
                  <button
                    className="text-button shop-clear-filters"
                    onClick={() => {
                      setSearch("");
                      setKind("all");
                      setCollection("all");
                      setInStock(false);
                    }}
                  >
                    {t("Clear filters")}
                  </button>
                )}
              </div>
              <div className="shop-grid">
                {products.map((p) => (
                  <article className="gem-card" key={p.id}>
                    <Link to={stonePath(p)} className="gem-card-visual">
                      <GemstoneArt product={p} />
                      <span className="gem-card-label">
                        {p.sample
                          ? t("Availability pending")
                          : p.preview
                            ? t("Preview")
                            : p.available > 0
                              ? t("Available")
                              : t("Out of stock")}
                      </span>
                    </Link>
                    <div className="gem-card-copy">
                      <span className="shop-kicker">
                        {t(
                          p.kind === "semi-precious"
                            ? "Semi-precious"
                            : p.kind === "spiritual"
                              ? "Spiritual stone"
                              : "Precious stone",
                        )}
                      </span>
                      <h3>
                        <Link to={stonePath(p)}>{name(p)}</Link>
                      </h3>
                      <p>
                        {p.preview
                          ? t("Product details coming soon")
                          : [
                              carats(p.weight_milli_carat) + " " + t("ct"),
                              p.shape || p.origin,
                            ]
                              .filter(Boolean)
                              .join(" · ")}
                      </p>
                      <div className="gem-card-bottom">
                        <span>
                          {p.preview ? t("Collection preview") : money(p.price)}
                          {!p.preview && <small>{t("per piece")}</small>}
                        </span>
                        <Link to={stonePath(p)} className="gem-card-arrow">
                          <ArrowRight size={20} />
                          <span className="sr-only">
                            {t("View stone")} {name(p)}
                          </span>
                        </Link>
                      </div>
                      {!p.preview && <AddToCartButton product={p} />}
                    </div>
                  </article>
                ))}
              </div>
              {products.length === 0 && (
                <div className="shop-empty">
                  <Gem size={38} />
                  <h3>
                    {t(
                      catalog.products.length
                        ? "No stones match your search"
                        : "The collection is being prepared",
                    )}
                  </h3>
                  <p>
                    {t(
                      catalog.products.length
                        ? "Try another name or category."
                        : "The seller will publish individual stones here with their prices and details.",
                    )}
                  </p>
                </div>
              )}
            </section>
          </>
        )}
        {catalog && slug && product && (
          <GemstoneDetail
            key={product.id}
            product={product}
            catalog={catalog}
          />
        )}
        {catalog && isBag && (
          <div className="shop-container shop-bag checkout-page">
            <Link to="/shop" className="shop-back">
              <ArrowLeft size={16} />
              {t("Continue exploring")}
            </Link>
            <div className="checkout-heading">
              <div>
                <span className="shop-kicker">{t("The Ratna collection")}</span>
                <h1>{t("Shopping bag")}</h1>
                <p>
                  {t("Review your stones, then arrange delivery and payment.")}
                </p>
              </div>
              <ol
                className="checkout-progress"
                aria-label={t("Checkout progress")}
              >
                {[
                  ["Bag", ShoppingBag],
                  ["Delivery", Truck],
                  ["Payment", CreditCard],
                ].map(([label, Icon], index) => {
                  const StepIcon = Icon as typeof ShoppingBag;
                  return (
                    <li
                      key={label as string}
                      className={index <= checkoutStep ? "is-active" : ""}
                      aria-current={index === checkoutStep ? "step" : undefined}
                    >
                      <span>
                        {index < checkoutStep ? (
                          <Check size={15} />
                        ) : (
                          <StepIcon size={16} />
                        )}
                      </span>
                      <small>{t(label as string)}</small>
                    </li>
                  );
                })}
              </ol>
            </div>
            {order && config && account ? (
              <div className="shop-checkout-review">
                <h2>{t("Review your reserved order")}</h2>
                <ShopOrders orders={[order]} />
                <div className="checkout-payment-card">
                  <div className="checkout-card-heading">
                    <h2>
                      <CreditCard size={20} />
                      {t("Payment")}
                    </h2>
                    <span>{t("Step 3 of 3")}</span>
                  </div>
                  <PaymentView
                    order={order}
                    config={config}
                    csrf={account.csrf}
                    onPaid={paid}
                  />
                  <Button
                    className="secondary"
                    busy={busy}
                    onClick={() => void cancel(order)}
                  >
                    {t("Cancel reservation")}
                  </Button>
                </div>
              </div>
            ) : bag.length === 0 ? (
              <div className="shop-empty">
                <ShoppingBag size={38} />
                <h2>{t("A little room for something meaningful")}</h2>
                <p>
                  {t("Explore the collection and add a stone to your bag.")}
                </p>
                <Link to="/shop" className="button">
                  {t("Explore the collection")}
                  <ArrowRight size={17} />
                </Link>
              </div>
            ) : (
              <div className="shop-checkout-grid">
                <div className="checkout-main">
                  <section
                    className="checkout-items"
                    aria-labelledby="checkout-items-title"
                  >
                    <div className="checkout-card-heading">
                      <h2 id="checkout-items-title">{t("Selected stones")}</h2>
                      <span>{t("Items: {v1}", { v1: itemCount })}</span>
                    </div>
                    <div className="shop-bag-lines">
                      {bagRows.map(({ line, product: p }) => (
                        <article key={line.product_id}>
                          {p && (
                            <Link to={stonePath(p)}>
                              <GemstoneArt product={p} />
                            </Link>
                          )}
                          <div className="checkout-line-info">
                            <h3>
                              {p ? (
                                <Link to={stonePath(p)}>{name(p)}</Link>
                              ) : (
                                t("Unavailable stone")
                              )}
                            </h3>
                            <p>
                              {p
                                ? carats(p.weight_milli_carat) + " " + t("ct")
                                : t("Remove this item to continue.")}
                            </p>
                            {p?.sample && (
                              <span className="checkout-sample-tag">
                                {t("Availability pending")}
                              </span>
                            )}
                            <div className="checkout-line-controls">
                              <div className="checkout-quantity">
                                <button
                                  type="button"
                                  aria-label={
                                    t("Decrease quantity") +
                                    " " +
                                    (p ? name(p) : "")
                                  }
                                  disabled={
                                    line.quantity <= 1 || !p || p.available < 1
                                  }
                                  onClick={() =>
                                    changeBagQuantity(
                                      line.product_id,
                                      line.quantity - 1,
                                      p?.available || 0,
                                    )
                                  }
                                >
                                  <Minus size={13} />
                                </button>
                                <input
                                  aria-label={
                                    t("Quantity") + " " + (p ? name(p) : "")
                                  }
                                  type="number"
                                  min="1"
                                  max={Math.min(p?.available || 1, 10)}
                                  value={line.quantity}
                                  onChange={(e) => {
                                    const q = Number(e.target.value);
                                    changeBagQuantity(
                                      line.product_id,
                                      q,
                                      p?.available || 0,
                                    );
                                  }}
                                />
                                <button
                                  type="button"
                                  aria-label={
                                    t("Increase quantity") +
                                    " " +
                                    (p ? name(p) : "")
                                  }
                                  disabled={
                                    !p ||
                                    line.quantity >= Math.min(p.available, 10)
                                  }
                                  onClick={() =>
                                    changeBagQuantity(
                                      line.product_id,
                                      line.quantity + 1,
                                      p?.available || 0,
                                    )
                                  }
                                >
                                  <Plus size={13} />
                                </button>
                              </div>
                              <button
                                type="button"
                                className="checkout-remove"
                                onClick={() =>
                                  setBag((previous) =>
                                    previous.filter(
                                      (item) =>
                                        item.product_id !== line.product_id,
                                    ),
                                  )
                                }
                              >
                                <Trash2 size={14} />
                                <span>{t("Remove")}</span>
                              </button>
                            </div>
                            {p && line.quantity > p.available && (
                              <p className="error">
                                {t(
                                  "Stock changed. Update the quantity or remove this stone.",
                                )}
                              </p>
                            )}
                          </div>
                          <div className="checkout-line-price">
                            <strong>
                              {p ? money(p.price * line.quantity) : "—"}
                            </strong>
                            {p && (
                              <small>
                                {money(p.price)} {t("per piece")}
                              </small>
                            )}
                          </div>
                        </article>
                      ))}
                    </div>
                    <Link to="/shop" className="checkout-add-more">
                      <Plus size={15} />
                      {t("Explore more stones")}
                      <ArrowUpRight size={15} />
                    </Link>
                  </section>
                  {hasSamples && (
                    <div className="stone-sample-note checkout-estimate-note">
                      <span className="checkout-note-icon">
                        <Gem size={22} />
                      </span>
                      <div>
                        <strong>{t("Stock confirmation")}</strong>
                        <p>
                          {t(
                            "The seller must confirm availability before checkout. The prices shown are the current catalogue prices.",
                          )}
                        </p>
                      </div>
                    </div>
                  )}
                  {!hasSamples && catalog.checkout_available && account && (
                    <form
                      id="checkout-delivery"
                      className="shop-delivery-form"
                      onSubmit={checkout}
                    >
                      <div className="checkout-card-heading">
                        <h2>
                          <Truck size={20} />
                          {t("Delivery details")}
                        </h2>
                        <span>{t("Step 2 of 3")}</span>
                      </div>
                      <p className="muted">
                        {t(
                          "Delivery within India. Please enter a complete address.",
                        )}
                      </p>
                      <div className="shop-form-grid">
                        {field("name", "Recipient name", {
                          autoComplete: "shipping name",
                        })}
                        {field("phone", "Delivery mobile number", {
                          type: "tel",
                          autoComplete: "shipping tel",
                          maxLength: 13,
                          pattern: "(?:\\+91)?[6-9][0-9]{9}",
                        })}
                        {field("line1", "Address line 1", {
                          autoComplete: "shipping address-line1",
                        })}
                        {field("line2", "Address line 2 (optional)", {
                          autoComplete: "shipping address-line2",
                        })}
                        {field("city", "City", {
                          autoComplete: "shipping address-level2",
                        })}
                        {field("state", "State", {
                          autoComplete: "shipping address-level1",
                        })}
                        {field("pin", "PIN code", {
                          autoComplete: "shipping postal-code",
                          inputMode: "numeric",
                          pattern: "[1-9][0-9]{5}",
                          maxLength: 6,
                        })}
                        <label>
                          {t("Country")}
                          <input value={t("India")} readOnly />
                        </label>
                      </div>
                      <label className="shop-policy-consent">
                        <input type="checkbox" required />
                        {t(
                          "I have reviewed the delivery and return policies shown in the order summary.",
                        )}
                      </label>
                    </form>
                  )}
                  {!hasSamples &&
                    !account &&
                    resolved &&
                    catalog.checkout_available && (
                      <div className="shop-signin-card">
                        <UserRound size={25} />
                        <h2>{t("Keep your order connected to you")}</h2>
                        <p>
                          {t(
                            "Log in with your mobile number to complete your purchase.",
                          )}
                        </p>
                      </div>
                    )}
                  {!hasSamples && !catalog.checkout_available && (
                    <div className="shop-signin-card">
                      <ShoppingBag size={25} />
                      <h2>{t("The shop is getting ready")}</h2>
                      <p>
                        {t(
                          "Online purchases open when the seller publishes stock and configures checkout.",
                        )}
                      </p>
                      <Link to="/shop" className="button secondary">
                        {t("Back to collection")}
                      </Link>
                    </div>
                  )}
                </div>
                <aside
                  className="shop-order-summary"
                  aria-label={t("Order summary")}
                >
                  <div className="checkout-summary-heading">
                    <span>
                      <ShoppingBag size={18} />
                      {t("Order summary")}
                    </span>
                    <small>{t("Items: {v1}", { v1: itemCount })}</small>
                  </div>
                  <div className="checkout-summary-body">
                    <dl>
                      <div>
                        <dt>{t("Stones")}</dt>
                        <dd>{money(subtotal)}</dd>
                      </div>
                      <div>
                        <dt>{t("Delivery")}</dt>
                        <dd>
                          {hasSamples ? t("To be confirmed") : money(shipping)}
                        </dd>
                      </div>
                      {exclusive && (
                        <div>
                          <dt>{t("Tax")}</dt>
                          <dd>{money(estimatedTax)}</dd>
                        </div>
                      )}
                      <div className="shop-summary-total">
                        <dt>{t("Estimated total")}</dt>
                        <dd>
                          {money(
                            hasSamples
                              ? subtotal
                              : subtotal + shipping + estimatedTax,
                          )}
                        </dd>
                      </div>
                    </dl>
                    <p className="muted">
                      {t(
                        hasSamples
                          ? "Stone estimate only; delivery and applicable tax will be confirmed before purchase."
                          : catalog.settings.tax_mode === "inclusive"
                            ? "Listed prices include configured tax."
                            : "Final totals are confirmed before payment.",
                      )}
                    </p>
                    {hasSamples ? (
                      <div className="checkout-next-step">
                        <span>{t("Next step")}</span>
                        <p>
                          {t(
                            "The seller will confirm your selected stones before payment opens.",
                          )}
                        </p>
                        <Link to="/shop" className="button">
                          {t("Continue exploring")}
                          <ArrowRight size={16} />
                        </Link>
                      </div>
                    ) : catalog.checkout_available && account ? (
                      <Button
                        form="checkout-delivery"
                        type="submit"
                        className="checkout-summary-action"
                        busy={busy}
                        disabled={bagRows.some(
                          (r) =>
                            !r.product || r.line.quantity > r.product.available,
                        )}
                      >
                        {t("Review order & payment")}
                        <ArrowRight size={16} />
                      </Button>
                    ) : !account && resolved && catalog.checkout_available ? (
                      <Link
                        to={loginPath("/shop/bag")}
                        className="button checkout-summary-action"
                      >
                        {t("Log in to continue")}
                        <ArrowRight size={16} />
                      </Link>
                    ) : null}
                    <details>
                      <summary>{t("Delivery policy")}</summary>
                      <p>
                        {catalog.settings.delivery_policy ||
                          t("Ask the seller for delivery details.")}
                      </p>
                    </details>
                    <details>
                      <summary>{t("Return policy")}</summary>
                      <p>
                        {catalog.settings.return_policy ||
                          t("Ask the seller for return terms before ordering.")}
                      </p>
                    </details>
                    <div className="checkout-summary-footnote">
                      <Truck size={16} />
                      <span>{t("Loose stones · Delivery within India")}</span>
                    </div>
                  </div>
                </aside>
              </div>
            )}
          </div>
        )}
        {catalog && isOrders && (
          <div className="shop-container shop-order-page">
            <div className="shop-section-heading">
              <div>
                <span className="shop-kicker">
                  {t("From our collection to you")}
                </span>
                <h1>{t("My stone orders")}</h1>
              </div>
              <Package size={32} />
            </div>
            {account ? (
              <>
                <button
                  className="button secondary"
                  onClick={() =>
                    void refreshAccount().catch((e) => setError(e.message))
                  }
                >
                  {t("Refresh orders")}
                </button>
                <ShopOrders
                  orders={account.orders}
                  onResume={(o) => {
                    setOrder(o);
                    navigate("/shop/bag");
                  }}
                  onCancel={(o) => void cancel(o)}
                />
              </>
            ) : resolved ? (
              <div className="shop-empty">
                <UserRound size={36} />
                <h2>{t("Your orders, in one place")}</h2>
                <p>
                  {t(
                    "Log in with the same mobile number used for your purchase.",
                  )}
                </p>
                <Link to={loginPath("/shop/orders")} className="button">
                  {t("Log in")}
                </Link>
              </div>
            ) : (
              <p role="status">{t("Checking your session…")}</p>
            )}
          </div>
        )}
        {catalog &&
          route.pathname !== "/shop" &&
          !isBag &&
          !isOrders &&
          (!slug || !product) && (
            <div className="shop-empty">
              <h1>{t("Stone not found")}</h1>
              <Link to="/shop">{t("Back to collection")}</Link>
            </div>
          )}
      </main>
      <footer className="shop-footer">
        <div className="shop-container">
          <div>
            <Link to="/" className="brand">
              <Brand subtitle={t("Ratna collection")} />
            </Link>
            <p>{t("Thoughtful choices. Meaningful connections.")}</p>
          </div>
          <div>
            <Link to="/shop">{t("The collection")}</Link>
            <Link to="/shop/orders">{t("My stone orders")}</Link>
            <Link to="/">{t("Back to website")}</Link>
          </div>
          <small>© {new Date().getFullYear()} Mishra 4 You</small>
        </div>
      </footer>
    </div>
  );
}
