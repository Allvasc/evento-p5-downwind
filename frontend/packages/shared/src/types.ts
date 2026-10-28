export type ProductType = "class" | "combo";

export interface Activity {
  id: string;
  title: string;
  slug: string;
  instructor?: string;
  durationMinutes?: number;
  description?: string;
}

export interface Product {
  id: string;
  title: string;
  slug: string;
  description?: string;
  type: ProductType;
  includesBreakfast: boolean;
  priceCents: number;
  featured: boolean;
  /** true = customer picks exactly one of `activities` (e.g. "Yoga ou HYROX") instead of booking all of them. */
  chooseOneActivity: boolean;
  activities: Activity[];
}

export function formatBRL(cents: number): string {
  return new Intl.NumberFormat("pt-BR", {
    style: "currency",
    currency: "BRL",
  }).format(cents / 100);
}

/** Turma com esta capacidade (ou mais) é exibida como "vagas ilimitadas" — o banco segue com um número comum. */
export const UNLIMITED_CAPACITY = 9999;

export function isUnlimitedCapacity(capacity: number): boolean {
  return capacity >= UNLIMITED_CAPACITY;
}

export function cleanTitle(title: string): string {
  if (!title) return "";
  return title.replace(/^(P5 DownWind(?: Day)?)\s*[—\-–].*$/i, "$1").trim();
}

