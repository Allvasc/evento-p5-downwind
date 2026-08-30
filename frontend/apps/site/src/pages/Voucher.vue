<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useRouter, useRoute } from "vue-router";
import { useQuery } from "@tanstack/vue-query";
import { ArrowLeft, ArrowRight, Ticket, CheckCircle2 } from "lucide-vue-next";
import { type Product } from "@p5wellness/shared";
import { api, ApiError } from "@/lib/api";
import WellnessHeader from "@/components/WellnessHeader.vue";
import { useAuthStore } from "@/stores/auth";
import { useProductSlots, formatSlotDay, formatSlotSummary } from "@/composables/useProductSlots";

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();

onMounted(async () => {
  if (!authStore.token) {
    router.replace({ path: "/entrar", query: { redirect: "/voucher" } });
    return;
  }
  await authStore.fetchMe();
  if (!authStore.token) {
    router.replace({ path: "/entrar", query: { redirect: "/voucher" } });
    return;
  }
  const fromQuery = typeof route.query.code === "string" ? route.query.code : "";
  if (fromQuery) {
    code.value = fromQuery;
    checkCode();
  }
});

// ── Etapa 0: código do voucher ────────────────────────────────────────────
const code = ref("");
const codeChecked = ref(false);
const voucherInfo = ref<{ name: string; companyName: string } | null>(null);
const checkingCode = ref(false);
const codeError = ref("");

async function checkCode() {
  const trimmed = code.value.trim();
  if (!trimmed) return;
  codeError.value = "";
  checkingCode.value = true;
  try {
    voucherInfo.value = await api.get<{ name: string; companyName: string }>(`/me/vouchers/${encodeURIComponent(trimmed.toUpperCase())}`);
    codeChecked.value = true;
  } catch (err) {
    codeError.value = err instanceof ApiError ? err.message : "Não foi possível verificar o voucher.";
  } finally {
    checkingCode.value = false;
  }
}

function changeCode() {
  codeChecked.value = false;
  voucherInfo.value = null;
  selectedId.value = null;
}

// ── Etapa 1/2: mesma escolha de experiência e data da compra normal, mas o voucher só
// vale pra uma aula (Yoga OU HYROX, à escolha do cliente) + café da manhã — não o combo
// das duas aulas, nem uma aula avulsa sem café.
const { data, isLoading } = useQuery({
  queryKey: ["public-products"],
  queryFn: () => api.get<{ products: Product[] }>("/public/products"),
});
const products = computed(() => (data.value?.products ?? []).filter((p) => p.chooseOneActivity && p.includesBreakfast));

const selectedId = ref<string | null>(null);
const selected = computed(() => products.value.find((p) => p.id === selectedId.value) ?? null);
const { chosenActivityId, selectedSlotKey, loadingSlots, slotsError, slots, selectedSlot, chooseActivity } = useProductSlots(selected);

function selectProduct(id: string) {
  selectedId.value = id;
  selectedSlotKey.value = null;
  chosenActivityId.value = null;
}

const needsSlot = computed(() => (selected.value?.activities.length ?? 0) > 0);
const needsActivityChoice = computed(() => selected.value?.chooseOneActivity ?? false);
const canRedeem = computed(
  () => !!selected.value && (!needsActivityChoice.value || !!chosenActivityId.value) && (!needsSlot.value || !!selectedSlot.value),
);

const submitting = ref(false);
const errorMessage = ref("");
const redeemed = ref(false);
const redeemedOrderNumber = ref("");

