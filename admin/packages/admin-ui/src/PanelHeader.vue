<script setup lang="ts">
withDefaults(defineProps<{
  title?: string
}>(), {
  title: '',
})
</script>

<template>
  <div class="panel-hd">
    <h3 v-if="title">{{ title }}</h3>
    <slot v-else name="title" />
    <div v-if="$slots['more']" class="panel-hd-more">
      <slot name="more" />
    </div>
    <slot name="header-extra" />
  </div>
</template>
