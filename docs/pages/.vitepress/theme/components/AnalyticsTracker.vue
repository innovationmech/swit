<template>
  <div class="analytics-tracker">
    <!-- 隐私同意横幅 -->
    <Transition name="banner">
      <div v-if="showConsentBanner" class="consent-banner">
        <div class="consent-content">
          <div class="consent-text">
            <h4>{{ isZh ? '数据收集和隐私' : 'Data Collection and Privacy' }}</h4>
            <p>
              {{ isZh 
                ? '我们使用分析工具来改善网站体验。您可以选择是否允许数据收集。' 
                : 'We use analytics to improve the website experience. You can choose whether to allow data collection.' 
              }}
            </p>
          </div>
          <div class="consent-actions">
            <button @click="acceptAnalytics" class="consent-btn accept">
              {{ isZh ? '接受' : 'Accept' }}
            </button>
            <button @click="declineAnalytics" class="consent-btn decline">
              {{ isZh ? '拒绝' : 'Decline' }}
            </button>
            <button @click="showPrivacySettings" class="consent-btn settings">
              {{ isZh ? '设置' : 'Settings' }}
            </button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- 隐私设置模态框 -->
    <Transition name="modal">
      <div v-if="showSettings" class="privacy-modal" @click="closeSettings">
        <div class="privacy-content" @click.stop>
          <div class="privacy-header">
            <h3>{{ isZh ? '隐私设置' : 'Privacy Settings' }}</h3>
            <button @click="closeSettings" class="close-btn">
              <Icon name="x" />
            </button>
          </div>
          
          <div class="privacy-body">
            <div class="privacy-section">
              <h4>{{ isZh ? '数据收集选项' : 'Data Collection Options' }}</h4>
              
              <div class="privacy-option">
                <label class="option-label">
                  <input 
                    type="checkbox" 
                    v-model="preferences.analytics"
                    @change="updatePreferences"
                  >
                  <span class="option-text">
                    <strong>{{ isZh ? '网站分析' : 'Website Analytics' }}</strong>
                    <br>
                    {{ isZh 
                      ? '收集匿名使用数据以改善网站性能和用户体验' 
                      : 'Collect anonymous usage data to improve website performance and user experience' 
                    }}
                  </span>
                </label>
              </div>
              
              <div class="privacy-option">
                <label class="option-label">
                  <input 
                    type="checkbox" 
                    v-model="preferences.performance"
                    @change="updatePreferences"
                  >
                  <span class="option-text">
                    <strong>{{ isZh ? '性能监控' : 'Performance Monitoring' }}</strong>
                    <br>
                    {{ isZh 
                      ? '监控页面性能以识别和修复技术问题' 
                      : 'Monitor page performance to identify and fix technical issues' 
                    }}
                  </span>
                </label>
              </div>
              
              <div class="privacy-option">
                <label class="option-label">
                  <input 
                    type="checkbox" 
                    v-model="preferences.errorTracking"
                    @change="updatePreferences"
                  >
                  <span class="option-text">
                    <strong>{{ isZh ? '错误追踪' : 'Error Tracking' }}</strong>
                    <br>
                    {{ isZh 
                      ? '自动报告技术错误以提高网站稳定性' 
                      : 'Automatically report technical errors to improve website stability' 
                    }}
                  </span>
                </label>
              </div>
            </div>
            
            <div class="privacy-section">
              <h4>{{ isZh ? '数据保护说明' : 'Data Protection Notice' }}</h4>
              <div class="privacy-notice">
                <ul>
                  <li>{{ isZh ? '我们不收集个人身份信息' : 'We do not collect personal identifiable information' }}</li>
                  <li>{{ isZh ? '所有数据都经过匿名化处理' : 'All data is anonymized' }}</li>
                  <li>{{ isZh ? '您可以随时更改这些设置' : 'You can change these settings at any time' }}</li>
                  <li>{{ isZh ? '数据仅用于改善网站体验' : 'Data is only used to improve website experience' }}</li>
                </ul>
              </div>
            </div>
          </div>
          
          <div class="privacy-footer">
            <button @click="saveSettings" class="privacy-btn primary">
              {{ isZh ? '保存设置' : 'Save Settings' }}
            </button>
            <button @click="closeSettings" class="privacy-btn secondary">
              {{ isZh ? '取消' : 'Cancel' }}
            </button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- 隐私控制按钮 -->
    <button 
      v-if="hasConsentData" 
      @click="showPrivacySettings" 
      class="privacy-control-btn"
      :title="isZh ? '隐私设置' : 'Privacy Settings'"
    >
      <Icon name="shield" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed, watch } from 'vue'
