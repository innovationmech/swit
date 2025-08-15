<template>
  <div class="code-example">
    <div class="code-header" v-if="showHeader">
      <div class="code-info">
        <span v-if="title" class="code-title">{{ title }}</span>
        <span v-else class="code-language">{{ language.toUpperCase() }}</span>
      </div>
      <div class="code-actions">
        <button 
          v-if="showLineNumbers"
          @click="toggleLineNumbers"
          class="action-button"
          :title="lineNumbersTitle"
        >
          <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 10h18M3 14h18M9 6l-6 12M15 6l-6 12M21 6l-6 12"/>
          </svg>
        </button>
        <button 
          @click="copyCode" 
          class="copy-button"
          :class="{ 'copied': copied }"
        >
          <svg v-if="!copied" class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
          </svg>
          <svg v-else class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="20 6 9 17 4 12"></polyline>
          </svg>
          {{ copied ? copiedText : copyText }}
        </button>
      </div>
    </div>
    <div class="code-content" :class="{ 'with-line-numbers': showNumbers }">
      <div v-if="showNumbers" class="line-numbers">
        <span 
          v-for="(line, index) in lines" 
          :key="index"
          class="line-number"
        >
          {{ index + 1 }}
        </span>
      </div>
      <pre class="code-pre"><code 
        ref="codeElement"
        :class="`language-${language}`"
        v-html="highlightedCode"
      ></code></pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useData } from 'vitepress'

const props = withDefaults(defineProps<{
  code: string
  language?: string
  title?: string
  showHeader?: boolean
  showLineNumbers?: boolean
  highlightLines?: number[]
}>(), {
  language: 'javascript',
  showHeader: true,
  showLineNumbers: false,
  highlightLines: () => []
})

const { lang } = useData()
const copied = ref(false)
const showNumbers = ref(props.showLineNumbers)
const codeElement = ref<HTMLElement>()
const highlightedCode = ref('')

// Computed properties for i18n
const copyText = computed(() => 
  lang.value === 'zh-CN' ? '复制' : 'Copy'
)

const copiedText = computed(() => 
  lang.value === 'zh-CN' ? '已复制' : 'Copied'
)

const lineNumbersTitle = computed(() => 
  lang.value === 'zh-CN' ? '切换行号' : 'Toggle line numbers'
)

// Split code into lines for line numbers
const lines = computed(() => props.code.split('\n'))

// Copy code to clipboard
async function copyCode() {
  try {
    await navigator.clipboard.writeText(props.code)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch (err) {
    console.error('Failed to copy code:', err)
    // Fallback for older browsers
    const textarea = document.createElement('textarea')
    textarea.value = props.code
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  }
}

// Toggle line numbers
function toggleLineNumbers() {
  showNumbers.value = !showNumbers.value
}

// Simple syntax highlighting (in production, use Shiki or Prism.js)
function highlightCode(code: string, language: string): string {
  // This is a simplified version. In production, integrate with Shiki
  const escaped = code
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;')
  
  // Basic keyword highlighting for Go
  if (language === 'go') {
    return escaped
      .replace(/\b(package|import|func|var|const|type|struct|interface|return|if|else|for|range|switch|case|default|break|continue|defer|go|select|chan|map)\b/g, '<span class="keyword">$1</span>')
      .replace(/\b(string|int|int64|int32|bool|float64|float32|byte|rune|error|nil|true|false)\b/g, '<span class="type">$1</span>')
      .replace(/(\/\/.*$)/gm, '<span class="comment">$1</span>')
      .replace(/(".*?"|`.*?`)/g, '<span class="string">$1</span>')
  }
  
  // Basic keyword highlighting for JavaScript/TypeScript
  if (language === 'javascript' || language === 'typescript' || language === 'ts' || language === 'js') {
    return escaped
      .replace(/\b(const|let|var|function|class|extends|import|export|from|return|if|else|for|while|do|switch|case|default|break|continue|async|await|try|catch|finally|throw|new|this|super)\b/g, '<span class="keyword">$1</span>')
      .replace(/\b(string|number|boolean|any|void|null|undefined|true|false)\b/g, '<span class="type">$1</span>')
      .replace(/(\/\/.*$)/gm, '<span class="comment">$1</span>')
      .replace(/(".*?"|'.*?'|`.*?`)/g, '<span class="string">$1</span>')
  }
  
  return escaped
}

// Initialize highlighted code
onMounted(() => {
  highlightedCode.value = highlightCode(props.code, props.language)
})

// Watch for code changes
watch(() => props.code, (newCode) => {
  highlightedCode.value = highlightCode(newCode, props.language)
})
</script>

<style scoped>
.code-example {
  border-radius: var(--swit-border-radius);
  overflow: hidden;
  background: var(--vp-c-code-block-bg);
  border: 1px solid var(--vp-c-border);
  margin: 1.5rem 0;
}

.code-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 1rem;
  background: var(--vp-c-bg-soft);
  border-bottom: 1px solid var(--vp-c-border);
}

