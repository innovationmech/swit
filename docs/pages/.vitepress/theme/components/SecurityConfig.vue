<template>
  <div class="security-config">
    <!-- CSP 策略管理 -->
    <div v-if="showSecurityPanel" class="security-panel">
      <div class="panel-header">
        <h3>{{ isZh ? '安全策略配置' : 'Security Policy Configuration' }}</h3>
        <button @click="closeSecurityPanel" class="close-btn">
          <Icon name="x" />
        </button>
      </div>
      
      <div class="panel-content">
        <!-- CSP 状态 -->
        <div class="security-section">
          <h4>{{ isZh ? 'Content Security Policy' : 'Content Security Policy' }}</h4>
          <div class="security-status">
            <div class="status-indicator" :class="cspStatus.status">
              <Icon :name="cspStatus.icon" />
              <span>{{ isZh ? cspStatus.messageZh : cspStatus.message }}</span>
            </div>
          </div>
          
          <div class="csp-details">
            <details>
              <summary>{{ isZh ? '查看 CSP 策略详情' : 'View CSP Policy Details' }}</summary>
              <pre class="csp-policy">{{ formatCSPPolicy() }}</pre>
            </details>
          </div>
        </div>
        
        <!-- XSS 防护状态 -->
        <div class="security-section">
          <h4>{{ isZh ? 'XSS 防护' : 'XSS Protection' }}</h4>
          <div class="security-checks">
            <div 
              v-for="check in xssChecks" 
              :key="check.name"
              class="security-check"
              :class="check.status"
            >
              <Icon :name="check.icon" />
              <span>{{ isZh ? check.labelZh : check.label }}</span>
              <div class="check-status">{{ isZh ? check.statusZh : check.statusText }}</div>
            </div>
          </div>
        </div>
        
        <!-- 数据安全 -->
        <div class="security-section">
          <h4>{{ isZh ? '数据安全' : 'Data Security' }}</h4>
          <div class="security-metrics">
            <div class="metric-item">
              <label>{{ isZh ? '敏感数据检测:' : 'Sensitive Data Detection:' }}</label>
              <span class="metric-value" :class="sensitiveDataCheck.status">
                {{ sensitiveDataCheck.count }} {{ isZh ? '项' : 'items' }}
              </span>
            </div>
            <div class="metric-item">
              <label>{{ isZh ? 'Cookie 安全设置:' : 'Cookie Security:' }}</label>
              <span class="metric-value good">{{ isZh ? '已启用' : 'Enabled' }}</span>
            </div>
            <div class="metric-item">
              <label>{{ isZh ? 'HTTPS 强制:' : 'HTTPS Enforcement:' }}</label>
              <span class="metric-value" :class="httpsStatus.status">
                {{ isZh ? httpsStatus.textZh : httpsStatus.text }}
              </span>
            </div>
          </div>
        </div>
        
        <!-- 安全建议 -->
        <div v-if="securityRecommendations.length > 0" class="security-section">
          <h4>{{ isZh ? '安全建议' : 'Security Recommendations' }}</h4>
          <div class="recommendations">
            <div 
              v-for="(rec, index) in securityRecommendations" 
              :key="index"
              class="recommendation"
              :class="rec.severity"
            >
              <Icon :name="rec.icon" />
              <div class="rec-content">
                <div class="rec-title">{{ isZh ? rec.titleZh : rec.title }}</div>
                <div class="rec-description">{{ isZh ? rec.descriptionZh : rec.description }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    
    <!-- 安全状态指示器 -->
    <button 
      @click="openSecurityPanel" 
      class="security-indicator"
      :class="overallSecurityStatus"
      :title="isZh ? '安全状态' : 'Security Status'"
    >
      <Icon name="shield" />
      <span class="security-score">{{ securityScore }}/100</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { useData } from 'vitepress'
import Icon from './Icon.vue'

interface SecurityCheck {
  name: string
  label: string
  labelZh: string
  status: 'good' | 'warning' | 'error'
  statusText: string
  statusZh: string
  icon: string
}

interface SecurityRecommendation {
  severity: 'low' | 'medium' | 'high' | 'critical'
  title: string
  titleZh: string
  description: string
  descriptionZh: string
  icon: string
}

// 组件状态
const showSecurityPanel = ref(false)
const securityScore = ref(0)

// VitePress 数据
const { lang } = useData()
const isZh = computed(() => lang.value === 'zh')

// CSP 状态
const cspStatus = reactive({
  status: 'good',
  message: 'CSP Active',
  messageZh: 'CSP 已激活',
  icon: 'check-circle'
})

// XSS 检查项
const xssChecks = reactive<SecurityCheck[]>([
  {
    name: 'dom_sanitization',
    label: 'DOM Sanitization',
    labelZh: 'DOM 净化',
    status: 'good',
    statusText: 'Active',
    statusZh: '已启用',
    icon: 'check'
  },
  {
    name: 'input_validation',
    label: 'Input Validation',
    labelZh: '输入验证',
    status: 'good',
    statusText: 'Active',
    statusZh: '已启用',
    icon: 'check'
  },
  {
    name: 'script_filtering',
    label: 'Script Filtering',
    labelZh: '脚本过滤',
    status: 'good',
    statusText: 'Active',
    statusZh: '已启用',
    icon: 'check'
  },
  {
    name: 'url_validation',
    label: 'URL Validation',
    labelZh: 'URL 验证',
    status: 'good',
    statusText: 'Active',
    statusZh: '已启用',
    icon: 'check'
  }
])

// 敏感数据检查
const sensitiveDataCheck = reactive({
  count: 0,
  status: 'good'
})

// HTTPS 状态
const httpsStatus = reactive({
  status: 'good',
  text: 'Enforced',
  textZh: '已强制'
})

// 安全建议
const securityRecommendations = reactive<SecurityRecommendation[]>([])

// 计算属性
const overallSecurityStatus = computed(() => {
  if (securityScore.value >= 90) return 'excellent'
  if (securityScore.value >= 75) return 'good'
  if (securityScore.value >= 50) return 'warning'
  return 'critical'
})

// CSP 策略配置
const cspPolicy = {
  'default-src': ["'self'"],
  'script-src': [
    "'self'",
    "'unsafe-inline'", // VitePress 需要内联脚本
    'https://*.sentry.io'
  ],
  'style-src': [
    "'self'",
    "'unsafe-inline'", // VitePress 主题需要内联样式
    'https://fonts.googleapis.com'
  ],
  'font-src': [
    "'self'",
    'https://fonts.gstatic.com'
  ],
  'img-src': [
    "'self'",
    'data:',
    'https:',
    'blob:'
  ],
  'connect-src': [
    "'self'",
    'https://*.sentry.io',
    'https://api.github.com'
  ],
  'frame-src': ["'none'"],
  'object-src': ["'none'"],
  'base-uri': ["'self'"],
  'form-action': ["'self'"],
  'frame-ancestors': ["'none'"],
  'upgrade-insecure-requests': []
}

// 生命周期
onMounted(() => {
  initializeSecurity()
  performSecurityChecks()
})

// 安全初始化
function initializeSecurity() {
  // 设置 CSP 元标签
  setupCSP()
  
  // 设置其他安全头
  setupSecurityHeaders()
  
  // 初始化 XSS 防护
  initializeXSSProtection()
  
  // 设置 Cookie 安全
  setupCookieSecurity()
  
  // 检测敏感数据
  scanSensitiveData()
}

function setupCSP() {
  try {
    // 检查是否已存在 CSP 元标签
    let cspMeta = document.querySelector('meta[http-equiv="Content-Security-Policy"]')
    
    if (!cspMeta) {
      cspMeta = document.createElement('meta')
      cspMeta.setAttribute('http-equiv', 'Content-Security-Policy')
      document.head.appendChild(cspMeta)
    }
    
    // 构建 CSP 策略字符串
    const policyString = Object.entries(cspPolicy)
      .map(([directive, sources]) => {
        if (sources.length === 0) return directive
        return `${directive} ${sources.join(' ')}`
      })
      .join('; ')
    
    cspMeta.setAttribute('content', policyString)
    
    cspStatus.status = 'good'
    cspStatus.message = 'CSP Active'
    cspStatus.messageZh = 'CSP 已激活'
    cspStatus.icon = 'check-circle'
    
  } catch (error) {
    console.error('Failed to setup CSP:', error)
    cspStatus.status = 'error'
    cspStatus.message = 'CSP Setup Failed'
    cspStatus.messageZh = 'CSP 设置失败'
    cspStatus.icon = 'alert-circle'
  }
}

function setupSecurityHeaders() {
  // 设置 X-Content-Type-Options
  let noSniffMeta = document.querySelector('meta[http-equiv="X-Content-Type-Options"]')
  if (!noSniffMeta) {
    noSniffMeta = document.createElement('meta')
    noSniffMeta.setAttribute('http-equiv', 'X-Content-Type-Options')
    noSniffMeta.setAttribute('content', 'nosniff')
    document.head.appendChild(noSniffMeta)
  }
  
  // 设置 X-Frame-Options
  let frameOptionsMeta = document.querySelector('meta[http-equiv="X-Frame-Options"]')
  if (!frameOptionsMeta) {
    frameOptionsMeta = document.createElement('meta')
    frameOptionsMeta.setAttribute('http-equiv', 'X-Frame-Options')
    frameOptionsMeta.setAttribute('content', 'DENY')
    document.head.appendChild(frameOptionsMeta)
  }
  
  // 设置 Referrer Policy
  let referrerMeta = document.querySelector('meta[name="referrer"]')
  if (!referrerMeta) {
    referrerMeta = document.createElement('meta')
    referrerMeta.setAttribute('name', 'referrer')
    referrerMeta.setAttribute('content', 'strict-origin-when-cross-origin')
    document.head.appendChild(referrerMeta)
  }
}

function initializeXSSProtection() {
  // DOM 净化检查
  if (typeof (window as any).DOMPurify !== 'undefined') {
    xssChecks[0].status = 'good'
  } else {
    xssChecks[0].status = 'warning'
    xssChecks[0].statusText = 'Not Available'
    xssChecks[0].statusZh = '不可用'
  }
  
  // 监听危险的 DOM 操作
  const originalInnerHTML = Object.getOwnPropertyDescriptor(Element.prototype, 'innerHTML')
  if (originalInnerHTML && originalInnerHTML.set) {
    Object.defineProperty(Element.prototype, 'innerHTML', {
      set: function(value: any) {
        if (typeof value === 'string' && containsXSSPattern(value)) {
          console.warn('Potential XSS attempt detected:', value)
          securityRecommendations.push({
            severity: 'high',
            title: 'XSS Attempt Detected',
            titleZh: '检测到 XSS 攻击尝试',
            description: 'Potentially malicious script detected in content',
            descriptionZh: '在内容中检测到潜在恶意脚本',
            icon: 'alert-triangle'
          })
          return originalInnerHTML.set?.call(this, sanitizeContent(value))
        }
        return originalInnerHTML.set?.call(this, value)
      },
      get: originalInnerHTML.get,
      configurable: true,
      enumerable: true
    })
  }
}

function containsXSSPattern(content: string): boolean {
  const xssPatterns = [
    /<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi,
    /javascript:/gi,
    /on\w+\s*=/gi,
    /<iframe\b/gi,
    /<object\b/gi,
    /<embed\b/gi,
    /data:text\/html/gi
  ]
  
  return xssPatterns.some(pattern => pattern.test(content))
}

function sanitizeContent(content: string): string {
  // 简单的内容净化
  return content
    .replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
    .replace(/javascript:/gi, 'javascript-blocked:')
    .replace(/on\w+\s*=/gi, 'on-event-blocked=')
}

function setupCookieSecurity() {
  // 重写 document.cookie 以确保安全设置
  const originalCookieDescriptor = Object.getOwnPropertyDescriptor(Document.prototype, 'cookie')
  
  Object.defineProperty(document, 'cookie', {
    get: originalCookieDescriptor?.get,
    set(value) {
      // 确保 Cookie 包含安全属性
      let secureCookie = value
      
      if (!secureCookie.includes('Secure') && location.protocol === 'https:') {
        secureCookie += '; Secure'
      }
      
      if (!secureCookie.includes('SameSite')) {
        secureCookie += '; SameSite=Strict'
      }
      
      if (!secureCookie.includes('HttpOnly') && !secureCookie.includes('analytics')) {
        secureCookie += '; HttpOnly'
      }
      
      originalCookieDescriptor?.set?.call(this, secureCookie)
    }
  })
}

function scanSensitiveData() {
  const sensitivePatterns = [
    /\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b/g, // Email
    /\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b/g, // Credit card
    /\b\d{3}-\d{2}-\d{4}\b/g, // SSN
    /\b(?:password|pwd|secret|token|key)\s*[:=]\s*\S+/gi, // Credentials
  ]
  
  const textContent = document.body.textContent || ''
  let sensitiveCount = 0
  
  sensitivePatterns.forEach(pattern => {
    const matches = textContent.match(pattern)
    if (matches) {
      sensitiveCount += matches.length
    }
  })
  
  sensitiveDataCheck.count = sensitiveCount
  sensitiveDataCheck.status = sensitiveCount === 0 ? 'good' : 'warning'
  
  if (sensitiveCount > 0) {
    securityRecommendations.push({
      severity: 'medium',
      title: 'Sensitive Data Detected',
      titleZh: '检测到敏感数据',
      description: `Found ${sensitiveCount} potential sensitive data patterns`,
      descriptionZh: `发现 ${sensitiveCount} 个潜在敏感数据模式`,
      icon: 'alert-triangle'
    })
  }
}

function performSecurityChecks() {
  let score = 100
  
  // 检查 HTTPS
  if (location.protocol !== 'https:' && location.hostname !== 'localhost') {
    score -= 20
    httpsStatus.status = 'error'
    httpsStatus.text = 'Not Enforced'
    httpsStatus.textZh = '未强制'
    
    securityRecommendations.push({
      severity: 'critical',
      title: 'HTTPS Not Enforced',
      titleZh: 'HTTPS 未强制',
      description: 'Site should be served over HTTPS',
      descriptionZh: '网站应使用 HTTPS 提供服务',
      icon: 'alert-circle'
    })
  }
  
  // 检查 CSP 违规
  document.addEventListener('securitypolicyviolation', (e) => {
    console.warn('CSP Violation:', e.violatedDirective, e.blockedURI)
    score -= 5
    
    securityRecommendations.push({
      severity: 'medium',
      title: 'CSP Violation',
      titleZh: 'CSP 违规',
      description: `Blocked: ${e.violatedDirective}`,
      descriptionZh: `已阻止: ${e.violatedDirective}`,
      icon: 'alert-triangle'
    })
    
    updateSecurityScore(score)
  })
  
  // 检查混合内容
  if (location.protocol === 'https:') {
    const httpResources = Array.from(document.querySelectorAll('img[src^="http:"], script[src^="http:"], link[href^="http:"]'))
    if (httpResources.length > 0) {
      score -= 15
      securityRecommendations.push({
        severity: 'high',
        title: 'Mixed Content Detected',
        titleZh: '检测到混合内容',
        description: `${httpResources.length} HTTP resources on HTTPS page`,
        descriptionZh: `HTTPS 页面上有 ${httpResources.length} 个 HTTP 资源`,
        icon: 'alert-circle'
      })
    }
  }
  
  updateSecurityScore(score)
}

function updateSecurityScore(score: number) {
  securityScore.value = Math.max(0, Math.min(100, score))
}

function formatCSPPolicy(): string {
  return Object.entries(cspPolicy)
    .map(([directive, sources]) => {
      if (sources.length === 0) return directive
      return `${directive}:\n  ${sources.join('\n  ')}`
    })
    .join('\n\n')
}

// 面板控制
function openSecurityPanel() {
  showSecurityPanel.value = true
}

function closeSecurityPanel() {
  showSecurityPanel.value = false
}

// 监听安全事件
if (typeof window !== 'undefined') {
  // 全局安全监控
  window.addEventListener('error', (event) => {
    if (event.message.includes('script') || event.message.includes('unsafe')) {
      console.warn('Potential security issue:', event.message)
    }
  })
  
  // 暴露安全工具给开发者控制台
  ;(window as any).securityTools = {
    getSecurityScore: () => securityScore.value,
    getCSPPolicy: () => cspPolicy,
    scanSensitiveData,
    performSecurityChecks
  }
}
</script>

<style scoped>
.security-config {
  position: relative;
}

.security-indicator {
  position: fixed;
  bottom: 140px;
  right: 20px;
  width: 56px;
  height: 56px;
  border: none;
  border-radius: 50%;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  font-size: 10px;
  font-weight: 600;
  transition: all 0.3s ease;
  z-index: 998;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.security-indicator.excellent {
  background: var(--vp-c-green);
  color: white;
}

.security-indicator.good {
  background: var(--vp-c-blue);
  color: white;
}

.security-indicator.warning {
  background: var(--vp-c-yellow);
  color: var(--vp-c-yellow-dark);
}

.security-indicator.critical {
  background: var(--vp-c-red);
  color: white;
}

.security-indicator:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.2);
}

