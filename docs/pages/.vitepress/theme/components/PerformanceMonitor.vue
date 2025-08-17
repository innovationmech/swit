<template>
  <div v-if="showMonitor" class="performance-monitor" :class="{ 'monitor-minimized': isMinimized }">
    <!-- 性能监控面板 -->
    <div class="monitor-header" @click="toggleMinimize">
      <div class="monitor-title">
        <Icon name="activity" class="monitor-icon" />
        <span>性能监控</span>
      </div>
      <div class="monitor-controls">
        <button 
          class="monitor-btn" 
          @click.stop="refreshMetrics"
          :disabled="isLoading"
          title="刷新指标"
        >
          <Icon name="refresh" :class="{ 'spinning': isLoading }" />
        </button>
        <button 
          class="monitor-btn" 
          @click.stop="isMinimized = !isMinimized"
          :title="isMinimized ? '展开' : '收起'"
        >
          <Icon :name="isMinimized ? 'chevron-up' : 'chevron-down'" />
        </button>
        <button 
          class="monitor-btn close-btn" 
          @click.stop="hideMonitor"
          title="关闭"
        >
          <Icon name="x" />
        </button>
      </div>
    </div>

    <!-- 监控内容 -->
    <div v-if="!isMinimized" class="monitor-content">
      <!-- Core Web Vitals -->
      <div class="metrics-section">
        <h4>Core Web Vitals</h4>
        <div class="metrics-grid">
          <div 
            v-for="vital in coreWebVitals" 
            :key="vital.name"
            class="metric-card"
            :class="getMetricStatus(vital)"
          >
            <div class="metric-label">{{ vital.label }}</div>
            <div class="metric-value">
              {{ formatMetricValue(vital.value, vital.unit) }}
            </div>
            <div class="metric-threshold">
              目标: {{ formatMetricValue(vital.threshold || 0, vital.unit) }}
            </div>
          </div>
        </div>
      </div>

      <!-- 页面性能指标 -->
      <div class="metrics-section">
        <h4>页面性能</h4>
        <div class="metrics-grid">
          <div 
            v-for="metric in performanceMetrics" 
            :key="metric.name"
            class="metric-card"
          >
            <div class="metric-label">{{ metric.label }}</div>
            <div class="metric-value">
              {{ formatMetricValue(metric.value, metric.unit) }}
            </div>
          </div>
        </div>
      </div>

      <!-- 资源性能 -->
      <div class="metrics-section">
        <h4>资源加载</h4>
        <div class="resource-list">
          <div 
            v-for="resource in resourceMetrics" 
            :key="resource.name"
            class="resource-item"
          >
            <div class="resource-name">{{ resource.name }}</div>
            <div class="resource-stats">
              <span class="resource-size">{{ formatBytes(resource.size) }}</span>
              <span class="resource-time">{{ resource.loadTime }}ms</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 性能历史图表 -->
      <div class="metrics-section">
        <h4>性能趋势</h4>
        <div class="chart-container">
          <canvas ref="performanceChart" class="performance-chart"></canvas>
        </div>
      </div>

      <!-- 性能建议 -->
      <div v-if="recommendations.length > 0" class="metrics-section">
        <h4>优化建议</h4>
        <div class="recommendations">
          <div 
            v-for="(rec, index) in recommendations" 
            :key="index"
            class="recommendation"
            :class="rec.priority"
          >
            <Icon :name="getRecommendationIcon(rec.priority)" />
            <span>{{ rec.message }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 性能状态指示器（最小化时显示） -->
    <div v-if="isMinimized" class="monitor-indicator">
      <div class="indicator-dots">
        <div 
          v-for="vital in coreWebVitals.slice(0, 3)" 
          :key="vital.name"
          class="indicator-dot"
          :class="getMetricStatus(vital)"
          :title="vital.label"
        ></div>
      </div>
    </div>
  </div>

  <!-- 性能提醒通知 -->
  <Transition name="notification">
    <div v-if="notification" class="performance-notification" :class="notification.type">
      <Icon :name="notification.icon" />
      <span>{{ notification.message }}</span>
      <button @click="dismissNotification" class="notification-close">
        <Icon name="x" />
      </button>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, computed, nextTick } from 'vue'