.code-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.code-title {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--vp-c-text-1);
}

.code-language {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--vp-c-text-3);
  text-transform: uppercase;
  letter-spacing: 0.025em;
}

.code-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.action-button,
.copy-button {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.75rem;
  font-size: 0.875rem;
  background: transparent;
  border: 1px solid var(--vp-c-border);
  border-radius: 6px;
  color: var(--vp-c-text-2);
  cursor: pointer;
  transition: var(--swit-transition);
}

.action-button:hover,
.copy-button:hover {
  background: var(--vp-c-bg);
  border-color: var(--vp-c-brand-1);
  color: var(--vp-c-brand-1);
}

.copy-button.copied {
  background: var(--vp-c-success-soft);
  border-color: var(--vp-c-success-1);
  color: var(--vp-c-success-1);
}

.icon {
  width: 16px;
  height: 16px;
}

.code-content {
  position: relative;
  display: flex;
  overflow-x: auto;
}

.code-content.with-line-numbers {
  padding-left: 0;
}

.line-numbers {
  display: flex;
  flex-direction: column;
  padding: 1rem 0;
  background: var(--vp-c-code-line-number-bg, var(--vp-c-bg-soft));
  border-right: 1px solid var(--vp-c-border);
  user-select: none;
}

.line-number {
  padding: 0 1rem;
  font-family: var(--vp-font-family-mono);
  font-size: 0.875rem;
  line-height: 1.7;
  color: var(--vp-c-text-3);
  text-align: right;
  min-width: 3ch;
}

.code-pre {
  flex: 1;
  margin: 0;
  padding: 1rem;
  overflow-x: auto;
  font-family: var(--vp-font-family-mono);
  font-size: 0.875rem;
  line-height: 1.7;
}

.code-content:not(.with-line-numbers) .code-pre {
  padding: 1rem 1.5rem;
}

.code-pre code {
  display: block;
  color: var(--vp-c-text-1);
}

/* Syntax highlighting styles */
:deep(.keyword) {
  color: var(--vp-c-brand-1);
  font-weight: 500;
}

:deep(.type) {
  color: var(--vp-c-warning-1);
}

:deep(.string) {
  color: var(--vp-c-success-1);
}

:deep(.comment) {
  color: var(--vp-c-text-3);
  font-style: italic;
}

:deep(.function) {
  color: var(--vp-c-tip-1);
}

:deep(.number) {
  color: var(--vp-c-danger-1);
}

/* Dark mode adjustments */
.dark .code-example {
  background: var(--vp-c-code-block-bg);
}

.dark :deep(.keyword) {
  color: #79b8ff;
}

.dark :deep(.type) {
  color: #ffab70;
}

.dark :deep(.string) {
  color: #79e676;
}

.dark :deep(.comment) {
  color: #6a737d;
}

/* Scrollbar styling */
.code-pre::-webkit-scrollbar {
  height: 6px;
}

.code-pre::-webkit-scrollbar-track {
  background: transparent;
}

.code-pre::-webkit-scrollbar-thumb {
  background: var(--vp-c-border);
  border-radius: 3px;
}

.code-pre::-webkit-scrollbar-thumb:hover {
  background: var(--vp-c-text-3);
}

/* Mobile responsive */
@media (max-width: 640px) {
  .code-header {
    padding: 0.5rem 0.75rem;
  }
  
  .code-title {
    font-size: 0.8125rem;
  }
  
  .action-button,
  .copy-button {
    padding: 0.25rem 0.5rem;
    font-size: 0.8125rem;
  }
  
  .code-pre {
    padding: 0.75rem;
    font-size: 0.8125rem;
  }
}
</style>