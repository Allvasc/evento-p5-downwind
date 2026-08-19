<script setup lang="ts">
import { computed } from "vue";
import { useRouter } from "vue-router";
import { useQuery } from "@tanstack/vue-query";
import { Sun, Check, ArrowRight, Heart, Waves, Sparkles, Coffee, Clock } from "lucide-vue-next";
import { formatBRL, type Product } from "@p5wellness/shared";
import { api } from "@/lib/api";
import WellnessHeader from "@/components/WellnessHeader.vue";
import HelpFab from "@/components/HelpFab.vue";

const router = useRouter();

const detailFor = (p: Product) => {
  if (p.type === "combo") return "Benefícios independentes para cada momento.";
  return "Escolha a sua turma e garanta sua vaga.";
};

const { data } = useQuery({
  queryKey: ["public-products"],
  queryFn: () => api.get<{ products: Product[] }>("/public/products"),
  staleTime: 60_000,
});

const products = computed(() => data.value?.products ?? []);

function scrollToExperiencia() {
  document.querySelector("#experiencia")?.scrollIntoView({ behavior: "smooth" });
}

// ── Próxima data ────────────────────────────────────────────────────────────
// Visitors used to only find out when the next class actually happens after picking a
// product at /comprar — pull the real turma dates straight from the admin's cadastro so
// the landing page shows them up front, before checkout.
interface NextSession {
  activityTitle: string;
  startsAt: string;
}

const { data: nextSessionsData } = useQuery({
  queryKey: ["public-next-sessions"],
  queryFn: () => api.get<{ sessions: NextSession[] }>("/public/next-sessions"),
  staleTime: 60_000,
});

const nextSessions = computed(() => nextSessionsData.value?.sessions ?? []);

function formatEventDate(iso: string) {
  const label = new Date(iso).toLocaleDateString("pt-BR", { weekday: "long", day: "2-digit", month: "long" });
  // Capitalize only the leading letter — pt-BR's "de" shouldn't be title-cased.
  return label.charAt(0).toUpperCase() + label.slice(1);
}