import { useData } from 'vitepress'

interface PerformanceMetric {
  name: string
  label: string
  value: number
  unit: string
  threshold?: number
}

interface ResourceMetric {
  name: string
  size: number
  loadTime: number
}

interface Recommendation {
  priority: 'high' | 'medium' | 'low'
  message: string
}

interface Notification {
  type: 'warning' | 'error' | 'info'
  icon: string
  message: string
}

// 组件状态
const showMonitor = ref(false)
const isMinimized = ref(false)
const isLoading = ref(false)
const performanceChart = ref<HTMLCanvasElement>()
const notification = ref<Notification | null>(null)

// 性能数据
const coreWebVitals = reactive<PerformanceMetric[]>([
  { name: 'lcp', label: 'LCP', value: 0, unit: 'ms', threshold: 2500 },
  { name: 'fid', label: 'FID', value: 0, unit: 'ms', threshold: 100 },
  { name: 'cls', label: 'CLS', value: 0, unit: '', threshold: 0.1 },
  { name: 'fcp', label: 'FCP', value: 0, unit: 'ms', threshold: 1800 }
])

const performanceMetrics = reactive<PerformanceMetric[]>([
  { name: 'ttfb', label: 'TTFB', value: 0, unit: 'ms' },
  { name: 'domContentLoaded', label: 'DOM Content Loaded', value: 0, unit: 'ms' },
  { name: 'loadComplete', label: 'Load Complete', value: 0, unit: 'ms' },
  { name: 'memoryUsage', label: 'Memory Usage', value: 0, unit: 'MB' }
])

const resourceMetrics = reactive<ResourceMetric[]>([])
const recommendations = reactive<Recommendation[]>([])

// 性能历史数据
const performanceHistory = reactive({
  timestamps: [] as string[],
  lcp: [] as number[],
  fcp: [] as number[],
  cls: [] as number[]
})

// VitePress 数据
const { isDark } = useData()

// 计算属性
const shouldShowMonitor = computed(() => {
  // 在开发环境或特定条件下显示
  return (import.meta as any).env?.DEV || 
         localStorage.getItem('vitepress-performance-monitor') === 'true'
})

// 生命周期
onMounted(async () => {
  if (shouldShowMonitor.value) {
    showMonitor.value = true
    await initializeMonitor()
    startPerformanceMonitoring()
  }
})

onUnmounted(() => {
  stopPerformanceMonitoring()
})

// 监控方法
async function initializeMonitor() {
  try {
    // SSR protection
    if (typeof window === 'undefined') return
    
    // 检查 Performance API 支持
    if (!window.performance) {
      console.warn('Performance API not supported')
      return
    }

    // 初始化性能数据收集
    await collectInitialMetrics()
    
    // 设置性能观察器
    setupPerformanceObservers()
    
    // 初始化图表
    await nextTick()
    initializeChart()
    
  } catch (error) {
    console.error('Failed to initialize performance monitor:', error)
  }
}

async function collectInitialMetrics() {
  isLoading.value = true
  
  try {
    // 收集导航性能数据
    const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming
    if (navigation) {
      updateMetric('ttfb', navigation.responseStart - navigation.requestStart)
      updateMetric('domContentLoaded', navigation.domContentLoadedEventEnd - navigation.domContentLoadedEventStart)
      updateMetric('loadComplete', navigation.loadEventEnd - navigation.loadEventStart)
    }

    // 收集 Paint 性能数据
    const paintEntries = performance.getEntriesByType('paint')
    paintEntries.forEach(entry => {
      if (entry.name === 'first-contentful-paint') {
        updateCoreWebVital('fcp', entry.startTime)
      }
    })

    // 收集内存使用情况
    if ('memory' in performance) {
      const memory = (performance as any).memory
      updateMetric('memoryUsage', memory.usedJSHeapSize / 1024 / 1024)
    }

    // 收集资源性能数据
    collectResourceMetrics()
    
    // 生成性能建议
    generateRecommendations()
    
  } catch (error) {
    console.error('Failed to collect metrics:', error)
  } finally {
    isLoading.value = false
  }
}

