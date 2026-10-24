import type { DashaPeriod } from "./types";
import type { MangalAssessment } from "./matching-types";
export type Varga = {
  factor: number;
  name: string;
  method: string;
  lagna_sign: number;
  planets: {
    name: string;
    rashi_number: number;
    house: number;
    vargottama: boolean;
  }[];
};
export type KundliDetails = {
  mangal: MangalAssessment;
  mangal_method: string;
  method: string;
  birth: {
    at: string;
    civil_weekday: number;
    hindu_weekday: number;
    paksha: string;
    limbs: { number: number; name: string; start: string; end: string }[];
    sunrise: string;
    sunset: string;
    day_start: string;
    day_end: string;
    ayanamsa: number;
    julian_day_ut1: number;
    chakra: {
      lagna_lord: string;
      rashi_lord: string;
      nakshatra_lord: string;
      varna: string;
      vashya: string;
      yoni: string;
      gana: string;
      nadi: string;
    };
    settings: Record<string, string>;
  };
  chalit: {
    method: string;
    lagna_sign: number;
    houses: {
      house: number;
      start_longitude: number;
      middle_longitude: number;
      end_longitude: number;
    }[];
    planets: {
      name: string;
      rashi_number: number;
      natal_house: number;
      house: number;
    }[];
  };
  vargas: Varga[];
  analysis: {
    varga_strength: {
      planet: string;
      total: number;
      maximum: number;
      parts: {
        factor: number;
        sign: number;
        owner: string;
        relationship: string;
        weight: number;
        points: number;
        contribution: number;
      }[];
    }[];
    method: string;
    conditions: {
      name: string;
      sign_lord: string;
      dignity: string;
      baladi: string;
      sun_separation: number;
      combust: boolean | null;
      combust_limit: number | null;
    }[];
    friendships: {
      from: string;
      to: string;
      natural: string;
      temporary: string;
      compound: string;
    }[];
    aspects: {
      planet: string;
      distance: number;
      sign: number;
      house: number;
      occupants: string[];
    }[];
    patterns: {
      code: string;
      name: string;
      matches: boolean;
      planets: string[];
      condition: string;
    }[];
  };
  ashtakavarga: {
    method: string;
    rows: {
      planet: string;
      points: number[];
      total: number;
      contributors: number[][];
    }[];
    sarva: number[];
    total: number;
    sources: string[];
  };
  yogini: {
    method: string;
    as_of: string;
    year_days: number;
    birth_lord: string;
    balance_years: number;
    current_lord: string;
    current_sub_lord: string;
    lords: { name: string; lord: string; years: number }[];
    periods: DashaPeriod[];
  };
};
export type SaturnTimelineData = {
  method: string;
  moon_sign: number;
  start: string;
  end: string;
  periods: {
    kind: string;
    phase: string;
    sign: number;
    from_moon: number;
    start: string;
    end: string;
    start_clipped: boolean;
    end_clipped: boolean;
  }[];
};
