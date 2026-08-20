<script setup lang="ts">
import { ref, computed } from "vue";
import { useRouter, useRoute } from "vue-router";
import { Lock, User, KeyRound, ArrowLeft, ArrowRight, Check } from "lucide-vue-next";
import { api, ApiError } from "@/lib/api";
import WellnessHeader from "@/components/WellnessHeader.vue";
import { useAuthStore } from "@/stores/auth";

const authStore = useAuthStore();
const router = useRouter();
const route = useRoute();

type Mode = "login" | "register" | "recover" | "reset";
const mode = ref<Mode>("login");
const loading = ref(false);
const errorMessage = ref("");
const successMessage = ref("");

const name = ref("");
const phone = ref("");
const cpf = ref("");
const email = ref("");
const password = ref("");
const resetToken = ref("");

const heading = computed(() => {
  if (mode.value === "login") return "Entrar";
  if (mode.value === "register") return "Criar cadastro";
  if (mode.value === "recover") return "Recuperar senha";
  return "Redefinir senha";
});

const leftTitle = computed(() => {
  if (mode.value === "login") return "Que bom ter";
  if (mode.value === "register") return "Comece sua";
  return "Vamos recuperar";
});

const leftTitleItalic = computed(() => {
  if (mode.value === "login") return "você de volta.";
  if (mode.value === "register") return "jornada de bem-estar.";
  return "seu acesso.";
});

const leftSubtitle = computed(() => {
  if (mode.value === "login") return "Entre para acessar sua jornada.";
  if (mode.value === "register") return "Crie seu cadastro em menos de 1 minuto para reservar aulas, combos e consultar seus acessos.";
  return "Informe seu e-mail para receber as orientações.";
});

const leftCheckmark = computed(() => {
  if (mode.value === "login") return "Cadastro único para suas próximas aulas e combos.";
  if (mode.value === "register") return "Seus dados salvos para futuras compras.";
  return "Suporte rápido e seguro para sua conta.";
});

