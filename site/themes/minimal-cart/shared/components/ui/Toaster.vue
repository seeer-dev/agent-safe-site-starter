<script setup lang="ts">
import { X } from 'lucide-vue-next'
import { useToast } from '@/shared/composables/use-toast'
import { cn } from '@/shared/lib/utils'

const { toasts, dismiss } = useToast()
</script>

<template>
  <Teleport to="body">
    <div class="fixed top-0 z-[100] flex max-h-screen w-full flex-col-reverse items-end gap-2 p-4 sm:bottom-0 sm:right-0 sm:top-auto sm:flex-col sm:max-w-[420px]">
      <TransitionGroup name="toast">
        <div
          v-for="t in toasts"
          :key="t.id"
          :class="cn(
            'group pointer-events-auto relative flex w-full items-center justify-between gap-3 overflow-hidden rounded-md border p-4 pr-8 shadow-lg transition-all',
            t.variant === 'destructive'
              ? 'border-destructive/50 bg-destructive text-destructive-foreground'
              : 'border bg-background text-foreground',
          )"
        >
          <div class="grid gap-1">
            <div v-if="t.title" class="text-sm font-semibold">{{ t.title }}</div>
            <div v-if="t.description" class="text-sm opacity-90">{{ t.description }}</div>
          </div>
          <button
            class="absolute right-1 top-1 rounded-md p-1 text-foreground/50 opacity-0 transition-opacity hover:text-foreground focus:opacity-100 group-hover:opacity-100"
            @click="dismiss(t.id)"
          >
            <X class="h-4 w-4" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>
