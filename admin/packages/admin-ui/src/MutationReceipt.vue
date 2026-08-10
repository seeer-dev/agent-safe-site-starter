<script setup lang="ts">
withDefaults(defineProps<{
  title?: string
  summary?: string
  meta?: string
}>(), {
  title: '',
  summary: '',
  meta: '',
})
</script>

<template>
  <article class="receipt" data-fixture-receipt>
    <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6"/><path d="m9 15 2 2 4-4"/></svg>
    <div>
      <strong v-if="title">{{ title }}</strong>
      <p v-if="summary">{{ summary }}</p>
      <small v-if="meta">{{ meta }}</small>
    </div>
  </article>
</template>
