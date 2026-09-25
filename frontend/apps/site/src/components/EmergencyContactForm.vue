<script setup lang="ts">
import { ref, watch } from "vue";
import { Check } from "lucide-vue-next";
import { api, ApiError } from "@/lib/api";
import { useAuthStore } from "@/stores/auth";

// Contato de emergência do cliente: obrigatório no cadastro novo, e pedido na próxima
// compra/resgate de voucher para contas criadas antes dele existir. Salva direto no
// perfil (PUT /me/emergency-contact) e atualiza o authStore.
const props = withDefaults(defineProps<{ saveLabel?: string }>(), { saveLabel: "Salvar contato" });
const emit = defineEmits<{ saved: [] }>();

const authStore = useAuthStore();
const name = ref(authStore.me?.emergencyContactName ?? "");
const phone = ref(authStore.me?.emergencyContactPhone ?? "");
const saving = ref(false);
const error = ref("");
const success = ref(false);

watch(
  () => authStore.me,
  (me) => {
    if (me && !name.value && !phone.value) {
      name.value = me.emergencyContactName ?? "";
      phone.value = me.emergencyContactPhone ?? "";
    }
  },
);

async function save() {
  error.value = "";
  success.value = false;
  saving.value = true;
  try {
    const res = await api.put<{ emergencyContactName: string; emergencyContactPhone: string }>("/me/emergency-contact", {
      name: name.value,
      phone: phone.value,
    });
    if (authStore.me) {
      authStore.me.emergencyContactName = res.emergencyContactName;
      authStore.me.emergencyContactPhone = res.emergencyContactPhone;
    }
    success.value = true;
    emit("saved");
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : "Não foi possível salvar o contato de emergência.";
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <form class="space-y-3" @submit.prevent="save">
    <div class="grid gap-3 sm:grid-cols-2">
      <div>
        <label for="emergency-name" class="mb-1.5 block text-xs font-bold uppercase tracking-wider text-ink">Nome do contato</label>
        <input
          id="emergency-name"
          v-model="name"
          required
          placeholder="Quem avisar em uma emergência"
          class="w-full rounded-xl border border-line bg-white px-4 py-3 text-sm text-ink placeholder:text-ink-soft/50 focus:border-magenta focus:outline-none focus:ring-1 focus:ring-magenta"
        />
      </div>
      <div>
        <label for="emergency-phone" class="mb-1.5 block text-xs font-bold uppercase tracking-wider text-ink">Telefone do contato</label>
        <input
          id="emergency-phone"
          v-model="phone"
          required
          type="tel"
          placeholder="(00) 00000-0000"
          class="w-full rounded-xl border border-line bg-white px-4 py-3 text-sm text-ink placeholder:text-ink-soft/50 focus:border-magenta focus:outline-none focus:ring-1 focus:ring-magenta"
        />
      </div>
    </div>
    <div class="flex flex-wrap items-center gap-3">
      <button type="submit" class="button-magenta" :disabled="saving || !name.trim() || !phone.trim()">
        {{ saving ? "Salvando..." : props.saveLabel }}
      </button>
      <p v-if="success" class="inline-flex items-center gap-1 text-xs font-medium text-[#237438]"><Check :size="14" /> Contato de emergência salvo.</p>
    </div>
    <p v-if="error" class="text-xs font-medium text-red-600">{{ error }}</p>
  </form>
</template>
