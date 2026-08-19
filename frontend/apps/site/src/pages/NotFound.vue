<script setup lang="ts">
import { RouterLink } from "vue-router";
</script>

<template>
  <div class="flex min-h-screen flex-col items-center justify-center gap-4 bg-paper px-6 text-center">
    <p class="eyebrow">Página não encontrada</p>
    <h1 class="font-serif text-3xl font-bold text-ink">Esse caminho não existe por aqui.</h1>
    <RouterLink to="/" class="button-magenta">Voltar para o início</RouterLink>
  </div>
</template>