import { useData, useRoute } from 'vitepress'

interface AnalyticsEvent {
  category: string
  action: string
  label?: string
  value?: number
  custom_parameters?: Record<string, any>
}

interface UserPreferences {
  analytics: boolean
  performance: boolean
  errorTracking: boolean
  timestamp: number
}

interface AnalyticsConfig {
  googleAnalyticsId?: string
  sentryDsn?: string
  enableDebug: boolean
}

// 组件状态
const showConsentBanner = ref(false)
const showSettings = ref(false)
const hasConsentData = ref(false)

// 用户偏好设置
const preferences = reactive<UserPreferences>({
  analytics: false,
  performance: false,
  errorTracking: false,
  timestamp: Date.now()
})

// VitePress 数据
const { site, lang } = useData()
const route = useRoute()

// 计算属性
const isZh = computed(() => lang.value === 'zh')

// 分析配置
const analyticsConfig: AnalyticsConfig = {
  googleAnalyticsId: 'G-XXXXXXXXXX', // 替换为实际的 GA4 ID
  sentryDsn: 'https://xxx@xxx.ingest.sentry.io/xxx', // 替换为实际的 Sentry DSN
  enableDebug: import.meta.env.DEV
}

// 生命周期
onMounted(async () => {
  await initializeAnalytics()
  setupRouteTracking()
})

// 监听路由变化
watch(() => route.path, (newPath, oldPath) => {
  if (newPath !== oldPath) {
    trackPageView(newPath)
  }
})

// 初始化分析系统
async function initializeAnalytics() {
  try {
    // 加载用户偏好
    loadUserPreferences()
    
    // 检查是否需要显示同意横幅
    if (!hasConsentData.value) {
      showConsentBanner.value = true
    } else {
      // 根据用户偏好初始化分析工具
      await initializeTrackingServices()
    }
    
    if (analyticsConfig.enableDebug) {
      console.log('Analytics initialized:', preferences)
    }
    
  } catch (error) {
    console.error('Failed to initialize analytics:', error)
  }
}

function loadUserPreferences() {
  try {
    const stored = localStorage.getItem('vitepress-analytics-preferences')
    if (stored) {
      const parsed = JSON.parse(stored)
      Object.assign(preferences, parsed)
      hasConsentData.value = true
    }
  } catch (error) {
    console.warn('Failed to load analytics preferences:', error)
  }
}

function saveUserPreferences() {
  try {
    localStorage.setItem('vitepress-analytics-preferences', JSON.stringify(preferences))
    hasConsentData.value = true
  } catch (error) {
    console.error('Failed to save analytics preferences:', error)
  }
}

async function initializeTrackingServices() {
  // 初始化 Google Analytics
  if (preferences.analytics && analyticsConfig.googleAnalyticsId) {
    await initializeGoogleAnalytics()
  }
  
  // 初始化 Sentry 错误追踪
  if (preferences.errorTracking && analyticsConfig.sentryDsn) {
    await initializeSentry()
  }
  
  // 初始化性能监控
  if (preferences.performance) {
    initializePerformanceMonitoring()
  }
}

async function initializeGoogleAnalytics() {
  try {
    // 动态加载 GA4 脚本
    const script = document.createElement('script')
    script.async = true
    script.src = `https://www.googletagmanager.com/gtag/js?id=${analyticsConfig.googleAnalyticsId}`
    document.head.appendChild(script)
    
    // 初始化 gtag
    ;(window as any).dataLayer = (window as any).dataLayer || []
    const gtag = function(...args: any[]) {
      ;(window as any).dataLayer.push(args)
    }
    ;(window as any).gtag = gtag
    
    gtag('js', new Date())
    gtag('config', analyticsConfig.googleAnalyticsId, {
      // 隐私友好配置
      anonymize_ip: true,
      allow_google_signals: false,
      allow_ad_personalization_signals: false,
      cookie_expires: 60 * 60 * 24 * 30, // 30 天
      custom_map: {
        'custom_dimension_1': 'user_language',
        'custom_dimension_2': 'page_theme'
      }
    })
    
    if (analyticsConfig.enableDebug) {
      console.log('Google Analytics initialized')
    }
    
  } catch (error) {
    console.error('Failed to initialize Google Analytics:', error)
  }
}

