import { useState } from "react";
import { Gem } from "lucide-react";
import type { Gemstone } from "./shop-types";
import { t, useLanguage } from "./i18n";
import "./gemstone-art.css";

const samples: Record<string, string> = {
  ruby: "/images/gemstones/ruby.webp",
  "yellow-sapphire": "/images/gemstones/yellow-sapphire.webp",
  emerald: "/images/gemstones/emerald.webp",
  "blue-sapphire": "/images/gemstones/blue-sapphire.webp",
  pearl: "/images/gemstones/pearl.webp",
  amethyst: "/images/gemstones/amethyst.webp",
  "red-coral": "/images/gemstones/red-coral.webp",
  hessonite: "/images/gemstones/hessonite.webp",
  "cat's-eye": "/images/gemstones/cats-eye.webp",
  diamond: "/images/gemstones/diamond.webp",
  "white-sapphire": "/images/gemstones/white-sapphire.webp",
  opal: "/images/gemstones/opal.webp",
  citrine: "/images/gemstones/citrine.webp",
  turquoise: "/images/gemstones/turquoise.webp",
  peridot: "/images/gemstones/peridot.webp",
  moonstone: "/images/gemstones/moonstone.webp",
};

/** GemstoneArt prioritizes the main product photo when requested and supplies catalogue imagery for missing or failed uploads. */
export default function GemstoneArt({
  product,
  priority = false,
}: {
  product: Gemstone;
  priority?: boolean;
}) {
  const { language } = useLanguage();
  const [failed, setFailed] = useState<string[]>([]);
  const name =
    language === "hi" && product.hindi_name ? product.hindi_name : product.name;
  const sample =
    samples[product.name.toLowerCase().trim().replace(/\s+/g, "-")];
  const source =
    product.image && !failed.includes(product.image)
      ? product.image
      : sample && !failed.includes(sample)
        ? sample
        : "";
  return (
    <div className={source ? "gem-photo" : "gem-photo gem-photo-missing"}>
      {source ? (
        <>
          <img
            className="gem-product-photo"
            src={source}
            alt={name}
            loading={priority ? "eager" : "lazy"}
            fetchPriority={priority ? "high" : "auto"}
            decoding="async"
            onError={() => setFailed((previous) => [...previous, source])}
          />
        </>
      ) : (
        <>
          <Gem size={36} aria-hidden="true" />
          <span>{t("Photo coming soon")}</span>
        </>
      )}
    </div>
  );
}
