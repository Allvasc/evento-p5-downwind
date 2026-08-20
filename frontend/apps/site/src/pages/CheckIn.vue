<script setup lang="ts">
import { ref, onBeforeUnmount, onMounted, watch } from "vue";
import { useRouter } from "vue-router";
import { ScanLine, Video, VideoOff, CheckCircle2, XCircle, AlertTriangle, ShieldCheck, LogOut, WifiOff, RefreshCw, X, LayoutDashboard, ListChecks, Search } from "lucide-vue-next";
import { api, ApiError } from "@/lib/teamApi";
import { useTeamAuthStore as useAuthStore } from "@/stores/teamAuth";
import { enqueueScan, listPendingScans, removeScans, countPendingScans, type PendingScan } from "@/offline/queue";

const router = useRouter();
const authStore = useAuthStore();

const videoRef = ref<HTMLVideoElement | null>(null);
const cameraOn = ref(false);
const cameraError = ref("");
const manualToken = ref("");
const validating = ref(false);

type Outcome = {
  result: string;
  label?: string;
  vendorName?: string;
  studentName?: string;
  orderNumber?: string;
  usedAt?: string;
  message: string;
};
const lastResult = ref<Outcome | null>(null);
const showBanner = ref(false);
let bannerTimer: number | null = null;

function announceResult(outcome: Outcome) {
  lastResult.value = outcome;
  showBanner.value = true;
  if (bannerTimer) window.clearTimeout(bannerTimer);
  bannerTimer = window.setTimeout(() => { showBanner.value = false; }, 8000);
}

function formatDateTime(iso?: string) {
  if (!iso) return "";
  return new Date(iso).toLocaleString("pt-BR", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" });
}

// ── Offline queue ──────────────────────────────────────────────────────────
// The kiosk's connection is unreliable enough that a scan taken offline must never be
// silently lost — it's persisted locally right away and replayed once the connection
// (or the next manual sync) comes back, via the idempotent batch endpoint.
const isOnline = ref(navigator.onLine);
const pendingCount = ref(0);
const syncing = ref(false);

async function refreshPendingCount() {
  pendingCount.value = await countPendingScans();
}

async function syncPending() {
  if (syncing.value) return;
  const pending = await listPendingScans();
  if (!pending.length) return;
  syncing.value = true;
  try {
    const res = await api.post<{ results: (Outcome & { clientScanId: string })[] }>("/checkin/validate/batch", { scans: pending });
    await removeScans(res.results.map((r) => r.clientScanId));
    if (res.results.length === 1) {
      announceResult(res.results[0]);
    } else if (res.results.length > 1) {
      const successes = res.results.filter((r: Outcome) => r.result === "success").length;
      announceResult({ result: "queued", message: `${res.results.length} scans sincronizados (${successes} confirmados).` });
    }
  } catch {
    // still offline or the request failed — leave the queue intact, try again later
  } finally {
    syncing.value = false;
    await refreshPendingCount();
  }
}

function handleOnline() {
  isOnline.value = true;
  syncPending();
}
function handleOffline() {
  isOnline.value = false;
}

let stream: MediaStream | null = null;
let detector: any = null;
let scanLoop: number | null = null;

async function startCamera() {
  cameraError.value = "";
  try {
    stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: "environment" } });
    if (videoRef.value) {
      videoRef.value.srcObject = stream;
      await videoRef.value.play();
    }
    cameraOn.value = true;

    const BarcodeDetectorCtor = (window as any).BarcodeDetector;
    if (BarcodeDetectorCtor) {
      detector = new BarcodeDetectorCtor({ formats: ["qr_code"] });
      scanLoop = window.setInterval(async () => {
        if (!videoRef.value || validating.value) return;
        try {
          const codes = await detector.detect(videoRef.value);
          if (codes.length > 0) {
            await validate(codes[0].rawValue);
          }
        } catch {
          // detection hiccup, keep trying
        }
      }, 600);
    } else {
      cameraError.value = "Este navegador não suporta leitura automática. Use a entrada manual abaixo.";
    }
  } catch {
    cameraError.value = "Não foi possível acessar a câmera. Use o código manual.";
  }
}

function stopCamera() {
  if (scanLoop) window.clearInterval(scanLoop);
  stream?.getTracks().forEach((t) => t.stop());
  stream = null;
  cameraOn.value = false;
}

