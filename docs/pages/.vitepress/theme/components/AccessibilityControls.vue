<template>
  <div class="accessibility-controls">
    <!-- 跳转到主内容链接 -->
    <a href="#main-content" class="skip-to-content" @click="skipToMain">
      {{ t('accessibility.skipToMain') }}
    </a>
    
    <!-- 字体大小控制 -->
    <div class="font-size-controls" role="region" :aria-label="t('accessibility.fontSizeControls')">
      <button
        v-for="size in fontSizes"
        :key="size.key"
        class="font-size-btn"
        :class="{ active: currentFontSize === size.key }"
        :aria-label="t('accessibility.setFontSize', { size: size.label })"
        :aria-pressed="currentFontSize === size.key"
        @click="setFontSize(size.key)"
      >
        {{ size.display }}
      </button>
    </div>
    
    <!-- 高对比度切换 -->
    <button
      class="contrast-toggle"
      :aria-label="t('accessibility.toggleContrast')"
      :aria-pressed="highContrast"
      @click="toggleContrast"
    >
      <svg
        v-if="!highContrast"
        width="24"
        height="24"
        viewBox="0 0 24 24"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        aria-hidden="true"
      >
        <path
          d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"
          fill="currentColor"
        />
      </svg>
      <svg
        v-else
        width="24"
        height="24"
        viewBox="0 0 24 24"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        aria-hidden="true"
      >
        <path
          d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zM12 20V4c4.41 0 8 3.59 8 8s-3.59 8-8 8z"
          fill="currentColor"
        />
      </svg>
    </button>
    
    <!-- 辅助功能设置面板 -->
    <div v-if="showSettingsPanel" class="accessibility-panel" role="dialog" :aria-label="t('accessibility.settingsPanel')">
      <div class="panel-header">
        <h3>{{ t('accessibility.accessibilitySettings') }}</h3>
        <button class="close-btn" @click="showSettingsPanel = false" :aria-label="t('accessibility.closePanel')">
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M12 4L4 12M4 4l8 8" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          </svg>
        </button>
      </div>
      
      <div class="panel-content">
        <!-- 字体大小设置 -->
        <fieldset>
          <legend>{{ t('accessibility.fontSize') }}</legend>
          <div class="radio-group">
            <label v-for="size in fontSizes" :key="size.key" class="radio-label">
              <input
                type="radio"
                :value="size.key"
                v-model="currentFontSize"
                @change="setFontSize(size.key)"
                class="radio-input"
              />
              <span class="radio-custom"></span>
              {{ size.label }}
            </label>
          </div>
        </fieldset>
        
        <!-- 对比度设置 -->
        <fieldset>
          <legend>{{ t('accessibility.contrast') }}</legend>
          <label class="checkbox-label">
            <input
              type="checkbox"
              v-model="highContrast"
              @change="toggleContrast"
              class="checkbox-input"
            />
            <span class="checkbox-custom"></span>
            {{ t('accessibility.highContrast') }}
          </label>
        </fieldset>
        
        <!-- 动画设置 */
        <fieldset>
          <legend>{{ t('accessibility.animations') }}</legend>
          <label class="checkbox-label">
            <input
              type="checkbox"
              v-model="reduceMotion"
              @change="toggleMotion"
              class="checkbox-input"
            />
            <span class="checkbox-custom"></span>
            {{ t('accessibility.reduceMotion') }}
          </label>
        </fieldset>
      </div>
    </div>
    
    <!-- 设置按钮 -->
    <button
      class="settings-toggle"
      :aria-label="t('accessibility.openSettings')"
      @click="showSettingsPanel = !showSettingsPanel"
    >
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path
          d="M10 12.5a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5z"
          stroke="currentColor"
          stroke-width="1.5"
        />
        <path
          d="M10 1.875A8.125 8.125 0 0 0 1.875 10H.625v1.25h1.25A8.125 8.125 0 0 0 10 18.125v1.25h1.25v-1.25A8.125 8.125 0 0 0 18.125 11.25h1.25V10h-1.25A8.125 8.125 0 0 0 11.25 1.875V.625H10v1.25z"
          stroke="currentColor"
          stroke-width="1.5"
        />
      </svg>
    </button>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vitepress'

// 响应式状态
const currentFontSize = ref('normal')
const highContrast = ref(false)
const reduceMotion = ref(false)
const showSettingsPanel = ref(false)

