<script setup>
defineProps({
  summary: { type: String, required: true },
  goVersion: { type: String, required: true },
})
</script>

<template>
  <aside class="topic-meta" aria-label="Версия и краткое описание темы">
    <p class="topic-meta__summary">{{ summary }}</p>
    <span class="topic-meta__version">Семантика Go {{ goVersion }}</span>
  </aside>
</template>
