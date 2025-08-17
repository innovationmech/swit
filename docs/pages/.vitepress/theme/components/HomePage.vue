<template>
  <div class="home-page">
    <!-- Hero Section -->
    <div class="hero-section">
      <div class="hero-content">
        <h1 class="hero-title">
          <span class="gradient-text">Swit</span>
          <span class="hero-subtitle">微服务框架</span>
        </h1>
        <p class="hero-description">
          高性能、可扩展的 Go 微服务框架，提供完整的企业级解决方案
        </p>
        <div class="hero-actions">
          <a href="/guide/getting-started" class="btn-primary">快速开始</a>
          <a href="/api/" class="btn-secondary">API 文档</a>
        </div>
      </div>
      <div class="hero-image">
        <div class="code-preview">
          <div class="code-header">
            <span class="code-lang">main.go</span>
          </div>
          <pre class="code-content"><code>package main

import (
    "context"
    "github.com/innovationmech/swit/pkg/server"
)

func main() {
    config := &server.ServerConfig{
        Name: "my-service",
        Port: 8080,
    }
    
    srv, _ := server.NewBusinessServerCore(
        config, myService, deps
    )
    
    srv.Start(context.Background())
}</code></pre>
        </div>
      </div>
    </div>

    <!-- Stats Section -->
    <div class="stats-section">
      <div class="stats-grid">
        <div class="stat-card fade-in-up">
          <span class="stat-number">{{ stats.services }}</span>
          <span class="stat-label">服务数量</span>
        </div>
        <div class="stat-card fade-in-up">
          <span class="stat-number">{{ stats.endpoints }}</span>
          <span class="stat-label">API 端点</span>
        </div>
        <div class="stat-card fade-in-up">
          <span class="stat-number">{{ stats.coverage }}%</span>
          <span class="stat-label">测试覆盖率</span>
        </div>
        <div class="stat-card fade-in-up">
          <span class="stat-number">{{ stats.performance }}ms</span>
          <span class="stat-label">平均响应时间</span>
        </div>
      </div>
    </div>

    <!-- Services Section -->
    <div class="services-section">
      <h2 class="section-title">核心服务</h2>
      <div class="service-grid">
        <div v-for="service in services" :key="service.name" class="service-card fade-in-up">
          <h3>{{ service.name }}</h3>
          <p>{{ service.description }}</p>
          <div class="service-stats">
            <span>端点: {{ service.endpoints }}</span> | 
            <span>状态: <span class="status-badge" :class="service.status">{{ service.status }}</span></span>
          </div>
          <p><a :href="service.docs">查看文档 →</a></p>
        </div>
      </div>
    </div>

    <!-- Features Section -->
    <div class="features-section">
      <h2 class="section-title">核心特性</h2>
      <div class="features-grid">
        <FeatureCard
          v-for="feature in features"
          :key="feature.title"
          :icon="feature.icon"
          :title="feature.title"
          :description="feature.description"
          :link="feature.link"
        />
      </div>
    </div>

    <!-- Code Examples Section -->
    <div class="examples-section">
      <h2 class="section-title">代码示例</h2>
      <div class="examples-grid">
        <CodeExample
          v-for="example in examples"
          :key="example.title"
          :title="example.title"
          :description="example.description"
          :code="example.code"
          :language="example.language"
        />
      </div>
    </div>
  </div>
</template>

<script>
import { ref, onMounted } from 'vue'
import FeatureCard from './FeatureCard.vue'
import CodeExample from './CodeExample.vue'

