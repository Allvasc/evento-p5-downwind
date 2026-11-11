<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useRouter } from "vue-router";
import { useQuery } from "@tanstack/vue-query";
import {
  ArrowRight,
  Check,
  Compass,
  Ticket,
  Coffee,
  Gift,
  Bus,
  LifeBuoy,
  Wind,
  PartyPopper,
  MapPin,
  ExternalLink,
  Download,
  Maximize2,
  Eye,
  X,
  Sparkles,
} from "lucide-vue-next";
import { formatBRL, cleanTitle, isUnlimitedCapacity, type Product } from "@p5wellness/shared";
import { api } from "@/lib/api";
import WellnessHeader from "@/components/WellnessHeader.vue";
import HelpFab from "@/components/HelpFab.vue";
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

const formattedPrice = computed(() => (activeProduct.value ? formatBRL(activeProduct.value.priceCents) : ""));

// Gratuidade liga/desliga pelo preço do produto no Admin: R$ 0 = evento gratuito.
const isFree = computed(() => activeProduct.value?.priceCents === 0);

const productTitle = computed(() => cleanTitle(activeProduct.value?.title ?? "P5 DownWind Day"));

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
  if (!sessions.length) return "Sábado, 28 de novembro";
  return formatEventDate(sessions[0].startsAt);
});

// A capacidade vem da turma (40 vagas nesta edição); 40 é só o fallback.
const totalSpots = computed(() => nextSessions.value[0]?.capacity ?? 40);
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

const included = [
  { icon: Ticket, title: "Inscrição Sertões Kitesurf", detail: "Sua inscrição na Categoria Experience do Sertões Kitesurf 2026." },
  { icon: Coffee, title: "Café da manhã na P5", detail: "Estilo buffet a partir das 5h: café, sucos, frutas, tapiocas, cuscuz, pães e opções leves." },
  { icon: Gift, title: "Kit participante oficial", detail: "Lycra, camiseta, boné e sacochila do Sertões Kitesurf." },
  { icon: Bus, title: "Ônibus executivo", detail: "P5 → Icapuí na ida e Fortim → P5 na volta." },
  { icon: LifeBuoy, title: "Equipe de acompanhamento", detail: "Água, isotônicos, frutas e snacks ao longo do dia." },
  { icon: Wind, title: "Logística dos equipamentos", detail: "Transporte e espaço para o seu equipamento de kite." },
  { icon: PartyPopper, title: "Entretenimento", detail: "Playlist e confraternização no retorno à P5." },
];

const itinerary = [
  { time: "1", title: "05h00 · Abertura da P5", detail: "Recepção e café da manhã." },
  { time: "2", title: "05h30 · Check-in", detail: "Conferência de participantes e equipamentos." },
  { time: "3", title: "06h15 · Embarque", detail: "Embarque no ônibus." },
  { time: "4", title: "06h30 · Saída", detail: "Saída da P5 rumo a Ponta Grossa, Icapuí." },
  { time: "5", title: "09h30 (estimativa) · Icapuí", detail: "Chegada e desembarque na Praia de Ponta Grossa, em Icapuí." },
  { time: "6", title: "A confirmar · Largada oficial", detail: "Largada da Categoria Experience do Sertões Kitesurf: downwind de aproximadamente 40 km até Fortim." },
  { time: "7", title: "Após o percurso · Fortim", detail: "Chegada ao Jaguaribe Lodge, em Fortim: encontro, hidratação e organização do embarque." },
  { time: "8", title: "Final da tarde/noite · Retorno", detail: "Confraternização e retorno; chegada noturna à P5." },
];

// ── Material Visual / Artes do Evento ─────────────────────────────────────────
type ArtworkTab = "banner" | "poster" | "flyer";
const activeArtworkTab = ref<ArtworkTab>("banner");
const isLightboxOpen = ref(false);
const lightboxSrc = ref("/arte-evento-p5-sertoes.webp");
const lightboxTitle = ref("Banner Oficial · P5 DownWind Day × Sertões Kitesurf");

