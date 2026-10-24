import { createContext, useContext, useEffect, useState } from "react";
import type { Dispatch, ReactNode, SetStateAction } from "react";
import type { BagItem, Gemstone } from "./shop-types";

const ShopBagContext = createContext<{
  bag: BagItem[];
  setBag: Dispatch<SetStateAction<BagItem[]>>;
  addToBag: (product: Gemstone, quantity?: number) => boolean;
} | null>(null);

/** readBag restores valid cart quantities from this browser tab, ignoring malformed entries. */
function readBag(): BagItem[] {
  try {
    const value = JSON.parse(sessionStorage.getItem("m4y-shop-bag") || "[]");
    return Array.isArray(value)
      ? value
          .filter(
            (p) =>
              p &&
              typeof p.product_id === "string" &&
              Number.isInteger(p.quantity) &&
              p.quantity > 0 &&
              p.quantity <= 10,
          )
          .slice(0, 10)
      : [];
  } catch {
    return [];
  }
}

/** ShopBagProvider keeps one tab-local cart across customer routes and persists quantity changes. */
export function ShopBagProvider({ children }: { children: ReactNode }) {
  const [bag, setBag] = useState<BagItem[]>(readBag);
  /** addToBag adds a bounded quantity to the shared cart and rejects unavailable items or cart limits. */
  function addToBag(product: Gemstone, quantity = 1) {
    const existing = bag.find((item) => item.product_id === product.id);
    if (
      product.preview ||
      !Number.isInteger(quantity) ||
      quantity < 1 ||
      quantity + (existing?.quantity || 0) > Math.min(product.available, 10) ||
      (!existing && bag.length >= 10)
    )
      return false;
    setBag(
      existing
        ? bag.map((item) =>
            item.product_id === product.id
              ? { ...item, quantity: item.quantity + quantity }
              : item,
          )
        : [...bag, { product_id: product.id, quantity }],
    );
    return true;
  }
  useEffect(() => {
    try {
      sessionStorage.setItem("m4y-shop-bag", JSON.stringify(bag));
    } catch {
      // Browsers that block storage can still use the cart during this visit.
    }
  }, [bag]);
  return (
    <ShopBagContext.Provider value={{ bag, setBag, addToBag }}>
      {children}
    </ShopBagContext.Provider>
  );
}

/** useShopBag shares cart items and quantity updates between storefront and customer headers. */
export function useShopBag() {
  const value = useContext(ShopBagContext);
  if (!value) throw new Error("ShopBagProvider is required");
  return value;
}