export default {
  name: 'HomePage',
  components: {
    FeatureCard,
    CodeExample
  },
  setup() {
    const stats = ref({
      services: 0,
      endpoints: 0,
      coverage: 0,
      performance: 0
    })

    const services = ref([
      {
        name: 'swit-serve',
        description: '用户管理和核心业务服务',
        endpoints: 12,
        status: 'healthy',
        docs: '/api/switserve'
      },
      {
        name: 'swit-auth',
        description: '身份认证和授权服务',
        endpoints: 8,
        status: 'healthy',
        docs: '/api/switauth'
      }
    ])

    const features = ref([
      {
        icon: '🚀',
        title: '高性能',
        description: '基于 Go 构建，支持高并发处理',
        link: '/guide/performance'
      },
      {
        icon: '🔧',
        title: '易于使用',
        description: '简洁的 API 设计，快速上手',
        link: '/guide/getting-started'
      },
      {
        icon: '📦',
        title: '模块化',
        description: '组件化架构，按需使用',
        link: '/guide/architecture'
      },
      {
        icon: '🔒',
        title: '安全可靠',
        description: '内置安全机制和错误处理',
        link: '/guide/security'
      },
      {
        icon: '📊',
        title: '监控完善',
        description: '内置性能监控和日志系统',
        link: '/guide/monitoring'
      },
      {
        icon: '🌐',
        title: '多协议',
        description: '支持 HTTP 和 gRPC 协议',
        link: '/guide/protocols'
      }
    ])

    const examples = ref([
      {
        title: 'HTTP 服务',
        description: '创建一个简单的 HTTP 服务',
        language: 'go',
        code: `package main

import (
    "github.com/innovationmech/swit/pkg/server"
    "github.com/innovationmech/swit/pkg/transport"
)

type MyHTTPHandler struct{}

func (h *MyHTTPHandler) RegisterRoutes(router interface{}) error {
    r := router.(*gin.Engine)
    r.GET("/hello", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "Hello, World!"})
    })
    return nil
}

func main() {
    config := &server.ServerConfig{Name: "my-service", Port: 8080}
    srv, _ := server.NewBusinessServerCore(config, &MyHTTPHandler{}, nil)
    srv.Start(context.Background())
}`
      },
      {
        title: 'gRPC 服务',
        description: '创建一个 gRPC 服务',
        language: 'go',
        code: `package main

import (
    "context"
    pb "your-project/api/gen"
)

type MyGRPCService struct {
    pb.UnimplementedMyServiceServer
}

func (s *MyGRPCService) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
    return &pb.HelloResponse{
        Message: "Hello " + req.Name,
    }, nil
}

func (s *MyGRPCService) RegisterGRPC(server interface{}) error {
    grpcServer := server.(*grpc.Server)
    pb.RegisterMyServiceServer(grpcServer, s)
    return nil
}`
      }
    ])

    // 模拟数据加载动画
    const animateStats = () => {
      const targets = { services: 2, endpoints: 20, coverage: 85, performance: 45 }
      const duration = 2000
      const steps = 60
      const interval = duration / steps

      let step = 0
      const timer = setInterval(() => {
        step++
        const progress = step / steps

        stats.value = {
          services: Math.floor(targets.services * progress),
          endpoints: Math.floor(targets.endpoints * progress),
          coverage: Math.floor(targets.coverage * progress),
          performance: Math.floor(targets.performance * progress)
        }

        if (step >= steps) {
          clearInterval(timer)
          stats.value = targets
        }
      }, interval)
    }

    onMounted(() => {
      animateStats()
    })

    return {
      stats,
      services,
      features,
      examples
    }
  }
}
</script>

<style scoped>
.home-page {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 1rem;
}

/* Hero Section */
.hero-section {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4rem;
  align-items: center;
  min-height: 80vh;
  padding: 4rem 0;
}

.hero-content {
  max-width: 500px;
}

.hero-title {
  font-size: 3.5rem;
  font-weight: bold;
  margin-bottom: 1rem;
  line-height: 1.1;
}

.gradient-text {
  background: linear-gradient(135deg, var(--swit-primary), var(--swit-secondary));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.hero-subtitle {
  display: block;
  font-size: 2rem;
  color: var(--vp-c-text-2);
  margin-top: 0.5rem;
}

.hero-description {
  font-size: 1.25rem;
  color: var(--vp-c-text-2);
  margin-bottom: 2rem;
  line-height: 1.6;
}

.hero-actions {
  display: flex;
  gap: 1rem;
}

.btn-primary, .btn-secondary {
  padding: 0.75rem 1.5rem;
  border-radius: var(--swit-radius-lg);
  text-decoration: none;
  font-weight: 600;
  transition: all 0.3s ease;
}

.btn-primary {
  background: var(--swit-primary);
  color: white;
}

.btn-primary:hover {
  background: var(--swit-secondary);
  transform: translateY(-2px);
  box-shadow: var(--swit-shadow-lg);
}

.btn-secondary {
  background: transparent;
  color: var(--swit-primary);
  border: 2px solid var(--swit-primary);
}

.btn-secondary:hover {
  background: var(--swit-primary);
  color: white;
}

/* Code Preview */
.hero-image {
  display: flex;
  justify-content: center;
}

.code-preview {
  background: var(--vp-c-bg-alt);
  border-radius: var(--swit-radius-lg);
  overflow: hidden;
  box-shadow: var(--swit-shadow-lg);
  max-width: 400px;
  width: 100%;
}

.code-header {
  background: var(--vp-c-bg-soft);
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--vp-c-border);
  font-size: 0.875rem;
  font-weight: 600;
}

.code-content {
  padding: 1rem;
  margin: 0;
  font-size: 0.875rem;
  line-height: 1.5;
  overflow-x: auto;
}

/* Sections */
.stats-section,
.services-section,
.features-section,
.examples-section {
  padding: 4rem 0;
}

.section-title {
  text-align: center;
  font-size: 2.5rem;
  font-weight: bold;
  margin-bottom: 3rem;
  color: var(--vp-c-text-1);
}

/* Features Grid */
.features-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 2rem;
}

/* Examples Grid */
.examples-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(500px, 1fr));
  gap: 2rem;
}

/* Status Badge */
.status-badge {
  padding: 0.25rem 0.5rem;
  border-radius: var(--swit-radius-sm);
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
}

.status-badge.healthy {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

/* Responsive Design */
@media (max-width: 768px) {
  .hero-section {
    grid-template-columns: 1fr;
    gap: 2rem;
    text-align: center;
  }
  
  .hero-title {
    font-size: 2.5rem;
  }
  
  .hero-subtitle {
    font-size: 1.5rem;
  }
  
  .features-grid {
    grid-template-columns: 1fr;
  }
  
  .examples-grid {
    grid-template-columns: 1fr;
  }
  
  .examples-grid > * {
    min-width: unset;
  }
}
</style>