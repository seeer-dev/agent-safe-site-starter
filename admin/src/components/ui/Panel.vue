<script setup lang="ts">
defineProps<{
  title?: string
}>()
</script>

<template>
  <div class="panel">
    <div
      v-if="title || $slots.header || $slots.actions"
      class="phd"
    >
      <slot name="header">
        <h3 v-if="title">{{ title }}</h3>
      </slot>
      <div class="more">
        <slot name="actions" />
      </div>
    </div>
    <slot />
  </div>
</template>