async function submit() {
  errorMessage.value = "";
  successMessage.value = "";
  loading.value = true;
  try {
    if (mode.value === "login") {
      const res = await api.post<{ token: string }>("/auth/student/login", { email: email.value, password: password.value });
      authStore.setToken(res.token);
      const redirectPath = (route.query.redirect as string) || "/portal";
      router.push(redirectPath);
    } else if (mode.value === "register") {
      const res = await api.post<{ token: string }>("/auth/student/register", {
        fullName: name.value,
        phone: phone.value,
        cpf: cpf.value || undefined,
        email: email.value,
        password: password.value,
      });
      authStore.setToken(res.token);
      const redirectPath = (route.query.redirect as string) || "/portal";
      router.push(redirectPath);
    } else if (mode.value === "recover") {
      await api.post("/auth/student/request-password-reset", { email: email.value });
      successMessage.value = "Se o e-mail existir, você receberá um código de recuperação.";
      mode.value = "reset";
    } else {
      await api.post("/auth/student/reset-password", { email: email.value, code: resetToken.value, password: password.value });
      successMessage.value = "Senha redefinida com sucesso. Você já pode entrar.";
      mode.value = "login";
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
    <WellnessHeader />

    <main class="my-auto mx-auto w-full max-w-5xl px-6 py-10">
      <div class="overflow-hidden rounded-[2rem] border border-line/80 shadow-2xl grid md:grid-cols-2">
        <!-- LEFT PANE (DARK BLUE) -->
        <div class="bg-ink p-10 text-white flex flex-col justify-between min-h-[440px]">
          <div>
            <p class="font-mono text-xs font-bold tracking-widest text-magenta uppercase">
              ✦ P5 DOWNWIND DAY
            </p>

            <h1 class="mt-8 font-serif text-4xl font-bold leading-tight text-white md:text-5xl">
              {{ leftTitle }}<br />
              <em class="font-serif italic text-magenta">{{ leftTitleItalic }}</em>
            </h1>

            <p class="mt-4 text-sm leading-relaxed text-white/70">
              {{ leftSubtitle }}
            </p>
          </div>

          <div class="mt-10 flex items-center gap-3 text-xs text-white/80 border-t border-white/10 pt-6">
            <div class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-magenta/20 text-magenta">
              <Check :size="13" />
            </div>
            <span>{{ leftCheckmark }}</span>
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

            <div class="flex items-center gap-3 mb-6">
              <div class="flex h-9 w-9 items-center justify-center rounded-xl bg-magenta/10 text-magenta">
                <Lock v-if="mode === 'login'" :size="20" />
                <User v-else-if="mode === 'register'" :size="20" />
                <KeyRound v-else :size="20" />
              </div>
              <h2 class="font-serif text-3xl font-bold text-ink">{{ heading }}</h2>
            </div>

            <!-- FORM -->
            <form class="space-y-4" @submit.prevent="submit">
              <template v-if="mode === 'register'">
                <div>
                  <label for="name" class="mb-1.5 block text-xs font-bold uppercase tracking-wider text-ink">Nome completo</label>
                  <input
                    id="name"
                    v-model="name"
                    required
                    placeholder="Seu nome completo"
                    class="w-full rounded-xl border border-line bg-white px-4 py-3 text-sm text-ink placeholder:text-ink-soft/50 focus:border-magenta focus:outline-none focus:ring-1 focus:ring-magenta"
                  />
                </div>
              </template>

              <div v-if="mode !== 'reset'">
                <label for="email" class="mb-1.5 block text-xs font-bold uppercase tracking-wider text-ink">E-mail</label>
                <input
                  id="email"
                  v-model="email"
                  type="email"
                  required
                  placeholder="voce@email.com"
                  class="w-full rounded-xl border border-line bg-white px-4 py-3 text-sm text-ink placeholder:text-ink-soft/50 focus:border-magenta focus:outline-none focus:ring-1 focus:ring-magenta"
                />
              </div>

              <template v-if="mode === 'register'">
                <div class="grid grid-cols-2 gap-3">
                  <div>
                    <label for="phone" class="mb-1.5 block text-xs font-bold uppercase tracking-wider text-ink">Telefone</label>
                    <input
                      id="phone"
                      v-model="phone"
                      required
                      placeholder="(00) 00000-0000"
                      class="w-full rounded-xl border border-line bg-white px-4 py-3 text-sm text-ink placeholder:text-ink-soft/50 focus:border-magenta focus:outline-none focus:ring-1 focus:ring-magenta"
                    />
                  </div>
                  <div>
                    <label for="cpf" class="mb-1.5 block text-xs font-bold uppercase tracking-wider text-ink">CPF</label>
                    <input
                      id="cpf"
                      v-model="cpf"
                      placeholder="000.000.000-00"
                      class="w-full rounded-xl border border-line bg-white px-4 py-3 text-sm text-ink placeholder:text-ink-soft/50 focus:border-magenta focus:outline-none focus:ring-1 focus:ring-magenta"
                    />
                  </div>
                </div>
                <p class="-mt-2 text-[11px] text-ink-soft">O CPF é necessário para finalizar o pagamento — pode preencher agora ou depois, no seu perfil.</p>
              </template>

              <div v-if="mode === 'reset'">
                <label for="token" class="mb-1.5 block text-xs font-bold uppercase tracking-wider text-ink">Código de recuperação</label>
                <input
                  id="token"
                  v-model="resetToken"
                  required
                  placeholder="Código de 6 dígitos"
                  class="w-full rounded-xl border border-line bg-white px-4 py-3 text-sm text-ink placeholder:text-ink-soft/50 focus:border-magenta focus:outline-none focus:ring-1 focus:ring-magenta"
                />
              </div>

              <div v-if="mode !== 'recover'">
                <label for="password" class="mb-1.5 block text-xs font-bold uppercase tracking-wider text-ink">
                  {{ mode === "reset" ? "Nova senha" : "Senha" }}
                </label>
                <input
                  id="password"
                  v-model="password"
                  type="password"
                  required
                  minlength="8"
                  placeholder="Mínimo de 8 caracteres"
                  class="w-full rounded-xl border border-line bg-white px-4 py-3 text-sm text-ink placeholder:text-ink-soft/50 focus:border-magenta focus:outline-none focus:ring-1 focus:ring-magenta"
                />
              </div>

              <p v-if="errorMessage" class="text-xs font-medium text-red-600">{{ errorMessage }}</p>
              <p v-if="successMessage" class="text-xs font-medium text-green-700">{{ successMessage }}</p>

              <button type="submit" :disabled="loading" class="button-magenta w-full justify-center text-sm py-3.5 mt-2">
                <span>{{ loading ? "Carregando..." : mode === 'register' ? 'Criar meu cadastro' : mode === 'login' ? 'Entrar' : 'Enviar código' }}</span>
                <ArrowRight :size="16" />
              </button>
            </form>
          </div>

          <!-- BOTTOM TOGGLE LINKS -->
          <div class="mt-8 space-y-2 text-center text-xs text-ink-soft">
            <p v-if="mode === 'login'">
              <button class="font-medium text-magenta hover:underline" @click="mode = 'recover'">
                Esqueci minha senha
              </button>
            </p>
            <p v-if="mode === 'login'">
              Ainda não tem cadastro?
              <button class="font-bold text-magenta hover:underline ml-1" @click="mode = 'register'">
                Comece agora
              </button>
            </p>
            <p v-if="mode === 'register' || mode === 'recover' || mode === 'reset'">
              Já tem cadastro?
              <button class="font-bold text-magenta hover:underline ml-1" @click="mode = 'login'">
                Entrar
              </button>
            </p>
          </div>
        </div>
      </div>
    </main>

    <footer class="py-6 text-center text-xs text-ink-soft border-t border-line/40 space-y-3">
      <p>&copy; {{ new Date().getFullYear() }} P5 DownWind Day. Todos os direitos reservados.</p>
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

