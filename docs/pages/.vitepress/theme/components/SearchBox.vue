<template>
  <div class="search-box" ref="searchContainer">
    <!-- Search Input -->
    <div class="search-input-container">
      <div class="search-input-wrapper">
        <span class="search-icon">🔍</span>
        <input
          ref="searchInput"
          v-model="searchQuery"
          @input="handleSearch"
          @focus="showResults = true"
          @keydown="handleKeydown"
          type="text"
          placeholder="搜索文档、API、示例..."
          class="search-input"
        />
        <div v-if="searchQuery" class="search-actions">
          <span class="result-count">{{ totalResults }} 个结果</span>
          <button @click="clearSearch" class="clear-btn" title="清除搜索">✕</button>
        </div>
      </div>
    </div>

    <!-- Search Results -->
    <div v-if="showResults && searchQuery" class="search-results">
      <div class="search-results-header">
        <span class="results-title">搜索结果</span>
        <div class="search-filters">
          <button
            v-for="filter in filters"
            :key="filter.key"
            :class="['filter-btn', { active: activeFilter === filter.key }]"
            @click="activeFilter = filter.key"
          >
            {{ filter.label }}
          </button>
        </div>
      </div>

      <!-- Results Content -->
      <div class="results-container">
        <!-- No Results -->
        <div v-if="filteredResults.length === 0" class="no-results">
          <div class="no-results-icon">📄</div>
          <h3>没有找到相关内容</h3>
          <p>尝试使用不同的关键词或检查拼写</p>
          <div class="search-suggestions">
            <span>建议搜索：</span>
            <button
              v-for="suggestion in suggestions"
              :key="suggestion"
              @click="applySuggestion(suggestion)"
              class="suggestion-btn"
            >
              {{ suggestion }}
            </button>
          </div>
        </div>

        <!-- Results List -->
        <div v-else class="results-list">
          <div
            v-for="(result, index) in filteredResults"
            :key="result.id"
            :class="['result-item', { highlighted: selectedIndex === index }]"
            @click="navigateToResult(result)"
            @mouseenter="selectedIndex = index"
          >
            <div class="result-header">
              <div class="result-info">
                <span :class="['result-type', result.type]">{{ getTypeLabel(result.type) }}</span>
                <h4 class="result-title">{{ highlightMatch(result.title) }}</h4>
              </div>
              <span class="result-url">{{ result.url }}</span>
            </div>
            <div class="result-content">
              <p class="result-excerpt">{{ highlightMatch(result.excerpt) }}</p>
              <div v-if="result.breadcrumb" class="result-breadcrumb">
                <span v-for="(crumb, i) in result.breadcrumb" :key="i">
                  {{ crumb }}<span v-if="i < result.breadcrumb.length - 1"> › </span>
                </span>
              </div>
            </div>
            <div v-if="result.tags" class="result-tags">
              <span v-for="tag in result.tags" :key="tag" class="result-tag">{{ tag }}</span>
            </div>
          </div>
        </div>

        <!-- Load More -->
        <div v-if="hasMoreResults" class="load-more">
          <button @click="loadMoreResults" class="load-more-btn">
            加载更多结果
          </button>
        </div>
      </div>

      <!-- Search Tips -->
      <div class="search-tips">
        <details>
          <summary>搜索提示</summary>
          <div class="tips-content">
            <ul>
              <li><code>"完整短语"</code> - 搜索完整短语</li>
              <li><code>API server</code> - 搜索包含所有关键词的内容</li>
              <li><code>type:api</code> - 按内容类型筛选</li>
              <li><code>tag:http</code> - 按标签筛选</li>
              <li>支持中英文混合搜索</li>
            </ul>
          </div>
        </details>
      </div>
    </div>

    <!-- Search Overlay -->
    <div
      v-if="showResults && searchQuery"
      class="search-overlay"
      @click="showResults = false"
    ></div>
  </div>
</template>

<script>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'

