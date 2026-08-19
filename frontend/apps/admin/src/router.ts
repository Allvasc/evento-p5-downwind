import { createRouter, createWebHistory } from "vue-router";
import { useAuthStore } from "./stores/auth";

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", redirect: "/check-in" },
    { path: "/acesso-admin", name: "acesso-admin", component: () => import("./pages/AcessoAdmin.vue") },
    { path: "/check-in", name: "check-in", component: () => import("./pages/CheckIn.vue"), meta: { requiresAuth: true } },
    { path: "/admin", name: "admin", component: () => import("./pages/Admin.vue"), meta: { requiresAuth: true, requiresAdmin: true } },
    { path: "/:pathMatch(.*)*", name: "not-found", component: () => import("./pages/NotFound.vue") },
  ],
});

router.beforeEach((to) => {
  const auth = useAuthStore();

  // Handoff do login feito em /acesso-admin do site público (origem separada — o token
  // não pode ser lido via localStorage entre portas/domínios diferentes, então chega
  // pela URL uma única vez e é persistido aqui).
  const { token, role, name, vendorId, vendorName } = to.query;
  if (typeof token === "string" && typeof role === "string" && typeof name === "string") {
    auth.setSession(token, role, name, typeof vendorId === "string" ? vendorId : "", typeof vendorName === "string" ? vendorName : "");
    return { path: hasAdminAreaAccess(role) ? "/admin" : "/check-in" };
  }

  if (to.meta.requiresAuth && !auth.token) {
    return { path: "/acesso-admin" };
  }
  if (to.meta.requiresAdmin && !hasAdminAreaAccess(auth.role)) {
    return { path: "/check-in" };
  }
  if (to.path === "/acesso-admin" && auth.token) {
    return { path: hasAdminAreaAccess(auth.role) ? "/admin" : "/check-in" };
  }
  return true;
});

// "admin" enxerga o painel inteiro; "reports" enxerga só a aba Relatórios dentro dele
// (Admin.vue filtra o resto). "staff" não entra no /admin, só no check-in.
function hasAdminAreaAccess(role: string | null) {
  return role === "admin" || role === "reports";
}