async function initializeSentry() {
  try {
    // 动态导入 Sentry，使用可选导入
    const { init, BrowserTracing } = await import('@sentry/browser').catch(() => {
      console.warn('Sentry not available, error tracking disabled')
      return { init: null, BrowserTracing: null }
    })
    
    if (!init) return
    
    init({
      dsn: analyticsConfig.sentryDsn,
      environment: import.meta.env.MODE,
      integrations: BrowserTracing ? [new BrowserTracing()] : [],
      tracesSampleRate: 0.1,
      beforeSend(event) {
        // 隐私过滤 - 移除敏感信息
        if (event.user) {
          delete event.user.email
          delete event.user.username
        }
        return event
      }
    })
    
    if (analyticsConfig.enableDebug) {
      console.log('Sentry initialized')
    }
    
  } catch (error) {
    console.error('Failed to initialize Sentry:', error)
  }
}

function initializePerformanceMonitoring() {
  try {
    // Web Vitals 监控
    if ('PerformanceObserver' in window) {
      // Core Web Vitals 收集
      const observer = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          trackEvent({
            category: 'Performance',
            action: 'Core Web Vitals',
            label: entry.entryType,
            value: Math.round(entry.startTime),
            custom_parameters: {
              metric_type: entry.entryType,
              metric_value: entry.startTime
            }
          })
        }
      })
      
      observer.observe({ entryTypes: ['largest-contentful-paint', 'first-input'] })
    }
    
    // 页面加载性能
    window.addEventListener('load', () => {
      setTimeout(() => {
        const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming
        if (navigation) {
          trackEvent({
            category: 'Performance',
            action: 'Page Load',
            label: 'Load Time',
            value: Math.round(navigation.loadEventEnd - navigation.loadEventStart),
            custom_parameters: {
              dom_content_loaded: navigation.domContentLoadedEventEnd - navigation.domContentLoadedEventStart,
              first_byte: navigation.responseStart - navigation.requestStart
            }
          })
        }
      }, 1000)
    })
    
    if (analyticsConfig.enableDebug) {
      console.log('Performance monitoring initialized')
    }
    
  } catch (error) {
    console.error('Failed to initialize performance monitoring:', error)
  }
}

function setupRouteTracking() {
  // 初始页面视图
  trackPageView(route.path)
  
  // 用户行为追踪
  setupUserInteractionTracking()
}

function setupUserInteractionTracking() {
  if (!preferences.analytics) return
  
  // 搜索行为追踪
  document.addEventListener('click', (event) => {
    const target = event.target as HTMLElement
    
    // 搜索按钮点击
    if (target.closest('.DocSearch-Button, .VPNavBarSearch')) {
      trackEvent({
        category: 'User Interaction',
        action: 'Search',
        label: 'Open Search'
      })
    }
    
    // 导航链接点击
    if (target.closest('nav a')) {
      const link = target.closest('a') as HTMLAnchorElement
      trackEvent({
        category: 'Navigation',
        action: 'Click',
        label: link.textContent || 'Unknown Link',
        custom_parameters: {
          href: link.href,
          internal: link.host === window.location.host
        }
      })
    }
    
    // 代码复制
    if (target.closest('.copy-code-button')) {
      trackEvent({
        category: 'User Interaction',
        action: 'Copy Code',
        label: 'Code Block'
      })
    }
  })
  
  // 滚动深度追踪
  let maxScrollDepth = 0
  const trackScrollDepth = () => {
    const scrollDepth = Math.round((window.scrollY / (document.body.scrollHeight - window.innerHeight)) * 100)
    if (scrollDepth > maxScrollDepth && scrollDepth % 25 === 0) {
      maxScrollDepth = scrollDepth
      trackEvent({
        category: 'User Engagement',
        action: 'Scroll Depth',
        label: `${scrollDepth}%`,
        value: scrollDepth
      })
    }
  }
  
  window.addEventListener('scroll', trackScrollDepth, { passive: true })
}

