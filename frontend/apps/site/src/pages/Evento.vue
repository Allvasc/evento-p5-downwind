<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { ArrowRight, Check, Compass, MapPin, CalendarDays, Sparkles } from "lucide-vue-next";
import { formatBRL, cleanTitle, isUnlimitedCapacity } from "@p5wellness/shared";
import WellnessHeader from "@/components/WellnessHeader.vue";
import {
  useEventSlug,
  eventPaths,
  usePublicEvent,
  useEventProducts,
  useEventNextSessions,
  parseEventLines,
} from "@/composables/usePublicEvent";

// Página genérica dos eventos criados pelo Admin (/<slug>). Todo o conteúdo vem do
// cadastro do evento; produtos, preços e datas vêm só do catálogo desse evento.
const router = useRouter();
const slug = useEventSlug();
const paths = computed(() => eventPaths(slug.value));

const { data: event, isLoading: loadingEvent, isError: eventMissing } = usePublicEvent(slug);
const { data: productsData } = useEventProducts(slug);
const { data: nextSessionsData } = useEventNextSessions(slug);

// O evento principal tem a página "/" (Home.vue); o endereço dele só redireciona pra lá.
watch(event, (e) => {
  if (e?.isDefault) router.replace("/");
}, { immediate: true });

watch(event, (e) => {
  if (e) document.title = e.name;
}, { immediate: true });

const products = computed(() => productsData.value?.products ?? []);
const selectedIndex = ref(0);
watch(products, (list) => {
  const featured = list.findIndex((p) => p.featured);
  selectedIndex.value = featured !== -1 ? featured : 0;
}, { immediate: true });
const activeProduct = computed(() => products.value[selectedIndex.value] ?? products.value[0] ?? null);
const isFree = computed(() => activeProduct.value?.priceCents === 0);

const nextSession = computed(() => nextSessionsData.value?.sessions?.[0] ?? null);
function formatEventDate(iso: string) {
  const label = new Date(iso).toLocaleDateString("pt-BR", { weekday: "long", day: "2-digit", month: "long", timeZone: "America/Fortaleza" });
  return label.charAt(0).toUpperCase() + label.slice(1);
}
const dateLabel = computed(() => event.value?.dateLabel || (nextSession.value ? formatEventDate(nextSession.value.startsAt) : ""));
const spotsLabel = computed(() => {
  const s = nextSession.value;
  if (!s) return "";
  if (isUnlimitedCapacity(s.capacity)) return "Vagas ilimitadas";
  const left = Math.max(s.capacity - s.booked, 0);
  if (left === 0) return "Vagas esgotadas";
  return left < s.capacity ? `Restam ${left} de ${s.capacity} vagas` : `Apenas ${s.capacity} vagas`;
});

const includes = computed(() => parseEventLines(event.value?.includes ?? ""));
const schedule = computed(() => parseEventLines(event.value?.schedule ?? ""));
const descriptionParagraphs = computed(() =>
  (event.value?.description ?? "").split(/\n\s*\n/).map((p) => p.trim()).filter(Boolean),
);

function goBuy() {
  router.push(paths.value.buy);
}
</script>