const artworks = [
  {
    id: "banner" as const,
    title: "Banner Panorâmico",
    badge: "Formato 16:9 · Alta Resolução",
    desc: "Banner oficial do evento com percurso Icapuí → Fortim, horário, benefícios e co-branding.",
    src: "/arte-evento-p5-sertoes.webp",
    downloadSrc: "/arte-evento-p5-sertoes.webp",
    fileName: "P5-DownWind-Day-Sertoes-Kitesurf-Banner.webp",
  },
  {
    id: "poster" as const,
    title: "Cartaz Vertical",
    badge: "Formato 4:5 · Feed & Stories",
    desc: "Arte vertical otimizada para divulgação nas redes sociais, feed e grupos de WhatsApp.",
    src: "/cartaz-p5-sertoes-experience.webp",
    downloadSrc: "/cartaz-p5-sertoes-experience.webp",
    fileName: "P5-DownWind-Day-Sertoes-Cartaz.webp",
  },
  {
    id: "flyer" as const,
    title: "Flyer Oficial Sertões",
    badge: "Material Oficial da Organização",
    desc: "Flyer oficial de divulgação da Categoria Experience do Sertões Kitesurf 2026.",
    src: "/cartaz-sertoes-experience.webp",
    downloadSrc: "/cartaz-sertoes-experience.webp",
    fileName: "Sertoes-Kitesurf-Experience-Flyer.webp",
  },
];

const currentArtwork = computed(() => {
  return artworks.find((a) => a.id === activeArtworkTab.value) ?? artworks[0];
});