export default {
  name: 'SearchBox',
  setup() {
    const searchQuery = ref('')
    const showResults = ref(false)
    const selectedIndex = ref(-1)
    const activeFilter = ref('all')
    const searchContainer = ref(null)
    const searchInput = ref(null)
    const currentPage = ref(1)
    const pageSize = 10

    // Search filters
    const filters = ref([
      { key: 'all', label: '全部' },
      { key: 'docs', label: '文档' },
      { key: 'api', label: 'API' },
      { key: 'examples', label: '示例' },
      { key: 'guides', label: '指南' }
    ])

    // Mock search data - in real implementation, this would come from an index
    const searchData = ref([
      {
        id: '1',
        title: 'BusinessServerCore 接口',
        excerpt: 'BusinessServerCore 是 Swit 框架的核心服务器接口，提供完整的生命周期管理功能...',
        url: '/zh/api/server/',
        type: 'api',
        breadcrumb: ['API 文档', '核心接口', 'BusinessServerCore'],
        tags: ['server', 'lifecycle', 'interface']
      },
      {
        id: '2',
        title: '快速开始指南',
        excerpt: '学习如何使用 Swit 框架创建你的第一个微服务应用，包括环境设置和基本配置...',
        url: '/zh/guide/getting-started',
        type: 'guide',
        breadcrumb: ['指南', '快速开始'],
        tags: ['getting-started', 'tutorial']
      },
      {
        id: '3',
        title: 'HTTP 服务示例',
        excerpt: '完整的 HTTP 服务实现示例，展示如何使用 Swit 框架构建 RESTful API...',
        url: '/zh/examples/http-service',
        type: 'example',
        breadcrumb: ['示例', 'HTTP 服务'],
        tags: ['http', 'rest', 'api']
      },
      {
        id: '4',
        title: 'gRPC 传输层配置',
        excerpt: 'gRPC 传输层的详细配置选项，包括端口设置、keepalive 参数和反射服务...',
        url: '/zh/guide/grpc-transport',
        type: 'docs',
        breadcrumb: ['文档', '传输层', 'gRPC'],
        tags: ['grpc', 'transport', 'config']
      },
      {
        id: '5',
        title: '用户管理 API',
        excerpt: '用户管理相关的 API 端点，包括创建、查询、更新和删除用户信息...',
        url: '/zh/api/users',
        type: 'api',
        breadcrumb: ['API 文档', '用户管理'],
        tags: ['users', 'crud', 'management']
      },
      {
        id: '6',
        title: '依赖注入系统',
        excerpt: 'Swit 框架的依赖注入系统说明，包括工厂模式、单例和瞬态依赖的注册...',
        url: '/zh/guide/dependency-injection',
        type: 'docs',
        breadcrumb: ['文档', '核心概念', '依赖注入'],
        tags: ['di', 'dependency', 'injection', 'factory']
      },
      {
        id: '7',
        title: '认证中间件示例',
        excerpt: 'JWT 认证中间件的实现示例，展示如何在 Swit 框架中集成认证功能...',
        url: '/zh/examples/auth-middleware',
        type: 'example',
        breadcrumb: ['示例', '中间件', '认证'],
        tags: ['auth', 'jwt', 'middleware']
      }
    ])

    const suggestions = ref(['server', 'api', 'grpc', 'http', '认证', '配置'])

    // Search logic
    const searchResults = ref([])
    const isLoading = ref(false)

    const performSearch = (query) => {
      if (!query.trim()) {
        searchResults.value = []
        return
      }

      const queryLower = query.toLowerCase()
      const results = searchData.value.filter(item => {
        return (
          item.title.toLowerCase().includes(queryLower) ||
          item.excerpt.toLowerCase().includes(queryLower) ||
          item.tags.some(tag => tag.toLowerCase().includes(queryLower)) ||
          item.breadcrumb.some(crumb => crumb.toLowerCase().includes(queryLower))
        )
      })

      // Simple relevance scoring
      results.forEach(result => {
        let score = 0
        if (result.title.toLowerCase().includes(queryLower)) score += 10
        if (result.excerpt.toLowerCase().includes(queryLower)) score += 5
        result.tags.forEach(tag => {
          if (tag.toLowerCase().includes(queryLower)) score += 3
        })
        result.score = score
      })

      searchResults.value = results.sort((a, b) => b.score - a.score)
    }

    const filteredResults = computed(() => {
      let results = searchResults.value
      if (activeFilter.value !== 'all') {
        results = results.filter(result => result.type === activeFilter.value)
      }
      return results.slice(0, currentPage.value * pageSize)
    })

    const totalResults = computed(() => {
      if (activeFilter.value === 'all') {
        return searchResults.value.length
      }
      return searchResults.value.filter(result => result.type === activeFilter.value).length
    })

    const hasMoreResults = computed(() => {
      const total = activeFilter.value === 'all' 
        ? searchResults.value.length 
        : searchResults.value.filter(result => result.type === activeFilter.value).length
      return currentPage.value * pageSize < total
    })

    // Event handlers
    const handleSearch = () => {
      performSearch(searchQuery.value)
      selectedIndex.value = -1
      currentPage.value = 1
    }

    const handleKeydown = (event) => {
      if (!showResults.value) return

      switch (event.key) {
        case 'ArrowDown':
          event.preventDefault()
          selectedIndex.value = Math.min(selectedIndex.value + 1, filteredResults.value.length - 1)
          break
        case 'ArrowUp':
          event.preventDefault()
          selectedIndex.value = Math.max(selectedIndex.value - 1, -1)
          break
        case 'Enter':
          event.preventDefault()
          if (selectedIndex.value >= 0) {
            navigateToResult(filteredResults.value[selectedIndex.value])
          }
          break
        case 'Escape':
          showResults.value = false
          searchInput.value.blur()
          break
      }
    }

    const navigateToResult = (result) => {
      // In a real implementation, use router navigation
      console.log('Navigating to:', result.url)
      showResults.value = false
      // window.location.href = result.url
    }

    const clearSearch = () => {
      searchQuery.value = ''
      searchResults.value = []
      showResults.value = false
      selectedIndex.value = -1
    }

    const applySuggestion = (suggestion) => {
      searchQuery.value = suggestion
      handleSearch()
      nextTick(() => {
        searchInput.value.focus()
      })
    }

    const loadMoreResults = () => {
      currentPage.value += 1
    }

    const getTypeLabel = (type) => {
      const typeMap = {
        'docs': '文档',
        'api': 'API',
        'example': '示例',
        'guide': '指南'
      }
      return typeMap[type] || type
    }

    const highlightMatch = (text) => {
      if (!searchQuery.value) return text
      
      const query = searchQuery.value.trim()
      const regex = new RegExp(`(${query})`, 'gi')
      return text.replace(regex, '<mark>$1</mark>')
    }

    // Click outside handler
    const handleClickOutside = (event) => {
      if (searchContainer.value && !searchContainer.value.contains(event.target)) {
        showResults.value = false
      }
    }

    // Keyboard shortcut handler
    const handleKeyboardShortcut = (event) => {
      // Ctrl/Cmd + K to focus search
      if ((event.ctrlKey || event.metaKey) && event.key === 'k') {
        event.preventDefault()
        searchInput.value.focus()
        showResults.value = true
      }
    }

    // Lifecycle
    onMounted(() => {
      document.addEventListener('click', handleClickOutside)
      document.addEventListener('keydown', handleKeyboardShortcut)
    })

    onUnmounted(() => {
      document.removeEventListener('click', handleClickOutside)
      document.removeEventListener('keydown', handleKeyboardShortcut)
    })

    // Watch for filter changes
    watch(activeFilter, () => {
      currentPage.value = 1
      selectedIndex.value = -1
    })

    return {
      searchQuery,
      showResults,
      selectedIndex,
      activeFilter,
      searchContainer,
      searchInput,
      filters,
      searchResults,
      filteredResults,
      totalResults,
      hasMoreResults,
      suggestions,
      isLoading,
      handleSearch,
      handleKeydown,
      navigateToResult,
      clearSearch,
      applySuggestion,
      loadMoreResults,
      getTypeLabel,
      highlightMatch
    }
  }
}
</script>

