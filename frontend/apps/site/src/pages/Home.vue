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
import { formatBRL, cleanTitle, isUnlimitedCapacity, type Product } from "@p5wellness/shared";
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

// Gratuidade liga/desliga pelo preço do produto no Admin: R$ 0 = evento gratuito.
const isFree = computed(() => activeProduct.value?.priceCents === 0);

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
const unlimitedSpots = computed(() => !!nextSessions.value[0] && isUnlimitedCapacity(nextSessions.value[0].capacity));
// Texto das vagas da próxima turma — null até a turma carregar, para não exibir um número errado.
const spotsLabel = computed(() => {
  if (!nextSessions.value.length) return null;
  if (unlimitedSpots.value) return "Vagas ilimitadas";
  return `Apenas ${totalSpots.value} vagas` + (spotsLeftLabel.value ? ` · ${spotsLeftLabel.value}` : "");
});
const headerBadge = computed(() => [isFree.value ? "Gratuito" : "", spotsLabel.value ?? ""].filter(Boolean).join(" · "));

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
        detail: "Saída prevista às 10h da manhã da P5 Kite House direto para o ponto de largada no Sababa.",
        tag: "Saída prevista 10h",
      },
      {
        icon: LifeBuoy,
        title: "Apoio Aquático e Terrestre",
        detail: "Translado da P5 para ponto de partida (Sababa) e acompanhamento da equipe P5 no mar.",
        tag: "Segurança Total",
      },
      {
        icon: Droplets,
        title: "Caldeirão de Hidratação & Ar",
        detail: "Compressor para inflar seu kite na praia, banheiros e hidratação gelada no ponto de partida.",
        tag: "Estrutura na Saída",
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
        detail: "Encontro dos velejadores fantasiados na P5 Kite House para fotos, briefing e saída prevista às 10h.",
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
    { icon: Bus, title: "Transporte incluso", detail: "Saída prevista às 10h da P5 Kite House em direção ao Sababa.", tag: "Transporte P5" },
    {
      icon: LifeBuoy,
      title: "Apoio aquático e terrestre",
      detail: "Translado da P5 para ponto de partida (Sababa) e acompanhamento da equipe P5 no mar.",
      tag: "Segurança Total",
    },
    { icon: Droplets, title: "Estrutura na saída", detail: "Compressor, banheiro e hidratação no ponto de partida.", tag: "Estrutura" },
    { icon: HandHelping, title: "Receptivo na chegada", detail: "Auxílio na desmontagem do equipamento na P5 Kite House.", tag: "Receptivo" },
    { icon: Clock, title: "Concentração às 9h30", detail: "Concentração na P5 Kite House.", tag: "Horário" },
    { icon: Flag, title: "Evento à fantasia", detail: "Venha caracterizado e entre no clima do Halloween!", tag: "Fantasia" },
  ];
});

const itinerary = computed(() => {
  if (isHalloween.value) {
    return [
      { time: "1", title: "9h30 · Caldeirão P5 (Concentração)", detail: "Encontro dos velejadores fantasiados na P5 Kite House para briefing e fotos sob o sol da manhã." },
      { time: "2", title: "10h00 (previsto) · Expresso Rumo ao Sababa", detail: "Saída de transporte exclusivo com pranchas e kites em direção à largada no Sababa." },
      { time: "3", title: "10h30 · Preparação & Hidratação", detail: "Compressor na praia para encher os kites, hidratação gelada e últimos ajustes nas fantasias." },
      { time: "4", title: "11h00 · Velejo Sababa → P5", detail: "Grande descida de downwind com vento a favor e a equipe P5 escoltando na água e em terra." },
      { time: "5", title: "13h00 · Chegada & Celebração", detail: "Receptivo na P5 Kite House com auxílio na desmontagem e confraternização dos velejadores." },
    ];
  }

  return [
    { time: "1", title: "9h30 · Concentração", detail: "Concentração na P5 Kite House." },
    { time: "2", title: "10h (previsto) · Saída para o Sababa", detail: "Saída de transporte da P5 rumo ao ponto de partida no Sababa." },
    { time: "3", title: "Estrutura na saída", detail: "Compressor, banheiro e hidratação no ponto de partida para preparar o equipamento." },
    { time: "4", title: "Sababa → P5", detail: "Percurso guiado com a equipe P5 acompanhando na água e em terra, do início ao fim." },
    { time: "5", title: "Chegada · P5 Kite House", detail: "Receptivo com auxílio na desmontagem do equipamento." },
  ];
});
</script>

