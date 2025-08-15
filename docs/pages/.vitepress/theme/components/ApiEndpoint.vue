<template>
  <div class="api-endpoint">
    <div class="endpoint-header">
      <div class="endpoint-title">
        <span class="method-badge" :class="endpoint.method.toLowerCase()">
          {{ endpoint.method }}
        </span>
        <code class="endpoint-path">{{ endpoint.path }}</code>
      </div>
      <div v-if="endpoint.deprecated" class="deprecated-badge">
        {{ $t('api.deprecated', 'Deprecated') }}
      </div>
    </div>

    <div v-if="endpoint.summary" class="endpoint-summary">
      <h4>{{ endpoint.summary }}</h4>
    </div>

    <div v-if="endpoint.description" class="endpoint-description">
      <p>{{ endpoint.description }}</p>
    </div>

    <div v-if="endpoint.tags && endpoint.tags.length" class="endpoint-tags">
      <span 
        v-for="tag in endpoint.tags" 
        :key="tag" 
        class="tag-badge"
      >
        {{ tag }}
      </span>
    </div>

    <!-- Parameters Section -->
    <div v-if="hasParameters" class="parameters-section">
      <h4>{{ $t('api.parameters', 'Parameters') }}</h4>
      
      <div v-if="pathParams.length" class="param-group">
        <h5>{{ $t('api.pathParams', 'Path Parameters') }}</h5>
        <div class="param-table">
          <div class="param-header">
            <div class="param-name">{{ $t('api.name', 'Name') }}</div>
            <div class="param-type">{{ $t('api.type', 'Type') }}</div>
            <div class="param-required">{{ $t('api.required', 'Required') }}</div>
            <div class="param-description">{{ $t('api.description', 'Description') }}</div>
          </div>
          <div 
            v-for="param in pathParams" 
            :key="param.name"
            class="param-row"
          >
            <div class="param-name">
              <code>{{ param.name }}</code>
            </div>
            <div class="param-type">{{ param.type }}</div>
            <div class="param-required">
              <span :class="param.required ? 'required' : 'optional'">
                {{ param.required ? $t('api.yes', 'Yes') : $t('api.no', 'No') }}
              </span>
            </div>
            <div class="param-description">{{ param.description || '-' }}</div>
          </div>
        </div>
      </div>

      <div v-if="queryParams.length" class="param-group">
        <h5>{{ $t('api.queryParams', 'Query Parameters') }}</h5>
        <div class="param-table">
          <div class="param-header">
            <div class="param-name">{{ $t('api.name', 'Name') }}</div>
            <div class="param-type">{{ $t('api.type', 'Type') }}</div>
            <div class="param-required">{{ $t('api.required', 'Required') }}</div>
            <div class="param-description">{{ $t('api.description', 'Description') }}</div>
          </div>
          <div 
            v-for="param in queryParams" 
            :key="param.name"
            class="param-row"
          >
            <div class="param-name">
              <code>{{ param.name }}</code>
            </div>
            <div class="param-type">{{ param.type }}</div>
            <div class="param-required">
              <span :class="param.required ? 'required' : 'optional'">
                {{ param.required ? $t('api.yes', 'Yes') : $t('api.no', 'No') }}
              </span>
            </div>
            <div class="param-description">{{ param.description || '-' }}</div>
          </div>
        </div>
      </div>

      <div v-if="bodyParams.length" class="param-group">
        <h5>{{ $t('api.requestBody', 'Request Body') }}</h5>
        <div class="param-table">
          <div class="param-header">
            <div class="param-name">{{ $t('api.field', 'Field') }}</div>
            <div class="param-type">{{ $t('api.type', 'Type') }}</div>
            <div class="param-required">{{ $t('api.required', 'Required') }}</div>
            <div class="param-description">{{ $t('api.description', 'Description') }}</div>
          </div>
          <div 
            v-for="param in bodyParams" 
            :key="param.name"
            class="param-row"
          >
            <div class="param-name">
              <code>{{ param.name }}</code>
            </div>
            <div class="param-type">{{ param.type }}</div>
            <div class="param-required">
              <span :class="param.required ? 'required' : 'optional'">
                {{ param.required ? $t('api.yes', 'Yes') : $t('api.no', 'No') }}
              </span>
            </div>
            <div class="param-description">{{ param.description || '-' }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Responses Section -->
    <div v-if="endpoint.responses && endpoint.responses.length" class="responses-section">
      <h4>{{ $t('api.responses', 'Responses') }}</h4>
      <div class="response-table">
        <div class="response-header">
          <div class="response-status">{{ $t('api.status', 'Status') }}</div>
          <div class="response-description">{{ $t('api.description', 'Description') }}</div>
        </div>
        <div 
          v-for="response in endpoint.responses" 
          :key="response.statusCode"
          class="response-row"
        >
          <div class="response-status">
            <span :class="getStatusClass(response.statusCode)">
              {{ response.statusCode }}
            </span>
          </div>
          <div class="response-description">{{ response.description || '-' }}</div>
        </div>
      </div>
    </div>

    <!-- Example Section -->
    <div class="example-section">
      <h4>{{ $t('api.example', 'Example') }}</h4>
      <div class="example-tabs">
        <button 
          :class="['tab-button', { active: activeTab === 'curl' }]"
          @click="activeTab = 'curl'"
        >
          cURL
        </button>
        <button 
          :class="['tab-button', { active: activeTab === 'javascript' }]"
          @click="activeTab = 'javascript'"
        >
          JavaScript
        </button>
        <button 
          :class="['tab-button', { active: activeTab === 'go' }]"
          @click="activeTab = 'go'"
        >
          Go
        </button>
      </div>
      
      <div class="example-content">
        <CodeExample 
          v-if="activeTab === 'curl'"
          :code="curlExample"
          language="bash"
          :copy="true"
        />
        <CodeExample 
          v-if="activeTab === 'javascript'"
          :code="jsExample"
          language="javascript"
          :copy="true"
        />
        <CodeExample 
          v-if="activeTab === 'go'"
          :code="goExample"
          language="go"
          :copy="true"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import CodeExample from './CodeExample.vue'

const props = defineProps({
  endpoint: {
    type: Object,
    required: true
  },
  baseUrl: {
    type: String,
    default: 'http://localhost:8080'
  }
})

const activeTab = ref('curl')

const hasParameters = computed(() => {
  return pathParams.value.length > 0 || 
         queryParams.value.length > 0 || 
         bodyParams.value.length > 0
})

const pathParams = computed(() => {
  return props.endpoint.parameters?.filter(p => p.in === 'path') || []
})

const queryParams = computed(() => {
  return props.endpoint.parameters?.filter(p => p.in === 'query') || []
})

const bodyParams = computed(() => {
  return props.endpoint.parameters?.filter(p => p.in === 'body' || p.in === 'formData') || []
})

const curlExample = computed(() => {
  let curl = `curl -X ${props.endpoint.method} "${props.baseUrl}${props.endpoint.path}"`
  
  // Add headers
  if (props.endpoint.method !== 'GET' && bodyParams.value.length > 0) {
    curl += ' \\\n  -H "Content-Type: application/json"'
  }
  
  // Add auth header example
  if (props.endpoint.security && props.endpoint.security.length > 0) {
    curl += ' \\\n  -H "Authorization: Bearer <your-token>"'
  }
  
  // Add body example
  if (bodyParams.value.length > 0) {
    curl += ' \\\n  -d \'{\n'
    bodyParams.value.forEach((param, index) => {
      const comma = index < bodyParams.value.length - 1 ? ',' : ''
      curl += `    "${param.name}": "example_value"${comma}\n`
    })
    curl += '  }\''
  }
  
  return curl
})

const jsExample = computed(() => {
  let js = `const response = await fetch('${props.baseUrl}${props.endpoint.path}', {\n`
  js += `  method: '${props.endpoint.method}',\n`
  
  // Add headers
  js += `  headers: {\n`
  if (props.endpoint.method !== 'GET' && bodyParams.value.length > 0) {
    js += `    'Content-Type': 'application/json',\n`
  }
  if (props.endpoint.security && props.endpoint.security.length > 0) {
    js += `    'Authorization': 'Bearer <your-token>',\n`
  }
  js += `  },\n`
  
  // Add body
  if (bodyParams.value.length > 0) {
    js += `  body: JSON.stringify({\n`
    bodyParams.value.forEach((param, index) => {
      const comma = index < bodyParams.value.length - 1 ? ',' : ''
      js += `    ${param.name}: 'example_value'${comma}\n`
    })
    js += `  })\n`
  }
  
  js += `});\n\nconst data = await response.json();\nconsole.log(data);`
  
  return js
})

const goExample = computed(() => {
  let go = `package main\n\nimport (\n    "bytes"\n    "encoding/json"\n    "fmt"\n    "net/http"\n)\n\n`
  go += `func main() {\n`
  
  if (bodyParams.value.length > 0) {
    go += `    payload := map[string]interface{}{\n`
    bodyParams.value.forEach(param => {
      go += `        "${param.name}": "example_value",\n`
    })
    go += `    }\n    \n    jsonData, _ := json.Marshal(payload)\n    \n`
    go += `    req, _ := http.NewRequest("${props.endpoint.method}", "${props.baseUrl}${props.endpoint.path}", bytes.NewBuffer(jsonData))\n`
    go += `    req.Header.Set("Content-Type", "application/json")\n`
  } else {
    go += `    req, _ := http.NewRequest("${props.endpoint.method}", "${props.baseUrl}${props.endpoint.path}", nil)\n`
  }
  
  if (props.endpoint.security && props.endpoint.security.length > 0) {
    go += `    req.Header.Set("Authorization", "Bearer <your-token>")\n`
  }
  
  go += `    \n    client := &http.Client{}\n    resp, err := client.Do(req)\n    if err != nil {\n        fmt.Println(err)\n        return\n    }\n    defer resp.Body.Close()\n    \n    fmt.Println("Status:", resp.Status)\n}`
  
  return go
})

const getStatusClass = (status) => {
  const code = parseInt(status)
  if (code >= 200 && code < 300) return 'status-success'
  if (code >= 400 && code < 500) return 'status-client-error'
  if (code >= 500) return 'status-server-error'
  return 'status-info'
}
</script>

<style scoped>
.api-endpoint {
  max-width: none;
}

.endpoint-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--vp-c-divider);
}

