<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  Search, ShoppingBag, Heart, User, PackageSearch,
  Sun, Moon, Palette, Check,
} from 'lucide-vue-next'
import { useCartStore } from '@/shared/stores/cart'
import { useUserStore } from '@/shared/stores/user'
import { useThemeStore } from '@/shared/stores/theme'
import { useUiStore } from '@/shared/stores/ui'
import type { AccentColor } from '@/shared/stores/theme'

const cart = useCartStore()
const userStore = useUserStore()
const theme = useThemeStore()
const ui = useUiStore()

const itemCount = computed(() => cart.itemCount)
const favCount = computed(() => cart.favorites.length)
const isLoggedIn = computed(() => !!userStore.user)

const localSearch = ref('')
const showPalette = ref(false)

function onSearchInput() {
  // Broadcast search query to ProductGrid island via a CustomEvent.
  // (Search is the one piece of state that doesn't belong in a store
  //  because it's page-scoped and ephemeral.)
  window.dispatchEvent(new CustomEvent('minimal-cart:search', { detail: localSearch.value }))
}

function scrollToShop() {
  document.getElementById('shop')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

const accentSwatches: Record<AccentColor, string> = {
  amber:   '#E89538',
  emerald: '#33A06B',
  rose:    '#E0556B',
  sky:     '#2DA6F0',
  violet:  '#8B5CE6',
}
</script>

<template>
  <header class="sticky top-0 z-40 border-b border-border/60 bg-background/80 backdrop-blur-lg">
    <div class="mx-auto flex h-16 max-w-7xl items-center gap-3 px-4 sm:px-6 lg:px-8">
      <!-- Logo -->
      <button @click="scrollToShop" class="flex items-center gap-2">
        <span class="grid h-8 w-8 place-items-center rounded-full bg-cta text-sm font-bold text-cta-foreground">質</span>
        <span class="hidden text-sm font-semibold tracking-tight sm:block">質物選物</span>
      </button>

      <!-- Search (desktop) -->
      <div class="relative hidden flex-1 md:block">
        <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <input
          v-model="localSearch"
          @input="onSearchInput"
          type="text"
          placeholder="搜尋商品..."
          class="h-9 w-full rounded-full border border-border bg-muted/40 pl-9 pr-4 text-sm outline-none transition-colors placeholder:text-muted-foreground focus:border-cta focus:bg-background"
        />
      </div>

      <!-- Nav icons -->
      <div class="ml-auto flex items-center gap-1">
        <!-- Theme: dark mode toggle -->
        <button
          @click="theme.toggleMode()"
          class="grid h-9 w-9 place-items-center rounded-full text-foreground transition-colors hover:bg-muted"
          :aria-label="theme.mode === 'dark' ? '切換亮色模式' : '切換暗色模式'"
        >
          <Sun v-if="theme.mode === 'dark'" class="h-4 w-4" />
          <Moon v-else class="h-4 w-4" />
        </button>

        <!-- Theme: accent color picker -->
        <div class="relative">
          <button
            @click="showPalette = !showPalette"
            class="grid h-9 w-9 place-items-center rounded-full text-foreground transition-colors hover:bg-muted"
            aria-label="主色切換"
          >
            <Palette class="h-4 w-4" />
          </button>
          <Transition name="fade">
            <div
              v-if="showPalette"
              class="absolute right-0 top-11 z-50 w-44 rounded-xl border border-border bg-popover p-3 shadow-lg"
              @click.stop
            >
              <p class="mb-2 text-xs font-medium text-muted-foreground">主色切換</p>
              <div class="space-y-1">
                <button
                  v-for="(preset, id) in theme.accents"
                  :key="id"
                  @click="theme.setAccent(id as AccentColor); showPalette = false"
                  class="flex w-full items-center gap-2.5 rounded-lg px-2 py-1.5 text-sm transition-colors hover:bg-muted"
                >
                  <span
                    class="h-4 w-4 rounded-full border border-border/40"
                    :style="{ backgroundColor: accentSwatches[id as AccentColor] }"
                  />
                  <span class="flex-1 text-left">{{ preset.label }}</span>
                  <Check v-if="theme.accent === id" class="h-3.5 w-3.5 text-cta" />
                </button>
              </div>
            </div>
          </Transition>
        </div>

        <button
          @click="ui.openTrackOrder()"
          class="grid h-9 w-9 place-items-center rounded-full text-foreground transition-colors hover:bg-muted"
          aria-label="訂單查詢"
        >
          <PackageSearch class="h-4 w-4" />
        </button>

        <button
          @click="scrollToShop"
          class="relative grid h-9 w-9 place-items-center rounded-full text-foreground transition-colors hover:bg-muted"
          aria-label="收藏清單"
        >
          <Heart class="h-4 w-4" />
          <span v-if="favCount > 0" class="absolute -right-0.5 -top-0.5 grid h-4 min-w-4 place-items-center rounded-full bg-cta px-1 text-[10px] font-medium text-cta-foreground">
            {{ favCount }}
          </span>
        </button>

        <button
          @click="ui.openAccount()"
          class="grid h-9 w-9 place-items-center rounded-full text-foreground transition-colors hover:bg-muted"
          aria-label="帳號"
        >
          <User class="h-4 w-4" />
        </button>

        <button
          @click="ui.openCart()"
          class="relative grid h-9 w-9 place-items-center rounded-full text-foreground transition-colors hover:bg-muted"
          aria-label="購物車"
        >
          <ShoppingBag class="h-4 w-4" />
          <span v-if="itemCount > 0" class="absolute -right-0.5 -top-0.5 grid h-4 min-w-4 place-items-center rounded-full bg-cta px-1 text-[10px] font-medium text-cta-foreground">
            {{ itemCount }}
          </span>
        </button>
      </div>
    </div>

    <!-- Search (mobile) -->
    <div class="border-t border-border/60 px-4 py-2 md:hidden">
      <div class="relative">
        <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <input
          v-model="localSearch"
          @input="onSearchInput"
          type="text"
          placeholder="搜尋商品..."
          class="h-9 w-full rounded-full border border-border bg-muted/40 pl-9 pr-4 text-sm outline-none transition-colors placeholder:text-muted-foreground focus:border-cta focus:bg-background"
        />
      </div>
    </div>
  </header>
</template>
