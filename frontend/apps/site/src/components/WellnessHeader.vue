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
  { label: "O que inclui", target: "#inclui" },
  { label: "Percurso", target: "#percurso" },
  { label: "Preço", target: "#preco" },
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
    <div class="mx-auto flex max-w-6xl items-center justify-between px-6 py-3.5">
      <RouterLink to="/" aria-label="P5 DownWind Day">
        <img src="/logo-downwind.webp" alt="P5 DownWind Day" class="h-8 w-auto sm:h-9" />
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

      <div class="flex items-center gap-2 sm:gap-3">
        <template v-if="authStore.token">
          <button class="flex items-center gap-1.5 rounded-full px-2 py-2 text-xs font-medium text-ink-soft hover:text-ink sm:gap-2 sm:px-4 sm:text-sm" @click="router.push('/portal')" aria-label="Minha área">
            <ShieldCheck :size="16" />
            <span class="hidden min-[375px]:inline sm:hidden">Área</span>
            <span class="hidden sm:inline">Minha área</span>
          </button>
          <button class="rounded-full p-2 text-ink-soft hover:text-ink" aria-label="Sair" @click="handleLogout">
            <LogOut :size="16" />
          </button>
        </template>
        <template v-else>
          <button class="hidden text-sm font-medium text-ink-soft hover:text-ink sm:block" @click="router.push('/acesso-admin')">
            Equipe
          </button>
          <button class="flex items-center gap-1.5 rounded-full border border-line bg-white/60 px-2.5 py-2 text-xs font-semibold text-ink transition-colors hover:border-ink sm:gap-2 sm:px-4 sm:text-sm" @click="router.push('/entrar')" aria-label="Portal do aluno">
            <ShieldCheck :size="16" />
            <span class="hidden min-[375px]:inline sm:hidden">Entrar</span>
            <span class="hidden sm:inline">Portal do aluno</span>
          </button>
        </template>
        <button class="button-magenta" @click="router.push('/comprar')">
          <CalendarCheck :size="16" />
          Garantir vaga
        </button>
      </div>
    </div>
  </header>
</template>
