import { ShoppingBag } from "lucide-react";
import { t, useLanguage } from "./i18n";
import { followRoute, useAppLocation } from "./routing";
import { useShopBag } from "./ShopBag";
import "./cart.css";

/** CartLink opens the shared cart and shows its total quantity on every customer-facing header. */
export default function CartLink() {
  const { language } = useLanguage();
  const { bag } = useShopBag();
  const route = useAppLocation();
  const quantity = bag.reduce((sum, item) => sum + item.quantity, 0);
  const count = new Intl.NumberFormat(language).format(quantity);
  return (
    <a
      className="cart-link"
      href="/shop/bag"
      aria-label={`${t("Shopping bag")}, ${count}`}
      aria-current={route.pathname === "/shop/bag" ? "page" : undefined}
      onClick={(event) => followRoute(event, "/shop/bag")}
    >
      <ShoppingBag size={21} aria-hidden="true" />
      <span className="cart-count" aria-hidden="true">
        {count}
      </span>
    </a>
  );
}
