import { createRouter, createWebHistory } from "vue-router";
import { useTeamAuthStore } from "@/stores/teamAuth";

export const router = createRouter({
  history: createWebHistory(),
  scrollBehavior(to, _from, savedPosition) {
    if (savedPosition) return savedPosition;
    if (to.hash) return { el: to.hash, behavior: "smooth" };
    return { top: 0 };
  },
  routes: [
    { path: "/", name: "home", component: () => import("./pages/Home.vue") },
    { path: "/comprar", name: "comprar", component: () => import("./pages/Comprar.vue") },
    { path: "/comprar/:orderId", name: "pagamento", component: () => import("./pages/Pagamento.vue") },
    { path: "/entrar", name: "entrar", component: () => import("./pages/Entrar.vue") },
    { path: "/acesso-admin", name: "acesso-admin", component: () => import("./pages/AcessoAdmin.vue") },
    { path: "/portal", name: "portal", component: () => import("./pages/Portal.vue") },
    { path: "/perfil", name: "perfil", component: () => import("./pages/Perfil.vue") },
    { path: "/check-in", name: "check-in", component: () => import("./pages/CheckIn.vue"), meta: { requiresTeamAuth: true } },
    { path: "/admin", name: "admin", component: () => import("./pages/Admin.vue"), meta: { requiresTeamAuth: true, requiresAdminArea: true } },
    { path: "/:pathMatch(.*)*", name: "not-found", component: () => import("./pages/NotFound.vue") },
  ],
});

// "admin" enxerga o painel inteiro; "reports" enxerga só a aba Relatórios dentro dele
// (Admin.vue filtra o resto). "staff" não entra no /admin, só no check-in.
function hasAdminAreaAccess(role: string | null) {
  return role === "admin" || role === "reports";
}

router.beforeEach((to) => {
  if (!to.meta.requiresTeamAuth && !to.meta.requiresAdminArea) return true;

  const teamAuth = useTeamAuthStore();
  if (to.meta.requiresTeamAuth && !teamAuth.token) {
    return { path: "/acesso-admin" };
  }
  if (to.meta.requiresAdminArea && !hasAdminAreaAccess(teamAuth.role)) {
    return { path: "/check-in" };
  }
  return true;
});
