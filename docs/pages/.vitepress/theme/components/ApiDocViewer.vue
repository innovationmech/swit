<template>
  <div class="api-doc-viewer">
    <!-- API Service Header -->
    <div class="api-header">
      <div class="service-info">
        <h1>{{ serviceInfo.title }}</h1>
        <p class="service-description">{{ serviceInfo.description }}</p>
        <div class="service-badges">
          <span class="version-badge">v{{ serviceInfo.version }}</span>
          <span class="base-url-badge">{{ serviceInfo.baseUrl }}</span>
          <span :class="['status-badge', serviceInfo.status]">{{ serviceInfo.status }}</span>
        </div>
      </div>
      <div class="api-actions">
        <button @click="downloadOpenAPI" class="action-btn">
          📄 下载 OpenAPI
        </button>
        <button @click="viewInSwagger" class="action-btn">
          🗂️ Swagger UI
        </button>
      </div>
    </div>

    <!-- API Statistics -->
    <div class="api-stats">
      <div class="stats-grid">
        <div class="stat-card">
          <span class="stat-number">{{ stats.endpoints }}</span>
          <span class="stat-label">端点数量</span>
        </div>
        <div class="stat-card">
          <span class="stat-number">{{ stats.models }}</span>
          <span class="stat-label">数据模型</span>
        </div>
        <div class="stat-card">
          <span class="stat-number">{{ stats.methods }}</span>
          <span class="stat-label">HTTP 方法</span>
        </div>
        <div class="stat-card">
          <span class="stat-number">{{ stats.tags }}</span>
          <span class="stat-label">分组标签</span>
        </div>
      </div>
    </div>

    <!-- API Navigation -->
    <div class="api-navigation">
      <div class="nav-tabs">
        <button 
          v-for="section in sections" 
          :key="section.id"
          :class="['nav-tab', { active: activeSection === section.id }]"
          @click="activeSection = section.id"
        >
          {{ section.title }}
        </button>
      </div>
    </div>

    <!-- API Content -->
    <div class="api-content">
      <!-- Endpoints Section -->
      <div v-if="activeSection === 'endpoints'" class="endpoints-section">
        <div class="endpoint-groups">
          <div 
            v-for="group in endpointGroups" 
            :key="group.tag"
            class="endpoint-group"
          >
            <h2 class="group-title">{{ group.title }}</h2>
            <div class="endpoints-list">
              <div 
                v-for="endpoint in group.endpoints" 
                :key="endpoint.path + endpoint.method"
                class="endpoint-card"
              >
                <div class="endpoint-header" @click="toggleEndpoint(endpoint.id)">
                  <div class="endpoint-method-path">
                    <span :class="['method-badge', endpoint.method.toLowerCase()]">
                      {{ endpoint.method.toUpperCase() }}
                    </span>
                    <code class="endpoint-path">{{ endpoint.path }}</code>
                  </div>
                  <div class="endpoint-summary">{{ endpoint.summary }}</div>
                  <span :class="['expand-icon', { expanded: expandedEndpoints.has(endpoint.id) }]">
                    🔽
                  </span>
                </div>
                
                <div v-if="expandedEndpoints.has(endpoint.id)" class="endpoint-details">
                  <div class="endpoint-description">
                    <p>{{ endpoint.description }}</p>
                  </div>

                  <!-- Parameters -->
                  <div v-if="endpoint.parameters?.length" class="parameters-section">
                    <h4>参数</h4>
                    <div class="parameters-table">
                      <table>
                        <thead>
                          <tr>
                            <th>名称</th>
                            <th>类型</th>
                            <th>位置</th>
                            <th>必需</th>
                            <th>描述</th>
                          </tr>
                        </thead>
                        <tbody>
                          <tr v-for="param in endpoint.parameters" :key="param.name">
                            <td><code>{{ param.name }}</code></td>
                            <td><span class="type-badge">{{ param.type }}</span></td>
                            <td><span class="location-badge">{{ param.in }}</span></td>
                            <td>
                              <span :class="['required-badge', { required: param.required }]">
                                {{ param.required ? '是' : '否' }}
                              </span>
                            </td>
                            <td>{{ param.description }}</td>
                          </tr>
                        </tbody>
                      </table>
                    </div>
                  </div>

                  <!-- Request Body -->
                  <div v-if="endpoint.requestBody" class="request-section">
                    <h4>请求体</h4>
                    <div class="code-block">
                      <pre><code>{{ JSON.stringify(endpoint.requestBody, null, 2) }}</code></pre>
                    </div>
                  </div>

                  <!-- Responses -->
                  <div v-if="endpoint.responses" class="responses-section">
                    <h4>响应</h4>
                    <div class="response-tabs">
                      <div 
                        v-for="(response, statusCode) in endpoint.responses" 
                        :key="statusCode"
                        class="response-item"
                      >
                        <div class="response-header">
                          <span :class="['status-code', getStatusClass(statusCode)]">
                            {{ statusCode }}
                          </span>
                          <span class="response-description">{{ response.description }}</span>
                        </div>
                        <div v-if="response.example" class="response-example">
                          <div class="code-block">
                            <pre><code>{{ JSON.stringify(response.example, null, 2) }}</code></pre>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <!-- Try It Out -->
                  <div class="try-it-section">
                    <button @click="tryEndpoint(endpoint)" class="try-btn">
                      🚀 测试接口
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Models Section -->
      <div v-if="activeSection === 'models'" class="models-section">
        <div class="models-grid">
          <div 
            v-for="model in models" 
            :key="model.name"
            class="model-card"
          >
            <h3>{{ model.name }}</h3>
            <p class="model-description">{{ model.description }}</p>
            <div class="model-properties">
              <table>
                <thead>
                  <tr>
                    <th>属性</th>
                    <th>类型</th>
                    <th>必需</th>
                    <th>描述</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="prop in model.properties" :key="prop.name">
                    <td><code>{{ prop.name }}</code></td>
                    <td><span class="type-badge">{{ prop.type }}</span></td>
                    <td>
                      <span :class="['required-badge', { required: prop.required }]">
                        {{ prop.required ? '是' : '否' }}
                      </span>
                    </td>
                    <td>{{ prop.description }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>

      <!-- Overview Section -->
      <div v-if="activeSection === 'overview'" class="overview-section">
        <div class="overview-content">
          <h2>服务概览</h2>
          <p>{{ serviceInfo.longDescription }}</p>
          
          <h3>主要功能</h3>
          <ul>
            <li v-for="feature in serviceInfo.features" :key="feature">{{ feature }}</li>
          </ul>

          <h3>认证方式</h3>
          <div class="auth-info">
            <div v-for="auth in serviceInfo.authentication" :key="auth.type" class="auth-method">
              <h4>{{ auth.name }}</h4>
              <p>{{ auth.description }}</p>
              <div class="code-block">
                <pre><code>{{ auth.example }}</code></pre>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted } from 'vue'