// 字体大小选项
const fontSizes = [
  { key: 'small', label: '小', display: 'A-' },
  { key: 'normal', label: '正常', display: 'A' },
  { key: 'large', label: '大', display: 'A+' },
  { key: 'xlarge', label: '超大', display: 'A++' }
]

// 国际化函数（简化版本）
const t = (key, params = {}) => {
  const translations = {
    'accessibility.skipToMain': 'Skip to main content',
    'accessibility.fontSizeControls': 'Font size controls',
    'accessibility.setFontSize': `Set font size to ${params.size || ''}`,
    'accessibility.toggleContrast': 'Toggle high contrast mode',
    'accessibility.settingsPanel': 'Accessibility settings panel',
    'accessibility.accessibilitySettings': 'Accessibility Settings',
    'accessibility.closePanel': 'Close settings panel',
    'accessibility.fontSize': 'Font Size',
    'accessibility.contrast': 'Contrast',
    'accessibility.highContrast': 'High contrast mode',
    'accessibility.animations': 'Animations',
    'accessibility.reduceMotion': 'Reduce motion',
    'accessibility.openSettings': 'Open accessibility settings'
  }
  
  return translations[key] || key
}

// 设置字体大小
const setFontSize = (size) => {
  currentFontSize.value = size
  document.documentElement.className = document.documentElement.className
    .replace(/font-(small|normal|large|xlarge)/g, '')
    .trim()
  document.documentElement.classList.add(`font-${size}`)
  
  // 保存到本地存储
  localStorage.setItem('swit-font-size', size)
  
  // 通知屏幕阅读器
  announceToScreenReader(`Font size changed to ${fontSizes.find(f => f.key === size)?.label}`)
}

// 切换高对比度
const toggleContrast = () => {
  highContrast.value = !highContrast.value
  
  if (highContrast.value) {
    document.documentElement.classList.add('high-contrast')
  } else {
    document.documentElement.classList.remove('high-contrast')
  }
  
  // 保存到本地存储
  localStorage.setItem('swit-high-contrast', highContrast.value.toString())
  
  // 通知屏幕阅读器
  announceToScreenReader(`High contrast mode ${highContrast.value ? 'enabled' : 'disabled'}`)
}

// 切换动画减少
const toggleMotion = () => {
  reduceMotion.value = !reduceMotion.value
  
  if (reduceMotion.value) {
    document.documentElement.classList.add('reduce-motion')
  } else {
    document.documentElement.classList.remove('reduce-motion')
  }
  
  // 保存到本地存储
  localStorage.setItem('swit-reduce-motion', reduceMotion.value.toString())
  
  // 通知屏幕阅读器
  announceToScreenReader(`Motion reduction ${reduceMotion.value ? 'enabled' : 'disabled'}`)
}

// 跳转到主内容
const skipToMain = (event) => {
  event.preventDefault()
  const mainContent = document.getElementById('main-content') || document.querySelector('main')
  if (mainContent) {
    mainContent.focus()
    mainContent.scrollIntoView({ behavior: 'smooth' })
  }
}

// 向屏幕阅读器宣布消息
const announceToScreenReader = (message) => {
  const announcement = document.createElement('div')
  announcement.setAttribute('aria-live', 'polite')
  announcement.setAttribute('aria-atomic', 'true')
  announcement.style.position = 'absolute'
  announcement.style.left = '-10000px'
  announcement.style.width = '1px'
  announcement.style.height = '1px'
  announcement.style.overflow = 'hidden'
  announcement.textContent = message
  
  document.body.appendChild(announcement)
  
  setTimeout(() => {
    document.body.removeChild(announcement)
  }, 1000)
}

// 键盘事件处理
const handleKeyDown = (event) => {
  // Escape 键关闭设置面板
  if (event.key === 'Escape' && showSettingsPanel.value) {
    showSettingsPanel.value = false
  }
  
  // Alt + A 开关可访问性设置
  if (event.altKey && event.key === 'a') {
    event.preventDefault()
    showSettingsPanel.value = !showSettingsPanel.value
  }
  
  // Alt + 数字键快速设置字体大小
  if (event.altKey && /^[1-4]$/.test(event.key)) {
    event.preventDefault()
    const sizes = ['small', 'normal', 'large', 'xlarge']
    setFontSize(sizes[parseInt(event.key) - 1])
  }
  
  // Alt + C 切换对比度
  if (event.altKey && event.key === 'c') {
    event.preventDefault()
    toggleContrast()
  }
}