<template>
  <div :class="['min-h-screen transition-colors duration-300', isHalloween ? 'bg-[#fbf7f4] text-slate-900' : 'bg-paper']">
    <WellnessHeader :is-halloween="isHalloween" :banner-badge="headerBadge" />
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

        <!-- Morcegos voando no céu (Silhuetas fiéis nas margens do céu, sem cobrir textos) -->
        <template v-if="isHalloween">
          <div class="pointer-events-none absolute inset-0 overflow-hidden z-0" aria-hidden="true">
            <!-- Morcego 1: Líder na margem superior esquerda do céu -->
            <div class="bat-day bat-day--1">
              <svg class="bat-svg-inner" viewBox="15.5 7.5 225.4 124.4">
                <path fill="#1e1b4b" d="M236.062,70.277c-11.115-5.777-23.436-7.583-36.618-5.369c-15.044,2.527-29.146,9.812-40.501,15.676c-3.835,1.979-7.455,3.851-10.721,5.336c-1.468,0.67-2.826,1.112-4.044,1.317c-2.854,0.479-4.845-0.327-6.083-2.462c-3.323-5.721-3.104-19.211,4.01-26.125c0.508-0.493,0.508-1.304-0.003-1.846c-0.499-0.527-1.356-0.626-1.928-0.197c-1.915,1.441-4.288,2.432-6.863,2.865c-4.495,0.754-9.191-0.227-12.559-2.626c-2.737-1.952-4.427-4.716-4.884-7.995c-0.101-0.732-0.804-1.284-1.563-1.23c-0.775,0.054-1.314,0.692-1.232,1.424c0.65,5.78-2.178,25.02-11.104,29.953c-2.06,1.138-4.07-0.201-4.07-0.201c-3.197-2.016-7.639-10.04-8.135-10.948c-4.787-8.768-11.342-20.771-21.944-31.571c-12.515-12.75-28.166-21.05-46.514-24.67c-0.714-0.141-1.404,0.272-1.583,0.916c-0.184,0.666,0.193,1.372,0.875,1.645c4.573,1.825,13.094,7.328,16.022,16.37c2.313,7.139,0.471,14.763-5.474,22.658c-0.327,0.436-0.338,1.024-0.026,1.499c0.311,0.477,0.875,0.735,1.439,0.661c11.13-1.461,15.726,2.577,17.298,6.364c4.879,11.759-2.65,15.871-2.65,15.871c-0.643,0.356-0.858,0.923-0.669,1.536c0.195,0.629,0.826,1.059,1.498,1.021c10.236-0.571,30.378,9.618,43.107,21.803c8.75,8.379,13.327,16.995,12.888,24.263c-0.039,0.63,0.378,1.213,1.015,1.418c0.621,0.201,1.313-0.046,1.621-0.566c3.495-5.875,11.885-10.196,23.628-12.169c9.821-1.649,21.245-1.552,31.339,0.266c9.852,1.774,17.163,5.054,19.565,8.771c0.363,0.565,1.104,0.827,1.741,0.596c0.637-0.229,0.979-0.868,0.813-1.521c-1.035-4.088-0.159-8.504,2.465-12.433c2.859-4.277,7.401-7.307,12.154-8.105c3.127-0.525,6.163-0.091,9.021,1.288c0.488,0.236,1.09,0.185,1.501-0.134c0.416-0.321,0.587-0.858,0.438-1.368c-1.408-4.818-0.403-10.246,2.828-15.289c4.567-7.125,12.757-12.45,21.37-13.897c0.701-0.119,1.41-0.209,2.107-0.275c0.604-0.055,1.091-0.495,1.188-1.065C236.924,71.15,236.616,70.566,236.062,70.277z" />
              </svg>
            </div>

            <!-- Morcego 2: Majestoso de asas abertas na margem superior direita -->
            <div class="bat-day bat-day--2">
              <svg class="bat-svg-inner" viewBox="-4.0 98.6 520.0 314.4">
                <path fill="#0f172a" d="M511.636,269.628c-0.424-2.366-1.152-4.744-2.093-7.334c-1.777-4.829-4.386-10.368-7.734-16.616c-5.848-10.908-13.928-23.889-23.514-37.629c-14.377-20.602-32.091-42.859-50.606-61.877c-9.263-9.506-18.733-18.205-28.148-25.491c-9.433-7.28-18.8-13.17-28.111-16.974c-2.827-1.147-6.078-0.691-8.481,1.183c-2.402,1.887-3.628,4.932-3.191,7.953c0.449,3.13,0.661,6.164,0.661,9.124c-0.006,6.139-0.904,11.951-2.53,17.484c-2.833,9.682-7.892,18.563-14.14,26.462c-9.354,11.854-21.384,21.42-31.642,27.935c-5.12,3.258-9.797,5.764-13.376,7.432c-1.007,0.467-1.911,0.862-2.718,1.189c0.236-1.262,0.57-2.736,0.976-4.368c0.868-3.506,2.069-7.71,3.203-12.321c1.141-4.622,2.239-9.652,2.882-14.942c0.17-1.419,0.272-2.73,0.272-4.034c0-0.97-0.054-1.935-0.23-2.996c-0.133-0.795-0.334-1.644-0.698-2.584c-0.279-0.71-0.655-1.468-1.201-2.244c-0.801-1.159-2.05-2.36-3.573-3.136c-0.752-0.388-1.546-0.667-2.323-0.844c-0.782-0.176-1.529-0.248-2.221-0.242c-1.043,0-1.941,0.151-2.718,0.352c-1.48,0.4-2.536,0.94-3.458,1.474c-0.873,0.509-1.614,1.044-2.33,1.595c-1.34,1.037-2.584,2.172-3.876,3.397c-2.239,2.148-4.586,4.592-7.031,7.074c-1.832,1.862-3.712,3.737-5.575,5.478c-2.785,2.609-5.575,4.902-7.868,6.339c-1.153,0.722-2.16,1.226-2.912,1.499c-0.752,0.279-1.213,0.321-1.328,0.315c-0.073,0-0.261-0.012-0.564-0.084c-0.57-0.127-1.547-0.492-2.73-1.165c-2.087-1.159-4.75-3.209-7.45-5.648c-2.038-1.826-4.113-3.876-6.139-5.92c-1.529-1.548-3.021-3.082-4.477-4.556c-1.092-1.104-2.154-2.166-3.197-3.173c-1.577-1.505-3.07-2.87-4.762-4.095c-0.861-0.619-1.771-1.207-2.942-1.759c-0.588-0.273-1.256-0.534-2.044-0.752c-0.789-0.206-1.711-0.37-2.803-0.37c-0.813-0.006-1.711,0.097-2.645,0.352c-0.812,0.218-1.638,0.564-2.39,1.013c-1.335,0.783-2.402,1.875-3.125,2.924c-0.54,0.795-0.916,1.566-1.189,2.281c-0.406,1.073-0.612,2.026-0.734,2.918c-0.122,0.892-0.164,1.729-0.164,2.566c0,1.298,0.103,2.608,0.272,4.028c0.989,8.05,3.003,15.536,4.659,21.809c0.837,3.124,1.583,5.939,2.087,8.268c0.121,0.552,0.224,1.068,0.316,1.559c-1.347-0.558-3.022-1.305-4.902-2.251c-4.883-2.432-11.198-6.102-17.823-10.858c-9.936-7.122-20.553-16.713-28.482-28.118c-3.968-5.709-7.274-11.86-9.572-18.399c-2.306-6.54-3.622-13.474-3.622-20.881c0-2.96,0.206-5.993,0.655-9.124c0.437-3.015-0.782-6.066-3.19-7.953c-2.402-1.874-5.654-2.33-8.481-1.183c-7.814,3.191-15.657,7.856-23.555,13.571c-13.807,9.997-27.79,23.264-41.16,37.866c-20.05,21.905-38.667,46.82-52.432,68.24c-6.879,10.72-12.539,20.541-16.579,28.827c-2.026,4.15-3.646,7.905-4.829,11.32c-0.589,1.711-1.068,3.343-1.42,4.95C0.243,270.089,0,271.673,0,273.413c0,1.195,0.109,2.481,0.522,3.931c0.212,0.722,0.497,1.487,0.91,2.263c0.412,0.77,0.958,1.559,1.626,2.281v-0.006c1.474,1.595,3.228,2.742,4.926,3.524c1.59,0.728,3.154,1.177,4.659,1.487c2.839,0.564,5.544,0.698,8.329,0.782c2.432,0.061,4.92,0.061,7.48,0.109c4.477,0.092,9.173,0.34,13.959,1.25c3.597,0.685,7.237,1.722,10.919,3.348c5.52,2.439,11.168,6.158,16.894,12.254c5.703,6.091,11.478,14.608,16.877,26.577c1.134,2.529,3.415,4.338,6.133,4.878c2.718,0.534,5.514-0.274,7.523-2.184c4.434-4.192,8.899-7.31,13.164-9.33c4.277-2.026,8.305-2.967,12.09-2.967c3.324,0.012,6.503,0.704,9.767,2.257c2.444,1.177,4.932,2.851,7.443,5.174c3.767,3.47,7.571,8.408,11.138,15.057c3.567,6.643,6.885,14.984,9.736,25.127c0.746,2.663,2.754,4.816,5.368,5.738c2.615,0.928,5.532,0.522,7.796-1.08c6.381-4.526,12.036-7.54,16.962-9.397c4.938-1.862,9.123-2.578,12.739-2.578c2.712,0,5.12,0.401,7.389,1.098c1.983,0.618,3.858,1.48,5.678,2.572c3.166,1.905,6.163,4.55,8.954,7.692c4.18,4.714,7.85,10.543,11.053,16.209c1.602,2.839,3.1,5.63,4.537,8.256c1.444,2.627,2.821,5.084,4.295,7.328c1.583,2.408,4.252,3.852,7.14,3.852c2.881,0,5.55-1.444,7.14-3.852c1.28-1.959,2.481-4.064,3.731-6.315c2.172-3.931,4.464-8.299,7.048-12.636c1.935-3.246,4.022-6.46,6.279-9.409c3.385-4.428,7.146-8.208,11.198-10.768c2.026-1.28,4.125-2.275,6.357-2.961c2.232-0.686,4.617-1.062,7.28-1.068c3.616,0.006,7.802,0.722,12.74,2.578c4.926,1.856,10.58,4.871,16.962,9.397c2.269,1.602,5.18,2.008,7.795,1.08c2.614-0.922,4.623-3.075,5.369-5.738c2.536-9.015,5.436-16.61,8.56-22.846c2.336-4.683,4.799-8.608,7.298-11.823c3.749-4.828,7.559-8.044,11.235-10.058c3.695-2.002,7.256-2.876,10.992-2.888c3.791,0,7.819,0.94,12.096,2.967c4.271,2.026,8.736,5.138,13.164,9.336c2.014,1.904,4.81,2.712,7.528,2.178c2.711-0.54,4.992-2.348,6.133-4.878c3.567-7.904,7.292-14.31,11.059-19.498c3.294-4.537,6.618-8.147,9.894-11.047c5.764-5.071,11.405-8.02,16.938-9.876c8.298-2.779,16.446-2.979,23.914-3.051c1.868-0.024,3.694-0.043,5.502-0.103c2.712-0.116,5.375-0.304,8.184-0.983c1.408-0.352,2.864-0.844,4.332-1.601c1.468-0.752,2.942-1.802,4.21-3.173c0.674-0.722,1.213-1.504,1.626-2.275c0.62-1.171,0.959-2.293,1.165-3.325c0.206-1.031,0.267-1.971,0.267-2.869C512,272.085,511.855,270.853,511.636,269.628z" />
              </svg>
            </div>

            <!-- Morcego 3: Ágil na margem direita ao lado do card (apenas em telas grandes para manter mobile limpo) -->
            <div class="bat-day bat-day--3 hidden lg:block">
              <svg class="bat-svg-inner" viewBox="-4.1 123.7 150.1 84.6">
                <path fill="#2e1065" d="M140.965,127.705c-8.191,0.52-15.716,3.676-22.363,9.38c-7.586,6.509-13.035,15.37-17.421,22.502c-1.482,2.408-2.88,4.684-4.209,6.622c-0.597,0.873-1.21,1.581-1.823,2.106c-1.44,1.235-2.829,1.457-4.247,0.678c-3.801-2.085-8.243-9.729-6.592-16.016c0.118-0.449-0.156-0.904-0.626-1.035c-0.458-0.127-0.973,0.107-1.148,0.541c-0.587,1.458-1.583,2.816-2.882,3.932c-2.267,1.943-5.234,2.982-7.935,2.775c-2.196-0.169-4.08-1.147-5.446-2.833c-0.304-0.377-0.886-0.448-1.293-0.162c-0.417,0.293-0.503,0.833-0.21,1.217c2.321,3.022,7.245,14.775,3.906,20.564c-0.771,1.335-2.352,1.265-2.352,1.265c-2.477-0.049-7.684-3.048-8.27-3.39c-5.653-3.299-13.394-7.816-22.997-10.288c-11.338-2.919-22.929-2.279-34.449,1.899c-0.448,0.162-0.696,0.628-0.578,1.05c0.122,0.436,0.572,0.704,1.047,0.626c3.184-0.523,9.827-0.319,14.531,3.763c3.713,3.225,5.26,8.125,4.597,14.566c-0.036,0.354,0.157,0.688,0.492,0.85c0.336,0.163,0.74,0.117,1.032-0.116c5.75-4.587,9.696-3.876,11.86-2.282c6.717,4.946,3.884,9.802,3.884,9.802c-0.24,0.417-0.169,0.809,0.145,1.088c0.322,0.287,0.822,0.315,1.186,0.066c5.551-3.785,20.301-4.885,31.567-2.355c7.746,1.741,13.23,5.026,15.443,9.253c0.191,0.367,0.623,0.553,1.049,0.452c0.417-0.097,0.721-0.47,0.718-0.866c-0.027-4.479,3.218-9.744,9.14-14.825c4.953-4.25,11.396-8.061,17.675-10.457c6.127-2.338,11.341-2.973,13.946-1.698c0.396,0.193,0.9,0.09,1.18-0.255c0.278-0.346,0.255-0.819-0.059-1.13c-1.964-1.942-2.967-4.718-2.824-7.81c0.157-3.368,1.681-6.604,4.077-8.662c1.577-1.353,3.428-2.137,5.497-2.33c0.354-0.033,0.674-0.266,0.797-0.583c0.125-0.32,0.039-0.68-0.218-0.916c-2.42-2.227-3.693-5.612-3.587-9.536c0.151-5.543,2.944-11.303,7.288-15.029c0.353-0.304,0.72-0.596,1.09-0.868c0.32-0.236,0.444-0.646,0.305-1C141.745,127.901,141.375,127.678,140.965,127.705z" />
              </svg>
            </div>

            <!-- Morcego 4: Distante no horizonte alto central (apenas em telas grandes) -->
            <div class="bat-day bat-day--4 hidden lg:block">
              <svg class="bat-svg-inner" viewBox="153.8 150.9 83.3 78.4">
                <path fill="#3b0764" d="M220.366,205.274c-6.166-1.559-12.771-1.144-18.087-0.811c-1.795,0.111-3.491,0.219-4.986,0.239c-0.673,0.01-1.266-0.05-1.764-0.177c-1.17-0.295-1.794-0.937-1.909-1.959c-0.309-2.741,2.038-7.853,5.912-9.298c0.277-0.104,0.413-0.413,0.309-0.705c-0.102-0.285-0.413-0.467-0.702-0.397c-0.973,0.229-2.044,0.209-3.101-0.059c-1.842-0.466-3.47-1.628-4.352-3.107c-0.718-1.204-0.899-2.543-0.523-3.871c0.084-0.296-0.092-0.625-0.39-0.732c-0.307-0.108-0.618,0.044-0.71,0.336c-0.722,2.315-5.027,9.185-9.263,9.57c-0.977,0.089-1.52-0.76-1.52-0.76c-0.882-1.307-1.232-5.113-1.27-5.542c-0.355-4.149-0.845-9.83-3.08-15.729c-2.638-6.966-7.22-12.758-13.614-17.218c-0.249-0.173-0.582-0.131-0.758,0.084c-0.183,0.223-0.156,0.556,0.058,0.774c1.438,1.464,3.769,4.992,3.37,8.935c-0.315,3.113-2.296,5.714-5.89,7.729c-0.197,0.112-0.301,0.335-0.261,0.568c0.039,0.234,0.21,0.427,0.438,0.494c4.493,1.309,5.569,3.621,5.534,5.33c-0.11,5.307-3.674,5.613-3.674,5.613c-0.305,0.027-0.481,0.208-0.513,0.475c-0.031,0.272,0.137,0.542,0.4,0.641c4.002,1.499,9.98,8.767,12.796,15.552c1.934,4.666,2.234,8.722,0.849,11.422c-0.121,0.233-0.06,0.526,0.149,0.711c0.203,0.181,0.509,0.203,0.714,0.056c2.319-1.656,6.246-1.897,11.06-0.682c4.025,1.018,8.368,2.971,11.916,5.356c3.462,2.329,5.703,4.809,5.996,6.63c0.044,0.277,0.283,0.502,0.564,0.521s0.52-0.167,0.565-0.444c0.29-1.733,1.365-3.271,3.026-4.332c1.808-1.152,4.05-1.547,5.998-1.055c1.282,0.324,2.368,0.999,3.226,2.004c0.147,0.172,0.387,0.253,0.597,0.201c0.213-0.053,0.367-0.229,0.396-0.449c0.271-2.075,1.564-3.978,3.645-5.359c2.938-1.954,6.957-2.612,10.486-1.721c0.287,0.072,0.573,0.156,0.851,0.249c0.239,0.08,0.5-0.006,0.632-0.208c0.135-0.206,0.115-0.479-0.048-0.683C230.169,209.396,225.771,206.64,220.366,205.274z" />
              </svg>
            </div>
          </div>
        </template>

        <!-- Teia de aranha autêntica no canto superior direito e aranha balançando -->
        <div v-if="isHalloween" class="spider-corner pointer-events-none absolute top-0 right-0 z-20 hidden sm:block">
          <svg class="spiderweb-svg" viewBox="0 0 180 180" fill="none">
            <!-- Raios estruturais da teia partindo do vértice superior direito (180, 0) -->
            <g stroke="#581c87" stroke-width="1.1" opacity="0.38" stroke-linecap="round">
              <line x1="180" y1="0" x2="180.0" y2="180.0" />
              <line x1="180" y1="0" x2="133.4" y2="173.9" />
              <line x1="180" y1="0" x2="90.0" y2="155.9" />
              <line x1="180" y1="0" x2="52.7" y2="127.3" />
              <line x1="180" y1="0" x2="24.1" y2="90.0" />
              <line x1="180" y1="0" x2="6.1" y2="46.6" />
              <line x1="180" y1="0" x2="0.0" y2="0.0" />
            </g>
            <!-- Fios espirais concêntricos com curvatura côncava natural -->
            <g stroke="#7c3aed" stroke-width="1.2" opacity="0.45" stroke-linecap="round" stroke-linejoin="round">
              <path d="M 180.0 25.0 Q 177.1 21.8 173.5 24.1 Q 171.6 20.3 167.5 21.7 Q 166.6 17.5 162.3 17.7 Q 162.5 13.4 158.3 12.5 Q 159.7 8.4 155.9 6.5 Q 158.2 2.9 155.0 0.0" fill="none" />
              <path d="M 180.0 55.0 Q 173.7 48.0 165.8 53.1 Q 161.5 44.7 152.5 47.6 Q 150.5 38.4 141.1 38.9 Q 141.6 29.5 132.4 27.5 Q 135.3 18.5 126.9 14.2 Q 132.0 6.3 125.0 0.0" fill="none" />
              <path d="M 180.0 90.0 Q 169.7 78.5 156.7 86.9 Q 149.7 73.2 135.0 77.9 Q 131.8 62.8 116.4 63.6 Q 117.2 48.2 102.1 45.0 Q 106.8 30.3 93.1 23.3 Q 101.5 10.3 90.0 0.0" fill="none" stroke-width="1.3" />
              <path d="M 180.0 130.0 Q 165.1 113.4 146.4 125.6 Q 136.2 105.7 115.0 112.6 Q 110.4 90.8 88.1 91.9 Q 89.2 69.6 67.4 65.0 Q 74.3 43.8 54.4 33.6 Q 66.6 14.9 50.0 0.0" fill="none" />
              <path d="M 180.0 175.0 Q 159.9 152.7 134.7 169.0 Q 121.1 142.3 92.5 151.6 Q 86.3 122.2 56.3 123.7 Q 57.8 93.7 28.4 87.5 Q 37.7 58.9 11.0 45.3 Q 27.3 20.1 5.0 0.0" fill="none" />
            </g>
            <!-- Gotinhas sutis de orvalho violeta -->
            <g fill="#c084fc" opacity="0.6">
              <circle cx="156.7" cy="86.9" r="1.4" />
              <circle cx="115.0" cy="112.6" r="1.4" />
              <circle cx="67.4" cy="65.0" r="1.4" />
              <circle cx="141.1" cy="38.9" r="1.2" />
            </g>
          </svg>

          <!-- Aranha pendurada no fio de seda com 8 pernas articuladas -->
          <div class="spider-dangle">
            <div class="spider-thread"></div>
            <svg class="spider-body" viewBox="0 0 24 24" fill="none">
              <!-- Pernas Esquerdas articuladas -->
              <path d="M 10 9 C 6 6, 3 5, 2 8 C 1 10, 2 12, 1 14" stroke="#2e1065" stroke-width="1.3" stroke-linecap="round" fill="none" />
              <path d="M 9 11 C 5 9, 2 9, 1 12 C 0 14, 1 16, 1 17" stroke="#2e1065" stroke-width="1.3" stroke-linecap="round" fill="none" />
              <path d="M 9 13 C 5 14, 2 16, 2 18 C 2 20, 3 22, 3 23" stroke="#2e1065" stroke-width="1.3" stroke-linecap="round" fill="none" />
              <path d="M 10 15 C 6 18, 4 20, 5 23" stroke="#2e1065" stroke-width="1.3" stroke-linecap="round" fill="none" />

              <!-- Pernas Direitas articuladas -->
              <path d="M 14 9 C 18 6, 21 5, 22 8 C 23 10, 22 12, 23 14" stroke="#2e1065" stroke-width="1.3" stroke-linecap="round" fill="none" />
              <path d="M 15 11 C 19 9, 22 9, 23 12 C 24 14, 23 16, 23 17" stroke="#2e1065" stroke-width="1.3" stroke-linecap="round" fill="none" />
              <path d="M 15 13 C 19 14, 22 16, 22 18 C 22 20, 21 22, 21 23" stroke="#2e1065" stroke-width="1.3" stroke-linecap="round" fill="none" />
              <path d="M 14 15 C 18 18, 20 20, 19 23" stroke="#2e1065" stroke-width="1.3" stroke-linecap="round" fill="none" />

              <!-- Abdômen (Corpo) -->
              <ellipse cx="12" cy="15" rx="4.5" ry="5.5" fill="#1e1b4b" stroke="#3b0764" stroke-width="1" />
              <ellipse cx="12" cy="14" rx="1.5" ry="2.5" fill="#f97316" opacity="0.9" />

              <!-- Cefalotórax (Cabeça) -->
              <circle cx="12" cy="8.5" r="3" fill="#0f172a" />
              <!-- Olhos Vermelhos de Aranha -->
              <circle cx="11" cy="8" r="0.8" fill="#ef4444" />
              <circle cx="13" cy="8" r="0.8" fill="#ef4444" />
              <!-- Presas / Quelíceras -->
              <path d="M 11 11 L 11 12.5 M 13 11 L 13 12.5" stroke="#f8fafc" stroke-width="0.8" stroke-linecap="round" />
            </svg>
          </div>
        </div>

        <div class="relative z-20 mx-auto max-w-6xl px-6">
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
                  <span>Garantir minha vaga<template v-if="activeProduct"> · {{ isFree ? "Gratuito" : formattedPrice }}</template></span>
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
                <span v-if="isFree" class="inline-flex items-center gap-1.5 rounded-full bg-green-50 border border-green-300 px-3 py-1 text-green-900 shadow-2xs">
                  <span class="text-sm">🎟️</span>
                  Evento gratuito · R$ 0,00
                </span>
                <span class="inline-flex items-center gap-1.5 rounded-full bg-white/80 border border-slate-200 px-3 py-1 shadow-2xs">
                  <span class="text-orange-600 text-sm">🎃</span>
                  Velejo à fantasia sob o sol
                </span>
                <span class="inline-flex items-center gap-1.5 rounded-full bg-white/80 border border-slate-200 px-3 py-1 shadow-2xs">
                  <span class="text-amber-500 text-sm">☀️</span>
                  Concentração 9h30 · Saída prevista 10h
                </span>
                <span class="inline-flex items-center gap-1.5 rounded-full bg-white/80 border border-slate-200 px-3 py-1 shadow-2xs">
                  <span class="text-purple-600 text-sm">🏄‍♂️</span>
                  Apoio da equipe no mar e em terra
                </span>
                <span v-if="spotsLabel" class="inline-flex items-center gap-1.5 rounded-full bg-orange-50 border border-orange-300/60 px-3 py-1 text-orange-900 shadow-2xs">
                  <span class="text-orange-600 text-sm">⚡</span>
                  {{ spotsLabel }}
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
                      {{ isFree ? "Inscrição gratuita" : "Ingresso único" }} com tudo incluso.<template v-if="spotsLabel"> {{ unlimitedSpots ? "Vagas ilimitadas." : "Vagas estritamente limitadas." }}</template>
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
                        <span v-if="p.priceCents === 0" class="block font-mono text-[9px] font-bold uppercase tracking-wider text-green-700">Gratuito</span>
                        <span class="font-sans text-base font-bold text-orange-600">{{ formatBRL(p.priceCents) }}</span>
                      </div>
                    </div>
                  </div>

                  <!-- Route details footer inside card -->
                  <div class="mt-5 flex items-center justify-between border-t border-slate-200 pt-3 text-xs text-slate-600">
                    <div>
                      <p class="font-mono text-[9px] font-bold uppercase tracking-wider text-orange-600">Saída prevista · 10h</p>
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
                <Compass :size="12" /> {{ nextDateLabel }}<template v-if="spotsLabel"> · {{ unlimitedSpots ? "vagas ilimitadas" : `apenas ${totalSpots} vagas` }}</template>
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
                <p v-if="item.detail" class="mt-2 text-xs leading-relaxed text-slate-600">{{ item.detail }}</p>
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

          <div class="mt-12 grid gap-8 md:grid-cols-2">
            <div class="flex flex-col gap-3 rounded-2xl bg-white/5 border border-white/10 p-6 backdrop-blur-xs">
              <div class="flex items-center gap-3">
                <div class="p-2.5 rounded-xl bg-orange-500/20 text-orange-400 border border-orange-400/30">
                  <LifeBuoy :size="24" />
                </div>
                <h3 class="font-serif text-lg font-bold text-white">Apoio aquático e terrestre</h3>
              </div>
              <p class="text-sm leading-relaxed text-purple-200/70">
                Translado da P5 para ponto de partida (Sababa) e acompanhamento da equipe P5 no mar.
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
            <span>🎃</span> {{ isFree ? "EVENTO GRATUITO · R$ 0,00" : "LOTE EXCLUSIVO HALLOWEEN" }}
          </div>
          <h2 class="font-serif text-3xl font-bold text-slate-900 md:text-5xl">
            {{ isFree ? "Evento gratuito," : "Um valor," }} <span class="text-orange-600">tudo incluso.</span>
          </h2>
          <p class="mt-3 text-sm text-slate-600 max-w-xl mx-auto">
            <template v-if="spotsLabel">{{ spotsLabel }}. </template>Sábado, 31 de outubro · Sababa → P5 Kite House. Velejo à fantasia com transporte, suporte aquático, hidratação e receptivo.
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
                <span v-if="p.priceCents === 0" class="rounded-full bg-green-100 border border-green-300 px-2.5 py-0.5 font-mono text-[10px] font-black tracking-wider text-green-800 uppercase">Gratuito</span>
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
            <template v-if="spotsLabel">{{ spotsLabel }} · </template>{{ isFree ? "Inscrição gratuita com confirmação instantânea." : "Pagamento seguro via Pix com confirmação instantânea." }}
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
/* ── Morcegos Autênticos em Voo (Silhuetas fiéis nas margens do céu, sem cobrir textos) ── */
.bat-day {
  position: absolute;
  pointer-events: none;
  z-index: 0;
  opacity: 0.82;
  filter: drop-shadow(0 2px 6px rgba(15, 23, 42, 0.35));
}

