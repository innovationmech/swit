<template>
  <div class="home-page">
    <!-- Hero Section -->
    <section class="home-hero">
      <div class="container mx-auto px-4">
        <h1 class="home-hero-title gradient-text">
          {{ $t?.('hero.title') || heroTitle }}
        </h1>
        <p class="home-hero-description">
          {{ $t?.('hero.description') || heroDescription }}
        </p>
        <div class="home-hero-actions">
          <a :href="quickStartLink" class="btn-primary">
            {{ $t?.('hero.getStarted') || 'Get Started' }}
          </a>
          <a :href="apiDocsLink" class="btn-secondary">
            {{ $t?.('hero.viewDocs') || 'View Documentation' }}
          </a>
        </div>
        <div class="hero-code mt-12">
          <CodeExample 
            :code="heroCode" 
            language="go"
            :title="$t?.('hero.codeTitle') || 'Quick Example'"
          />
        </div>
      </div>
    </section>

    <!-- Features Section -->
    <section class="features py-16 px-4">
      <div class="container mx-auto">
        <h2 class="text-3xl font-bold text-center mb-12">
          {{ $t?.('features.title') || 'Core Features' }}
        </h2>
        <div class="features-grid">
          <FeatureCard 
            v-for="feature in features" 
            :key="feature.id"
            :feature="feature" 
          />
        </div>
      </div>
    </section>

    <!-- Stats Section -->
    <section class="stats py-16 px-4">
      <div class="container mx-auto">
        <div class="stats-grid">
          <div class="stat-item animate-fade-in">
            <span class="stat-number">{{ githubStats.stars }}</span>
            <span class="stat-label">GitHub Stars</span>
          </div>
          <div class="stat-item animate-fade-in" style="animation-delay: 0.1s">
            <span class="stat-number">{{ githubStats.version }}</span>
            <span class="stat-label">Latest Version</span>
          </div>
          <div class="stat-item animate-fade-in" style="animation-delay: 0.2s">
            <span class="stat-number">{{ githubStats.contributors }}</span>
            <span class="stat-label">Contributors</span>
          </div>
          <div class="stat-item animate-fade-in" style="animation-delay: 0.3s">
            <span class="stat-number">{{ githubStats.license }}</span>
            <span class="stat-label">License</span>
          </div>
        </div>
      </div>
    </section>

    <!-- Quick Links Section -->
    <section class="quick-links py-16 px-4">
      <div class="container mx-auto">
        <h2 class="text-3xl font-bold text-center mb-12">
          {{ $t?.('quickLinks.title') || 'Quick Links' }}
        </h2>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
          <a href="/zh/guide/getting-started" class="card hover:border-swit-primary">
            <h3 class="text-xl font-semibold mb-2">{{ $t?.('quickLinks.guide') || 'Getting Started Guide' }}</h3>
            <p class="text-gray-600 dark:text-gray-300">
              {{ $t?.('quickLinks.guideDesc') || 'Learn how to quickly set up and use the Swit framework' }}
            </p>
          </a>
          <a href="/zh/api/overview" class="card hover:border-swit-primary">
            <h3 class="text-xl font-semibold mb-2">{{ $t?.('quickLinks.api') || 'API Reference' }}</h3>
            <p class="text-gray-600 dark:text-gray-300">
              {{ $t?.('quickLinks.apiDesc') || 'Complete API documentation for all framework components' }}
            </p>
          </a>
          <a href="/zh/examples/quick-start" class="card hover:border-swit-primary">
            <h3 class="text-xl font-semibold mb-2">{{ $t?.('quickLinks.examples') || 'Examples' }}</h3>
            <p class="text-gray-600 dark:text-gray-300">
              {{ $t?.('quickLinks.examplesDesc') || 'Practical examples to help you understand the framework' }}
            </p>
          </a>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useData } from 'vitepress'
import FeatureCard from './FeatureCard.vue'
import CodeExample from './CodeExample.vue'

const { lang } = useData()

// Computed properties for language-specific content
const isZh = computed(() => lang.value === 'zh-CN')

const heroTitle = computed(() => 
  isZh.value ? 'Swit - Go 微服务框架' : 'Swit - Go Microservice Framework'
)

