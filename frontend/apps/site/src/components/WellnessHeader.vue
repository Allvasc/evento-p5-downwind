<script setup lang="ts">
import { computed } from "vue";
import { RouterLink, useRouter, useRoute } from "vue-router";
import { ShieldCheck, LogOut, CalendarCheck } from "lucide-vue-next";
import { useAuthStore } from "@/stores/auth";

// eventSlug/eventName: páginas de outros eventos (/<slug>). Sem eles o cabeçalho é o do
// evento principal, exatamente como sempre foi.
const props = withDefaults(defineProps<{ transparent?: boolean; eventSlug?: string; eventName?: string }>(), {
  transparent: false,
  eventSlug: "",
  eventName: "",
});
const isEventMode = computed(() => !!props.eventSlug);
const eventHome = computed(() => `/${props.eventSlug}`);
const eventNavItems = [
  { label: "O que inclui", target: "#inclui" },
  { label: "Programação", target: "#programacao" },
  { label: "Ingressos", target: "#preco" },
];

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();

const navItems = [
  { label: "O que inclui", target: "#inclui" },
  { label: "Percurso", target: "#percurso" },
  { label: "Sertões Kitesurf", target: "#sertoes" },
  { label: "Arte do Evento", target: "#arte-evento" },
  { label: "Preço", target: "#preco" },
];

function scrollTo(target: string) {
  const home = isEventMode.value ? eventHome.value : "/";
  if (route.path !== home) {
    router.push({ path: home, hash: target });
  } else {
    document.querySelector(target)?.scrollIntoView({ behavior: "smooth" });
  }
}

function goBuy() {
  router.push(isEventMode.value ? `${eventHome.value}/comprar` : "/comprar");
}

function handleLogout() {
  authStore.logout();
  router.push("/");
}
</script>

<template>
  <header :class="['sticky top-0 z-40 transition-all duration-200', transparent ? 'bg-paper/85 backdrop-blur-md border-b border-line/70' : 'bg-paper/95 backdrop-blur-md border-b border-line shadow-xs']">
    <div class="mx-auto flex max-w-6xl items-center justify-between gap-2 px-6 py-3.5 sm:gap-6">
      <RouterLink v-if="isEventMode" :to="eventHome" :aria-label="eventName || 'Evento P5'" class="flex min-w-0 items-center gap-2.5">
        <span class="font-serif text-lg font-black text-ink">P5</span>
        <span class="h-5 w-px shrink-0 bg-line/80"></span>
        <span class="truncate font-serif text-sm font-bold text-ink sm:text-base">{{ eventName }}</span>
      </RouterLink>
      <!-- No celular só o logo DownWind; o selo Sertões aparece logo abaixo, no hero. -->
      <RouterLink v-else to="/" aria-label="P5 DownWind Day & Sertões Kitesurf" class="flex shrink-0 items-center gap-2.5 sm:gap-3 group">
        <img src="/logo-downwind.webp" alt="P5 DownWind Day" class="h-7 w-auto sm:h-9" />
        <span class="hidden h-5 w-px bg-line/80 sm:block"></span>
        <img src="/sertoes-kitesurf-verde.webp" alt="Sertões Kitesurf" class="hidden h-8 w-auto rounded-sm shadow-2xs transition-transform group-hover:scale-105 sm:block" />
      </RouterLink>

      <!-- O menu só aparece quando cabe inteiro numa linha (xl); abaixo disso o header fica com logo + acesso + CTA. -->
      <nav class="hidden items-center gap-6 xl:flex" aria-label="Navegação principal">
        <button
          v-for="item in (isEventMode ? eventNavItems : navItems)"
          :key="item.label"
          class="whitespace-nowrap text-sm font-medium text-ink-soft transition-colors hover:text-ink"
          @click="scrollTo(item.target)"
        >
          {{ item.label }}
        </button>
      </nav>

      <div class="flex shrink-0 items-center gap-2 sm:gap-3">
        <template v-if="authStore.token">
          <button class="flex items-center gap-1.5 whitespace-nowrap rounded-full px-2 py-2 text-xs font-medium text-ink-soft hover:text-ink sm:gap-2 sm:px-4 sm:text-sm" @click="router.push('/portal')" aria-label="Minha área">
            <ShieldCheck :size="16" />
            <span class="hidden min-[440px]:inline sm:hidden">Área</span>
            <span class="hidden sm:inline">Minha área</span>
          </button>
          <button class="rounded-full p-2 text-ink-soft hover:text-ink" aria-label="Sair" @click="handleLogout">
            <LogOut :size="16" />
          </button>
        </template>
        <template v-else>
          <!-- No desktop o menu ocupa esse espaço; o acesso da equipe segue no rodapé ("Acesso equipe"). -->
          <button class="hidden whitespace-nowrap text-sm font-medium text-ink-soft hover:text-ink md:block xl:hidden" @click="router.push('/acesso-admin')">
            Equipe
          </button>
          <button class="flex items-center gap-1.5 whitespace-nowrap rounded-full border border-line bg-white/60 px-2.5 py-2 text-xs font-semibold text-ink transition-colors hover:border-ink sm:gap-2 sm:px-4 sm:text-sm" @click="router.push('/entrar')" aria-label="Portal do aluno">
            <ShieldCheck :size="16" />
            <span class="hidden min-[440px]:inline sm:hidden">Entrar</span>
            <span class="hidden sm:inline">Portal do aluno</span>
          </button>
        </template>
        <button class="button-magenta whitespace-nowrap px-3.5! py-2.5! text-sm sm:px-5!" @click="goBuy">
          <CalendarCheck :size="16" class="hidden sm:block" />
          <span>Garantir<span class="hidden min-[400px]:inline">&nbsp;vaga</span></span>
        </button>
      </div>
    </div>
  </header>
</template>
