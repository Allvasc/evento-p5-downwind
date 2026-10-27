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
  Wind,
  Droplets,
  Clock,
  Flag,
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
  capacity: number;
  booked: number;
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
  if (!activeProduct.value) return "R$ 100";
  return formatBRL(activeProduct.value.priceCents);
});

const productTitle = computed(() => cleanTitle(activeProduct.value?.title ?? "P5 DownWind Day"));

function hasLycra(p: Product) {
  return /com lycra/i.test(p.title);
}

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

// Apenas 16 vagas — a capacidade vem da turma, 16 é só o fallback.
const totalSpots = computed(() => nextSessions.value[0]?.capacity ?? 16);
const spotsLeft = computed(() => {
  const s = nextSessions.value[0];
  return s ? Math.max(s.capacity - s.booked, 0) : null;
});
const spotsLeftLabel = computed(() => {
  if (spotsLeft.value === null || spotsLeft.value >= totalSpots.value) return null;
  return spotsLeft.value === 1 ? "resta 1 vaga" : `restam ${spotsLeft.value} vagas`;
});

// Ativa o tema Halloween enquanto o produto halloween estiver ativo no banco.
// Quando a próxima migration desativar esse produto, os estilos somem automaticamente.
const isHalloween = computed(() =>
  products.value.some(p => /halloween/i.test(p.title))
);

const included = [
  { icon: MapPin, title: "Percurso guiado", detail: "Sababa → P5 Kite House, com toda a rota acompanhada pela equipe P5." },
  { icon: Bus, title: "Transporte incluso", detail: "Saída da P5 Kite House às 10h em direção ao Sababa." },
  { icon: LifeBuoy, title: "Apoio aquático e terrestre", detail: "Equipe P5 acompanhando você na água e em terra durante todo o percurso." },
  { icon: Droplets, title: "Estrutura na saída", detail: "Compressor, banheiro e hidratação no ponto de partida." },
  { icon: Wind, title: "Monitoramento Wind Maps", detail: "Condições de vento acompanhadas em tempo real." },
  { icon: HandHelping, title: "Receptivo na chegada", detail: "Auxílio na desmontagem do equipamento na P5 Kite House." },
  { icon: Clock, title: "Concentração às 9h30", detail: "Concentração na P5 Kite House." },
  { icon: Flag, title: "Evento à fantasia", detail: "Venha caracterizado e entre no clima do Halloween!" },
];

const itinerary = [
  { time: "1", title: "9h30 · Concentração", detail: "Concentração na P5 Kite House." },
  { time: "2", title: "10h · Saída para o Sababa", detail: "Saída de transporte da P5 rumo ao ponto de partida no Sababa." },
  { time: "3", title: "Estrutura na saída", detail: "Compressor, banheiro e hidratação no ponto de partida para preparar o equipamento." },
  { time: "4", title: "Sababa → P5", detail: "Percurso guiado com a equipe P5 acompanhando na água e em terra, do início ao fim." },
  { time: "5", title: "Chegada · P5 Kite House", detail: "Receptivo com auxílio na desmontagem do equipamento." },
];
</script>

