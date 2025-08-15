# GitHub Pages 网站设计文档

## 概述

本设计文档描述了为 Swit Go 微服务框架创建 GitHub Pages 网站的技术架构和实现方案。网站将采用现代静态网站生成技术，支持中英文双语，提供完整的项目文档和 API 展示功能。

## 架构

### 技术栈选择

- **静态网站生成器**: VitePress 4.x
  - 基于 Vue 3 和 Vite，性能优秀
  - 内置多语言支持
  - 优秀的文档展示能力
  - 支持 Markdown 和 Vue 组件混合
  - 内置搜索功能

- **UI 框架**: 
  - VitePress 默认主题（自定义样式）
  - Tailwind CSS 用于自定义组件样式
  - 响应式设计，支持移动端

- **部署平台**: GitHub Pages
  - 与 GitHub 仓库深度集成
  - 支持自定义域名
  - 免费且稳定

- **CI/CD**: GitHub Actions
  - 自动构建和部署
  - 支持多环境部署
  - 集成测试和质量检查

### 项目结构

```
docs/
├── pages/                    # GitHub Pages 网站源码
│   ├── .vitepress/
│   │   ├── config/
│   │   │   ├── index.ts      # 主配置文件
│   │   │   ├── zh.ts         # 中文配置
│   │   │   └── en.ts         # 英文配置
│   │   ├── theme/
│   │   │   ├── index.ts      # 主题入口
│   │   │   ├── components/   # 自定义组件
│   │   │   │   ├── HomePage.vue  # 首页组件
│   │   │   │   ├── FeatureCard.vue
│   │   │   │   ├── CodeExample.vue
│   │   │   │   └── ApiDoc.vue
│   │   │   └── styles/
│   │   │       ├── vars.css  # CSS 变量
│   │   │       └── custom.css # 自定义样式
│   │   └── public/           # 静态资源
│   │       ├── images/
│   │       ├── icons/
│   │       └── api/          # API 文档 JSON
│   ├── zh/                   # 中文内容
│   │   ├── index.md         # 中文首页
│   │   ├── guide/           # 使用指南
│   │   ├── api/             # API 文档
│   │   ├── examples/        # 示例代码
│   │   └── community/       # 社区相关
│   ├── en/                   # 英文内容
│   │   ├── index.md         # 英文首页
│   │   ├── guide/
│   │   ├── api/
│   │   ├── examples/
│   │   └── community/
│   ├── package.json
│   ├── tailwind.config.js
│   └── vite.config.ts
├── architecture/             # 现有架构文档
├── services/                 # 现有服务文档
└── generated/                # 现有生成的文档
```

## 组件和接口

### 核心组件设计

#### 1. 首页组件 (HomePage.vue)

```vue
<template>
  <div class="home-page">
    <!-- Hero Section -->
    <section class="hero">
      <div class="hero-content">
        <h1 class="hero-title">{{ $t('hero.title') }}</h1>
        <p class="hero-description">{{ $t('hero.description') }}</p>
        <div class="hero-actions">
          <a href="/guide/getting-started" class="btn-primary">
            {{ $t('hero.getStarted') }}
          </a>
          <a href="/api/" class="btn-secondary">
            {{ $t('hero.viewDocs') }}
          </a>
        </div>
      </div>
      <div class="hero-code">
        <CodeExample :code="heroCode" language="go" />
      </div>
    </section>

    <!-- Features Section -->
    <section class="features">
      <h2>{{ $t('features.title') }}</h2>
      <div class="features-grid">
        <FeatureCard 
          v-for="feature in features" 
          :key="feature.id"
          :feature="feature" 
        />
      </div>
    </section>

    <!-- Stats Section -->
    <section class="stats">
      <div class="stats-grid">
        <div class="stat-item">
          <span class="stat-number">{{ githubStats.stars }}</span>
          <span class="stat-label">GitHub Stars</span>
        </div>
        <div class="stat-item">
          <span class="stat-number">{{ githubStats.version }}</span>
          <span class="stat-label">Latest Version</span>
        </div>
        <div class="stat-item">
          <span class="stat-number">{{ githubStats.license }}</span>
          <span class="stat-label">License</span>
        </div>
      </div>
    </section>
  </div>
</template>
```

#### 2. 功能特性卡片 (FeatureCard.vue)

```vue
<template>
  <div class="feature-card">
    <div class="feature-icon">
      <component :is="feature.icon" />
    </div>
    <h3 class="feature-title">{{ $t(feature.titleKey) }}</h3>
    <p class="feature-description">{{ $t(feature.descriptionKey) }}</p>
    <a v-if="feature.link" :href="feature.link" class="feature-link">
      {{ $t('common.learnMore') }} →
    </a>
  </div>
</template>
```

#### 3. 代码示例组件 (CodeExample.vue)

```vue
<template>
  <div class="code-example">
    <div class="code-header">
      <span class="code-language">{{ language }}</span>
      <button @click="copyCode" class="copy-button">
        {{ copied ? $t('common.copied') : $t('common.copy') }}
      </button>
    </div>
    <pre class="code-content"><code v-html="highlightedCode"></code></pre>
  </div>
</template>
```

