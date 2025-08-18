<template>
  <div class="vp-layout">
    <!-- 精简的可访问性控制组件 -->
    <AccessibilityControls />
    
    <!-- 使用默认主题布局，但添加可访问性增强 -->
    <DefaultLayout>
      <!-- 添加语义化 HTML 结构 -->
      <template #layout-top>
        <div id="main-content" tabindex="-1" style="position: absolute; top: -1px; left: -1px; width: 1px; height: 1px; opacity: 0;"></div>
      </template>
    </DefaultLayout>
    
    <!-- 精简化，移除不必要的 ARIA 组件 -->
  </div>
</template>

<script setup>
import { ref, onMounted, provide } from 'vue'
import DefaultTheme from 'vitepress/theme'
import AccessibilityControls from './components/AccessibilityControls.vue'

const DefaultLayout = DefaultTheme.Layout

// 精简的可访问性支持

// 简化的键盘导航
const setupKeyboardNavigation = () => {
  // 只保留基本的键盘导航支持
  document.addEventListener('keydown', (event) => {
    // Esc 键关闭模态框
    if (event.key === 'Escape') {
      const modals = document.querySelectorAll('[role="dialog"]')
      modals.forEach(modal => modal.style.display = 'none')
    }
  })
}

// 删除了复杂的可访问性功能

// 精简的初始化
onMounted(() => {
  // 基本的可访问性设置
  setupKeyboardNavigation()
})

// 已精简化，不需要暴露全局函数
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
</style>