<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useRouter } from "vue-router";
import { useQuery } from "@tanstack/vue-query";
import { ShieldCheck, ArrowLeft, ArrowRight } from "lucide-vue-next";
import { formatBRL, type Product } from "@p5wellness/shared";
import { api, ApiError } from "@/lib/api";
import WellnessHeader from "@/components/WellnessHeader.vue";

const router = useRouter();
const selectedId = ref<string | null>(null);
const selectedSlotKey = ref<string | null>(null);
const submitting = ref(false);
const errorMessage = ref("");

const { data, isLoading } = useQuery({
  queryKey: ["public-products"],
  queryFn: () => api.get<{ products: Product[] }>("/public/products"),
});

const products = computed(() => data.value?.products ?? []);
const selected = computed(() => products.value.find((p) => p.id === selectedId.value) ?? null);

// What to bring/wear, shown as a highlight under each activity's badge in step 1 — the
// moment the customer already knows which activities they're booking. Keyed by slug
// (stable) rather than title (display text, could be reworded in the catalog).
const ACTIVITY_TIPS: Record<string, string> = {
  yoga: "Traga seu tapete, toalha ou canga",
  hyrox: "Use roupas leves e confortáveis",
};
function activityTip(slug: string): string | null {
  return ACTIVITY_TIPS[slug] ?? null;
}

// For chooseOneActivity products (e.g. "Yoga ou HYROX"), the customer first picks which
// activity, then sees only that activity's dates — instead of the usual "every linked
// activity needs a matching session" grouping used for combos.
const chosenActivityId = ref<string | null>(null);

function selectProduct(id: string) {
  selectedId.value = id;
  selectedSlotKey.value = null;
  chosenActivityId.value = null;
}

function chooseActivity(activityId: string) {
  chosenActivityId.value = activityId;
  selectedSlotKey.value = null;
}

// ── Etapa 2: turmas disponíveis, agrupadas por dia ───────────────────────────
// Cada atividade do produto tem suas próprias turmas cadastradas no admin. Um "slot" só
// aparece se TODAS as atividades do produto tiverem uma turma no mesmo dia (fuso de
// Fortaleza) — os horários dentro do dia podem divergir entre atividades — assim a data
// escolhida cobre a experiência inteira. Café da manhã não entra aqui: não tem turma, é
// liberado direto no pagamento.
interface RawSession { id: string; activityId: string; activityTitle: string; startsAt: string; capacity: number; booked: number }
interface SlotSession { activityId: string; activityTitle: string; startsAt: string; spotsLeft: number }
interface Slot { key: string; day: string; sessions: SlotSession[]; sessionIds: Record<string, string>; minSpotsLeft: number }

// The event runs as a single-day gathering (Yoga + HYROX together, see Home.vue), so two
// activities scheduled at different times on the same Fortaleza calendar day still count
// as one bookable date — matching on the exact instant instead used to leave combos with
// zero slots whenever the two turmas weren't cadastradas at the identical minute.
const EVENT_TZ = "America/Fortaleza";
function fortalezaDay(iso: string) {
  return new Intl.DateTimeFormat("en-CA", { timeZone: EVENT_TZ, year: "numeric", month: "2-digit", day: "2-digit" }).format(new Date(iso));
}

const sessionsByActivity = ref<Record<string, RawSession[]>>({});
const loadingSlots = ref(false);
const slotsError = ref(false);

watch(selected, async (product) => {
  selectedSlotKey.value = null;
  chosenActivityId.value = null;
  sessionsByActivity.value = {};
  slotsError.value = false;
  if (!product || !product.activities.length) return;
  loadingSlots.value = true;
  try {
    const results = await Promise.all(
      product.activities.map((a) => api.get<{ sessions: RawSession[] }>(`/public/activities/${a.id}/sessions`).then((r) => [a.id, r.sessions] as const)),
    );
    sessionsByActivity.value = Object.fromEntries(results);
  } catch {
    // A fetch failure must not be silently mistaken for "no turmas cadastradas" — that
    // used to happen here since only `finally` handled the promise, leaving
    // sessionsByActivity empty and reusing the generic no-availability message below.
    slotsError.value = true;
  } finally {
    loadingSlots.value = false;
  }
});

