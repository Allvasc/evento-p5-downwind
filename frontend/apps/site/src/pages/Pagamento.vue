<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Copy, CheckCircle2 } from "lucide-vue-next";
import { api } from "@/lib/api";
import WellnessHeader from "@/components/WellnessHeader.vue";

const route = useRoute();
const router = useRouter();
const orderId = route.params.orderId as string;

const status = ref("pending");
const copied = ref(false);
let poller: ReturnType<typeof setInterval> | undefined;

const pixCopyPaste = (history.state?.pixCopyPaste as string) ?? "";
const pixQrImage = (history.state?.pixQrImage as string) ?? "";

async function checkStatus() {
  try {
    const res = await api.get<{ status: string }>(`/checkout/orders/${orderId}/status`);
    status.value = res.status;
    if (res.status === "paid") {
      clearInterval(poller);
      setTimeout(() => router.push("/portal"), 1500);
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
});
onUnmounted(() => clearInterval(poller));
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
      <template v-else>
        <p class="eyebrow mb-3">Pagamento via Pix</p>
        <h1 class="font-serif text-2xl font-bold text-ink">Escaneie ou copie o código</h1>
        <p class="mt-2 text-sm text-ink-soft">O pagamento é processado pelo Asaas. Sua vaga é confirmada conforme o status da cobrança.</p>

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