.bat-svg-inner {
  width: 100%;
  height: 100%;
  transform-origin: center center;
}

/* No mobile: tamanho reduzido (~36-40px) e fixado nos cantos extremos do céu (topo) */
.bat-day--1 {
  top: 8px;
  left: 10px;
  width: 36px;
  animation: flight-swoop-mobile-1 8s ease-in-out infinite;
}
.bat-day--1 .bat-svg-inner {
  animation: bat-flap-1 0.75s ease-in-out infinite alternate;
}

.bat-day--2 {
  top: 8px;
  right: 12px;
  width: 40px;
  animation: flight-swoop-mobile-2 9s ease-in-out infinite 1s;
}
.bat-day--2 .bat-svg-inner {
  animation: bat-flap-2 0.85s ease-in-out infinite alternate;
}

/* No desktop: morcegos ampliados e posicionados estrategicamente nas margens e no céu alto */
@media (min-width: 1024px) {
  .bat-day {
    opacity: 0.9;
    filter: drop-shadow(0 4px 10px rgba(15, 23, 42, 0.45));
  }

  /* Morcego 1: Canto superior esquerdo bem acima do bloco de texto */
  .bat-day--1 {
    top: 18px;
    left: 24px;
    width: 68px;
    animation: flight-swoop-1 11s ease-in-out infinite;
  }

  /* Morcego 2: Céu superior direito acima do card */
  .bat-day--2 {
    top: 26px;
    right: 20%;
    width: 74px;
    animation: flight-swoop-2 13s ease-in-out infinite 2s;
  }

  /* Morcego 3: Margem lateral direita externa ao card */
  .bat-day--3 {
    top: 36%;
    right: 20px;
    width: 58px;
    animation: flight-swoop-3 10s ease-in-out infinite 4s;
  }
  .bat-day--3 .bat-svg-inner {
    animation: bat-flap-3 0.7s ease-in-out infinite alternate;
  }

  /* Morcego 4: No topo central entre as duas colunas */
  .bat-day--4 {
    top: 14px;
    left: 45%;
    width: 38px;
    animation: flight-swoop-4 15s ease-in-out infinite 1.5s;
  }
  .bat-day--4 .bat-svg-inner {
    animation: bat-flap-4 0.6s ease-in-out infinite alternate;
  }
}

