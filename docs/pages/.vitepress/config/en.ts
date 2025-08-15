import { defineConfig, type DefaultTheme } from 'vitepress'

export const en = defineConfig({
  lang: 'en-US',
  description: 'A comprehensive Go microservice framework providing production-ready components for building scalable microservices',
  themeConfig: {
    nav: nav(),
    sidebar: {
      '/en/guide/': { base: '/en/guide/', items: sidebarGuide() },
      '/en/api/': { base: '/en/api/', items: sidebarApi() },
      '/en/examples/': { base: '/en/examples/', items: sidebarExamples() },
      '/en/reference/': { base: '/en/reference/', items: sidebarReference() }
    },
    editLink: {
      pattern: 'https://github.com/innovationmech/swit/edit/main/docs/pages/:path',
      text: 'Edit this page on GitHub'
    },
    footer: {
      message: 'Released under the MIT License.',
      copyright: `Copyright © 2024-${new Date().getFullYear()} Swit Project`
    }
  }
})

function nav(): DefaultTheme.NavItem[] {
  return [
    {
      text: 'Guide',
      link: '/en/guide/getting-started',
      activeMatch: '/en/guide/'
    },
    {
      text: 'API',
      link: '/en/api/overview',
      activeMatch: '/en/api/'
    },
    {
      text: 'Examples',
      link: '/en/examples/quick-start',
      activeMatch: '/en/examples/'
    },
    {
      text: 'Reference',
      items: [
        {
          text: 'Config Reference',
          link: '/en/reference/config'
        },
        {
          text: 'CLI Reference',
          link: '/en/reference/cli'
        },
        {
          text: 'Error Handling',
          link: '/en/reference/errors'
        }
      ]
    },
    {
      text: 'v1.0.0',
      items: [
        {
          text: 'Changelog',
          link: 'https://github.com/innovationmech/swit/blob/main/CHANGELOG.md'
        },
        {
          text: 'Contributing',
          link: 'https://github.com/innovationmech/swit/blob/main/CONTRIBUTING.md'
        }
      ]
    }
  ]
}

function sidebarGuide(): DefaultTheme.SidebarItem[] {
  return [
    {
      text: 'Introduction',
      collapsed: false,
      items: [
        { text: 'What is Swit?', link: 'what-is-swit' },
        { text: 'Getting Started', link: 'getting-started' },
        { text: 'Project Structure', link: 'project-structure' }
      ]
    },
    {
      text: 'Basics',
      collapsed: false,
      items: [
        { text: 'Server Configuration', link: 'server-config' },
        { text: 'HTTP Service', link: 'http-service' },
        { text: 'gRPC Service', link: 'grpc-service' },
        { text: 'Service Registration', link: 'service-registration' }
      ]
    },
    {
      text: 'Advanced',
      collapsed: false,
      items: [
        { text: 'Middleware', link: 'middleware' },
        { text: 'Service Discovery', link: 'service-discovery' },
        { text: 'Dependency Injection', link: 'dependency-injection' },
        { text: 'Performance Monitoring', link: 'performance-monitoring' }
      ]
    },
    {
      text: 'Best Practices',
      collapsed: false,
      items: [
        { text: 'Error Handling', link: 'error-handling' },
        { text: 'Logging', link: 'logging' },
        { text: 'Testing', link: 'testing' },
        { text: 'Deployment', link: 'deployment' }
      ]
    }
  ]
}

function sidebarApi(): DefaultTheme.SidebarItem[] {
  return [
    {
      text: 'API Reference',
      items: [
        { text: 'Overview', link: 'overview' },
        { text: 'Server API', link: 'server' },
        { text: 'Transport API', link: 'transport' },
        { text: 'Middleware API', link: 'middleware' },
        { text: 'Utilities', link: 'utils' }
      ]
    },
    {
      text: 'Service APIs',
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
      text: 'Getting Started',
      items: [
        { text: 'Quick Start', link: 'quick-start' },
        { text: 'Simple HTTP Service', link: 'simple-http' },
        { text: 'gRPC Service', link: 'grpc-service' }
      ]
    },
    {
      text: 'Complete Examples',
      items: [
        { text: 'Full Featured Service', link: 'full-featured' },
        { text: 'User Management System', link: 'user-management' },
        { text: 'Authentication Service', link: 'auth-service' }
      ]
    },
    {
      text: 'Integration Examples',
      items: [
        { text: 'Database Integration', link: 'database' },
        { text: 'Cache Integration', link: 'cache' },
        { text: 'Message Queue', link: 'message-queue' }
      ]
    }
  ]
}

function sidebarReference(): DefaultTheme.SidebarItem[] {
  return [
    {
      text: 'Configuration',
      items: [
        { text: 'Server Config', link: 'config' },
        { text: 'Environment Variables', link: 'env-vars' },
        { text: 'YAML Configuration', link: 'yaml-config' }
      ]
    },
    {
      text: 'CLI',
      items: [
        { text: 'CLI Commands', link: 'cli' },
        { text: 'Make Commands', link: 'make-commands' },
        { text: 'Development Tools', link: 'dev-tools' }
      ]
    },
    {
      text: 'Troubleshooting',
      items: [
        { text: 'FAQ', link: 'faq' },
        { text: 'Error Codes', link: 'errors' },
        { text: 'Debugging Tips', link: 'debugging' }
      ]
    }
  ]
}