import { defineConfig, type DefaultTheme } from 'vitepress'

export const zh = defineConfig({
  lang: 'zh-CN',
  description: '一个全面的 Go 微服务框架，提供生产就绪的组件用于构建可扩展的微服务',
  themeConfig: {
    nav: nav(),
    sidebar: {
      '/zh/guide/': { base: '/zh/guide/', items: sidebarGuide() },
      '/zh/api/': { base: '/zh/api/', items: sidebarApi() },
      '/zh/examples/': { base: '/zh/examples/', items: sidebarExamples() },
      '/zh/reference/': { base: '/zh/reference/', items: sidebarReference() }
    },
    editLink: {
      pattern: 'https://github.com/innovationmech/swit/edit/main/docs/pages/:path',
      text: '在 GitHub 上编辑此页面'
    },
    footer: {
      message: '基于 MIT 许可发布',
      copyright: `版权所有 © 2024-${new Date().getFullYear()} Swit 项目`
    },
    docFooter: {
      prev: '上一页',
      next: '下一页'
    },
    outline: {
      label: '页面导航',
      level: [2, 3]
    },
    lastUpdated: {
      text: '最后更新于',
      formatOptions: {
        dateStyle: 'short',
        timeStyle: 'medium'
      }
    },
    langMenuLabel: '多语言',
    returnToTopLabel: '回到顶部',
    sidebarMenuLabel: '菜单',
    darkModeSwitchLabel: '主题',
    lightModeSwitchTitle: '切换到浅色模式',
    darkModeSwitchTitle: '切换到深色模式'
  }
})

function nav(): DefaultTheme.NavItem[] {
  return [
    {
      text: '指南',
      link: '/zh/guide/getting-started',
      activeMatch: '/zh/guide/'
    },
    {
      text: 'API 文档',
      link: '/zh/api/overview',
      activeMatch: '/zh/api/'
    },
    {
      text: '示例',
      link: '/zh/examples/quick-start',
      activeMatch: '/zh/examples/'
    },
    {
      text: '参考',
      items: [
        {
          text: '配置参考',
          link: '/zh/reference/config'
        },
        {
          text: 'CLI 参考',
          link: '/zh/reference/cli'
        },
        {
          text: '错误处理',
          link: '/zh/reference/errors'
        }
      ]
    },
    {
      text: 'v1.0.0',
      items: [
        {
          text: '更新日志',
          link: 'https://github.com/innovationmech/swit/blob/main/CHANGELOG.md'
        },
        {
          text: '贡献指南',
          link: 'https://github.com/innovationmech/swit/blob/main/CONTRIBUTING.md'
        }
      ]
    }
  ]
}

function sidebarGuide(): DefaultTheme.SidebarItem[] {
  return [
    {
      text: '介绍',
      collapsed: false,
      items: [
        { text: '什么是 Swit？', link: 'what-is-swit' },
        { text: '快速开始', link: 'getting-started' },
        { text: '项目结构', link: 'project-structure' }
      ]
    },
    {
      text: '基础',
      collapsed: false,
      items: [
        { text: '服务器配置', link: 'server-config' },
        { text: 'HTTP 服务', link: 'http-service' },
        { text: 'gRPC 服务', link: 'grpc-service' },
        { text: '服务注册', link: 'service-registration' }
      ]
    },
    {
      text: '进阶',
      collapsed: false,
      items: [
        { text: '中间件', link: 'middleware' },
        { text: '服务发现', link: 'service-discovery' },
        { text: '依赖注入', link: 'dependency-injection' },
        { text: '性能监控', link: 'performance-monitoring' }
      ]
    },
    {
      text: '最佳实践',
      collapsed: false,
      items: [
        { text: '错误处理', link: 'error-handling' },
        { text: '日志记录', link: 'logging' },
        { text: '测试策略', link: 'testing' },
        { text: '部署指南', link: 'deployment' }
      ]
    }
  ]
}

function sidebarApi(): DefaultTheme.SidebarItem[] {
  return [
    {
      text: 'API 参考',
      items: [
        { text: '概览', link: 'overview' },
        { text: '服务器 API', link: 'server' },
        { text: '传输层 API', link: 'transport' },
        { text: '中间件 API', link: 'middleware' },
        { text: '工具函数', link: 'utils' }
      ]
    },
    {
      text: '服务 API',
      items: [
        { text: 'Swit Serve API', link: 'swit-serve' },
        { text: 'Swit Auth API', link: 'swit-auth' },
        { text: 'Swit CTL API', link: 'swit-ctl' }
      ]
    }
  ]
}

function sidebarExamples(): DefaultTheme.SidebarItem[] {
  return [
    {
      text: '入门示例',
      items: [
        { text: '快速开始', link: 'quick-start' },
        { text: '简单 HTTP 服务', link: 'simple-http' },
        { text: 'gRPC 服务', link: 'grpc-service' }
      ]
    },
    {
      text: '完整示例',
      items: [
        { text: '全功能服务', link: 'full-featured' },
        { text: '用户管理系统', link: 'user-management' },
        { text: '认证服务', link: 'auth-service' }
      ]
    },
    {
      text: '集成示例',
      items: [
        { text: '数据库集成', link: 'database' },
        { text: '缓存集成', link: 'cache' },
        { text: '消息队列', link: 'message-queue' }
      ]
    }
  ]
}

function sidebarReference(): DefaultTheme.SidebarItem[] {
  return [
    {
      text: '配置',
      items: [
        { text: '服务器配置', link: 'config' },
        { text: '环境变量', link: 'env-vars' },
        { text: 'YAML 配置', link: 'yaml-config' }
      ]
    },
    {
      text: 'CLI',
      items: [
        { text: 'CLI 命令', link: 'cli' },
        { text: 'Make 命令', link: 'make-commands' },
        { text: '开发工具', link: 'dev-tools' }
      ]
    },
    {
      text: '故障排除',
      items: [
        { text: '常见问题', link: 'faq' },
        { text: '错误代码', link: 'errors' },
        { text: '调试技巧', link: 'debugging' }
      ]
    }
  ]
}