/* Animação realista de bater de asas com perspectiva 3D */
@keyframes bat-flap-1 {
  0% { transform: scaleY(1) scaleX(1) rotate(0deg); }
  100% { transform: scaleY(0.68) scaleX(0.96) rotate(-3deg); }
}

@keyframes bat-flap-2 {
  0% { transform: scaleY(1) scaleX(1) rotate(0deg); }
  100% { transform: scaleY(0.65) scaleX(0.97) rotate(2deg); }
}

@keyframes bat-flap-3 {
  0% { transform: scaleY(1) scaleX(1) rotate(0deg); }
  100% { transform: scaleY(0.70) scaleX(0.95) rotate(-4deg); }
}

@keyframes bat-flap-4 {
  0% { transform: scaleY(1) scaleX(1) rotate(0deg); }
  100% { transform: scaleY(0.72) scaleX(0.96) rotate(3deg); }
}

/* Trajetórias suaves e contidas para NUNCA invadir a área de texto */
@keyframes flight-swoop-mobile-1 {
  0%, 100% { transform: translate3d(0, 0, 0) rotate(-4deg); }
  50% { transform: translate3d(5px, 4px, 0) rotate(3deg); }
}

@keyframes flight-swoop-mobile-2 {
  0%, 100% { transform: translate3d(0, 0, 0) rotate(4deg); }
  50% { transform: translate3d(-5px, 4px, 0) rotate(-3deg); }
}

