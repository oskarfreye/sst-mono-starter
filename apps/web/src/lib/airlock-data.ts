export type MissionStatus =
  | "DECLARED"
  | "PROOF_PENDING"
  | "CONFIRMED"
  | "AIRLOCKED";

export type SeatStatus = "OCCUPIED" | "OPEN" | "VACATED";

export interface SeatSnapshot {
  seatNumber: number;
  cohort: "THE_100" | "POST_100";
  status: SeatStatus;
  handle?: string;
  displayName?: string;
  role?: string;
  mission?: string;
  hatchAt?: string;
  vacatedAt?: string;
  vacatedOn?: string;
  price?: number;
}

export interface MissionRow {
  missionId: string;
  seatNumber: number;
  handle: string;
  displayName: string;
  role: string;
  mission: string;
  declarationUrl?: string;
  declaredAt: string;
  deadlineAt: string;
  status: MissionStatus;
  proofUrl?: string;
}

export interface PriceTier {
  id: string;
  price: number;
  seats: number;
  range: string;
  filled: number;
}

export interface EventEntry {
  time: string;
  icon: string;
  handle: string;
  event: string;
  tag: string;
  tone?: "launch" | "airlock" | "pending";
}

export interface WakeEntry {
  id: string;
  occurredAt: string;
  handle: string;
  seatLabel: string;
  mission: string;
  summary: string;
}

export const declaredAt = "2026-05-13T18:00:00Z";
export const deadlineAt = "2026-06-12T18:00:00Z";

export const currentMission: MissionRow = {
  missionId: "MISSION-001",
  seatNumber: 1,
  handle: "oskar",
  displayName: "Oskar Freye",
  role: "founder",
  mission: "Ship Airlock v1.0 + Stripe checkout",
  declarationUrl: "https://theairlock.space",
  declaredAt,
  deadlineAt,
  status: "DECLARED",
};

export const pricingTiers: PriceTier[] = [
  { id: "01", price: 42, seats: 10, range: "01-10", filled: 1 },
  { id: "02", price: 84, seats: 15, range: "11-25", filled: 0 },
  { id: "03", price: 168, seats: 25, range: "26-50", filled: 0 },
  { id: "04", price: 336, seats: 50, range: "51-100", filled: 0 },
];

export const seats: SeatSnapshot[] = Array.from({ length: 100 }, (_, index) => {
  const seatNumber = index + 1;
  if (seatNumber === 1) {
    return {
      seatNumber,
      cohort: "THE_100",
      status: "OCCUPIED",
      handle: currentMission.handle,
      displayName: currentMission.displayName,
      role: currentMission.role,
      mission: currentMission.mission,
      hatchAt: currentMission.deadlineAt,
    };
  }

  return {
    seatNumber,
    cohort: "THE_100",
    status: "OPEN",
    price: priceForSeat(seatNumber),
  };
});

export const missionRows: MissionRow[] = [currentMission];

export const events: EventEntry[] = [
  {
    time: "18:00:00",
    icon: "->",
    handle: "@oskar",
    event: "mission declared · Airlock v1.0 + Stripe checkout",
    tag: "SEAT 01 · DECLARED",
  },
  {
    time: "17:52:41",
    icon: "o",
    handle: "@oskar",
    event: "boarding pass purchased · seat 01 taken",
    tag: "THE 100 · +1",
    tone: "pending",
  },
];

export const wakeEntries: WakeEntry[] = [];

export const receipts = [
  {
    name: "fr3n.fan",
    meta: "Co-founder · 2024",
    href: "https://fr3n.fan",
  },
  {
    name: "creavings.com",
    meta: "Co-founder · 2025",
    href: "https://creavings.com",
  },
  {
    name: "freye.tech",
    meta: "Freelance studio",
    href: "https://freye.tech",
  },
];

export const metrics = {
  occupied: 1,
  vacated: 0,
  open: 99,
  confirmedThisWeek: 0,
  confirmedLifetime: 0,
  nextSeat: 2,
  nextSeatPrice: 42,
  nextTierPrice: 84,
  seatsUntilNextTier: 9,
};

export function seatLabel(seatNumber: number): string {
  return seatNumber <= 100
    ? `THE 100 · SEAT ${padSeat(seatNumber)}`
    : `SEAT ${seatNumber}`;
}

export function padSeat(seatNumber: number): string {
  return String(seatNumber).padStart(2, "0");
}

export function formatDate(iso: string): string {
  return new Intl.DateTimeFormat("en", {
    timeZone: "UTC",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date(iso));
}

export function formatUtc(iso: string): string {
  return new Intl.DateTimeFormat("en", {
    timeZone: "UTC",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  })
    .format(new Date(iso))
    .replace(",", "");
}

export function priceForSeat(seatNumber: number): number {
  if (seatNumber <= 10) return 42;
  if (seatNumber <= 25) return 84;
  if (seatNumber <= 50) return 168;
  return 336;
}
