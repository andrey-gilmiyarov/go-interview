<script setup>
defineProps({
  summary: { type: String, required: true },
  goVersion: { type: String, default: '' },
  postgresVersion: { type: String, default: '' },
})
</script>

<template>
  <aside class="topic-meta" aria-label="Версия и краткое описание темы">
    <p class="topic-meta__summary">{{ summary }}</p>
    <span v-if="postgresVersion" class="topic-meta__version">PostgreSQL {{ postgresVersion }}</span>
    <span v-else class="topic-meta__version">Семантика Go {{ goVersion }}</span>
  </aside>
</template>