function setupPerformanceObservers() {
  // Web Vitals 观察器
  if ('PerformanceObserver' in window) {
    try {
      // LCP 观察器
      const lcpObserver = new PerformanceObserver((list) => {
        const entries = list.getEntries()
        const lastEntry = entries[entries.length - 1]
        updateCoreWebVital('lcp', lastEntry.startTime)
      })
      lcpObserver.observe({ type: 'largest-contentful-paint', buffered: true })

      // FID 观察器
      const fidObserver = new PerformanceObserver((list) => {
        const entries = list.getEntries()
        entries.forEach(entry => {
          updateCoreWebVital('fid', (entry as any).processingStart - entry.startTime)
        })
      })
      fidObserver.observe({ type: 'first-input', buffered: true })

      // CLS 观察器
      let clsValue = 0
      const clsObserver = new PerformanceObserver((list) => {
        const entries = list.getEntries()
        entries.forEach(entry => {
          if (!(entry as any).hadRecentInput) {
            clsValue += (entry as any).value
            updateCoreWebVital('cls', clsValue)
          }
        })
      })
      clsObserver.observe({ type: 'layout-shift', buffered: true })

    } catch (error) {
      console.warn('Performance observers not fully supported:', error)
    }
  }
}

function collectResourceMetrics() {
  const resources = performance.getEntriesByType('resource')
  const resourceData: ResourceMetric[] = []
  
  resources.forEach(resource => {
    const entry = resource as PerformanceResourceTiming
    if (entry.transferSize > 0) {
      resourceData.push({
        name: entry.name.split('/').pop() || 'Unknown',
        size: entry.transferSize,
        loadTime: entry.responseEnd - entry.requestStart
      })
    }
  })
  
  // 按大小排序，取前 10 个
  resourceMetrics.splice(0, resourceMetrics.length, 
    ...resourceData
      .sort((a, b) => b.size - a.size)
      .slice(0, 10)
  )
}

function updateMetric(name: string, value: number) {
  const metric = performanceMetrics.find(m => m.name === name)
  if (metric) {
    metric.value = Math.round(value * 100) / 100
  }
}

function updateCoreWebVital(name: string, value: number) {
  const vital = coreWebVitals.find(v => v.name === name)
  if (vital) {
    vital.value = Math.round(value * 100) / 100
    
    // 更新历史数据
    updatePerformanceHistory(name, value)
    
    // 检查阈值并发送通知
    checkThresholds(vital)
  }
}

function updatePerformanceHistory(metricName: string, value: number) {
  const now = new Date().toLocaleTimeString()
  
  if (performanceHistory.timestamps.length >= 20) {
    performanceHistory.timestamps.shift()
    performanceHistory.lcp.shift()
    performanceHistory.fcp.shift()
    performanceHistory.cls.shift()
  }
  
  performanceHistory.timestamps.push(now)
  
  switch (metricName) {
    case 'lcp':
      performanceHistory.lcp.push(value)
      break
    case 'fcp':
      performanceHistory.fcp.push(value)
      break
    case 'cls':
      performanceHistory.cls.push(value * 1000) // 放大显示
      break
  }
  
  // 更新图表
  updateChart()
}

