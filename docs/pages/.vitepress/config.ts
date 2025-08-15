import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Swit Framework',
  description: 'Modern Go Microservice Framework',
  
  // Language and localization
  locales: {
    root: {
      label: 'English',
      lang: 'en',
      link: '/en/',
      themeConfig: {
        nav: [
          { text: 'Guide', link: '/en/guide/getting-started' },
          { text: 'Examples', link: '/en/examples/' },
          { text: 'API', link: '/en/api/' },
          { text: 'Community', link: '/en/community/' }
        ],
        sidebar: {
          '/en/guide/': [
            {
              text: 'Getting Started',
              items: [
                { text: 'Quick Start', link: '/en/guide/getting-started' },
                { text: 'Installation', link: '/en/guide/installation' },
                { text: 'Configuration', link: '/en/guide/configuration' }
              ]
            }
          ],
          '/en/api/': [
            {
              text: 'API Documentation',
              items: [
                { text: 'Overview', link: '/en/api/' },
                { text: 'SwitServe API', link: '/en/api/switserve' },
                { text: 'SwitAuth API', link: '/en/api/switauth' }
              ]
            }
          ]
        }
      }
    },
    zh: {
      label: '中文',
      lang: 'zh-CN',
      link: '/zh/',
      themeConfig: {
        nav: [
          { text: '指南', link: '/zh/guide/getting-started' },
          { text: '示例', link: '/zh/examples/' },
          { text: 'API', link: '/zh/api/' },
          { text: '社区', link: '/zh/community/' }
        ],
        sidebar: {
          '/zh/guide/': [
            {
              text: '快速开始',
              items: [
                { text: '快速开始', link: '/zh/guide/getting-started' },
                { text: '安装指南', link: '/zh/guide/installation' },
                { text: '配置说明', link: '/zh/guide/configuration' }
              ]
            }
          ],
          '/zh/api/': [
            {
              text: 'API 文档',
              items: [
                { text: '概览', link: '/zh/api/' },
                { text: 'SwitServe API', link: '/zh/api/switserve' },
                { text: 'SwitAuth API', link: '/zh/api/switauth' }
              ]
            }
          ]
        }
      }
    }
  },

  // Theme configuration
  themeConfig: {
    logo: '/images/logo.svg',
    
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
                  navigateText: '切换'
                }
              }
            }
          }
        }
      }
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/innovationmech/swit' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2025 Swit Framework Contributors'
    }
  },

  // Build configuration
  outDir: 'dist',
  cacheDir: '.vitepress/cache',
  
  // Head configuration for SEO
  head: [
    ['meta', { name: 'theme-color', content: '#646cff' }],
    ['meta', { name: 'keywords', content: 'go, golang, microservice, framework, http, grpc, rest, api' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:site_name', content: 'Swit Framework' }],
    ['meta', { property: 'og:image', content: '/images/og-image.png' }],
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/images/favicon.svg' }]
  ],

  // Markdown configuration
  markdown: {
    theme: {
      light: 'github-light',
      dark: 'github-dark'
    },
    lineNumbers: true,
    config: (md) => {
      // Add custom markdown plugins if needed
    }
  },

  // Vite configuration
  vite: {
    define: {
      __VUE_OPTIONS_API__: false
    },
    resolve: {
      alias: {
        '@': require('path').resolve(__dirname, './theme')
      }
    }
  }
})