import { defineStore } from "pinia";
import { ref } from "vue";
import { api } from "@/lib/api";

interface Me {
  id: string;
  fullName: string;
  email: string;
  phone: string;
  cpfLast4: string;
}

const TOKEN_KEY = "p5_student_token";

export const useAuthStore = defineStore("auth", () => {
  const token = ref<string | null>(localStorage.getItem(TOKEN_KEY));
  const me = ref<Me | null>(null);
  const loading = ref(false);

  function setToken(value: string) {
    token.value = value;
    localStorage.setItem(TOKEN_KEY, value);
  }

  function logout() {
    token.value = null;
    me.value = null;
    localStorage.removeItem(TOKEN_KEY);
  }

  async function fetchMe() {
    if (!token.value) return;
    loading.value = true;
    try {
      me.value = await api.get<Me>("/me/");
    } catch {
      logout();
    } finally {
      loading.value = false;
    }
  }

  return { token, me, loading, setToken, logout, fetchMe };
});