function generateRecommendations() {
  recommendations.splice(0)
  
  // 检查各项指标并生成建议
  const lcp = coreWebVitals.find(v => v.name === 'lcp')
  if (lcp && lcp.value > lcp.threshold!) {
    recommendations.push({
      priority: 'high',
      message: `LCP 过慢 (${lcp.value}ms)，考虑优化首屏内容加载`
    })
  }
  
  const cls = coreWebVitals.find(v => v.name === 'cls')
  if (cls && cls.value > cls.threshold!) {
    recommendations.push({
      priority: 'medium',
      message: `布局偏移过多 (${cls.value})，检查动态内容加载`
    })
  }
  
  const memory = performanceMetrics.find(m => m.name === 'memoryUsage')
  if (memory && memory.value > 50) {
    recommendations.push({
      priority: 'low',
      message: `内存使用较高 (${memory.value}MB)，考虑优化内存管理`
    })
  }
  
  // 检查大资源文件
  const largeResources = resourceMetrics.filter(r => r.size > 500 * 1024)
  if (largeResources.length > 0) {
    recommendations.push({
      priority: 'medium',
      message: `发现 ${largeResources.length} 个大文件，考虑压缩或延迟加载`
    })
  }
}

function checkThresholds(vital: PerformanceMetric) {
  if (vital.threshold && vital.value > vital.threshold) {
    showNotification({
      type: 'warning',
      icon: 'alert-triangle',
      message: `${vital.label} 超过阈值: ${formatMetricValue(vital.value, vital.unit)}`
    })
  }
}

// 图表相关
function initializeChart() {
  if (!performanceChart.value) return
  
  const ctx = performanceChart.value.getContext('2d')
  if (!ctx) return
  
  // 简单的图表实现
  drawChart(ctx)
}

function updateChart() {
  if (!performanceChart.value) return
  
  const ctx = performanceChart.value.getContext('2d')
  if (!ctx) return
  
  drawChart(ctx)
}

function drawChart(ctx: CanvasRenderingContext2D) {
  const canvas = ctx.canvas
  const width = canvas.width = canvas.offsetWidth
  const height = canvas.height = canvas.offsetHeight
  
  // 清空画布
  ctx.clearRect(0, 0, width, height)
  
  if (performanceHistory.timestamps.length === 0) return
  
  // 绘制网格
  ctx.strokeStyle = isDark.value ? '#333' : '#eee'
  ctx.lineWidth = 1
  
  // 垂直网格线
  for (let i = 0; i <= 10; i++) {
    const x = (width / 10) * i
    ctx.beginPath()
    ctx.moveTo(x, 0)
    ctx.lineTo(x, height)
    ctx.stroke()
  }
  
  // 水平网格线
  for (let i = 0; i <= 5; i++) {
    const y = (height / 5) * i
    ctx.beginPath()
    ctx.moveTo(0, y)
    ctx.lineTo(width, y)
    ctx.stroke()
  }
  
  // 绘制数据线
  if (performanceHistory.lcp.length > 1) {
    drawLine(ctx, performanceHistory.lcp, '#ff6b6b', width, height)
  }
  
  if (performanceHistory.fcp.length > 1) {
    drawLine(ctx, performanceHistory.fcp, '#4ecdc4', width, height)
  }
  
  if (performanceHistory.cls.length > 1) {
    drawLine(ctx, performanceHistory.cls, '#45b7d1', width, height)
  }
}

function drawLine(ctx: CanvasRenderingContext2D, data: number[], color: string, width: number, height: number) {
  const maxValue = Math.max(...data, 1)
  const step = width / (data.length - 1)
  
  ctx.strokeStyle = color
  ctx.lineWidth = 2
  ctx.beginPath()
  
  data.forEach((value, index) => {
    const x = index * step
    const y = height - (value / maxValue) * height
    
    if (index === 0) {
      ctx.moveTo(x, y)
    } else {
      ctx.lineTo(x, y)
    }
  })
  
  ctx.stroke()
}

// 监控控制
let monitoringInterval: number | null = null

function startPerformanceMonitoring() {
  if (monitoringInterval || typeof window === 'undefined') return
  
  monitoringInterval = window.setInterval(() => {
    collectInitialMetrics()
  }, 5000) // 每 5 秒更新一次
}

