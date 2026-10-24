export type ShopSettings = {
  enabled: boolean;
  shipping_fee: number;
  tax_mode: string;
  tax_bps: number;
  delivery_policy: string;
  return_policy: string;
};
export type Gemstone = {
  id: string;
  slug: string;
  name: string;
  hindi_name: string;
  kind: string;
  description: string;
  hindi_description: string;
  price: number;
  weight_milli_carat: number;
  stock: number;
  available: number;
  origin: string;
  material: string;
  treatment: string;
  shape: string;
  dimensions: string;
  image: string;
  certificate_lab: string;
  certificate_no: string;
  certificate_url: string;
  status: string;
  version: number;
  updated: string;
  preview: boolean;
  sample?: boolean;
  details?: Record<string, { en: string; hi: string }>;
  reference?: {
    checked_at: string;
    rate_paise_per_carat: number;
    source: string;
  };
};
export type DeliveryAddress = {
  name: string;
  phone: string;
  line1: string;
  line2: string;
  city: string;
  state: string;
  pin: string;
  country: string;
};
export type GemstoneSale = {
  items: { product_id: string; quantity: number; product: Gemstone }[];
  address: DeliveryAddress;
  subtotal: number;
  shipping: number;
  delivery_policy: string;
  return_policy: string;
  hold_until: string;
  fulfilment: string;
  inventory_taken: boolean;
  restocked: boolean;
  courier: string;
  tracking_number: string;
  tracking_url: string;
  shipped_at?: string;
  delivered_at?: string;
  shipment_version: number;
};
export type ShopCatalog = {
  products: Gemstone[];
  settings: ShopSettings;
  preview: boolean;
  checkout_available: boolean;
};
export type BagItem = { product_id: string; quantity: number };
/** carats formats stored integer milli-carats for readable product weight display. */
export const carats = (n: number) =>
  (n / 1000).toFixed(3).replace(/\.?0+$/, "");
/** parseCarats parses a bounded carat input into exact integer milli-carats. */
export function parseCarats(value: string) {
  const match = /^(\d{1,6})(?:\.(\d{1,3}))?$/.exec(value.trim());
  if (!match)
    throw new Error("Enter carat weight with up to three decimal places.");
  return Number(match[1]) * 1000 + Number((match[2] || "").padEnd(3, "0"));
}
