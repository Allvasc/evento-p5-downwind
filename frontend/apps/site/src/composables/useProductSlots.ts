import { ref, computed, watch, type Ref } from "vue";
import type { Product } from "@p5wellness/shared";
import { api } from "@/lib/api";

// Shared between Comprar.vue (paid checkout) and Voucher.vue (free redemption) — both
// let the customer pick a product then a bookable date the same way, so the tricky part
// (grouping each activity's own turmas into one bookable day, see below) lives in one
// place instead of two copies that could quietly drift apart.

export interface RawSession {
  id: string;
  activityId: string;
  activityTitle: string;
  startsAt: string;
  capacity: number;
  booked: number;
}
export interface SlotSession {
  activityId: string;
  activityTitle: string;
  startsAt: string;
  spotsLeft: number;
}
export interface Slot {
  key: string;
  day: string;
  sessions: SlotSession[];
  sessionIds: Record<string, string>;
  minSpotsLeft: number;
}

// The event runs as a single-day gathering (Yoga + HYROX together), so two activities
// scheduled at different times on the same Fortaleza calendar day still count as one
// bookable date — matching on the exact instant instead used to leave combos with zero
// slots whenever the two turmas weren't cadastradas at the identical minute.
export const EVENT_TZ = "America/Fortaleza";

export function fortalezaDay(iso: string) {
  return new Intl.DateTimeFormat("en-CA", { timeZone: EVENT_TZ, year: "numeric", month: "2-digit", day: "2-digit" }).format(new Date(iso));
}

export function formatSlotDay(day: string) {
  return new Date(`${day}T00:00:00Z`).toLocaleDateString("pt-BR", { timeZone: "UTC", weekday: "long", day: "2-digit", month: "2-digit" });
}

export function formatSlotTime(iso: string) {
  return new Date(iso).toLocaleTimeString("pt-BR", { timeZone: EVENT_TZ, hour: "2-digit", minute: "2-digit" });
}

// Only spell out each activity's own time when they actually differ (the common case is
// everything at the same hour) — otherwise a single time reads cleaner than a repeated list.
export function formatSlotSummary(slot: Slot) {
  const times = new Set(slot.sessions.map((s) => formatSlotTime(s.startsAt)));
  if (times.size <= 1) return times.values().next().value ?? "";
  return slot.sessions.map((s) => `${s.activityTitle} ${formatSlotTime(s.startsAt)}`).join(" · ");
}

export function useProductSlots(selected: Ref<Product | null>) {
  // For chooseOneActivity products (e.g. "Yoga ou HYROX"), the customer first picks which
  // activity, then sees only that activity's dates — instead of the usual "every linked
  // activity needs a matching session" grouping used for combos.
  const chosenActivityId = ref<string | null>(null);
  const selectedSlotKey = ref<string | null>(null);
  const sessionsByActivity = ref<Record<string, RawSession[]>>({});
  const loadingSlots = ref(false);
  const slotsError = ref(false);

  watch(selected, async (product) => {
    selectedSlotKey.value = null;
    chosenActivityId.value = null;
    sessionsByActivity.value = {};
    slotsError.value = false;
    if (!product || !product.activities.length) return;
    loadingSlots.value = true;
    try {
      const results = await Promise.all(
        product.activities.map((a) => api.get<{ sessions: RawSession[] }>(`/public/activities/${a.id}/sessions`).then((r) => [a.id, r.sessions] as const)),
      );
      sessionsByActivity.value = Object.fromEntries(results);
    } catch {
      // A fetch failure must not be silently mistaken for "no turmas cadastradas".
      slotsError.value = true;
    } finally {
      loadingSlots.value = false;
    }
  });

  const slots = computed<Slot[]>(() => {
    const product = selected.value;
    if (!product || !product.activities.length) return [];

    if (product.chooseOneActivity) {
      if (!chosenActivityId.value) return [];
      const activity = product.activities.find((a) => a.id === chosenActivityId.value);
      const sessions = sessionsByActivity.value[chosenActivityId.value] ?? [];
      return sessions.map((s) => ({
        key: s.id,
        day: fortalezaDay(s.startsAt),
        sessions: [{ activityId: s.activityId, activityTitle: activity?.title ?? "", startsAt: s.startsAt, spotsLeft: s.capacity - s.booked }],
        sessionIds: { [s.activityId]: s.id },
        minSpotsLeft: s.capacity - s.booked,
      }));
    }

    // Group each activity's own sessions by Fortaleza calendar day, then keep only the
    // days where every linked activity has at least one open session. Picks the earliest
    // session per activity when more than one falls on the same day.
    const byDay = new Map<string, Map<string, RawSession>>();
    for (const activity of product.activities) {
      for (const session of sessionsByActivity.value[activity.id] ?? []) {
        const day = fortalezaDay(session.startsAt);
        const forDay = byDay.get(day) ?? new Map<string, RawSession>();
        const existing = forDay.get(activity.id);
        if (!existing || session.startsAt < existing.startsAt) forDay.set(activity.id, session);
        byDay.set(day, forDay);
      }
    }

    const result: Slot[] = [];
    for (const [day, byActivity] of byDay) {
      if (byActivity.size !== product.activities.length) continue;
      const sessions = product.activities.map((a) => {
        const s = byActivity.get(a.id)!;
        return { activityId: a.id, activityTitle: a.title, startsAt: s.startsAt, spotsLeft: s.capacity - s.booked };
      });
      result.push({
        key: day,
        day,
        sessions,
        sessionIds: Object.fromEntries(product.activities.map((a) => [a.id, byActivity.get(a.id)!.id])),
        minSpotsLeft: Math.min(...sessions.map((s) => s.spotsLeft)),
      });
    }
    result.sort((a, b) => a.day.localeCompare(b.day));
    return result;
  });

  const selectedSlot = computed(() => slots.value.find((s) => s.key === selectedSlotKey.value) ?? null);

  function chooseActivity(activityId: string) {
    chosenActivityId.value = activityId;
    selectedSlotKey.value = null;
  }

  return { chosenActivityId, selectedSlotKey, sessionsByActivity, loadingSlots, slotsError, slots, selectedSlot, chooseActivity };
}
