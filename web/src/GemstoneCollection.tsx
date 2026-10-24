import { useEffect, useState } from "react";
import { ArrowRight, Gem, LoaderCircle } from "lucide-react";
import { api, money } from "./api";
import { t, useLanguage } from "./i18n";
import { followRoute } from "./routing";
import { carats } from "./shop-types";
import type { ShopCatalog } from "./shop-types";
import GemstoneArt from "./GemstoneArt";
import AddToCartButton from "./AddToCartButton";
import { Button, ErrorBox } from "./UI";
import "./gemstone-collection.css";

/** GemstoneCollection introduces three catalogue items and links to the full shop, retaining stock disclosures and server prices. */
export default function GemstoneCollection() {
  const { language } = useLanguage();
  const [catalog, setCatalog] = useState<ShopCatalog | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError("");
    api<ShopCatalog>("/shop")
      .then((result) => {
        if (active) setCatalog(result);
      })
      .catch((cause: Error) => {
        if (active) setError(cause.message);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [attempt]);

  return (
    <section
      className="home-gemstones"
      id="gemstones"
      aria-labelledby="gemstones-title"
    >
      <div className="container">
        <div className="home-gemstones-heading">
          <div>
            <h2 id="gemstones-title">{t("Find your stone")}</h2>
            <p>
              {t(
                "Explore gemstones and meaningful stones. Compare their details, choose your favourites, and order directly.",
              )}
            </p>
          </div>
          <a
            className="button secondary"
            href="/shop"
            onClick={(event) => followRoute(event, "/shop")}
          >
            {t("Explore the collection")}
            <ArrowRight size={17} aria-hidden="true" />
          </a>
        </div>

        {loading ? (
          <div className="home-gemstones-state" role="status">
            <LoaderCircle className="spin" size={24} aria-hidden="true" />
            <p>{t("Opening the collection...")}</p>
          </div>
        ) : error ? (
          <div className="home-gemstones-state">
            <ErrorBox error={error} />
            <Button onClick={() => setAttempt((value) => value + 1)}>
              {t("Try again")}
            </Button>
          </div>
        ) : catalog && catalog.products.length > 0 ? (
          <>
            <div className="home-gemstones-grid">
              {catalog.products.slice(0, 3).map((product) => {
                const path = "/shop/stones/" + product.slug;
                const name =
                  language === "hi" && product.hindi_name
                    ? product.hindi_name
                    : product.name;
                return (
                  <article key={product.id} className="home-stone">
                    <a
                      className="home-stone-image-link"
                      href={path}
                      onClick={(event) => followRoute(event, path)}
                    >
                      <div className="home-stone-visual">
                        <GemstoneArt product={product} />
                        <span className="home-stone-status">
                          {t(
                            product.sample
                              ? "Availability pending"
                              : product.preview
                                ? "Preview"
                                : product.available > 0
                                  ? "Available"
                                  : "Out of stock",
                          )}
                        </span>
                      </div>
                    </a>
                    <div className="home-stone-copy">
                      <h3>
                        <a
                          href={path}
                          onClick={(event) => followRoute(event, path)}
                        >
                          {name}
                        </a>
                      </h3>
                      <p>
                        {product.preview
                          ? t("Product details coming soon")
                          : [
                              carats(product.weight_milli_carat) +
                                " " +
                                t("ct"),
                              product.shape || product.origin,
                            ]
                              .filter(Boolean)
                              .join(" · ")}
                      </p>
                      <div className="home-stone-bottom">
                        <span>
                          {product.preview
                            ? t("Collection preview")
                            : money(product.price)}
                        </span>
                        <a
                          className="home-stone-arrow"
                          href={path}
                          aria-label={t("View stone") + " " + name}
                          onClick={(event) => followRoute(event, path)}
                        >
                          <ArrowRight size={18} />
                        </a>
                      </div>
                      {!product.preview && (
                        <AddToCartButton product={product} />
                      )}
                    </div>
                  </article>
                );
              })}
            </div>
          </>
        ) : (
          <div className="home-gemstones-state">
            <Gem size={30} aria-hidden="true" />
            <h3>{t("The collection is being prepared")}</h3>
            <p>
              {t(
                "The seller will publish individual stones here with their prices and details.",
              )}
            </p>
          </div>
        )}
      </div>
    </section>
  );
}
