<template>
  <div class="accessibility-controls">
    <!-- 跳转到主内容链接 -->
    <a href="#main-content" class="skip-to-content" @click="skipToMain">
      跳转到主内容
    </a>
    
    <!-- 字体大小控制 -->
    <div class="font-size-controls" role="region" aria-label="字体大小控制">
      <button
        v-for="size in fontSizes"
        :key="size.key"
        class="font-size-btn"
        :class="{ active: currentFontSize === size.key }"
        :aria-label="`设置字体大小为 ${size.label}`"
        :aria-pressed="currentFontSize === size.key"
        @click="setFontSize(size.key)"
      >
        {{ size.display }}
      </button>
    </div>
    
    <!-- 高对比度切换 -->
    <button
      class="contrast-toggle"
      aria-label="切换高对比度"
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
    <div v-if="showSettingsPanel" class="accessibility-panel" role="dialog" aria-label="辅助功能设置面板">
      <div class="panel-header">
        <h3>辅助功能设置</h3>
        <button class="close-btn" @click="showSettingsPanel = false" aria-label="关闭面板">
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M12 4L4 12M4 4l8 8" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          </svg>
        </button>
      </div>
      
      <div class="panel-content">
        <!-- 字体大小设置 -->
        <fieldset>
          <legend>字体大小</legend>
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
          <legend>对比度</legend>
          <label class="checkbox-label">
            <input
              type="checkbox"
              v-model="highContrast"
              @change="toggleContrast"
              class="checkbox-input"
            />
            <span class="checkbox-custom"></span>
            高对比度
          </label>
        </fieldset>
        
        <!-- 动画设置 -->
        <fieldset>
          <legend>动画</legend>
          <label class="checkbox-label">
            <input
              type="checkbox"
              v-model="reduceMotion"
              @change="toggleMotion"
              class="checkbox-input"
            />
            <span class="checkbox-custom"></span>
            减少动画
          </label>
        </fieldset>
      </div>
    </div>
    
    <!-- 设置按钮 -->
    <button
      class="settings-toggle"
      aria-label="打开设置"
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
import { ref, onMounted, onUnmounted } from 'vue'

// 响应式状态
const currentFontSize = ref('normal')
const highContrast = ref(false)
const reduceMotion = ref(false)
const showSettingsPanel = ref(false)

// 字体大小选项
const fontSizes = [
  { key: 'small', label: '小', display: 'A' },
  { key: 'normal', label: '正常', display: 'A' },
  { key: 'large', label: '大', display: 'A' },
  { key: 'xlarge', label: '特大', display: 'A' }
]

// 方法
function skipToMain() {
  const main = document.querySelector('#main-content')
  if (main) {
    main.focus()
    main.scrollIntoView({ behavior: 'smooth' })
  }
}

function setFontSize(size) {
  currentFontSize.value = size
  document.documentElement.className = document.documentElement.className
    .replace(/font-size-\w+/g, '')
  document.documentElement.classList.add(`font-size-${size}`)
  
  // 保存到本地存储
  localStorage.setItem('accessibility-font-size', size)
}

function toggleContrast() {
  highContrast.value = !highContrast.value
  document.documentElement.classList.toggle('high-contrast', highContrast.value)
  
  // 保存到本地存储
  localStorage.setItem('accessibility-high-contrast', highContrast.value.toString())
}

function toggleMotion() {
  reduceMotion.value = !reduceMotion.value
  document.documentElement.classList.toggle('reduce-motion', reduceMotion.value)
  
  // 保存到本地存储
  localStorage.setItem('accessibility-reduce-motion', reduceMotion.value.toString())
}

// 初始化
onMounted(() => {
  // SSR protection
  if (typeof window === 'undefined') return
  
  // 从本地存储恢复设置
  const savedFontSize = localStorage.getItem('accessibility-font-size')
  if (savedFontSize) {
    setFontSize(savedFontSize)
  }
  
  const savedContrast = localStorage.getItem('accessibility-high-contrast')
  if (savedContrast === 'true') {
    highContrast.value = true
    document.documentElement.classList.add('high-contrast')
  }
  
  const savedMotion = localStorage.getItem('accessibility-reduce-motion')
  if (savedMotion === 'true') {
    reduceMotion.value = true
    document.documentElement.classList.add('reduce-motion')
  }
  
  // 检测系统偏好
  if (window.matchMedia) {
    const contrastQuery = window.matchMedia('(prefers-contrast: high)')
    const motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
    
    if (contrastQuery.matches && !localStorage.getItem('accessibility-high-contrast')) {
      toggleContrast()
    }
    
    if (motionQuery.matches && !localStorage.getItem('accessibility-reduce-motion')) {
      toggleMotion()
    }
  }
})
</script>

<style scoped>
.accessibility-controls {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 1000;
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-items: flex-end;
}

.skip-to-content {
  position: absolute;
  top: -40px;
  left: 6px;
  background: var(--vp-c-brand);
  color: white;
  padding: 8px 16px;
  text-decoration: none;
  border-radius: 4px;
  font-size: 14px;
  opacity: 0;
  pointer-events: none;
  transition: all 0.3s ease;
}

.skip-to-content:focus {
  top: 0;
  opacity: 1;
  pointer-events: auto;
}

.font-size-controls {
  display: flex;
  gap: 4px;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-divider);
  border-radius: 6px;
  padding: 4px;
}

.font-size-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: bold;
  transition: all 0.2s ease;
}

.font-size-btn:hover {
  background: var(--vp-c-gray-soft);
}

.font-size-btn.active {
  background: var(--vp-c-brand);
  color: white;
}

.contrast-toggle {
  width: 48px;
  height: 48px;
  border: none;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-divider);
  border-radius: 50%;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.contrast-toggle:hover {
  background: var(--vp-c-gray-soft);
}

.settings-toggle {
  width: 48px;
  height: 48px;
  border: none;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-divider);
  border-radius: 50%;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.settings-toggle:hover {
  background: var(--vp-c-gray-soft);
}

.accessibility-panel {
  position: absolute;
  top: 0;
  right: 60px;
  width: 300px;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: var(--vp-c-bg-soft);
  border-bottom: 1px solid var(--vp-c-divider);
}

.panel-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.close-btn {
  background: none;
  border: none;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  color: var(--vp-c-text-2);
}

.close-btn:hover {
  background: var(--vp-c-gray-soft);
  color: var(--vp-c-text-1);
}

.panel-content {
  padding: 20px;
}

fieldset {
  border: none;
  margin: 0 0 20px 0;
  padding: 0;
}

legend {
  font-weight: 600;
  margin-bottom: 12px;
  color: var(--vp-c-text-1);
}

.radio-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.radio-label,
.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  font-size: 14px;
  color: var(--vp-c-text-2);
}

.radio-input,
.checkbox-input {
  margin: 0;
}

/* 高对比度样式 */
.high-contrast {
  filter: contrast(150%);
}

/* 减少动画样式 */
.reduce-motion * {
  animation-duration: 0.01ms !important;
  animation-iteration-count: 1 !important;
  transition-duration: 0.01ms !important;
}

/* 字体大小样式 */
.font-size-small {
  font-size: 14px;
}

.font-size-normal {
  font-size: 16px;
}

.font-size-large {
  font-size: 18px;
}

.font-size-xlarge {
  font-size: 20px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .accessibility-controls {
    top: 10px;
    right: 10px;
  }
  
  .accessibility-panel {
    right: 0;
    width: calc(100vw - 20px);
    max-width: 300px;
  }
}
</style>