function stopPerformanceMonitoring() {
  if (monitoringInterval) {
    clearInterval(monitoringInterval)
    monitoringInterval = null
  }
}

async function refreshMetrics() {
  await collectInitialMetrics()
}

function toggleMinimize() {
  isMinimized.value = !isMinimized.value
}

function hideMonitor() {
  showMonitor.value = false
  localStorage.setItem('vitepress-performance-monitor', 'false')
}

function showNotification(notif: Notification) {
  notification.value = notif
  setTimeout(() => {
    notification.value = null
  }, 5000)
}

function dismissNotification() {
  notification.value = null
}

// 工具函数
function getMetricStatus(metric: PerformanceMetric): string {
  if (!metric.threshold) return 'unknown'
  
  if (metric.value <= metric.threshold * 0.8) return 'good'
  if (metric.value <= metric.threshold) return 'needs-improvement'
  return 'poor'
}

function formatMetricValue(value: number, unit: string): string {
  if (unit === 'ms') {
    return `${Math.round(value)}ms`
  } else if (unit === 'MB') {
    return `${value.toFixed(1)}MB`
  } else if (unit === '') {
    return value.toFixed(3)
  }
  return `${value}${unit}`
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function getRecommendationIcon(priority: string): string {
  switch (priority) {
    case 'high': return 'alert-circle'
    case 'medium': return 'alert-triangle'
    case 'low': return 'info'
    default: return 'help-circle'
  }
}

// 暴露给开发者控制台
if ((import.meta as any).env?.DEV) {
  ;(window as any).performanceMonitor = {
    show: () => { showMonitor.value = true },
    hide: () => { showMonitor.value = false },
    refresh: refreshMetrics,
    data: { coreWebVitals, performanceMetrics, resourceMetrics }
  }
}
</script>

<style scoped>
.performance-monitor {
  position: fixed;
  top: 80px;
  right: 20px;
  width: 320px;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.1);
  z-index: 1000;
  font-size: 14px;
  max-height: calc(100vh - 100px);
  overflow: hidden;
  transition: all 0.3s ease;
}

.monitor-minimized {
  width: 200px;
  height: auto;
}

.monitor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: var(--vp-c-bg-soft);
  border-bottom: 1px solid var(--vp-c-divider);
  cursor: pointer;
  user-select: none;
}

.monitor-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.monitor-icon {
  width: 16px;
  height: 16px;
  color: var(--vp-c-brand);
}

.monitor-controls {
  display: flex;
  gap: 4px;
}

.monitor-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: var(--vp-c-text-2);
  cursor: pointer;
  transition: all 0.2s ease;
}

.monitor-btn:hover {
  background: var(--vp-c-gray-soft);
  color: var(--vp-c-text-1);
}

.monitor-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.close-btn:hover {
  background: var(--vp-c-red-soft);
  color: var(--vp-c-red);
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.monitor-content {
  max-height: calc(100vh - 200px);
  overflow-y: auto;
  padding: 16px;
}

.metrics-section {
  margin-bottom: 20px;
}

.metrics-section h4 {
  margin: 0 0 12px 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--vp-c-text-1);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
}

.metric-card {
  padding: 12px;
  background: var(--vp-c-bg-soft);
  border-radius: 6px;
  border-left: 3px solid var(--vp-c-divider);
  transition: all 0.2s ease;
}

.metric-card.good {
  border-left-color: var(--vp-c-green);
  background: var(--vp-c-green-soft);
}

.metric-card.needs-improvement {
  border-left-color: var(--vp-c-yellow);
  background: var(--vp-c-yellow-soft);
}

.metric-card.poor {
  border-left-color: var(--vp-c-red);
  background: var(--vp-c-red-soft);
}

.metric-label {
  font-size: 11px;
  color: var(--vp-c-text-2);
  margin-bottom: 4px;
  font-weight: 500;
}