// 事件追踪
function trackEvent(event: AnalyticsEvent) {
  if (!preferences.analytics) return
  
  try {
    // Google Analytics 事件
    if ((window as any).gtag) {
      ;(window as any).gtag('event', event.action, {
        event_category: event.category,
        event_label: event.label,
        value: event.value,
        ...event.custom_parameters
      })
    }
    
    if (analyticsConfig.enableDebug) {
      console.log('Analytics event:', event)
    }
    
  } catch (error) {
    console.error('Failed to track event:', error)
  }
}

function trackPageView(path: string) {
  if (!preferences.analytics) return
  
  try {
    if ((window as any).gtag) {
      ;(window as any).gtag('config', analyticsConfig.googleAnalyticsId, {
        page_path: path,
        page_title: document.title,
        page_location: window.location.href,
        custom_map: {
          user_language: lang.value,
          page_theme: document.documentElement.classList.contains('dark') ? 'dark' : 'light'
        }
      })
    }
    
    if (analyticsConfig.enableDebug) {
      console.log('Page view tracked:', path)
    }
    
  } catch (error) {
    console.error('Failed to track page view:', error)
  }
}

function trackError(error: Error, context?: string) {
  if (!preferences.errorTracking) return
  
  try {
    // Sentry 错误报告
    if ((window as any).Sentry) {
      ;(window as any).Sentry.captureException(error, {
        tags: {
          context: context || 'unknown'
        }
      })
    }
    
    // Google Analytics 错误事件
    trackEvent({
      category: 'Error',
      action: 'JavaScript Error',
      label: error.message,
      custom_parameters: {
        error_stack: error.stack,
        error_context: context
      }
    })
    
  } catch (trackingError) {
    console.error('Failed to track error:', trackingError)
  }
}

// 同意横幅控制
function acceptAnalytics() {
  preferences.analytics = true
  preferences.performance = true
  preferences.errorTracking = true
  preferences.timestamp = Date.now()
  
  saveUserPreferences()
  showConsentBanner.value = false
  
  // 初始化追踪服务
  initializeTrackingServices()
}

function declineAnalytics() {
  preferences.analytics = false
  preferences.performance = false
  preferences.errorTracking = false
  preferences.timestamp = Date.now()
  
  saveUserPreferences()
  showConsentBanner.value = false
}

function showPrivacySettings() {
  showSettings.value = true
}

function closeSettings() {
  showSettings.value = false
}

function updatePreferences() {
  preferences.timestamp = Date.now()
}

function saveSettings() {
  saveUserPreferences()
  showSettings.value = false
  
  // 重新初始化追踪服务
  initializeTrackingServices()
  
  trackEvent({
    category: 'Privacy',
    action: 'Update Settings',
    custom_parameters: {
      analytics_enabled: preferences.analytics,
      performance_enabled: preferences.performance,
      error_tracking_enabled: preferences.errorTracking
    }
  })
}

// 全局错误处理
window.addEventListener('error', (event) => {
  trackError(new Error(event.message), 'Global Error Handler')
})

window.addEventListener('unhandledrejection', (event) => {
  trackError(new Error(event.reason), 'Unhandled Promise Rejection')
})

// 暴露追踪函数给其他组件使用
if (typeof window !== 'undefined') {
  ;(window as any).trackEvent = trackEvent
  ;(window as any).trackPageView = trackPageView
  ;(window as any).trackError = trackError
}
</script>

<style scoped>
.analytics-tracker {
  position: relative;
}

.consent-banner {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  background: var(--vp-c-bg);
  border-top: 1px solid var(--vp-c-divider);
  box-shadow: 0 -4px 12px rgba(0, 0, 0, 0.1);
  z-index: 1000;
  padding: 20px;
}

.consent-content {
  max-width: 1200px;
  margin: 0 auto;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 20px;
}

