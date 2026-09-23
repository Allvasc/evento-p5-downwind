<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useRouter } from "vue-router";
import { useQuery } from "@tanstack/vue-query";
import {
  ArrowRight,
  Check,
  Compass,
  MapPin,
  Bus,
  LifeBuoy,
  HandHelping,
  Shirt,
  Sparkles,
  Video,
  Music,
  Moon,
  Sunset,
} from "lucide-vue-next";
import { formatBRL, cleanTitle, type Product } from "@p5wellness/shared";
import { api } from "@/lib/api";
import WellnessHeader from "@/components/WellnessHeader.vue";
import HelpFab from "@/components/HelpFab.vue";
import KitesurferIcon from "@/components/KitesurferIcon.vue";
import WaveIcon from "@/components/WaveIcon.vue";

const router = useRouter();

function scrollToInclui() {
  document.querySelector("#inclui")?.scrollIntoView({ behavior: "smooth" });
}

// ── Produtos e Próxima data ──────────────────────────────────────────────────
interface NextSession {
  activityTitle: string;
  startsAt: string;
}

const { data: productsData } = useQuery({
  queryKey: ["public-products"],
  queryFn: () => api.get<{ products: Product[] }>("/public/products"),
  staleTime: 60_000,
});

const products = computed(() => productsData.value?.products ?? []);

const selectedProductIndex = ref(0);

watch(products, (list) => {
  if (list.length > 0) {
    const featuredIdx = list.findIndex((p) => p.featured);
    selectedProductIndex.value = featuredIdx !== -1 ? featuredIdx : 0;
  }
}, { immediate: true });

const activeProduct = computed(() => {
  if (!products.value.length) return null;
  return products.value[selectedProductIndex.value] ?? products.value[0];
});

const formattedPrice = computed(() => {
  if (!activeProduct.value) return "R$ 200";
  return formatBRL(activeProduct.value.priceCents);
});

const productTitle = computed(() => cleanTitle(activeProduct.value?.title ?? "Downwind do Luau P5"));
const productDescription = computed(
  () => activeProduct.value?.description ?? "Downwind Sababa → P5 Kite House do pôr do sol à lua, com kites iluminados por LEDs e Luau P5 na chegada."
);

const { data: nextSessionsData } = useQuery({
  queryKey: ["public-next-sessions"],
  queryFn: () => api.get<{ sessions: NextSession[] }>("/public/next-sessions"),
  staleTime: 60_000,
});

const nextSessions = computed(() => nextSessionsData.value?.sessions ?? []);

function formatEventDate(iso: string) {
  const label = new Date(iso).toLocaleDateString("pt-BR", { weekday: "long", day: "2-digit", month: "long" });
  return label.charAt(0).toUpperCase() + label.slice(1);
}

const nextDateLabel = computed(() => {
  const sessions = nextSessions.value;
  if (!sessions.length) return null;
  return formatEventDate(sessions[0].startsAt);
});

const included = [
  { icon: Shirt, title: "Lycra exclusiva", detail: "Kit do participante com a lycra P5 — Temporada 2026." },
  { icon: Sparkles, title: "LEDs nos kites", detail: "Instalação dos LEDs no Sababa para navegar com o kite iluminado." },
  { icon: Bus, title: "Concentração na P5", detail: "Concentração às 15h na P5 Kite House e ida ao Sababa." },
  { icon: LifeBuoy, title: "Apoio na água", detail: "Acompanhamento da equipe do LPD na água durante todo o percurso." },
  { icon: MapPin, title: "Percurso", detail: "Sababa → P5 Kite House, do pôr do sol à noite de lua." },
  { icon: Video, title: "Registro da experiência", detail: "Equipe filmando tudo, da preparação à chegada." },
  { icon: HandHelping, title: "Beach Boys na chegada", detail: "Ajuda na desmontagem dos equipamentos na P5." },
  { icon: Music, title: "Luau P5", detail: "A chegada é só o começo da noite: música, lua e aquela energia P5." },
];

const itinerary = [
  { time: "1", title: "15h · Concentração", detail: "Concentração na P5 Kite House e ida ao Sababa." },
  { time: "2", title: "Preparação no Sababa", detail: "Preparamos os equipamentos e instalamos os LEDs nos kites." },
  { time: "3", title: "17h30 · Nascer da lua", detail: "Com a lua começando a nascer, saímos do Sababa rumo à P5, na transição do pôr do sol para a noite." },
  { time: "4", title: "18h · Chegada na P5", detail: "Chegada prevista entre 18h e 18h30, com os Beach Boys ajudando na desmontagem." },
  { time: "5", title: "Luau P5", detail: "Você só guarda o kite e entra no clima. A chegada é só o começo da noite." },
];
</script>