const heroDescription = computed(() => 
  isZh.value 
    ? '一个全面的 Go 微服务框架，提供生产就绪的组件用于构建可扩展的微服务'
    : 'A comprehensive Go microservice framework providing production-ready components for building scalable microservices'
)

const quickStartLink = computed(() => 
  isZh.value ? '/zh/guide/getting-started' : '/en/guide/getting-started'
)

const apiDocsLink = computed(() => 
  isZh.value ? '/zh/api/overview' : '/en/api/overview'
)

// Hero code example
const heroCode = ref(`package main

import (
    "github.com/innovationmech/swit/pkg/server"
    "github.com/innovationmech/swit/pkg/transport"
)

func main() {
    // Create server configuration
    config := &server.ServerConfig{
        Name:    "my-service",
        Version: "1.0.0",
        HTTP:    server.HTTPConfig{Port: 8080},
        GRPC:    server.GRPCConfig{Port: 9090},
    }
    
    // Initialize service
    service := NewMyService()
    
    // Create and start server
    srv, _ := server.NewBusinessServerCore(config, service, nil)
    srv.Start(context.Background())
}`)

// Features data
const features = computed(() => [
  {
    id: 'microservice',
    icon: '🚀',
    titleKey: isZh.value ? '微服务架构' : 'Microservice Architecture',
    descriptionKey: isZh.value 
      ? '完整的微服务框架，支持 HTTP 和 gRPC 双协议'
      : 'Complete microservice framework with HTTP and gRPC dual-protocol support'
  },
  {
    id: 'performance',
    icon: '⚡',
    titleKey: isZh.value ? '高性能' : 'High Performance',
    descriptionKey: isZh.value 
      ? '内置性能监控和优化，确保服务高效运行'
      : 'Built-in performance monitoring and optimization for efficient service operation'
  },
  {
    id: 'discovery',
    icon: '🔍',
    titleKey: isZh.value ? '服务发现' : 'Service Discovery',
    descriptionKey: isZh.value 
      ? '集成 Consul 服务发现，支持自动注册和健康检查'
      : 'Integrated Consul service discovery with automatic registration and health checks'
  },
  {
    id: 'middleware',
    icon: '🛡️',
    titleKey: isZh.value ? '中间件支持' : 'Middleware Support',
    descriptionKey: isZh.value 
      ? '丰富的中间件生态，包括认证、限流、追踪等'
      : 'Rich middleware ecosystem including authentication, rate limiting, tracing, and more'
  },
  {
    id: 'di',
    icon: '📦',
    titleKey: isZh.value ? '依赖注入' : 'Dependency Injection',
    descriptionKey: isZh.value 
      ? '强大的依赖注入容器，简化服务组件管理'
      : 'Powerful dependency injection container for simplified service component management'
  },
  {
    id: 'config',
    icon: '⚙️',
    titleKey: isZh.value ? '配置管理' : 'Configuration Management',
    descriptionKey: isZh.value 
      ? '灵活的配置系统，支持环境变量和 YAML 配置'
      : 'Flexible configuration system with environment variables and YAML support'
  }
])

// GitHub stats (will be fetched from API in production)
const githubStats = ref({
  stars: '1.2k',
  version: 'v1.0.0',
  contributors: '25+',
  license: 'MIT'
})

// Fetch GitHub stats on mount
onMounted(async () => {
  try {
    // In production, fetch from GitHub API
    // const response = await fetch('https://api.github.com/repos/innovationmech/swit')
    // const data = await response.json()
    // githubStats.value = {
    //   stars: formatNumber(data.stargazers_count),
    //   version: data.tag_name || 'v1.0.0',
    //   contributors: '25+',
    //   license: data.license?.spdx_id || 'MIT'
    // }
  } catch (error) {
    console.error('Failed to fetch GitHub stats:', error)
  }
})

function formatNumber(num: number): string {
  if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'k'
  }
  return num.toString()
}
</script>

<style scoped>
.home-page {
  padding-top: var(--vp-nav-height);
}

.container {
  max-width: 1280px;
}

.quick-links .card {
  transition: all 0.3s ease;
  text-decoration: none;
  display: block;
}

.quick-links .card:hover {
  transform: translateY(-2px);
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.1);
}
</style>