.consent-text h4 {
  margin: 0 0 8px 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.consent-text p {
  margin: 0;
  font-size: 14px;
  color: var(--vp-c-text-2);
  line-height: 1.5;
}

.consent-actions {
  display: flex;
  gap: 12px;
  flex-shrink: 0;
}

.consent-btn {
  padding: 8px 16px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 6px;
  background: var(--vp-c-bg);
  color: var(--vp-c-text-1);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.consent-btn:hover {
  border-color: var(--vp-c-brand);
  color: var(--vp-c-brand);
}

.consent-btn.accept {
  background: var(--vp-c-brand);
  border-color: var(--vp-c-brand);
  color: var(--vp-c-white);
}

.consent-btn.accept:hover {
  background: var(--vp-c-brand-dark);
  border-color: var(--vp-c-brand-dark);
}

.consent-btn.decline {
  color: var(--vp-c-text-2);
}

.consent-btn.decline:hover {
  color: var(--vp-c-text-1);
  border-color: var(--vp-c-text-2);
}

.privacy-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 1001;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.privacy-content {
  background: var(--vp-c-bg);
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
  max-width: 600px;
  width: 100%;
  max-height: 80vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.privacy-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid var(--vp-c-divider);
}

.privacy-header h3 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.close-btn {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--vp-c-text-2);
  padding: 4px;
  border-radius: 4px;
  transition: all 0.2s ease;
}

.close-btn:hover {
  background: var(--vp-c-gray-soft);
  color: var(--vp-c-text-1);
}

.privacy-body {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}

.privacy-section {
  margin-bottom: 32px;
}

.privacy-section h4 {
  margin: 0 0 16px 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.privacy-option {
  margin-bottom: 16px;
}

.option-label {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  cursor: pointer;
  padding: 12px;
  border-radius: 6px;
  transition: background-color 0.2s ease;
}

.option-label:hover {
  background: var(--vp-c-gray-soft);
}

.option-label input[type="checkbox"] {
  margin-top: 2px;
  flex-shrink: 0;
}

.option-text {
  font-size: 14px;
  line-height: 1.5;
  color: var(--vp-c-text-2);
}

.option-text strong {
  color: var(--vp-c-text-1);
  display: block;
  margin-bottom: 4px;
}

.privacy-notice {
  background: var(--vp-c-bg-soft);
  padding: 16px;
  border-radius: 6px;
  font-size: 13px;
  color: var(--vp-c-text-2);
}

.privacy-notice ul {
  margin: 0;
  padding-left: 16px;
}

.privacy-notice li {
  margin-bottom: 8px;
}

.privacy-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 20px 24px;
  border-top: 1px solid var(--vp-c-divider);
}

.privacy-btn {
  padding: 8px 16px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.privacy-btn.primary {
  background: var(--vp-c-brand);
  border-color: var(--vp-c-brand);
  color: var(--vp-c-white);
}

.privacy-btn.primary:hover {
  background: var(--vp-c-brand-dark);
  border-color: var(--vp-c-brand-dark);
}

.privacy-btn.secondary {
  background: var(--vp-c-bg);
  color: var(--vp-c-text-1);
}

.privacy-btn.secondary:hover {
  border-color: var(--vp-c-brand);
  color: var(--vp-c-brand);
}

.privacy-control-btn {
  position: fixed;
  bottom: 20px;
  left: 20px;
  width: 48px;
  height: 48px;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-divider);
  border-radius: 50%;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  z-index: 999;
  transition: all 0.2s ease;
}

.privacy-control-btn:hover {
  border-color: var(--vp-c-brand);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.15);
}

.privacy-control-btn:hover .icon {
  color: var(--vp-c-brand);
}

/* 过渡动画 */
.banner-enter-active,
.banner-leave-active {
  transition: all 0.3s ease;
}

.banner-enter-from,
.banner-leave-to {
  transform: translateY(100%);
  opacity: 0;
}

.modal-enter-active,
.modal-leave-active {
  transition: all 0.3s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .privacy-content,
.modal-leave-to .privacy-content {
  transform: scale(0.9);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .consent-content {
    flex-direction: column;
    text-align: center;
  }
  
  .consent-actions {
    justify-content: center;
  }
  
  .privacy-content {
    margin: 10px;
    max-height: calc(100vh - 20px);
  }
  
  .privacy-control-btn {
    bottom: 80px;
  }
}

/* 滚动条样式 */
.privacy-body::-webkit-scrollbar {
  width: 6px;
}

.privacy-body::-webkit-scrollbar-thumb {
  background: var(--vp-c-divider);
  border-radius: 3px;
}

.privacy-body::-webkit-scrollbar-track {
  background: transparent;
}
</style>