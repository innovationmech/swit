<template>
  <div class="home-page">
    <!-- Hero Section -->
    <section class="hero">
      <div class="hero-content">
        <h1 class="hero-title">{{ heroData.name }}</h1>
        <p class="hero-tagline">{{ heroData.tagline }}</p>
        <div class="hero-actions">
          <a 
            v-for="action in heroData.actions" 
            :key="action.text"
            :href="action.link"
            :class="['hero-button', `hero-button--${action.theme}`]"
          >
            {{ action.text }}
          </a>
        </div>
      </div>
      <div v-if="heroCode" class="hero-code">
        <CodeExample 
          :code="heroCode" 
          language="go" 
          title="Quick Start Example"
          :copy="true"
        />
      </div>
    </section>

    <!-- Features Section -->
    <section v-if="features.length" class="features">
      <h2 class="features-title">{{ $t('features.title', 'Key Features') }}</h2>
      <div class="features-grid">
        <FeatureCard 
          v-for="feature in features" 
          :key="feature.title"
          :feature="feature" 
        />
      </div>
    </section>

    <!-- Stats Section -->
    <section v-if="showStats" class="stats">
      <h2 class="stats-title">{{ $t('stats.title', 'Project Statistics') }}</h2>
      <div class="stats-grid">
        <div class="stat-item">
          <div class="stat-number">{{ stats.stars || '⭐' }}</div>
          <div class="stat-label">{{ $t('stats.stars', 'GitHub Stars') }}</div>
        </div>
        <div class="stat-item">
          <div class="stat-number">{{ stats.version || 'v1.0.0' }}</div>
          <div class="stat-label">{{ $t('stats.version', 'Latest Version') }}</div>
        </div>
        <div class="stat-item">
          <div class="stat-number">{{ stats.license || 'MIT' }}</div>
          <div class="stat-label">{{ $t('stats.license', 'License') }}</div>
        </div>
        <div class="stat-item">
          <div class="stat-number">{{ stats.goVersion || 'Go 1.24+' }}</div>
          <div class="stat-label">{{ $t('stats.goVersion', 'Go Version') }}</div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useData } from 'vitepress'
import CodeExample from './CodeExample.vue'
import FeatureCard from './FeatureCard.vue'

const { frontmatter } = useData()

const heroData = computed(() => frontmatter.value.hero || {})
const features = computed(() => frontmatter.value.features || [])
const showStats = computed(() => frontmatter.value.showStats !== false)

const stats = computed(() => ({
  stars: frontmatter.value.stats?.stars,
  version: frontmatter.value.stats?.version,
  license: frontmatter.value.stats?.license,
  goVersion: frontmatter.value.stats?.goVersion
}))

const heroCode = computed(() => {
  return `package main

import (
    "context"
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/innovationmech/swit/pkg/server"
)

type MyService struct{}

func (s *MyService) RegisterServices(registry server.BusinessServiceRegistry) error {
    return registry.RegisterBusinessHTTPHandler(&MyHTTPHandler{})
}

type MyHTTPHandler struct{}

func (h *MyHTTPHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(*gin.Engine)
    ginRouter.GET("/hello", h.handleHello)
    return nil
}

func (h *MyHTTPHandler) GetServiceName() string {
    return "my-service"
}

func (h *MyHTTPHandler) handleHello(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"message": "Hello from Swit!"})
}

func main() {
    config := &server.ServerConfig{
        ServiceName: "my-service",
        HTTP: server.HTTPConfig{Port: "8080", Enabled: true},
    }
    
    service := &MyService{}
    baseServer, _ := server.NewBusinessServerCore(config, service, nil)
    
    ctx := context.Background()
    baseServer.Start(ctx)
    defer baseServer.Shutdown()
    
    select {} // Keep running
}`
})
</script>

<style scoped>
.home-page {
  padding-top: 0;
}

.hero {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4rem;
  align-items: center;
  padding: 4rem 2rem;
  max-width: 1200px;
  margin: 0 auto;
}

.hero-content {
  text-align: left;
}

.hero-title {
  font-size: 3.5rem;
  font-weight: 700;
  line-height: 1.1;
  margin-bottom: 1.5rem;
  background: linear-gradient(135deg, var(--vp-c-brand-1), var(--vp-c-brand-2));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.hero-tagline {
  font-size: 1.5rem;
  line-height: 1.4;
  color: var(--vp-c-text-2);
  margin-bottom: 2rem;
}

.hero-actions {
  display: flex;
  gap: 1rem;
  flex-wrap: wrap;
}

.hero-button {
  display: inline-flex;
  align-items: center;
  padding: 0.75rem 1.5rem;
  border-radius: 8px;
  text-decoration: none;
  font-weight: 600;
  transition: all 0.3s ease;
}

.hero-button--brand {
  background: var(--vp-c-brand-1);
  color: white;
}

.hero-button--brand:hover {
  background: var(--vp-c-brand-2);
  transform: translateY(-2px);
}

.hero-button--alt {
  background: var(--vp-c-bg-soft);
  color: var(--vp-c-text-1);
  border: 1px solid var(--vp-c-divider);
}

.hero-button--alt:hover {
  background: var(--vp-c-bg-elv);
  transform: translateY(-2px);
}

.hero-code {
  max-height: 500px;
  overflow-y: auto;
}

.features {
  padding: 4rem 2rem;
  max-width: 1200px;
  margin: 0 auto;
  background: var(--vp-c-bg-alt);
}

.features-title {
  text-align: center;
  font-size: 2.5rem;
  font-weight: 700;
  margin-bottom: 3rem;
  color: var(--vp-c-text-1);
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
  gap: 2rem;
}

.stats {
  padding: 4rem 2rem;
  max-width: 1200px;
  margin: 0 auto;
}

.stats-title {
  text-align: center;
  font-size: 2rem;
  font-weight: 700;
  margin-bottom: 2rem;
  color: var(--vp-c-text-1);
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 2rem;
  max-width: 800px;
  margin: 0 auto;
}

.stat-item {
  text-align: center;
  padding: 2rem;
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  background: var(--vp-c-bg-soft);
  transition: all 0.3s ease;
}

.stat-item:hover {
  border-color: var(--vp-c-brand-1);
  transform: translateY(-2px);
}

.stat-number {
  font-size: 2.5rem;
  font-weight: 700;
  color: var(--vp-c-brand-1);
  margin-bottom: 0.5rem;
}

.stat-label {
  color: var(--vp-c-text-2);
  font-weight: 500;
}

@media (max-width: 768px) {
  .hero {
    grid-template-columns: 1fr;
    gap: 2rem;
    padding: 2rem 1rem;
    text-align: center;
  }
  
  .hero-title {
    font-size: 2.5rem;
  }
  
  .hero-tagline {
    font-size: 1.25rem;
  }
  
  .features,
  .stats {
    padding: 2rem 1rem;
  }
  
  .features-title,
  .stats-title {
    font-size: 2rem;
  }
  
  .features-grid {
    grid-template-columns: 1fr;
    gap: 1.5rem;
  }
  
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>