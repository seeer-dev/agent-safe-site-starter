<script setup lang="ts">
// Skeleton placeholders — structure only, never data. Rendered while a
// real request is in flight; MUST never carry fixture or invented rows.
withDefaults(defineProps<{
  /** lines: stacked bars. table: list-row shaped bars (a leading block +
   *  a wide bar + short bars), matching the reference loading pattern. */
  variant?: 'lines' | 'table'
  rows?: number
  /** Bar height for variant="lines". */
  height?: string
}>(), {
  variant: 'lines',
  rows: 3,
  height: '14px',
})
</script>

<template>
  <div
    v-if="variant === 'table'"
    class="skel-rows"
    role="status"
    aria-label="載入中"
  >
    <div v-for="i in rows" :key="i" class="skel-row">
      <div class="skel" style="height:14px;width:72px;flex:none" />
      <div class="skel" style="height:14px;flex:1;max-width:220px" />
      <div class="skel" style="height:14px;width:56px;flex:none" />
      <div class="skel" style="height:14px;width:40px;flex:none" />
      <div class="skel" style="height:22px;width:64px;flex:none;border-radius:11px" />
    </div>
  </div>
  <div
    v-else
    class="skel-stack"
    role="status"
    aria-label="載入中"
  >
    <div
      v-for="i in rows"
      :key="i"
      class="skel"
      :style="{ height, width: i === rows && rows > 1 ? '55%' : '100%' }"
    />
  </div>
</template>
