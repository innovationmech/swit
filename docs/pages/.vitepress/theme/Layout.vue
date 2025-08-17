<template>
  <div class="vp-layout" :class="{ 'high-contrast': highContrast, 'reduce-motion': reduceMotion }">
    <!-- 可访问性控制组件 -->
    <AccessibilityControls />
    
    <!-- 性能监控 -->
    <PerformanceMonitor />
    
    <!-- 用户反馈 -->
    <FeedbackWidget />
    
    <!-- 安全配置 -->
    <SecurityConfig />
    
    <!-- 隐私政策 -->
    <PrivacyPolicy />
    
    <!-- 使用默认主题布局，但添加可访问性增强 -->
    <DefaultLayout>
      <!-- 添加语义化 HTML 结构 -->
      <template #layout-top>
        <div id="main-content" tabindex="-1" style="position: absolute; top: -1px; left: -1px; width: 1px; height: 1px; opacity: 0;"></div>
      </template>
    </DefaultLayout>
    
    <!-- ARIA live region for announcements -->
    <div
      id="aria-live-region"
      aria-live="polite"
      aria-atomic="true"
      class="sr-only"
    ></div>
    
    <!-- Focus trap for modals and panels -->
    <div id="focus-trap" tabindex="-1" style="position: absolute; left: -10000px;"></div>
  </div>
</template>

<script setup>
import { ref, onMounted, provide } from 'vue'
import DefaultTheme from 'vitepress/theme'
import AccessibilityControls from './components/AccessibilityControls.vue'
import PerformanceMonitor from './components/PerformanceMonitor.vue'
import FeedbackWidget from './components/FeedbackWidget.vue'
import SecurityConfig from './components/SecurityConfig.vue'
import PrivacyPolicy from './components/PrivacyPolicy.vue'

const DefaultLayout = DefaultTheme.Layout

// 可访问性状态
const highContrast = ref(false)
const reduceMotion = ref(false)

// 提供可访问性状态给子组件
provide('accessibility', {
  highContrast,
  reduceMotion
})

// 键盘导航助手
const setupKeyboardNavigation = () => {
  // 为主要地标添加快捷键导航
  const landmarkKeys = {
    '1': () => focusElement('#main-content'),
    '2': () => focusElement('nav[role="navigation"], .vp-nav'),
    '3': () => focusElement('aside, .vp-sidebar'),
    '4': () => focusElement('footer[role="contentinfo"]'),
    's': () => focusElement('.VPNavBarSearch, .DocSearch-Button'),
    'h': () => focusElement('h1, h2, h3, h4, h5, h6'),
    't': () => scrollToTop()
  }
  
  document.addEventListener('keydown', (event) => {
    // Alt + number/letter keys for landmark navigation
    if (event.altKey && landmarkKeys[event.key]) {
      event.preventDefault()
      landmarkKeys[event.key]()
    }
  })
}

// 焦点管理工具
const focusElement = (selector) => {
  const element = document.querySelector(selector)
  if (element) {
    element.focus()
    element.scrollIntoView({ 
      behavior: reduceMotion.value ? 'auto' : 'smooth',
      block: 'start'
    })
  }
}

// 滚动到顶部
const scrollToTop = () => {
  window.scrollTo({
    top: 0,
    behavior: reduceMotion.value ? 'auto' : 'smooth'
  })
}

// ARIA live region 管理
const announceToScreenReader = (message, priority = 'polite') => {
  const liveRegion = document.getElementById('aria-live-region')
  if (liveRegion) {
    liveRegion.setAttribute('aria-live', priority)
    liveRegion.textContent = message
    
    // 清除消息以便下次使用
    setTimeout(() => {
      liveRegion.textContent = ''
    }, 1000)
  }
}

// 路由变化通知
const setupRouteAnnouncements = () => {
  // 监听路由变化
  if (typeof window !== 'undefined') {
    const observer = new MutationObserver((mutations) => {
      mutations.forEach((mutation) => {
        if (mutation.type === 'childList' && mutation.target.tagName === 'TITLE') {
          const title = document.title
          announceToScreenReader(`Navigated to ${title}`)
        }
      })
    })
    
    observer.observe(document.head, {
      childList: true,
      subtree: true
    })
  }
}

