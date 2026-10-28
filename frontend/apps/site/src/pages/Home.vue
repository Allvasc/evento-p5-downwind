<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useRouter } from "vue-router";
import { useQuery } from "@tanstack/vue-query";
import {
  ArrowRight,
  Compass,
  MapPin,
  Bus,
  LifeBuoy,
  HandHelping,
  Wind,
  Droplets,
  Clock,
  Flag,
  Award,
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

// ── Easter egg interativo da abóbora ─────────────────────────────────────────
const pumpkinBoo = ref(false);
function triggerPumpkinBoo() {
  pumpkinBoo.value = true;
  setTimeout(() => {
    pumpkinBoo.value = false;
  }, 2400);
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

const productTitle = computed(() => cleanTitle(activeProduct.value?.title ?? "Downwind Halloween P5"));

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
  if (!sessions.length) return "Sábado, 31 de outubro";
  return formatEventDate(sessions[0].startsAt);
});

const totalSpots = computed(() => nextSessions.value[0]?.capacity ?? 16);
const spotsLeft = computed(() => {
  const s = nextSessions.value[0];
  return s ? Math.max(s.capacity - s.booked, 0) : null;
});
const spotsLeftLabel = computed(() => {
  if (spotsLeft.value === null || spotsLeft.value >= totalSpots.value) return null;
  return spotsLeft.value === 1 ? "resta 1 vaga" : `restam ${spotsLeft.value} vagas`;
});

// Ativa o tema Halloween para a edição de 31 de outubro
const isHalloween = computed(() => {
  if (products.value.length > 0) {
    return products.value.some(p => /halloween/i.test(p.title));
  }
  return true;
});

const included = computed(() => {
  if (isHalloween.value) {
    return [
      {
        icon: MapPin,
        title: "Percurso Guiado Sababa → P5",
        detail: "Trajeto 100% escoltado na água e em terra pela equipe P5 durante toda a descida.",
        tag: "Sababa → P5",
      },
      {
        icon: Bus,
        title: "Transporte dos Bruxos",
        detail: "Saída confortável da P5 Kite House às 10h da manhã direto para o ponto de largada no Sababa.",
        tag: "Saída 10h",
      },
      {
        icon: LifeBuoy,
        title: "Apoio Aquático e Terrestre",
        detail: "Jet ski na água e resgate de prontidão na praia para você velejar sem nenhum receio.",
        tag: "Jet Ski & Equipe",
      },
      {
        icon: Droplets,
        title: "Caldeirão de Hidratação & Ar",
        detail: "Compressor para inflar seu kite na praia, banheiros e hidratação gelada no ponto de partida.",
        tag: "Estrutura na Saída",
      },
      {
        icon: Wind,
        title: "Monitoramento Wind Maps",
        detail: "Condições de vento e maré acompanhadas em tempo real pela tecnologia do Wind Maps.",
        tag: "Vento em Tempo Real",
      },
      {
        icon: HandHelping,
        title: "Receptivo dos Sobreviventes",
        detail: "Auxílio total para pousar o kite e desmontar seu equipamento na chegada à P5 Kite House.",
        tag: "Pouso & Apoio P5",
      },
      {
        icon: Clock,
        title: "Concentração às 9h30",
        detail: "Encontro dos velejadores fantasiados na P5 Kite House para fotos, briefing e saída pontual às 10h.",
        tag: "Manhã de Sol",
      },
      {
        icon: Award,
        title: "Velejo à Fantasia & Brindes",
        detail: "Venha caracterizado sob o sol! Capas, máscaras e pinturas. A fantasia mais criativa ganha brinde!",
        tag: "Concurso Simbólico",
      },
    ];
  }

  return [
    { icon: MapPin, title: "Percurso guiado", detail: "Sababa → P5 Kite House, com toda a rota acompanhada pela equipe P5.", tag: "Rota Guiada" },
    { icon: Bus, title: "Transporte incluso", detail: "Saída da P5 Kite House às 10h em direção ao Sababa.", tag: "Transporte P5" },
    { icon: LifeBuoy, title: "Apoio aquático e terrestre", detail: "Equipe P5 acompanhando você na água e em terra durante todo o percurso.", tag: "Apoio Seguro" },
    { icon: Droplets, title: "Estrutura na saída", detail: "Compressor, banheiro e hidratação no ponto de partida.", tag: "Estrutura" },
    { icon: Wind, title: "Monitoramento Wind Maps", detail: "Condições de vento acompanhadas em tempo real.", tag: "Vento" },
    { icon: HandHelping, title: "Receptivo na chegada", detail: "Auxílio na desmontagem do equipamento na P5 Kite House.", tag: "Receptivo" },
    { icon: Clock, title: "Concentração às 9h30", detail: "Concentração na P5 Kite House.", tag: "Horário" },
    { icon: Flag, title: "Evento à fantasia", detail: "Venha caracterizado e entre no clima do Halloween!", tag: "Fantasia" },
  ];
});

