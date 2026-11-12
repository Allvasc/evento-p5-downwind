import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { api, TEAM_EVENT_KEY } from "@/lib/teamApi";

export interface TeamEvent {
  id: string;
  name: string;
  slug: string;
  isDefault: boolean;
  active: boolean;
}

// Evento selecionado no Admin e no terminal de check-in — a mesma escolha vale para os
// dois (fica no localStorage do aparelho). Trocar de evento troca toda a visão do sistema:
// o teamApi manda o id em cada chamada e o backend filtra por ele.
export const useTeamEventStore = defineStore("team-event", () => {
  const events = ref<TeamEvent[]>([]);
  const currentId = ref<string | null>(localStorage.getItem(TEAM_EVENT_KEY));
  const loaded = ref(false);

  const current = computed(() => events.value.find((e) => e.id === currentId.value) ?? null);

  function select(id: string) {
    currentId.value = id;
    localStorage.setItem(TEAM_EVENT_KEY, id);
  }

  // Carrega a lista e garante um evento válido selecionado: o último escolhido neste
  // aparelho, se ainda existir; senão o evento principal.
  async function load() {
    const res = await api.get<{ events: TeamEvent[] }>("/team/events");
    events.value = res.events;
    loaded.value = true;
    if (!current.value) {
      const fallback = res.events.find((e) => e.isDefault) ?? res.events[0];
      if (fallback) select(fallback.id);
    }
  }

  function clear() {
    currentId.value = null;
    localStorage.removeItem(TEAM_EVENT_KEY);
  }

  return { events, currentId, current, loaded, select, load, clear };
});