// The business runs as a single-day event (Yoga + HYROX + Café da Manhã together), so
// the common case is every activity landing on the same date — show one line for that,
// and only fall back to a per-activity list if they've actually diverged.
const sameDayLabel = computed(() => {
  const sessions = nextSessions.value;
  if (!sessions.length) return null;
  const days = new Set(sessions.map((s: NextSession) => s.startsAt.slice(0, 10)));
  return days.size === 1 ? formatEventDate(sessions[0].startsAt) : null;
});
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
              <Sun :size="14" class="text-magenta" />
              P5 KITE HOUSE × AYO WELLNESS
            </p>

            <h1 class="font-serif text-5xl font-bold leading-[1.08] text-ink md:text-6xl">
              Respire fundo.<br />
              <em class="font-serif italic text-magenta">Movimente-se livre.</em>
            </h1>

            <p class="mt-6 max-w-md text-base leading-relaxed text-ink-soft md:text-lg">
              Uma manhã feita de movimento, energia e conexão. Escolha sua experiência no P5 Wellness Club.
            </p>

            <div class="mt-8 flex flex-wrap items-center gap-5">
              <button class="button-magenta" @click="router.push('/comprar')">
                Escolher experiência
                <ArrowRight :size="18" />
              </button>
              <button
                class="text-sm font-semibold text-ink underline underline-offset-4 transition-colors hover:text-magenta"
                @click="scrollToExperiencia"
              >
                Conhecer o clube
              </button>
            </div>

            <div class="mt-8 flex flex-wrap gap-x-6 gap-y-2 text-xs font-medium text-ink-soft">
              <span class="inline-flex items-center gap-1.5"><Check :size="15" class="text-magenta" /> Vagas limitadas por turma</span>
              <span class="inline-flex items-center gap-1.5"><Check :size="15" class="text-magenta" /> Check-in digital</span>
            </div>
          </div>

          <!-- POSTER CARD GRAPHIC -->
          <div class="relative mx-auto w-full max-w-sm">
            <div class="relative rounded-t-[7rem] rounded-b-[2rem] border border-line/80 bg-[#fbf6ee] p-6 pt-10 shadow-2xl backdrop-blur">
              <div class="text-center">
                <span class="font-sans text-xl font-black tracking-tighter text-ink">P5</span>
                <p class="font-serif text-2xl italic font-normal tracking-tight text-ink -mt-1">wellness</p>
                <p class="font-sans text-[10px] font-bold tracking-[0.3em] uppercase text-ink">C L U B</p>

                <p class="mt-5 text-xs italic leading-relaxed text-ink-soft px-2">
                  Tudo pensado para proporcionar uma experiência completa, unindo movimento, energia, bem-estar e momentos de conexão.
                </p>
                <p class="mt-2 text-[11px] text-ink-soft">Escolha como você quer viver essa experiência.</p>
              </div>

              <!-- Poster Pricing Boxes -->
              <div class="mt-6 space-y-2.5 text-left">
                <!-- Box 1 -->
                <div class="rounded-lg border-2 border-magenta bg-white/80 p-2.5">
                  <p class="font-serif italic text-xs font-bold text-ink">Aulas</p>
                  <p class="text-[11px] text-ink-soft">Yoga + HYROX</p>
                  <p class="font-sans text-sm font-bold text-ink">R$ 40,00</p>
                </div>

                <!-- Box 2 (Featured Gold) -->
                <div class="rounded-lg border-2 border-[#b8860b] bg-[#faf3e6] p-2.5">
                  <p class="font-serif italic text-xs font-bold text-ink">Aulas + Café</p>
                  <p class="text-[11px] text-ink-soft">Yoga + HYROX + Café da Manhã</p>
                  <p class="font-sans text-sm font-bold text-ink">R$ 70,00</p>
                </div>

                <!-- Box 3 -->
                <div class="rounded-lg border border-ink/40 bg-white/50 p-2.5">
                  <p class="font-serif italic text-xs font-bold text-ink">Aulas individuais</p>
                  <p class="text-[11px] text-ink-soft">Yoga ou HYROX</p>
                  <p class="font-sans text-sm font-bold text-ink">R$ 25,00</p>
                </div>

                <!-- Box 4 -->
                <div class="rounded-lg border border-ink/40 bg-white/50 p-2.5">
                  <p class="font-serif italic text-xs font-bold text-ink">Aula individual + Café</p>
                  <p class="text-[11px] text-ink-soft">Yoga ou HYROX + Café da Manhã</p>
                  <p class="font-sans text-sm font-bold text-ink">R$ 55,00</p>
                </div>
              </div>

              <p class="mt-5 text-center font-serif italic text-[11px] font-semibold text-ink">
                Vagas limitadas a 50 pessoas por turma.
              </p>

              <!-- Footer Logos inside Poster -->
              <div class="mt-5 flex items-center justify-between border-t border-line/60 pt-4 text-[10px] font-bold tracking-wider uppercase text-ink">
                <div class="flex flex-col">
                  <span>P5</span>
                  <span class="text-[8px] text-ink-soft tracking-widest">KITE HOUSE</span>
                </div>
                <div class="flex flex-col text-right text-magenta">
                  <span class="font-sans lowercase font-bold tracking-tighter text-base -mb-1">ayo</span>
                  <span class="text-[9px] font-semibold tracking-normal italic text-magenta">gym</span>
                </div>
              </div>
            </div>

            <!-- Floating Top Badge -->
            <span class="absolute -top-3 -right-2 inline-flex items-center gap-1.5 rounded-full border border-magenta/20 bg-white px-3 py-1 text-[11px] font-semibold text-magenta shadow-sm">
              <Heart :size="12" class="fill-magenta" /> wellness à beira-mar
            </span>

            <!-- Bottom Label -->
            <span class="absolute -bottom-3 left-6 rounded-full border border-line bg-white px-3.5 py-1 text-[11px] font-medium text-ink-soft shadow-sm">
              por P5 • com AYO
            </span>
          </div>
        </div>
      </section>

      <!-- SECTION 1: O seu bem-estar encontra a praia -->
      <section id="experiencia" class="border-t border-line/60 bg-warm/40 py-20">
        <div class="mx-auto max-w-6xl px-6">
          <p class="eyebrow mb-2">UMA EXPERIÊNCIA COMPLETA</p>
          <div class="grid gap-6 md:grid-cols-[1.2fr_1fr] md:items-end">
            <h2 class="font-serif text-3xl font-bold leading-tight text-ink md:text-5xl">
              O seu bem-estar<br />
              <em class="font-serif italic text-magenta">encontra a praia.</em>
            </h2>
            <p class="text-base leading-relaxed text-ink-soft">
              Do primeiro alongamento ao café da manhã, cada etapa foi pensada para você sair mais leve do que chegou.
            </p>
          </div>

          <!-- 3 Cards -->
          <div class="mt-12 grid gap-6 md:grid-cols-3">
            <!-- Card 1 -->
            <div class="relative flex flex-col justify-between rounded-[var(--radius-card)] bg-sky p-7 text-ink min-h-[220px]">
              <div>
                <div class="flex items-center justify-between">
                  <Waves :size="24" class="text-ink" />
                  <span class="font-mono text-xs font-semibold text-ink/60">01</span>
                </div>
                <h3 class="mt-6 font-serif text-2xl font-bold text-ink">Movimento</h3>
                <p class="mt-3 text-sm leading-relaxed text-ink/80">
                  Yoga e HYROX para acordar o corpo com intenção, presença e energia.
                </p>
              </div>
            </div>

            <!-- Card 2 -->
            <div class="relative flex flex-col justify-between rounded-[var(--radius-card)] bg-warm p-7 text-ink min-h-[220px]">
              <div>
                <div class="flex items-center justify-between">
                  <Sparkles :size="24" class="text-ink" />
                  <span class="font-mono text-xs font-semibold text-ink/60">02</span>
                </div>
                <h3 class="mt-6 font-serif text-2xl font-bold text-ink">Energia</h3>
                <p class="mt-3 text-sm leading-relaxed text-ink-soft">
                  Práticas guiadas para todos os níveis, em uma atmosfera acolhedora.
                </p>
              </div>
            </div>

            <!-- Card 3 -->
            <div class="relative flex flex-col justify-between rounded-[var(--radius-card)] bg-ink p-7 text-white min-h-[220px]">
              <div>
                <div class="flex items-center justify-between">
                  <Coffee :size="24" class="text-white" />
                  <span class="font-mono text-xs font-semibold text-white/50">03</span>
                </div>
                <h3 class="mt-6 font-serif text-2xl font-bold text-white">Conexão</h3>
                <p class="mt-3 text-sm leading-relaxed text-white/70">
                  Combos com café da manhã para prolongar o encontro à mesa.
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- SECTION 2: MAGENTA BANNER -->
      <section class="bg-magenta py-16 text-white">
        <div class="mx-auto flex max-w-6xl flex-col items-start justify-between gap-8 px-6 md:flex-row md:items-center">
          <div>
            <p class="font-mono text-xs font-medium tracking-widest uppercase text-white/80">
              📅 PREPARE-SE PARA O SEU PRÓXIMO ENCONTRO
            </p>
            <h2 class="mt-2 font-serif text-3xl font-bold leading-tight md:text-4xl">
              Uma agenda que acompanha<br />
              <em class="font-serif italic text-white">o seu ritmo.</em>
            </h2>
            <p v-if="sameDayLabel" class="mt-3 text-sm font-semibold text-white/90">
              Próximo encontro: {{ sameDayLabel }}
            </p>
            <ul v-else-if="nextSessions.length" class="mt-3 space-y-0.5 text-sm font-semibold text-white/90">
              <li v-for="s in nextSessions" :key="s.activityTitle">
                {{ s.activityTitle }}: {{ formatEventDate(s.startsAt) }}
              </li>
            </ul>
          </div>
          <button
            class="inline-flex items-center gap-2 rounded-full bg-white px-7 py-3.5 font-semibold text-ink shadow-md transition-transform hover:scale-105"
            @click="router.push('/comprar')"
          >
            Ver agenda
            <ArrowRight :size="18" />
          </button>
        </div>
      </section>

      <!-- SECTION 3: SEU MOMENTO, SUA ESCOLHA -->
      <section id="aulas-combos" class="py-20">
        <div class="mx-auto max-w-6xl px-6 text-center">
          <p class="eyebrow mb-2">ESCOLHA COMO VIVER</p>
          <h2 class="font-serif text-3xl font-bold text-ink md:text-5xl">
            Seu momento, <em class="font-serif italic text-magenta">sua escolha.</em>
          </h2>
          <p class="mt-4 max-w-lg mx-auto text-base text-ink-soft">
            Selecione uma aula ou combine práticas e sabores para uma experiência ainda mais completa.
          </p>

          <div v-if="!products.length" class="mt-12 rounded-[var(--radius-card)] border border-line bg-white p-10 text-center text-ink-soft">
            Agenda em preparação — novas experiências chegando em breve.
          </div>

          <div v-else class="mt-12 grid gap-6 md:grid-cols-3 text-left">
            <article
              v-for="p in products"
              :key="p.id"
              :class="[
                'flex flex-col justify-between rounded-[var(--radius-card)] p-7 transition-all',
                p.featured
                  ? 'bg-ink text-white shadow-xl'
                  : 'border border-line bg-white text-ink shadow-sm hover:border-ink/40'
              ]"
            >
              <div>
                <span
                  :class="[
                    'mb-4 inline-block font-mono text-[10px] font-bold tracking-widest uppercase',
                    p.featured ? 'text-magenta' : 'text-magenta'
                  ]"
                >
                  {{ p.featured ? 'MAIS COMPLETA ✦' : 'P5 WELLNESS' }}
                </span>
                <h3 class="font-serif text-xl font-bold leading-snug">{{ p.title }}</h3>
                <p :class="['mt-2 text-xs leading-relaxed', p.featured ? 'text-white/70' : 'text-ink-soft']">
                  {{ p.description }}
                </p>
                <p :class="['mt-4 text-xs italic', p.featured ? 'text-white/50' : 'text-ink-soft/70']">
                  {{ detailFor(p) }}
                </p>
              </div>

              <div class="mt-8 flex items-center justify-between border-t border-line/30 pt-6">
                <div>
                  <span :class="['block text-[10px] uppercase font-mono', p.featured ? 'text-white/50' : 'text-ink-soft']">Valor</span>
                  <span class="font-sans text-xl font-bold">{{ formatBRL(p.priceCents) }}</span>
                </div>
                <button
                  :class="[
                    'px-5 py-2.5 text-xs font-semibold rounded-full inline-flex items-center gap-1.5 transition-colors',
                    p.featured ? 'bg-warm text-ink hover:bg-white' : 'button-outline'
                  ]"
                  @click="router.push('/comprar')"
                >
                  Escolher
                  <ArrowRight :size="14" />
                </button>
              </div>
            </article>
          </div>

          <p class="mt-8 text-xs italic font-medium text-ink-soft">Vagas limitadas a 50 pessoas por turma.</p>
        </div>
      </section>

      <!-- SECTION: CAFÉ DA MANHÃ - AO NASCER DO VENTO -->
      <section id="cafe-da-manha" class="border-t border-line/60 bg-[#FAF5EE] py-20 relative overflow-hidden">
        <div class="hero-orb hero-orb--cream"></div>
        <div class="hero-orb hero-orb--blue"></div>

        <div class="relative z-10 mx-auto max-w-6xl px-6">
          <div class="flex flex-col items-center text-center">
            <p class="eyebrow mb-2">
              <Coffee :size="14" class="text-magenta" />
              AO NASCER DO VENTO
            </p>
            <h2 class="font-serif text-3xl font-bold leading-tight text-ink md:text-5xl">
              Café da Manhã<br />
              <em class="font-serif italic text-magenta">ao nascer do vento.</em>
            </h2>
            <p class="mt-4 max-w-xl text-base leading-relaxed text-ink-soft">
              Deliciosas opções preparadas com ingredientes selecionados para renovar suas energias após a aula no P5 Wellness Club.
            </p>
          </div>

          <!-- Highlight Box replicating the image pink framed badge -->
          <div class="mt-8 mx-auto max-w-lg rounded-2xl border-2 border-magenta/40 bg-white/90 p-5 text-center shadow-xs backdrop-blur-xs">
            <div class="inline-flex items-center gap-2 font-serif text-base font-bold text-ink">
              <Sparkles :size="18" class="text-magenta" />
              Pacote Aulas + Café da Manhã
            </div>
            <p class="mt-1 text-xs text-ink-soft">
              Opções de café da manhã para você escolher após a aula.
            </p>
            <p class="mt-2 inline-flex items-center gap-1.5 font-mono text-xs font-semibold text-magenta">
              <Clock :size="14" />
              Funcionamento a partir das 7h
            </p>
          </div>

          <!-- Menu Board Graphic styled like the physical menu card -->
          <div class="mt-12 mx-auto max-w-4xl rounded-3xl border border-line/80 bg-[#FFFDF9] p-8 md:p-12 shadow-xl relative">
            <!-- Header inside menu card -->
            <div class="border-b border-line/70 pb-6 mb-10">
              <div class="flex flex-wrap items-center justify-between gap-4">
                <div class="flex flex-col">
                  <div class="flex items-center gap-1.5">
                    <span class="font-sans text-2xl font-black tracking-tighter text-ink">P5</span>
                    <span class="font-serif text-2xl italic text-ink">wellness</span>
                  </div>
                  <span class="font-sans text-[10px] font-bold tracking-[0.3em] uppercase text-ink">CLUB</span>
                </div>
                <div class="text-right">
                  <p class="font-mono text-xs tracking-widest uppercase text-ink-soft font-semibold">AO NASCER DO VENTO</p>
                  <p class="font-serif italic text-base text-magenta font-semibold">Café da Manhã</p>
                </div>
              </div>
              <p class="mt-4 text-center font-serif text-lg md:text-xl font-bold text-magenta">
                Para tornar sua experiência completa, escolha uma opção.
              </p>
            </div>

            <!-- Two-Column Menu Categories -->
            <div class="grid gap-10 md:grid-cols-2">
              <!-- Left Column: Cuscuz, Omelete, Pão -->
              <div class="space-y-8">
                <!-- Cuscuz -->
                <div>
                  <h3 class="flex items-center gap-2 font-serif text-xl font-bold text-ink border-b border-line/50 pb-2 mb-4">
                    <span class="text-magenta">•</span> Cuscuz
                  </h3>
                  <div class="space-y-3 pl-2">
                    <div>
                      <h4 class="font-sans text-sm font-bold text-ink">Sabor do Sertão Wellness</h4>
                      <p class="text-xs text-ink-soft mt-0.5">Cuscuz com Carne de Sol + Suco de Laranja 200 ml</p>
                    </div>
                  </div>
                </div>

                <!-- Omelete -->
                <div>
                  <h3 class="flex items-center gap-2 font-serif text-xl font-bold text-ink border-b border-line/50 pb-2 mb-4">
                    <span class="text-magenta">•</span> Omelete
                  </h3>
                  <div class="space-y-3 pl-2">
                    <div>
                      <h4 class="font-sans text-sm font-bold text-ink">Omelete Equilíbrio Wellness</h4>
                      <p class="text-xs text-ink-soft mt-0.5">Omelete de Frango Desfiado + Suco de Laranja 200 ml</p>
                    </div>
                  </div>
                </div>

                <!-- Pão -->
                <div>
                  <h3 class="flex items-center gap-2 font-serif text-xl font-bold text-ink border-b border-line/50 pb-2 mb-4">
                    <span class="text-magenta">•</span> Pão
                  </h3>
                  <div class="space-y-4 pl-2">
                    <div>
                      <h4 class="font-sans text-sm font-bold text-ink">Pão Sertanejo Wellness</h4>
                      <p class="text-xs text-ink-soft mt-0.5">Pão com Carne de Sol + Café com Leite 150 ml</p>
                    </div>
                    <div>
                      <h4 class="font-sans text-sm font-bold text-ink">Misto & Cappuccino Wellness</h4>
                      <p class="text-xs text-ink-soft mt-0.5">Pão Misto Quente + Cappuccino 150 ml</p>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Right Column: Tapioca & Crepioca, Croissant -->
              <div class="space-y-8">
                <!-- Tapioca & Crepioca -->
                <div>
                  <h3 class="flex items-center gap-2 font-serif text-xl font-bold text-ink border-b border-line/50 pb-2 mb-4">
                    <span class="text-magenta">•</span> Tapioca & Crepioca
                  </h3>
                  <div class="space-y-4 pl-2">
                    <div>
                      <h4 class="font-sans text-sm font-bold text-ink">Tapioca Tropical Wellness</h4>
                      <p class="text-xs text-ink-soft mt-0.5">Tapioca com Frango Desfiado + Suco de Maracujá 200 ml</p>
                    </div>
                    <div>
                      <h4 class="font-sans text-sm font-bold text-ink">Crepioca Energia Wellness</h4>
                      <p class="text-xs text-ink-soft mt-0.5">Crepioca com Frango Desfiado + Café com Leite 150 ml</p>
                    </div>
                  </div>
                </div>

                <!-- Croissant -->
                <div>
                  <h3 class="flex items-center gap-2 font-serif text-xl font-bold text-ink border-b border-line/50 pb-2 mb-4">
                    <span class="text-magenta">•</span> Croissant
                  </h3>
                  <div class="space-y-4 pl-2">
                    <div>
                      <h4 class="font-sans text-sm font-bold text-ink">Croissant Clássico Wellness</h4>
                      <p class="text-xs text-ink-soft mt-0.5">Croissant de Queijo + Café Expresso Supremo</p>
                    </div>
                    <div>
                      <h4 class="font-sans text-sm font-bold text-ink">Croissant Misto Wellness</h4>
                      <p class="text-xs text-ink-soft mt-0.5">Croissant Misto + Café Expresso Supremo</p>
                    </div>
                    <div>
                      <h4 class="font-sans text-sm font-bold text-ink">Croissant Frango Wellness</h4>
                      <p class="text-xs text-ink-soft mt-0.5">Croissant de Frango Desfiado + Café Expresso Supremo</p>
                    </div>
                    <div>
                      <h4 class="font-sans text-sm font-bold text-ink">Croissant Doce Wellness</h4>
                      <p class="text-xs text-ink-soft mt-0.5">Croissant de Nutella + Café Expresso Supremo</p>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- Footer Co-branding inside menu card -->
            <div class="mt-12 flex items-center justify-between border-t border-line/70 pt-6 text-[10px] font-bold tracking-wider uppercase text-ink">
              <div class="flex flex-col">
                <span>P5</span>
                <span class="text-[8px] text-ink-soft tracking-widest">KITE HOUSE</span>
              </div>
              <div class="flex flex-col text-right text-magenta">
                <span class="font-sans lowercase font-bold tracking-tighter text-base -mb-1">ayo</span>
                <span class="text-[9px] font-semibold tracking-normal italic text-magenta">gym</span>
              </div>
            </div>
          </div>

          <!-- Bottom Action Button -->
          <div class="mt-10 text-center">
            <button class="button-magenta" @click="router.push('/comprar')">
              Escolher experiência com Café
              <ArrowRight :size="18" />
            </button>
          </div>
        </div>
      </section>

      <!-- SECTION 5: SEU PASSE PARA VIVER O AGORA -->
      <section id="como-funciona" class="border-t border-line/60 bg-warm/30 py-20">
        <div class="mx-auto max-w-6xl px-6 grid gap-12 md:grid-cols-[1fr_1.2fr] md:items-center">
          <div>
            <p class="eyebrow mb-2">SIMPLES DO INÍCIO AO FIM</p>
            <h2 class="font-serif text-3xl font-bold leading-tight text-ink md:text-5xl">
              Seu passe para<br />
              <em class="font-serif italic text-magenta">viver o agora.</em>
            </h2>
          </div>

          <div class="space-y-6">
            <div class="flex items-start gap-5 border-b border-line/80 pb-6">
              <span class="font-mono text-xs font-bold text-magenta">01</span>
              <div>
                <h3 class="font-sans text-base font-bold text-ink">Crie seu cadastro.</h3>
                <p class="mt-1 text-sm text-ink-soft">Seus dados ficam guardados para as próximas experiências.</p>
              </div>
            </div>

            <div class="flex items-start gap-5 border-b border-line/80 pb-6">
              <span class="font-mono text-xs font-bold text-magenta">02</span>
              <div>
                <h3 class="font-sans text-base font-bold text-ink">Escolha data e modalidade.</h3>
                <p class="mt-1 text-sm text-ink-soft">Reserve uma aula ou seu combo preferido.</p>
              </div>
            </div>

            <div class="flex items-start gap-5">
              <span class="font-mono text-xs font-bold text-magenta">03</span>
              <div>
                <h3 class="font-sans text-base font-bold text-ink">Apresente seu QR Code.</h3>
                <p class="mt-1 text-sm text-ink-soft">Um check-in rápido e seguro na entrada e no café, quando incluso.</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- FOOTER -->
      <footer class="border-t border-line/80 bg-ink py-12 text-white">
        <div class="mx-auto flex max-w-6xl flex-col items-center justify-between gap-6 px-6 md:flex-row">
          <div class="flex items-center gap-2">
            <span class="font-sans text-xl font-bold text-white">P5</span>
            <span class="text-lg font-light text-magenta">/</span>
            <span class="font-sans text-sm font-bold tracking-wider text-magenta">AYO</span>
          </div>

          <p class="text-xs text-white/60 text-center md:text-left">
            P5 Kite House × AYO Wellness.<br />
            Movimento, energia e conexão.
          </p>

          <div class="flex items-center gap-6 text-xs font-medium text-white/80">
            <button class="hover:text-white" @click="router.push('/entrar')">Área do aluno</button>
            <button class="hover:text-white" @click="router.push('/acesso-admin')">Acesso equipe</button>
          </div>
        </div>
      </footer>
    </main>
  </div>
</template>