.metric-value {
  font-size: 16px;
  font-weight: 700;
  color: var(--vp-c-text-1);
  margin-bottom: 2px;
}

.metric-threshold {
  font-size: 10px;
  color: var(--vp-c-text-3);
}

.resource-list {
  max-height: 120px;
  overflow-y: auto;
}

.resource-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px;
  background: var(--vp-c-bg-soft);
  border-radius: 4px;
  margin-bottom: 4px;
}

.resource-name {
  font-size: 12px;
  color: var(--vp-c-text-1);
  font-weight: 500;
  max-width: 150px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.resource-stats {
  display: flex;
  gap: 8px;
  font-size: 11px;
  color: var(--vp-c-text-2);
}

.chart-container {
  height: 120px;
  background: var(--vp-c-bg-soft);
  border-radius: 6px;
  overflow: hidden;
}

.performance-chart {
  width: 100%;
  height: 100%;
}

.recommendations {
  max-height: 100px;
  overflow-y: auto;
}

.recommendation {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  background: var(--vp-c-bg-soft);
  border-radius: 4px;
  margin-bottom: 4px;
  font-size: 12px;
}

.recommendation.high {
  background: var(--vp-c-red-soft);
  color: var(--vp-c-red-dark);
}

.recommendation.medium {
  background: var(--vp-c-yellow-soft);
  color: var(--vp-c-yellow-dark);
}

.recommendation.low {
  background: var(--vp-c-blue-soft);
  color: var(--vp-c-blue-dark);
}

.monitor-indicator {
  padding: 8px 16px;
}

.indicator-dots {
  display: flex;
  gap: 6px;
  justify-content: center;
}

.indicator-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--vp-c-divider);
}

.indicator-dot.good {
  background: var(--vp-c-green);
}

.indicator-dot.needs-improvement {
  background: var(--vp-c-yellow);
}

.indicator-dot.poor {
  background: var(--vp-c-red);
}

.performance-notification {
  position: fixed;
  top: 20px;
  right: 20px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-divider);
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  z-index: 1001;
  max-width: 300px;
  font-size: 13px;
}

.performance-notification.warning {
  border-color: var(--vp-c-yellow);
  background: var(--vp-c-yellow-soft);
  color: var(--vp-c-yellow-dark);
}

.performance-notification.error {
  border-color: var(--vp-c-red);
  background: var(--vp-c-red-soft);
  color: var(--vp-c-red-dark);
}

.performance-notification.info {
  border-color: var(--vp-c-blue);
  background: var(--vp-c-blue-soft);
  color: var(--vp-c-blue-dark);
}

.notification-close {
  background: none;
  border: none;
  cursor: pointer;
  color: inherit;
  opacity: 0.7;
  margin-left: auto;
}

.notification-close:hover {
  opacity: 1;
}

.notification-enter-active,
.notification-leave-active {
  transition: all 0.3s ease;
}

.notification-enter-from,
.notification-leave-to {
  opacity: 0;
  transform: translateX(20px);
}

/* 滚动条样式 */
.monitor-content::-webkit-scrollbar,
.resource-list::-webkit-scrollbar,
.recommendations::-webkit-scrollbar {
  width: 4px;
}

.monitor-content::-webkit-scrollbar-thumb,
.resource-list::-webkit-scrollbar-thumb,
.recommendations::-webkit-scrollbar-thumb {
  background: var(--vp-c-divider);
  border-radius: 2px;
}

.monitor-content::-webkit-scrollbar-track,
.resource-list::-webkit-scrollbar-track,
.recommendations::-webkit-scrollbar-track {
  background: transparent;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .performance-monitor {
    width: calc(100vw - 40px);
    right: 20px;
    left: 20px;
  }
  
  .monitor-minimized {
    width: 150px;
    left: auto;
  }
  
  .performance-notification {
    right: 20px;
    left: 20px;
    max-width: none;
  }
}
</style>