// 组件挂载时初始化
onMounted(() => {
  // 从本地存储恢复设置
  const savedFontSize = localStorage.getItem('swit-font-size')
  if (savedFontSize && fontSizes.some(f => f.key === savedFontSize)) {
    setFontSize(savedFontSize)
  }
  
  const savedContrast = localStorage.getItem('swit-high-contrast')
  if (savedContrast === 'true') {
    highContrast.value = true
    document.documentElement.classList.add('high-contrast')
  }
  
  const savedMotion = localStorage.getItem('swit-reduce-motion')
  if (savedMotion === 'true') {
    reduceMotion.value = true
    document.documentElement.classList.add('reduce-motion')
  }
  
  // 检测系统偏好设置
  if (window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    reduceMotion.value = true
    document.documentElement.classList.add('reduce-motion')
  }
  
  if (window.matchMedia && window.matchMedia('(prefers-contrast: high)').matches) {
    highContrast.value = true
    document.documentElement.classList.add('high-contrast')
  }
  
  // 添加键盘事件监听
  document.addEventListener('keydown', handleKeyDown)
  
  // 添加页面focus管理
  document.body.addEventListener('click', (event) => {
    if (!event.target.closest('.accessibility-controls')) {
      showSettingsPanel.value = false
    }
  })
})

// 组件卸载时清理
onUnmounted(() => {
  document.removeEventListener('keydown', handleKeyDown)
})
</script>

<style scoped>
.accessibility-controls {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
  z-index: 1000;
}

.accessibility-controls > * {
  pointer-events: auto;
}

.settings-toggle {
  position: fixed;
  top: 150px;
  right: 16px;
  width: 48px;
  height: 48px;
  border: 1px solid var(--vp-c-border);
  border-radius: 50%;
  background: var(--vp-c-bg);
  color: var(--vp-c-text-1);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  box-shadow: var(--swit-shadow-md);
}

.settings-toggle:hover {
  background: var(--vp-c-brand-1);
  color: white;
  border-color: var(--vp-c-brand-1);
}

.accessibility-panel {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 90%;
  max-width: 400px;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-border);
  border-radius: var(--swit-radius-lg);
  box-shadow: var(--swit-shadow-xl);
  z-index: 1001;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  border-bottom: 1px solid var(--vp-c-border);
}

.panel-header h3 {
  margin: 0;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.close-btn {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: var(--swit-radius-sm);
  background: transparent;
  color: var(--vp-c-text-2);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.close-btn:hover {
  background: var(--vp-c-bg-soft);
  color: var(--vp-c-text-1);
}

.panel-content {
  padding: 1rem;
}

fieldset {
  border: none;
  margin: 0 0 1.5rem 0;
  padding: 0;
}

legend {
  font-weight: 600;
  margin-bottom: 0.75rem;
  color: var(--vp-c-text-1);
  font-size: 0.9rem;
}

.radio-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.radio-label,
.checkbox-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  padding: 0.25rem;
  border-radius: var(--swit-radius-sm);
  transition: background-color 0.2s ease;
}

.radio-label:hover,
.checkbox-label:hover {
  background: var(--vp-c-bg-soft);
}

.radio-input,
.checkbox-input {
  opacity: 0;
  position: absolute;
  pointer-events: none;
}

.radio-custom,
.checkbox-custom {
  width: 16px;
  height: 16px;
  border: 2px solid var(--vp-c-border);
  border-radius: 50%;
  position: relative;
  transition: all 0.2s ease;
}

.checkbox-custom {
  border-radius: var(--swit-radius-sm);
}

.radio-input:checked + .radio-custom,
.checkbox-input:checked + .checkbox-custom {
  border-color: var(--vp-c-brand-1);
  background: var(--vp-c-brand-1);
}

.radio-input:checked + .radio-custom::after {
  content: '';
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 6px;
  height: 6px;
  background: white;
  border-radius: 50%;
}

.checkbox-input:checked + .checkbox-custom::after {
  content: '✓';
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: white;
  font-size: 10px;
  font-weight: bold;
}

.radio-input:focus + .radio-custom,
.checkbox-input:focus + .checkbox-custom {
  outline: 2px solid var(--vp-c-brand-1);
  outline-offset: 2px;
}

.font-size-btn.active {
  background: var(--vp-c-brand-1);
  color: white;
  border-color: var(--vp-c-brand-1);
}

@media (max-width: 768px) {
  .settings-toggle {
    right: 8px;
    width: 44px;
    height: 44px;
  }
  
  .accessibility-panel {
    width: 95%;
    max-width: none;
  }
}
</style>