import { createRouter, createWebHistory } from "vue-router";

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
    { path: "/:pathMatch(.*)*", name: "not-found", component: () => import("./pages/NotFound.vue") },
  ],
});