const itinerary = computed(() => {
  if (isHalloween.value) {
    return [
      { time: "1", title: "9h30 · Caldeirão P5 (Concentração)", detail: "Encontro dos velejadores fantasiados na P5 Kite House para briefing e fotos sob o sol da manhã." },
      { time: "2", title: "10h00 · Expresso Rumo ao Sababa", detail: "Saída de transporte exclusivo com pranchas e kites em direção à largada no Sababa." },
      { time: "3", title: "10h30 · Preparação & Hidratação", detail: "Compressor na praia para encher os kites, hidratação gelada e últimos ajustes nas fantasias." },
      { time: "4", title: "11h00 · Velejo Sababa → P5", detail: "Grande descida de downwind com vento a favor, jet ski de apoio e a equipe P5 escoltando." },
      { time: "5", title: "13h00 · Chegada & Celebração", detail: "Receptivo na P5 Kite House com auxílio na desmontagem e confraternização dos velejadores." },
    ];
  }

  return [
    { time: "1", title: "9h30 · Concentração", detail: "Concentração na P5 Kite House." },
    { time: "2", title: "10h · Saída para o Sababa", detail: "Saída de transporte da P5 rumo ao ponto de partida no Sababa." },
    { time: "3", title: "Estrutura na saída", detail: "Compressor, banheiro e hidratação no ponto de partida para preparar o equipamento." },
    { time: "4", title: "Sababa → P5", detail: "Percurso guiado com a equipe P5 acompanhando na água e em terra, do início ao fim." },
    { time: "5", title: "Chegada · P5 Kite House", detail: "Receptivo com auxílio na desmontagem do equipamento." },
  ];
});
</script>

