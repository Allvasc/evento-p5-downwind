import { computed, type Ref } from "vue";
import { useRoute } from "vue-router";
import { useQuery } from "@tanstack/vue-query";
import type { Product } from "@p5wellness/shared";
import { api } from "@/lib/api";

// Páginas públicas de eventos além do principal: /<slug> e /<slug>/comprar. O evento
// principal continua em "/" e "/comprar", sem slug — e sem nenhuma dessas chamadas novas.

export interface PublicEvent {
  id: string;
  name: string;
  slug: string;
  isDefault: boolean;
  headline: string;
  subtitle: string;
  dateLabel: string;
  locationLabel: string;
  description: string;
  bannerUrl: string;
  includes: string;
  schedule: string;
}

export interface NextSession {
  activityTitle: string;
  startsAt: string;
  capacity: number;
  booked: number;
}

/** Slug do evento na rota atual ("" no evento principal). */
export function useEventSlug() {
  const route = useRoute();
  return computed(() => {
    const raw = route.params.eventSlug;
    return typeof raw === "string" ? raw : "";
  });
}

/** Caminhos do site para um evento: o principal mantém "/" e "/comprar". */
export function eventPaths(slug: string) {
  return {
    home: slug ? `/${slug}` : "/",
    buy: slug ? `/${slug}/comprar` : "/comprar",
  };
}

/** Querystring do catálogo público: vazia no evento principal (mesma chamada de sempre). */
export function eventQuery(slug: string) {
  return slug ? `?event=${encodeURIComponent(slug)}` : "";
}

export function usePublicEvent(slug: Ref<string>) {
  return useQuery({
    queryKey: computed(() => ["public-event", slug.value]),
    queryFn: () => api.get<PublicEvent>(`/public/events/${encodeURIComponent(slug.value)}`),
    enabled: computed(() => !!slug.value),
    retry: false,
    staleTime: 60_000,
  });
}

// O evento principal usa a mesma queryKey de sempre (["public-products"]), compartilhada
// com a Home; os outros eventos têm a sua própria. Um endereço de evento inexistente
// responde 404 — sem novas tentativas nesse caso.
export function useEventProducts(slug: Ref<string>, options: { staleTime?: number } = {}) {
  return useQuery({
    queryKey: computed(() => (slug.value ? ["public-products", slug.value] : ["public-products"])),
    queryFn: () => api.get<{ products: Product[] }>(`/public/products${eventQuery(slug.value)}`),
    retry: (failureCount: number) => !slug.value && failureCount < 3,
    staleTime: options.staleTime ?? 60_000,
  });
}

export function useEventNextSessions(slug: Ref<string>) {
  return useQuery({
    queryKey: computed(() => (slug.value ? ["public-next-sessions", slug.value] : ["public-next-sessions"])),
    queryFn: () => api.get<{ sessions: NextSession[] }>(`/public/next-sessions${eventQuery(slug.value)}`),
    retry: false,
    staleTime: 60_000,
  });
}

/** "Título | detalhe" por linha → itens da página. Linhas vazias são ignoradas. */
export function parseEventLines(text: string) {
  return text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const i = line.indexOf("|");
      if (i === -1) return { title: line, detail: "" };
      return { title: line.slice(0, i).trim(), detail: line.slice(i + 1).trim() };
    });
}
