<script setup lang="ts">
import { computed } from "vue";
import { useRouter } from "vue-router";
import { useQuery } from "@tanstack/vue-query";
import {
  ArrowRight,
  Check,
  Compass,
  Wind,
  MapPin,
  Bus,
  LifeBuoy,
  Droplets,
  Radar,
  HandHelping,
  Clock,
  Flag,
  ShieldCheck,
} from "lucide-vue-next";
import { api } from "@/lib/api";
import WellnessHeader from "@/components/WellnessHeader.vue";
import HelpFab from "@/components/HelpFab.vue";
import BrandMark from "@/components/BrandMark.vue";
import WaveIcon from "@/components/WaveIcon.vue";
import KiteVectorArt from "@/components/KiteVectorArt.vue";

const router = useRouter();

function scrollToInclui() {
  document.querySelector("#inclui")?.scrollIntoView({ behavior: "smooth" });
}

// ── Próxima data ────────────────────────────────────────────────────────────
// Pulls the real turma date straight from the admin's cadastro so the landing page
// shows it up front, before checkout.
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
  return label.charAt(0).toUpperCase() + label.slice(1);
}

const nextDateLabel = computed(() => {
  const sessions = nextSessions.value;
  if (!sessions.length) return null;
  return formatEventDate(sessions[0].startsAt);
});

const included = [
  { icon: MapPin, title: "Percurso guiado", detail: "Praia do Presídio, com toda a rota acompanhada pela equipe P5." },
  { icon: Bus, title: "Transporte incluso", detail: "Ida da P5 Kite House até o ponto de partida." },
  { icon: LifeBuoy, title: "Apoio aquático e terrestre", detail: "Equipe P5 acompanhando você na água e em terra." },
  { icon: Droplets, title: "Estrutura na saída", detail: "Compressor, banheiro e hidratação no ponto de partida." },
  { icon: Radar, title: "Monitoramento Wind Maps", detail: "Condições de vento acompanhadas em tempo real." },
  { icon: HandHelping, title: "Receptivo na chegada", detail: "Auxílio na desmontagem do equipamento." },
  { icon: Clock, title: "Saída às 7:30", detail: "Concentração na P5 Kite House." },
  { icon: Flag, title: "Chegada até as 13h", detail: "Encerramento dentro do previsto." },
];