async function validate(token: string) {
  if (!token || validating.value) return;
  validating.value = true;
  const scan: PendingScan = { token, clientScanId: crypto.randomUUID(), deviceScannedAt: new Date().toISOString() };
  try {
    if (!isOnline.value) throw new Error("offline");
    const res = await api.post<Outcome>("/checkin/validate", scan);
    announceResult(res);
  } catch (err) {
    if (err instanceof ApiError) {
      // A real answer came back from the server (bad token, expired, etc.) — not a
      // connectivity problem, so there's nothing to queue.
      announceResult({ result: "error", message: err.message });
    } else {
      // Network failure or known-offline: never lose the scan — queue it and let the
      // guest go through, to be confirmed once this device syncs.
      await enqueueScan(scan);
      await refreshPendingCount();
      announceResult({ result: "queued", message: "Sem conexão — scan salvo e será confirmado assim que sincronizar." });
    }
  } finally {
    validating.value = false;
    manualToken.value = "";
    // Stopping the camera after every read (instead of leaving it looping) frees the
    // screen for the result banner and stops it re-scanning the same code by accident.
    stopCamera();
  }
}

function handleLogout() {
  stopCamera();
  authStore.logout();
  router.push("/acesso-admin");
}

// ── Validação por lista (2ª via) ───────────────────────────────────────────
// Fallback to scanning: staff pick the guest by name off today's paid roster and
// confirm with a tap — for a dead phone, a QR that won't scan, or just a short line
// where reading a name is faster. Same server-side rules apply (sector, expiry,
// exactly-once), it's just a different way to find the ticket.
type RosterEntry = {
  entitlementId: string;
  studentName: string;
  orderNumber: string;
  status: string;
  usedAt?: string;
};
type RosterGroup = {
  key: string;
  label: string;
  vendorName: string;
  entries: RosterEntry[];
};
const rosterGroups = ref<RosterGroup[]>([]);
const rosterLoading = ref(false);
const rosterFilter = ref("");
const validatingEntitlementId = ref("");

// Which days show up here comes straight from the registered turmas (class_sessions),
// not from assuming "hoje" is the only valid event day — so staff can pull up a
// different date (the next Saturday, one that ran late) without guessing.
const rosterDates = ref<string[]>([]);
const selectedDate = ref("");

function todayLocal() {
  return new Date().toLocaleDateString("sv-SE"); // sv-SE gives YYYY-MM-DD
}

async function loadRosterDates() {
  try {
    const res = await api.get<{ dates: string[] }>("/checkin/roster/dates");
    rosterDates.value = res.dates;
    const today = todayLocal();
    // Prefer today if it has a turma; otherwise the closest upcoming date; otherwise
    // fall back to the most recent one so the list isn't just empty by default.
    selectedDate.value =
      res.dates.find((d: string) => d === today) ?? res.dates.find((d: string) => d >= today) ?? res.dates[res.dates.length - 1] ?? today;
  } catch {
    selectedDate.value = todayLocal();
  }
}

function formatDatePt(iso: string) {
  if (!iso) return "";
  const [y, m, d] = iso.split("-");
  return `${d}/${m}/${y}`;
}

async function loadRoster() {
  if (!selectedDate.value) return;
  rosterLoading.value = true;
  try {
    const res = await api.get<{ date: string; groups: RosterGroup[] }>(`/checkin/roster?date=${selectedDate.value}`);
    rosterGroups.value = res.groups;
  } catch {
    // stays with whatever was loaded before; the retry button covers a flaky request
  } finally {
    rosterLoading.value = false;
  }
}

watch(selectedDate, loadRoster);

function filteredEntries(entries: RosterEntry[]) {
  const q = rosterFilter.value.trim().toLowerCase();
  if (!q) return entries;
  return entries.filter((e) => e.studentName.toLowerCase().includes(q) || e.orderNumber.toLowerCase().includes(q));
}

async function validateByList(entitlementId: string) {
  if (validatingEntitlementId.value) return;
  validatingEntitlementId.value = entitlementId;
  try {
    const res = await api.post<Outcome>("/checkin/validate/by-id", {
      entitlementId,
      clientScanId: crypto.randomUUID(),
      deviceScannedAt: new Date().toISOString(),
    });
    announceResult(res);
    await loadRoster();
  } catch (err) {
    announceResult({ result: "error", message: err instanceof ApiError ? err.message : "sem conexão — tente de novo" });
  } finally {
    validatingEntitlementId.value = "";
  }
}

