<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useRouter } from "vue-router";
import { LayoutDashboard, Dumbbell, Package, Users, LogOut, Plus, ScanLine, CalendarDays, Search, ClipboardList, Mail, RotateCcw, CalendarClock, X, FileText, Menu } from "lucide-vue-next";
import { Line } from "vue-chartjs";
import { Chart, LineElement, PointElement, LinearScale, CategoryScale, Filler, Tooltip } from "chart.js";
import { formatBRL } from "@p5wellness/shared";
import { api, ApiError } from "@/lib/api";
import { useAuthStore } from "@/stores/auth";

Chart.register(LineElement, PointElement, LinearScale, CategoryScale, Filler, Tooltip);

const router = useRouter();
const authStore = useAuthStore();
// A role "reports" enxerga dashboard (gráficos), pedidos (vendas, só leitura) e
// relatórios — sem acesso a config de admin (catálogo, equipe, clientes, estorno/reenvio
// de e-mail). O backend também bloqueia essas rotas, isso aqui só evita chamadas fadadas
// a 403 e esconde ações que a role não pode executar.
const isReportsOnly = computed(() => authStore.role === "reports");
const REPORTS_ROLE_TABS = ["dashboard", "pedidos", "relatorios"] as const;
const tab = ref<"dashboard" | "produtos" | "atividades" | "turmas" | "equipe" | "clientes" | "pedidos" | "relatorios">("dashboard");
const mobileNavOpen = ref(false);
function selectTab(next: typeof tab.value) {
  tab.value = next;
  mobileNavOpen.value = false;
}
const navItems = [
  { key: "dashboard", label: "Dashboard", icon: LayoutDashboard },
  { key: "produtos", label: "Produtos", icon: Package },
  { key: "atividades", label: "Atividades", icon: Dumbbell },
  { key: "turmas", label: "Turmas", icon: CalendarDays },
  { key: "equipe", label: "Equipe", icon: Users },
  { key: "clientes", label: "Clientes", icon: Search },
  { key: "pedidos", label: "Pedidos", icon: ClipboardList },
  { key: "relatorios", label: "Relatórios", icon: FileText },
] as const;
const visibleNavItems = computed(() =>
  isReportsOnly.value ? navItems.filter((item) => (REPORTS_ROLE_TABS as readonly string[]).includes(item.key)) : navItems
);

// Filtro de data compartilhado por dashboard, pedidos e relatórios — cada tela mantém seu
// próprio par de refs (o período de uma não deve mudar o da outra) mas monta a querystring
// da mesma forma.
function dateRangeQuery(from: string, to: string) {
  const params = new URLSearchParams();
  if (from) params.set("from", from);
  if (to) params.set("to", to);
  const qs = params.toString();
  return qs ? `?${qs}` : "";
}

// ── Setores (empresas parceiras: P5, AYO...) ──────────────────────────────
interface Vendor { id: string; name: string; slug: string }
const vendors = ref<Vendor[]>([]);
async function loadVendors() {
  const res = await api.get<{ vendors: Vendor[] }>("/admin/vendors");
  vendors.value = res.vendors;
}

// ── Dashboard ──────────────────────────────────────────────────────────────
interface Summary { totalRevenueCents: number; paidOrders: number; entitlementsIssued: number; entitlementsUsed: number; activeStudents: number }
interface SalesPoint { day: string; revenueCents: number; orders: number }
interface SalesByProduct { productTitle: string; quantitySold: number; revenueCents: number }

const summary = ref<Summary | null>(null);
const salesPoints = ref<SalesPoint[]>([]);
const salesByProduct = ref<SalesByProduct[]>([]);
const loadingDashboard = ref(false);
// Vazio = padrão do backend (últimos 14 dias na série, "desde sempre" no resumo/produtos).
const dashboardDateFrom = ref("");
const dashboardDateTo = ref("");
const dashboardDateActive = computed(() => !!dashboardDateFrom.value || !!dashboardDateTo.value);

const chartData = computed(() => ({
  labels: salesPoints.value.map((p) => p.day.slice(5)),
  datasets: [
    {
      label: "Receita (R$)",
      data: salesPoints.value.map((p) => p.revenueCents / 100),
      borderColor: "#d91977",
      backgroundColor: "rgba(217,25,119,0.1)",
      tension: 0.35,
      fill: true,
      pointRadius: 2,
    },
  ],
}));
const chartOptions = { responsive: true, plugins: { legend: { display: false } }, scales: { y: { beginAtZero: true } } };

async function loadDashboard() {
  loadingDashboard.value = true;
  try {
    const qs = dateRangeQuery(dashboardDateFrom.value, dashboardDateTo.value);
    const [s, ts, sp] = await Promise.all([
      api.get<Summary>(`/admin/dashboard/summary${qs}`),
      api.get<{ points: SalesPoint[] }>(`/admin/dashboard/sales-timeseries${qs}`),
      api.get<{ sales: SalesByProduct[] }>(`/admin/dashboard/sales-by-product${qs}`),
    ]);
    summary.value = s;
    salesPoints.value = ts.points;
    salesByProduct.value = sp.sales;
  } finally {
    loadingDashboard.value = false;
  }
}

function clearDashboardDateFilter() {
  dashboardDateFrom.value = "";
  dashboardDateTo.value = "";
  loadDashboard();
}

// ── Activities ─────────────────────────────────────────────────────────────
interface Activity { id: string; title: string; instructor: string; durationMinutes: number; description: string; active: boolean; vendorId: string; vendorName: string }
const activities = ref<Activity[]>([]);
const newActivity = ref({ title: "", instructor: "", durationMinutes: 60, description: "", vendorId: "" });
const savingActivity = ref(false);
const activityError = ref("");

async function loadActivities() {
  const res = await api.get<{ activities: Activity[] }>("/admin/activities");
  activities.value = res.activities;
}

const editingActivityId = ref("");

async function createActivity() {
  activityError.value = "";
  if (!newActivity.value.title || !newActivity.value.vendorId) {
    activityError.value = "Título e setor responsável são obrigatórios.";
    return;
  }
  savingActivity.value = true;
  try {
    if (editingActivityId.value) {
      await api.put(`/admin/activities/${editingActivityId.value}`, newActivity.value);
    } else {
      await api.post("/admin/activities", newActivity.value);
    }
    cancelEditActivity();
    await loadActivities();
  } catch (err) {
    activityError.value = err instanceof ApiError ? err.message : "Não foi possível salvar a atividade.";
  } finally {
    savingActivity.value = false;
  }
}

function editActivity(a: Activity) {
  editingActivityId.value = a.id;
  newActivity.value = { title: a.title, instructor: a.instructor, durationMinutes: a.durationMinutes, description: a.description, vendorId: a.vendorId };
}

function cancelEditActivity() {
  editingActivityId.value = "";
  newActivity.value = { title: "", instructor: "", durationMinutes: 60, description: "", vendorId: "" };
  activityError.value = "";
}

async function toggleActivity(a: Activity) {
  await api.patch(`/admin/activities/${a.id}/active`, { active: !a.active });
  await loadActivities();
}

// ── Products ───────────────────────────────────────────────────────────────
interface Product { id: string; title: string; description: string; type: string; includesBreakfast: boolean; priceCents: number; featured: boolean; active: boolean; chooseOneActivity: boolean; activityIds: string[] }
const products = ref<Product[]>([]);
const newProduct = ref({ title: "", description: "", type: "class", includesBreakfast: false, priceReais: "", featured: false, chooseOneActivity: false, activityIds: [] as string[] });
const savingProduct = ref(false);
const productError = ref("");

async function loadProducts() {
  const res = await api.get<{ products: Product[] }>("/admin/products");
  products.value = res.products;
}

const editingProductId = ref("");

async function createProduct() {
  productError.value = "";
  const priceCents = Math.round(parseFloat(newProduct.value.priceReais.replace(",", ".")) * 100);
  if (!newProduct.value.title || !priceCents) {
    productError.value = "Título e preço são obrigatórios.";
    return;
  }
  savingProduct.value = true;
  try {
    const payload = {
      title: newProduct.value.title,
      description: newProduct.value.description,
      type: newProduct.value.type,
      includesBreakfast: newProduct.value.includesBreakfast,
      priceCents,
      featured: newProduct.value.featured,
      chooseOneActivity: newProduct.value.chooseOneActivity,
      activityIds: newProduct.value.activityIds,
    };
    if (editingProductId.value) {
      await api.put(`/admin/products/${editingProductId.value}`, payload);
    } else {
      await api.post("/admin/products", payload);
    }
    cancelEditProduct();
    await loadProducts();
  } catch (err) {
    productError.value = err instanceof ApiError ? err.message : "Não foi possível salvar.";
  } finally {
    savingProduct.value = false;
  }
}

