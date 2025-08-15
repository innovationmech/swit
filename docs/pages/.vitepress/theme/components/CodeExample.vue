<template>
  <div class="code-example">
    <div class="code-header">
      <div class="code-info">
        <h3 class="code-title">{{ title }}</h3>
        <p class="code-description">{{ description }}</p>
      </div>
      <div class="code-actions">
        <button 
          @click="copyCode" 
          :class="['copy-btn', { 'copied': isCopied }]"
          :title="isCopied ? '已复制!' : '复制代码'"
        >
          <span v-if="!isCopied">📋</span>
          <span v-else>✅</span>
        </button>
        <button 
          @click="toggleExpanded" 
          class="expand-btn"
          :title="isExpanded ? '收起' : '展开'"
        >
          <span :class="{ 'rotated': isExpanded }">🔽</span>
        </button>
      </div>
    </div>
    
    <div :class="['code-container', { 'expanded': isExpanded }]">
      <div class="code-lang-tag">{{ language }}</div>
      <pre class="code-content"><code :class="`language-${language}`" v-html="highlightedCode"></code></pre>
    </div>
    
    <div v-if="!isExpanded && codeLines.length > maxCollapsedLines" class="code-fade">
      <div class="fade-overlay"></div>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted } from 'vue'

export default {
  name: 'CodeExample',
  props: {
    title: {
      type: String,
      required: true
    },
    description: {
      type: String,
      required: true
    },
    code: {
      type: String,
      required: true
    },
    language: {
      type: String,
      default: 'javascript'
    },
    maxLines: {
      type: Number,
      default: 20
    }
  },
  setup(props) {
    const isExpanded = ref(false)
    const isCopied = ref(false)
    const maxCollapsedLines = 15

    const codeLines = computed(() => {
      return props.code.split('\n')
    })

    const displayCode = computed(() => {
      if (isExpanded.value || codeLines.value.length <= maxCollapsedLines) {
        return props.code
      }
      return codeLines.value.slice(0, maxCollapsedLines).join('\n')
    })

    // 简单的语法高亮（可以后续集成 Prism.js 或其他库）
    const highlightedCode = computed(() => {
      let code = displayCode.value
      
      if (props.language === 'go') {
        // Go 语法高亮
        code = code
          .replace(/(package|import|func|type|struct|interface|var|const|if|else|for|range|return|defer|go|chan|select|case|default|switch|break|continue)\b/g, '<span class="keyword">$1</span>')
          .replace(/(string|int|int64|float64|bool|error|context\.Context|interface\{\})/g, '<span class="type">$1</span>')
          .replace(/(".*?")/g, '<span class="string">$1</span>')
          .replace(/(\/\/.*$)/gm, '<span class="comment">$1</span>')
          .replace(/(\d+)/g, '<span class="number">$1</span>')
      } else if (props.language === 'javascript' || props.language === 'js') {
        // JavaScript 语法高亮
        code = code
          .replace(/(function|const|let|var|if|else|for|while|return|import|export|from|class|extends|async|await|try|catch|finally)\b/g, '<span class="keyword">$1</span>')
          .replace(/('.*?'|".*?")/g, '<span class="string">$1</span>')
          .replace(/(\/\/.*$)/gm, '<span class="comment">$1</span>')
          .replace(/(\d+)/g, '<span class="number">$1</span>')
      }
      
      return code
    })

    const copyCode = async () => {
      try {
        await navigator.clipboard.writeText(props.code)
        isCopied.value = true
        setTimeout(() => {
          isCopied.value = false
        }, 2000)
      } catch (err) {
        console.error('Failed to copy code:', err)
      }
    }

    const toggleExpanded = () => {
      isExpanded.value = !isExpanded.value
    }

    return {
      isExpanded,
      isCopied,
      codeLines,
      displayCode,
      highlightedCode,
      maxCollapsedLines,
      copyCode,
      toggleExpanded
    }
  }
}
</script>

<style scoped>
.code-example {
  border: 1px solid var(--vp-c-border);
  border-radius: var(--swit-radius-lg);
  background: var(--vp-c-bg-soft);
  overflow: hidden;
  transition: all 0.3s ease;
  position: relative;
}

.code-example:hover {
  border-color: var(--vp-c-brand-1);
  box-shadow: var(--swit-shadow-md);
}