.endpoint-title {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.method-badge {
  padding: 0.25rem 0.5rem;
  font-size: 0.875rem;
  font-weight: 600;
  text-transform: uppercase;
  border-radius: 4px;
  color: white;
}

.method-badge.get { background: #10b981; }
.method-badge.post { background: #3b82f6; }
.method-badge.put { background: #f59e0b; }
.method-badge.delete { background: #ef4444; }
.method-badge.patch { background: #8b5cf6; }

.endpoint-path {
  font-family: var(--vp-font-family-mono);
  font-size: 1.1rem;
  background: var(--vp-c-bg-soft);
  padding: 0.5rem;
  border-radius: 4px;
  color: var(--vp-c-text-1);
}

.deprecated-badge {
  padding: 0.25rem 0.5rem;
  background: #ef4444;
  color: white;
  font-size: 0.75rem;
  border-radius: 4px;
  text-transform: uppercase;
}

.endpoint-summary h4 {
  margin: 0 0 0.5rem 0;
  color: var(--vp-c-text-1);
}

.endpoint-description {
  margin-bottom: 1.5rem;
  color: var(--vp-c-text-2);
}

.endpoint-tags {
  margin-bottom: 1.5rem;
}

.tag-badge {
  display: inline-block;
  padding: 0.125rem 0.5rem;
  margin-right: 0.5rem;
  background: var(--vp-c-brand-soft);
  color: var(--vp-c-brand-1);
  font-size: 0.75rem;
  border-radius: 12px;
}

.parameters-section,
.responses-section {
  margin-bottom: 2rem;
}

.parameters-section h4,
.responses-section h4 {
  margin-bottom: 1rem;
  color: var(--vp-c-text-1);
}

.param-group {
  margin-bottom: 1.5rem;
}

.param-group h5 {
  margin-bottom: 0.75rem;
  color: var(--vp-c-text-1);
  font-size: 1rem;
}

.param-table,
.response-table {
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  overflow: hidden;
}

.param-header,
.response-header {
  display: grid;
  grid-template-columns: 1fr 1fr 100px 2fr;
  background: var(--vp-c-bg-soft);
  font-weight: 600;
  border-bottom: 1px solid var(--vp-c-divider);
}

.response-header {
  grid-template-columns: 100px 1fr;
}

.param-header > div,
.response-header > div,
.param-row > div,
.response-row > div {
  padding: 0.75rem;
  border-right: 1px solid var(--vp-c-divider);
}

.param-header > div:last-child,
.response-header > div:last-child,
.param-row > div:last-child,
.response-row > div:last-child {
  border-right: none;
}

.param-row,
.response-row {
  display: grid;
  grid-template-columns: 1fr 1fr 100px 2fr;
  border-bottom: 1px solid var(--vp-c-divider);
}

.response-row {
  grid-template-columns: 100px 1fr;
}

.param-row:last-child,
.response-row:last-child {
  border-bottom: none;
}

.param-name code {
  background: var(--vp-c-bg-soft);
  padding: 0.25rem;
  border-radius: 3px;
  font-size: 0.875rem;
}

.required {
  color: #ef4444;
  font-weight: 600;
}

.optional {
  color: var(--vp-c-text-3);
}

.status-success { color: #10b981; font-weight: 600; }
.status-client-error { color: #f59e0b; font-weight: 600; }
.status-server-error { color: #ef4444; font-weight: 600; }
.status-info { color: var(--vp-c-text-2); }

.example-section {
  margin-top: 2rem;
}

.example-section h4 {
  margin-bottom: 1rem;
  color: var(--vp-c-text-1);
}

.example-tabs {
  display: flex;
  margin-bottom: 1rem;
  border-bottom: 1px solid var(--vp-c-divider);
}

.tab-button {
  padding: 0.5rem 1rem;
  border: none;
  background: none;
  color: var(--vp-c-text-2);
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: all 0.2s;
}

.tab-button:hover {
  color: var(--vp-c-text-1);
}

.tab-button.active {
  color: var(--vp-c-brand-1);
  border-bottom-color: var(--vp-c-brand-1);
}

@media (max-width: 768px) {
  .param-header,
  .param-row {
    grid-template-columns: 1fr;
    gap: 0;
  }
  
  .param-header > div,
  .param-row > div {
    border-right: none;
    border-bottom: 1px solid var(--vp-c-divider);
  }
  
  .param-header > div:last-child,
  .param-row > div:last-child {
    border-bottom: none;
  }
}
</style>