<template>
  <div :class="['min-h-screen transition-colors duration-300', isHalloween ? 'bg-[#fbf7f4] text-slate-900' : 'bg-paper']">
    <WellnessHeader :is-halloween="isHalloween" />
    <HelpFab />

    <main>
      <!-- ════════════════════════════════════════════════════════════════════════
           HERO SECTION: HALLOWEEN TROPICAL DE DIA
           ════════════════════════════════════════════════════════════════════════ -->
      <section class="relative overflow-hidden pt-4 pb-16 md:py-20">
        <!-- Fundos e orbs decorativos de dia -->
        <template v-if="!isHalloween">
          <div class="hero-orb hero-orb--blue"></div>
          <div class="hero-orb hero-orb--cream"></div>
        </template>
        <template v-else>
          <!-- Brilho de sol e atmosfera tropical de Halloween -->
          <div class="hero-orb" style="background: radial-gradient(circle, rgba(251, 146, 60, 0.22) 0%, rgba(253, 224, 71, 0.12) 40%, transparent 70%); top: -100px; left: -100px; width: 620px; height: 620px;"></div>
          <div class="hero-orb" style="background: radial-gradient(circle, rgba(168, 85, 247, 0.16) 0%, rgba(56, 189, 248, 0.10) 50%, transparent 70%); bottom: -80px; right: -80px; width: 560px; height: 560px;"></div>
        </template>

        <!-- Morcegos voando no céu azul com asas batendo -->
        <template v-if="isHalloween">
          <div class="pointer-events-none absolute inset-0 overflow-hidden z-10" aria-hidden="true">
            <!-- Morcego 1 -->
            <svg class="bat-day bat-day--1" viewBox="0 0 100 50" fill="#4c1d95">
              <path class="bat-wing-left" d="M50 25 C40 10, 20 2, 0 12 C12 18, 22 24, 25 30 C28 35, 22 42, 18 45 C25 42, 32 35, 35 30 C38 25, 42 22, 50 25 Z" />
              <path class="bat-wing-right" d="M50 25 C60 10, 80 2, 100 12 C88 18, 78 24, 75 30 C72 35, 78 42, 82 45 C75 42, 68 35, 65 30 C62 25, 58 22, 50 25 Z" />
              <ellipse cx="50" cy="24" rx="6" ry="8" fill="#3b0764" />
              <!-- orelhinhas -->
              <polygon points="46,18 48,12 50,18" fill="#3b0764" />
              <polygon points="54,18 52,12 50,18" fill="#3b0764" />
            </svg>
            <!-- Morcego 2 -->
            <svg class="bat-day bat-day--2" viewBox="0 0 100 50" fill="#6d28d9">
              <path class="bat-wing-left" d="M50 25 C40 10, 20 2, 0 12 C12 18, 22 24, 25 30 C28 35, 22 42, 18 45 C25 42, 32 35, 35 30 C38 25, 42 22, 50 25 Z" />
              <path class="bat-wing-right" d="M50 25 C60 10, 80 2, 100 12 C88 18, 78 24, 75 30 C72 35, 78 42, 82 45 C75 42, 68 35, 65 30 C62 25, 58 22, 50 25 Z" />
              <ellipse cx="50" cy="24" rx="6" ry="8" fill="#581c87" />
            </svg>
            <!-- Morcego 3 -->
            <svg class="bat-day bat-day--3" viewBox="0 0 100 50" fill="#ea580c">
              <path class="bat-wing-left" d="M50 25 C40 10, 20 2, 0 12 C12 18, 22 24, 25 30 C28 35, 22 42, 18 45 C25 42, 32 35, 35 30 C38 25, 42 22, 50 25 Z" />
              <path class="bat-wing-right" d="M50 25 C60 10, 80 2, 100 12 C88 18, 78 24, 75 30 C72 35, 78 42, 82 45 C75 42, 68 35, 65 30 C62 25, 58 22, 50 25 Z" />
              <ellipse cx="50" cy="24" rx="6" ry="8" fill="#c2410c" />
            </svg>
          </div>
        </template>

        <!-- Teia decorativa e aranha que balança no canto -->
        <div v-if="isHalloween" class="spider-corner pointer-events-none absolute -top-2 right-4 z-20 hidden sm:block">
          <svg class="spiderweb-svg" viewBox="0 0 120 120" fill="none">
            <g stroke="#9333ea" stroke-width="1" opacity="0.45">
              <line x1="120" y1="0" x2="0" y2="120"/>
              <line x1="120" y1="40" x2="40" y2="120"/>
              <line x1="120" y1="80" x2="80" y2="120"/>
              <line x1="80" y1="0" x2="0" y2="80"/>
              <line x1="40" y1="0" x2="0" y2="40"/>
              <path d="M120 20 Q100 20 100 0" stroke-width="1.2"/>
              <path d="M120 50 Q80 50 80 0" stroke-width="1.2"/>
              <path d="M120 85 Q50 85 50 0" stroke-width="1.2"/>
            </g>
          </svg>
          <!-- Aranha pendurada no fio de seda -->
          <div class="spider-dangle">
            <div class="spider-thread"></div>
            <svg class="spider-body" viewBox="0 0 24 24" fill="#581c87">
              <ellipse cx="12" cy="14" rx="5" ry="6" />
              <circle cx="12" cy="7" r="3.5" fill="#3b0764" />
              <!-- olhos vermelhos -->
              <circle cx="11" cy="6.5" r="0.7" fill="#ef4444" />
              <circle cx="13" cy="6.5" r="0.7" fill="#ef4444" />
              <!-- perninhas -->
              <path d="M7 6 Q3 4 2 8" stroke="#581c87" stroke-width="1.2" fill="none" stroke-linecap="round"/>
              <path d="M17 6 Q21 4 22 8" stroke="#581c87" stroke-width="1.2" fill="none" stroke-linecap="round"/>
              <path d="M6 10 Q1 9 1 13" stroke="#581c87" stroke-width="1.2" fill="none" stroke-linecap="round"/>
              <path d="M18 10 Q23 9 23 13" stroke="#581c87" stroke-width="1.2" fill="none" stroke-linecap="round"/>
              <path d="M7 14 Q2 16 3 20" stroke="#581c87" stroke-width="1.2" fill="none" stroke-linecap="round"/>
              <path d="M17 14 Q22 16 21 20" stroke="#581c87" stroke-width="1.2" fill="none" stroke-linecap="round"/>
            </svg>
          </div>
        </div>

        <div class="relative z-10 mx-auto max-w-6xl px-6">
          <div class="grid gap-12 lg:grid-cols-[1.1fr_0.9fr] lg:items-center">
            <!-- COLUNA DA ESQUERDA: TEXTOS E CALL TO ACTION -->
            <div>
              <!-- Eyebrow temático com mascote fantasma transparente -->
              <div class="mb-5 inline-flex items-center gap-3 rounded-full border border-orange-300/80 bg-orange-100/70 py-1 pl-1.5 pr-4 text-xs font-bold text-orange-950 shadow-2xs backdrop-blur-xs">
                <img
                  v-if="isHalloween"
                  src="/halloween-ghost-kiter.png"
                  alt="Fantasma Kiter"
                  class="h-7 w-7 object-contain"
                />
                <span class="tracking-wider uppercase font-mono text-[11px]">
                  P5 KITE HOUSE · DOWNWIND HALLOWEEN DE DIA
                </span>
                <span class="text-sm">🎃</span>
              </div>

              <!-- Título principal -->
              <h1 class="font-serif text-4xl font-black leading-[1.08] text-slate-900 sm:text-5xl md:text-6xl">
                Prepare sua fantasia e
                <span
                  :class="[
                    'relative inline-block transition-all',
                    isHalloween
                      ? 'text-transparent bg-clip-text bg-gradient-to-r from-orange-600 via-amber-600 to-orange-500'
                      : 'text-magenta'
                  ]"
                >
                  venha pro mar!
                </span>
              </h1>

              <!-- Descrição do evento -->
              <p class="mt-6 max-w-lg text-base leading-relaxed text-slate-700 md:text-lg">
                <strong class="font-bold text-slate-900">Sábado de sol, 31 de outubro</strong> · Sababa → P5 Kite House.
                Concentração às 9h30 da manhã na P5. O velejo mais divertido do ano com fantasias, sol, vento constante e suporte aquático e terrestre total!
              </p>

              <!-- Botões de ação -->
              <div class="mt-8 flex flex-wrap items-center gap-4">
                <button
                  :class="[
                    'group cursor-pointer font-bold transition-all transform hover:-translate-y-0.5 active:translate-y-0',
                    isHalloween
                      ? 'bg-gradient-to-r from-orange-600 via-orange-500 to-amber-500 hover:from-orange-500 hover:to-amber-400 text-white rounded-full px-7 py-3.5 text-base shadow-[0_8px_30px_rgba(249,115,22,0.45)] hover:shadow-[0_12px_36px_rgba(249,115,22,0.6)] flex items-center gap-2.5 border border-orange-300/40'
                      : 'button-magenta'
                  ]"
                  @click="router.push('/comprar')"
                >
                  <span>Garantir minha vaga · {{ formattedPrice }}</span>
                  <ArrowRight :size="18" class="transition-transform group-hover:translate-x-1" />
                </button>
                <button
                  class="cursor-pointer text-sm font-bold text-slate-800 underline underline-offset-4 transition-colors hover:text-orange-600 flex items-center gap-1.5"
                  @click="scrollToInclui"
                >
                  <span>Ver o que está incluído</span>
                </button>
              </div>

              <!-- Destaques em pílulas com ícones nítidos -->
              <div class="mt-8 flex flex-wrap gap-x-5 gap-y-2.5 text-xs font-semibold text-slate-700">
                <span class="inline-flex items-center gap-1.5 rounded-full bg-white/80 border border-slate-200 px-3 py-1 shadow-2xs">
                  <span class="text-orange-600 text-sm">🎃</span>
                  Velejo à fantasia sob o sol
                </span>
                <span class="inline-flex items-center gap-1.5 rounded-full bg-white/80 border border-slate-200 px-3 py-1 shadow-2xs">
                  <span class="text-amber-500 text-sm">☀️</span>
                  Concentração 9h30 · Saída 10h
                </span>
                <span class="inline-flex items-center gap-1.5 rounded-full bg-white/80 border border-slate-200 px-3 py-1 shadow-2xs">
                  <span class="text-purple-600 text-sm">🏄‍♂️</span>
                  Jet ski e apoio no mar e em terra
                </span>
                <span class="inline-flex items-center gap-1.5 rounded-full bg-orange-50 border border-orange-300/60 px-3 py-1 text-orange-900 shadow-2xs">
                  <span class="text-orange-600 text-sm">⚡</span>
                  Apenas {{ totalSpots }} vagas<template v-if="spotsLeftLabel"> · {{ spotsLeftLabel }}</template>
                </span>
              </div>

              <!-- Banner interativo da Abóbora de praia (sem sobrepor textos) -->
              <div
                v-if="isHalloween"
                class="mt-6 flex items-center gap-4 rounded-2xl border-2 border-orange-300/80 bg-gradient-to-r from-orange-50 via-amber-50 to-orange-50 p-4 shadow-sm transition-transform hover:scale-[1.01] cursor-pointer relative select-none max-w-lg"
                @click="triggerPumpkinBoo"
              >
                <img
                  src="/halloween-beach-pumpkin.png"
                  alt="Abóbora com óculos escuros na praia"
                  class="w-16 h-16 sm:w-20 sm:h-20 object-contain shrink-0 filter drop-shadow-md"
                />
                <div>
                  <div class="flex items-center gap-2">
                    <span class="rounded-full bg-orange-600 px-2 py-0.5 text-[9px] font-bold text-white uppercase tracking-wider font-mono">
                      Velejo à Fantasia
                    </span>
                    <span class="text-[11px] font-bold text-orange-700">Clique na abóbora! 🎃</span>
                  </div>
                  <p class="mt-1 font-serif text-sm font-bold text-slate-900">
                    Venha caracterizado sob o sol do Ceará!
                  </p>
                  <p class="text-xs text-slate-600 mt-0.5">
                    Traga sua fantasia leve para a água. A mais criativa ganha brinde da P5!
                  </p>
                </div>
                <!-- Balão interativo BOO -->
                <transition enter-active-class="transition duration-200 ease-out" enter-from-class="transform scale-75 opacity-0" enter-to-class="transform scale-100 opacity-100" leave-active-class="transition duration-150 ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
                  <div
                    v-if="pumpkinBoo"
                    class="absolute -top-3 right-4 whitespace-nowrap rounded-xl bg-orange-600 px-3 py-1 text-xs font-black text-white shadow-xl border-2 border-yellow-300 animate-bounce"
                  >
                    🎃 BOO! Vento a favor! 👻
                  </div>
                </transition>
              </div>
            </div>

            <!-- COLUNA DA DIREITA: CARD DO PRODUTO UNIFICADO COM FOTO INTEGRADA -->
            <div class="relative mx-auto w-full max-w-md">
              <!-- CARD PRINCIPAL DE INGRESSO -->
              <div class="relative rounded-[2rem] border-2 border-orange-400/40 bg-white shadow-xl overflow-hidden backdrop-blur-sm">
                <!-- Foto de capa do evento de dia integrada ao card -->
                <div v-if="isHalloween" class="relative h-44 overflow-hidden bg-slate-950">
                  <img
                    src="/halloween-day-hero.jpg"
                    alt="Downwind de Halloween de dia no Ceará - Velejador fantasiado"
                    class="w-full h-full object-cover object-center"
                  />
                  <div class="absolute inset-0 bg-gradient-to-t from-slate-950/80 via-transparent to-transparent flex items-end p-4">
                    <div class="text-white">
                      <span class="inline-flex items-center gap-1.5 rounded-full bg-orange-600 px-2 py-0.5 font-mono text-[9px] font-bold text-white uppercase tracking-wider">
                        ☀️ Edição de Dia · 31 de Outubro
                      </span>
                      <p class="font-serif text-sm font-bold text-white mt-1 drop-shadow-sm">
                        Sababa → P5 Kite House
                      </p>
                    </div>
                  </div>
                </div>

                <div class="p-6">
                  <div class="text-center">
                    <div class="inline-flex items-center gap-2">
                      <img src="/logo-downwind.webp" :alt="productTitle" class="h-6 w-auto" />
                      <span
                        v-if="isHalloween"
                        class="rounded-full border border-orange-500/40 bg-orange-50 px-2.5 py-0.5 font-mono text-[10px] font-bold tracking-wider text-orange-600 uppercase"
                      >
                        🎃 Ed. Halloween
                      </span>
                    </div>
                    <p class="mt-2 text-xs font-medium text-slate-600">
                      Ingresso único com tudo incluso. Vagas estritamente limitadas.
                    </p>
                  </div>

                  <!-- Stacked Product Cards -->
                  <div class="mt-5 space-y-3">
                    <div
                      v-for="(p, index) in products"
                      :key="p.id"
                      :class="[
                        'relative flex cursor-pointer items-center justify-between rounded-2xl border-2 p-3.5 transition-all',
                        selectedProductIndex === index
                          ? 'border-orange-500 bg-orange-50/50 ring-2 ring-orange-500/20 shadow-md'
                          : 'border-slate-200 bg-slate-50/60 hover:border-orange-400/60 hover:bg-white',
                      ]"
                      @click="selectedProductIndex = index"
                    >
                      <div class="pr-2">
                        <div class="flex flex-wrap items-center gap-2">
                          <h3 class="font-serif text-sm font-bold text-slate-900">{{ cleanTitle(p.title) }}</h3>
                          <span
                            v-if="hasLycra(p)"
                            class="rounded-full bg-purple-100 border border-purple-300 px-2 py-0.5 font-mono text-[9px] font-bold text-purple-700 uppercase"
                          >
                            + Lycra P5
                          </span>
                        </div>
                        <p class="mt-1 text-xs text-slate-500">
                          Sababa → P5 · apoio aquático e terrestre incluídos
                        </p>
                      </div>

                      <div class="shrink-0 text-right">
                        <span class="font-sans text-base font-bold text-orange-600">{{ formatBRL(p.priceCents) }}</span>
                      </div>
                    </div>
                  </div>

                  <!-- Route details footer inside card -->
                  <div class="mt-5 flex items-center justify-between border-t border-slate-200 pt-3 text-xs text-slate-600">
                    <div>
                      <p class="font-mono text-[9px] font-bold uppercase tracking-wider text-orange-600">Saída · 10h</p>
                      <p class="font-serif font-bold text-slate-900">Sababa</p>
                    </div>
                    <div class="flex flex-col items-center">
                      <Wind :size="16" class="text-orange-500 shrink-0" />
                      <span class="font-mono text-[9px] text-slate-400">vento a favor</span>
                    </div>
                    <div class="text-right">
                      <p class="font-mono text-[9px] font-bold uppercase tracking-wider text-purple-600">Concentração · 9h30</p>
                      <p class="font-serif font-bold text-slate-900">P5 Kite House</p>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Floating Top Badge -->
              <span class="absolute -top-3.5 -right-2 z-20 inline-flex items-center gap-1.5 rounded-full border-2 border-orange-400 bg-orange-500 px-3.5 py-1 text-[11px] font-bold text-white shadow-md">
                <Compass :size="12" /> {{ nextDateLabel }} · apenas {{ totalSpots }} vagas
              </span>

              <!-- Bottom Label -->
              <span class="absolute -bottom-3 left-6 z-20 rounded-full border border-slate-200 bg-white px-3.5 py-1 text-[11px] font-semibold text-slate-600 shadow-sm">
                por P5 Kite House
              </span>
            </div>
          </div>
        </div>
      </section>

      <!-- ════════════════════════════════════════════════════════════════════════
           SECTION: O QUE ESTÁ INCLUSO NO CALDEIRÃO
           ════════════════════════════════════════════════════════════════════════ -->
      <section
        id="inclui"
        class="relative overflow-hidden border-t border-orange-200/60 py-20 bg-gradient-to-b from-orange-50/60 via-purple-50/40 to-white"
      >
        <div class="relative z-10 mx-auto max-w-6xl px-6">
          <!-- Cabeçalho da seção com o Caldeirão de Bruxa bem integrado -->
          <div class="flex flex-col sm:flex-row sm:items-center gap-5 mb-10">
            <img
              v-if="isHalloween"
              src="/halloween-cauldron.png"
              alt="Caldeirão Mágico de Halloween"
              class="w-20 h-20 sm:w-24 sm:h-24 object-contain filter drop-shadow-lg shrink-0"
            />
            <div>
              <div class="flex items-center gap-2 mb-1">
                <span class="text-xl">🎃</span>
                <p class="eyebrow" :style="isHalloween ? 'color:#c2410c;' : ''">
                  100% INCLUÍDO NO SEU INGRESSO
                </p>
              </div>
              <h2 class="font-serif text-3xl font-bold leading-tight text-slate-900 md:text-5xl">
                Tudo o que está <span :class="isHalloween ? 'text-transparent bg-clip-text bg-gradient-to-r from-orange-600 via-amber-600 to-orange-500' : 'text-magenta'">incluso no seu dia.</span>
              </h2>
              <p class="mt-2 text-base leading-relaxed text-slate-600 max-w-2xl">
                Da concentração às 9h30 na P5 Kite House à chegada ao percurso Sababa → P5, passando pelo transporte e pelo apoio de elite na água e em terra, tudo já está coberto pelo seu ingresso.
              </p>
            </div>
          </div>

          <!-- Cards dos itens inclusos -->
          <div class="mt-12 grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
            <div
              v-for="(item, i) in included"
              :key="item.title"
              class="group relative flex flex-col justify-between rounded-2xl border-2 bg-white p-6 transition-all duration-300 hover:-translate-y-1 hover:shadow-lg"
              :class="isHalloween ? (i % 2 === 0 ? 'border-orange-200 hover:border-orange-400' : 'border-purple-200 hover:border-purple-400') : 'border-slate-200'"
            >
              <div>
                <div class="flex items-center justify-between mb-4">
                  <span
                    class="flex h-12 w-12 items-center justify-center rounded-2xl shadow-xs transition-transform group-hover:scale-110"
                    :class="isHalloween
                      ? (i % 2 === 0 ? 'bg-orange-100 text-orange-600 border border-orange-200' : 'bg-purple-100 text-purple-700 border border-purple-200')
                      : 'bg-magenta/10 text-magenta'"
                  >
                    <component :is="item.icon" :size="22" />
                  </span>
                  <span
                    v-if="item.tag"
                    class="rounded-full px-2.5 py-0.5 font-mono text-[9px] font-bold tracking-wider uppercase"
                    :class="isHalloween
                      ? (i % 2 === 0 ? 'bg-orange-50 text-orange-700 border border-orange-200' : 'bg-purple-50 text-purple-700 border border-purple-200')
                      : 'bg-slate-100 text-slate-600'"
                  >
                    {{ item.tag }}
                  </span>
                </div>
                <h3 class="font-sans text-base font-bold text-slate-900 group-hover:text-orange-600 transition-colors">
                  {{ item.title }}
                </h3>
                <p class="mt-2 text-xs leading-relaxed text-slate-600">{{ item.detail }}</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- ════════════════════════════════════════════════════════════════════════
           SECTION: COMO FUNCIONA O SEU DIA (ITINERÁRIO)
           ════════════════════════════════════════════════════════════════════════ -->
      <section id="percurso" class="py-20 bg-white border-t border-slate-200/80">
        <div class="mx-auto max-w-6xl px-6">
          <div class="flex items-center gap-2 mb-2">
            <span class="text-lg">🌊</span>
            <p class="eyebrow"><WaveIcon :size="14" /> CRONOGRAMA OFICIAL · SÁBADO, 31 DE OUTUBRO</p>
          </div>
          <h2 class="font-serif text-3xl font-bold text-slate-900 md:text-5xl">
            Como funciona <span class="text-orange-600">o seu dia no mar.</span>
          </h2>
          <p class="mt-3 text-sm text-slate-600 max-w-xl">
            Um cronograma pontual para você aproveitar cada minuto do vento, da praia e da diversão com tranquilidade.
          </p>

          <div class="mt-14 grid gap-8 sm:grid-cols-2 md:grid-cols-5">
            <div v-for="(step, i) in itinerary" :key="step.title" class="relative group">
              <div class="flex items-center gap-3">
                <span
                  class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl font-mono text-base font-black shadow-sm transition-transform group-hover:scale-105"
                  :class="isHalloween
                    ? (i === 3 ? 'bg-gradient-to-br from-orange-500 to-amber-500 text-white shadow-orange-300' : 'bg-slate-900 text-white')
                    : 'bg-ink text-white'"
                >
                  {{ step.time }}
                </span>
                <div v-if="i < itinerary.length - 1" class="hidden h-0.5 flex-1 bg-slate-200 md:block"></div>
              </div>
              <h3 class="mt-5 font-serif text-base font-bold text-slate-900 group-hover:text-orange-600 transition-colors">
                {{ step.title }}
              </h3>
              <p class="mt-2 text-xs leading-relaxed text-slate-600">{{ step.detail }}</p>
            </div>
          </div>
        </div>
      </section>

      <!-- ════════════════════════════════════════════════════════════════════════
           SECTION: SEGURANÇA E CONFIANÇA (VENTO A FAVOR, MAR SEM SUSTOS)
           ════════════════════════════════════════════════════════════════════════ -->
      <section class="border-t border-purple-900/60 bg-[#16082e] py-20 text-white relative overflow-hidden">
        <!-- Detalhes de iluminação de fundo -->
        <div class="hero-orb" style="background: radial-gradient(circle, rgba(249, 115, 22, 0.15) 0%, transparent 70%); top: -50px; right: 10%; width: 500px; height: 500px;"></div>

        <div class="relative z-10 mx-auto max-w-6xl px-6">
          <div class="mb-6 inline-flex h-16 w-16 items-center justify-center rounded-2xl bg-white/10 border border-white/20 shadow-lg">
            <KitesurferIcon :size="34" class="text-orange-400" />
          </div>
          <p class="font-mono text-xs font-semibold tracking-widest uppercase text-orange-300">
            🎃 SEGURANÇA TOTAL · APOIO NA ÁGUA E EM TERRA
          </p>
          <h2 class="mt-2 font-serif text-3xl font-bold leading-tight md:text-4xl text-white">
            Vento a favor, <span class="text-transparent bg-clip-text bg-gradient-to-r from-orange-400 to-amber-300">mar sem sustos.</span>
          </h2>
          <p class="mt-2 text-sm text-purple-200/80 max-w-2xl">
            A diversão é garantida e o tema é de arrepiar, mas na segurança a seriedade da P5 Kite House é absoluta. Você veleja 100% amparado do primeiro ao último metro.
          </p>

          <div class="mt-12 grid gap-8 md:grid-cols-3">
            <div class="flex flex-col gap-3 rounded-2xl bg-white/5 border border-white/10 p-6 backdrop-blur-xs">
              <div class="flex items-center gap-3">
                <div class="p-2.5 rounded-xl bg-orange-500/20 text-orange-400 border border-orange-400/30">
                  <LifeBuoy :size="24" />
                </div>
                <h3 class="font-serif text-lg font-bold text-white">Apoio aquático e terrestre</h3>
              </div>
              <p class="text-sm leading-relaxed text-purple-200/70">
                Jet ski de prontidão durante todo o trajeto marítimo e veículos de apoio terrestre na faixa de areia para qualquer eventualidade.
              </p>
            </div>

            <div class="flex flex-col gap-3 rounded-2xl bg-white/5 border border-white/10 p-6 backdrop-blur-xs">
              <div class="flex items-center gap-3">
                <div class="p-2.5 rounded-xl bg-purple-500/20 text-purple-300 border border-purple-400/30">
                  <Wind :size="24" />
                </div>
                <h3 class="font-serif text-lg font-bold text-white">Monitoramento Wind Maps</h3>
              </div>
              <p class="text-sm leading-relaxed text-purple-200/70">
                Condições de vento e ondulação acompanhadas por telemetria em tempo real antes e durante a descida, garantindo a janela ideal.
              </p>
            </div>

            <div class="flex flex-col gap-3 rounded-2xl bg-white/5 border border-white/10 p-6 backdrop-blur-xs">
              <div class="flex items-center gap-3">
                <div class="p-2.5 rounded-xl bg-amber-500/20 text-amber-400 border border-amber-400/30">
                  <Droplets :size="24" />
                </div>
                <h3 class="font-serif text-lg font-bold text-white">Estrutura completa</h3>
              </div>
              <p class="text-sm leading-relaxed text-purple-200/70">
                Compressor de alta vazão no ponto de saída para você não cansar os braços, mais hidratação gelada e receptivo com auxílio na desmontagem.
              </p>
            </div>
          </div>
        </div>
      </section>

      <!-- ════════════════════════════════════════════════════════════════════════
           SECTION: PREÇO / CTA
           ════════════════════════════════════════════════════════════════════════ -->
      <section id="preco" class="border-t border-slate-200 py-20 bg-gradient-to-b from-white via-orange-50/40 to-white">
        <div class="mx-auto max-w-5xl px-6 text-center">
          <div class="inline-flex items-center gap-1.5 rounded-full bg-orange-100 border border-orange-300 px-3 py-1 text-xs font-bold text-orange-800 mb-3">
            <span>🎃</span> LOTE EXCLUSIVO HALLOWEEN
          </div>
          <h2 class="font-serif text-3xl font-bold text-slate-900 md:text-5xl">
            Um valor, <span class="text-orange-600">tudo incluso.</span>
          </h2>
          <p class="mt-3 text-sm text-slate-600 max-w-xl mx-auto">
            Apenas {{ totalSpots }} vagas. Sábado, 31 de outubro · Sababa → P5 Kite House. Velejo à fantasia com transporte, suporte aquático, hidratação e receptivo.
          </p>

          <div class="mt-10 max-w-2xl mx-auto flex flex-col gap-3 text-left">
            <div
              v-for="p in products"
              :key="p.id"
              :class="[
                'flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 rounded-2xl border-2 p-5 transition-all shadow-sm',
                p.featured
                  ? 'border-orange-500 bg-white ring-2 ring-orange-500/20'
                  : 'border-slate-200 bg-white/90 hover:border-orange-400',
              ]"
            >
              <div class="flex flex-wrap items-center gap-2.5">
                <span class="text-lg">🎃</span>
                <h3 class="font-sans text-base font-bold text-slate-900">{{ cleanTitle(p.title) }}</h3>
                <span v-if="hasLycra(p)" class="rounded-full bg-yellow-300 px-2.5 py-0.5 font-mono text-[9px] font-black tracking-wider text-slate-900 uppercase">
                  LYCRA EXCLUSIVA P5
                </span>
              </div>

              <div class="flex w-full sm:w-auto items-center justify-between sm:justify-end gap-5">
                <span class="font-serif text-2xl font-black text-slate-900">{{ formatBRL(p.priceCents) }}</span>
                <button
                  class="cursor-pointer bg-gradient-to-r from-orange-600 to-orange-500 hover:from-orange-500 hover:to-amber-500 text-white font-bold px-6 py-2.5 rounded-full text-xs shrink-0 shadow-md transition-all hover:scale-105 active:scale-95 flex items-center gap-1.5"
                  @click="router.push('/comprar')"
                >
                  <span>Garantir vaga</span>
                  <ArrowRight :size="14" />
                </button>
              </div>
            </div>
          </div>

          <!-- Lembrete da fantasia -->
          <div class="mt-8 mx-auto max-w-lg rounded-2xl border border-orange-300/80 bg-orange-100/60 p-4 text-xs font-semibold text-orange-950 flex items-center justify-center gap-2 shadow-2xs">
            <span class="text-lg">🧙‍♂️</span>
            <span>Capriche na fantasia! Teremos registros fotográficos e brinde simbólico para a caracterização mais criativa!</span>
          </div>

          <p class="mt-4 text-xs italic font-medium text-slate-500">
            Apenas {{ totalSpots }} vagas<template v-if="spotsLeftLabel"> · {{ spotsLeftLabel }}</template> · Pagamento seguro via Pix com confirmação instantânea.
          </p>
        </div>
      </section>

      <!-- ════════════════════════════════════════════════════════════════════════
           FOOTER
           ════════════════════════════════════════════════════════════════════════ -->
      <footer class="border-t border-slate-900 bg-slate-950 py-12 text-white">
        <div class="mx-auto flex max-w-6xl flex-col items-center justify-between gap-6 px-6 md:flex-row">
          <div class="flex items-center gap-3">
            <img src="/logo-downwind-dark.webp" alt="P5 DownWind Day" class="h-8 w-auto" />
            <span v-if="isHalloween" class="text-lg">🎃</span>
          </div>

          <p class="text-xs text-white/60 text-center md:text-left leading-relaxed">
            P5 Kite House · Downwind Halloween 2026.<br />
            Vento a favor, mar em movimento.
          </p>

          <div class="flex items-center gap-6 text-xs font-medium text-white/80">
            <button class="hover:text-orange-400 transition-colors" @click="router.push('/entrar')">Área do aluno</button>
            <button class="hover:text-orange-400 transition-colors" @click="router.push('/acesso-admin')">Acesso equipe</button>
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
/* ── Morcegos voando no céu azul com asas batendo ───────────────── */
.bat-day {
  position: absolute;
  pointer-events: none;
  z-index: 10;
  filter: drop-shadow(0 4px 6px rgba(0, 0, 0, 0.25));
}