const itinerary = [
  { time: "7:30", title: "Concentração", detail: "Chegada na P5 Kite House, checagem de equipamento e briefing." },
  { time: "→", title: "Transporte", detail: "Deslocamento estruturado até o ponto de partida do percurso." },
  { time: "↝", title: "Percurso", detail: "Downwind até a Praia do Presídio, com apoio aquático e terrestre o tempo todo." },
  { time: "13h", title: "Chegada", detail: "Receptivo, ajuda na desmontagem e encerramento do dia." },
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
              P5 KITE HOUSE · PRAIA DO PRESÍDIO
            </p>

            <h1 class="font-serif text-5xl font-black leading-[1.05] text-ink md:text-6xl">
              Vento a favor,<br />
              <span class="text-magenta">mar em movimento.</span>
            </h1>

            <p class="mt-6 max-w-md text-base leading-relaxed text-ink-soft md:text-lg">
              O <strong class="font-semibold text-ink">P5 DownWind Day</strong> é o percurso guiado da P5 Kite House até a Praia do Presídio — transporte, apoio completo e estrutura do início ao fim. Você entra na água, a gente cuida do resto.
            </p>

            <div class="mt-8 flex flex-wrap items-center gap-5">
              <button class="button-magenta" @click="router.push('/comprar')">
                Garantir minha vaga · R$ 100
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
              <span class="inline-flex items-center gap-1.5"><Check :size="15" class="text-magenta" /> Saída 7:30 da P5 Kite House</span>
              <span class="inline-flex items-center gap-1.5"><Check :size="15" class="text-magenta" /> Apoio aquático e terrestre</span>
            </div>
          </div>

          <!-- TICKET CARD GRAPHIC -->
          <div class="relative mx-auto w-full max-w-sm">
            <div class="relative rounded-[1.75rem] border border-line/80 bg-white p-6 shadow-2xl">
              <div class="flex items-center justify-between">
                <BrandMark :size="30" />
                <span class="font-mono text-[10px] font-bold tracking-widest text-ink-soft uppercase">Acesso único</span>
              </div>

              <div class="mt-6 flex items-center justify-between gap-2 px-1">
                <div>
                  <p class="font-mono text-[10px] font-bold tracking-widest text-ink-soft uppercase">Partida</p>
                  <p class="font-serif text-lg font-bold text-ink">P5 Kite House</p>
                  <p class="text-xs text-ink-soft">7:30</p>
                </div>
                <Wind :size="22" class="shrink-0 text-magenta" />
                <div class="text-right">
                  <p class="font-mono text-[10px] font-bold tracking-widest text-ink-soft uppercase">Percurso</p>
                  <p class="font-serif text-lg font-bold text-ink">Praia do Presídio</p>
                  <p class="text-xs text-ink-soft">até 13h</p>
                </div>
              </div>

              <div class="relative my-6">
                <div class="ticket-perforation"></div>
                <span class="ticket-notch -left-9"></span>
                <span class="ticket-notch -right-9"></span>
              </div>

              <div class="flex items-center justify-between px-1">
                <div>
                  <p class="font-mono text-[10px] font-bold tracking-widest text-ink-soft uppercase">Valor</p>
                  <p class="font-serif text-2xl font-black text-ink">R$ 100</p>
                </div>
                <span class="rounded-full border border-magenta/30 bg-magenta/10 px-3 py-1.5 text-[11px] font-semibold text-magenta">
                  Vagas limitadas
                </span>
              </div>
            </div>

            <!-- Floating Top Badge -->
            <span v-if="nextDateLabel" class="absolute -top-3 -right-2 inline-flex items-center gap-1.5 rounded-full border border-magenta/20 bg-white px-3 py-1 text-[11px] font-semibold text-magenta shadow-sm">
              <Compass :size="12" /> próxima data: {{ nextDateLabel }}
            </span>

            <!-- Bottom Label -->
            <span class="absolute -bottom-3 left-6 rounded-full border border-line bg-white px-3.5 py-1 text-[11px] font-medium text-ink-soft shadow-sm">
              por P5 Kite House
            </span>
          </div>
        </div>
      </section>

      <!-- SECTION: FOTO ATMOSFÉRICA -->
      <section class="relative h-80 overflow-hidden border-t border-line/60 md:h-104">
        <img
          src="/photo-ocean-band.webp"
          alt="Kitesurfista no mar aberto, vento forte e ondas"
          class="absolute inset-0 h-full w-full object-cover object-[center_35%]"
        />
        <div class="absolute inset-0 bg-linear-to-t from-ink/80 via-ink/10 to-transparent"></div>
        <div class="relative z-10 mx-auto flex h-full max-w-6xl flex-col justify-end px-6 pb-8">
          <p class="font-mono text-xs font-semibold tracking-widest text-sky-300 uppercase">Mar aberto</p>
          <p class="mt-1 max-w-md font-serif text-2xl font-bold text-white md:text-3xl">
            Onde o vento encontra o mar aberto.
          </p>
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
              Do embarque na P5 Kite House até o encerramento na Praia do Presídio, toda a estrutura já está no seu ingresso.
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
          <p class="eyebrow mb-2"><WaveIcon :size="14" /> DO EMBARQUE AO DESEMBARQUE</p>
          <h2 class="font-serif text-3xl font-bold text-ink md:text-5xl">
            Como funciona <span class="text-magenta">o seu dia.</span>
          </h2>

          <div class="mt-14 grid gap-8 md:grid-cols-4">
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
          <div class="grid gap-10 md:grid-cols-[1.1fr_0.9fr] md:items-center">
            <div>
              <p class="font-mono text-xs font-medium tracking-widest uppercase text-white/60">SEGURANÇA E ESTRUTURA</p>
              <h2 class="mt-2 font-serif text-3xl font-bold leading-tight md:text-4xl">
                Feito para quem leva o vento <span class="text-sky-300">a sério.</span>
              </h2>

              <div class="mt-12 grid gap-8 sm:grid-cols-3 md:grid-cols-1 md:gap-7">
                <div class="flex flex-col gap-3">
                  <LifeBuoy :size="26" class="text-sky-300" />
                  <h3 class="font-serif text-lg font-bold">Apoio o tempo todo</h3>
                  <p class="text-sm leading-relaxed text-white/70">
                    Equipe P5 acompanhando na água e em terra durante todo o percurso, do início ao fim.
                  </p>
                </div>
                <div class="flex flex-col gap-3">
                  <Radar :size="26" class="text-sky-300" />
                  <h3 class="font-serif text-lg font-bold">Monitoramento em tempo real</h3>
                  <p class="text-sm leading-relaxed text-white/70">
                    Condições de vento acompanhadas pelo aplicativo Wind Maps antes e durante o dia.
                  </p>
                </div>
                <div class="flex flex-col gap-3">
                  <ShieldCheck :size="26" class="text-sky-300" />
                  <h3 class="font-serif text-lg font-bold">Estrutura completa</h3>
                  <p class="text-sm leading-relaxed text-white/70">
                    Compressor, banheiro e hidratação no ponto de saída, mais receptivo na chegada.
                  </p>
                </div>
              </div>
            </div>

            <div class="relative mx-auto h-105 w-full max-w-55 md:h-140">
              <div class="absolute inset-0 -z-10 rounded-full bg-sky-400/10 blur-3xl"></div>
              <KiteVectorArt inverted />
            </div>
          </div>
        </div>
      </section>

      <!-- SECTION: PREÇO / CTA -->
      <section id="preco" class="border-t border-line/60 py-20">
        <div class="mx-auto max-w-4xl px-6 text-center">
          <p class="eyebrow mb-2 justify-center">INGRESSO ÚNICO</p>
          <h2 class="font-serif text-3xl font-bold text-ink md:text-5xl">
            Um valor, <span class="text-magenta">tudo incluso.</span>
          </h2>
          <p class="mt-4 font-serif text-6xl font-black text-ink md:text-7xl">R$ 100</p>
          <p class="mt-3 text-sm text-ink-soft">
            Percurso, transporte, apoio aquático e terrestre e estrutura completa no ponto de saída.
          </p>

          <div class="mt-8">
            <button class="button-magenta" @click="router.push('/comprar')">
              Garantir minha vaga
              <ArrowRight :size="18" />
            </button>
          </div>
          <p class="mt-4 text-xs italic font-medium text-ink-soft">Vagas limitadas · Pagamento via Pix.</p>
        </div>
      </section>

      <!-- FOOTER -->
      <footer class="border-t border-line/80 bg-ink py-12 text-white">
        <div class="mx-auto flex max-w-6xl flex-col items-center justify-between gap-6 px-6 md:flex-row">
          <BrandMark :size="32" inverted />

          <p class="text-xs text-white/60 text-center md:text-left">
            P5 Kite House.<br />
            Vento a favor, sempre.
          </p>

          <div class="flex items-center gap-6 text-xs font-medium text-white/80">
            <button class="hover:text-white" @click="router.push('/entrar')">Área do aluno</button>
            <button class="hover:text-white" @click="router.push('/acesso-admin')">Acesso equipe</button>
          </div>
        </div>

        <div class="mt-10 flex justify-center border-t border-white/10 px-6 pt-6">
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