export default {
  name: 'ApiDocViewer',
  props: {
    serviceName: {
      type: String,
      required: true
    },
    apiData: {
      type: Object,
      default: () => ({})
    }
  },
  setup(props) {
    const activeSection = ref('overview')
    const expandedEndpoints = ref(new Set())

    // Mock service data - in real implementation, this would come from props.apiData
    const serviceInfo = ref({
      title: 'Swit Framework API',
      description: '高性能微服务框架 API 文档',
      longDescription: 'Swit Framework 提供完整的微服务开发解决方案，包括HTTP和gRPC双协议支持、服务发现、依赖注入、性能监控等企业级功能。',
      version: '1.0.0',
      baseUrl: 'https://api.example.com/v1',
      status: 'healthy',
      features: [
        '统一的服务器框架，支持完整的生命周期管理',
        'HTTP 和 gRPC 双协议支持',
        '基于工厂模式的依赖注入系统',
        '内置性能监控和指标收集',
        '服务发现和健康检查集成',
        '丰富的中间件支持'
      ],
      authentication: [
        {
          type: 'bearer',
          name: 'Bearer Token',
          description: '使用JWT Bearer Token进行API认证',
          example: 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...'
        }
      ]
    })

    const stats = ref({
      endpoints: 12,
      models: 8,
      methods: 5,
      tags: 3
    })

    const sections = ref([
      { id: 'overview', title: '概览' },
      { id: 'endpoints', title: 'API 端点' },
      { id: 'models', title: '数据模型' }
    ])

    const endpointGroups = ref([
      {
        tag: 'users',
        title: '用户管理',
        endpoints: [
          {
            id: 'get-users',
            method: 'GET',
            path: '/api/v1/users',
            summary: '获取用户列表',
            description: '获取系统中所有用户的列表，支持分页和筛选',
            parameters: [
              { name: 'page', type: 'integer', in: 'query', required: false, description: '页码，从1开始' },
              { name: 'limit', type: 'integer', in: 'query', required: false, description: '每页数量，默认10' },
              { name: 'search', type: 'string', in: 'query', required: false, description: '搜索关键词' }
            ],
            responses: {
              '200': {
                description: '成功返回用户列表',
                example: {
                  status: 'success',
                  data: {
                    users: [
                      { id: 1, name: '张三', email: 'zhang@example.com' }
                    ],
                    pagination: { page: 1, limit: 10, total: 100 }
                  }
                }
              }
            }
          },
          {
            id: 'create-user',
            method: 'POST',
            path: '/api/v1/users',
            summary: '创建新用户',
            description: '在系统中创建一个新的用户账户',
            requestBody: {
              name: '张三',
              email: 'zhang@example.com',
              password: 'securepassword123'
            },
            responses: {
              '201': {
                description: '用户创建成功',
                example: {
                  status: 'success',
                  data: { id: 1, name: '张三', email: 'zhang@example.com' }
                }
              }
            }
          }
        ]
      },
      {
        tag: 'auth',
        title: '认证服务',
        endpoints: [
          {
            id: 'login',
            method: 'POST',
            path: '/api/v1/auth/login',
            summary: '用户登录',
            description: '使用邮箱和密码进行用户登录，返回JWT令牌',
            requestBody: {
              email: 'user@example.com',
              password: 'password123'
            },
            responses: {
              '200': {
                description: '登录成功',
                example: {
                  status: 'success',
                  data: {
                    token: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...',
                    user: { id: 1, name: '张三', email: 'zhang@example.com' }
                  }
                }
              }
            }
          }
        ]
      }
    ])

    const models = ref([
      {
        name: 'User',
        description: '用户数据模型',
        properties: [
          { name: 'id', type: 'integer', required: true, description: '用户唯一标识符' },
          { name: 'name', type: 'string', required: true, description: '用户姓名' },
          { name: 'email', type: 'string', required: true, description: '用户邮箱' },
          { name: 'created_at', type: 'datetime', required: true, description: '创建时间' },
          { name: 'updated_at', type: 'datetime', required: true, description: '更新时间' }
        ]
      },
      {
        name: 'AuthToken',
        description: '认证令牌模型',
        properties: [
          { name: 'token', type: 'string', required: true, description: 'JWT令牌字符串' },
          { name: 'expires_at', type: 'datetime', required: true, description: '令牌过期时间' },
          { name: 'type', type: 'string', required: true, description: '令牌类型' }
        ]
      }
    ])

    const toggleEndpoint = (endpointId) => {
      if (expandedEndpoints.value.has(endpointId)) {
        expandedEndpoints.value.delete(endpointId)
      } else {
        expandedEndpoints.value.add(endpointId)
      }
    }

    const getStatusClass = (statusCode) => {
      const code = parseInt(statusCode)
      if (code >= 200 && code < 300) return 'success'
      if (code >= 300 && code < 400) return 'redirect'
      if (code >= 400 && code < 500) return 'client-error'
      if (code >= 500) return 'server-error'
      return ''
    }

    const downloadOpenAPI = () => {
      // Implementation for downloading OpenAPI spec
      console.log('Downloading OpenAPI spec...')
    }

    const viewInSwagger = () => {
      // Implementation for opening Swagger UI
      console.log('Opening Swagger UI...')
    }

    const tryEndpoint = (endpoint) => {
      // Implementation for API testing
      console.log('Testing endpoint:', endpoint)
    }

    return {
      activeSection,
      expandedEndpoints,
      serviceInfo,
      stats,
      sections,
      endpointGroups,
      models,
      toggleEndpoint,
      getStatusClass,
      downloadOpenAPI,
      viewInSwagger,
      tryEndpoint
    }
  }
}
</script>

