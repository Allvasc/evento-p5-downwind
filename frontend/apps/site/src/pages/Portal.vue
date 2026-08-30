<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from "vue";
import { useRouter } from "vue-router";
import { ArrowRight, UserCircle, QrCode, CheckCircle2, Ticket, CalendarClock } from "lucide-vue-next";
import WellnessHeader from "@/components/WellnessHeader.vue";
import HelpFab from "@/components/HelpFab.vue";
import { useAuthStore } from "@/stores/auth";
import { api, ApiError } from "@/lib/api";

interface Ticket {
  ID: string;
  Status: string;
  Label: string;
  VendorName: string;
  OrderNumber: string;
  ValidFrom: string;
  ValidUntil: string;
  UsedAt: string;
  NoShowRescheduleUsedAt: string;
}

const router = useRouter();
const authStore = useAuthStore();
const tickets = ref<Ticket[]>([]);
const loadingTickets = ref(true);
const activeTicket = ref<Ticket | null>(null);
const qrImageUrl = ref<string | null>(null);

function formatDateTime(iso: string) {
  if (!iso) return "";
  return new Date(iso).toLocaleString("pt-BR", { day: "2-digit", month: "2-digit", year: "numeric", hour: "2-digit", minute: "2-digit" });
}

function formatDateOnly(iso: string) {
  if (!iso) return "";
  return new Date(`${iso}T12:00:00`).toLocaleDateString("pt-BR", { weekday: "short", day: "2-digit", month: "2-digit" });
}

// Se faltou (ficou "available" depois da própria data), tem até 60 dias corridos pra
// remarcar sozinho — checagem só pra decidir se mostra o botão; o backend valida de novo.
function todayInEventTZ() {
  return new Date().toLocaleDateString("en-CA", { timeZone: "America/Fortaleza" });
}
function isNoShowEligible(t: Ticket) {
  return t.Status === "available" && t.ValidUntil < todayInEventTZ() && !t.NoShowRescheduleUsedAt;
}

// ── Remarcar por falta ────────────────────────────────────────────────────
const noShowModalOpen = ref(false);
const noShowTicket = ref<Ticket | null>(null);
const noShowLoading = ref(false);
const noShowOptions = ref<{ eligible: boolean; reason: string; dates: string[] } | null>(null);
const noShowSelectedDate = ref("");
const noShowSubmitting = ref(false);
const noShowError = ref("");

async function openNoShowModal(t: Ticket) {
  noShowTicket.value = t;
  noShowModalOpen.value = true;
  noShowLoading.value = true;
  noShowOptions.value = null;
  noShowSelectedDate.value = "";
  noShowError.value = "";
  try {
    noShowOptions.value = await api.get(`/me/entitlements/${t.ID}/reschedule-no-show-options`);
  } catch (err) {
    noShowError.value = err instanceof ApiError ? err.message : "Não foi possível carregar as opções de remarcação.";
  } finally {
    noShowLoading.value = false;
  }
}

function closeNoShowModal() {
  noShowModalOpen.value = false;
  noShowTicket.value = null;
}

async function confirmNoShowReschedule() {
  if (!noShowTicket.value || !noShowSelectedDate.value) return;
  noShowError.value = "";
  noShowSubmitting.value = true;
  try {
    await api.post(`/me/entitlements/${noShowTicket.value.ID}/reschedule-no-show`, { newDate: noShowSelectedDate.value });
    closeNoShowModal();
    await refreshTickets();
  } catch (err) {
    noShowError.value = err instanceof ApiError ? err.message : "Não foi possível remarcar.";
  } finally {
    noShowSubmitting.value = false;
  }
}

// Polls quietly while the portal is open so a benefit validated at the door by staff
// flips to "Utilizado" here within seconds, without the student needing to refresh.
let pollTimer: number | null = null;
async function refreshTickets() {
  try {
    const res = await api.get<{ tickets: Ticket[] }>("/me/tickets");
    tickets.value = res.tickets;
    if (activeTicket.value) {
      const updated = res.tickets.find((t) => t.ID === activeTicket.value?.ID);
      if (updated) activeTicket.value = updated;
    }
  } catch {
    // transient network hiccup — next poll tries again
  }
}