#### 4. API 文档组件 (ApiDoc.vue)

```vue
<template>
  <div class="api-doc">
    <div class="api-sidebar">
      <div class="api-search">
        <input 
          v-model="searchQuery" 
          :placeholder="$t('api.search')"
          class="search-input"
        />
      </div>
      <nav class="api-nav">
        <div v-for="service in filteredServices" :key="service.name">
          <h3>{{ service.name }}</h3>
          <ul>
            <li v-for="endpoint in service.endpoints" :key="endpoint.id">
              <a :href="`#${endpoint.id}`">{{ endpoint.name }}</a>
            </li>
          </ul>
        </div>
      </nav>
    </div>
    <div class="api-content">
      <ApiEndpoint 
        v-for="endpoint in currentEndpoints" 
        :key="endpoint.id"
        :endpoint="endpoint"
      />
    </div>
  </div>
</template>
```

### 数据模型

#### 配置接口

```typescript
interface SiteConfig {
  title: string;
  description: string;
  lang: string;
  head: HeadConfig[];
  themeConfig: ThemeConfig;
  locales: Record<string, LocaleConfig>;
}

interface ThemeConfig {
  nav: NavItem[];
  sidebar: SidebarConfig;
  socialLinks: SocialLink[];
  footer: FooterConfig;
  search: SearchConfig;
}

interface LocaleConfig {
  lang: string;
  title: string;
  description: string;
  themeConfig: ThemeConfig;
}
```

#### API 数据模型

```typescript
interface ApiService {
  name: string;
  version: string;
  description: string;
  baseUrl: string;
  endpoints: ApiEndpoint[];
}

interface ApiEndpoint {
  id: string;
  name: string;
  method: 'GET' | 'POST' | 'PUT' | 'DELETE';
  path: string;
  description: string;
  parameters: ApiParameter[];
  responses: ApiResponse[];
  examples: ApiExample[];
}

interface ApiParameter {
  name: string;
  type: string;
  required: boolean;
  description: string;
  example?: any;
}
```

## 错误处理

### 客户端错误处理

1. **404 页面处理**
   - 自定义 404 页面，提供导航建议
   - 多语言支持的错误信息
   - 自动重定向到相似页面

2. **JavaScript 错误处理**
   - 全局错误捕获和报告
   - 优雅降级，确保基本功能可用
   - 错误边界组件防止整页崩溃

3. **网络错误处理**
   - API 调用失败的重试机制
   - 离线状态检测和提示
   - 加载状态和错误状态的用户反馈

### 构建错误处理

1. **构建失败处理**
   - 详细的错误日志记录
   - 构建失败时的邮件通知
   - 自动回滚到上一个稳定版本

2. **内容验证**
   - Markdown 语法检查
   - 链接有效性验证
   - 图片资源存在性检查

## 测试策略

### 单元测试

1. **组件测试**
   - Vue 组件的渲染测试
   - 用户交互行为测试
   - Props 和 Events 测试

2. **工具函数测试**
   - 多语言切换逻辑测试
   - 搜索功能测试
   - 数据处理函数测试

### 集成测试

1. **页面集成测试**
   - 页面路由测试
   - 多语言页面切换测试
   - API 文档集成测试

2. **构建测试**
   - 静态资源生成测试
   - 多语言构建测试
   - 部署流程测试

### 端到端测试

1. **用户流程测试**
   - 首页到文档的完整流程
   - 语言切换功能测试
   - 搜索功能端到端测试

2. **性能测试**
   - 页面加载速度测试
   - 移动端性能测试
   - SEO 指标测试

### 测试工具

- **单元测试**: Vitest + Vue Test Utils
- **端到端测试**: Playwright
- **性能测试**: Lighthouse CI
- **可访问性测试**: axe-core

## 多语言实现方案

### 国际化架构

1. **语言文件结构**
```
locales/
├── zh/
│   ├── common.json
│   ├── home.json
│   ├── guide.json
│   └── api.json
└── en/
    ├── common.json
    ├── home.json
    ├── guide.json
    └── api.json
