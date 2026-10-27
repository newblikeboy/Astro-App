import type { KundliDetails } from "./kundli-details-types";
export type Birth = {
  name: string;
  date: string;
  time: string;
  place_id: string;
  unknown_time: boolean;
  utc_offset_minutes?: number;
};
export type Place = {
  id: string;
  name: string;
  hindi: string;
  lat: number;
  lon: number;
  zone: string;
};
export type Position = {
  longitude: number;
  rashi_number: number;
  rashi: string;
  degree_in_rashi: number;
  nakshatra: string;
  pada: number;
};
export type Planet = Position & {
  name: string;
  house: number;
  retrograde: boolean;
  longitude_speed_deg_per_day: number;
};
export type Rule = {
  id: string;
  version: number;
  planet: string;
  sign: number;
  house?: number;
  lagna_sign?: number;
  nakshatra_number?: number;
  dasha_lord?: string;
  category?: string;
  topic: string;
  hindi: string;
  english: string;
  mantra: string;
  gemstone: string;
  priority: number;
  conflict_group: string;
  status: string;
  approver: string;
  approved_at: string | null;
  demo: boolean;
};
export type Report = {
  numerology?: Numerology;
  details?: KundliDetails;
  id: string;
  user_id: string;
  birth: Birth;
  place: Place;
  chart: null | {
    lagna: Position;
    planets: Planet[];
    houses: { house: number; rashi_number: number; rashi: string }[];
    swiss_version: string;
    utc: string;
    settings: Record<string, string>;
  };
  rules: Rule[];
  navamsa?: {
    lagna_sign: number;
    planets: {
      name: string;
      rashi_number: number;
      house: number;
      vargottama: boolean;
    }[];
  };
  dasha?: {
    method?: string;
    max_depth?: number;
    current_path?: DashaStep[];
    as_of: string;
    year_days: number;
    birth_lord: string;
    balance_years: number;
    current_lord: string;
    current_sub_lord: string;
    periods: DashaPeriod[];
  };
  checks?: { code: string; matches: boolean; house?: number }[];
  transits?: {
    as_of: string;
    planets: (Planet & { from_moon: number })[];
    sade_sati: "none" | "rising" | "peak" | "setting";
  };
  limitations: string[];
  created: string;
  engine: string;
  demo: boolean;
};
export type NumerologyNumber = {
  value: number;
  planet: string;
  digits: number[];
  steps: number[];
};
export type Numerology = {
  method: string;
  birth_date: string;
  mulank: NumerologyNumber;
  bhagyank: NumerologyNumber;
  extended?: {
    method: string;
    name_status: "available" | "needs_latin_spelling";
    naamank?: {
      spelling: string;
      letters: string[];
      number: NumerologyNumber;
      meaning: NumerologyWording;
    };
    lucky_numbers: number[];
    lucky_dates: number[];
    profile: {
      number: number;
      facts: { id: string; label: NumerologyWording; value: NumerologyWording }[];
      guidance: { id: string; title: NumerologyWording; text: NumerologyWording }[];
      yantra: number[][];
    };
  };
};
export type NumerologyWording = { hi: string; en: string };
export type DashaPeriod = {
  lord: string;
  start: string;
  end: string;
  antardashas?: DashaPeriod[];
};
export type DashaStep = DashaPeriod & { level: number; path: number[] };
export type Order = {
  gemstone?: import("./shop-types").GemstoneSale;
  id: string;
  product: string;
  title: string;
  amount: number;
  status: string;
  provider_id: string;
  report_id: string;
  booking_id: string;
  created: string;
  demo: boolean;
  tax_mode: string;
  refunded: number;
};
export type Booking = {
  id: string;
  start: string;
  end: string;
  hold_until: string;
  status: string;
  join_url?: string;
  demo: boolean;
  order_id: string;
};
export type Job = {
  id: string;
  kind: string;
  status: string;
  attempts: number;
  error: string;
  ref: string;
  next: string;
};
export type Settings = {
  shop?: import("./shop-types").ShopSettings;
  pdf_price: number;
  consultation_price: number;
  duration: number;
  buffer: number;
  lead_hours: number;
  horizon_days: number;
  change_hours: number;
  zone: string;
  availability: { day: number; from: string; to: string }[];
  blocked: string[];
  bookings_enabled: boolean;
  pdf_enabled: boolean;
  tax_mode: string;
  tax_bps: number;
  licence_confirmed: boolean;
  content: {
    intro: string;
    videos: string[];
    faq: { question: string; answer: string }[];
    support: string;
    policies: Record<string, string>;
    approved: boolean;
  };
};
export type Config = {
  video_catalog?: {
    channel_id: string;
    channel_name: string;
    channel_url: string;
    checked_at: string;
    videos: ChannelVideo[];
  };
  simulation: boolean;
  development: boolean;
  otp_simulation: boolean;
  duration: number;
  duration_demo: boolean;
  settings: Settings;
  pdf_blockers: string[];
  booking_blockers: string[];
  payment_key: string;
  storage: string;
};
export type ChannelVideo = {
  id: string;
  title: string;
  english_title: string;
  views: number;
  duration: string;
  published_at: string;
};
export type Account = {
  user: { id: string; phone: string; marketing: boolean };
  csrf: string;
  reports: Report[];
  orders: Order[];
  bookings: Booking[];
  pdf_jobs: Job[];
};
export type Finance = {
  gemstone_sales: number;
  calculation_error?: string;
  month: string;
  pdf_sales: number;
  consultation_sales: number;
  gross: number;
  refunds: number;
  fees: number;
  tax: number;
  expenses: number;
  reserves: number;
  distributable: number;
  partner_a: number;
  partner_b: number;
  distributed: number;
  undistributed: number;
  tax_configured: boolean;
  unreconciled: boolean;
  rounding_remainder: number;
};
export type AdminData = {
  csrf: string;
  settings: Settings;
  rules: Record<string, Rule>;
  users: {
    id: string;
    phone: string;
    created: string;
    marketing: boolean;
    deleted: boolean;
  }[];
  profiles: { id: string; user_id: string; created: string; place: string }[];
  orders: Order[];
  bookings: Record<string, Booking>;
  jobs: Record<string, Job>;
  expenses: Record<
    string,
    {
      id: string;
      date: string;
      category: string;
      description: string;
      amount: number;
      receipt: string;
      status: string;
    }
  >;
  distributions: Record<
    string,
    {
      id: string;
      month: string;
      partner: string;
      amount: number;
      reference: string;
    }
  >;
  month: {
    month: string;
    reserve: number;
    statement_reference: string;
    closed: boolean;
  };
  finance: Finance;
  audit: { at: string; actor: string; action: string; ref: string }[];
  analytics: { name: string; product: string; source: string; at: string }[];
  pdf_blockers: string[];
  booking_blockers: string[];
};
