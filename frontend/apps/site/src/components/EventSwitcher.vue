<script setup lang="ts">
import { computed } from "vue";
import { CalendarRange } from "lucide-vue-next";
import { useTeamEventStore } from "@/stores/teamEvent";

// Seletor de evento do Admin (fundo escuro) e do check-in (fundo claro). Trocar aqui troca
// toda a visão do sistema — quem usa a tela decide o que fazer no evento "changed".
const props = withDefaults(defineProps<{ variant?: "dark" | "light" }>(), { variant: "dark" });
const emit = defineEmits<{ changed: [id: string] }>();

const store = useTeamEventStore();

const model = computed({
  get: () => store.currentId ?? "",
  set: (id: string) => {
    if (!id || id === store.currentId) return;
    store.select(id);
    emit("changed", id);
  },
});

function label(e: { name: string; isDefault: boolean; active: boolean }) {
  if (e.isDefault) return `★ ${e.name}`;
  return e.active ? e.name : `${e.name} (rascunho)`;
}
</script>

<template>
  <label :class="['block', props.variant === 'dark' ? 'text-white' : 'text-ink']">
    <span :class="['mb-1 flex items-center gap-1.5 font-mono text-[10px] font-bold uppercase tracking-widest', props.variant === 'dark' ? 'text-white/50' : 'text-ink-soft']">
      <CalendarRange :size="12" /> Evento
    </span>
    <select
      v-model="model"
      :disabled="!store.events.length"
      :class="[
        'w-full rounded-lg px-3 py-2 text-sm font-semibold focus:outline-none',
        props.variant === 'dark'
          ? 'border border-white/20 bg-white/10 text-white focus:border-magenta [&>option]:text-ink'
          : 'border border-line bg-white text-ink focus:border-magenta',
      ]"
      aria-label="Evento selecionado"
    >
      <option v-if="!store.events.length" value="">Carregando…</option>
      <option v-for="e in store.events" :key="e.id" :value="e.id">{{ label(e) }}</option>
    </select>
  </label>
</template>