<template>
  <div class="min-h-screen bg-paper">
    <WellnessHeader />
    <HelpFab />

    <main>
      <!-- HERO -->
      <section class="relative overflow-hidden pt-4 pb-16 md:py-20">
        <div class="hero-orb hero-orb--blue"></div>
        <div class="hero-orb hero-orb--cream"></div>

        <div class="relative z-10 mx-auto grid max-w-6xl gap-12 px-6 md:grid-cols-[1.1fr_0.9fr] md:items-center">
          <div>
            <p class="eyebrow mb-6 text-xs tracking-wider">
              <Compass :size="14" class="text-magenta" />
              P5 KITE HOUSE · DOWNWIND DO LUAU P5
            </p>

            <h1 class="font-serif text-5xl font-black leading-[1.05] text-ink md:text-6xl">
              Começamos com o sol.<br />
              <span class="text-magenta">Navegamos com a lua.</span>
            </h1>

            <p class="mt-6 max-w-md text-base leading-relaxed text-ink-soft md:text-lg">
              Terminamos no Luau P5. 🔥 O <strong class="font-semibold text-ink">{{ productTitle }}</strong> é uma experiência diferente no mar: {{ productDescription }}
            </p>

            <div class="mt-8 flex flex-wrap items-center gap-5">
              <button class="button-magenta" @click="router.push('/comprar')">
                Garantir minha vaga · {{ formattedPrice }}
                <ArrowRight :size="18" />
              </button>
              <button
                class="text-sm font-semibold text-ink underline underline-offset-4 transition-colors hover:text-magenta"
                @click="scrollToInclui"
              >
                Ver o que está incluído
              </button>
            </div>

            <div class="mt-8 flex flex-wrap gap-x-6 gap-y-2 text-xs font-medium text-ink-soft">
              <span class="inline-flex items-center gap-1.5"><Check :size="15" class="text-magenta" /> Vagas limitadas</span>
              <span class="inline-flex items-center gap-1.5"><Check :size="15" class="text-magenta" /> Lycra exclusiva P5 2026</span>
              <span class="inline-flex items-center gap-1.5"><Check :size="15" class="text-magenta" /> Kites com LEDs + Luau P5</span>
            </div>
          </div>

          <!-- STACKED PRODUCT CARDS CONTAINER (Image 2 style) -->
          <div class="relative mx-auto w-full max-w-md">
            <div class="relative rounded-[2rem] border border-line/80 bg-white p-6 shadow-2xl md:p-8">
              <div class="text-center">
                <img src="/logo-downwind.webp" :alt="productTitle" class="mx-auto h-7 w-auto" />
                <p class="mt-2 text-xs font-medium text-ink-soft">
                  Escolha como você quer viver essa experiência.
                </p>
              </div>

              <!-- Stacked Product Cards -->
              <div class="mt-6 space-y-3">
                <div
                  v-for="(p, index) in products"
                  :key="p.id"
                  :class="[
                    'relative flex cursor-pointer items-center justify-between rounded-2xl border p-4 transition-all',
                    selectedProductIndex === index
                      ? 'border-magenta bg-white ring-2 ring-magenta/20 shadow-md'
                      : 'border-line/80 bg-warm/30 hover:border-ink/40 hover:bg-white',
                  ]"
                  @click="selectedProductIndex = index"
                >
                  <div class="pr-2">
                    <div class="flex flex-wrap items-center gap-2">
                      <h3 class="font-serif text-sm font-bold text-ink">{{ cleanTitle(p.title) }}</h3>
                      <span
                        v-if="p.includesBreakfast"
                        class="rounded-full bg-magenta/10 px-2 py-0.5 font-mono text-[9px] font-bold text-magenta uppercase"
                      >
                        Com Café
                      </span>
                      <span
                        v-else
                        class="rounded-full bg-warm px-2 py-0.5 font-mono text-[9px] font-bold text-ink-soft uppercase"
                      >
                        Ingresso Único
                      </span>
                    </div>
                    <p class="mt-1 text-xs text-ink-soft">
                      {{ p.includesBreakfast ? 'Sababa → P5 + Café da Manhã' : 'Sababa → P5 · lycra exclusiva inclusa' }}
                    </p>
                  </div>

                  <div class="shrink-0 text-right">
                    <span class="font-sans text-base font-bold text-ink">{{ formatBRL(p.priceCents) }}</span>
                  </div>
                </div>
              </div>

              <!-- Route details footer inside card -->
              <div class="mt-6 flex items-center justify-between border-t border-line/60 pt-4 text-xs text-ink-soft">
                <div>
                  <p class="font-mono text-[9px] font-bold uppercase tracking-wider text-ink-soft">Concentração · 15h</p>
                  <p class="font-serif font-bold text-ink">P5 Kite House</p>
                </div>
                <Moon :size="18" class="text-magenta shrink-0" />
                <div class="text-right">
                  <p class="font-mono text-[9px] font-bold uppercase tracking-wider text-ink-soft">Percurso</p>
                  <p class="font-serif font-bold text-ink">Sababa &rarr; P5</p>
                </div>
              </div>
            </div>

            <!-- Floating Top Badge -->
            <span v-if="nextDateLabel" class="absolute -top-3 -right-2 inline-flex items-center gap-1.5 rounded-full border border-magenta/20 bg-white px-3.5 py-1 text-[11px] font-semibold text-magenta shadow-sm">
              <Compass :size="12" /> próxima data: {{ nextDateLabel }}
            </span>

            <!-- Bottom Label -->
            <span class="absolute -bottom-3 left-6 rounded-full border border-line bg-white px-3.5 py-1 text-[11px] font-medium text-ink-soft shadow-sm">
              por P5 Kite House
            </span>
          </div>
        </div>
      </section>

      <!-- SECTION: O QUE ESTÁ INCLUSO -->
      <section id="inclui" class="border-t border-line/60 bg-warm/40 py-20">
        <div class="mx-auto max-w-6xl px-6">
          <p class="eyebrow mb-2">TUDO INCLUÍDO NO SEU INGRESSO</p>
          <div class="grid gap-6 md:grid-cols-[1.2fr_1fr] md:items-end">
            <h2 class="font-serif text-3xl font-bold leading-tight text-ink md:text-5xl">
              O que está<br />
              <span class="text-magenta">incluso.</span>
            </h2>
            <p class="text-base leading-relaxed text-ink-soft">
              Da concentração na P5 Kite House à chegada no Luau, passando pela preparação no Sababa e pela navegação com os kites iluminados, tudo já está no seu ingresso.
            </p>
          </div>

          <div class="mt-12 grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
            <div
              v-for="item in included"
              :key="item.title"
              class="flex flex-col gap-4 rounded-[var(--radius-card)] border border-line/80 bg-white p-6"
            >
              <span class="flex h-10 w-10 items-center justify-center rounded-full bg-magenta/10 text-magenta">
                <component :is="item.icon" :size="19" />
              </span>
              <div>
                <h3 class="font-sans text-sm font-bold text-ink">{{ item.title }}</h3>
                <p class="mt-1.5 text-xs leading-relaxed text-ink-soft">{{ item.detail }}</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- SECTION: PERCURSO / COMO FUNCIONA -->
      <section id="percurso" class="py-20">
        <div class="mx-auto max-w-6xl px-6">
          <p class="eyebrow mb-2"><WaveIcon :size="14" /> DO SOL À LUA</p>
          <h2 class="font-serif text-3xl font-bold text-ink md:text-5xl">
            Como funciona <span class="text-magenta">a sua noite.</span>
          </h2>

          <div class="mt-14 grid gap-8 md:grid-cols-5">
            <div v-for="(step, i) in itinerary" :key="step.title" class="relative">
              <div class="flex items-center gap-3">
                <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-ink font-mono text-sm font-bold text-white">
                  {{ step.time }}
                </span>
                <div v-if="i < itinerary.length - 1" class="hidden h-px flex-1 bg-line md:block"></div>
              </div>
              <h3 class="mt-5 font-serif text-lg font-bold text-ink">{{ step.title }}</h3>
              <p class="mt-1.5 text-sm leading-relaxed text-ink-soft">{{ step.detail }}</p>
            </div>
          </div>
        </div>
      </section>

      <!-- SECTION: CONFIANÇA / SEGURANÇA -->
      <section class="border-t border-line/60 bg-ink py-20 text-white">
        <div class="mx-auto max-w-6xl px-6">
          <div class="mb-6 inline-flex h-16 w-16 items-center justify-center rounded-2xl bg-white/5">
            <KitesurferIcon :size="34" class="text-sky-300" />
          </div>
          <p class="font-mono text-xs font-medium tracking-widest uppercase text-white/60">MAR · LUA · KITE · LEDS · MÚSICA</p>
          <h2 class="mt-2 font-serif text-3xl font-bold leading-tight md:text-4xl">
            Do sol à lua. <span class="text-sky-300">Do mar ao luau.</span>
          </h2>

          <div class="mt-12 grid gap-8 md:grid-cols-3">
            <div class="flex flex-col gap-3">
              <Sunset :size="26" class="text-sky-300" />
              <h3 class="font-serif text-lg font-bold">Do pôr do sol à noite</h3>
              <p class="text-sm leading-relaxed text-white/70">
                Navegação Sababa → P5 na transição do pôr do sol para a noite, com a lua nascendo por volta das 17h30.
              </p>
            </div>
            <div class="flex flex-col gap-3">
              <LifeBuoy :size="26" class="text-sky-300" />
              <h3 class="font-serif text-lg font-bold">Apoio na água</h3>
              <p class="text-sm leading-relaxed text-white/70">
                Acompanhamento da equipe do LPD na água durante todo o percurso, com os kites iluminados por LEDs.
              </p>
            </div>
            <div class="flex flex-col gap-3">
              <Music :size="26" class="text-sky-300" />
              <h3 class="font-serif text-lg font-bold">Luau P5 na chegada</h3>
              <p class="text-sm leading-relaxed text-white/70">
                Beach Boys ajudam na desmontagem. Você só precisa guardar o kite e entrar no clima. 🔥
              </p>
            </div>
          </div>
        </div>
      </section>

      <!-- SECTION: PREÇO / CTA -->
      <section id="preco" class="border-t border-line/60 py-20">
        <div class="mx-auto max-w-5xl px-6 text-center">
          <p class="eyebrow mb-2 justify-center">OPÇÕES DE INGRESSO</p>
          <h2 class="font-serif text-3xl font-bold text-ink md:text-5xl">
            Garanta seu lugar <span class="text-magenta">no luau.</span>
          </h2>
          <p class="mt-3 text-sm text-ink-soft">
            Ingresso com lycra exclusiva P5 — Temporada 2026. Pagamento seguro via Pix.
          </p>

          <div class="mt-10 max-w-2xl mx-auto flex flex-col gap-3 text-left">
            <div
              v-for="p in products"
              :key="p.id"
              :class="[
                'flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 rounded-2xl border p-5 transition-all shadow-sm',
                p.featured ? 'border-magenta/80 bg-white ring-2 ring-magenta/20' : 'border-line/80 bg-white/80 hover:border-ink/40',
              ]"
            >
              <div class="flex flex-wrap items-center gap-2.5">
                <h3 class="font-sans text-base font-bold text-ink">{{ cleanTitle(p.title) }}</h3>
                <span v-if="p.featured" class="rounded-full bg-magenta/10 px-2.5 py-0.5 font-mono text-[9px] font-bold tracking-wider text-magenta uppercase">DESTAQUE</span>
              </div>

              <div class="flex w-full sm:w-auto items-center justify-between sm:justify-end gap-5">
                <span class="font-serif text-2xl font-black text-ink">{{ formatBRL(p.priceCents) }}</span>
                <button class="button-magenta px-5 py-2.5 text-xs font-bold shrink-0" @click="router.push('/comprar')">
                  Garantir <ArrowRight :size="14" />
                </button>
              </div>
            </div>
          </div>

          <p class="mt-8 text-xs italic font-medium text-ink-soft">Vagas limitadas · Pagamento via Pix.</p>
        </div>
      </section>

      <!-- FOOTER -->
      <footer class="border-t border-line/80 bg-ink py-12 text-white">
        <div class="mx-auto flex max-w-6xl flex-col items-center justify-between gap-6 px-6 md:flex-row">
          <img src="/logo-downwind-dark.webp" alt="P5 DownWind Day" class="h-8 w-auto" />

          <p class="text-xs text-white/60 text-center md:text-left">
            P5 Kite House.<br />
            Do sol à lua. Do mar ao luau.
          </p>

          <div class="flex items-center gap-6 text-xs font-medium text-white/80">
            <button class="hover:text-white" @click="router.push('/entrar')">Área do aluno</button>
            <button class="hover:text-white" @click="router.push('/acesso-admin')">Acesso equipe</button>
          </div>
        </div>

        <div class="mt-8 flex justify-center border-t border-white/10 px-6 pt-6">
          <a
            href="https://prolins.com.br"
            target="_blank"
            rel="noopener noreferrer"
            aria-label="Powered by Prolins Software House e Outsource"
            class="opacity-80 transition-opacity hover:opacity-100"
          >
            <img src="/prolins-selo.webp" alt="Powered by Prolins Software House e Outsource" width="56" height="56" class="h-14 w-14" />
          </a>
        </div>
      </footer>
    </main>
  </div>
</template>
