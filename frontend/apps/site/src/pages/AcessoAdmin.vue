<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { Lock, ArrowLeft, ArrowRight, Check } from "lucide-vue-next";
import { api, ApiError } from "@/lib/api";
import BrandMark from "@/components/BrandMark.vue";
import { useTeamAuthStore } from "@/stores/teamAuth";

const router = useRouter();
const teamAuth = useTeamAuthStore();

type Mode = "login" | "recover" | "reset";
const mode = ref<Mode>("login");

const email = ref("");
const password = ref("");
const resetCode = ref("");
const loading = ref(false);
const errorMessage = ref("");
const successMessage = ref("");

// "admin" e "reports" enxergam o painel em /admin; "staff" só entra no check-in.
function hasAdminAreaAccess(role: string) {
  return role === "admin" || role === "reports";
}

async function submit() {
  errorMessage.value = "";
  successMessage.value = "";
  loading.value = true;
  try {
    if (mode.value === "login") {
      const res = await api.post<{ token: string; role: string; name: string; vendorId: string; vendorName: string }>("/auth/team/login", {
        email: email.value,
        password: password.value,
      });
      teamAuth.setSession(res.token, res.role, res.name, res.vendorId, res.vendorName);
      router.push(hasAdminAreaAccess(res.role) ? "/admin" : "/check-in");
    } else if (mode.value === "recover") {
      await api.post("/auth/team/request-password-reset", { email: email.value });
      successMessage.value = "Se o e-mail existir, você receberá um código de recuperação.";
      mode.value = "reset";
    } else {
      await api.post("/auth/team/reset-password", { email: email.value, code: resetCode.value, password: password.value });
      successMessage.value = "Senha redefinida com sucesso. Você já pode entrar.";
      mode.value = "login";
      password.value = "";
    }
  } catch (err) {
    errorMessage.value = err instanceof ApiError ? err.message : "Não foi possível concluir. Tente novamente.";
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="min-h-screen bg-paper flex flex-col justify-between">
    <!-- Header -->
    <header class="sticky top-0 z-40 bg-paper/90 backdrop-blur border-b border-line">
      <div class="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
        <a href="/" aria-label="P5 DownWind Day">
          <BrandMark :size="30" />
        </a>
        <span class="font-mono text-xs font-bold tracking-widest text-magenta uppercase">
          ÁREA DA EQUIPE
        </span>
      </div>
    </header>

    <main class="my-auto mx-auto w-full max-w-5xl px-6 py-10">
      <div class="overflow-hidden rounded-[2rem] border border-line/80 shadow-2xl grid md:grid-cols-2">
        <!-- LEFT PANE (DARK BLUE) -->
        <div class="bg-ink p-10 text-white flex flex-col justify-between min-h-[440px]">
          <div>
            <p class="font-mono text-xs font-bold tracking-widest text-magenta uppercase">
              ✦ P5 DOWNWIND DAY
            </p>

            <h1 class="mt-8 font-serif text-4xl font-bold leading-tight text-white md:text-5xl">
              Operação<br />
              <span class="text-magenta">P5 Kite House.</span>
            </h1>

            <p class="mt-4 text-sm leading-relaxed text-white/70">
              Este acesso é exclusivo para administradores e funcionários responsáveis pela operação do evento.
            </p>
          </div>

          <div class="mt-10 flex items-center gap-3 text-xs text-white/80 border-t border-white/10 pt-6">
            <div class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-magenta/20 text-magenta">
              <Check :size="13" />
            </div>
            <span>Gestão de ingressos, vendas, check-in e equipe.</span>
          </div>
        </div>

        <!-- RIGHT PANE (LIGHT CREAM) -->
        <div class="bg-[#fbf8f3] p-10 flex flex-col justify-between">
          <div>
            <button
              class="inline-flex items-center gap-2 text-xs font-semibold text-ink-soft hover:text-ink mb-6 transition-colors"
              @click="router.push('/')"
            >
              <ArrowLeft :size="14" /> Voltar para o início
            </button>

            <div class="flex items-center gap-3 mb-2">
              <div class="flex h-9 w-9 items-center justify-center rounded-xl bg-magenta/10 text-magenta">
                <Lock :size="20" />
              </div>
              <h2 class="font-serif text-3xl font-bold text-ink">
                {{ mode === "login" ? "Acesso administrativo" : mode === "recover" ? "Recuperar senha" : "Redefinir senha" }}
              </h2>
            </div>
            <p class="text-xs text-ink-soft mb-6">
              {{ mode === "login" ? "Use a credencial cadastrada exclusivamente para a equipe." : mode === "recover" ? "Informe seu e-mail corporativo para receber um código de recuperação." : "Informe o código recebido por e-mail e a nova senha." }}
            </p>

            <!-- FORM -->
            <form class="space-y-4" @submit.prevent="submit">
              <div>
                <label for="admin-email" class="mb-1.5 block text-xs font-bold uppercase tracking-wider text-ink">E-mail corporativo</label>
                <input
                  id="admin-email"
                  v-model="email"
                  type="email"
                  required
                  placeholder="equipe@p5kitehouse.com"
                  class="w-full rounded-xl border border-line bg-white px-4 py-3 text-sm text-ink placeholder:text-ink-soft/50 focus:border-magenta focus:outline-none focus:ring-1 focus:ring-magenta"
                />
              </div>

              <div v-if="mode === 'reset'">
                <label for="admin-reset-code" class="mb-1.5 block text-xs font-bold uppercase tracking-wider text-ink">Código recebido por e-mail</label>
                <input
                  id="admin-reset-code"
                  v-model="resetCode"
                  required
                  placeholder="000000"
                  class="w-full rounded-xl border border-line bg-white px-4 py-3 text-sm text-ink placeholder:text-ink-soft/50 focus:border-magenta focus:outline-none focus:ring-1 focus:ring-magenta"
                />
              </div>

              <div v-if="mode !== 'recover'">
                <label for="admin-password" class="mb-1.5 block text-xs font-bold uppercase tracking-wider text-ink">{{ mode === "reset" ? "Nova senha" : "Senha" }}</label>
                <input
                  id="admin-password"
                  v-model="password"
                  type="password"
                  required
                  placeholder="Sua senha de equipe"
                  class="w-full rounded-xl border border-line bg-white px-4 py-3 text-sm text-ink placeholder:text-ink-soft/50 focus:border-magenta focus:outline-none focus:ring-1 focus:ring-magenta"
                />
              </div>

              <p v-if="errorMessage" class="text-xs font-medium text-red-600">{{ errorMessage }}</p>
              <p v-if="successMessage" class="text-xs font-medium text-[#237438]">{{ successMessage }}</p>

              <button type="submit" :disabled="loading" class="button-magenta w-full justify-center text-sm py-3.5 mt-2">
                <span>{{ loading ? "Enviando..." : mode === "login" ? "Entrar na operação" : mode === "recover" ? "Enviar código" : "Redefinir senha" }}</span>
                <ArrowRight :size="16" />
              </button>

              <button
                type="button"
                class="w-full text-center text-xs font-semibold text-ink-soft hover:text-ink"
                @click="mode = mode === 'login' ? 'recover' : 'login'; errorMessage = ''; successMessage = ''"
              >
                {{ mode === "login" ? "Esqueci minha senha" : "Voltar para o login" }}
              </button>
            </form>
          </div>

          <div class="mt-8 text-center text-xs text-ink-soft">
            Ambiente seguro de gestão operacional.
          </div>
        </div>
      </div>
    </main>

    <footer class="py-6 text-center text-xs text-ink-soft border-t border-line/40 space-y-3">
      <p>&copy; {{ new Date().getFullYear() }} P5 DownWind Day — Operação</p>
      <a
        href="https://prolins.com.br"
        target="_blank"
        rel="noopener noreferrer"
        aria-label="Powered by Prolins Software House e Outsource"
        class="inline-block opacity-80 transition-opacity hover:opacity-100"
      >
        <img src="/prolins-selo.webp" alt="Powered by Prolins Software House e Outsource" width="44" height="44" class="h-11 w-11" />
      </a>
    </footer>
  </div>
</template>