.code-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 1.5rem;
  border-bottom: 1px solid var(--vp-c-border);
  background: var(--vp-c-bg);
}

.code-info {
  flex: 1;
}

.code-title {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--vp-c-text-1);
  margin: 0 0 0.5rem 0;
}

.code-description {
  color: var(--vp-c-text-2);
  margin: 0;
  line-height: 1.5;
}

.code-actions {
  display: flex;
  gap: 0.5rem;
  margin-left: 1rem;
}

.copy-btn,
.expand-btn {
  background: transparent;
  border: 1px solid var(--vp-c-border);
  border-radius: var(--swit-radius-sm);
  padding: 0.5rem;
  cursor: pointer;
  transition: all 0.3s ease;
  font-size: 1rem;
  min-width: 2.5rem;
  height: 2.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
}

.copy-btn:hover,
.expand-btn:hover {
  background: var(--vp-c-bg-soft);
  border-color: var(--vp-c-brand-1);
}

.copy-btn.copied {
  background: rgba(34, 197, 94, 0.1);
  border-color: #22c55e;
  color: #22c55e;
}

.expand-btn .rotated {
  transform: rotate(180deg);
}

.code-container {
  position: relative;
  background: var(--vp-c-bg-alt);
  max-height: 400px;
  overflow: hidden;
  transition: max-height 0.3s ease;
}

.code-container.expanded {
  max-height: none;
}

.code-lang-tag {
  position: absolute;
  top: 0.5rem;
  right: 1rem;
  background: var(--vp-c-brand-1);
  color: white;
  padding: 0.25rem 0.5rem;
  border-radius: var(--swit-radius-sm);
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  z-index: 1;
}

.code-content {
  margin: 0;
  padding: 1.5rem;
  padding-top: 3rem;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.875rem;
  line-height: 1.6;
  color: var(--vp-c-text-1);
  overflow-x: auto;
}

.code-fade {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 3rem;
  pointer-events: none;
}

.fade-overlay {
  height: 100%;
  background: linear-gradient(transparent, var(--vp-c-bg-soft));
}

/* 语法高亮样式 */
.code-content :deep(.keyword) {
  color: #e73c7e;
  font-weight: 600;
}

.code-content :deep(.type) {
  color: #42b883;
  font-weight: 500;
}

.code-content :deep(.string) {
  color: #42b883;
}

.code-content :deep(.comment) {
  color: #999;
  font-style: italic;
}

.code-content :deep(.number) {
  color: #e73c7e;
}

/* Dark mode adaptations */
.dark .code-example {
  border-color: var(--vp-c-border-dark);
  background: var(--vp-c-bg-alt);
}

.dark .code-header {
  background: var(--vp-c-bg-soft);
  border-color: var(--vp-c-border-dark);
}

.dark .copy-btn,
.dark .expand-btn {
  border-color: var(--vp-c-border-dark);
}

.dark .copy-btn:hover,
.dark .expand-btn:hover {
  background: var(--vp-c-bg-alt);
}

.dark .code-container {
  background: var(--vp-c-bg);
}

.dark .fade-overlay {
  background: linear-gradient(transparent, var(--vp-c-bg-alt));
}

.dark .code-content :deep(.keyword) {
  color: #ff7875;
}

.dark .code-content :deep(.type) {
  color: #73d13d;
}

.dark .code-content :deep(.string) {
  color: #73d13d;
}

.dark .code-content :deep(.comment) {
  color: #8c8c8c;
}

.dark .code-content :deep(.number) {
  color: #ff7875;
}

/* Responsive design */
@media (max-width: 768px) {
  .code-header {
    padding: 1rem;
  }
  
  .code-content {
    padding: 1rem;
    padding-top: 2.5rem;
    font-size: 0.8rem;
  }
  
  .code-lang-tag {
    font-size: 0.7rem;
    padding: 0.2rem 0.4rem;
  }
  
  .code-actions {
    margin-left: 0.5rem;
  }
  
  .copy-btn,
  .expand-btn {
    min-width: 2rem;
    height: 2rem;
    font-size: 0.9rem;
  }
}

/* Animation */
.code-example {
  opacity: 0;
  transform: translateY(20px);
  animation: fadeInUp 0.6s ease-out forwards;
}

.code-example:nth-child(2) {
  animation-delay: 0.2s;
}

@keyframes fadeInUp {
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>