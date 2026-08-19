<script setup lang="ts">
import { RouterLink, useRouter, useRoute } from "vue-router";
import { ShieldCheck, LogOut, CalendarCheck } from "lucide-vue-next";
import { useAuthStore } from "@/stores/auth";

withDefaults(defineProps<{ transparent?: boolean }>(), {
  transparent: false,
});

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();

const navItems = [
  { label: "Experiência", target: "#experiencia" },
  { label: "Aulas & combos", target: "#aulas-combos" },
  { label: "Café da manhã", target: "#cafe-da-manha" },
  { label: "Como funciona", target: "#como-funciona" },
];

function scrollTo(target: string) {
  if (route.path !== "/") {
    router.push({ path: "/", hash: target });
  } else {
    document.querySelector(target)?.scrollIntoView({ behavior: "smooth" });
  }
}

function handleLogout() {
  authStore.logout();
  router.push("/");
}
</script>

<template>
  <header :class="['sticky top-0 z-40 transition-all duration-200', transparent ? 'bg-paper/85 backdrop-blur-md border-b border-line/70' : 'bg-paper/95 backdrop-blur-md border-b border-line shadow-xs']">
    <div class="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
      <RouterLink to="/" class="flex items-center gap-1.5 font-bold tracking-tight text-ink" aria-label="P5 Wellness Club × AYO">
        <span class="font-sans text-xl font-bold text-ink">P5</span>
        <span class="text-lg font-light text-magenta">/</span>
        <span class="font-sans text-sm font-bold tracking-wider text-magenta">AYO</span>
      </RouterLink>

      <nav class="hidden items-center gap-8 md:flex" aria-label="Navegação principal">
        <button
          v-for="item in navItems"
          :key="item.label"
          class="text-sm font-medium text-ink-soft transition-colors hover:text-ink"
          @click="scrollTo(item.target)"
        >
          {{ item.label }}
        </button>
      </nav>

      <div class="flex items-center gap-3">
        <template v-if="authStore.token">
          <button class="hidden items-center gap-2 rounded-full px-4 py-2 text-sm font-medium text-ink-soft hover:text-ink sm:flex" @click="router.push('/portal')">
            <ShieldCheck :size="16" />
            <span>Minha área</span>
          </button>
          <button class="rounded-full p-2 text-ink-soft hover:text-ink" aria-label="Sair" @click="handleLogout">
            <LogOut :size="16" />
          </button>
        </template>
        <template v-else>
          <button class="hidden text-sm font-medium text-ink-soft hover:text-ink sm:block" @click="router.push('/acesso-admin')">
            Equipe
          </button>
          <button class="hidden rounded-full border border-line bg-white/60 px-4 py-2 text-sm font-semibold text-ink transition-colors hover:border-ink sm:block" @click="router.push('/entrar')">
            Portal do aluno
          </button>
        </template>
        <button class="button-magenta" @click="router.push('/comprar')">
          <CalendarCheck :size="16" />
          Reservar
        </button>
      </div>
    </div>
  </header>
</template>