function openArtwork(src: string, title: string) {
  lightboxSrc.value = src;
  lightboxTitle.value = title;
  isLightboxOpen.value = true;
}
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
            <p class="eyebrow mb-6 text-xs tracking-wider flex flex-wrap items-center gap-2">
              <Compass :size="14" class="text-magenta" />
              <span>P5 KITE HOUSE</span>
              <span class="text-ink-soft/40">·</span>
              <img src="/sertoes-kitesurf-verde.webp" alt="Sertões Kitesurf" class="h-4 w-auto rounded-xs inline-block shadow-2xs" />
              <span>SERTÕES KITESURF 2026</span>
            </p>

            <h1 class="font-serif text-5xl font-black leading-[1.05] text-ink md:text-6xl">
              Feito para quem leva o vento
              <span class="text-magenta">a sério.</span>
            </h1>

            <p class="mt-6 max-w-md text-base leading-relaxed text-ink-soft md:text-lg">
              <strong class="font-semibold text-ink">{{ nextDateLabel }}</strong> · Categoria Experience do Sertões Kitesurf 2026.
              Downwind de aproximadamente 40 km de Ponta Grossa, em Icapuí, até Fortim, com a inscrição oficial, café da manhã, ônibus executivo e equipe de acompanhamento da P5.
            </p>

            <div class="mt-8 flex flex-wrap items-center gap-5">
              <button class="button-magenta" @click="router.push('/comprar')">
                Garantir minha vaga<template v-if="activeProduct"> · {{ isFree ? "Gratuito" : formattedPrice }}</template>
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
              <span class="inline-flex items-center gap-1.5"><Check :size="15" class="text-magenta" /> Inscrição Sertões Kitesurf inclusa</span>
              <span class="inline-flex items-center gap-1.5"><Check :size="15" class="text-magenta" /> Abertura 5h · saída 6h30 da P5</span>
              <span class="inline-flex items-center gap-1.5"><Check :size="15" class="text-magenta" /> Ônibus executivo ida e volta</span>
              <span v-if="spotsLabel" class="inline-flex items-center gap-1.5"><Check :size="15" class="text-magenta" /> {{ spotsLabel }}</span>
            </div>
          </div>

          <!-- STACKED PRODUCT CARDS CONTAINER -->
          <div class="relative mx-auto w-full max-w-md">
            <div class="relative rounded-[2rem] border border-line/80 bg-white p-6 shadow-2xl md:p-8">
              <div class="text-center">
                <div class="inline-flex items-center gap-2.5">
                  <img src="/logo-downwind.webp" :alt="productTitle" class="h-7 w-auto" />
                  <span class="text-xs font-semibold text-ink-soft/40">×</span>
                  <img src="/sertoes-kitesurf-verde.webp" alt="Sertões Kitesurf" class="h-7 w-auto rounded-sm shadow-2xs" />
                  <span class="rounded-full border border-magenta/30 bg-magenta/10 px-2.5 py-0.5 font-mono text-[9px] font-bold tracking-wider text-magenta uppercase">Experience</span>
                </div>
                <p class="mt-2 text-xs font-medium text-ink-soft">
                  {{ isFree ? "Inscrição gratuita" : "Um valor" }}, tudo incluso.
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
                    <h3 class="font-serif text-sm font-bold text-ink">{{ cleanTitle(p.title) }}</h3>
                    <p class="mt-1 text-xs text-ink-soft">Categoria Experience · Icapuí → Fortim</p>
                  </div>

                  <div class="shrink-0 text-right">
                    <span v-if="p.priceCents === 0" class="block font-mono text-[9px] font-bold uppercase tracking-wider text-magenta">Gratuito</span>
                    <span class="font-sans text-base font-bold text-ink">{{ formatBRL(p.priceCents) }}</span>
                  </div>
                </div>
              </div>

              <!-- Route details footer inside card -->
              <div class="mt-6 flex items-center justify-between border-t border-line/60 pt-4 text-xs text-ink-soft">
                <div>
                  <p class="font-mono text-[9px] font-bold uppercase tracking-wider text-ink-soft">Largada · Ponta Grossa</p>
                  <p class="font-serif font-bold text-ink">Icapuí</p>
                </div>
                <div class="flex flex-col items-center">
                  <Wind :size="18" class="text-magenta shrink-0" />
                  <span class="font-mono text-[9px] text-ink-soft">~40 km</span>
                </div>
                <div class="text-right">
                  <p class="font-mono text-[9px] font-bold uppercase tracking-wider text-ink-soft">Chegada · Jaguaribe Lodge</p>
                  <p class="font-serif font-bold text-ink">Fortim</p>
                </div>
              </div>
            </div>

            <!-- Floating Top Badge -->
            <span class="absolute -top-3 -right-2 inline-flex items-center gap-1.5 rounded-full border border-magenta/20 bg-white px-3.5 py-1 text-[11px] font-semibold text-magenta shadow-sm">
              <Compass :size="12" /> {{ nextDateLabel }}<template v-if="spotsLabel"> · {{ unlimitedSpots ? "vagas ilimitadas" : `apenas ${totalSpots} vagas` }}</template>
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
              Da recepção com café da manhã às 5h na P5 Kite House à chegada noturna, passando pela inscrição no Sertões Kitesurf, pelo ônibus executivo e pela equipe de acompanhamento, tudo já está no seu ingresso.
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
          <p class="eyebrow mb-2"><WaveIcon :size="14" /> PONTA GROSSA, ICAPUÍ → FORTIM · ~40 KM</p>
          <h2 class="font-serif text-3xl font-bold text-ink md:text-5xl">
            Como funciona <span class="text-magenta">o seu dia.</span>
          </h2>

          <div class="mt-14 grid gap-x-8 gap-y-12 sm:grid-cols-2 md:grid-cols-4">
            <div v-for="(step, i) in itinerary" :key="step.title" class="relative">
              <div class="flex items-center gap-3">
                <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-ink font-mono text-sm font-bold text-white">
                  {{ step.time }}
                </span>
                <div v-if="i % 4 !== 3 && i < itinerary.length - 1" class="hidden h-px flex-1 bg-line md:block"></div>
              </div>
              <h3 class="mt-5 font-serif text-lg font-bold text-ink">{{ step.title }}</h3>
              <p class="mt-1.5 text-sm leading-relaxed text-ink-soft">{{ step.detail }}</p>
            </div>
          </div>

          <p class="mt-12 text-xs italic text-ink-soft">
            Os horários dependem da programação oficial do Sertões Kitesurf, do trânsito, do vento e do mar.
          </p>
        </div>
      </section>

      <!-- SECTION: SERTÕES KITESURF · CATEGORIA EXPERIENCE -->
      <section id="sertoes" class="border-t border-line/60 bg-ink py-20 text-white">
        <div class="mx-auto max-w-6xl px-6">
          <div class="mb-8 flex flex-wrap items-center gap-4">
            <img src="/sertoes-kitesurf-verde.webp" alt="Sertões Kitesurf" class="h-16 w-auto rounded-xl shadow-lg ring-1 ring-white/10" />
            <img src="/sertoes-experience-white.webp" alt="Sertões Experience" class="h-10 w-auto opacity-95" />
          </div>
          <p class="font-mono text-xs font-medium tracking-widest uppercase text-white/60">SERTÕES KITESURF 2026 · CATEGORIA EXPERIENCE</p>
          <h2 class="mt-2 font-serif text-3xl font-bold leading-tight md:text-4xl">
            Sua largada no <span class="text-sky-300">Sertões Kitesurf.</span>
          </h2>
          <p class="mt-4 max-w-2xl text-sm leading-relaxed text-white/70">
            O pacote P5 inclui a sua inscrição na Categoria Experience do Sertões Kitesurf 2026: a etapa de 28 de novembro, com largada na Praia de Ponta Grossa, em Icapuí, e chegada ao Jaguaribe Lodge, em Fortim.
          </p>

          <div class="mt-12 grid gap-8 md:grid-cols-3">
            <div class="flex flex-col gap-3">
              <MapPin :size="26" class="text-sky-300" />
              <h3 class="font-serif text-lg font-bold">Percurso com guias exclusivos</h3>
              <p class="text-sm leading-relaxed text-white/70">
                Cerca de 40 km de downwind de Icapuí a Fortim, acompanhados por guias exclusivos.
              </p>
            </div>
            <div class="flex flex-col gap-3">
              <Gift :size="26" class="text-sky-300" />
              <h3 class="font-serif text-lg font-bold">Kit participante oficial</h3>
              <p class="text-sm leading-relaxed text-white/70">
                Lycra, camiseta, boné e sacochila do Sertões Kitesurf.
              </p>
            </div>
            <div class="flex flex-col gap-3">
              <PartyPopper :size="26" class="text-sky-300" />
              <h3 class="font-serif text-lg font-bold">Vilas e festa de encerramento</h3>
              <p class="text-sm leading-relaxed text-white/70">
                Livre acesso às vilas do evento e à festa de encerramento.
              </p>
            </div>
          </div>

          <div class="mt-12 flex flex-col gap-3 border-t border-white/10 pt-6 sm:flex-row sm:items-center sm:justify-between">
            <p class="max-w-xl text-xs leading-relaxed text-white/50">
              Inscrição na Categoria Experience sujeita à disponibilidade de vagas e às regras da organização do Sertões Kitesurf.
            </p>
            <a
              href="https://sertoes.com.br/kite/"
              target="_blank"
              rel="noopener noreferrer"
              class="inline-flex shrink-0 items-center gap-1.5 text-sm font-semibold text-sky-300 hover:text-white"
            >
              Conheça o Sertões Kitesurf <ExternalLink :size="14" />
            </a>
          </div>
        </div>
      </section>

      <!-- SECTION: ARTE OFICIAL DO EVENTO -->
      <section id="arte-evento" class="border-t border-line/60 bg-paper py-20">
        <div class="mx-auto max-w-6xl px-6">
          <div class="flex flex-col md:flex-row md:items-end justify-between gap-6 mb-10">
            <div>
              <p class="eyebrow mb-2">
                <Sparkles :size="14" class="text-magenta" /> MATERIAL VISUAL OFICIAL
              </p>
              <h2 class="font-serif text-3xl font-bold text-ink md:text-5xl">
                Arte oficial <span class="text-magenta">do evento.</span>
              </h2>
              <p class="mt-3 max-w-2xl text-sm leading-relaxed text-ink-soft">
                Material visual oficial do P5 DownWind Day integrado à Categoria Experience do Sertões Kitesurf 2026. Clique para ampliar ou baixe em alta resolução.
              </p>
            </div>

            <!-- Tabs de visualização -->
            <div class="flex items-center gap-1 p-1 bg-warm rounded-xl border border-line/70 self-start md:self-auto">
              <button
                v-for="art in artworks"
                :key="art.id"
                :class="[
                  'px-3.5 py-1.5 text-xs font-semibold rounded-lg transition-all',
                  activeArtworkTab === art.id
                    ? 'bg-white text-ink shadow-xs'
                    : 'text-ink-soft hover:text-ink',
                ]"
                @click="activeArtworkTab = art.id"
              >
                {{ art.title }}
              </button>
            </div>
          </div>

          <!-- Card de Apresentação da Arte -->
          <div class="overflow-hidden rounded-3xl border border-line/80 bg-white shadow-xl">
            <div
              class="relative group cursor-pointer overflow-hidden bg-ink/5"
              @click="openArtwork(currentArtwork.src, currentArtwork.title)"
            >
              <img
                :src="currentArtwork.src"
                :alt="currentArtwork.title"
                class="w-full object-cover transition-transform duration-500 group-hover:scale-[1.01]"
                :class="currentArtwork.id === 'banner' ? 'max-h-[620px]' : 'max-h-[720px] mx-auto object-contain bg-warm/30'"
              />

              <!-- Overlay Hover -->
              <div class="absolute inset-0 bg-ink/30 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-3 backdrop-blur-xs">
                <span class="inline-flex items-center gap-2 rounded-full bg-white px-5 py-2.5 text-xs font-bold text-ink shadow-lg">
                  <Maximize2 :size="15" /> Clique para ampliar em tela cheia
                </span>
              </div>
            </div>

            <!-- Rodapé do Card da Arte -->
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-6 border-t border-line/60 bg-white">
              <div>
                <div class="flex items-center gap-2.5">
                  <h3 class="font-serif text-lg font-bold text-ink">{{ currentArtwork.title }}</h3>
                  <span class="rounded-full bg-magenta/10 px-2.5 py-0.5 font-mono text-[10px] font-bold text-magenta">
                    {{ currentArtwork.badge }}
                  </span>
                </div>
                <p class="mt-1 text-xs text-ink-soft">{{ currentArtwork.desc }}</p>
              </div>

              <div class="flex items-center gap-3 shrink-0">
                <button
                  class="inline-flex items-center gap-1.5 rounded-full border border-line bg-warm/50 px-4 py-2 text-xs font-semibold text-ink transition-colors hover:bg-white hover:border-ink"
                  @click="openArtwork(currentArtwork.src, currentArtwork.title)"
                >
                  <Eye :size="14" /> Visualizar
                </button>
                <a
                  :href="currentArtwork.downloadSrc"
                  :download="currentArtwork.fileName"
                  class="button-magenta px-4 py-2 text-xs font-bold"
                >
                  <Download :size="14" /> Baixar em Alta Resolução
                </a>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- SECTION: PREÇO / CTA -->
      <section id="preco" class="border-t border-line/60 py-20">
        <div class="mx-auto max-w-5xl px-6 text-center">
          <p class="eyebrow mb-2 justify-center">{{ isFree ? "EVENTO GRATUITO · R$ 0,00" : "INGRESSO" }}</p>
          <h2 class="font-serif text-3xl font-bold text-ink md:text-5xl">
            {{ isFree ? "Evento gratuito," : "Um valor," }} <span class="text-magenta">tudo incluso.</span>
          </h2>
          <p class="mt-3 text-sm text-ink-soft">
            <template v-if="spotsLabel">{{ spotsLabel }}. </template>{{ nextDateLabel }} · Categoria Experience do Sertões Kitesurf · Icapuí → Fortim.
            {{ isFree ? "Inscrição gratuita com confirmação instantânea." : "Pagamento seguro via Pix." }}
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
                <span v-if="p.priceCents === 0" class="rounded-full bg-magenta/10 px-2.5 py-0.5 font-mono text-[9px] font-bold tracking-wider text-magenta uppercase">Gratuito</span>
              </div>

              <div class="flex w-full sm:w-auto items-center justify-between sm:justify-end gap-5">
                <span class="font-serif text-2xl font-black text-ink">{{ formatBRL(p.priceCents) }}</span>
                <button class="button-magenta px-5 py-2.5 text-xs font-bold shrink-0" @click="router.push('/comprar')">
                  Garantir <ArrowRight :size="14" />
                </button>
              </div>
            </div>
          </div>

          <p class="mt-8 text-xs italic font-medium text-ink-soft">
            <template v-if="spotsLabel">{{ spotsLabel }} · </template>{{ isFree ? "Inscrição gratuita." : "Pagamento via Pix." }}
          </p>
        </div>
      </section>

      <!-- FOOTER -->
      <footer class="border-t border-line/80 bg-ink py-12 text-white">
        <div class="mx-auto flex max-w-6xl flex-col items-center justify-between gap-6 px-6 md:flex-row">
          <div class="flex items-center gap-3">
            <img src="/logo-downwind-dark.webp" alt="P5 DownWind Day" class="h-8 w-auto" />
            <span class="text-white/30 text-sm font-light">×</span>
            <img src="/sertoes-kitesurf-white.webp" alt="Sertões Kitesurf" class="h-8 w-auto opacity-90 transition-opacity hover:opacity-100" />
          </div>

          <p class="text-xs text-white/60 text-center md:text-left">
            P5 Kite House & Sertões Kitesurf.<br />
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

      <!-- LIGHTBOX MODAL DE ARTE DO EVENTO -->
      <Teleport to="body">
        <Transition
          enter-active-class="transition duration-200 ease-out"
          enter-from-class="opacity-0"
          enter-to-class="opacity-100"
          leave-active-class="transition duration-150 ease-in"
          leave-from-class="opacity-100"
          leave-to-class="opacity-0"
        >
          <div
            v-if="isLightboxOpen"
            class="fixed inset-0 z-50 flex items-center justify-center bg-black/90 p-4 sm:p-6 backdrop-blur-md"
            @click.self="isLightboxOpen = false"
          >
            <div class="relative max-h-[92vh] max-w-5xl w-full flex flex-col items-center">
              <button
                class="absolute -top-12 right-0 flex items-center gap-1.5 rounded-full bg-white/10 px-3.5 py-1.5 text-xs font-semibold text-white hover:bg-white/20 transition-colors"
                @click="isLightboxOpen = false"
                aria-label="Fechar modal"
              >
                <X :size="16" /> Fechar
              </button>

              <img
                :src="lightboxSrc"
                :alt="lightboxTitle"
                class="max-h-[82vh] w-auto max-w-full rounded-2xl object-contain shadow-2xl"
              />

              <div class="mt-4 flex flex-wrap items-center justify-between gap-3 w-full text-white/80 text-xs px-2">
                <span class="font-medium">{{ lightboxTitle }}</span>
                <a
                  :href="lightboxSrc"
                  download
                  class="inline-flex items-center gap-1.5 rounded-full bg-magenta px-4 py-1.5 text-xs font-bold text-white hover:bg-magenta/90 shadow-sm"
                >
                  <Download :size="14" /> Baixar Imagem HD
                </a>
              </div>
            </div>
          </div>
        </Transition>
      </Teleport>
    </main>
  </div>
</template>
