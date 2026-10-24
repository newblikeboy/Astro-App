import { useState } from "react";
import { Check, ShoppingBag } from "lucide-react";
import { useShopBag } from "./ShopBag";
import { t } from "./i18n";
import type { Gemstone } from "./shop-types";

/** AddToCartButton shares quantity validation and accessible cart feedback across listing and detail pages. */
export default function AddToCartButton({
  product,
  quantity = 1,
}: {
  product: Gemstone;
  quantity?: number;
}) {
  const { addToBag } = useShopBag();
  const [message, setMessage] = useState("");
  const [added, setAdded] = useState(false);
  return (
    <div className="cart-purchase-control">
      <button
        className="button"
        disabled={product.preview || product.available < 1}
        onClick={() => {
          const success = addToBag(product, quantity);
          setAdded(success);
          setMessage(
            success
              ? "Added to cart"
              : "The available quantity or cart limit has been reached.",
          );
        }}
      >
        {added ? (
          <Check size={17} aria-hidden="true" />
        ) : (
          <ShoppingBag size={17} aria-hidden="true" />
        )}
        {t(product.available < 1 ? "Out of stock" : "Add to cart")}
      </button>
      {message && (
        <span
          className={
            added
              ? "cart-purchase-feedback"
              : "cart-purchase-feedback cart-purchase-error"
          }
          role="status"
        >
          {t(message)}
        </span>
      )}
    </div>
  );
}
