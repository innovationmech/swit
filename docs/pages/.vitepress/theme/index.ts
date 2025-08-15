import { h, App } from 'vue'
import DefaultTheme from 'vitepress/theme'
import type { Theme } from 'vitepress'
import './styles/vars.css'
import './styles/custom.css'
import HomePage from './components/HomePage.vue'
import FeatureCard from './components/FeatureCard.vue'
import CodeExample from './components/CodeExample.vue'

export default {
  extends: DefaultTheme,
  Layout: () => {
    return h(DefaultTheme.Layout, null, {
      // Custom layout slots
    })
  },
  enhanceApp({ app }: { app: App }) {
    // Register custom components globally
    app.component('HomePage', HomePage)
    app.component('FeatureCard', FeatureCard)
    app.component('CodeExample', CodeExample)
  }
} satisfies Theme