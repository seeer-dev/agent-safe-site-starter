<script setup lang="ts">
import { CheckIcon } from 'lucide-vue-next'

const props = withDefaults(defineProps<{
  checked?: boolean
  disabled?: boolean
  id?: string
  ariaLabel?: string
  ariaDescribedby?: string
  class?: string
}>(), {
  checked: false,
  disabled: false,
})

const emit = defineEmits<{ 'update:checked': [value: boolean] }>()

function onInput(event: Event) {
  if (props.disabled) return
  const target = event.target as HTMLInputElement
  emit('update:checked', target.checked)
}
</script>

<template>
  <span class="ck-box-wrapper inline-flex items-center" :class="$props.class">
    <input
      :id="id"
      type="checkbox"
      class="ck-input"
      :checked="checked"
      :disabled="disabled"
      :aria-label="ariaLabel"
      :aria-describedby="ariaDescribedby"
      @change="onInput"
    />
    <span class="ck-box" :data-checked="checked" aria-hidden="true">
      <CheckIcon v-if="checked" class="ck-icon" :size="14" />
    </span>
  </span>
</template>

<style scoped>
.ck-input {
  position: absolute;
  opacity: 0;
  width: 16px;
  height: 16px;
  margin: 0;
  cursor: pointer;
}
.ck-input:disabled {
  cursor: not-allowed;
}
.ck-box-wrapper {
  position: relative;
  width: 16px;
  height: 16px;
  flex: none;
}
.ck-box {
  display: inline-grid;
  place-items: center;
  box-sizing: border-box;
  width: 16px;
  height: 16px;
  border-radius: var(--radius-md, 4px);
  border: 1.5px solid var(--admin-border, var(--surface-300));
  background: var(--admin-surface, var(--surface-0));
  transition: border-color 0.15s, background 0.15s;
  pointer-events: none;
}
.ck-icon {
  display: block;
  width: 14px;
  height: 14px;
  flex: none;
}
.ck-box[data-checked="true"] {
  background: var(--brand-600);
  border-color: var(--brand-600);
  color: var(--admin-on-brand);
}
.ck-input:focus-visible + .ck-box {
  outline: 2px solid var(--brand-500);
  outline-offset: 2px;
}
.ck-input:disabled + .ck-box {
  opacity: 0.5;
}
</style>