<style scoped>
.search-box {
  position: relative;
  width: 100%;
  max-width: 600px;
}

/* Search Input */
.search-input-container {
  position: relative;
  z-index: 100;
}

.search-input-wrapper {
  display: flex;
  align-items: center;
  position: relative;
  background: var(--vp-c-bg);
  border: 2px solid var(--vp-c-border);
  border-radius: var(--swit-radius-lg);
  transition: all 0.3s ease;
}

.search-input-wrapper:focus-within {
  border-color: var(--vp-c-brand-1);
  box-shadow: 0 0 0 3px var(--vp-c-brand-soft);
}

.search-icon {
  padding: 0 0.75rem;
  color: var(--vp-c-text-2);
  font-size: 1.25rem;
}

.search-input {
  flex: 1;
  padding: 0.75rem 0;
  border: none;
  background: transparent;
  color: var(--vp-c-text-1);
  font-size: 1rem;
  outline: none;
}

.search-input::placeholder {
  color: var(--vp-c-text-3);
}

.search-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding-right: 0.75rem;
}

.result-count {
  font-size: 0.875rem;
  color: var(--vp-c-text-2);
}

.clear-btn {
  width: 1.5rem;
  height: 1.5rem;
  border: none;
  background: var(--vp-c-bg-soft);
  color: var(--vp-c-text-2);
  border-radius: 50%;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.75rem;
  transition: all 0.3s ease;
}