onMounted(async () => {
  window.addEventListener("online", handleOnline);
  window.addEventListener("offline", handleOffline);
  await refreshPendingCount();
  if (isOnline.value) await syncPending();
  await loadRosterDates();
});

onBeforeUnmount(() => {
  stopCamera();
  if (bannerTimer) window.clearTimeout(bannerTimer);
  window.removeEventListener("online", handleOnline);
  window.removeEventListener("offline", handleOffline);
});
</script>

<template>
  <div class="min-h-screen bg-paper">
    <header class="border-b border-line bg-white">
      <div class="mx-auto flex max-w-5xl items-center justify-between px-6 py-4">
        <div class="flex items-center gap-2">
          <span class="font-serif text-lg font-bold text-ink">P5</span>
          <span class="h-4 w-px bg-line"></span>
          <span class="font-mono text-xs uppercase tracking-widest text-magenta">Operação</span>
        </div>
        <div class="flex items-center gap-3">
          <button
            v-if="authStore.role === 'admin' || authStore.role === 'reports'"
            type="button"
            class="inline-flex items-center gap-1.5 rounded-full border border-line px-3 py-1.5 text-xs font-semibold text-ink hover:border-magenta hover:text-magenta"
            @click="router.push('/admin')"
          >
            <LayoutDashboard :size="14" /> Área da equipe
          </button>
          <span v-if="!isOnline" class="inline-flex items-center gap-1 rounded-full bg-[#fde8f1] px-3 py-1 text-xs font-semibold text-magenta">
            <WifiOff :size="14" /> Offline
          </span>
          <button
            v-if="pendingCount > 0"
            type="button"
            class="inline-flex items-center gap-1 rounded-full bg-[#fbefe2] px-3 py-1 text-xs font-semibold text-[#946213] disabled:opacity-60"
            :disabled="syncing || !isOnline"
            @click="syncPending"
          >
            <RefreshCw :size="13" :class="{ 'animate-spin': syncing }" /> {{ syncing ? "Sincronizando..." : `${pendingCount} pendente${pendingCount > 1 ? "s" : ""}` }}
          </button>
          <span class="inline-flex items-center gap-1 rounded-full bg-warm px-3 py-1 text-xs font-semibold text-ink-soft">
            <ShieldCheck :size="14" /> {{ authStore.role === "admin" ? "Admin" : authStore.role === "reports" ? "Relatórios" : "Funcionário" }}
            <template v-if="authStore.vendorName"> · {{ authStore.vendorName }}</template>
          </span>
          <button class="rounded-full p-2 text-ink-soft hover:text-ink" aria-label="Sair" @click="handleLogout">
            <LogOut :size="16" />
          </button>
        </div>
      </div>
    </header>

    <!-- Big, hard-to-miss result banner: staff kept saying a scan seemed to do nothing
         because the confirmation only lived in a card below the camera, off-screen on
         mobile. This sits sticky at the very top, in full view regardless of scroll. -->
    <Transition name="banner">
      <div
        v-if="showBanner && lastResult"
        class="sticky top-0 z-40 px-4 py-4 text-white shadow-lg sm:px-6"
        :class="{
          'bg-[#1f8a3d]': lastResult.result === 'success',
          'bg-[#b8791a]': ['already_used', 'expired', 'queued'].includes(lastResult.result),
          'bg-magenta': ['not_found', 'invalid_signature', 'wrong_sector', 'error'].includes(lastResult.result),
        }"
      >
        <div class="mx-auto flex max-w-5xl items-start justify-between gap-4">
          <div class="flex items-start gap-3">
            <CheckCircle2 v-if="lastResult.result === 'success'" :size="28" class="mt-0.5 shrink-0" />
            <AlertTriangle v-else-if="['already_used', 'expired', 'queued'].includes(lastResult.result)" :size="28" class="mt-0.5 shrink-0" />
            <XCircle v-else :size="28" class="mt-0.5 shrink-0" />
            <div>
              <p class="text-lg font-bold leading-tight sm:text-xl">{{ lastResult.message }}</p>
              <p v-if="lastResult.studentName" class="mt-1 text-sm font-medium opacity-90">
                {{ lastResult.studentName }} · {{ lastResult.label }}
                <template v-if="lastResult.orderNumber"> · pedido {{ lastResult.orderNumber }}</template>
              </p>
              <p v-if="lastResult.usedAt" class="mt-0.5 text-sm opacity-90">Validado em {{ formatDateTime(lastResult.usedAt) }}</p>
            </div>
          </div>
          <button class="shrink-0 rounded-full p-1.5 hover:bg-white/20" aria-label="Fechar" @click="showBanner = false">
            <X :size="20" />
          </button>
        </div>
      </div>
    </Transition>

    <main class="mx-auto max-w-5xl px-6 py-10">
      <div class="mb-8">
        <p class="eyebrow eyebrow-dark mb-2">Terminal da equipe</p>
        <h1 class="font-serif text-3xl font-bold text-ink">Validação de <em class="italic text-magenta">presença.</em></h1>
        <p class="mt-1 text-sm text-ink-soft">
          <template v-if="authStore.vendorName">Você valida somente benefícios do setor <strong class="text-ink">{{ authStore.vendorName }}</strong>.</template>
          <template v-else>Acesso de administrador: valida benefícios de qualquer setor.</template>
        </p>
      </div>

      <div class="grid gap-6 md:grid-cols-2">
        <section class="rounded-[var(--radius-card)] border border-line bg-white p-6">
          <div class="mb-4 flex items-center gap-3">
            <div class="flex h-11 w-11 items-center justify-center rounded-full bg-[#fde8f1] text-magenta">
              <ScanLine :size="22" />
            </div>
            <div>
              <h2 class="font-serif text-lg font-semibold text-ink">Escanear QR Code</h2>
              <p class="text-xs text-ink-soft">Leia o passe digital exibido pelo aluno.</p>
            </div>
          </div>

          <div class="flex aspect-video items-center justify-center overflow-hidden rounded-xl bg-ink">
            <video v-show="cameraOn" ref="videoRef" class="h-full w-full object-cover" muted playsinline autoplay></video>
            <div v-if="!cameraOn" class="flex flex-col items-center gap-2 text-white/70">
              <Video :size="40" />
              <p class="max-w-[220px] text-center text-xs">{{ cameraError || "Câmera desligada" }}</p>
            </div>
          </div>

          <button class="button-ink mt-4 flex w-full items-center justify-center gap-2 rounded-full bg-ink px-4 py-3 text-sm font-semibold text-white" @click="cameraOn ? stopCamera() : startCamera()">
            <VideoOff v-if="cameraOn" :size="17" />
            <Video v-else :size="17" />
            {{ cameraOn ? "Encerrar câmera" : "Abrir câmera" }}
          </button>
          <p class="mt-3 text-xs text-ink-soft">
            A câmera reconhece QR Codes compatíveis automaticamente. A entrada manual permanece disponível como contingência.
          </p>
        </section>

        <section class="rounded-[var(--radius-card)] border border-line bg-white p-6">
          <h2 class="font-serif text-lg font-semibold text-ink">Confirmar validação</h2>
          <p class="mt-1 text-xs text-ink-soft">Os benefícios dos combos são independentes. Cada QR Code valida um item por vez.</p>

          <div class="mt-4">
            <label for="qr-token" class="mb-1 block text-sm font-medium text-ink">Código do QR</label>
            <div class="flex gap-2">
              <input id="qr-token" v-model="manualToken" placeholder="p5_..." class="w-full rounded-lg border border-line px-3 py-2 text-sm focus:border-magenta focus:outline-none" @keyup.enter="validate(manualToken)" />
              <button class="button-magenta shrink-0" :disabled="validating" @click="validate(manualToken)">Validar</button>
            </div>
          </div>

          <div v-if="lastResult" class="mt-6 rounded-xl border p-4" :class="{
            'border-green-200 bg-[#e5f5e8]': lastResult.result === 'success',
            'border-amber-200 bg-[#fbefe2]': ['already_used', 'expired', 'queued'].includes(lastResult.result),
            'border-red-200 bg-[#fde8f1]': ['not_found', 'invalid_signature', 'wrong_sector', 'error'].includes(lastResult.result),
          }">
            <div class="flex items-center gap-2">
              <CheckCircle2 v-if="lastResult.result === 'success'" :size="20" class="text-[#237438]" />
              <WifiOff v-else-if="lastResult.result === 'queued'" :size="20" class="text-[#946213]" />
              <AlertTriangle v-else-if="['already_used', 'expired'].includes(lastResult.result)" :size="20" class="text-[#946213]" />
              <XCircle v-else :size="20" class="text-magenta" />
              <p class="font-semibold text-ink">{{ lastResult.message }}</p>
            </div>
            <div v-if="lastResult.studentName" class="mt-3 space-y-1 text-sm text-ink-soft">
              <p><strong class="text-ink">{{ lastResult.studentName }}</strong></p>
              <p>{{ lastResult.label }} <template v-if="lastResult.vendorName">· {{ lastResult.vendorName }}</template> — pedido {{ lastResult.orderNumber }}</p>
              <p v-if="lastResult.usedAt">Validado em {{ formatDateTime(lastResult.usedAt) }}</p>
            </div>
          </div>
          <p v-else class="mt-6 text-sm text-ink-soft">As validações feitas nesta sessão aparecerão aqui.</p>
        </section>
      </div>

      <section class="mt-6 rounded-[var(--radius-card)] border border-line bg-white p-6">
        <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
          <div class="flex items-center gap-3">
            <div class="flex h-11 w-11 items-center justify-center rounded-full bg-[#fde8f1] text-magenta">
              <ListChecks :size="22" />
            </div>
            <div>
              <h2 class="font-serif text-lg font-semibold text-ink">Validar por lista <span class="text-ink-soft">— 2ª via</span></h2>
              <p class="text-xs text-ink-soft">Sem QR Code à mão? Confirme direto pelo nome de quem já pagou.</p>
            </div>
          </div>
          <button type="button" class="inline-flex items-center gap-1.5 rounded-full border border-line px-3 py-1.5 text-xs font-semibold text-ink hover:border-magenta hover:text-magenta disabled:opacity-60" :disabled="rosterLoading" @click="loadRosterDates">
            <RefreshCw :size="13" :class="{ 'animate-spin': rosterLoading }" /> Atualizar
          </button>
        </div>

        <div class="mb-4 flex flex-col gap-3 sm:flex-row">
          <select v-if="rosterDates.length" v-model="selectedDate" class="rounded-lg border border-line bg-white px-3 py-2 text-sm focus:border-magenta focus:outline-none">
            <option v-for="d in rosterDates" :key="d" :value="d">{{ formatDatePt(d) }}</option>
          </select>
          <p v-else class="text-sm text-ink-soft">Nenhuma turma cadastrada ainda.</p>
          <div class="relative flex-1">
            <Search :size="16" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-ink-soft" />
            <input v-model="rosterFilter" type="text" placeholder="Buscar por nome ou pedido..." class="w-full rounded-lg border border-line py-2 pl-9 pr-3 text-sm focus:border-magenta focus:outline-none" />
          </div>
        </div>

        <p v-if="!rosterLoading && rosterGroups.length === 0" class="text-sm text-ink-soft">Ninguém pago pra {{ selectedDate ? formatDatePt(selectedDate) : "essa data" }} ainda no seu setor.</p>

        <div v-for="group in rosterGroups" :key="group.key" class="mb-5 last:mb-0">
          <h3 class="mb-2 font-serif text-sm font-semibold text-ink">{{ group.label }}</h3>
          <ul class="divide-y divide-line rounded-xl border border-line">
            <li v-for="entry in filteredEntries(group.entries)" :key="entry.entitlementId" class="flex items-center justify-between gap-3 px-4 py-3">
              <div class="min-w-0">
                <p class="truncate text-sm font-medium text-ink">{{ entry.studentName }}</p>
                <p class="text-xs text-ink-soft">pedido {{ entry.orderNumber }}</p>
              </div>
              <button
                v-if="entry.status === 'available'"
                type="button"
                class="button-magenta shrink-0 px-4! py-2! text-xs disabled:opacity-60"
                :disabled="validatingEntitlementId === entry.entitlementId"
                @click="validateByList(entry.entitlementId)"
              >
                {{ validatingEntitlementId === entry.entitlementId ? "Validando..." : "Confirmar" }}
              </button>
              <span v-else class="inline-flex shrink-0 items-center gap-1 rounded-full bg-[#e5f5e8] px-3 py-1.5 text-xs font-semibold text-[#237438]">
                <CheckCircle2 :size="14" /> Validado
              </span>
            </li>
            <li v-if="filteredEntries(group.entries).length === 0" class="px-4 py-3 text-sm text-ink-soft">Nenhum nome bate com a busca.</li>
          </ul>
        </div>
      </section>
    </main>
  </div>
</template>

<style scoped>
.banner-enter-active,
.banner-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.banner-enter-from,
.banner-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
