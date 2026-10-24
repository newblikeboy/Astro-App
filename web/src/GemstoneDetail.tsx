import { useEffect, useRef, useState } from "react";
import {
  ArrowLeft,
  ArrowRight,
  Gem,
  ZoomIn,
  X,
  Truck,
  FileCheck2,
  BookOpen,
} from "lucide-react";
import { money } from "./api";
import { t, useLanguage } from "./i18n";
import { followRoute } from "./routing";
import { carats } from "./shop-types";
import type { Gemstone, ShopCatalog } from "./shop-types";
import GemstoneArt from "./GemstoneArt";
import AddToCartButton from "./AddToCartButton";
import "./gemstone-detail.css";

/** GemstoneDetail presents editable stone information, price context, cart controls, care and related products. */
export default function GemstoneDetail({
  product,
  catalog,
}: {
  product: Gemstone;
  catalog: ShopCatalog;
}) {
  const { language } = useLanguage();
  const [quantity, setQuantity] = useState(1);
  const zoom = useRef<HTMLDialogElement>(null);
  const purchaseActions = useRef<HTMLDivElement>(null);
  const [showMobileCart, setShowMobileCart] = useState(false);
  useEffect(() => {
    setShowMobileCart(false);
    const actions = purchaseActions.current;
    if (!actions) return;
    let frame = 0;
    /** updateCartVisibility follows scroll jumps as well as gradual scrolling without duplicate layout reads in a frame. */
    const updateCartVisibility = () => {
      if (frame) return;
      frame = requestAnimationFrame(() => {
        frame = 0;
        setShowMobileCart(actions.getBoundingClientRect().bottom < 0);
      });
    };
    window.addEventListener("scroll", updateCartVisibility, { passive: true });
    window.addEventListener("resize", updateCartVisibility);
    updateCartVisibility();
    return () => {
      window.removeEventListener("scroll", updateCartVisibility);
      window.removeEventListener("resize", updateCartVisibility);
      cancelAnimationFrame(frame);
    };
  }, [product.id, product.preview, language]);
  const name =
    language === "hi" && product.hindi_name ? product.hindi_name : product.name;
  /** detail selects the seller's bilingual guide text with an English fallback. */
  const detail = (key: string) =>
    product.details?.[key]?.[language] || product.details?.[key]?.en || "";
  const description =
    language === "hi" && product.hindi_description
      ? product.hindi_description
      : product.description;
  const related = catalog.products
    .filter((p) => p.id !== product.id)
    .sort(
      (a, b) =>
        Number(b.details?.planet?.en === product.details?.planet?.en) -
        Number(a.details?.planet?.en === product.details?.planet?.en),
    )
    .slice(0, 4);
  const rate =
    product.weight_milli_carat > 0
      ? Math.round((product.price * 1000) / product.weight_milli_carat)
      : 0;
  return (
    <div className="shop-container gemstone-detail">
      <nav className="stone-breadcrumb" aria-label={t("Breadcrumb")}>
        <a href="/shop" onClick={(event) => followRoute(event, "/shop")}>
          <ArrowLeft size={15} />
          {t("The collection")}
        </a>
        <span aria-hidden="true">/</span>
        <span>{name}</span>
      </nav>
      <div className="stone-detail-top">
        <div className="stone-gallery">
          <div className="stone-main-photo">
            <GemstoneArt product={product} priority />
            <button
              className="stone-zoom"
              aria-label={t("Enlarge image")}
              onClick={() => zoom.current?.showModal()}
            >
              <ZoomIn size={20} />
            </button>
          </div>
          <p className="stone-photo-note">
            {t(
              product.image
                ? "Review the photograph and report for this individual stone."
                : "Representative image. Colour and appearance vary between individual stones.",
            )}
          </p>
        </div>
        <div className="stone-buy-panel">
          <div className="stone-heading">
            <span className="shop-kicker">
              <Gem size={16} />
              {detail("group") || t("The Ratna collection")}
            </span>
            <h1>{name}</h1>
            {language === "en" && product.hindi_name && (
              <p className="stone-local-name">{product.hindi_name}</p>
            )}
            {detail("planet") && (
              <span className="stone-planet">
                {t("Traditional association")}: {detail("planet")}
              </span>
            )}
          </div>
          <div className="stone-purchase">
            <p className="stone-summary">
              {description ||
                t("Ask the seller for full details of this stone.")}
            </p>
            <div className="stone-price">
              <strong>{money(product.price)}</strong>
              <span>{t("Price per piece")}</span>
            </div>
            <p className="stone-rate">
              {carats(product.weight_milli_carat)} {t("ct")} · {money(rate)} /{" "}
              {t("ct")} · {t("Loose stone; setting not included")}
            </p>
            {product.sample && (
              <div className="stone-sample-note">
                <strong>{t("Stock confirmation")}</strong>
                <p>
                  {t(
                    "Stock, origin, treatment and certification await confirmation. You can add this stone to your cart; purchase opens after the seller confirms availability.",
                  )}
                </p>
              </div>
            )}
            {!product.preview && (
              <div className="stone-order-controls">
                <label className="stone-quantity">
                  {t("Quantity")}
                  <div>
                    <button
                      aria-label={t("Decrease quantity")}
                      disabled={quantity <= 1}
                      onClick={() => setQuantity(quantity - 1)}
                    >
                      −
                    </button>
                    <input
                      aria-label={t("Quantity")}
                      type="number"
                      min="1"
                      max={Math.min(10, product.available)}
                      value={quantity}
                      onChange={(event) => {
                        const next = Number(event.target.value);
                        if (
                          Number.isInteger(next) &&
                          next >= 1 &&
                          next <= Math.min(10, product.available)
                        )
                          setQuantity(next);
                      }}
                    />
                    <button
                      aria-label={t("Increase quantity")}
                      disabled={quantity >= Math.min(10, product.available)}
                      onClick={() => setQuantity(quantity + 1)}
                    >
                      +
                    </button>
                  </div>
                </label>
                <div className="stone-order-buttons" ref={purchaseActions}>
                  <AddToCartButton product={product} quantity={quantity} />
                  <a
                    className="button secondary stone-view-cart"
                    href="/shop/bag"
                    onClick={(event) => followRoute(event, "/shop/bag")}
                  >
                    {t("View cart")}
                    <ArrowRight size={15} />
                  </a>
                </div>
                <p className="stone-stock">
                  {t(
                    product.sample
                      ? "Stock availability will be confirmed by the seller."
                      : product.available > 0
                        ? "Available to add to cart"
                        : "Out of stock",
                  )}
                </p>
              </div>
            )}
            <dl className="stone-quick-facts">
              {[
                ["Mineral", detail("mineral")],
                ["Mohs hardness", detail("hardness")],
                ["Colour", detail("color")],
                ["Shape", product.shape],
              ]
                .filter(([, value]) => value)
                .map(([label, value]) => (
                  <div key={label}>
                    <dt>{t(label)}</dt>
                    <dd>{value}</dd>
                  </div>
                ))}
            </dl>
            <div className="stone-service-row">
              <span>
                <FileCheck2 size={18} />
                {t("Transparent disclosures")}
              </span>
              <span>
                <Truck size={18} />
                {t("Indian delivery addresses")}
              </span>
            </div>
          </div>
        </div>
      </div>
      <nav
        className="stone-section-links"
        aria-label={t("Product information")}
      >
        {[
          ["stone-overview", "Stone details"],
          ["stone-tradition", "Astrological tradition"],
          ["stone-care", "Care & buying guide"],
          ["stone-policies", "Delivery & returns"],
        ].map(([id, label]) => (
          <a key={id} href={"#" + id}>
            {t(label)}
          </a>
        ))}
      </nav>
      <div className="stone-info-grid">
        <section id="stone-overview" className="stone-info-card">
          <h2>{t("Stone details")}</h2>
          <dl className="stone-specs">
            {[
              ["Listing ID", product.id],
              [
                "Carat weight",
                carats(product.weight_milli_carat) + " " + t("ct"),
              ],
              [
                "Origin",
                product.sample
                  ? t("Not verified")
                  : product.origin || t("Not specified"),
              ],
              [
                "Material",
                product.material === "natural"
                  ? t("Natural")
                  : product.material === "lab-grown"
                    ? t("Lab-grown")
                    : t("Not specified"),
              ],
              [
                "Treatment disclosure",
                product.sample
                  ? t("Not assessed")
                  : product.treatment || t("Not specified"),
              ],
              [
                "Dimensions",
                product.sample
                  ? t("To be measured")
                  : product.dimensions || t("Not specified"),
              ],
            ].map(([label, value]) => (
              <div key={label}>
                <dt>{t(label)}</dt>
                <dd>{value}</dd>
              </div>
            ))}
          </dl>
          <h3>{t("Laboratory report")}</h3>
          {product.certificate_lab ? (
            <>
              <p>
                {product.certificate_lab} · {product.certificate_no}
              </p>
              {product.certificate_url && (
                <a
                  href={product.certificate_url}
                  target="_blank"
                  rel="noreferrer"
                >
                  {t("View laboratory report")} ↗
                </a>
              )}
            </>
          ) : (
            <p>{t("No laboratory report has been supplied for this stone.")}</p>
          )}
        </section>
        <section id="stone-tradition" className="stone-info-card">
          <h2>
            <BookOpen size={22} />
            {t("Astrological tradition")}
          </h2>
          <p>
            {detail("tradition") ||
              t(
                "Ask for a personal consultation before selecting an astrological stone.",
              )}
          </p>
          <h3>{t("Choosing how to wear it")}</h3>
          <p>{detail("wearing")}</p>
          <p className="stone-belief-note">
            {t(
              "These are cultural and astrological beliefs, not scientifically established effects. Gemstones do not guarantee health, financial or relationship outcomes.",
            )}
          </p>
          <a
            href="/#guidance"
            onClick={(event) => followRoute(event, "/#guidance")}
          >
            {t("Discuss your requirements")}
            <ArrowRight size={15} />
          </a>
        </section>
        <section id="stone-care" className="stone-info-card">
          <h2>{t("Care & buying guide")}</h2>
          <h3>{t("Everyday care")}</h3>
          <p>
            {detail("care") ||
              t(
                "Ask the seller for care advice based on material and treatment.",
              )}
          </p>
          <h3>{t("Before you buy")}</h3>
          <p>{detail("buying")}</p>
          <a
            href="https://www.gia.edu/gem-encyclopedia"
            target="_blank"
            rel="noreferrer"
          >
            {t("Learn about gemstones at GIA")} ↗
          </a>
        </section>
        <section id="stone-policies" className="stone-info-card">
          <h2>{t("Delivery & returns")}</h2>
          <h3>{t("Delivery policy")}</h3>
          <p>
            {catalog.settings.delivery_policy ||
              t(
                "Dispatch timing and delivery charges will be confirmed by the seller before purchase.",
              )}
          </p>
          <h3>{t("Return policy")}</h3>
          <p>
            {catalog.settings.return_policy ||
              t("Ask the seller for return terms before ordering.")}
          </p>
          <p>
            {t(
              "Ring, pendant, metal and making charges are separate unless expressly included in the listing.",
            )}
          </p>
        </section>
      </div>
      <section className="stone-faq">
        <h2>{t("Before you choose")}</h2>
        {[
          [
            "Is this the exact stone I will receive?",
            "Product images show the stone type. Review the individual inventory photo and matching laboratory report, where supplied, before purchasing.",
          ],
          [
            "Which weight should I choose?",
            "Choose weight after discussing your requirements. Listed carat weights describe the product and are not personalised recommendations.",
          ],
          [
            "When can I complete my purchase?",
            "Add your selected stones to the cart. Payment becomes available when the seller confirms stock and enables checkout.",
          ],
        ].map(([question, answer]) => (
          <details key={question}>
            <summary>{t(question)}</summary>
            <p>{t(answer)}</p>
          </details>
        ))}
      </section>
      {!!related.length && (
        <section className="stone-related">
          <h2>{t("Explore more stones")}</h2>
          <div>
            {related.map((p) => {
              const path = "/shop/stones/" + p.slug;
              return (
                <article key={p.id}>
                  <a href={path} onClick={(event) => followRoute(event, path)}>
                    <div className="stone-related-photo">
                      <GemstoneArt product={p} />
                    </div>
                    <h3>
                      {language === "hi" && p.hindi_name
                        ? p.hindi_name
                        : p.name}
                    </h3>
                  </a>
                  <p>{money(p.price)}</p>
                  <AddToCartButton product={p} />
                </article>
              );
            })}
          </div>
        </section>
      )}
      <dialog
        ref={zoom}
        aria-label={t("Enlarge image")}
        className="stone-photo-dialog"
        onClick={(event) => {
          if (event.target === event.currentTarget) zoom.current?.close();
        }}
      >
        <button
          className="stone-dialog-close icon-button"
          aria-label={t("Close image")}
          onClick={() => zoom.current?.close()}
        >
          <X size={22} />
        </button>
        <GemstoneArt product={product} />
      </dialog>
      {!product.preview && showMobileCart && (
        <div className="stone-mobile-cart">
          <div>
            <strong>{money(product.price)}</strong>
            <small>{t("Price per piece")}</small>
          </div>
          <AddToCartButton product={product} quantity={quantity} />
        </div>
      )}
    </div>
  );
}