.bat-day--1 {
  top: 8%;
  left: 6%;
  width: 64px;
  animation: bat-flight-1 12s ease-in-out infinite;
}

.bat-day--2 {
  top: 14%;
  left: 36%;
  width: 48px;
  animation: bat-flight-2 9s ease-in-out infinite 1.5s;
}

.bat-day--3 {
  top: 18%;
  right: 14%;
  width: 42px;
  animation: bat-flight-3 14s ease-in-out infinite 3s;
}

/* Animação do bater de asas dos morcegos */
.bat-wing-left {
  transform-origin: 50px 25px;
  animation: bat-flap-left 0.4s ease-in-out infinite alternate;
}
.bat-wing-right {
  transform-origin: 50px 25px;
  animation: bat-flap-right 0.4s ease-in-out infinite alternate;
}

@keyframes bat-flap-left {
  0% { transform: scaleY(1); }
  100% { transform: scaleY(-0.6) translateY(6px); }
}

@keyframes bat-flap-right {
  0% { transform: scaleY(1); }
  100% { transform: scaleY(-0.6) translateY(6px); }
}

@keyframes bat-flight-1 {
  0%, 100% { transform: translate(0, 0) rotate(-6deg); }
  50% { transform: translate(25px, -18px) rotate(8deg); }
}

@keyframes bat-flight-2 {
  0%, 100% { transform: translate(0, 0) rotate(5deg); }
  50% { transform: translate(-20px, -14px) rotate(-7deg); }
}

@keyframes bat-flight-3 {
  0%, 100% { transform: translate(0, 0) rotate(-4deg); }
  50% { transform: translate(18px, -12px) rotate(6deg); }
}

/* ── Teia de aranha e aranha balançando no canto ───────────────── */
.spider-corner {
  width: 140px;
  height: 140px;
}

.spiderweb-svg {
  width: 100%;
  height: 100%;
}

.spider-dangle {
  position: absolute;
  top: 35px;
  right: 45px;
  display: flex;
  flex-direction: column;
  align-items: center;
  animation: spider-sway 4s ease-in-out infinite;
}

.spider-thread {
  width: 1.5px;
  height: 38px;
  background: rgba(109, 40, 217, 0.45);
}

.spider-body {
  width: 22px;
  height: 22px;
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.3));
}

@keyframes spider-sway {
  0%, 100% { transform: rotate(8deg); }
  50% { transform: rotate(-8deg); }
}
</style>
