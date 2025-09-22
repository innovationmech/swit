import { defineConfig } from 'vitepress'
import { sharedConfig } from './shared.mjs'

export const zhConfig = defineConfig({
  ...sharedConfig,
  
  lang: 'zh-CN',
  title: 'Swit 框架',
  description: '生产级 Go 微服务开发框架',
  
  // Chinese-specific SEO configuration
  head: [
    // Canonical URL and alternate language versions
    ['link', { rel: 'canonical', href: 'https://innovationmech.github.io/swit/zh/' }],
    ['link', { rel: 'alternate', hreflang: 'en', href: 'https://innovationmech.github.io/swit/' }],
    ['link', { rel: 'alternate', hreflang: 'zh', href: 'https://innovationmech.github.io/swit/zh/' }],
    ['link', { rel: 'alternate', hreflang: 'x-default', href: 'https://innovationmech.github.io/swit/' }],
    
    // Chinese-specific Open Graph tags
    ['meta', { property: 'og:title', content: 'Swit 框架 - 现代 Go 微服务框架' }],
    ['meta', { property: 'og:description', content: '生产级 Go 微服务开发框架，支持 HTTP 和 gRPC、依赖注入、性能监控等功能' }],
    ['meta', { property: 'og:url', content: 'https://innovationmech.github.io/swit/zh/' }],
    ['meta', { property: 'og:locale', content: 'zh_CN' }],
    
    // Chinese-specific Twitter Card tags
    ['meta', { name: 'twitter:title', content: 'Swit 框架 - 现代 Go 微服务框架' }],
    ['meta', { name: 'twitter:description', content: '生产级 Go 微服务开发框架，支持 HTTP 和 gRPC、依赖注入、性能监控等功能' }],
    
    // Additional structured data for Chinese version
    ['script', { type: 'application/ld+json' }, JSON.stringify({
      "@context": "https://schema.org",
      "@type": "TechArticle",
      "headline": "Swit 框架文档",
      "description": "Swit Go 微服务框架的完整文档",
      "url": "https://innovationmech.github.io/swit/zh/",
      "inLanguage": "zh-CN",
      "author": {
        "@type": "Organization",
        "name": "Swit Framework Contributors"
      },
      "publisher": {
        "@type": "Organization",
        "name": "Innovation Mechanism",
        "logo": {
          "@type": "ImageObject",
          "url": "https://innovationmech.github.io/swit/images/logo.svg"
        }
      },
      "dateModified": new Date().toISOString(),
      "mainEntityOfPage": {
        "@type": "WebPage",
        "@id": "https://innovationmech.github.io/swit/zh/"
      }
    })],
    
    // Chinese-specific meta keywords
    ['meta', { name: 'keywords', content: 'go, golang, 微服务, 框架, http, grpc, rest, api, swit, 云原生, 分布式系统, 生产级' }]
  ],
  
  themeConfig: {
    ...sharedConfig.themeConfig,
    
    nav: [
      { text: '指南', link: '/zh/guide/getting-started' },
      { text: 'CLI 工具', link: '/zh/cli/' },
      { text: '示例', link: '/zh/examples/' },
      { text: 'API', link: '/zh/api/' },
      { text: '社区', link: '/zh/community/' }
    ],
    
    sidebar: {
      '/zh/reference/': [
        {
          text: '参考',
          items: [
            { text: '能力矩阵', link: '/zh/reference/capabilities' },
            { text: '配置兼容性矩阵', link: '/zh/reference/config-compatibility' }
          ]
        }
      ],
      '/zh/guide/': [
        {
          text: '快速开始',
          items: [
            { text: '快速开始', link: '/zh/guide/getting-started' },
            { text: '安装指南', link: '/zh/guide/installation' },
            { text: '配置说明', link: '/zh/guide/configuration' }
          ]
        },
        {
          text: '核心概念',
          items: [
            { text: '服务器框架', link: '/zh/guide/server-framework' },
            { text: '传输层', link: '/zh/guide/transport-layer' },
            { text: '依赖注入', link: '/zh/guide/dependency-injection' }
          ]
        },
        {
          text: '高级主题',
          items: [
            { text: '错误监控', link: '/zh/guide/monitoring' },
            { text: '性能监控', link: '/zh/guide/performance' },
            { text: '服务发现', link: '/zh/guide/service-discovery' },
            { text: '测试', link: '/zh/guide/testing' }
          ]
        },
        {
          text: '网站使用',
          items: [
            { text: '网站使用指南', link: '/zh/guide/website-usage' },
            { text: '故障排除与常见问题', link: '/zh/guide/troubleshooting' }
          ]
        }
      ],
      
      '/zh/cli/': [
        {
          text: 'CLI 工具 (switctl)',
          items: [
            { text: '概览', link: '/zh/cli/' },
            { text: '入门指南', link: '/zh/cli/getting-started' },
            { text: '命令参考', link: '/zh/cli/commands' },
            { text: '模板系统', link: '/zh/cli/templates' }
          ]
        }
      ],
      
      '/zh/examples/': [
        {
          text: '示例',
          items: [
            { text: '概览', link: '/zh/examples/' },
            { text: '简单 HTTP 服务', link: '/zh/examples/simple-http' },
            { text: 'gRPC 服务', link: '/zh/examples/grpc-service' },
            { text: '全功能服务', link: '/zh/examples/full-featured' },
            { text: 'Sentry 监控', link: '/zh/examples/sentry-service' }
          ]
        }
      ],
      
      '/zh/api/': [
        {
          text: 'API 文档',
          items: [
            { text: '概览', link: '/zh/api/' },
            { text: '完整参考', link: '/zh/api/complete' },
            { text: 'SwitServe API', link: '/zh/api/switserve' },
            { text: 'SwitAuth API', link: '/zh/api/switauth' }
          ]
        }
      ],
      
      '/zh/community/': [
        {
          text: '社区',
          items: [
            { text: '概览', link: '/zh/community/' },
            { text: '贡献指南', link: '/zh/community/contributing' },
            { text: '行为准则', link: '/zh/community/code_of_conduct' },
            { text: '安全政策', link: '/zh/community/security' }
          ]
        },
        {
          text: '开发',
          items: [
            { text: '开发指南', link: '/zh/community/development' },
            { text: '网站开发指南', link: '/zh/community/website-development' },
            { text: '用户反馈与贡献', link: '/zh/community/user-feedback' }
          ]
        }
      ]
    },
    
    search: {
      provider: 'local',
      options: {
        locales: {
          zh: {
            translations: {
              button: {
                buttonText: '搜索文档',
                buttonAriaLabel: '搜索文档'
              },
              modal: {
                noResultsText: '无法找到相关结果',
                resetButtonTitle: '清除查询条件',
                footer: {
                  selectText: '选择',
                  navigateText: '切换',
                  closeText: '关闭'
                }
              }
            }
          }
        }
      }
    }
  }
})