function editProduct(p: Product) {
  editingProductId.value = p.id;
  newProduct.value = {
    title: p.title, description: p.description, type: p.type, includesBreakfast: p.includesBreakfast,
    priceReais: (p.priceCents / 100).toFixed(2).replace(".", ","), featured: p.featured,
    chooseOneActivity: p.chooseOneActivity, activityIds: [...p.activityIds],
  };
}

function cancelEditProduct() {
  editingProductId.value = "";
  newProduct.value = { title: "", description: "", type: "class", includesBreakfast: false, priceReais: "", featured: false, chooseOneActivity: false, activityIds: [] };
  productError.value = "";
}

async function toggleProduct(p: Product) {
  await api.patch(`/admin/products/${p.id}/active`, { active: !p.active });
  await loadProducts();
}

// ── Turmas (class_sessions) ───────────────────────────────────────────────
interface SessionRow { id: string; activityId: string; activityTitle: string; startsAt: string; endsAt: string; capacity: number; booked: number; status: string }
const sessions = ref<SessionRow[]>([]);
const newSession = ref({ activityId: "", date: "", time: "", durationMinutes: 60, capacity: 20 });
const savingSession = ref(false);
const sessionError = ref("");

async function loadSessions() {
  const res = await api.get<{ sessions: SessionRow[] }>("/admin/sessions");
  sessions.value = res.sessions;
}

const editingSessionId = ref("");

async function createSession() {
  sessionError.value = "";
  if (!newSession.value.activityId || !newSession.value.date || !newSession.value.time || !newSession.value.capacity) {
    sessionError.value = "Atividade, data, horário e vagas são obrigatórios.";
    return;
  }
  const startsAt = new Date(`${newSession.value.date}T${newSession.value.time}:00`);
  const endsAt = new Date(startsAt.getTime() + newSession.value.durationMinutes * 60_000);
  savingSession.value = true;
  try {
    const payload = {
      activityId: newSession.value.activityId,
      startsAt: startsAt.toISOString(),
      endsAt: endsAt.toISOString(),
      capacity: newSession.value.capacity,
    };
    if (editingSessionId.value) {
      await api.put(`/admin/sessions/${editingSessionId.value}`, payload);
    } else {
      await api.post("/admin/sessions", payload);
    }
    cancelEditSession();
    await loadSessions();
  } catch (err) {
    sessionError.value = err instanceof ApiError ? err.message : "Não foi possível salvar a turma.";
  } finally {
    savingSession.value = false;
  }
}

function pad2(n: number) {
  return String(n).padStart(2, "0");
}

function editSession(s: SessionRow) {
  editingSessionId.value = s.id;
  const starts = new Date(s.startsAt);
  const ends = new Date(s.endsAt);
  // Local (not UTC) components — createSession builds startsAt from `${date}T${time}:00`
  // interpreted as local time, so editing must round-trip through the same local fields.
  newSession.value = {
    activityId: s.activityId,
    date: `${starts.getFullYear()}-${pad2(starts.getMonth() + 1)}-${pad2(starts.getDate())}`,
    time: `${pad2(starts.getHours())}:${pad2(starts.getMinutes())}`,
    durationMinutes: Math.round((ends.getTime() - starts.getTime()) / 60_000),
    capacity: s.capacity,
  };
}

function cancelEditSession() {
  editingSessionId.value = "";
  newSession.value = { activityId: "", date: "", time: "", durationMinutes: 60, capacity: 20 };
  sessionError.value = "";
}

async function cancelSession(s: SessionRow) {
  await api.patch(`/admin/sessions/${s.id}/status`, { status: "cancelled" });
  await loadSessions();
}