<template>
  <div class="min-h-screen bg-paper">
    <WellnessHeader :event-slug="slug" :event-name="event?.name" />

    <main v-if="loadingEvent" class="mx-auto max-w-6xl px-6 py-24 text-sm text-ink-soft">Carregando evento…</main>

    <main v-else-if="eventMissing || !event" class="mx-auto max-w-xl px-6 py-24 text-center">
      <p class="eyebrow mb-3 justify-center">P5 KITE HOUSE</p>
      <h1 class="font-serif text-3xl font-bold text-ink">Evento não encontrado</h1>
      <p class="mt-3 text-sm text-ink-soft">Este endereço não corresponde a nenhum evento com inscrições abertas.</p>
      <button class="button-magenta mx-auto mt-8" @click="router.push('/')">Ver o evento em destaque <ArrowRight :size="16" /></button>
    </main>

    <main v-else-if="!event.isDefault">
      <!-- HERO -->
      <section class="relative overflow-hidden pt-4 pb-16 md:py-20">
        <div class="hero-orb hero-orb--blue"></div>
        <div class="hero-orb hero-orb--cream"></div>

        <div class="relative z-10 mx-auto grid max-w-6xl gap-12 px-6 md:grid-cols-[1.1fr_0.9fr] md:items-center">
          <div>
            <p class="eyebrow mb-6 flex flex-wrap items-center gap-2 text-xs tracking-wider">
              <Compass :size="14" class="text-magenta" />
              <span>P5 KITE HOUSE</span>
              <span class="text-ink-soft/40">·</span>
              <span class="uppercase">{{ event.name }}</span>
            </p>

            <h1 class="font-serif text-4xl font-black leading-[1.05] text-ink md:text-6xl">
              {{ event.headline || event.name }}
            </h1>
            <p v-if="event.subtitle" class="mt-4 font-serif text-xl font-semibold text-magenta md:text-2xl">{{ event.subtitle }}</p>

            <p v-if="descriptionParagraphs.length" class="mt-6 max-w-md text-base leading-relaxed text-ink-soft md:text-lg">
              <strong v-if="dateLabel" class="font-semibold text-ink">{{ dateLabel }} · </strong>{{ descriptionParagraphs[0] }}
            </p>

            <div class="mt-8 flex flex-wrap items-center gap-5">
              <button class="button-magenta" :disabled="!products.length" @click="goBuy">
                <template v-if="!products.length">Inscrições em breve</template>
                <template v-else-if="isFree">Garantir inscrição gratuita</template>
                <template v-else>Garantir minha vaga<template v-if="activeProduct"> · {{ formatBRL(activeProduct.priceCents) }}</template></template>
                <ArrowRight :size="18" />
              </button>
              <a v-if="includes.length" href="#inclui" class="text-sm font-semibold text-ink underline underline-offset-4 transition-colors hover:text-magenta">
                Ver o que está incluído
              </a>
            </div>

            <div class="mt-8 flex flex-wrap gap-x-6 gap-y-2 text-xs font-medium text-ink-soft">
              <span v-if="dateLabel" class="inline-flex items-center gap-1.5"><CalendarDays :size="15" class="text-magenta" /> {{ dateLabel }}</span>
              <span v-if="event.locationLabel" class="inline-flex items-center gap-1.5"><MapPin :size="15" class="text-magenta" /> {{ event.locationLabel }}</span>
              <span v-if="spotsLabel" class="inline-flex items-center gap-1.5"><Check :size="15" class="text-magenta" /> {{ spotsLabel }}</span>
            </div>
          </div>

          <!-- Ingressos do evento -->
          <div class="relative mx-auto w-full max-w-md">
            <div class="relative rounded-[2rem] border border-line/80 bg-white p-6 shadow-2xl md:p-8">
              <p class="text-center font-serif text-lg font-bold text-ink">{{ event.name }}</p>
              <p class="mt-1 text-center text-xs font-medium text-ink-soft">
                {{ products.length ? (isFree ? "Inscrição gratuita." : "Escolha o seu ingresso.") : "Os ingressos serão publicados em breve." }}
              </p>

              <div v-if="products.length" class="mt-6 space-y-3">
                <div
                  v-for="(p, index) in products"
                  :key="p.id"
                  :class="[
                    'relative flex cursor-pointer items-center justify-between rounded-2xl border p-4 transition-all',
                    selectedIndex === index
                      ? 'border-magenta bg-white ring-2 ring-magenta/20 shadow-md'
                      : 'border-line/80 bg-warm/30 hover:border-ink/40 hover:bg-white',
                  ]"
                  @click="selectedIndex = index"
                >
                  <div class="pr-2">
                    <h3 class="font-serif text-sm font-bold text-ink">{{ cleanTitle(p.title) }}</h3>
                    <p v-if="p.description" class="mt-1 line-clamp-2 text-xs text-ink-soft">{{ p.description }}</p>
                  </div>
                  <div class="shrink-0 text-right">
                    <span v-if="p.priceCents === 0" class="block font-mono text-[9px] font-bold uppercase tracking-wider text-magenta">Gratuito</span>
                    <span class="font-sans text-base font-bold text-ink">{{ formatBRL(p.priceCents) }}</span>
                  </div>
                </div>
              </div>

              <button v-if="products.length" class="button-magenta mt-6 w-full justify-center" @click="goBuy">
                {{ isFree ? "Garantir inscrição gratuita" : "Comprar ingresso" }} <ArrowRight :size="16" />
              </button>
            </div>

            <span v-if="dateLabel" class="absolute -top-3 -right-2 inline-flex items-center gap-1.5 rounded-full border border-magenta/20 bg-white px-3.5 py-1 text-[11px] font-semibold text-magenta shadow-sm">
              <Compass :size="12" /> {{ dateLabel }}
            </span>
            <span class="absolute -bottom-3 left-6 rounded-full border border-line bg-white px-3.5 py-1 text-[11px] font-medium text-ink-soft shadow-sm">
              por P5 Kite House
            </span>
          </div>
        </div>
      </section>

      <!-- BANNER / CAMPANHA -->
      <section v-if="event.bannerUrl" class="border-t border-line/60 py-16">
        <div class="mx-auto max-w-6xl px-6">
          <img :src="event.bannerUrl" :alt="event.name" class="w-full rounded-[var(--radius-card)] border border-line/80 object-cover shadow-xl" loading="lazy" />
          <div class="mt-8 flex flex-col items-start justify-between gap-5 md:flex-row md:items-center">
            <p class="max-w-2xl font-serif text-2xl font-bold leading-snug text-ink">{{ event.headline || event.name }}</p>
            <button class="button-magenta shrink-0" :disabled="!products.length" @click="goBuy">
              {{ isFree ? "Garantir inscrição gratuita" : "Comprar ingresso" }} <ArrowRight :size="16" />
            </button>
          </div>
        </div>
      </section>

      <!-- SOBRE O EVENTO (parágrafos além do primeiro) -->
      <section v-if="descriptionParagraphs.length > 1" class="border-t border-line/60 py-16">
        <div class="mx-auto max-w-3xl space-y-4 px-6 text-base leading-relaxed text-ink-soft">
          <p v-for="(paragraph, i) in descriptionParagraphs.slice(1)" :key="i">{{ paragraph }}</p>
        </div>
      </section>

      <!-- O QUE ESTÁ INCLUSO -->
      <section v-if="includes.length" id="inclui" class="border-t border-line/60 bg-warm/40 py-20">
        <div class="mx-auto max-w-6xl px-6">
          <p class="eyebrow mb-2">TUDO INCLUÍDO NO SEU INGRESSO</p>
          <h2 class="font-serif text-3xl font-bold leading-tight text-ink md:text-5xl">
            O que está <span class="text-magenta">incluso.</span>
          </h2>

          <div class="mt-12 grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
            <div v-for="item in includes" :key="item.title" class="flex flex-col gap-4 rounded-[var(--radius-card)] border border-line/80 bg-white p-6">
              <span class="flex h-10 w-10 items-center justify-center rounded-full bg-magenta/10 text-magenta">
                <Sparkles :size="19" />
              </span>
              <div>
                <h3 class="font-sans text-sm font-bold text-ink">{{ item.title }}</h3>
                <p v-if="item.detail" class="mt-1.5 text-xs leading-relaxed text-ink-soft">{{ item.detail }}</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- PROGRAMAÇÃO -->
      <section v-if="schedule.length" id="programacao" class="py-20">
        <div class="mx-auto max-w-6xl px-6">
          <p class="eyebrow mb-2">PROGRAMAÇÃO</p>
          <h2 class="font-serif text-3xl font-bold text-ink md:text-5xl">
            Como funciona <span class="text-magenta">o seu dia.</span>
          </h2>

          <div class="mt-14 grid gap-x-8 gap-y-12 sm:grid-cols-2 md:grid-cols-4">
            <div v-for="(step, i) in schedule" :key="i" class="relative">
              <div class="flex items-center gap-3">
                <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-ink font-mono text-sm font-bold text-white">{{ i + 1 }}</span>
                <div v-if="i % 4 !== 3 && i < schedule.length - 1" class="hidden h-px flex-1 bg-line md:block"></div>
              </div>
              <h3 class="mt-5 font-serif text-lg font-bold text-ink">{{ step.title }}</h3>
              <p v-if="step.detail" class="mt-1.5 text-sm leading-relaxed text-ink-soft">{{ step.detail }}</p>
            </div>
          </div>
        </div>
      </section>

      <!-- INGRESSOS -->
      <section id="preco" class="border-t border-line/60 bg-warm/40 py-20">
        <div class="mx-auto max-w-6xl px-6 text-center">
          <p class="eyebrow mb-2 justify-center">INGRESSOS</p>
          <h2 class="font-serif text-3xl font-bold text-ink md:text-5xl">
            Garanta sua vaga no <span class="text-magenta">{{ event.name }}.</span>
          </h2>
          <p class="mt-3 text-sm text-ink-soft">
            <template v-if="spotsLabel">{{ spotsLabel }}. </template><template v-if="dateLabel">{{ dateLabel }}. </template>
            {{ isFree ? "Inscrição gratuita com confirmação instantânea." : "Pagamento seguro via Pix." }}
          </p>

          <p v-if="!products.length" class="mt-10 text-sm text-ink-soft">Os ingressos deste evento serão publicados aqui em breve.</p>
          <div v-else class="mx-auto mt-10 flex max-w-2xl flex-col gap-3 text-left">
            <div
              v-for="p in products"
              :key="p.id"
              :class="[
                'flex flex-col items-start justify-between gap-4 rounded-2xl border p-5 shadow-sm transition-all sm:flex-row sm:items-center',
                p.featured ? 'border-magenta/80 bg-white ring-2 ring-magenta/20' : 'border-line/80 bg-white/80 hover:border-ink/40',
              ]"
            >
              <div class="flex flex-wrap items-center gap-2.5">
                <h3 class="font-sans text-base font-bold text-ink">{{ cleanTitle(p.title) }}</h3>
                <span v-if="p.priceCents === 0" class="rounded-full bg-magenta/10 px-2.5 py-0.5 font-mono text-[9px] font-bold uppercase tracking-wider text-magenta">Gratuito</span>
              </div>
              <div class="flex w-full items-center justify-between gap-5 sm:w-auto sm:justify-end">
                <span class="font-serif text-2xl font-black text-ink">{{ formatBRL(p.priceCents) }}</span>
                <button class="button-magenta shrink-0 px-5! py-2.5! text-xs font-bold" @click="goBuy">
                  Garantir <ArrowRight :size="14" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- FOOTER -->
      <footer class="border-t border-line/80 bg-ink py-12 text-white">
        <div class="mx-auto flex max-w-6xl flex-col items-center justify-between gap-6 px-6 md:flex-row">
          <p class="font-serif text-lg font-bold">{{ event.name }}</p>
          <p class="text-center text-xs text-white/60 md:text-left">Um evento P5 Kite House.</p>
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