// 焦点管理
const setupFocusManagement = () => {
  // 处理模态框和弹出层的焦点管理
  document.addEventListener('focusin', (event) => {
    const activeModals = document.querySelectorAll('.accessibility-panel[aria-modal="true"]')
    if (activeModals.length > 0) {
      const currentModal = activeModals[activeModals.length - 1]
      if (!currentModal.contains(event.target)) {
        // 如果焦点离开了模态框，将其重新设置到模态框内
        const firstFocusable = currentModal.querySelector(
          'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        )
        if (firstFocusable) {
          firstFocusable.focus()
        }
      }
    }
  })
}

// 表单验证增强
const enhanceFormAccessibility = () => {
  document.addEventListener('invalid', (event) => {
    const field = event.target
    const errorMessage = field.validationMessage
    
    // 创建或更新错误消息
    let errorElement = field.parentNode.querySelector('.error-message')
    if (!errorElement) {
      errorElement = document.createElement('span')
      errorElement.className = 'error-message'
      errorElement.setAttribute('role', 'alert')
      field.parentNode.appendChild(errorElement)
    }
    
    errorElement.textContent = errorMessage
    field.setAttribute('aria-describedby', errorElement.id || 'error-' + Math.random())
    field.setAttribute('aria-invalid', 'true')
  })
  
  document.addEventListener('input', (event) => {
    const field = event.target
    if (field.checkValidity()) {
      field.removeAttribute('aria-invalid')
      const errorElement = field.parentNode.querySelector('.error-message')
      if (errorElement) {
        errorElement.remove()
      }
    }
  })
}

// 组件挂载时设置可访问性功能
onMounted(() => {
  // 初始化可访问性设置
  const savedContrast = localStorage.getItem('swit-high-contrast')
  const savedMotion = localStorage.getItem('swit-reduce-motion')
  
  highContrast.value = savedContrast === 'true'
  reduceMotion.value = savedMotion === 'true'
  
  // 检测系统偏好
  if (window.matchMedia) {
    const contrastQuery = window.matchMedia('(prefers-contrast: high)')
    const motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
    
    if (contrastQuery.matches) {
      highContrast.value = true
    }
    
    if (motionQuery.matches) {
      reduceMotion.value = true
    }
    
    // 监听系统偏好变化
    contrastQuery.addEventListener('change', (e) => {
      if (e.matches) {
        highContrast.value = true
        document.documentElement.classList.add('high-contrast')
      }
    })
    
    motionQuery.addEventListener('change', (e) => {
      if (e.matches) {
        reduceMotion.value = true
        document.documentElement.classList.add('reduce-motion')
      }
    })
  }
  
  // 设置各种可访问性功能
  setupKeyboardNavigation()
  setupRouteAnnouncements()
  setupFocusManagement()
  enhanceFormAccessibility()
  
  // 为所有图片添加 alt 文本检查
  const images = document.querySelectorAll('img:not([alt])')
  images.forEach(img => {
    console.warn('Image missing alt text:', img.src)
    img.setAttribute('alt', 'Image description missing')
    img.setAttribute('role', 'presentation')
  })
  
  // 为外部链接添加指示器
  const externalLinks = document.querySelectorAll('a[href^="http"]:not([href*="innovationmech.github.io"])')
  externalLinks.forEach(link => {
    link.setAttribute('target', '_blank')
    link.setAttribute('rel', 'noopener noreferrer')
    link.setAttribute('aria-label', `${link.textContent} (opens in new tab)`)
  })
  
  // 设置页面标题更新通知
  const titleObserver = new MutationObserver(() => {
    announceToScreenReader(`Page title changed to ${document.title}`)
  })
  
  const titleElement = document.querySelector('title')
  if (titleElement) {
    titleObserver.observe(titleElement, { childList: true })
  }
  
  // 增强表格可访问性
  const tables = document.querySelectorAll('table')
  tables.forEach((table, index) => {
    if (!table.querySelector('caption')) {
      const caption = document.createElement('caption')
      caption.textContent = `Table ${index + 1}`
      table.insertBefore(caption, table.firstChild)
    }
    
    // 为表头添加 scope 属性
    const headers = table.querySelectorAll('th')
    headers.forEach(th => {
      if (!th.getAttribute('scope')) {
        const isColumnHeader = th.parentNode === table.querySelector('tr')
        th.setAttribute('scope', isColumnHeader ? 'col' : 'row')
      }
    })
  })
})

// 暴露工具函数给全局使用
if (typeof window !== 'undefined') {
  window.announceToScreenReader = announceToScreenReader
  window.focusElement = focusElement
}
</script>

<style scoped>
.vp-layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

main[role="main"] {
  flex: 1;
  outline: none;
}

/* 确保焦点管理的视觉反馈 */
main[role="main"]:focus {
  outline: 2px solid transparent; /* 隐藏默认焦点轮廓 */
}

/* 高对比度模式下的调整 */
.high-contrast {
  filter: contrast(150%);
}

/* 减少动画的调整 */
.reduce-motion * {
  animation-duration: 0.01ms !important;
  animation-iteration-count: 1 !important;
  transition-duration: 0.01ms !important;
  scroll-behavior: auto !important;
}
</style>