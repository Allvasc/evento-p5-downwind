<script setup lang="ts">
withDefaults(
  defineProps<{
    size?: number;
    wordmark?: boolean;
    inverted?: boolean;
  }>(),
  {
    size: 38,
    wordmark: true,
    inverted: false,
  }
);
</script>

<template>
  <div class="inline-flex items-center gap-2.5">
    <svg :width="size" :height="size" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" class="shrink-0">
      <rect width="32" height="32" rx="9" :fill="inverted ? '#ffffff' : '#0a0f1a'" :fill-opacity="inverted ? 0.1 : 1" />
      <path d="M7 12 Q11 8 15 12 T23 12" stroke="#7dd3fc" stroke-width="1.6" fill="none" stroke-linecap="round" />
      <path d="M6 17 Q11.5 11.5 17 17 T27 17" stroke="#38bdf8" stroke-width="1.9" fill="none" stroke-linecap="round" />
      <path d="M5 22.5 Q11.5 15 18 22.5 T29 22.5" stroke="url(#brandmark-g)" stroke-width="2.2" fill="none" stroke-linecap="round" />
      <defs>
        <linearGradient id="brandmark-g" x1="5" y1="15" x2="29" y2="23" gradientUnits="userSpaceOnUse">
          <stop offset="0" stop-color="#7dd3fc" />
          <stop offset="1" stop-color="#0b63d6" />
        </linearGradient>
      </defs>
    </svg>
    <span v-if="wordmark" class="flex flex-col leading-none">
      <span :class="['font-serif font-black tracking-tight', inverted ? 'text-white' : 'text-ink']" :style="{ fontSize: size * 0.48 + 'px' }">
        P5
      </span>
      <span
        :class="['font-mono font-bold tracking-[0.18em] uppercase', inverted ? 'text-white/60' : 'text-magenta']"
        :style="{ fontSize: Math.max(size * 0.225, 8) + 'px', marginTop: '-1px' }"
      >
        DownWind Day
      </span>
    </span>
  </div>
</template>
