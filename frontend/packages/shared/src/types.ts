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
