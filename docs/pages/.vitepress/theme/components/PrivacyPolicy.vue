<template>
  <div class="privacy-policy">
    <!-- 隐私政策浮动按钮 -->
    <button 
      @click="showPrivacyModal = true" 
      class="privacy-policy-btn"
      :title="isZh ? '隐私政策' : 'Privacy Policy'"
    >
      <Icon name="shield" />
      <span>{{ isZh ? '隐私' : 'Privacy' }}</span>
    </button>
    
    <!-- GDPR Cookie 横幅 -->
    <Transition name="banner">
      <div v-if="showCookieBanner && !hasAcceptedCookies" class="cookie-banner">
        <div class="banner-content">
          <div class="banner-text">
            <h4>{{ isZh ? 'Cookie 和隐私通知' : 'Cookie and Privacy Notice' }}</h4>
            <p>
              {{ isZh 
                ? '我们使用 Cookie 和类似技术来改善您的浏览体验、分析网站流量并个性化内容。您可以选择接受所有 Cookie 或自定义您的偏好。' 
                : 'We use cookies and similar technologies to improve your browsing experience, analyze website traffic, and personalize content. You can choose to accept all cookies or customize your preferences.' 
              }}
            </p>
          </div>
          <div class="banner-actions">
            <button @click="acceptAllCookies" class="cookie-btn accept">
              {{ isZh ? '接受所有' : 'Accept All' }}
            </button>
            <button @click="rejectAllCookies" class="cookie-btn reject">
              {{ isZh ? '拒绝所有' : 'Reject All' }}
            </button>
            <button @click="showCookieSettings" class="cookie-btn customize">
              {{ isZh ? '自定义设置' : 'Customize' }}
            </button>
            <button @click="showPrivacyModal = true" class="cookie-btn link">
              {{ isZh ? '隐私政策' : 'Privacy Policy' }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
    
    <!-- Cookie 设置模态框 -->
    <Transition name="modal">
      <div v-if="showCookieModal" class="cookie-modal" @click="closeCookieModal">
        <div class="modal-content" @click.stop>
          <div class="modal-header">
            <h3>{{ isZh ? 'Cookie 设置' : 'Cookie Settings' }}</h3>
            <button @click="closeCookieModal" class="close-btn">
              <Icon name="x" />
            </button>
          </div>
          
          <div class="modal-body">
            <div class="cookie-category">
              <div class="category-header">
                <h4>{{ isZh ? '必要 Cookie' : 'Necessary Cookies' }}</h4>
                <div class="category-toggle">
                  <span class="always-on">{{ isZh ? '始终启用' : 'Always On' }}</span>
                </div>
              </div>
              <p class="category-description">
                {{ isZh 
                  ? '这些 Cookie 对于网站的基本功能是必需的，不能禁用。它们通常只在响应您的操作时设置，例如设置隐私偏好、登录或填写表单。'
                  : 'These cookies are necessary for the basic functionality of the website and cannot be disabled. They are usually only set in response to actions made by you, such as setting privacy preferences, logging in, or filling in forms.'
                }}
              </p>
            </div>
            
            <div class="cookie-category">
              <div class="category-header">
                <h4>{{ isZh ? '功能 Cookie' : 'Functional Cookies' }}</h4>
                <label class="category-toggle">
                  <input 
                    type="checkbox" 
                    v-model="cookieSettings.functional"
                    @change="updateCookieSettings"
                  >
                  <span class="toggle-slider"></span>
                </label>
              </div>
              <p class="category-description">
                {{ isZh 
                  ? '这些 Cookie 允许网站记住您的选择和偏好，为您提供更个性化的体验。'
                  : 'These cookies allow the website to remember your choices and preferences to provide you with a more personalized experience.'
                }}
              </p>
              <div class="cookie-details">
                <details>
                  <summary>{{ isZh ? '查看详细信息' : 'View Details' }}</summary>
                  <ul>
                    <li>Language Preference: {{ isZh ? '语言偏好设置' : 'Language preference settings' }}</li>
                    <li>Theme Settings: {{ isZh ? '主题和显示设置' : 'Theme and display settings' }}</li>
                    <li>Accessibility Settings: {{ isZh ? '可访问性设置' : 'Accessibility settings' }}</li>
                  </ul>
                </details>
              </div>
            </div>
            
            <div class="cookie-category">
              <div class="category-header">
                <h4>{{ isZh ? '营销 Cookie' : 'Marketing Cookies' }}</h4>
                <label class="category-toggle">
                  <input 
                    type="checkbox" 
                    v-model="cookieSettings.marketing"
                    @change="updateCookieSettings"
                  >
                  <span class="toggle-slider"></span>
                </label>
              </div>
              <p class="category-description">
                {{ isZh 
                  ? '这些 Cookie 用于跟踪网站访问者，目的是显示相关和吸引人的广告。目前本网站不使用营销 Cookie。'
                  : 'These cookies are used to track website visitors with the intention of displaying relevant and engaging advertisements. This website currently does not use marketing cookies.'
                }}
              </p>
            </div>
          </div>
          
          <div class="modal-footer">
            <button @click="saveCookieSettings" class="save-btn">
              {{ isZh ? '保存设置' : 'Save Settings' }}
            </button>
            <button @click="acceptAllCookies" class="accept-all-btn">
              {{ isZh ? '接受所有' : 'Accept All' }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
    
    <!-- 隐私政策模态框 -->
    <Transition name="modal">
      <div v-if="showPrivacyModal" class="privacy-modal" @click="closePrivacyModal">
        <div class="modal-content large" @click.stop>
          <div class="modal-header">
            <h3>{{ isZh ? '隐私政策' : 'Privacy Policy' }}</h3>
            <button @click="closePrivacyModal" class="close-btn">
              <Icon name="x" />
            </button>
          </div>
          
          <div class="modal-body privacy-content">
            <div class="privacy-section">
              <h4>{{ isZh ? '1. 信息收集' : '1. Information Collection' }}</h4>
              <p>
                {{ isZh 
                  ? '我们收集以下类型的信息：(a) 您自愿提供的信息，如反馈和联系信息；(b) 自动收集的技术信息，如 IP 地址、浏览器类型、访问时间和页面浏览记录；(c) 通过 Cookie 和类似技术收集的使用数据。'
                  : 'We collect the following types of information: (a) Information you voluntarily provide, such as feedback and contact information; (b) Automatically collected technical information, such as IP addresses, browser types, access times, and page views; (c) Usage data collected through cookies and similar technologies.'
                }}
              </p>
            </div>
            
            <div class="privacy-section">
              <h4>{{ isZh ? '2. 信息使用' : '2. Information Use' }}</h4>
              <p>
                {{ isZh 
                  ? '我们使用收集的信息来：(a) 提供和改善我们的服务；(b) 分析网站使用情况和性能；(c) 回应您的询问和反馈；(d) 确保网站安全；(e) 遵守法律义务。'
                  : 'We use the collected information to: (a) Provide and improve our services; (b) Analyze website usage and performance; (c) Respond to your inquiries and feedback; (d) Ensure website security; (e) Comply with legal obligations.'
                }}
              </p>
            </div>
            
            <div class="privacy-section">
              <h4>{{ isZh ? '3. 信息共享' : '3. Information Sharing' }}</h4>
              <p>
                {{ isZh 
                  ? '我们不会出售、交易或转让您的个人信息给第三方，除非：(a) 获得您的明确同意；(b) 法律要求；(c) 保护我们的权利和安全。'
                  : 'We do not sell, trade, or transfer your personal information to third parties unless: (a) We have your explicit consent; (b) Required by law; (c) To protect our rights and security.'
                }}
              </p>
            </div>
            
            <div class="privacy-section">
              <h4>{{ isZh ? '4. 数据安全' : '4. Data Security' }}</h4>
              <p>
                {{ isZh 
                  ? '我们实施适当的技术和组织措施来保护您的个人信息，包括：(a) HTTPS 加密传输；(b) 访问控制和身份验证；(c) 定期安全审计；(d) 数据最小化原则；(e) 安全的数据存储和处理。'
                  : 'We implement appropriate technical and organizational measures to protect your personal information, including: (a) HTTPS encrypted transmission; (b) Access control and authentication; (c) Regular security audits; (d) Data minimization principles; (e) Secure data storage and processing.'
                }}
              </p>
            </div>
            
            <div class="privacy-section">
              <h4>{{ isZh ? '5. 您的权利 (GDPR)' : '5. Your Rights (GDPR)' }}</h4>
              <p>
                {{ isZh 
                  ? '如果您位于欧盟，您有以下权利：(a) 访问权：请求获取我们处理的关于您的个人数据；(b) 更正权：请求更正不准确的个人数据；(c) 删除权：请求删除您的个人数据；(d) 限制处理权：请求限制对您个人数据的处理；(e) 数据可携权：请求以结构化、常用和机器可读的格式接收您的个人数据；(f) 反对权：反对处理您的个人数据。'
                  : 'If you are located in the EU, you have the following rights: (a) Right of access: Request access to personal data we process about you; (b) Right of rectification: Request correction of inaccurate personal data; (c) Right of erasure: Request deletion of your personal data; (d) Right to restrict processing: Request restriction of processing of your personal data; (e) Right to data portability: Request to receive your personal data in a structured, commonly used, and machine-readable format; (f) Right to object: Object to the processing of your personal data.'
                }}
              </p>
            </div>
            
            <div class="privacy-section">
              <h4>{{ isZh ? '6. Cookie 政策' : '6. Cookie Policy' }}</h4>
              <p>
                {{ isZh 
                  ? '我们使用 Cookie 来改善您的体验。您可以通过浏览器设置控制 Cookie，但禁用某些 Cookie 可能会影响网站功能。我们使用的 Cookie 类型包括：(a) 必要 Cookie：网站正常运行所必需；(b) 功能 Cookie：记住您的偏好设置。'
                  : 'We use cookies to improve your experience. You can control cookies through your browser settings, but disabling certain cookies may affect website functionality. The types of cookies we use include: (a) Necessary cookies: Essential for proper website operation; (b) Functional cookies: Remember your preference settings.'
                }}
              </p>
            </div>
            
            <div class="privacy-section">
              <h4>{{ isZh ? '7. 联系我们' : '7. Contact Us' }}</h4>
              <p>
                {{ isZh 
                  ? '如果您对此隐私政策有任何问题或关于行使您的权利，请通过以下方式联系我们：'
                  : 'If you have any questions about this privacy policy or about exercising your rights, please contact us:'
                }}
              </p>
              <ul>
                <li>{{ isZh ? '项目仓库：' : 'Project Repository: ' }}<a href="https://github.com/innovationmech/swit" target="_blank">GitHub</a></li>
                <li>{{ isZh ? '问题反馈：' : 'Issue Tracker: ' }}<a href="https://github.com/innovationmech/swit/issues" target="_blank">GitHub Issues</a></li>
              </ul>
            </div>
            
            <div class="privacy-section">
              <h4>{{ isZh ? '8. 政策更新' : '8. Policy Updates' }}</h4>
              <p>
                {{ isZh 
                  ? '我们可能会定期更新此隐私政策。重大更改将通过网站通知或其他适当方式通知您。建议您定期查看此政策以了解任何更改。'
                  : 'We may periodically update this privacy policy. Significant changes will be communicated through website notifications or other appropriate means. We recommend that you regularly review this policy to stay informed of any changes.'
                }}
              </p>
              <p class="last-updated">
                {{ isZh ? '最后更新：2024年1月' : 'Last updated: January 2024' }}
              </p>
            </div>
          </div>
          
          <div class="modal-footer">
            <button @click="closePrivacyModal" class="close-policy-btn">
              {{ isZh ? '我已阅读并理解' : 'I have read and understand' }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { useData } from 'vitepress'
import Icon from './Icon.vue'

interface CookieSettings {
  necessary: boolean
  functional: boolean
  marketing: boolean
}

// 组件状态
const showCookieBanner = ref(true)
const showCookieModal = ref(false)
const showPrivacyModal = ref(false)
const hasAcceptedCookies = ref(false)

// VitePress 数据
const { lang } = useData()
const isZh = computed(() => lang.value === 'zh')

// Cookie 设置
const cookieSettings = reactive<CookieSettings>({
  necessary: true, // 必要 Cookie 始终启用
  functional: false,
  marketing: false
})

// 生命周期
onMounted(() => {
  loadCookieSettings()
  checkCookieConsent()
})

// Cookie 管理
function loadCookieSettings() {
  try {
    const saved = localStorage.getItem('cookie-settings')
    if (saved) {
      const settings = JSON.parse(saved)
      Object.assign(cookieSettings, { ...settings, necessary: true })
    }
    
    const consent = localStorage.getItem('cookie-consent')
    if (consent) {
      hasAcceptedCookies.value = true
      showCookieBanner.value = false
    }
  } catch (error) {
    console.error('Failed to load cookie settings:', error)
  }
}

function saveCookieSettings() {
  try {
    localStorage.setItem('cookie-settings', JSON.stringify(cookieSettings))
    localStorage.setItem('cookie-consent', 'true')
    localStorage.setItem('cookie-consent-date', new Date().toISOString())
    
    hasAcceptedCookies.value = true
    showCookieBanner.value = false
    showCookieModal.value = false
    
    // 应用 Cookie 设置
    applyCookieSettings()
    
    // 触发设置更新事件
    if (typeof window !== 'undefined') {
      window.dispatchEvent(new CustomEvent('cookieSettingsUpdated', {
        detail: cookieSettings
      }))
    }
    
  } catch (error) {
    console.error('Failed to save cookie settings:', error)
  }
}

function acceptAllCookies() {
  cookieSettings.functional = true
  cookieSettings.marketing = false // 保持 marketing 为 false，因为我们不使用营销 Cookie
  
  saveCookieSettings()
}

function rejectAllCookies() {
  cookieSettings.functional = false
  cookieSettings.marketing = false
  
  saveCookieSettings()
}

function showCookieSettings() {
  showCookieModal.value = true
  showCookieBanner.value = false
}

function updateCookieSettings() {
  // 实时更新设置，但不保存直到用户点击保存
  applyCookieSettings()
}

function applyCookieSettings() {
  // 清除现有的非必要 Cookie
  if (!cookieSettings.functional) {
    clearFunctionalCookies()
  }
  
  if (!cookieSettings.marketing) {
    clearMarketingCookies()
  }
  
  // 更新 document.cookie 的安全设置
  updateCookieSecurityPolicy()
}


function clearFunctionalCookies() {
  // 清除功能性 Cookie（保留必要的设置）
  const functionalCookies = ['theme', 'language', 'accessibility-settings']
  functionalCookies.forEach(name => {
    // 注意：不删除这些 Cookie，因为它们对用户体验很重要
    // 但可以在用户明确拒绝时删除
    if (!cookieSettings.functional) {
      deleteCookie(name)
    }
  })
}

function clearMarketingCookies() {
  // 目前没有营销 Cookie，但保留此函数以备将来使用
  const marketingCookies = ['marketing_id', 'ad_preferences']
  marketingCookies.forEach(name => {
    deleteCookie(name)
  })
}

function deleteCookie(name: string) {
  // 删除 Cookie 的标准方法
  if (typeof document !== 'undefined' && typeof window !== 'undefined') {
    document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;`
    document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/; domain=${window.location.hostname};`
    document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/; domain=.${window.location.hostname};`
  }
}

function updateCookieSecurityPolicy() {
  // 确保所有新的 Cookie 都有适当的安全设置
  const originalCookieDescriptor = Object.getOwnPropertyDescriptor(Document.prototype, 'cookie')
  
  Object.defineProperty(document, 'cookie', {
    get: originalCookieDescriptor?.get,
    set(value) {
      let secureCookie = value
      
      // 添加安全属性
      if (location.protocol === 'https:' && !secureCookie.includes('Secure')) {
        secureCookie += '; Secure'
      }
      
      if (!secureCookie.includes('SameSite')) {
        secureCookie += '; SameSite=Lax'
      }
      
      // 为敏感 Cookie 添加 HttpOnly
      if ((secureCookie.includes('session') || secureCookie.includes('auth')) && !secureCookie.includes('HttpOnly')) {
        secureCookie += '; HttpOnly'
      }
      
      originalCookieDescriptor?.set?.call(this, secureCookie)
    }
  })
}

function checkCookieConsent() {
  const consentDate = localStorage.getItem('cookie-consent-date')
  if (consentDate) {
    const date = new Date(consentDate)
    const now = new Date()
    const daysDiff = (now.getTime() - date.getTime()) / (1000 * 3600 * 24)
    
    // 如果同意已超过 365 天，重新显示横幅
    if (daysDiff > 365) {
      hasAcceptedCookies.value = false
      showCookieBanner.value = true
      localStorage.removeItem('cookie-consent')
      localStorage.removeItem('cookie-consent-date')
    }
  }
}

// 模态框控制
function closeCookieModal() {
  showCookieModal.value = false
}

function closePrivacyModal() {
  showPrivacyModal.value = false
}

// 暴露 Cookie 管理功能给其他组件
if (typeof window !== 'undefined') {
  ;(window as any).cookieManager = {
    getSettings: () => cookieSettings,
    updateSettings: (settings: Partial<CookieSettings>) => {
      Object.assign(cookieSettings, settings)
      applyCookieSettings()
    },
    showSettings: () => {
      showCookieModal.value = true
    },
    hasConsent: () => hasAcceptedCookies.value
  }
}
</script>

<style scoped>
.privacy-policy {
  position: relative;
}

.privacy-policy-btn {
  position: fixed;
  bottom: 200px;
  right: 20px;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 14px;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-divider);
  border-radius: 20px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  color: var(--vp-c-text-1);
  transition: all 0.3s ease;
  z-index: 998;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.privacy-policy-btn:hover {
  background: var(--vp-c-bg-soft);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.cookie-banner {
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

.banner-content {
  max-width: 1200px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
}

.banner-text h4 {
  margin: 0 0 8px 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.banner-text p {
  margin: 0;
  font-size: 14px;
  color: var(--vp-c-text-2);
  line-height: 1.5;
  max-width: 600px;
}

.banner-actions {
  display: flex;
  gap: 12px;
  flex-shrink: 0;
}

.cookie-btn {
  padding: 8px 16px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 6px;
  background: var(--vp-c-bg);
  color: var(--vp-c-text-1);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.cookie-btn:hover {
  border-color: var(--vp-c-brand);
  color: var(--vp-c-brand);
}

.cookie-btn.accept {
  background: var(--vp-c-brand);
  border-color: var(--vp-c-brand);
  color: white;
}

.cookie-btn.accept:hover {
  background: var(--vp-c-brand-dark);
  border-color: var(--vp-c-brand-dark);
}

.cookie-btn.reject {
  color: var(--vp-c-text-2);
}

.cookie-btn.reject:hover {
  color: var(--vp-c-text-1);
  border-color: var(--vp-c-text-2);
}

.cookie-btn.link {
  background: transparent;
  border: none;
  color: var(--vp-c-brand);
  text-decoration: underline;
}

.cookie-modal,
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

.modal-content {
  background: var(--vp-c-bg);
  border-radius: 12px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
  max-width: 600px;
  width: 100%;
  max-height: 80vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.modal-content.large {
  max-width: 800px;
  max-height: 90vh;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 24px 28px;
  background: var(--vp-c-bg-soft);
  border-bottom: 1px solid var(--vp-c-divider);
}

.modal-header h3 {
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
  padding: 6px;
  border-radius: 4px;
  transition: all 0.2s ease;
}

.close-btn:hover {
  background: var(--vp-c-gray-soft);
  color: var(--vp-c-text-1);
}

.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 28px;
}

.cookie-category {
  margin-bottom: 32px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--vp-c-divider-light);
}

.cookie-category:last-child {
  border-bottom: none;
  margin-bottom: 0;
}

.category-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.category-header h4 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.category-toggle {
  display: flex;
  align-items: center;
}

.always-on {
  font-size: 12px;
  color: var(--vp-c-text-3);
  font-weight: 500;
}

.category-toggle input[type="checkbox"] {
  display: none;
}

.toggle-slider {
  position: relative;
  width: 44px;
  height: 24px;
  background: var(--vp-c-gray-light);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.3s ease;
}

.toggle-slider::before {
  content: '';
  position: absolute;
  top: 2px;
  left: 2px;
  width: 20px;
  height: 20px;
  background: white;
  border-radius: 50%;
  transition: all 0.3s ease;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
}

.category-toggle input[type="checkbox"]:checked + .toggle-slider {
  background: var(--vp-c-brand);
}

.category-toggle input[type="checkbox"]:checked + .toggle-slider::before {
  transform: translateX(20px);
}

.category-description {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: var(--vp-c-text-2);
  line-height: 1.5;
}

.cookie-details details {
  margin-top: 8px;
}

.cookie-details summary {
  cursor: pointer;
  font-size: 13px;
  color: var(--vp-c-brand);
  font-weight: 500;
}

.cookie-details ul {
  margin: 8px 0 0 20px;
  padding: 0;
}

.cookie-details li {
  font-size: 13px;
  color: var(--vp-c-text-3);
  margin-bottom: 4px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 20px 28px;
  background: var(--vp-c-bg-soft);
  border-top: 1px solid var(--vp-c-divider);
}

.save-btn,
.accept-all-btn,
.close-policy-btn {
  padding: 10px 20px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.save-btn {
  background: var(--vp-c-brand);
  border-color: var(--vp-c-brand);
  color: white;
}

.save-btn:hover {
  background: var(--vp-c-brand-dark);
  border-color: var(--vp-c-brand-dark);
}

.accept-all-btn {
  background: var(--vp-c-bg);
  color: var(--vp-c-text-1);
}

.accept-all-btn:hover {
  border-color: var(--vp-c-brand);
  color: var(--vp-c-brand);
}

.close-policy-btn {
  background: var(--vp-c-brand);
  border-color: var(--vp-c-brand);
  color: white;
}

.close-policy-btn:hover {
  background: var(--vp-c-brand-dark);
  border-color: var(--vp-c-brand-dark);
}

/* 隐私政策内容样式 */
.privacy-content {
  font-size: 14px;
  line-height: 1.6;
}

.privacy-section {
  margin-bottom: 24px;
}

.privacy-section h4 {
  margin: 0 0 12px 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.privacy-section p {
  margin: 0 0 12px 0;
  color: var(--vp-c-text-2);
}

.privacy-section ul {
  margin: 12px 0;
  padding-left: 20px;
}

.privacy-section li {
  margin-bottom: 6px;
  color: var(--vp-c-text-2);
}

.privacy-section a {
  color: var(--vp-c-brand);
  text-decoration: none;
}

.privacy-section a:hover {
  text-decoration: underline;
}

.last-updated {
  font-style: italic;
  color: var(--vp-c-text-3);
  font-size: 13px;
  margin-top: 16px;
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

.modal-enter-from .modal-content,
.modal-leave-to .modal-content {
  transform: scale(0.9);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .privacy-policy-btn {
    bottom: 80px;
    right: 20px;
  }
  
  .banner-content {
    flex-direction: column;
    text-align: center;
  }
  
  .banner-actions {
    flex-wrap: wrap;
    justify-content: center;
  }
  
  .modal-content {
    margin: 10px;
    max-height: calc(100vh - 20px);
  }
  
  .modal-content.large {
    max-height: calc(100vh - 20px);
  }
  
  .modal-header,
  .modal-body,
  .modal-footer {
    padding-left: 20px;
    padding-right: 20px;
  }
}

/* 滚动条样式 */
.modal-body::-webkit-scrollbar {
  width: 6px;
}

.modal-body::-webkit-scrollbar-thumb {
  background: var(--vp-c-divider);
  border-radius: 3px;
}

.modal-body::-webkit-scrollbar-track {
  background: transparent;
}
</style>