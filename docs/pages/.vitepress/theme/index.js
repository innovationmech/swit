// 自定义主题入口文件
import DefaultTheme from 'vitepress/theme'
import './styles/vars.css'
import './styles/custom.css'
import HomePage from './components/HomePage.vue'
import FeatureCard from './components/FeatureCard.vue'
import CodeExample from './components/CodeExample.vue'
import ApiDocViewer from './components/ApiDocViewer.vue'
import SearchBox from './components/SearchBox.vue'
import AccessibilityControls from './components/AccessibilityControls.vue'
// 删除了不必要的组件：PerformanceMonitor, FeedbackWidget, SecurityConfig, PrivacyPolicy
import Layout from './Layout.vue'

export default {
  extends: DefaultTheme,
  Layout,
  
  // 注册全局组件
  enhanceApp({ app, router, siteData }) {
    // 注册所有自定义组件
    app.component('HomePage', HomePage)
    app.component('FeatureCard', FeatureCard)
    app.component('CodeExample', CodeExample)
    app.component('ApiDocViewer', ApiDocViewer)
    app.component('SearchBox', SearchBox)
    app.component('AccessibilityControls', AccessibilityControls)
    // 已删除不必要的组件注册
  }
}