```

2. **语言切换实现**
```typescript
// 语言切换逻辑
export function useI18n() {
  const currentLang = ref('zh');
  
  const switchLanguage = (lang: string) => {
    currentLang.value = lang;
    localStorage.setItem('preferred-language', lang);
    // 更新 URL 路径
    const newPath = window.location.pathname.replace(/^\/(zh|en)/, `/${lang}`);
    window.location.href = newPath;
  };
  
  const t = (key: string, params?: Record<string, any>) => {
    return translate(key, currentLang.value, params);
  };
  
  return { currentLang, switchLanguage, t };
}
```

3. **内容同步策略**
   - 使用脚本自动从项目 README 提取内容
   - 建立中英文内容对照表
   - 定期检查内容一致性

## 性能优化方案

### 构建优化

1. **代码分割**
   - 按页面分割 JavaScript 代码
   - 按语言分割国际化资源
   - 懒加载非关键组件

2. **资源优化**
   - 图片压缩和 WebP 格式支持
   - CSS 和 JavaScript 压缩
   - 字体文件优化和子集化

3. **缓存策略**
   - 静态资源长期缓存
   - HTML 文件短期缓存
   - Service Worker 离线缓存

### 运行时优化

1. **首屏加载优化**
   - 关键 CSS 内联
   - 预加载关键资源
   - 骨架屏加载状态

2. **交互优化**
   - 防抖搜索输入
   - 虚拟滚动长列表
   - 图片懒加载

## SEO 和可访问性

### SEO 优化

1. **元数据管理**
```typescript
// SEO 配置
export const seoConfig = {
  title: 'Swit - Go Microservice Framework',
  description: 'A comprehensive microservice framework for Go...',
  keywords: ['go', 'microservice', 'framework', 'grpc', 'http'],
  ogImage: '/images/og-image.png',
  twitterCard: 'summary_large_image'
};
```

2. **结构化数据**
   - JSON-LD 格式的软件项目信息
   - 面包屑导航标记
   - 文档结构标记

3. **URL 结构优化**
   - 语义化 URL 路径
   - 多语言 URL 结构
   - 规范链接设置

### 可访问性

1. **键盘导航**
   - 完整的键盘导航支持
   - 焦点管理和视觉指示
   - 跳转链接支持

2. **屏幕阅读器支持**
   - 语义化 HTML 结构
   - ARIA 标签和属性
   - 图片 alt 文本

3. **视觉辅助**
   - 高对比度模式支持
   - 字体大小调节
   - 色彩无障碍设计

## 部署和 CI/CD

### GitHub Actions 工作流

```yaml
name: Deploy to GitHub Pages

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3
        
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'
          cache: 'npm'
          
      - name: Install dependencies
        run: npm ci
        
      - name: Generate API docs
        run: |
          # 从项目中提取 API 文档
          npm run generate-api-docs
          
      - name: Build
        run: |
          cd docs/pages
          npm run build
        
      - name: Test
        run: npm run test
        
      - name: Deploy to GitHub Pages
        if: github.ref == 'refs/heads/main'
        uses: peaceiris/actions-gh-pages@v3
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./docs/pages/dist
```

### 自动化内容同步

1. **文档同步脚本**
```typescript
// scripts/sync-docs.ts
export async function syncDocs() {
  // 从主项目同步 README 内容
  const readmeContent = await fs.readFile('../README.md', 'utf-8');
  const readmeCnContent = await fs.readFile('../README-CN.md', 'utf-8');
  
  // 转换为网站格式
  const webContent = convertReadmeToWeb(readmeContent);
  const webCnContent = convertReadmeToWeb(readmeCnContent);
  
  // 写入网站文件
  await fs.writeFile('./docs/pages/en/index.md', webContent);
  await fs.writeFile('./docs/pages/zh/index.md', webCnContent);
}
```

2. **API 文档生成**
```typescript
// scripts/generate-api-docs.ts
export async function generateApiDocs() {
  // 读取 Swagger/OpenAPI 规格
  const switserveSpec = await loadSwaggerSpec('../docs/generated/switserve/swagger.json');
  const switauthSpec = await loadSwaggerSpec('../docs/generated/switauth/swagger.json');
  
  // 生成 VitePress 格式的 API 文档
  const apiDocs = generateVitePressApiDocs([switserveSpec, switauthSpec]);
  
  // 写入文档文件
  await writeApiDocs('./docs/pages/zh/api/', apiDocs, 'zh');
  await writeApiDocs('./docs/pages/en/api/', apiDocs, 'en');
}
```

### 监控和分析

1. **性能监控**
   - Google PageSpeed Insights 集成
   - Core Web Vitals 监控
   - 构建时间和大小监控

2. **用户分析**
   - Google Analytics 4 集成
   - 隐私友好的分析方案
   - 用户行为热力图

3. **错误监控**
   - Sentry 错误追踪
   - 构建失败通知
   - 性能回归检测

## 安全考虑

### 内容安全

1. **CSP 策略**
```html
<meta http-equiv="Content-Security-Policy" 
      content="default-src 'self'; 
               script-src 'self' 'unsafe-inline' https://www.googletagmanager.com;
               style-src 'self' 'unsafe-inline';
               img-src 'self' data: https:;">
```

2. **XSS 防护**
   - 用户输入内容过滤
   - Markdown 内容安全渲染
   - 外部链接安全处理

### 隐私保护

1. **GDPR 合规**
   - Cookie 使用声明
   - 数据收集透明度
   - 用户选择权保护

2. **数据最小化**
   - 仅收集必要的分析数据
   - 本地存储数据加密
   - 定期清理过期数据

这个设计文档提供了完整的技术架构和实现方案，涵盖了从前端组件设计到部署自动化的所有方面，确保网站能够满足所有需求并提供优秀的用户体验。