@keyframes flight-swoop-1 {
  0%, 100% { transform: translate3d(0, 0, 0) rotate(-5deg); }
  35% { transform: translate3d(14px, -8px, 0) rotate(4deg); }
  70% { transform: translate3d(6px, 10px, 0) rotate(-2deg); }
}

@keyframes flight-swoop-2 {
  0%, 100% { transform: translate3d(0, 0, 0) rotate(4deg); }
  45% { transform: translate3d(14px, 10px, 0) rotate(-4deg); }
  80% { transform: translate3d(-8px, -6px, 0) rotate(4deg); }
}

@keyframes flight-swoop-3 {
  0%, 100% { transform: translate3d(0, 0, 0) rotate(-4deg); }
  50% { transform: translate3d(-10px, 12px, 0) rotate(5deg); }
}

@keyframes flight-swoop-4 {
  0%, 100% { transform: translate3d(0, 0, 0) rotate(3deg); }
  50% { transform: translate3d(12px, -6px, 0) rotate(-4deg); }
}

/* ── Teia de aranha e aranha balançando no canto superior direito ── */
.spider-corner {
  top: 0;
  right: 0;
  width: 180px;
  height: 180px;
}

.spiderweb-svg {
  width: 100%;
  height: 100%;
}

.spider-dangle {
  position: absolute;
  top: 86px;
  right: 90px;
  display: flex;
  flex-direction: column;
  align-items: center;
  transform-origin: top center;
  animation: spider-sway 5s ease-in-out infinite;
}

.spider-thread {
  width: 1px;
  height: 46px;
  background: linear-gradient(to bottom, rgba(88, 28, 135, 0.4), rgba(109, 40, 217, 0.7));
  box-shadow: 0 0 2px rgba(147, 51, 234, 0.3);
}

.spider-body {
  width: 24px;
  height: 24px;
  margin-top: -1px;
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.35));
}

@keyframes spider-sway {
  0%, 100% { transform: rotate(9deg); }
  50% { transform: rotate(-9deg); }
}
</style>
