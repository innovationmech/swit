<template>
  <div class="accessibility-controls">
    <!-- 跳转到主内容链接 -->
    <a href="#main-content" class="skip-to-content" @click="skipToMain">
      跳转到主内容
    </a>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'

// 方法
function skipToMain() {
  const main = document.querySelector('#main-content')
  if (main) {
    main.focus()
    main.scrollIntoView({ behavior: 'smooth' })
  }
}

// 初始化 - 保持基础的可访问性功能
onMounted(() => {
  // SSR protection
  if (typeof window === 'undefined') return
  
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
})
</script>

<style scoped>
.accessibility-controls {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 1000;
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
</style>