import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { VueQueryPlugin } from '@tanstack/vue-query'
import { applyAdminThemeToDocument } from '@sitecore/admin-ui/theme'
import App from './App.vue'
import { router } from './app/router'
import { i18n } from './app/i18n'
import { ADMIN_RUNTIME_KEY } from './app/runtime'
import { createFixtureRuntime } from './app/runtime.fixture'
import './styles.css'

applyAdminThemeToDocument({
  packKey: 'admin-mint',
  brand: 'teal',
  mode: 'light',
  density: 'comfortable',
}, i18n.global.locale.value)

const app = createApp(App)
const runtime = createFixtureRuntime()
app.provide(ADMIN_RUNTIME_KEY, runtime)
app.use(createPinia()).use(VueQueryPlugin).use(router).use(i18n).mount('#app')