.clear-btn:hover {
  background: var(--vp-c-bg-alt);
  color: var(--vp-c-text-1);
}

/* Search Results */
.search-results {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  margin-top: 0.5rem;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-border);
  border-radius: var(--swit-radius-lg);
  box-shadow: var(--swit-shadow-lg);
  max-height: 70vh;
  overflow: hidden;
  z-index: 99;
}

.search-results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  border-bottom: 1px solid var(--vp-c-border);
  background: var(--vp-c-bg-soft);
}

.results-title {
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.search-filters {
  display: flex;
  gap: 0.25rem;
}

.filter-btn {
  padding: 0.25rem 0.5rem;
  border: 1px solid var(--vp-c-border);
  border-radius: var(--swit-radius-sm);
  background: var(--vp-c-bg);
  color: var(--vp-c-text-2);
  font-size: 0.75rem;
  cursor: pointer;
  transition: all 0.3s ease;
}

.filter-btn:hover,
.filter-btn.active {
  background: var(--vp-c-brand-1);
  color: white;
  border-color: var(--vp-c-brand-1);
}

.results-container {
  max-height: 50vh;
  overflow-y: auto;
}

/* No Results */
.no-results {
  text-align: center;
  padding: 3rem 2rem;
  color: var(--vp-c-text-2);
}

.no-results-icon {
  font-size: 3rem;
  margin-bottom: 1rem;
}

.no-results h3 {
  margin: 0 0 0.5rem 0;
  color: var(--vp-c-text-1);
}

.search-suggestions {
  margin-top: 1.5rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  justify-content: center;
  align-items: center;
}

.suggestion-btn {
  padding: 0.25rem 0.5rem;
  border: 1px solid var(--vp-c-border);
  border-radius: var(--swit-radius-sm);
  background: var(--vp-c-bg-soft);
  color: var(--vp-c-brand-1);
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.3s ease;
}

.suggestion-btn:hover {
  background: var(--vp-c-brand-1);
  color: white;
}

/* Results List */
.results-list {
  padding: 0.5rem;
}

.result-item {
  padding: 1rem;
  border-radius: var(--swit-radius-md);
  cursor: pointer;
  transition: all 0.3s ease;
  margin-bottom: 0.5rem;
}

.result-item:hover,
.result-item.highlighted {
  background: var(--vp-c-bg-soft);
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 0.5rem;
}

.result-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.result-type {
  padding: 0.125rem 0.375rem;
  border-radius: var(--swit-radius-sm);
  font-size: 0.75rem;
  font-weight: 500;
  color: white;
}

.result-type.docs { background: #3b82f6; }
.result-type.api { background: #10b981; }
.result-type.example { background: #f59e0b; }
.result-type.guide { background: #8b5cf6; }

.result-title {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.result-title :deep(mark) {
  background: var(--vp-c-brand-soft);
  color: var(--vp-c-brand-1);
  padding: 0.125rem 0.25rem;
  border-radius: var(--swit-radius-sm);
}

.result-url {
  font-size: 0.75rem;
  color: var(--vp-c-text-3);
  font-family: monospace;
}

.result-excerpt {
  margin: 0;
  color: var(--vp-c-text-2);
  font-size: 0.875rem;
  line-height: 1.5;
}

.result-excerpt :deep(mark) {
  background: var(--vp-c-brand-soft);
  color: var(--vp-c-brand-1);
  padding: 0.125rem 0.25rem;
  border-radius: var(--swit-radius-sm);
}

.result-breadcrumb {
  font-size: 0.75rem;
  color: var(--vp-c-text-3);
  margin-top: 0.5rem;
}

.result-tags {
  display: flex;
  gap: 0.25rem;
  margin-top: 0.75rem;
  flex-wrap: wrap;
}

.result-tag {
  padding: 0.125rem 0.375rem;
  background: var(--vp-c-bg-alt);
  color: var(--vp-c-text-2);
  font-size: 0.625rem;
  border-radius: var(--swit-radius-sm);
}

/* Load More */
.load-more {
  padding: 1rem;
  text-align: center;
  border-top: 1px solid var(--vp-c-border);
}

.load-more-btn {
  padding: 0.5rem 1rem;
  border: 1px solid var(--vp-c-border);
  border-radius: var(--swit-radius-md);
  background: var(--vp-c-bg);
  color: var(--vp-c-text-1);
  cursor: pointer;
  transition: all 0.3s ease;
}

.load-more-btn:hover {
  background: var(--vp-c-bg-soft);
  border-color: var(--vp-c-brand-1);
}

/* Search Tips */
.search-tips {
  border-top: 1px solid var(--vp-c-border);
  padding: 1rem;
  background: var(--vp-c-bg-soft);
}

.search-tips summary {
  cursor: pointer;
  font-size: 0.875rem;
  color: var(--vp-c-text-2);
  font-weight: 500;
}

.tips-content {
  margin-top: 0.5rem;
}

.tips-content ul {
  margin: 0;
  padding-left: 1.5rem;
  font-size: 0.875rem;
  color: var(--vp-c-text-2);
}

.tips-content code {
  background: var(--vp-c-bg-alt);
  padding: 0.125rem 0.25rem;
  border-radius: var(--swit-radius-sm);
  font-family: monospace;
}

/* Search Overlay */
.search-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.3);
  z-index: 98;
}

/* Responsive Design */
@media (max-width: 768px) {
  .search-results {
    left: -1rem;
    right: -1rem;
    margin-top: 0.25rem;
  }

  .search-results-header {
    flex-direction: column;
    gap: 0.5rem;
    align-items: flex-start;
  }

  .result-header {
    flex-direction: column;
    gap: 0.25rem;
    align-items: flex-start;
  }

  .result-info {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.25rem;
  }

  .search-suggestions {
    flex-direction: column;
  }
}

/* Dark mode adaptations */
.dark .search-overlay {
  background: rgba(0, 0, 0, 0.6);
}
</style>