const slots = computed<Slot[]>(() => {
  const product = selected.value;
  if (!product || !product.activities.length) return [];

  if (product.chooseOneActivity) {
    if (!chosenActivityId.value) return [];
    const activity = product.activities.find((a) => a.id === chosenActivityId.value);
    const sessions = sessionsByActivity.value[chosenActivityId.value] ?? [];
    return sessions.map((s) => ({
      key: s.id,
      day: fortalezaDay(s.startsAt),
      sessions: [{ activityId: s.activityId, activityTitle: activity?.title ?? "", startsAt: s.startsAt, spotsLeft: s.capacity - s.booked }],
      sessionIds: { [s.activityId]: s.id },
      minSpotsLeft: s.capacity - s.booked,
    }));
  }

  // Group each activity's own sessions by Fortaleza calendar day, then keep only the days
  // where every linked activity has at least one open session — that day is bookable even
  // if, say, Yoga is at 07:00 and HYROX is at 08:00. Picks the earliest session per
  // activity when more than one falls on the same day.
  const byDay = new Map<string, Map<string, RawSession>>();
  for (const activity of product.activities) {
    for (const session of sessionsByActivity.value[activity.id] ?? []) {
      const day = fortalezaDay(session.startsAt);
      const forDay = byDay.get(day) ?? new Map<string, RawSession>();
      const existing = forDay.get(activity.id);
      if (!existing || session.startsAt < existing.startsAt) forDay.set(activity.id, session);
      byDay.set(day, forDay);
    }
  }

  const result: Slot[] = [];
  for (const [day, byActivity] of byDay) {
    if (byActivity.size !== product.activities.length) continue;
    const sessions = product.activities.map((a) => {
      const s = byActivity.get(a.id)!;
      return { activityId: a.id, activityTitle: a.title, startsAt: s.startsAt, spotsLeft: s.capacity - s.booked };
    });
    result.push({
      key: day,
      day,
      sessions,
      sessionIds: Object.fromEntries(product.activities.map((a) => [a.id, byActivity.get(a.id)!.id])),
      minSpotsLeft: Math.min(...sessions.map((s) => s.spotsLeft)),
    });
  }
  result.sort((a, b) => a.day.localeCompare(b.day));
  return result;
});

const selectedSlot = computed(() => slots.value.find((s) => s.key === selectedSlotKey.value) ?? null);

function formatSlotDay(day: string) {
  return new Date(`${day}T00:00:00Z`).toLocaleDateString("pt-BR", { timeZone: "UTC", weekday: "long", day: "2-digit", month: "2-digit" });
}

function formatSlotTime(iso: string) {
  return new Date(iso).toLocaleTimeString("pt-BR", { timeZone: EVENT_TZ, hour: "2-digit", minute: "2-digit" });
}

// Only spell out each activity's own time when they actually differ (the common case is
// everything at the same hour) — otherwise a single time reads cleaner than a repeated list.
function formatSlotSummary(slot: Slot) {
  const times = new Set(slot.sessions.map((s) => formatSlotTime(s.startsAt)));
  if (times.size <= 1) return times.values().next().value ?? "";
  return slot.sessions.map((s) => `${s.activityTitle} ${formatSlotTime(s.startsAt)}`).join(" · ");
}

const needsSlot = computed(() => (selected.value?.activities.length ?? 0) > 0);
const needsActivityChoice = computed(() => selected.value?.chooseOneActivity ?? false);
const canCheckout = computed(
  () => !!selected.value && (!needsActivityChoice.value || !!chosenActivityId.value) && (!needsSlot.value || !!selectedSlot.value),
);

