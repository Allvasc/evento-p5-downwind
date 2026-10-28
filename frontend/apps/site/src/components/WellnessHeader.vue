<script setup lang="ts">
import { RouterLink, useRouter, useRoute } from "vue-router";
import { ShieldCheck, LogOut, CalendarCheck } from "lucide-vue-next";
import { useAuthStore } from "@/stores/auth";

withDefaults(defineProps<{ transparent?: boolean; isHalloween?: boolean }>(), {
  transparent: false,
  isHalloween: false,
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
  <div>
    <!-- HALLOWEEN TOP BANNER -->
    <div
      v-if="isHalloween"
      class="relative z-50 overflow-hidden bg-gradient-to-r from-orange-600 via-purple-950 to-orange-600 px-4 py-2 text-center text-xs font-bold text-white shadow-md border-b border-orange-500/30"
    >
      <div class="mx-auto flex max-w-6xl items-center justify-center gap-2 sm:gap-3 text-[11px] sm:text-xs font-semibold tracking-wide">
        <span class="animate-bounce inline-block">🎃</span>
        <span class="text-orange-200">EDIÇÃO ESPECIAL HALLOWEEN · 31 DE OUTUBRO</span>
        <span class="hidden md:inline text-white/50">·</span>
        <span class="hidden md:inline text-white/90">SABABA → P5 KITE HOUSE · VELEJO À FANTASIA</span>
        <span class="rounded-full bg-orange-500/30 border border-orange-400/40 px-2 py-0.5 text-[10px] text-amber-200 font-mono uppercase tracking-wider">
          Apenas 16 vagas
        </span>
        <span class="animate-bounce inline-block [animation-delay:0.3s]">👻</span>
      </div>
    </div>

    <header
      :class="[
        'sticky top-0 z-40 transition-all duration-200',
        isHalloween
          ? 'bg-[#0d0418]/95 backdrop-blur-md border-b border-purple-900/50 shadow-[0_4px_25px_rgba(13,4,24,0.6)] text-white'
          : transparent
            ? 'bg-paper/85 backdrop-blur-md border-b border-line/70'
            : 'bg-paper/95 backdrop-blur-md border-b border-line shadow-xs'
      ]"
    >
      <div class="mx-auto flex max-w-6xl items-center justify-between px-6 py-3.5">
        <RouterLink to="/" aria-label="P5 DownWind Day" class="relative group inline-flex items-center">
          <!-- Mini chapéu de bruxa no logo quando Halloween -->
          <svg
            v-if="isHalloween"
            class="absolute -top-3.5 -left-2.5 h-6 w-6 transform -rotate-12 drop-shadow-[0_2px_4px_rgba(0,0,0,0.8)] pointer-events-none transition-transform group-hover:rotate-0 group-hover:scale-110"
            viewBox="0 0 40 40"
            fill="none"
          >
            <!-- aba do chapéu -->
            <ellipse cx="20" cy="33" rx="18" ry="4.5" fill="#4c1d95" stroke="#f97316" stroke-width="1.2" />
            <!-- cone -->
            <path d="M11 31 C15 24, 18 10, 26 5 C24 13, 24 22, 29 31 Z" fill="#2e1065" stroke="#7c3aed" stroke-width="1" />
            <!-- faixa laranja -->
            <path d="M11.5 30 C16 29, 24 29, 28.5 30 L28 32 C23 31, 17 31, 12 32 Z" fill="#ea580c" />
            <!-- fivela dourada -->
            <rect x="18" y="28.5" width="4" height="3" fill="#fbbf24" stroke="#d97706" stroke-width="0.5" rx="0.5" />
          </svg>
          <img
            :src="isHalloween ? '/logo-downwind-dark.webp' : '/logo-downwind.webp'"
            alt="P5 DownWind Day"
            class="h-8 w-auto sm:h-9"
          />
          <span
            v-if="isHalloween"
            class="ml-2 hidden sm:inline-flex items-center gap-1 rounded-full border border-orange-500/40 bg-orange-950/60 px-2 py-0.5 text-[10px] font-bold text-orange-400 font-mono tracking-wider uppercase shadow-[0_0_10px_rgba(249,115,22,0.3)]"
          >
            🎃 Halloween
          </span>
        </RouterLink>

        <nav class="hidden items-center gap-8 md:flex" aria-label="Navegação principal">
          <button
            v-for="item in navItems"
            :key="item.label"
            :class="[
              'text-sm font-medium transition-colors',
              isHalloween ? 'text-purple-200/80 hover:text-orange-400' : 'text-ink-soft hover:text-ink'
            ]"
            @click="scrollTo(item.target)"
          >
            {{ item.label }}
          </button>
        </nav>

        <div class="flex items-center gap-2 sm:gap-3">
          <template v-if="authStore.token">
            <button
              :class="[
                'flex items-center gap-1.5 rounded-full px-2 py-2 text-xs font-medium sm:gap-2 sm:px-4 sm:text-sm transition-colors',
                isHalloween ? 'text-purple-200 hover:text-white' : 'text-ink-soft hover:text-ink'
              ]"
              @click="router.push('/portal')"
              aria-label="Minha área"
            >
              <ShieldCheck :size="16" />
              <span class="hidden min-[375px]:inline sm:hidden">Área</span>
              <span class="hidden sm:inline">Minha área</span>
            </button>
            <button
              :class="[
                'rounded-full p-2 transition-colors',
                isHalloween ? 'text-purple-200 hover:text-white' : 'text-ink-soft hover:text-ink'
              ]"
              aria-label="Sair"
              @click="handleLogout"
            >
              <LogOut :size="16" />
            </button>
          </template>
          <template v-else>
            <button
              :class="[
                'hidden text-sm font-medium sm:block transition-colors',
                isHalloween ? 'text-purple-300 hover:text-orange-400' : 'text-ink-soft hover:text-ink'
              ]"
              @click="router.push('/acesso-admin')"
            >
              Equipe
            </button>
            <button
              :class="[
                'flex items-center gap-1.5 rounded-full px-2.5 py-2 text-xs font-semibold sm:gap-2 sm:px-4 sm:text-sm transition-colors',
                isHalloween
                  ? 'border border-purple-800/80 bg-purple-950/50 text-purple-200 hover:border-orange-500/80 hover:text-white hover:bg-purple-900/40 shadow-xs'
                  : 'border border-line bg-white/60 text-ink hover:border-ink'
              ]"
              @click="router.push('/entrar')"
              aria-label="Portal do aluno"
            >
              <ShieldCheck :size="16" />
              <span class="hidden min-[375px]:inline sm:hidden">Entrar</span>
              <span class="hidden sm:inline">Portal do aluno</span>
            </button>
          </template>
          <button
            :class="[
              isHalloween
                ? 'bg-gradient-to-r from-orange-600 via-orange-500 to-amber-500 hover:from-orange-500 hover:to-amber-400 text-white font-bold shadow-[0_0_20px_rgba(249,115,22,0.6)] hover:shadow-[0_0_28px_rgba(249,115,22,0.85)] border border-orange-400/40 rounded-full px-4 py-2.5 text-xs sm:text-sm inline-flex items-center gap-1.5 transition-all hover:scale-[1.02] active:scale-[0.98] cursor-pointer'
                : 'button-magenta'
            ]"
            @click="router.push('/comprar')"
          >
            <CalendarCheck :size="16" />
            <span>Garantir vaga</span>
          </button>
        </div>
      </div>
    </header>
  </div>
</template>
