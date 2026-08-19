<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from "vue";
import { useRouter } from "vue-router";
import { ArrowRight, UserCircle, QrCode, CheckCircle2 } from "lucide-vue-next";
import WellnessHeader from "@/components/WellnessHeader.vue";
import { useAuthStore } from "@/stores/auth";
import { api } from "@/lib/api";

interface Ticket {
  ID: string;
  Status: string;
  Label: string;
  VendorName: string;
  OrderNumber: string;
  ValidFrom: string;
  ValidUntil: string;
  UsedAt: string;
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
    <main class="mx-auto max-w-5xl px-6 py-16">
      <div class="flex flex-wrap items-center justify-between gap-4">
        <div>
          <p class="eyebrow mb-2">Portal do aluno</p>
          <h1 class="font-serif text-3xl font-bold text-ink">Olá, {{ authStore.me?.fullName?.split(" ")[0] ?? "aluno" }}.</h1>
          <p class="mt-1 text-ink-soft">Seu próximo momento de bem-estar começa aqui.</p>
        </div>
        <div class="flex gap-3">
          <button class="rounded-full border border-line px-4 py-2 text-sm font-semibold text-ink" @click="router.push('/perfil')">
            <UserCircle :size="16" class="mr-2 inline" /> Meu perfil
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
          <button
            v-for="t in tickets"
            :key="t.ID"
            type="button"
            :class="[
              'flex flex-col items-start rounded-[var(--radius-card)] border bg-white p-5 text-left transition-colors',
              t.Status === 'used' ? 'border-line opacity-60' : 'border-line hover:border-magenta',
            ]"
            @click="openTicket(t)"
          >
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
        </div>
      </section>

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