<style scoped>
.api-doc-viewer {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 1rem;
}

/* API Header */
.api-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 2rem 0;
  border-bottom: 1px solid var(--vp-c-border);
  margin-bottom: 2rem;
}

.service-info h1 {
  margin: 0 0 0.5rem 0;
  font-size: 2.5rem;
  font-weight: bold;
}

.service-description {
  color: var(--vp-c-text-2);
  font-size: 1.1rem;
  margin: 0 0 1rem 0;
}

.service-badges {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.version-badge,
.base-url-badge,
.status-badge {
  padding: 0.25rem 0.75rem;
  border-radius: var(--swit-radius-sm);
  font-size: 0.875rem;
  font-weight: 600;
}

.version-badge {
  background: var(--vp-c-brand-soft);
  color: var(--vp-c-brand-1);
}

.base-url-badge {
  background: var(--vp-c-bg-alt);
  color: var(--vp-c-text-2);
  font-family: monospace;
}

.status-badge.healthy {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.api-actions {
  display: flex;
  gap: 0.5rem;
}

.action-btn {
  padding: 0.5rem 1rem;
  border: 1px solid var(--vp-c-border);
  border-radius: var(--swit-radius-md);
  background: var(--vp-c-bg);
  color: var(--vp-c-text-1);
  cursor: pointer;
  transition: all 0.3s ease;
}

.action-btn:hover {
  background: var(--vp-c-bg-soft);
  border-color: var(--vp-c-brand-1);
}

/* API Stats */
.api-stats {
  margin-bottom: 2rem;
}

/* API Navigation */
.api-navigation {
  margin-bottom: 2rem;
}

.nav-tabs {
  display: flex;
  border-bottom: 1px solid var(--vp-c-border);
}

.nav-tab {
  padding: 0.75rem 1.5rem;
  border: none;
  background: transparent;
  color: var(--vp-c-text-2);
  cursor: pointer;
  transition: all 0.3s ease;
  border-bottom: 2px solid transparent;
}

.nav-tab:hover,
.nav-tab.active {
  color: var(--vp-c-brand-1);
  border-bottom-color: var(--vp-c-brand-1);
}

/* Endpoints Section */
.endpoint-group {
  margin-bottom: 3rem;
}

.group-title {
  font-size: 1.5rem;
  font-weight: 600;
  margin-bottom: 1rem;
  color: var(--vp-c-text-1);
}

.endpoint-card {
  border: 1px solid var(--vp-c-border);
  border-radius: var(--swit-radius-lg);
  margin-bottom: 1rem;
  overflow: hidden;
}

.endpoint-header {
  padding: 1rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 1rem;
  background: var(--vp-c-bg-soft);
  transition: background 0.3s ease;
}

.endpoint-header:hover {
  background: var(--vp-c-bg-alt);
}

.endpoint-method-path {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.method-badge {
  padding: 0.25rem 0.5rem;
  border-radius: var(--swit-radius-sm);
  font-size: 0.75rem;
  font-weight: bold;
  color: white;
  min-width: 60px;
  text-align: center;
}

.method-badge.get { background: #22c55e; }
.method-badge.post { background: #3b82f6; }
.method-badge.put { background: #f59e0b; }
.method-badge.delete { background: #ef4444; }
.method-badge.patch { background: #8b5cf6; }

.endpoint-path {
  font-family: monospace;
  background: var(--vp-c-bg);
  padding: 0.25rem 0.5rem;
  border-radius: var(--swit-radius-sm);
  color: var(--vp-c-brand-1);
}

.endpoint-summary {
  flex: 1;
  color: var(--vp-c-text-1);
  font-weight: 500;
}

.expand-icon {
  transition: transform 0.3s ease;
}

.expand-icon.expanded {
  transform: rotate(180deg);
}

.endpoint-details {
  padding: 1.5rem;
  background: var(--vp-c-bg);
  border-top: 1px solid var(--vp-c-border);
}

.endpoint-description {
  margin-bottom: 1.5rem;
}

.parameters-section,
.request-section,
.responses-section,
.try-it-section {
  margin-bottom: 1.5rem;
}

.parameters-section h4,
.request-section h4,
.responses-section h4 {
  margin: 0 0 1rem 0;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.parameters-table table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 0.5rem;
}

.parameters-table th,
.parameters-table td {
  padding: 0.75rem;
  text-align: left;
  border: 1px solid var(--vp-c-border);
}

.parameters-table th {
  background: var(--vp-c-bg-soft);
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.type-badge,
.location-badge {
  padding: 0.125rem 0.375rem;
  border-radius: var(--swit-radius-sm);
  font-size: 0.75rem;
  font-weight: 500;
}

.type-badge {
  background: var(--vp-c-brand-soft);
  color: var(--vp-c-brand-1);
}

.location-badge {
  background: var(--vp-c-bg-alt);
  color: var(--vp-c-text-2);
}

.required-badge.required {
  color: #ef4444;
  font-weight: 600;
}

.code-block {
  background: var(--vp-c-bg-alt);
  border: 1px solid var(--vp-c-border);
  border-radius: var(--swit-radius-md);
  overflow-x: auto;
}

.code-block pre {
  margin: 0;
  padding: 1rem;
  font-family: monospace;
  font-size: 0.875rem;
  line-height: 1.5;
}

.response-item {
  margin-bottom: 1rem;
}

.response-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 0.5rem;
}

.status-code {
  padding: 0.25rem 0.5rem;
  border-radius: var(--swit-radius-sm);
  font-weight: bold;
  color: white;
}

.status-code.success { background: #22c55e; }
.status-code.redirect { background: #3b82f6; }
.status-code.client-error { background: #f59e0b; }
.status-code.server-error { background: #ef4444; }

.try-btn {
  padding: 0.5rem 1rem;
  background: var(--vp-c-brand-1);
  color: white;
  border: none;
  border-radius: var(--swit-radius-md);
  cursor: pointer;
  font-weight: 500;
  transition: background 0.3s ease;
}

.try-btn:hover {
  background: var(--vp-c-brand-2);
}

/* Models Section */
.models-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 1.5rem;
}

.model-card {
  border: 1px solid var(--vp-c-border);
  border-radius: var(--swit-radius-lg);
  padding: 1.5rem;
  background: var(--vp-c-bg-soft);
}

.model-card h3 {
  margin: 0 0 0.5rem 0;
  color: var(--vp-c-brand-1);
}

.model-description {
  color: var(--vp-c-text-2);
  margin: 0 0 1rem 0;
}

.model-properties table {
  width: 100%;
  border-collapse: collapse;
}

.model-properties th,
.model-properties td {
  padding: 0.5rem;
  text-align: left;
  border: 1px solid var(--vp-c-border);
  font-size: 0.875rem;
}

.model-properties th {
  background: var(--vp-c-bg);
  font-weight: 600;
}

/* Overview Section */
.overview-content h2,
.overview-content h3 {
  color: var(--vp-c-text-1);
}

.auth-info {
  margin-top: 1.5rem;
}

.auth-method {
  margin-bottom: 1.5rem;
  padding: 1rem;
  border: 1px solid var(--vp-c-border);
  border-radius: var(--swit-radius-lg);
  background: var(--vp-c-bg-soft);
}

.auth-method h4 {
  margin: 0 0 0.5rem 0;
  color: var(--vp-c-brand-1);
}

/* Responsive Design */
@media (max-width: 768px) {
  .api-header {
    flex-direction: column;
    gap: 1rem;
  }

  .nav-tabs {
    overflow-x: auto;
    white-space: nowrap;
  }

  .endpoint-method-path {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.5rem;
  }

  .parameters-table,
  .model-properties table {
    font-size: 0.75rem;
  }
  
  .models-grid {
    grid-template-columns: 1fr;
  }
}
</style>