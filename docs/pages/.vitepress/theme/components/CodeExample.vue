<template>
  <div class="code-example">
    <div v-if="title || copy" class="code-header">
      <span v-if="title" class="code-title">{{ title }}</span>
      <span class="code-language">{{ language }}</span>
      <button 
        v-if="copy" 
        @click="copyCode" 
        class="copy-button"
        :class="{ copied: isCopied }"
      >
        <svg v-if="!isCopied" width="16" height="16" fill="currentColor" viewBox="0 0 16 16">
          <path d="M4 1.5H3a2 2 0 0 0-2 2V14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V3.5a2 2 0 0 0-2-2h-1v1h1a1 1 0 0 1 1 1V14a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V3.5a1 1 0 0 1 1-1h1v-1z"/>
          <path d="M9.5 1a.5.5 0 0 1 .5.5v1a.5.5 0 0 1-.5.5h-3a.5.5 0 0 1-.5-.5v-1a.5.5 0 0 1 .5-.5h3zm-3-1A1.5 1.5 0 0 0 5 1.5v1A1.5 1.5 0 0 0 6.5 4h3A1.5 1.5 0 0 0 11 2.5v-1A1.5 1.5 0 0 0 9.5 0h-3z"/>
        </svg>
        <svg v-else width="16" height="16" fill="currentColor" viewBox="0 0 16 16">
          <path d="M13.854 3.646a.5.5 0 0 1 0 .708l-7 7a.5.5 0 0 1-.708 0l-3.5-3.5a.5.5 0 1 1 .708-.708L6.5 10.293l6.646-6.647a.5.5 0 0 1 .708 0z"/>
        </svg>
        {{ isCopied ? 'Copied!' : 'Copy' }}
      </button>
    </div>
    
    <div class="code-content">
      <pre><code>{{ code }}</code></pre>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const props = defineProps({
  code: {
    type: String,
    required: true
  },
  language: {
    type: String,
    default: 'text'
  },
  title: {
    type: String,
    default: ''
  },
  copy: {
    type: Boolean,
    default: false
  }
})

const isCopied = ref(false)

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
</script>

<style scoped>
.code-example {
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  overflow: hidden;
  margin: 1rem 0;
}

.code-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  background: var(--vp-c-bg-soft);
  border-bottom: 1px solid var(--vp-c-divider);
  font-size: 0.875rem;
}

.code-title {
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.code-language {
  color: var(--vp-c-text-2);
  font-family: var(--vp-font-family-mono);
}

.copy-button {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.375rem 0.75rem;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-divider);
  border-radius: 4px;
  color: var(--vp-c-text-2);
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.2s;
}

.copy-button:hover {
  background: var(--vp-c-bg-alt);
  color: var(--vp-c-text-1);
}

.copy-button.copied {
  background: var(--vp-c-brand-soft);
  color: var(--vp-c-brand-1);
  border-color: var(--vp-c-brand-1);
}

.code-content {
  overflow-x: auto;
}

.code-content pre {
  margin: 0;
  padding: 1rem;
  background: var(--vp-c-bg);
  font-family: var(--vp-font-family-mono);
  font-size: 0.875rem;
  line-height: 1.5;
  color: var(--vp-c-text-1);
}

.code-content code {
  background: none;
  padding: 0;
  font-size: inherit;
  color: inherit;
}
</style>