function formatSessionDate(iso: string) {
  return new Date(iso).toLocaleString("pt-BR", { weekday: "short", day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit" });
}

// ── Team ───────────────────────────────────────────────────────────────────
interface TeamMember { ID: string; Name: string; Email: string; Role: string; Active: boolean; VendorID: string | null; VendorName: string }
const team = ref<TeamMember[]>([]);
const newMember = ref({ name: "", email: "", phone: "", password: "", role: "staff", vendorId: "" });
const savingMember = ref(false);
const memberError = ref("");

async function loadTeam() {
  const res = await api.get<{ members: TeamMember[] }>("/admin/team");
  team.value = res.members;
}

async function createMember() {
  memberError.value = "";
  if (newMember.value.role === "staff" && !newMember.value.vendorId) {
    memberError.value = "Selecione o setor deste funcionário.";
    return;
  }
  savingMember.value = true;
  try {
    await api.post("/admin/team", newMember.value);
    newMember.value = { name: "", email: "", phone: "", password: "", role: "staff", vendorId: "" };
    await loadTeam();
  } catch (err) {
    memberError.value = err instanceof ApiError ? err.message : "Não foi possível criar o acesso.";
  } finally {
    savingMember.value = false;
  }
}

async function toggleMember(m: TeamMember) {
  await api.patch(`/admin/team/${m.ID}/active`, { active: !m.Active });
  await loadTeam();
}

// ── Clientes ───────────────────────────────────────────────────────────────
interface CustomerSummary { id: string; fullName: string; email: string; phone: string; cpfLast4: string; ordersCount: number; createdAt: string }
interface CustomerTicket {
  id: string; label: string; vendorName: string; status: string;
  validFrom: string; validUntil: string; issuedAt: string; usedAt: string | null; usedByName?: string;
}
interface CustomerOrder {
  id: string; orderNumber: string; status: string; totalCents: number; paymentMethod: string;
  productTitle: string; createdAt: string; paidAt?: string; tickets: CustomerTicket[];
}
interface CustomerDetail extends CustomerSummary { orders: CustomerOrder[] }

function paymentMethodLabel(method: string) {
  return { pix: "Pix", credit_card: "Cartão", boleto: "Boleto" }[method] ?? "—";
}
function ticketStatusLabel(status: string) {
  return { available: "Disponível", used: "Utilizado", expired: "Expirado", cancelled: "Cancelado" }[status] ?? status;
}
function formatDateTime(iso?: string | null) {
  if (!iso) return "";
  return new Date(iso).toLocaleString("pt-BR", { day: "2-digit", month: "2-digit", year: "numeric", hour: "2-digit", minute: "2-digit" });
}
function formatDateOnly(isoDate?: string | null) {
  if (!isoDate) return "";
  const [y, m, d] = isoDate.split("-");
  return `${d}/${m}/${y}`;
}

const customerQuery = ref("");
const customers = ref<CustomerSummary[]>([]);
const searchingCustomers = ref(false);
const customerSearchError = ref("");
const selectedCustomer = ref<CustomerDetail | null>(null);
const anonymizing = ref(false);

async function searchCustomers() {
  customerSearchError.value = "";
  const q = customerQuery.value.trim();
  if (q.length === 1) {
    customerSearchError.value = "Digite ao menos 2 caracteres.";
    return;
  }
  searchingCustomers.value = true;
  try {
    const res = await api.get<{ customers: CustomerSummary[] }>(`/admin/customers?q=${encodeURIComponent(q)}`);
    customers.value = res.customers;
  } catch (err) {
    customerSearchError.value = err instanceof ApiError ? err.message : "Não foi possível buscar.";
  } finally {
    searchingCustomers.value = false;
  }
}

async function openCustomer(c: CustomerSummary) {
  selectedCustomer.value = await api.get<CustomerDetail>(`/admin/customers/${c.id}`);
}

async function anonymizeCustomer(c: CustomerDetail) {
  if (!window.confirm(`Anonimizar o cadastro de ${c.fullName}? Essa ação apaga nome, e-mail, telefone e CPF permanentemente — o histórico de pedidos é mantido, mas o cliente não poderá mais entrar.`)) return;
  anonymizing.value = true;
  try {
    await api.post(`/admin/customers/${c.id}/anonymize`);
    selectedCustomer.value = null;
    await searchCustomers();
  } finally {
    anonymizing.value = false;
  }
}

// ── Pedidos ────────────────────────────────────────────────────────────────
interface OrderRow { id: string; orderNumber: string; status: string; totalCents: number; studentName: string; studentEmail: string; createdAt: string }
interface OrderReschedule { reason: string; previousDate: string; newDate: string; changedByName: string; createdAt: string }
interface OrderDetail extends OrderRow {
  asaasPaymentId: string;
  items: { productTitle: string; activityTitle: string | null; benefitType: string; sessionStartsAt: string | null }[];
  entitlements: { id: string; label: string; vendorName: string; status: string; validFrom: string; validUntil: string; issuedAt: string; usedAt: string | null; usedByName?: string }[];
  reschedules: OrderReschedule[];
}

const orderStatusFilter = ref("");
const orderQuery = ref("");
const orderDateFrom = ref("");
const orderDateTo = ref("");
const orderDateActive = computed(() => !!orderDateFrom.value || !!orderDateTo.value);
const orders = ref<OrderRow[]>([]);
const loadingOrders = ref(false);
const selectedOrder = ref<OrderDetail | null>(null);
const orderActionError = ref("");
const orderActionBusy = ref(false);

function orderStatusLabel(status: string) {
  return { paid: "Pago", pending: "Aguardando pagamento", refunded: "Estornado", failed: "Falhou", expired: "Expirado", cancelled: "Cancelado" }[status] ?? status;
}

async function loadOrders() {
  loadingOrders.value = true;
  try {
    const params = new URLSearchParams();
    if (orderStatusFilter.value) params.set("status", orderStatusFilter.value);
    if (orderQuery.value.trim()) params.set("q", orderQuery.value.trim());
    if (orderDateFrom.value) params.set("from", orderDateFrom.value);
    if (orderDateTo.value) params.set("to", orderDateTo.value);
    const res = await api.get<{ orders: OrderRow[] }>(`/admin/orders?${params.toString()}`);
    orders.value = res.orders;
  } finally {
    loadingOrders.value = false;
  }
}

function clearOrderDateFilter() {
  orderDateFrom.value = "";
  orderDateTo.value = "";
  loadOrders();
}

async function openOrder(o: OrderRow) {
  orderActionError.value = "";
  resendAltEmailOpen.value = false;
  resendAltEmail.value = "";
  selectedOrder.value = await api.get<OrderDetail>(`/admin/orders/${o.id}`);
}

const resendAltEmailOpen = ref(false);
const resendAltEmail = ref("");

async function resendOrderEmail(o: OrderDetail, email?: string) {
  orderActionError.value = "";
  orderActionBusy.value = true;
  try {
    await api.post(`/admin/orders/${o.id}/resend-email`, email ? { email } : undefined);
    if (email) {
      resendAltEmailOpen.value = false;
      resendAltEmail.value = "";
    }
  } catch (err) {
    orderActionError.value = err instanceof ApiError ? err.message : "Não foi possível reenviar.";
  } finally {
    orderActionBusy.value = false;
  }
}

async function refundOrder(o: OrderDetail) {
  if (!window.confirm(`Estornar o pedido ${o.orderNumber}? O pagamento será revertido na Asaas e os benefícios ainda não utilizados serão cancelados. Essa ação não pode ser desfeita.`)) return;
  orderActionError.value = "";
  orderActionBusy.value = true;
  try {
    await api.post(`/admin/orders/${o.id}/refund`);
    await openOrder(o);
    await loadOrders();
  } catch (err) {
    orderActionError.value = err instanceof ApiError ? err.message : "Não foi possível estornar.";
  } finally {
    orderActionBusy.value = false;
  }
}

// ── Remarcar pedido (não compareceu, pediu outra data) ───────────────────────
const rescheduleModalOpen = ref(false);
const rescheduleDate = ref("");
const rescheduleReason = ref("");
const rescheduleError = ref("");
const reschedulingOrder = ref(false);

function openReschedule() {
  rescheduleDate.value = "";
  rescheduleReason.value = "";
  rescheduleError.value = "";
  rescheduleModalOpen.value = true;
}

async function submitReschedule(o: OrderDetail) {
  if (!rescheduleDate.value || !rescheduleReason.value.trim()) {
    rescheduleError.value = "Informe a nova data e o motivo.";
    return;
  }
  rescheduleError.value = "";
  reschedulingOrder.value = true;
  try {
    await api.post(`/admin/orders/${o.id}/reschedule`, { newDate: rescheduleDate.value, reason: rescheduleReason.value.trim() });
    rescheduleModalOpen.value = false;
    await openOrder(o);
    await loadOrders();
  } catch (err) {
    rescheduleError.value = err instanceof ApiError ? err.message : "Não foi possível remarcar.";
  } finally {
    reschedulingOrder.value = false;
  }
}

// ── Relatórios ─────────────────────────────────────────────────────────────
interface Attendee { name: string; email: string; orderNumber: string }
interface SessionRoster { sessionId: string; activityTitle: string; startsAt: string; capacity: number; attendees: Attendee[] }
interface ProductRoster { productId: string; title: string; buyers: Attendee[] }
interface ActivityRoster { label: string; attendees: Attendee[] }

const reportView = ref<"turma" | "produto" | "atividade">("turma");
const sessionRosters = ref<SessionRoster[]>([]);
const productRosters = ref<ProductRoster[]>([]);
const activityRosters = ref<ActivityRoster[]>([]);
const loadingReports = ref(false);
// "por turma" filtra pela data da própria aula; "por produto"/"por atividade" filtram
// pela data da compra — ver comentários no repositório (admin_reports.go) do porquê.
const reportDateFrom = ref("");
const reportDateTo = ref("");
const reportDateActive = computed(() => !!reportDateFrom.value || !!reportDateTo.value);

async function loadReports() {
  loadingReports.value = true;
  try {
    const qs = dateRangeQuery(reportDateFrom.value, reportDateTo.value);
    const [sessionsRes, productsRes, activitiesRes] = await Promise.all([
      api.get<{ sessions: SessionRoster[] }>(`/admin/reports/sessions${qs}`),
      api.get<{ products: ProductRoster[] }>(`/admin/reports/products${qs}`),
      api.get<{ activities: ActivityRoster[] }>(`/admin/reports/activities${qs}`),
    ]);
    sessionRosters.value = sessionsRes.sessions;
    productRosters.value = productsRes.products;
    activityRosters.value = activitiesRes.activities;
  } finally {
    loadingReports.value = false;
  }
}

function clearReportDateFilter() {
  reportDateFrom.value = "";
  reportDateTo.value = "";
  loadReports();
}

function formatSessionDateTime(iso: string) {
  return new Date(iso).toLocaleString("pt-BR", { weekday: "short", day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit" });
}

function handleLogout() {
  authStore.logout();
  router.push("/acesso-admin");
}

onMounted(async () => {
  if (isReportsOnly.value) {
    // O backend nega o resto (catálogo, equipe, clientes) para essa role — nem tenta,
    // pra não gerar 403 à toa.
    await Promise.all([loadDashboard(), loadOrders(), loadReports()]);
    return;
  }
  await Promise.all([loadDashboard(), loadVendors(), loadActivities(), loadProducts(), loadTeam(), loadSessions(), loadOrders(), loadReports(), searchCustomers()]);
});
</script>

<template>
  <div class="min-h-screen bg-paper">
    <!-- Mobile top bar: the sidebar below is desktop-only (md:flex), so this is the
         only way team members on a phone can switch tabs or log out. -->
    <header class="sticky top-0 z-30 flex items-center justify-between border-b border-line bg-ink px-4 py-3 text-white md:hidden">
      <div class="flex items-center gap-2">
        <span class="font-serif text-lg font-bold">P5</span>
        <span class="h-4 w-px bg-white/20"></span>
        <span class="font-mono text-xs uppercase tracking-widest text-magenta">Admin</span>
      </div>
      <button class="rounded-lg p-2 text-white/80 hover:bg-white/10" aria-label="Abrir menu" @click="mobileNavOpen = true">
        <Menu :size="22" />
      </button>
    </header>

    <div v-if="mobileNavOpen" class="fixed inset-0 z-40 bg-ink/50 md:hidden" @click="mobileNavOpen = false"></div>
    <aside
      :class="['fixed inset-y-0 left-0 z-50 flex w-64 -translate-x-full flex-col bg-ink px-4 py-6 text-white transition-transform duration-200 md:hidden', mobileNavOpen ? 'translate-x-0' : '']"
    >
      <div class="mb-8 flex items-center justify-between px-2">
        <div class="flex items-center gap-2">
          <span class="font-serif text-lg font-bold">P5</span>
          <span class="h-4 w-px bg-white/20"></span>
          <span class="font-mono text-xs uppercase tracking-widest text-magenta">Admin</span>
        </div>
        <button class="rounded-lg p-1.5 text-white/70 hover:bg-white/10" aria-label="Fechar menu" @click="mobileNavOpen = false">
          <X :size="18" />
        </button>
      </div>
      <nav class="flex flex-col gap-1 text-sm">
        <button v-for="item in visibleNavItems" :key="item.key" :class="['flex items-center gap-2 rounded-lg px-3 py-2 text-left', tab === item.key ? 'bg-white/10 font-semibold' : 'text-white/70 hover:bg-white/5']" @click="selectTab(item.key)">
          <component :is="item.icon" :size="16" /> {{ item.label }}
        </button>
        <a href="/check-in" class="mt-2 flex items-center gap-2 rounded-lg px-3 py-2 text-left text-white/70 hover:bg-white/5">
          <ScanLine :size="16" /> Terminal de check-in
        </a>
      </nav>
      <button class="mt-auto flex items-center gap-2 rounded-lg px-3 py-2 text-left text-sm text-white/70 hover:bg-white/5" @click="handleLogout">
        <LogOut :size="16" /> Sair
      </button>
    </aside>

    <div class="flex">
    <aside class="hidden w-56 shrink-0 flex-col border-r border-line bg-ink px-4 py-6 text-white md:flex sticky top-0 h-screen">
      <div class="mb-8 flex items-center gap-2 px-2">
          <span class="font-serif text-lg font-bold">P5</span>
          <span class="h-4 w-px bg-white/20"></span>
          <span class="font-mono text-xs uppercase tracking-widest text-magenta">Admin</span>
        </div>
        <nav class="flex flex-col gap-1 text-sm">
          <button v-for="item in visibleNavItems" :key="item.key" :class="['flex items-center gap-2 rounded-lg px-3 py-2 text-left', tab === item.key ? 'bg-white/10 font-semibold' : 'text-white/70 hover:bg-white/5']" @click="selectTab(item.key)">
            <component :is="item.icon" :size="16" /> {{ item.label }}
          </button>
          <a v-if="!isReportsOnly" href="/check-in" class="mt-2 flex items-center gap-2 rounded-lg px-3 py-2 text-left text-white/70 hover:bg-white/5">
            <ScanLine :size="16" /> Terminal de check-in
          </a>
        </nav>
        <button class="mt-auto flex items-center gap-2 rounded-lg px-3 py-2 text-left text-sm text-white/70 hover:bg-white/5" @click="handleLogout">
          <LogOut :size="16" /> Sair
        </button>
      </aside>

      <main class="flex-1 px-6 py-8 md:px-10">
        <!-- DASHBOARD -->
        <section v-if="tab === 'dashboard'">
          <div class="flex flex-wrap items-end justify-between gap-3">
            <h1 class="font-serif text-2xl font-bold text-ink">Visão geral</h1>
            <div class="flex flex-wrap items-end gap-2">
              <div>
                <label class="mb-1 block text-xs font-semibold uppercase tracking-wider text-ink-soft">De</label>
                <input v-model="dashboardDateFrom" type="date" class="rounded-lg border border-line px-3 py-1.5 text-sm" />
              </div>
              <div>
                <label class="mb-1 block text-xs font-semibold uppercase tracking-wider text-ink-soft">Até</label>
                <input v-model="dashboardDateTo" type="date" class="rounded-lg border border-line px-3 py-1.5 text-sm" />
              </div>
              <button class="button-magenta px-4 py-1.5 text-sm" :disabled="loadingDashboard" @click="loadDashboard">Filtrar</button>
              <button v-if="dashboardDateActive" type="button" class="text-xs font-semibold text-ink-soft hover:text-ink" @click="clearDashboardDateFilter">Limpar</button>
            </div>
          </div>
          <p class="mt-1 text-xs text-ink-soft">Sem filtro, mostra os últimos 14 dias na evolução e o total desde sempre nos outros números.</p>

          <div v-if="summary" class="mt-6 grid grid-cols-2 gap-4 md:grid-cols-5">
            <div class="rounded-[var(--radius-card)] border border-line bg-white p-4">
              <p class="text-xs text-ink-soft">Receita total</p>
              <p class="mt-1 font-mono text-xl font-semibold text-ink">{{ formatBRL(summary.totalRevenueCents) }}</p>
            </div>
            <div class="rounded-[var(--radius-card)] border border-line bg-white p-4">
              <p class="text-xs text-ink-soft">Pedidos pagos</p>
              <p class="mt-1 font-mono text-xl font-semibold text-ink">{{ summary.paidOrders }}</p>
            </div>
            <div class="rounded-[var(--radius-card)] border border-line bg-white p-4">
              <p class="text-xs text-ink-soft">Benefícios emitidos</p>
              <p class="mt-1 font-mono text-xl font-semibold text-ink">{{ summary.entitlementsIssued }}</p>
            </div>
            <div class="rounded-[var(--radius-card)] border border-line bg-white p-4">
              <p class="text-xs text-ink-soft">Já validados</p>
              <p class="mt-1 font-mono text-xl font-semibold text-ink">{{ summary.entitlementsUsed }}</p>
            </div>
            <div class="rounded-[var(--radius-card)] border border-line bg-white p-4">
              <p class="text-xs text-ink-soft">Alunos ativos</p>
              <p class="mt-1 font-mono text-xl font-semibold text-ink">{{ summary.activeStudents }}</p>
            </div>
          </div>

          <div class="mt-6 rounded-[var(--radius-card)] border border-line bg-white p-6">
            <h2 class="mb-4 font-serif text-lg font-semibold text-ink">Evolução de vendas</h2>
            <Line :data="chartData" :options="chartOptions" />
          </div>

          <div class="mt-6 rounded-[var(--radius-card)] border border-line bg-white p-6">
            <h2 class="mb-4 font-serif text-lg font-semibold text-ink">Quantidades vendidas por produto</h2>
            <table class="w-full text-sm">
              <thead>
                <tr class="border-b border-line text-left text-ink-soft">
                  <th class="pb-2">Produto</th>
                  <th class="pb-2">Quantidade vendida</th>
                  <th class="pb-2">Receita</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="s in salesByProduct" :key="s.productTitle" class="border-b border-line/50">
                  <td class="py-2 text-ink">{{ s.productTitle }}</td>
                  <td class="py-2 font-mono text-ink-soft">{{ s.quantitySold }}</td>
                  <td class="py-2 font-mono text-ink">{{ formatBRL(s.revenueCents) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- PRODUTOS -->
        <section v-else-if="tab === 'produtos'">
          <h1 class="font-serif text-2xl font-bold text-ink">Produtos</h1>

          <div class="mt-6 rounded-[var(--radius-card)] border border-line bg-white p-6">
            <h2 class="mb-4 font-serif text-lg font-semibold text-ink">{{ editingProductId ? "Editar oferta" : "Nova oferta" }}</h2>
            <div class="grid gap-4 md:grid-cols-2">
              <input v-model="newProduct.title" placeholder="Título (ex: Aulas + Café)" class="rounded-lg border border-line px-3 py-2 text-sm" />
              <input v-model="newProduct.priceReais" placeholder="Preço (ex: 90,00)" class="rounded-lg border border-line px-3 py-2 text-sm" />
              <input v-model="newProduct.description" placeholder="Descrição" class="rounded-lg border border-line px-3 py-2 text-sm md:col-span-2" />
              <select v-model="newProduct.type" class="rounded-lg border border-line px-3 py-2 text-sm">
                <option value="class">Aula avulsa</option>
                <option value="combo">Combo</option>
              </select>
              <div class="flex flex-wrap items-center gap-4 text-sm text-ink-soft">
                <label class="flex items-center gap-2"><input v-model="newProduct.includesBreakfast" type="checkbox" /> Inclui café da manhã</label>
                <label class="flex items-center gap-2"><input v-model="newProduct.featured" type="checkbox" /> Destacar (mais popular)</label>
              </div>
              <div class="md:col-span-2">
                <p class="mb-2 text-xs font-medium text-ink-soft">Atividades incluídas</p>
                <div class="flex flex-wrap gap-3">
                  <label v-for="a in activities" :key="a.id" class="flex items-center gap-2 rounded-full border border-line px-3 py-1.5 text-sm">
                    <input v-model="newProduct.activityIds" type="checkbox" :value="a.id" /> {{ a.title }}
                  </label>
                </div>
              </div>
              <div v-if="newProduct.activityIds.length >= 2" class="md:col-span-2">
                <label class="flex items-center gap-2 text-sm text-ink-soft">
                  <input v-model="newProduct.chooseOneActivity" type="checkbox" />
                  Cliente escolhe <strong class="text-ink">uma</strong> das atividades acima (em vez de reservar todas) — ex: "Yoga ou HYROX"
                </label>
              </div>
            </div>
            <p v-if="productError" class="mt-3 text-sm text-red-600">{{ productError }}</p>
            <div class="mt-4 flex gap-3">
              <button class="button-magenta" :disabled="savingProduct" @click="createProduct">
                <Plus :size="16" /> {{ editingProductId ? "Salvar alterações" : "Publicar produto" }}
              </button>
              <button v-if="editingProductId" type="button" class="text-sm font-semibold text-ink-soft hover:text-ink" @click="cancelEditProduct">Cancelar</button>
            </div>
          </div>

          <div class="mt-6 grid gap-4 md:grid-cols-2">
            <div v-for="p in products" :key="p.id" class="rounded-[var(--radius-card)] border border-line bg-white p-5">
              <div class="flex items-start justify-between">
                <div>
                  <h3 class="font-serif text-lg font-semibold text-ink">{{ p.title }}</h3>
                  <p class="text-xs text-ink-soft">{{ p.description }}</p>
                  <span v-if="p.chooseOneActivity" class="mt-1 inline-block rounded-full bg-[#eff7f8] px-2 py-0.5 text-[10px] font-semibold text-[#1A6FA8]">Cliente escolhe 1 atividade</span>
                </div>
                <span :class="['rounded-full px-2 py-1 text-xs font-semibold', p.active ? 'bg-[#e5f5e8] text-[#237438]' : 'bg-warm text-ink-soft']">{{ p.active ? "Ativo" : "Inativo" }}</span>
              </div>
              <p class="mt-2 font-mono text-lg font-semibold text-ink">{{ formatBRL(p.priceCents) }}</p>
              <div class="mt-3 flex gap-3">
                <button class="text-xs font-semibold text-magenta" @click="toggleProduct(p)">{{ p.active ? "Desativar" : "Ativar" }}</button>
                <button class="text-xs font-semibold text-ink-soft hover:text-ink" @click="editProduct(p)">Editar</button>
              </div>
            </div>
          </div>
        </section>

        <!-- ATIVIDADES -->
        <section v-else-if="tab === 'atividades'">
          <h1 class="font-serif text-2xl font-bold text-ink">Atividades</h1>

          <div class="mt-6 rounded-[var(--radius-card)] border border-line bg-white p-6">
            <h2 class="mb-4 font-serif text-lg font-semibold text-ink">{{ editingActivityId ? "Editar atividade" : "Nova atividade" }}</h2>
            <div class="grid gap-4 md:grid-cols-2">
              <input v-model="newActivity.title" placeholder="Título (ex: Pilates)" class="rounded-lg border border-line px-3 py-2 text-sm" />
              <input v-model="newActivity.instructor" placeholder="Instrutor" class="rounded-lg border border-line px-3 py-2 text-sm" />
              <input v-model.number="newActivity.durationMinutes" type="number" placeholder="Duração (minutos)" class="rounded-lg border border-line px-3 py-2 text-sm" />
              <select v-model="newActivity.vendorId" class="rounded-lg border border-line px-3 py-2 text-sm">
                <option value="" disabled>Setor responsável...</option>
                <option v-for="v in vendors" :key="v.id" :value="v.id">{{ v.name }}</option>
              </select>
              <input v-model="newActivity.description" placeholder="Descrição" class="rounded-lg border border-line px-3 py-2 text-sm md:col-span-2" />
            </div>
            <p v-if="activityError" class="mt-3 text-sm text-red-600">{{ activityError }}</p>
            <div class="mt-4 flex gap-3">
              <button class="button-magenta" :disabled="savingActivity" @click="createActivity">
                <Plus :size="16" /> {{ editingActivityId ? "Salvar alterações" : "Criar atividade" }}
              </button>
              <button v-if="editingActivityId" type="button" class="text-sm font-semibold text-ink-soft hover:text-ink" @click="cancelEditActivity">Cancelar</button>
            </div>
          </div>

          <div class="mt-6 grid gap-4 md:grid-cols-2">
            <div v-for="a in activities" :key="a.id" class="rounded-[var(--radius-card)] border border-line bg-white p-5">
              <div class="flex items-start justify-between">
                <div>
                  <h3 class="font-serif text-lg font-semibold text-ink">{{ a.title }}</h3>
                  <p class="text-xs text-ink-soft">{{ a.instructor }} · {{ a.durationMinutes }} min</p>
                </div>
                <span :class="['rounded-full px-2 py-1 text-xs font-semibold', a.active ? 'bg-[#e5f5e8] text-[#237438]' : 'bg-warm text-ink-soft']">{{ a.active ? "Ativa" : "Inativa" }}</span>
              </div>
              <span class="mt-2 inline-flex w-fit items-center rounded-full bg-[#eff7f8] px-2.5 py-1 text-xs font-semibold text-[#1A6FA8]">{{ a.vendorName }}</span>
              <div class="mt-3 flex gap-3">
                <button class="text-xs font-semibold text-magenta" @click="toggleActivity(a)">{{ a.active ? "Desativar" : "Ativar" }}</button>
                <button class="text-xs font-semibold text-ink-soft hover:text-ink" @click="editActivity(a)">Editar</button>
              </div>
            </div>
          </div>
        </section>

        <!-- TURMAS -->
        <section v-else-if="tab === 'turmas'">
          <h1 class="font-serif text-2xl font-bold text-ink">Turmas</h1>
          <p class="mt-1 text-sm text-ink-soft">As datas cadastradas aqui aparecem para o cliente escolher no checkout ("Selecione sua data").</p>

          <div class="mt-6 rounded-[var(--radius-card)] border border-line bg-white p-6">
            <h2 class="mb-4 font-serif text-lg font-semibold text-ink">{{ editingSessionId ? "Editar turma" : "Nova turma" }}</h2>
            <div class="grid gap-4 md:grid-cols-2">
              <select v-model="newSession.activityId" class="rounded-lg border border-line px-3 py-2 text-sm">
                <option value="" disabled>Atividade...</option>
                <option v-for="a in activities" :key="a.id" :value="a.id">{{ a.title }} · {{ a.vendorName }}</option>
              </select>
              <input v-model.number="newSession.capacity" type="number" min="1" placeholder="Vagas" class="rounded-lg border border-line px-3 py-2 text-sm" />
              <input v-model="newSession.date" type="date" class="rounded-lg border border-line px-3 py-2 text-sm" />
              <input v-model="newSession.time" type="time" class="rounded-lg border border-line px-3 py-2 text-sm" />
              <div>
                <label class="mb-1 block text-xs font-medium text-ink-soft">Duração (minutos)</label>
                <input v-model.number="newSession.durationMinutes" type="number" min="15" step="15" class="w-full rounded-lg border border-line px-3 py-2 text-sm" />
              </div>
            </div>
            <p v-if="sessionError" class="mt-3 text-sm text-red-600">{{ sessionError }}</p>
            <div class="mt-4 flex gap-3">
              <button class="button-magenta" :disabled="savingSession" @click="createSession">
                <Plus :size="16" /> {{ editingSessionId ? "Salvar alterações" : "Criar turma" }}
              </button>
              <button v-if="editingSessionId" type="button" class="text-sm font-semibold text-ink-soft hover:text-ink" @click="cancelEditSession">Cancelar</button>
            </div>
          </div>

          <div class="mt-6 overflow-hidden rounded-[var(--radius-card)] border border-line bg-white">
            <table class="w-full text-sm">
              <thead class="bg-warm/60">
                <tr class="text-left text-ink-soft">
                  <th class="px-5 py-3">Atividade</th>
                  <th class="px-5 py-3">Quando</th>
                  <th class="px-5 py-3">Ocupação</th>
                  <th class="px-5 py-3">Status</th>
                  <th class="px-5 py-3"></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="s in sessions" :key="s.id" class="border-t border-line">
                  <td class="px-5 py-3 text-ink">{{ s.activityTitle }}</td>
                  <td class="px-5 py-3 text-ink-soft">{{ formatSessionDate(s.startsAt) }}</td>
                  <td class="px-5 py-3 font-mono text-ink-soft">{{ s.booked }}/{{ s.capacity }}</td>
                  <td class="px-5 py-3">
                    <span :class="['rounded-full px-2 py-1 text-xs font-semibold', s.status === 'scheduled' ? 'bg-[#e5f5e8] text-[#237438]' : 'bg-warm text-ink-soft']">
                      {{ s.status === "scheduled" ? "Agendada" : s.status === "cancelled" ? "Cancelada" : "Concluída" }}
                    </span>
                  </td>
                  <td class="px-5 py-3">
                    <div class="flex gap-3">
                      <button v-if="s.status === 'scheduled'" class="text-xs font-semibold text-ink-soft hover:text-ink" @click="editSession(s)">Editar</button>
                      <button v-if="s.status === 'scheduled'" class="text-xs font-semibold text-magenta" @click="cancelSession(s)">Cancelar</button>
                    </div>
                  </td>
                </tr>
                <tr v-if="!sessions.length">
                  <td colspan="5" class="px-5 py-8 text-center text-ink-soft">Nenhuma turma cadastrada ainda.</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- EQUIPE -->
        <section v-else-if="tab === 'equipe'">
          <h1 class="font-serif text-2xl font-bold text-ink">Equipe</h1>

          <div class="mt-6 rounded-[var(--radius-card)] border border-line bg-white p-6">
            <h2 class="mb-4 font-serif text-lg font-semibold text-ink">Novo acesso</h2>
            <div class="grid gap-4 md:grid-cols-2">
              <input v-model="newMember.name" placeholder="Nome" class="rounded-lg border border-line px-3 py-2 text-sm" />
              <input v-model="newMember.email" type="email" placeholder="E-mail" class="rounded-lg border border-line px-3 py-2 text-sm" />
              <input v-model="newMember.phone" placeholder="Telefone" class="rounded-lg border border-line px-3 py-2 text-sm" />
              <input v-model="newMember.password" type="password" placeholder="Senha inicial" class="rounded-lg border border-line px-3 py-2 text-sm" />
              <select v-model="newMember.role" class="rounded-lg border border-line px-3 py-2 text-sm">
                <option value="staff">Funcionário — check-in e operação</option>
                <option value="admin">Administrador — gestão completa</option>
                <option value="reports">Relatórios — só vê os relatórios</option>
              </select>
              <select v-if="newMember.role === 'staff'" v-model="newMember.vendorId" class="rounded-lg border border-line px-3 py-2 text-sm">
                <option value="" disabled>Setor deste funcionário...</option>
                <option v-for="v in vendors" :key="v.id" :value="v.id">{{ v.name }}</option>
              </select>
              <p v-else-if="newMember.role === 'admin'" class="self-center text-xs text-ink-soft">Administrador enxerga e valida todos os setores.</p>
              <p v-else class="self-center text-xs text-ink-soft">Enxerga todos os relatórios, sem acesso à configuração do admin.</p>
            </div>
            <p v-if="memberError" class="mt-3 text-sm text-red-600">{{ memberError }}</p>
            <button class="button-magenta mt-4" :disabled="savingMember" @click="createMember">
              <Plus :size="16" /> Criar acesso
            </button>
          </div>

          <div class="mt-6 overflow-hidden rounded-[var(--radius-card)] border border-line bg-white">
            <table class="w-full text-sm">
              <thead class="bg-warm/60">
                <tr class="text-left text-ink-soft">
                  <th class="px-5 py-3">Nome</th>
                  <th class="px-5 py-3">E-mail</th>
                  <th class="px-5 py-3">Papel</th>
                  <th class="px-5 py-3">Setor</th>
                  <th class="px-5 py-3">Status</th>
                  <th class="px-5 py-3"></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="m in team" :key="m.ID" class="border-t border-line">
                  <td class="px-5 py-3 text-ink">{{ m.Name }}</td>
                  <td class="px-5 py-3 text-ink-soft">{{ m.Email }}</td>
                  <td class="px-5 py-3 text-ink-soft">{{ m.Role === "admin" ? "Administrador" : m.Role === "reports" ? "Relatórios" : "Funcionário" }}</td>
                  <td class="px-5 py-3 text-ink-soft">{{ m.VendorName || "Todos" }}</td>
                  <td class="px-5 py-3">
                    <span :class="['rounded-full px-2 py-1 text-xs font-semibold', m.Active ? 'bg-[#e5f5e8] text-[#237438]' : 'bg-warm text-ink-soft']">{{ m.Active ? "Ativo" : "Inativo" }}</span>
                  </td>
                  <td class="px-5 py-3">
                    <button class="text-xs font-semibold text-magenta" @click="toggleMember(m)">{{ m.Active ? "Desativar" : "Ativar" }}</button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- CLIENTES -->
        <section v-else-if="tab === 'clientes'">
          <h1 class="font-serif text-2xl font-bold text-ink">Clientes</h1>
          <p class="mt-1 text-sm text-ink-soft">Todos os clientes cadastrados. Busque por nome, e-mail, telefone ou os últimos 4 dígitos do CPF — útil quando um cliente diz que pagou mas não achou o e-mail.</p>

          <form class="mt-6 flex gap-3" @submit.prevent="searchCustomers">
            <input v-model="customerQuery" placeholder="Nome, e-mail, telefone ou CPF..." class="flex-1 rounded-lg border border-line px-3 py-2 text-sm" />
            <button type="submit" class="button-magenta" :disabled="searchingCustomers">
              <Search :size="16" /> Buscar
            </button>
          </form>
          <p v-if="customerSearchError" class="mt-2 text-sm text-red-600">{{ customerSearchError }}</p>

          <div class="mt-6 overflow-hidden rounded-[var(--radius-card)] border border-line bg-white">
            <table class="w-full text-sm">
              <thead class="bg-warm/60">
                <tr class="text-left text-ink-soft">
                  <th class="px-5 py-3">Nome</th>
                  <th class="px-5 py-3">E-mail</th>
                  <th class="px-5 py-3">Telefone</th>
                  <th class="px-5 py-3">CPF</th>
                  <th class="px-5 py-3">Pedidos</th>
                  <th class="px-5 py-3"></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="c in customers" :key="c.id" class="border-t border-line">
                  <td class="px-5 py-3 text-ink">{{ c.fullName }}</td>
                  <td class="px-5 py-3 text-ink-soft">{{ c.email }}</td>
                  <td class="px-5 py-3 text-ink-soft">{{ c.phone || "—" }}</td>
                  <td class="px-5 py-3 font-mono text-ink-soft">{{ c.cpfLast4 ? `···${c.cpfLast4}` : "—" }}</td>
                  <td class="px-5 py-3 font-mono text-ink-soft">{{ c.ordersCount }}</td>
                  <td class="px-5 py-3">
                    <button class="text-xs font-semibold text-magenta" @click="openCustomer(c)">Ver detalhes</button>
                  </td>
                </tr>
                <tr v-if="!customers.length && !searchingCustomers">
                  <td colspan="6" class="px-5 py-8 text-center text-ink-soft">Nenhum cliente encontrado.</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div v-if="selectedCustomer" class="fixed inset-0 z-50 flex items-center justify-center bg-ink/40 p-6" @click.self="selectedCustomer = null">
            <div class="max-h-[85vh] w-full max-w-2xl overflow-y-auto rounded-[var(--radius-card)] bg-white p-6">
              <div class="flex items-start justify-between">
                <div>
                  <h3 class="font-serif text-lg font-semibold text-ink">{{ selectedCustomer.fullName }}</h3>
                  <p class="text-xs text-ink-soft">{{ selectedCustomer.email }} · {{ selectedCustomer.phone || "sem telefone" }} · cliente desde {{ formatDateOnly(selectedCustomer.createdAt.slice(0, 10)) }}</p>
                </div>
                <button class="text-ink-soft hover:text-ink" @click="selectedCustomer = null"><X :size="18" /></button>
              </div>

              <h4 class="mt-4 mb-2 text-xs font-bold uppercase tracking-wider text-ink-soft">Pedidos</h4>
              <div class="max-h-[55vh] space-y-3 overflow-y-auto">
                <div v-for="o in selectedCustomer.orders" :key="o.id" class="rounded-lg border border-line p-3">
                  <div class="flex flex-wrap items-start justify-between gap-2">
                    <div>
                      <p class="text-sm font-semibold text-ink">{{ o.productTitle || o.orderNumber }} · {{ formatBRL(o.totalCents) }}</p>
                      <p class="text-xs text-ink-soft">{{ o.orderNumber }} · {{ paymentMethodLabel(o.paymentMethod) }}</p>
                    </div>
                    <span :class="['rounded-full px-2 py-1 text-xs font-semibold', o.status === 'paid' ? 'bg-[#e5f5e8] text-[#237438]' : o.status === 'refunded' ? 'bg-[#fdeaea] text-[#b3261e]' : 'bg-warm text-ink-soft']">
                      {{ orderStatusLabel(o.status) }}
                    </span>
                  </div>
                  <p class="mt-1 text-xs text-ink-soft">
                    Criado em {{ formatDateTime(o.createdAt) }}
                    <template v-if="o.paidAt"> · pago em {{ formatDateTime(o.paidAt) }}</template>
                  </p>

                  <div v-if="o.tickets.length" class="mt-2 space-y-1.5 border-t border-line pt-2">
                    <div v-for="t in o.tickets" :key="t.id" class="text-xs">
                      <div class="flex items-center justify-between gap-2">
                        <span class="font-medium text-ink">{{ t.label }} · {{ t.vendorName }}</span>
                        <span :class="['shrink-0 rounded-full px-2 py-0.5 font-semibold', t.status === 'available' ? 'bg-[#e5f5e8] text-[#237438]' : t.status === 'used' ? 'bg-warm text-ink-soft' : 'bg-[#fdeaea] text-[#b3261e]']">
                          {{ ticketStatusLabel(t.status) }}
                        </span>
                      </div>
                      <p class="mt-0.5 text-ink-soft">
                        Emitido {{ formatDateTime(t.issuedAt) }} · válido até {{ formatDateOnly(t.validUntil) }}
                        <template v-if="t.usedAt"> · usado em {{ formatDateTime(t.usedAt) }}<template v-if="t.usedByName"> por {{ t.usedByName }}</template></template>
                      </p>
                    </div>
                  </div>
                  <p v-else class="mt-2 border-t border-line pt-2 text-xs text-ink-soft">Nenhum QR emitido (pedido ainda não pago).</p>
                </div>
                <p v-if="!selectedCustomer.orders.length" class="py-2 text-sm text-ink-soft">Nenhum pedido ainda.</p>
              </div>

              <button class="mt-5 text-xs font-semibold text-red-600 hover:underline" :disabled="anonymizing" @click="anonymizeCustomer(selectedCustomer)">
                {{ anonymizing ? "Anonimizando..." : "Anonimizar cadastro (LGPD)" }}
              </button>
            </div>
          </div>
        </section>

        <!-- PEDIDOS -->
        <section v-else-if="tab === 'pedidos'">
          <h1 class="font-serif text-2xl font-bold text-ink">Pedidos</h1>

          <form class="mt-6 flex flex-wrap items-end gap-3" @submit.prevent="loadOrders">
            <input v-model="orderQuery" placeholder="Número do pedido, nome ou e-mail..." class="flex-1 min-w-[220px] rounded-lg border border-line px-3 py-2 text-sm" />
            <select v-model="orderStatusFilter" class="rounded-lg border border-line px-3 py-2 text-sm">
              <option value="">Todos os status</option>
              <option value="pending">Aguardando pagamento</option>
              <option value="paid">Pago</option>
              <option value="refunded">Estornado</option>
              <option value="failed">Falhou</option>
              <option value="expired">Expirado</option>
              <option value="cancelled">Cancelado</option>
            </select>
            <div>
              <label class="mb-1 block text-xs font-semibold uppercase tracking-wider text-ink-soft">De</label>
              <input v-model="orderDateFrom" type="date" class="rounded-lg border border-line px-3 py-2 text-sm" />
            </div>
            <div>
              <label class="mb-1 block text-xs font-semibold uppercase tracking-wider text-ink-soft">Até</label>
              <input v-model="orderDateTo" type="date" class="rounded-lg border border-line px-3 py-2 text-sm" />
            </div>
            <button type="submit" class="button-magenta" :disabled="loadingOrders">
              <Search :size="16" /> Filtrar
            </button>
            <button v-if="orderDateActive" type="button" class="text-xs font-semibold text-ink-soft hover:text-ink" @click="clearOrderDateFilter">Limpar datas</button>
          </form>

          <div class="mt-6 overflow-hidden rounded-[var(--radius-card)] border border-line bg-white">
            <table class="w-full text-sm">
              <thead class="bg-warm/60">
                <tr class="text-left text-ink-soft">
                  <th class="px-5 py-3">Pedido</th>
                  <th class="px-5 py-3">Cliente</th>
                  <th class="px-5 py-3">Status</th>
                  <th class="px-5 py-3">Total</th>
                  <th class="px-5 py-3"></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="o in orders" :key="o.id" class="border-t border-line">
                  <td class="px-5 py-3 font-mono text-ink">{{ o.orderNumber }}</td>
                  <td class="px-5 py-3 text-ink-soft">{{ o.studentName }}</td>
                  <td class="px-5 py-3">
                    <span :class="['rounded-full px-2 py-1 text-xs font-semibold', o.status === 'paid' ? 'bg-[#e5f5e8] text-[#237438]' : o.status === 'refunded' ? 'bg-[#fdeaea] text-[#b3261e]' : 'bg-warm text-ink-soft']">
                      {{ orderStatusLabel(o.status) }}
                    </span>
                  </td>
                  <td class="px-5 py-3 font-mono text-ink">{{ formatBRL(o.totalCents) }}</td>
                  <td class="px-5 py-3">
                    <button class="text-xs font-semibold text-magenta" @click="openOrder(o)">Ver detalhes</button>
                  </td>
                </tr>
                <tr v-if="!orders.length && !loadingOrders">
                  <td colspan="5" class="px-5 py-8 text-center text-ink-soft">Nenhum pedido encontrado.</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div v-if="selectedOrder" class="fixed inset-0 z-50 flex items-center justify-center bg-ink/40 p-6" @click.self="selectedOrder = null">
            <div class="w-full max-w-lg rounded-[var(--radius-card)] bg-white p-6">
              <div class="flex items-start justify-between">
                <div>
                  <h3 class="font-serif text-lg font-semibold text-ink">{{ selectedOrder.orderNumber }}</h3>
                  <p class="text-xs text-ink-soft">{{ selectedOrder.studentName }} · {{ selectedOrder.studentEmail }}</p>
                </div>
                <button class="text-ink-soft hover:text-ink" @click="selectedOrder = null"><X :size="18" /></button>
              </div>

              <h4 class="mt-4 mb-2 text-xs font-bold uppercase tracking-wider text-ink-soft">Itens</h4>
              <ul class="divide-y divide-line text-sm">
                <li v-for="(it, i) in selectedOrder.items" :key="i" class="py-2 text-ink">
                  {{ it.activityTitle || it.productTitle }} <span class="text-xs text-ink-soft">({{ it.benefitType === "breakfast" ? "café da manhã" : "aula" }})</span>
                </li>
              </ul>

              <h4 class="mt-4 mb-2 text-xs font-bold uppercase tracking-wider text-ink-soft">Benefícios (QR)</h4>
              <ul class="divide-y divide-line text-sm">
                <li v-for="e in selectedOrder.entitlements" :key="e.id" class="py-2">
                  <div class="flex items-center justify-between">
                    <span class="text-ink">{{ e.label }} · {{ e.vendorName }}</span>
                    <span :class="['rounded-full px-2 py-0.5 text-xs font-semibold', e.status === 'available' ? 'bg-[#e5f5e8] text-[#237438]' : e.status === 'used' ? 'bg-warm text-ink-soft' : 'bg-[#fdeaea] text-[#b3261e]']">
                      {{ ticketStatusLabel(e.status) }}
                    </span>
                  </div>
                  <p class="mt-0.5 text-xs text-ink-soft">
                    Emitido {{ formatDateTime(e.issuedAt) }} · válido até {{ formatDateOnly(e.validUntil) }}
                    <template v-if="e.usedAt"> · usado em {{ formatDateTime(e.usedAt) }}<template v-if="e.usedByName"> por {{ e.usedByName }}</template></template>
                  </p>
                </li>
                <li v-if="!selectedOrder.entitlements.length" class="py-2 text-sm text-ink-soft">Nenhum benefício emitido (pedido ainda não pago).</li>
              </ul>

              <template v-if="selectedOrder.reschedules.length">
                <h4 class="mt-4 mb-2 text-xs font-bold uppercase tracking-wider text-ink-soft">Histórico de remarcações</h4>
                <ul class="divide-y divide-line text-sm">
                  <li v-for="(rr, i) in selectedOrder.reschedules" :key="i" class="py-2">
                    <p class="text-ink">{{ formatDateOnly(rr.previousDate) }} → {{ formatDateOnly(rr.newDate) }}</p>
                    <p class="mt-0.5 text-xs text-ink-soft">{{ formatDateTime(rr.createdAt) }} · por {{ rr.changedByName }} · motivo: {{ rr.reason }}</p>
                  </li>
                </ul>
              </template>

              <p v-if="orderActionError" class="mt-3 text-xs font-medium text-red-600">{{ orderActionError }}</p>

              <div v-if="!isReportsOnly" class="mt-5 flex flex-wrap items-center gap-3">
                <button
                  v-if="selectedOrder.status === 'paid'"
                  class="inline-flex items-center gap-1.5 text-xs font-semibold text-ink-soft hover:text-ink disabled:opacity-50"
                  :disabled="orderActionBusy"
                  @click="resendOrderEmail(selectedOrder)"
                >
                  <Mail :size="13" /> Reenviar e-mail
                </button>

                <template v-if="selectedOrder.status === 'paid' && resendAltEmailOpen">
                  <input
                    v-model="resendAltEmail"
                    type="email"
                    placeholder="e-mail alternativo"
                    class="w-52 rounded-lg border border-line px-2.5 py-1.5 text-xs"
                    @keyup.enter="resendAltEmail.trim() && resendOrderEmail(selectedOrder, resendAltEmail.trim())"
                  />
                  <button
                    class="text-xs font-semibold text-magenta disabled:opacity-50"
                    :disabled="orderActionBusy || !resendAltEmail.trim()"
                    @click="resendOrderEmail(selectedOrder, resendAltEmail.trim())"
                  >
                    Enviar
                  </button>
                  <button class="text-xs font-semibold text-ink-soft hover:text-ink" @click="resendAltEmailOpen = false; resendAltEmail = ''">Cancelar</button>
                </template>
                <button
                  v-else-if="selectedOrder.status === 'paid'"
                  class="inline-flex items-center gap-1.5 text-xs font-semibold text-ink-soft hover:text-ink disabled:opacity-50"
                  :disabled="orderActionBusy"
                  @click="resendAltEmailOpen = true"
                >
                  <Mail :size="13" /> Outro e-mail
                </button>

                <button
                  v-if="selectedOrder.status === 'paid'"
                  class="inline-flex items-center gap-1.5 text-xs font-semibold text-ink-soft hover:text-ink disabled:opacity-50"
                  :disabled="orderActionBusy"
                  @click="openReschedule"
                >
                  <CalendarClock :size="13" /> Remarcar data
                </button>
                <button
                  v-if="selectedOrder.status === 'paid'"
                  class="inline-flex items-center gap-1.5 text-xs font-semibold text-red-600 hover:underline disabled:opacity-50"
                  :disabled="orderActionBusy"
                  @click="refundOrder(selectedOrder)"
                >
                  <RotateCcw :size="13" /> Estornar pedido
                </button>
              </div>
            </div>
          </div>

          <div v-if="rescheduleModalOpen && selectedOrder" class="fixed inset-0 z-60 flex items-center justify-center bg-ink/40 p-6" @click.self="rescheduleModalOpen = false">
            <div class="w-full max-w-sm rounded-[var(--radius-card)] bg-white p-6">
              <div class="flex items-start justify-between">
                <h3 class="font-serif text-lg font-semibold text-ink">Remarcar pedido</h3>
                <button class="text-ink-soft hover:text-ink" @click="rescheduleModalOpen = false"><X :size="18" /></button>
              </div>
              <p class="mt-1 text-xs text-ink-soft">Move todos os benefícios do pedido {{ selectedOrder.orderNumber }} pra uma nova data. O cliente recebe um e-mail avisando a mudança.</p>

              <label class="mt-4 block text-xs font-semibold uppercase tracking-wider text-ink-soft">Nova data</label>
              <input v-model="rescheduleDate" type="date" class="mt-1 w-full rounded-lg border border-line px-3 py-2 text-sm" />

              <label class="mt-4 block text-xs font-semibold uppercase tracking-wider text-ink-soft">Motivo</label>
              <textarea v-model="rescheduleReason" rows="3" placeholder="Ex: cliente não compareceu, solicitou remarcação para outra data." class="mt-1 w-full rounded-lg border border-line px-3 py-2 text-sm"></textarea>

              <p v-if="rescheduleError" class="mt-3 text-xs font-medium text-red-600">{{ rescheduleError }}</p>

              <div class="mt-5 flex justify-end gap-3">
                <button class="text-xs font-semibold text-ink-soft hover:text-ink" @click="rescheduleModalOpen = false">Cancelar</button>
                <button class="button-magenta py-2 text-xs" :disabled="reschedulingOrder" @click="submitReschedule(selectedOrder)">
                  {{ reschedulingOrder ? "Remarcando..." : "Confirmar remarcação" }}
                </button>
              </div>
            </div>
          </div>
        </section>

        <!-- RELATÓRIOS -->
        <section v-else-if="tab === 'relatorios'">
          <h1 class="font-serif text-2xl font-bold text-ink">Relatórios</h1>
          <p class="mt-1 text-sm text-ink-soft">Lista de nomes de quem comprou — por turma (data/hora), por produto ou por atividade.</p>

          <div class="mt-6 flex flex-wrap items-center justify-between gap-3">
            <div class="flex gap-2 rounded-full border border-line bg-white p-1 w-fit">
              <button
                :class="['rounded-full px-4 py-1.5 text-sm font-semibold transition-colors', reportView === 'turma' ? 'bg-magenta text-white' : 'text-ink-soft hover:text-ink']"
                @click="reportView = 'turma'"
              >
                Por turma
              </button>
              <button
                :class="['rounded-full px-4 py-1.5 text-sm font-semibold transition-colors', reportView === 'produto' ? 'bg-magenta text-white' : 'text-ink-soft hover:text-ink']"
                @click="reportView = 'produto'"
              >
                Por produto
              </button>
              <button
                :class="['rounded-full px-4 py-1.5 text-sm font-semibold transition-colors', reportView === 'atividade' ? 'bg-magenta text-white' : 'text-ink-soft hover:text-ink']"
                @click="reportView = 'atividade'"
              >
                Por atividade
              </button>
            </div>

            <div class="flex flex-wrap items-end gap-2">
              <div>
                <label class="mb-1 block text-xs font-semibold uppercase tracking-wider text-ink-soft">De</label>
                <input v-model="reportDateFrom" type="date" class="rounded-lg border border-line px-3 py-1.5 text-sm" />
              </div>
              <div>
                <label class="mb-1 block text-xs font-semibold uppercase tracking-wider text-ink-soft">Até</label>
                <input v-model="reportDateTo" type="date" class="rounded-lg border border-line px-3 py-1.5 text-sm" />
              </div>
              <button class="button-magenta px-4 py-1.5 text-sm" :disabled="loadingReports" @click="loadReports">Filtrar</button>
              <button v-if="reportDateActive" type="button" class="text-xs font-semibold text-ink-soft hover:text-ink" @click="clearReportDateFilter">Limpar</button>
            </div>
          </div>
          <p class="mt-2 text-xs text-ink-soft">
            O filtro de data considera a data da turma em "Por turma" e a data da compra em "Por produto"/"Por atividade".
          </p>

          <p v-if="loadingReports" class="mt-6 text-sm text-ink-soft">Carregando...</p>

          <!-- Por turma -->
          <div v-else-if="reportView === 'turma'" class="mt-6 grid gap-4 md:grid-cols-2">
            <div v-for="s in sessionRosters" :key="s.sessionId" class="rounded-[var(--radius-card)] border border-line bg-white p-5">
              <div class="flex items-start justify-between gap-3">
                <div>
                  <h3 class="font-serif text-lg font-semibold text-ink capitalize">{{ s.activityTitle }}</h3>
                  <p class="text-xs text-ink-soft capitalize">{{ formatSessionDateTime(s.startsAt) }}</p>
                </div>
                <span class="shrink-0 font-mono text-xs text-ink-soft">{{ s.attendees.length }}/{{ s.capacity }}</span>
              </div>
              <ul class="mt-3 divide-y divide-line/70">
                <li v-for="a in s.attendees" :key="a.orderNumber" class="py-2 text-sm">
                  <p class="text-ink">{{ a.name }}</p>
                  <p class="text-xs text-ink-soft">{{ a.email }} · {{ a.orderNumber }}</p>
                </li>
                <li v-if="!s.attendees.length" class="py-2 text-sm text-ink-soft">Nenhum nome ainda.</li>
              </ul>
            </div>
            <p v-if="!sessionRosters.length" class="text-sm text-ink-soft">Nenhuma turma cadastrada.</p>
          </div>

          <!-- Por produto -->
          <div v-else-if="reportView === 'produto'" class="mt-6 grid gap-4 md:grid-cols-2">
            <div v-for="p in productRosters" :key="p.productId" class="rounded-[var(--radius-card)] border border-line bg-white p-5">
              <div class="flex items-start justify-between gap-3">
                <h3 class="font-serif text-lg font-semibold text-ink">{{ p.title }}</h3>
                <span class="shrink-0 font-mono text-xs text-ink-soft">{{ p.buyers.length }}</span>
              </div>
              <ul class="mt-3 divide-y divide-line/70">
                <li v-for="b in p.buyers" :key="b.orderNumber" class="py-2 text-sm">
                  <p class="text-ink">{{ b.name }}</p>
                  <p class="text-xs text-ink-soft">{{ b.email }} · {{ b.orderNumber }}</p>
                </li>
                <li v-if="!p.buyers.length" class="py-2 text-sm text-ink-soft">Nenhum nome ainda.</li>
              </ul>
            </div>
            <p v-if="!productRosters.length" class="text-sm text-ink-soft">Nenhum produto cadastrado.</p>
          </div>

          <!-- Por atividade -->
          <div v-else class="mt-6 grid gap-4 md:grid-cols-2">
            <div v-for="a in activityRosters" :key="a.label" class="rounded-[var(--radius-card)] border border-line bg-white p-5">
              <div class="flex items-start justify-between gap-3">
                <h3 class="font-serif text-lg font-semibold text-ink">{{ a.label }}</h3>
                <span class="shrink-0 font-mono text-xs text-ink-soft">{{ a.attendees.length }}</span>
              </div>
              <ul class="mt-3 divide-y divide-line/70">
                <li v-for="(at, i) in a.attendees" :key="`${at.orderNumber}-${i}`" class="py-2 text-sm">
                  <p class="text-ink">{{ at.name }}</p>
                  <p class="text-xs text-ink-soft">{{ at.email }} · {{ at.orderNumber }}</p>
                </li>
                <li v-if="!a.attendees.length" class="py-2 text-sm text-ink-soft">Nenhum nome ainda.</li>
              </ul>
            </div>
            <p v-if="!activityRosters.length" class="text-sm text-ink-soft">Nenhuma atividade cadastrada.</p>
          </div>
        </section>
      </main>
    </div>
  </div>
</template>