async function goToPayment() {
  if (!selected.value || !canCheckout.value) return;
  errorMessage.value = "";
  if (!localStorage.getItem("p5_student_token")) {
    router.push({ path: "/entrar", query: { redirect: "/comprar" } });
    return;
  }
  submitting.value = true;
  try {
    const res = await api.post<{ orderId: string; pixCopyPaste: string; pixQrImage?: string }>("/checkout", {
      productId: selected.value.id,
      sessionIds: selectedSlot.value?.sessionIds ?? {},
    });
    router.push({ path: `/comprar/${res.orderId}`, state: { pixCopyPaste: res.pixCopyPaste, pixQrImage: res.pixQrImage ?? "" } });
  } catch (err) {
    errorMessage.value = err instanceof ApiError ? err.message : "Não foi possível iniciar o pagamento.";
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <div class="min-h-screen bg-paper">
    <WellnessHeader />

    <main class="mx-auto max-w-6xl px-6 py-10">
      <button class="inline-flex items-center gap-2 text-xs font-semibold text-ink-soft hover:text-ink" @click="router.push('/')">
        <ArrowLeft :size="14" /> Voltar
      </button>

      <div class="mt-6 flex flex-col justify-between gap-4 md:flex-row md:items-end">
        <div>
          <p class="eyebrow mb-2">RESERVE SUA EXPERIÊNCIA</p>
          <h1 class="font-serif text-4xl font-bold leading-tight text-ink md:text-5xl">
            Escolha o seu<br />
            <em class="font-serif italic text-magenta">próximo encontro.</em>
          </h1>
        </div>
        <p class="max-w-xs text-xs text-ink-soft leading-relaxed">
          Defina uma experiência, escolha a turma e finalize em um ambiente de pagamento seguro.
        </p>
      </div>

      <div v-if="isLoading" class="mt-12 text-sm text-ink-soft">Carregando experiências…</div>

      <div v-else-if="!products.length" class="mt-12 rounded-[var(--radius-card)] border border-line bg-white p-10 text-center">
        <h2 class="font-serif text-xl font-semibold text-ink">Agenda em preparação</h2>
        <p class="mt-2 text-ink-soft">As atividades e combinações do P5 Wellness Club serão publicadas aqui em breve.</p>
      </div>

      <div v-else class="mt-10 grid gap-10 md:grid-cols-[1.5fr_1fr] md:items-start">
        <div class="space-y-8">
          <!-- STEP 1: Escolha sua experiência -->
          <div>
            <div class="mb-6 flex items-center gap-3">
              <span class="flex h-7 w-7 items-center justify-center rounded-full bg-magenta text-xs font-bold text-white">1</span>
              <h2 class="font-serif text-xl font-bold text-ink">Escolha sua experiência</h2>
            </div>

            <div class="space-y-4">
              <div
                v-for="p in products"
                :key="p.id"
                :class="[
                  'relative flex cursor-pointer items-start justify-between rounded-[var(--radius-card)] border p-6 transition-all',
                  selectedId === p.id ? 'border-magenta/80 bg-white ring-2 ring-magenta/20 shadow-md' : 'border-line/80 bg-white/70 hover:border-ink/40',
                ]"
                @click="selectProduct(p.id)"
              >
                <div class="flex items-start gap-4">
                  <div :class="['mt-1 flex h-5 w-5 shrink-0 items-center justify-center rounded-full border transition-colors', selectedId === p.id ? 'border-magenta bg-magenta' : 'border-line']">
                    <div v-if="selectedId === p.id" class="h-2 w-2 rounded-full bg-white"></div>
                  </div>

                  <div>
                    <div class="flex flex-wrap items-center gap-2">
                      <h3 class="font-sans text-base font-bold text-ink">{{ p.title }}</h3>
                      <span v-if="p.type === 'combo'" class="rounded-full bg-magenta/10 px-2.5 py-0.5 font-mono text-[9px] font-bold tracking-wider text-magenta uppercase">COMBO</span>
                    </div>
                    <p class="mt-1.5 text-xs leading-relaxed text-ink-soft">{{ p.description }}</p>
                    <div v-if="p.activities.length" class="mt-3 flex flex-wrap gap-2">
                      <span v-for="a in p.activities" :key="a.id" class="inline-flex items-center gap-1 rounded-md bg-warm/80 px-2.5 py-1 text-[10px] font-semibold text-ink-soft">✦ {{ a.title }}</span>
                    </div>
                    <div v-if="p.activities.some((a) => activityTip(a.slug))" class="mt-2 space-y-1">
                      <p v-for="a in p.activities.filter((a) => activityTip(a.slug))" :key="a.id + '-tip'" class="flex items-start gap-1.5 text-[11px] font-medium text-magenta">
                        <span aria-hidden="true">📌</span>
                        <span>{{ a.title }}: {{ activityTip(a.slug) }}</span>
                      </p>
                    </div>
                  </div>
                </div>

                <div class="ml-4 shrink-0 text-right">
                  <span class="font-sans text-lg font-bold text-ink">{{ formatBRL(p.priceCents) }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- STEP 2: Selecione sua data -->
          <div v-if="selectedId && needsSlot" class="border-t border-line/60 pt-4">
            <div class="mb-6 flex items-center gap-3">
              <span class="flex h-7 w-7 items-center justify-center rounded-full bg-magenta text-xs font-bold text-white">2</span>
              <h2 class="font-serif text-xl font-bold text-ink">
                {{ needsActivityChoice && !chosenActivityId ? "Escolha a atividade" : "Selecione sua data" }}
              </h2>
            </div>

            <!-- Sub-etapa: para produtos "Yoga ou HYROX", primeiro escolhe qual atividade -->
            <div v-if="needsActivityChoice" class="mb-6 flex flex-wrap gap-3">
              <button
                v-for="a in selected?.activities ?? []"
                :key="a.id"
                type="button"
                :class="[
                  'rounded-full border px-4 py-2 text-sm font-semibold transition-all',
                  chosenActivityId === a.id ? 'border-magenta bg-magenta text-white' : 'border-line/80 bg-white/70 text-ink hover:border-ink/40',
                ]"
                @click="chooseActivity(a.id)"
              >
                {{ a.title }}
              </button>
            </div>

            <p v-if="needsActivityChoice && !chosenActivityId" class="text-sm text-ink-soft">Escolha uma atividade para ver as datas disponíveis.</p>
            <template v-else>
            <p v-if="loadingSlots" class="text-sm text-ink-soft">Carregando turmas...</p>
            <p v-else-if="slotsError" class="text-sm font-medium text-red-600">Não foi possível carregar as datas. Tente novamente em instantes.</p>
            <p v-else-if="!slots.length" class="text-sm text-ink-soft">Nenhuma turma disponível no momento para essa experiência.</p>

            <div v-else class="space-y-3">
              <div
                v-for="s in slots"
                :key="s.key"
                :class="[
                  'flex cursor-pointer items-center justify-between rounded-xl border p-4 transition-all',
                  selectedSlotKey === s.key ? 'border-magenta bg-white shadow-sm ring-1 ring-magenta' : 'border-line/80 bg-white/70 hover:border-ink/40',
                ]"
                @click="selectedSlotKey = s.key"
              >
                <div class="flex items-center gap-3">
                  <div :class="['flex h-4 w-4 shrink-0 items-center justify-center rounded-full border', selectedSlotKey === s.key ? 'border-magenta bg-magenta' : 'border-line']">
                    <div v-if="selectedSlotKey === s.key" class="h-1.5 w-1.5 rounded-full bg-white"></div>
                  </div>
                  <div>
                    <p class="text-sm font-bold capitalize text-ink">{{ formatSlotDay(s.day) }} · {{ formatSlotSummary(s) }}</p>
                    <p class="text-xs text-ink-soft">{{ s.sessions.map((a) => a.activityTitle).join(" / ") }}</p>
                  </div>
                </div>
                <span class="text-xs font-medium text-ink-soft">{{ s.minSpotsLeft }} vaga{{ s.minSpotsLeft === 1 ? "" : "s" }} disponíve{{ s.minSpotsLeft === 1 ? "l" : "is" }}</span>
              </div>
            </div>
            </template>
          </div>
        </div>

        <!-- RESUMO -->
        <aside class="sticky top-24 rounded-[var(--radius-card)] bg-ink p-7 text-white shadow-xl">
          <p class="font-mono text-[10px] font-bold uppercase tracking-widest text-white/50">RESUMO</p>
          <h2 class="mt-1 font-serif text-2xl font-bold text-white">Sua experiência</h2>

          <div v-if="selected" class="mt-6 space-y-4">
            <div class="border-b border-white/10 pb-4">
              <p class="text-sm font-bold text-white">{{ selected.title }}</p>
              <p class="mt-1 text-xs text-white/70">{{ selected.description }}</p>
              <p v-if="selectedSlot" class="mt-2 text-xs font-semibold capitalize text-magenta">🗓 {{ formatSlotDay(selectedSlot.day) }} · {{ formatSlotSummary(selectedSlot) }}</p>
            </div>

            <div class="flex items-center justify-between pt-2">
              <span class="text-xs text-white/70">Total</span>
              <span class="font-sans text-2xl font-bold text-white">{{ formatBRL(selected.priceCents) }}</span>
            </div>

            <button :disabled="submitting || !canCheckout" class="button-magenta mt-4 w-full justify-center py-3.5 text-sm" @click="goToPayment">
              {{ submitting ? "Gerando cobrança..." : "Continuar para pagamento" }}
              <ArrowRight :size="16" />
            </button>
            <p v-if="needsSlot && !selectedSlot" class="text-center text-[11px] text-white/50">Escolha uma data para continuar.</p>

            <p class="flex items-start gap-2 pt-2 text-[11px] leading-relaxed text-white/60">
              <ShieldCheck :size="16" class="mt-0.5 shrink-0 text-magenta" />
              O pagamento é processado pelo Asaas. Sua vaga é confirmada conforme o status da cobrança.
            </p>

            <p v-if="errorMessage" class="text-xs font-medium text-red-400">{{ errorMessage }}</p>
          </div>

          <div v-else class="mt-6 py-6 text-center">
            <p class="text-xs leading-relaxed text-white/60">Selecione uma experiência para ver o resumo da compra.</p>
          </div>
        </aside>
      </div>
    </main>
  </div>
</template>
