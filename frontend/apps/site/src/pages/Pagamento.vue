<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Copy, CheckCircle2, Clock, TimerOff } from "lucide-vue-next";
import { api } from "@/lib/api";
import WellnessHeader from "@/components/WellnessHeader.vue";

const route = useRoute();
const router = useRouter();
const orderId = route.params.orderId as string;

const status = ref("pending");
const copied = ref(false);
let poller: ReturnType<typeof setInterval> | undefined;
let ticker: ReturnType<typeof setInterval> | undefined;

// O Pix vale 10 minutos (postgres.PendingOrderTTL no backend); depois disso a cobrança é
// cancelada na Asaas e a vaga volta a ficar disponível. O servidor informa quantos
// segundos faltam e a contagem segue localmente entre uma consulta e outra.
const secondsLeft = ref<number | null>(null);
const expired = computed(() => status.value === "expired" || (status.value === "pending" && secondsLeft.value === 0));
const countdown = computed(() => {
  const s = secondsLeft.value ?? 0;
  return `${String(Math.floor(s / 60)).padStart(2, "0")}:${String(s % 60).padStart(2, "0")}`;
});

const pixCopyPaste = (history.state?.pixCopyPaste as string) ?? "";
const pixQrImage = (history.state?.pixQrImage as string) ?? "";

async function checkStatus() {
  try {
    const res = await api.get<{ status: string; secondsLeft: number }>(`/checkout/orders/${orderId}/status`);
    status.value = res.status;
    secondsLeft.value = res.secondsLeft;
    if (res.status === "paid") {
      clearInterval(poller);
      clearInterval(ticker);
      setTimeout(() => router.push("/portal"), 1500);
    } else if (res.status !== "pending") {
      // expirado/falhou: nada mais a esperar
      clearInterval(poller);
      clearInterval(ticker);
    }
  } catch {
    // keep polling silently
  }
}

async function copyCode() {
  await navigator.clipboard.writeText(pixCopyPaste);
  copied.value = true;
  setTimeout(() => (copied.value = false), 2000);
}

onMounted(() => {
  checkStatus();
  poller = setInterval(checkStatus, 3000);
  ticker = setInterval(() => {
    if (secondsLeft.value !== null && secondsLeft.value > 0) secondsLeft.value -= 1;
  }, 1000);
});
onUnmounted(() => {
  clearInterval(poller);
  clearInterval(ticker);
});
</script>

<template>
  <div class="min-h-screen bg-paper">
    <WellnessHeader />
    <main class="mx-auto max-w-lg px-6 py-16 text-center">
      <template v-if="status === 'paid'">
        <CheckCircle2 :size="48" class="mx-auto text-green-600" />
        <h1 class="mt-4 font-serif text-2xl font-bold text-ink">Pagamento confirmado!</h1>
        <p class="mt-2 text-ink-soft">Redirecionando para o seu portal...</p>
      </template>
      <template v-else-if="expired || status !== 'pending'">
        <TimerOff :size="48" class="mx-auto text-magenta" />
        <h1 class="mt-4 font-serif text-2xl font-bold text-ink">{{ expired ? "Pix expirado" : "Pagamento não concluído" }}</h1>
        <p class="mt-2 text-sm text-ink-soft">
          {{ expired
            ? "O código Pix vale por 10 minutos e não foi pago a tempo. A vaga foi liberada — se ainda houver vagas, gere um novo Pix para garantir a sua."
            : "Esta cobrança não está mais disponível. Gere um novo Pix para garantir sua vaga." }}
        </p>
        <p class="mt-2 text-xs text-ink-soft">Não pague o código antigo: ele não vale mais.</p>
        <button class="button-magenta mt-6" @click="router.push('/comprar')">Gerar novo Pix</button>
      </template>
      <template v-else>
        <p class="eyebrow mb-3">Pagamento via Pix</p>
        <h1 class="font-serif text-2xl font-bold text-ink">Escaneie ou copie o código</h1>
        <p class="mt-2 text-sm text-ink-soft">O pagamento é processado pelo Asaas. Sua vaga é confirmada conforme o status da cobrança.</p>

        <div v-if="secondsLeft !== null" class="mx-auto mt-5 w-fit rounded-full border border-magenta/30 bg-magenta/5 px-4 py-2 text-sm font-semibold text-magenta">
          <span class="inline-flex items-center gap-2">
            <Clock :size="16" /> Pague em até <span class="font-mono tabular-nums">{{ countdown }}</span>
          </span>
        </div>
        <p class="mt-2 text-xs text-ink-soft">Depois de 10 minutos o Pix expira e a vaga é liberada para outra pessoa.</p>

        <div v-if="pixQrImage" class="mx-auto mt-6 flex w-fit flex-col items-center gap-3 rounded-[var(--radius-card)] border border-line bg-white p-5">
          <img :src="`data:image/png;base64,${pixQrImage}`" alt="QR Code Pix" class="h-56 w-56" />
          <p class="text-xs text-ink-soft">Aponte a câmera do seu banco para o código</p>
        </div>

        <div class="mt-6 break-all rounded-[var(--radius-card)] border border-line bg-white p-4 text-left font-mono text-xs text-ink-soft">
          {{ pixCopyPaste || "Código indisponível — volte e tente novamente." }}
        </div>

        <button class="button-magenta mt-4" @click="copyCode">
          <Copy :size="16" /> {{ copied ? "Copiado!" : "Copiar código" }}
        </button>

        <p class="mt-8 inline-flex items-center gap-2 text-sm text-ink-soft">
          <span class="h-2 w-2 animate-pulse rounded-full bg-magenta"></span>
          Aguardando confirmação do pagamento...
        </p>
      </template>
    </main>
  </div>
</template>
