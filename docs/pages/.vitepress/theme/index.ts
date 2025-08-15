import { h } from 'vue'
import DefaultTheme from 'vitepress/theme'
import type { Theme } from 'vitepress'

// Import custom components
import ApiExplorer from './components/ApiExplorer.vue'
import ApiEndpoint from './components/ApiEndpoint.vue'
import CodeExample from './components/CodeExample.vue'
import FeatureCard from './components/FeatureCard.vue'
import HomePage from './components/HomePage.vue'

// Import custom styles
import './styles/vars.css'
import './styles/custom.css'

export default {
  extends: DefaultTheme,
  Layout: () => {
    return h(DefaultTheme.Layout, null, {
      // Custom layout slots if needed
    })
  },
  enhanceApp({ app, router, siteData }) {
    // Register global components
    app.component('ApiExplorer', ApiExplorer)
    app.component('ApiEndpoint', ApiEndpoint)
    app.component('CodeExample', CodeExample)
    app.component('FeatureCard', FeatureCard)
    app.component('HomePage', HomePage)
  }
} satisfies Theme