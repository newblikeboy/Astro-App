export type DailyPeriod = {
  number?: number;
  name: string;
  start: string;
  end: string;
  quality?: string;
  cautions?: string[];
};
export type DailyPosition = {
  rashi_number: number;
  rashi: string;
  nakshatra: string;
  pada: number;
  degree_dms: string;
};
export type LunarCalendar = {
  amanta: string;
  purnimanta: string;
  month_number: number;
  adhika: boolean;
  skipped_month?: string;
  paksha: string;
  vikram: number;
  shaka: number;
  lunation_start: string;
  lunation_end: string;
};
export type PanchangDetailsData = {
  moonrise: string | null;
  moonset: string | null;
  day_seconds: number;
  night_seconds: number;
  midday: string;
  sun: DailyPosition;
  moon: DailyPosition;
  moon_signs: DailyPeriod[];
  sun_signs: DailyPeriod[];
  padas: DailyPeriod[];
  lagna: DailyPeriod[];
  panchaka: DailyPeriod[];
  muhurats: DailyPeriod[];
  special: DailyPeriod[];
  balam: {
    start: string;
    end: string;
    moon_sign: number;
    nakshatra: number;
    chandra: number[];
    tara: number[];
  }[];
  directions: { name: string; value: string; start: string; end: string }[];
  calendar: LunarCalendar;
  tropical_ritu: string;
  lunar_ritu: string;
  tropical_ayana: string;
  sidereal_ayana: string;
  ayanamsa: number;
  julian_day_ut1: number;
};