<template>
  <div class="min-h-screen bg-paper">
    <WellnessHeader />
    <HelpFab />

    <main>
      <!-- HERO -->
      <section class="relative overflow-hidden pt-4 pb-16 md:py-20">
        <!-- orbs normais -->
        <template v-if="!isHalloween">
          <div class="hero-orb hero-orb--blue"></div>
          <div class="hero-orb hero-orb--cream"></div>
        </template>
        <!-- orbs Halloween -->
        <template v-else>
          <div class="hero-orb" style="background:radial-gradient(circle,rgba(124,58,237,.18) 0%,transparent 70%);top:-80px;left:-120px;width:600px;height:600px;"></div>
          <div class="hero-orb" style="background:radial-gradient(circle,rgba(249,115,22,.13) 0%,transparent 70%);bottom:-100px;right:-80px;width:500px;height:500px;"></div>
        </template>

        <!-- abóbora watermark -->
        <svg v-if="isHalloween" class="halloween-pumpkin" viewBox="0 0 200 220" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
          <path d="M100 32 C100 32 112 10 118 4 C122 0 128 2 124 10 C120 18 110 28 110 38" fill="#f97316"/>
          <ellipse cx="70" cy="118" rx="52" ry="72" fill="#f97316"/>
          <ellipse cx="100" cy="120" rx="42" ry="74" fill="#ea580c"/>
          <ellipse cx="130" cy="118" rx="52" ry="72" fill="#f97316"/>
          <ellipse cx="100" cy="118" rx="60" ry="70" fill="#f97316"/>
          <!-- linhas de nervura -->
          <path d="M100 50 Q85 118 100 188" stroke="#ea580c" stroke-width="2" opacity=".4"/>
          <path d="M100 50 Q70 110 72 188" stroke="#ea580c" stroke-width="2" opacity=".3"/>
          <path d="M100 50 Q130 110 128 188" stroke="#ea580c" stroke-width="2" opacity=".3"/>
          <!-- olhos -->
          <polygon points="76,100 66,120 86,120" fill="#1a0a00"/>
          <polygon points="124,100 114,120 134,120" fill="#1a0a00"/>
          <!-- nariz -->
          <polygon points="100,120 94,134 106,134" fill="#1a0a00"/>
          <!-- boca -->
          <path d="M72,148 L80,140 L88,150 L96,138 L104,150 L112,138 L120,150 L128,140 L136,148" stroke="#1a0a00" stroke-width="3" fill="none" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>

        <!-- morcegos decorativos -->
        <template v-if="isHalloween">
        <svg class="bat bat--1" viewBox="0 0 80 40" fill="currentColor" aria-hidden="true">
          <path d="M40 20 C35 10 20 5 0 12 C10 16 18 20 20 24 C22 28 18 34 14 36 C20 34 26 28 28 24 C30 20 34 18 40 20 Z"/>
          <path d="M40 20 C45 10 60 5 80 12 C70 16 62 20 60 24 C58 28 62 34 66 36 C60 34 54 28 52 24 C50 20 46 18 40 20 Z"/>
          <ellipse cx="40" cy="18" rx="5" ry="6"/>
        </svg>
        <svg class="bat bat--2" viewBox="0 0 60 30" fill="currentColor" aria-hidden="true">
          <path d="M30 15 C26 8 15 4 0 9 C8 12 14 15 15 18 C16 21 14 26 11 27 C15 26 20 21 21 18 C22 15 26 14 30 15 Z"/>
          <path d="M30 15 C34 8 45 4 60 9 C52 12 46 15 45 18 C44 21 46 26 49 27 C45 26 40 21 39 18 C38 15 34 14 30 15 Z"/>
          <ellipse cx="30" cy="14" rx="4" ry="5"/>
        </svg>
        <svg class="bat bat--3" viewBox="0 0 50 25" fill="currentColor" aria-hidden="true">
          <path d="M25 12 C22 7 12 3 0 7 C6 10 11 12 12 15 C13 17 11 21 9 22 C12 21 16 17 17 15 C18 12 21 11 25 12 Z"/>
          <path d="M25 12 C28 7 38 3 50 7 C44 10 39 12 38 15 C37 17 39 21 41 22 C38 21 34 17 33 15 C32 12 29 11 25 12 Z"/>
          <ellipse cx="25" cy="11" rx="3" ry="4"/>
        </svg>
        </template>

        <div class="relative z-10 mx-auto grid max-w-6xl gap-12 px-6 md:grid-cols-[1.1fr_0.9fr] md:items-center">
          <div>
            <p class="eyebrow mb-6 text-xs tracking-wider">
              <Compass :size="14" class="text-magenta" />
              P5 KITE HOUSE · DOWNWIND HALLOWEEN P5
            </p>

            <h1 class="font-serif text-5xl font-black leading-[1.05] text-ink md:text-6xl">
              Prepare sua fantasia e
              <span :style="isHalloween ? 'color:#c2410c;' : ''" :class="!isHalloween ? 'text-magenta' : ''">venha pro mar.</span>
            </h1>

            <p class="mt-6 max-w-md text-base leading-relaxed text-ink-soft md:text-lg">
              <strong class="font-semibold text-ink">Sábado, 31 de outubro</strong> · Sababa → P5 Kite House.
              Concentração às 9h30 na P5. Evento à fantasia com suporte aquático e terrestre incluídos.
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
              <span class="inline-flex items-center gap-1.5"><Check :size="15" :style="isHalloween ? 'color:#f97316;' : ''" :class="!isHalloween ? 'text-magenta' : ''" /> Concentração 9h30 · saída às 10h</span>
              <span class="inline-flex items-center gap-1.5"><Check :size="15" :style="isHalloween ? 'color:#f97316;' : ''" :class="!isHalloween ? 'text-magenta' : ''" /> Transporte + apoio na água e em terra</span>
              <span class="inline-flex items-center gap-1.5"><Check :size="15" :style="isHalloween ? 'color:#7c3aed;' : ''" :class="!isHalloween ? 'text-magenta' : ''" /> Apenas {{ totalSpots }} vagas<template v-if="spotsLeftLabel"> · {{ spotsLeftLabel }}</template></span>
            </div>
          </div>

          <!-- STACKED PRODUCT CARDS CONTAINER (Image 2 style) -->
          <div class="relative mx-auto w-full max-w-md">
            <div class="relative rounded-[2rem] border border-line/80 bg-white p-6 shadow-2xl md:p-8">
              <div class="text-center">
                <div class="inline-flex items-center gap-2">
                  <img src="/logo-downwind.webp" :alt="productTitle" class="h-7 w-auto" />
                  <span v-if="isHalloween" class="rounded-full border border-orange-400/40 bg-orange-50 px-2.5 py-0.5 font-mono text-[9px] font-bold tracking-wider text-orange-600 uppercase">🎃 Halloween</span>
                </div>
                <p class="mt-2 text-xs font-medium text-ink-soft">
                  Um valor, tudo incluso. Escolha seu ingresso.
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
                        v-if="hasLycra(p)"
                        class="rounded-full bg-magenta/10 px-2 py-0.5 font-mono text-[9px] font-bold text-magenta uppercase"
                      >
                        + Lycra P5
                      </span>
                    </div>
                    <p class="mt-1 text-xs text-ink-soft">
                      Sababa → P5 · suporte aquático e terrestre incluídos
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
                  <p class="font-mono text-[9px] font-bold uppercase tracking-wider text-ink-soft">Saída · 10h</p>
                  <p class="font-serif font-bold text-ink">Sababa</p>
                </div>
                <Wind :size="18" class="text-magenta shrink-0" />
                <div class="text-right">
                  <p class="font-mono text-[9px] font-bold uppercase tracking-wider text-ink-soft">Concentração · 9h30</p>
                  <p class="font-serif font-bold text-ink">P5 Kite House</p>
                </div>
              </div>
            </div>

            <!-- Floating Top Badge -->
            <span class="absolute -top-3 -right-2 inline-flex items-center gap-1.5 rounded-full border border-magenta/20 bg-white px-3.5 py-1 text-[11px] font-semibold text-magenta shadow-sm">
              <Compass :size="12" /> {{ nextDateLabel ?? "Sábado, 31 de outubro" }} · apenas {{ totalSpots }} vagas
            </span>

            <!-- Bottom Label -->
            <span class="absolute -bottom-3 left-6 rounded-full border border-line bg-white px-3.5 py-1 text-[11px] font-medium text-ink-soft shadow-sm">
              por P5 Kite House
            </span>
          </div>
        </div>
      </section>

      <!-- SECTION: O QUE ESTÁ INCLUSO -->
      <section
        id="inclui"
        class="relative overflow-hidden border-t border-line/60 py-20"
        :style="isHalloween ? 'background:linear-gradient(135deg,#fdf4ff 0%,#fff7ed 50%,#fdf4ff 100%);' : ''"
        :class="!isHalloween ? 'bg-warm/40' : ''"
      >
        <!-- teia de aranha decorativa -->
        <svg v-if="isHalloween" class="spiderweb" viewBox="0 0 120 120" fill="none" aria-hidden="true">
          <g stroke="#6d28d9" stroke-width="0.6" opacity="0.5">
            <line x1="60" y1="0" x2="60" y2="120"/><line x1="0" y1="60" x2="120" y2="60"/>
            <line x1="17" y1="17" x2="103" y2="103"/><line x1="103" y1="17" x2="17" y2="103"/>
            <circle cx="60" cy="60" r="15" fill="none"/><circle cx="60" cy="60" r="30" fill="none"/>
            <circle cx="60" cy="60" r="45" fill="none"/><circle cx="60" cy="60" r="58" fill="none"/>
          </g>
          <circle cx="60" cy="60" r="3" fill="#6d28d9" opacity="0.4"/>
        </svg>
        <div class="mx-auto max-w-6xl px-6">
          <p class="eyebrow mb-2" :style="isHalloween ? 'color:#7c3aed;' : ''">
            <template v-if="isHalloween">🎃 </template>TUDO INCLUÍDO NO SEU INGRESSO
          </p>
          <div class="grid gap-6 md:grid-cols-[1.2fr_1fr] md:items-end">
            <h2 class="font-serif text-3xl font-bold leading-tight text-ink md:text-5xl">
              O que está<br />
              <span :style="isHalloween ? 'color:#c2410c;' : ''" :class="!isHalloween ? 'text-magenta' : ''">incluso.</span>
            </h2>
            <p class="text-base leading-relaxed text-ink-soft">
              Da concentração às 9h30 na P5 Kite House à chegada no percurso Sababa → P5, passando pelo transporte e pelo apoio da equipe na água e em terra, tudo já está no seu ingresso.
            </p>
          </div>

          <div class="mt-12 grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
            <div
              v-for="(item, i) in included"
              :key="item.title"
              class="flex flex-col gap-4 rounded-[var(--radius-card)] border bg-white p-6"
              :style="isHalloween ? (i % 2 === 0 ? 'border-color:rgba(249,115,22,.25)' : 'border-color:rgba(124,58,237,.2)') : 'border-color:rgba(0,0,0,.08)'"
            >
              <span
                class="flex h-10 w-10 items-center justify-center rounded-full"
                :style="isHalloween
                  ? (i % 2 === 0 ? 'background:rgba(249,115,22,.1);color:#f97316;' : 'background:rgba(124,58,237,.1);color:#7c3aed;')
                  : 'background:rgba(var(--color-magenta),.1);color:var(--color-magenta,#e11d48);'"
                :class="!isHalloween ? 'bg-magenta/10 text-magenta' : ''"
              >
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
          <p class="eyebrow mb-2"><WaveIcon :size="14" /> SABABA → P5 KITE HOUSE · 31 DE OUTUBRO</p>
          <h2 class="font-serif text-3xl font-bold text-ink md:text-5xl">
            Como funciona <span class="text-magenta">o seu dia.</span>
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
          <p class="font-mono text-xs font-medium tracking-widest uppercase text-white/60">VENTO · MAR · KITE · APOIO</p>
          <h2 class="mt-2 font-serif text-3xl font-bold leading-tight md:text-4xl">
            Vento a favor, <span class="text-sky-300">mar em movimento.</span>
          </h2>

          <div class="mt-12 grid gap-8 md:grid-cols-3">
            <div class="flex flex-col gap-3">
              <LifeBuoy :size="26" class="text-sky-300" />
              <h3 class="font-serif text-lg font-bold">Apoio o tempo todo</h3>
              <p class="text-sm leading-relaxed text-white/70">
                Equipe P5 acompanhando na água e em terra durante todo o percurso, do início ao fim.
              </p>
            </div>
            <div class="flex flex-col gap-3">
              <Wind :size="26" class="text-sky-300" />
              <h3 class="font-serif text-lg font-bold">Monitoramento em tempo real</h3>
              <p class="text-sm leading-relaxed text-white/70">
                Condições de vento acompanhadas pelo aplicativo Wind Maps antes e durante o dia.
              </p>
            </div>
            <div class="flex flex-col gap-3">
              <Droplets :size="26" class="text-sky-300" />
              <h3 class="font-serif text-lg font-bold">Estrutura completa</h3>
              <p class="text-sm leading-relaxed text-white/70">
                Compressor, banheiro e hidratação no ponto de saída, mais receptivo na chegada.
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
            Um valor, <span class="text-magenta">tudo incluso.</span>
          </h2>
          <p class="mt-3 text-sm text-ink-soft">
            Apenas {{ totalSpots }} vagas. Sábado, 31 de outubro · Sababa → P5 Kite House. Evento à fantasia, ingresso único. Pagamento seguro via Pix.
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
                <span v-if="hasLycra(p)" class="rounded-full bg-yellow-300 px-2.5 py-0.5 font-mono text-[9px] font-bold tracking-wider text-ink uppercase">LYCRA EXCLUSIVA P5</span>
              </div>

              <div class="flex w-full sm:w-auto items-center justify-between sm:justify-end gap-5">
                <span class="font-serif text-2xl font-black text-ink">{{ formatBRL(p.priceCents) }}</span>
                <button class="button-magenta px-5 py-2.5 text-xs font-bold shrink-0" @click="router.push('/comprar')">
                  Garantir <ArrowRight :size="14" />
                </button>
              </div>
            </div>
          </div>

          <p class="mt-8 text-xs italic font-medium text-ink-soft">Apenas {{ totalSpots }} vagas<template v-if="spotsLeftLabel"> · {{ spotsLeftLabel }}</template> · Pagamento via Pix.</p>
        </div>
      </section>

      <!-- FOOTER -->
      <footer class="border-t border-line/80 bg-ink py-12 text-white">
        <div class="mx-auto flex max-w-6xl flex-col items-center justify-between gap-6 px-6 md:flex-row">
          <img src="/logo-downwind-dark.webp" alt="P5 DownWind Day" class="h-8 w-auto" />

          <p class="text-xs text-white/60 text-center md:text-left">
            P5 Kite House.<br />
            Vento a favor, mar em movimento.
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

