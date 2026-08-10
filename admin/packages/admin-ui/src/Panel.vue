<script setup lang="ts">
withDefaults(defineProps<{
  title?: string
}>(), {
  title: '',
})
</script>

<template>
  <section class="panel">
    <div v-if="title || $slots['header']" class="panel-hd">
      <h3 v-if="title">{{ title }}</h3>
      <slot v-else name="header" />
      <slot name="header-extra" />
    </div>
    <slot />
  </section>
</template>