async function redeem() {
  if (!selected.value || !canRedeem.value) return;
  errorMessage.value = "";
  submitting.value = true;
  try {
    const res = await api.post<{ orderId: string; orderNumber: string }>("/me/vouchers/redeem", {
      code: code.value.trim().toUpperCase(),
      productId: selected.value.id,
      sessionIds: selectedSlot.value?.sessionIds ?? {},
    });
    redeemedOrderNumber.value = res.orderNumber;
    redeemed.value = true;
  } catch (err) {
    errorMessage.value = err instanceof ApiError ? err.message : "Não foi possível resgatar o voucher.";
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <div class="min-h-screen bg-paper">
    <WellnessHeader />

    <main class="mx-auto max-w-6xl px-6 py-10">
      <button class="inline-flex items-center gap-2 text-xs font-semibold text-ink-soft hover:text-ink" @click="router.push('/portal')">
        <ArrowLeft :size="14" /> Voltar ao portal
      </button>

      <div class="mt-6">
        <p class="eyebrow mb-2">RESGATE SEU VOUCHER</p>
        <h1 class="font-serif text-4xl font-bold leading-tight text-ink md:text-5xl">
          Sua cortesia <em class="font-serif italic text-magenta">te espera.</em>
        </h1>
      </div>

      <!-- Sucesso -->
      <div v-if="redeemed" class="mt-10 max-w-lg rounded-[var(--radius-card)] border border-line bg-white p-10 text-center">
        <CheckCircle2 :size="36" class="mx-auto text-magenta" />
        <h2 class="mt-4 font-serif text-2xl font-bold text-ink">Voucher resgatado!</h2>
        <p class="mt-2 text-sm text-ink-soft">
          Pedido <strong>{{ redeemedOrderNumber }}</strong> confirmado. O QR Code chega por e-mail e também já está disponível no seu portal.
        </p>
        <button class="button-magenta mt-6" @click="router.push('/portal')">Ir para o portal</button>
      </div>

      <template v-else>
        <!-- STEP 0: código -->
        <div class="mt-10 max-w-lg">
          <div class="mb-4 flex items-center gap-3">
            <span class="flex h-7 w-7 items-center justify-center rounded-full bg-magenta text-xs font-bold text-white">1</span>
            <h2 class="font-serif text-xl font-bold text-ink">Código do voucher</h2>
          </div>

          <div v-if="!codeChecked" class="flex gap-3">
            <input
              v-model="code"
              type="text"
              placeholder="Ex: P5-X7K2M9"
              class="flex-1 rounded-lg border border-line px-4 py-2.5 text-sm uppercase tracking-wider"
              @keyup.enter="checkCode"
            />
            <button class="button-magenta px-5 py-2.5 text-sm" :disabled="checkingCode || !code.trim()" @click="checkCode">
              <Ticket :size="16" /> {{ checkingCode ? "Verificando..." : "Validar" }}
            </button>
          </div>
          <div v-else class="flex items-center justify-between rounded-xl border border-line/80 bg-white p-4">
            <div>
              <p class="text-sm font-bold text-ink">{{ voucherInfo?.name }}</p>
              <p class="text-xs text-ink-soft">Cortesia — {{ voucherInfo?.companyName }}</p>
            </div>
            <button class="text-xs font-semibold text-ink-soft hover:text-ink" @click="changeCode">Trocar código</button>
          </div>
          <p v-if="codeError" class="mt-2 text-xs font-medium text-red-600">{{ codeError }}</p>
        </div>

        <div v-if="codeChecked" class="mt-10 grid gap-10 md:grid-cols-[1.5fr_1fr] md:items-start">
          <div class="space-y-8">
            <!-- STEP 1: experiência -->
            <div>
              <div class="mb-6 flex items-center gap-3">
                <span class="flex h-7 w-7 items-center justify-center rounded-full bg-magenta text-xs font-bold text-white">2</span>
                <h2 class="font-serif text-xl font-bold text-ink">Escolha sua experiência</h2>
              </div>

              <div v-if="isLoading" class="text-sm text-ink-soft">Carregando experiências…</div>
              <div v-else-if="!products.length" class="rounded-[var(--radius-card)] border border-line bg-white p-10 text-center text-ink-soft">
                Nenhuma experiência disponível pra voucher no momento.
              </div>
              <div v-else class="space-y-4">
                <div
                  v-for="p in products"
                  :key="p.id"
                  :class="[
                    'relative flex cursor-pointer items-start gap-4 rounded-[var(--radius-card)] border p-6 transition-all',
                    selectedId === p.id ? 'border-magenta/80 bg-white ring-2 ring-magenta/20 shadow-md' : 'border-line/80 bg-white/70 hover:border-ink/40',
                  ]"
                  @click="selectProduct(p.id)"
                >
                  <div :class="['mt-1 flex h-5 w-5 shrink-0 items-center justify-center rounded-full border transition-colors', selectedId === p.id ? 'border-magenta bg-magenta' : 'border-line']">
                    <div v-if="selectedId === p.id" class="h-2 w-2 rounded-full bg-white"></div>
                  </div>
                  <div>
                    <h3 class="font-sans text-base font-bold text-ink">{{ p.title }}</h3>
                    <p class="mt-1.5 text-xs leading-relaxed text-ink-soft">{{ p.description }}</p>
                    <div v-if="p.activities.length" class="mt-3 flex flex-wrap gap-2">
                      <span v-for="a in p.activities" :key="a.id" class="inline-flex items-center gap-1 rounded-md bg-warm/80 px-2.5 py-1 text-[10px] font-semibold text-ink-soft">✦ {{ a.title }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- STEP 2: data -->
            <div v-if="selectedId && needsSlot" class="border-t border-line/60 pt-4">
              <div class="mb-6 flex items-center gap-3">
                <span class="flex h-7 w-7 items-center justify-center rounded-full bg-magenta text-xs font-bold text-white">3</span>
                <h2 class="font-serif text-xl font-bold text-ink">
                  {{ needsActivityChoice && !chosenActivityId ? "Escolha a atividade" : "Selecione sua data" }}
                </h2>
              </div>

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
            <h2 class="mt-1 font-serif text-2xl font-bold text-white">Sua cortesia</h2>

            <div v-if="selected" class="mt-6 space-y-4">
              <div class="border-b border-white/10 pb-4">
                <p class="text-sm font-bold text-white">{{ selected.title }}</p>
                <p class="mt-1 text-xs text-white/70">Cortesia — {{ voucherInfo?.companyName }}</p>
                <p v-if="selectedSlot" class="mt-2 text-xs font-semibold capitalize text-magenta">🗓 {{ formatSlotDay(selectedSlot.day) }} · {{ formatSlotSummary(selectedSlot) }}</p>
              </div>

              <div class="flex items-center justify-between pt-2">
                <span class="text-xs text-white/70">Valor</span>
                <span class="font-sans text-2xl font-bold text-white">Cortesia</span>
              </div>

              <button :disabled="submitting || !canRedeem" class="button-magenta mt-4 w-full justify-center py-3.5 text-sm" @click="redeem">
                {{ submitting ? "Resgatando..." : "Resgatar voucher" }}
                <ArrowRight :size="16" />
              </button>
              <p v-if="needsSlot && !selectedSlot" class="text-center text-[11px] text-white/50">Escolha uma data para continuar.</p>

              <p v-if="errorMessage" class="text-xs font-medium text-red-400">{{ errorMessage }}</p>
            </div>

            <div v-else class="mt-6 py-6 text-center">
              <p class="text-xs leading-relaxed text-white/60">Selecione uma experiência para ver o resumo.</p>
            </div>
          </aside>
        </div>
      </template>
    </main>
  </div>
</template>