<style scoped>
/* ── Abóbora watermark ─────────────────────────────────────────── */
.halloween-pumpkin {
  position: absolute;
  right: -6%;
  bottom: -8%;
  width: 420px;
  opacity: 0.045;
  pointer-events: none;
  z-index: 0;
}

/* ── Morcegos ──────────────────────────────────────────────────── */
.bat {
  position: absolute;
  pointer-events: none;
  z-index: 0;
  color: #6d28d9;
}
.bat--1 { top: 12%; left: 8%;   width: 56px; opacity: 0.07; animation: bat-float 9s  ease-in-out infinite; }
.bat--2 { top: 6%;  left: 38%;  width: 42px; opacity: 0.06; animation: bat-float 7s  ease-in-out infinite 1.5s; }
.bat--3 { top: 22%; right: 12%; width: 34px; opacity: 0.05; animation: bat-float 11s ease-in-out infinite 3s; }

@keyframes bat-float {
  0%, 100% { transform: translateY(0)     rotate(-4deg); }
  50%       { transform: translateY(-10px) rotate(4deg); }
}

/* ── Teia de aranha ────────────────────────────────────────────── */
.spiderweb {
  position: absolute;
  top: -10px;
  right: 0;
  width: 140px;
  opacity: 0.18;
  pointer-events: none;
}
</style>