async function openTicket(t: Ticket) {
  activeTicket.value = t;
  qrImageUrl.value = null;
  const res = await fetch(`/api/v1/me/tickets/${t.ID}/qrcode.png`, {
    headers: { Authorization: `Bearer ${authStore.token}` },
  });
  if (!res.ok) return;
  qrImageUrl.value = URL.createObjectURL(await res.blob());
}

function closeTicket() {
  activeTicket.value = null;
  if (qrImageUrl.value) URL.revokeObjectURL(qrImageUrl.value);
  qrImageUrl.value = null;
}

onMounted(async () => {
  if (!authStore.token) {
    router.replace("/entrar");
    return;
  }
  await authStore.fetchMe();
  if (!authStore.token) {
    router.replace("/entrar");
    return;
  }
  try {
    const res = await api.get<{ tickets: Ticket[] }>("/me/tickets");
    tickets.value = res.tickets;
  } finally {
    loadingTickets.value = false;
  }
  pollTimer = window.setInterval(refreshTickets, 8000);
});

onBeforeUnmount(() => {
  if (pollTimer) window.clearInterval(pollTimer);
});

</script>

<template>
  <div class="min-h-screen bg-paper">
    <WellnessHeader />
    <HelpFab />
    <main class="mx-auto max-w-5xl px-6 py-16">
      <div class="flex flex-wrap items-center justify-between gap-4">
        <div>
          <p class="eyebrow mb-2">Portal do aluno</p>
          <h1 class="font-serif text-3xl font-bold text-ink">Olá, {{ authStore.me?.fullName?.split(" ")[0] ?? "aluno" }}.</h1>
          <p class="mt-1 text-ink-soft">Seu próximo momento de bem-estar começa aqui.</p>
        </div>
        <div class="flex flex-wrap gap-3">
          <button class="rounded-full border border-line px-4 py-2 text-sm font-semibold text-ink" @click="router.push('/perfil')">
            <UserCircle :size="16" class="mr-2 inline" /> Meu perfil
          </button>
          <button class="rounded-full border border-line px-4 py-2 text-sm font-semibold text-ink" @click="router.push('/voucher')">
            <Ticket :size="16" class="mr-2 inline" /> Inserir voucher
          </button>
          <button class="button-magenta" @click="router.push('/comprar')">
            Comprar experiência <ArrowRight :size="16" />
          </button>
        </div>
      </div>

      <section class="mt-10">
        <h2 class="mb-4 font-serif text-lg font-semibold text-ink">Seus benefícios</h2>

        <p v-if="loadingTickets" class="text-ink-soft">Carregando...</p>

        <div v-else-if="!tickets.length" class="rounded-[var(--radius-card)] border border-line bg-white p-10 text-center text-ink-soft">
          Você ainda não tem uma experiência confirmada.
        </div>

        <div v-else class="grid gap-4 md:grid-cols-3">
          <div
            v-for="t in tickets"
            :key="t.ID"
            :class="[
              'flex flex-col items-start rounded-[var(--radius-card)] border bg-white p-5 text-left transition-colors',
              t.Status === 'used' ? 'border-line opacity-60' : 'border-line hover:border-magenta',
            ]"
          >
            <button type="button" class="flex w-full flex-col items-start text-left" @click="openTicket(t)">
              <span :class="['mb-3 inline-flex items-center gap-1 rounded-full px-3 py-1 text-xs font-semibold', t.Status === 'available' ? 'bg-[#e5f5e8] text-[#237438]' : 'bg-warm text-ink-soft']">
                <CheckCircle2 v-if="t.Status !== 'available'" :size="12" />
                {{ t.Status === "available" ? "Disponível" : "Utilizado" }}
              </span>
              <h3 class="font-serif text-lg font-semibold text-ink">{{ t.Label }}</h3>
              <p class="mt-1 text-xs font-medium text-ink-soft">Validado por {{ t.VendorName }}</p>
              <p class="mt-1 text-xs text-ink-soft">Pedido {{ t.OrderNumber }}</p>
              <p v-if="t.UsedAt" class="mt-1 text-xs text-ink-soft">Utilizado em {{ formatDateTime(t.UsedAt) }}</p>
              <p class="mt-3 inline-flex items-center gap-1 text-xs font-medium text-magenta">
                <QrCode :size="14" /> Ver QR Code
              </p>
            </button>
            <button
              v-if="isNoShowEligible(t)"
              type="button"
              class="mt-3 inline-flex items-center gap-1.5 rounded-full border border-magenta/30 bg-magenta/5 px-3 py-1.5 text-xs font-semibold text-magenta"
              @click="openNoShowModal(t)"
            >
              <CalendarClock :size="13" /> Remarcar (não compareci)
            </button>
          </div>
        </div>
      </section>

      <div v-if="noShowModalOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-ink/40 p-6" @click.self="closeNoShowModal">
        <div class="w-full max-w-sm rounded-[var(--radius-card)] bg-white p-6">
          <h3 class="font-serif text-lg font-semibold text-ink">Remarcar por falta</h3>
          <p class="mt-1 text-sm text-ink-soft">{{ noShowTicket?.Label }}</p>

          <p v-if="noShowLoading" class="mt-4 text-sm text-ink-soft">Carregando datas disponíveis...</p>

          <template v-else-if="noShowOptions">
            <p v-if="!noShowOptions.eligible" class="mt-4 text-sm text-ink-soft">{{ noShowOptions.reason }}</p>
            <template v-else>
              <p class="mt-4 text-xs font-semibold uppercase tracking-wider text-ink-soft">Escolha a nova data</p>
              <div class="mt-2 flex flex-wrap gap-2">
                <button
                  v-for="d in noShowOptions.dates"
                  :key="d"
                  type="button"
                  :class="['rounded-full border px-3 py-1.5 text-xs font-semibold capitalize', noShowSelectedDate === d ? 'border-magenta bg-magenta text-white' : 'border-line text-ink hover:border-magenta']"
                  @click="noShowSelectedDate = d"
                >
                  {{ formatDateOnly(d) }}
                </button>
              </div>
            </template>
          </template>

          <p v-if="noShowError" class="mt-3 text-sm text-red-600">{{ noShowError }}</p>

          <div class="mt-5 flex gap-3">
            <button type="button" class="flex-1 rounded-full border border-line px-4 py-2 text-sm font-semibold text-ink" @click="closeNoShowModal">Cancelar</button>
            <button
              v-if="noShowOptions?.eligible"
              type="button"
              class="button-magenta flex-1 justify-center py-2 text-sm disabled:opacity-60"
              :disabled="!noShowSelectedDate || noShowSubmitting"
              @click="confirmNoShowReschedule"
            >
              {{ noShowSubmitting ? "Remarcando..." : "Confirmar" }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="activeTicket" class="fixed inset-0 z-50 flex items-center justify-center bg-ink/40 p-6" @click.self="closeTicket">
        <div class="w-full max-w-sm rounded-[var(--radius-card)] bg-white p-6 text-center">
          <span :class="['mb-3 inline-flex items-center gap-1 rounded-full px-3 py-1 text-xs font-semibold', activeTicket.Status === 'available' ? 'bg-[#e5f5e8] text-[#237438]' : 'bg-warm text-ink-soft']">
            {{ activeTicket.Status === "available" ? "Disponível" : "Utilizado" }}
          </span>
          <h3 class="font-serif text-lg font-semibold text-ink">{{ activeTicket.Label }}</h3>
          <p class="mt-1 text-xs text-ink-soft">Validado pela equipe {{ activeTicket.VendorName }} na entrada.</p>
          <p v-if="activeTicket.UsedAt" class="mt-1 text-xs text-ink-soft">Utilizado em {{ formatDateTime(activeTicket.UsedAt) }}</p>
          <div class="mx-auto mt-4 flex h-56 w-56 items-center justify-center">
            <img v-if="qrImageUrl" :src="qrImageUrl" alt="QR Code" class="h-56 w-56" />
            <p v-else class="text-sm text-ink-soft">Gerando...</p>
          </div>
          <button class="button-magenta mt-4 w-full justify-center" @click="closeTicket">Fechar</button>
        </div>
      </div>
    </main>
  </div>
</template>
