<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { ArrowLeft, UserCircle, Check, Mail } from "lucide-vue-next";
import WellnessHeader from "@/components/WellnessHeader.vue";
import { useAuthStore } from "@/stores/auth";
import { api, ApiError } from "@/lib/api";

interface Order {
  ID: string;
  OrderNumber: string;
  Status: string;
  TotalCents: number;
  ProductTitle: string;
  CreatedAt: string;
}

const router = useRouter();
const authStore = useAuthStore();

const orders = ref<Order[]>([]);
const loadingOrders = ref(true);

const cpf = ref("");
const cpfSaving = ref(false);
const cpfError = ref("");
const cpfSuccess = ref("");

const resendingOrderId = ref("");
const resendMessage = ref<Record<string, string>>({});

function statusLabel(status: string) {
  return { paid: "Pago", pending: "Aguardando pagamento", refunded: "Estornado", failed: "Falhou", expired: "Expirado", cancelled: "Cancelado" }[status] ?? status;
}

function formatBRL(cents: number) {
  return new Intl.NumberFormat("pt-BR", { style: "currency", currency: "BRL" }).format(cents / 100);
}

async function saveCPF() {
  cpfError.value = "";
  cpfSuccess.value = "";
  cpfSaving.value = true;
  try {
    const res = await api.put<{ cpfLast4: string }>("/me/cpf", { cpf: cpf.value });
    if (authStore.me) authStore.me.cpfLast4 = res.cpfLast4;
    cpf.value = "";
    cpfSuccess.value = "CPF atualizado com sucesso.";
  } catch (err) {
    cpfError.value = err instanceof ApiError ? err.message : "Não foi possível salvar o CPF.";
  } finally {
    cpfSaving.value = false;
  }
}

async function resendEmail(orderId: string) {
  resendingOrderId.value = orderId;
  resendMessage.value = { ...resendMessage.value, [orderId]: "" };
  try {
    await api.post(`/me/orders/${orderId}/resend-email`);
    resendMessage.value = { ...resendMessage.value, [orderId]: "E-mail reenviado — confira sua caixa de entrada (e o spam)." };
  } catch (err) {
    resendMessage.value = { ...resendMessage.value, [orderId]: err instanceof ApiError ? err.message : "Não foi possível reenviar." };
  } finally {
    resendingOrderId.value = "";
  }
}

onMounted(async () => {
  if (!authStore.token) {
    router.replace("/entrar");
    return;
  }
  if (!authStore.me) await authStore.fetchMe();
  try {
    const res = await api.get<{ orders: Order[] }>("/me/orders");
    orders.value = res.orders;
  } finally {
    loadingOrders.value = false;
  }
});
</script>

<template>
  <div class="min-h-screen bg-paper">
    <WellnessHeader />
    <main class="mx-auto max-w-4xl px-6 py-16">
      <button class="mb-8 inline-flex items-center gap-2 text-sm text-ink-soft hover:text-ink" @click="router.push('/portal')">
        <ArrowLeft :size="16" /> Voltar ao portal
      </button>

      <section class="flex items-center gap-4">
        <div class="flex h-16 w-16 items-center justify-center rounded-full bg-warm text-ink-soft">
          <UserCircle :size="36" />
        </div>
        <div>
          <p class="eyebrow mb-1">Perfil do aluno</p>
          <h1 class="font-serif text-2xl font-bold text-ink">{{ authStore.me?.fullName ?? "Aluno P5" }}</h1>
          <p v-if="authStore.me" class="text-sm text-ink-soft">{{ authStore.me.email }} · {{ authStore.me.phone || "Telefone não informado" }}</p>
        </div>
      </section>

      <section class="mt-10 rounded-[var(--radius-card)] border border-line bg-white p-8">
        <h2 class="mb-1 font-serif text-lg font-semibold text-ink">CPF</h2>
        <p class="mb-4 text-sm text-ink-soft">
          Necessário para pagar com Pix.
          <span v-if="authStore.me?.cpfLast4"> Cadastrado terminando em <strong>{{ authStore.me.cpfLast4 }}</strong>.</span>
          <span v-else> Você ainda não cadastrou um CPF.</span>
        </p>
        <form class="flex flex-wrap items-end gap-3" @submit.prevent="saveCPF">
          <div class="flex-1 min-w-[200px]">
            <label for="cpf" class="mb-1.5 block text-xs font-bold uppercase tracking-wider text-ink">{{ authStore.me?.cpfLast4 ? "Novo CPF" : "CPF" }}</label>
            <input
              id="cpf"
              v-model="cpf"
              placeholder="000.000.000-00"
              class="w-full rounded-full border border-line bg-paper px-4 py-2.5 text-sm text-ink outline-none focus:border-magenta"
            />
          </div>
          <button type="submit" class="button-magenta" :disabled="cpfSaving || !cpf">
            {{ cpfSaving ? "Salvando..." : "Salvar" }}
          </button>
        </form>
        <p v-if="cpfError" class="mt-2 text-xs font-medium text-red-600">{{ cpfError }}</p>
        <p v-if="cpfSuccess" class="mt-2 inline-flex items-center gap-1 text-xs font-medium text-[#237438]"><Check :size="14" /> {{ cpfSuccess }}</p>
      </section>

      <section class="mt-8 rounded-[var(--radius-card)] border border-line bg-white p-8">
        <h2 class="mb-4 font-serif text-lg font-semibold text-ink">Meus pedidos</h2>

        <p v-if="loadingOrders" class="text-sm text-ink-soft">Carregando...</p>
        <p v-else-if="!orders.length" class="text-sm text-ink-soft">Você ainda não fez nenhum pedido.</p>

        <ul v-else class="divide-y divide-line">
          <li v-for="o in orders" :key="o.ID" class="flex flex-wrap items-center justify-between gap-3 py-4">
            <div>
              <p class="font-semibold text-ink">{{ o.ProductTitle || "Pedido " + o.OrderNumber }}</p>
              <p class="text-xs text-ink-soft">{{ o.OrderNumber }} · {{ statusLabel(o.Status) }} · {{ formatBRL(o.TotalCents) }}</p>
              <p v-if="resendMessage[o.ID]" class="mt-1 text-xs font-medium text-ink-soft">{{ resendMessage[o.ID] }}</p>
            </div>
            <button
              v-if="o.Status === 'paid'"
              type="button"
              class="inline-flex items-center gap-1.5 rounded-full border border-line px-3 py-1.5 text-xs font-semibold text-ink hover:border-magenta disabled:opacity-50"
              :disabled="resendingOrderId === o.ID"
              @click="resendEmail(o.ID)"
            >
              <Mail :size="13" /> {{ resendingOrderId === o.ID ? "Enviando..." : "Reenviar e-mail" }}
            </button>
          </li>
        </ul>
      </section>
    </main>
  </div>
</template>