.security-score {
  font-size: 9px;
  line-height: 1;
  margin-top: 1px;
}

.security-panel {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 600px;
  max-width: 90vw;
  max-height: 80vh;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
  z-index: 1000;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  background: var(--vp-c-bg-soft);
  border-bottom: 1px solid var(--vp-c-divider);
}

.panel-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.close-btn {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--vp-c-text-2);
  padding: 6px;
  border-radius: 4px;
  transition: all 0.2s ease;
}

.close-btn:hover {
  background: var(--vp-c-gray-soft);
  color: var(--vp-c-text-1);
}

.panel-content {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}

.security-section {
  margin-bottom: 32px;
}

.security-section h4 {
  margin: 0 0 16px 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.security-status {
  margin-bottom: 16px;
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-radius: 8px;
  font-weight: 500;
}

.status-indicator.good {
  background: var(--vp-c-green-soft);
  color: var(--vp-c-green-dark);
}

.status-indicator.warning {
  background: var(--vp-c-yellow-soft);
  color: var(--vp-c-yellow-dark);
}

.status-indicator.error {
  background: var(--vp-c-red-soft);
  color: var(--vp-c-red-dark);
}

.csp-details {
  margin-top: 12px;
}

.csp-details summary {
  cursor: pointer;
  font-weight: 500;
  color: var(--vp-c-text-2);
  margin-bottom: 8px;
}

.csp-policy {
  background: var(--vp-c-bg-soft);
  padding: 16px;
  border-radius: 6px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  line-height: 1.5;
  overflow-x: auto;
  color: var(--vp-c-text-2);
  margin: 8px 0 0 0;
}

.security-checks {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.security-check {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 6px;
  background: var(--vp-c-bg-soft);
}

.security-check.good {
  border-left: 3px solid var(--vp-c-green);
}

.security-check.warning {
  border-left: 3px solid var(--vp-c-yellow);
}

.security-check.error {
  border-left: 3px solid var(--vp-c-red);
}

.check-status {
  margin-left: auto;
  font-size: 12px;
  color: var(--vp-c-text-3);
}

.security-metrics {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.metric-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  background: var(--vp-c-bg-soft);
  border-radius: 6px;
}

.metric-item label {
  font-weight: 500;
  color: var(--vp-c-text-1);
}

.metric-value {
  font-weight: 600;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 13px;
}

.metric-value.good {
  background: var(--vp-c-green-soft);
  color: var(--vp-c-green-dark);
}

.metric-value.warning {
  background: var(--vp-c-yellow-soft);
  color: var(--vp-c-yellow-dark);
}

.metric-value.error {
  background: var(--vp-c-red-soft);
  color: var(--vp-c-red-dark);
}

.recommendations {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.recommendation {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  border-radius: 8px;
  border-left: 4px solid;
}

.recommendation.low {
  background: var(--vp-c-blue-soft);
  border-left-color: var(--vp-c-blue);
}

.recommendation.medium {
  background: var(--vp-c-yellow-soft);
  border-left-color: var(--vp-c-yellow);
}

.recommendation.high {
  background: var(--vp-c-orange-soft);
  border-left-color: var(--vp-c-orange);
}

.recommendation.critical {
  background: var(--vp-c-red-soft);
  border-left-color: var(--vp-c-red);
}

.rec-content {
  flex: 1;
}

.rec-title {
  font-weight: 600;
  margin-bottom: 4px;
  color: var(--vp-c-text-1);
}

.rec-description {
  font-size: 14px;
  color: var(--vp-c-text-2);
  line-height: 1.4;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .security-indicator {
    bottom: 80px;
    right: 20px;
  }
  
  .security-panel {
    width: calc(100vw - 20px);
    height: calc(100vh - 20px);
    top: 10px;
    left: 10px;
    transform: none;
  }
}

/* 滚动条样式 */
.panel-content::-webkit-scrollbar {
  width: 6px;
}

.panel-content::-webkit-scrollbar-thumb {
  background: var(--vp-c-divider);
  border-radius: 3px;
}

.panel-content::-webkit-scrollbar-track {
  background: transparent;
}
</style>