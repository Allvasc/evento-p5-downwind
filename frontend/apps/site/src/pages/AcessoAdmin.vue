<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { Lock, ArrowLeft, ArrowRight, Check } from "lucide-vue-next";
import { api, ApiError } from "@/lib/api";

const router = useRouter();

type Mode = "login" | "recover" | "reset";
const mode = ref<Mode>("login");

const email = ref("");
const password = ref("");
const resetCode = ref("");
const loading = ref(false);
const errorMessage = ref("");
const successMessage = ref("");

// O painel admin roda como um app/origem separado (bundle isolado do site público).
// Em dev e na stack local containerizada ele sempre está uma porta acima do site
// (5183→5184, 8081→8082); em produção vive em wellness-admin.p5beachclub.com.br,
// que não é um subdomínio previsível a partir do domínio do site (não é
// "admin.<host>"), então o valor vem gravado no build via VITE_ADMIN_URL (ver
// .env.production). O token não pode ser lido via localStorage entre origens
// diferentes, então ele viaja pela URL e o app admin o consome no primeiro
// carregamento (ver router.ts de apps/admin).
function adminBaseUrl(): string {
  if (import.meta.env.VITE_ADMIN_URL) return import.meta.env.VITE_ADMIN_URL;
  const { protocol, hostname, port } = window.location;
  if (port) return `${protocol}//${hostname}:${Number(port) + 1}`;
  return `${protocol}//admin.${hostname}`;
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
      const params = new URLSearchParams({
        token: res.token, role: res.role, name: res.name,
        vendorId: res.vendorId ?? "", vendorName: res.vendorName ?? "",
      });
      window.location.href = `${adminBaseUrl()}/?${params.toString()}`;
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
        <a href="/" class="flex items-center gap-1.5 font-bold tracking-tight text-ink">
          <span class="font-sans text-xl font-bold text-ink">P5</span>
          <span class="text-lg font-light text-magenta">/</span>
          <span class="font-sans text-sm font-bold tracking-wider text-magenta">AYO</span>
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
              ✦ P5 WELLNESS CLUB
            </p>

            <h1 class="mt-8 font-serif text-4xl font-bold leading-tight text-white md:text-5xl">
              Operação<br />
              <em class="font-serif italic text-magenta">P5 Wellness.</em>
            </h1>

            <p class="mt-4 text-sm leading-relaxed text-white/70">
              Este acesso é exclusivo para administradores e funcionários responsáveis pela operação do clube.
            </p>
          </div>

          <div class="mt-10 flex items-center gap-3 text-xs text-white/80 border-t border-white/10 pt-6">
            <div class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-magenta/20 text-magenta">
              <Check :size="13" />
            </div>
            <span>Gestão de aulas, vendas, presença e equipe.</span>
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
                  placeholder="equipe@p5wellness.com"
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

    <footer class="py-6 text-center text-xs text-ink-soft border-t border-line/40">
      &copy; {{ new Date().getFullYear() }} P5 Wellness Club — Operação
    </footer>
  </div>
</template>
