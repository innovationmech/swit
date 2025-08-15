<template>
  <div class="api-explorer">
    <div class="api-sidebar">
      <div class="api-search">
        <input 
          v-model="searchQuery" 
          :placeholder="$t('api.search', 'Search APIs...')"
          class="search-input"
          @input="filterEndpoints"
        />
      </div>
      
      <nav class="api-nav">
        <div v-for="service in filteredServices" :key="service.name" class="service-group">
          <h3 class="service-title">{{ service.title }}</h3>
          <ul class="endpoint-list">
            <li 
              v-for="endpoint in service.endpoints" 
              :key="endpoint.id"
              class="endpoint-item"
            >
              <a 
                :href="`#${endpoint.id}`"
                :class="['endpoint-link', endpoint.method.toLowerCase()]"
                @click="selectEndpoint(endpoint)"
              >
                <span class="method-badge" :class="endpoint.method.toLowerCase()">
                  {{ endpoint.method }}
                </span>
                <span class="endpoint-path">{{ endpoint.path }}</span>
              </a>
            </li>
          </ul>
        </div>
      </nav>
    </div>
    
    <div class="api-content">
      <div v-if="selectedEndpoint" class="endpoint-details">
        <ApiEndpoint :endpoint="selectedEndpoint" />
      </div>
      <div v-else class="endpoint-placeholder">
        <h3>{{ $t('api.selectEndpoint', 'Select an API endpoint') }}</h3>
        <p>{{ $t('api.selectDescription', 'Choose an endpoint from the sidebar to view its details.') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import ApiEndpoint from './ApiEndpoint.vue'

const props = defineProps({
  services: {
    type: Array,
    default: () => []
  }
})

const searchQuery = ref('')
const selectedEndpoint = ref(null)

const filteredServices = computed(() => {
  if (!searchQuery.value) {
    return props.services
  }
  
  return props.services.map(service => ({
    ...service,
    endpoints: service.endpoints.filter(endpoint => 
      endpoint.path.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
      endpoint.summary.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
      endpoint.method.toLowerCase().includes(searchQuery.value.toLowerCase())
    )
  })).filter(service => service.endpoints.length > 0)
})

const selectEndpoint = (endpoint) => {
  selectedEndpoint.value = endpoint
  // Update URL hash
  window.location.hash = endpoint.id
}

const filterEndpoints = () => {
  // Reset selection when filtering
  selectedEndpoint.value = null
}

onMounted(() => {
  // Select endpoint from URL hash if available
  const hash = window.location.hash.substring(1)
  if (hash) {
    for (const service of props.services) {
      const endpoint = service.endpoints.find(e => e.id === hash)
      if (endpoint) {
        selectedEndpoint.value = endpoint
        break
      }
    }
  }
})
</script>

<style scoped>
.api-explorer {
  display: flex;
  min-height: 600px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  overflow: hidden;
}

.api-sidebar {
  width: 300px;
  background: var(--vp-c-bg-soft);
  border-right: 1px solid var(--vp-c-divider);
  display: flex;
  flex-direction: column;
}

.api-search {
  padding: 1rem;
  border-bottom: 1px solid var(--vp-c-divider);
}

.search-input {
  width: 100%;
  padding: 0.5rem;
  border: 1px solid var(--vp-c-divider);
  border-radius: 4px;
  font-size: 0.875rem;
  background: var(--vp-c-bg);
  color: var(--vp-c-text-1);
}

.search-input:focus {
  outline: none;
  border-color: var(--vp-c-brand-1);
}

.api-nav {
  flex: 1;
  overflow-y: auto;
  padding: 1rem;
}

.service-group {
  margin-bottom: 2rem;
}

.service-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--vp-c-text-1);
  margin-bottom: 0.5rem;
}

.endpoint-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.endpoint-item {
  margin-bottom: 0.25rem;
}

.endpoint-link {
  display: flex;
  align-items: center;
  padding: 0.5rem;
  text-decoration: none;
  border-radius: 4px;
  transition: background-color 0.2s;
}

.endpoint-link:hover {
  background: var(--vp-c-bg);
}

.method-badge {
  display: inline-block;
  padding: 0.125rem 0.375rem;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  border-radius: 3px;
  margin-right: 0.5rem;
  min-width: 45px;
  text-align: center;
}

.method-badge.get {
  background: #10b981;
  color: white;
}

.method-badge.post {
  background: #3b82f6;
  color: white;
}

.method-badge.put {
  background: #f59e0b;
  color: white;
}

.method-badge.delete {
  background: #ef4444;
  color: white;
}

.endpoint-path {
  font-family: var(--vp-font-family-mono);
  font-size: 0.875rem;
  color: var(--vp-c-text-2);
}

.api-content {
  flex: 1;
  padding: 2rem;
  overflow-y: auto;
}

.endpoint-placeholder {
  text-align: center;
  color: var(--vp-c-text-2);
  margin-top: 2rem;
}

@media (max-width: 768px) {
  .api-explorer {
    flex-direction: column;
  }
  
  .api-sidebar {
    width: 100%;
    max-height: 300px;
  }
}
</style>