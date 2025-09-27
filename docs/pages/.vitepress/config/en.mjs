import { defineConfig } from 'vitepress'
import { sharedConfig } from './shared.mjs'

export const enConfig = defineConfig({
  ...sharedConfig,
  
  lang: 'en-US',
  title: 'Swit Framework',
  description: 'Production-ready Go microservice development framework',
  
  // English-specific SEO configuration
  head: [
    // Canonical URL and alternate language versions
    ['link', { rel: 'canonical', href: 'https://innovationmech.github.io/swit/' }],
    ['link', { rel: 'alternate', hreflang: 'en', href: 'https://innovationmech.github.io/swit/' }],
    ['link', { rel: 'alternate', hreflang: 'zh', href: 'https://innovationmech.github.io/swit/zh/' }],
    ['link', { rel: 'alternate', hreflang: 'x-default', href: 'https://innovationmech.github.io/swit/' }],
    
    // English-specific Open Graph tags
    ['meta', { property: 'og:title', content: 'Swit Framework - Modern Go Microservice Framework' }],
    ['meta', { property: 'og:description', content: 'Production-ready Go microservice development framework with HTTP and gRPC support, dependency injection, and performance monitoring' }],
    ['meta', { property: 'og:url', content: 'https://innovationmech.github.io/swit/' }],
    
    // English-specific Twitter Card tags
    ['meta', { name: 'twitter:title', content: 'Swit Framework - Modern Go Microservice Framework' }],
    ['meta', { name: 'twitter:description', content: 'Production-ready Go microservice development framework with HTTP and gRPC support, dependency injection, and performance monitoring' }],
    
    // Additional structured data for English version
    ['script', { type: 'application/ld+json' }, JSON.stringify({
      "@context": "https://schema.org",
      "@type": "TechArticle",
      "headline": "Swit Framework Documentation",
      "description": "Comprehensive documentation for the Swit Go microservice framework",
      "url": "https://innovationmech.github.io/swit/",
      "inLanguage": "en-US",
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
        "@id": "https://innovationmech.github.io/swit/"
      }
    })]
  ],
  
  themeConfig: {
    ...sharedConfig.themeConfig,
    
    nav: [
      { text: 'Guide', link: '/en/guide/getting-started' },
      { text: 'CLI Tools', link: '/en/cli/' },
      { text: 'Examples', link: '/en/examples/' },
      { text: 'API', link: '/en/api/' },
      { text: 'Community', link: '/en/community/' }
    ],
    
    sidebar: {
      '/en/reference/': [
        {
          text: 'Reference',
          items: [
            { text: 'Capabilities Matrix', link: '/en/reference/capabilities' },
            { text: 'Configuration Compatibility', link: '/en/reference/config-compatibility' }
          ]
        }
      ],
      '/en/guide/': [
        {
          text: 'Getting Started',
          items: [
            { text: 'Quick Start', link: '/en/guide/getting-started' },
            { text: 'Installation', link: '/en/guide/installation' },
            { text: 'Configuration', link: '/en/guide/configuration' }
          ]
        },
        {
          text: 'Core Concepts',
          items: [
            { text: 'Server Framework', link: '/en/guide/server-framework' },
            { text: 'Transport Layer', link: '/en/guide/transport-layer' },
            { text: 'Dependency Injection', link: '/en/guide/dependency-injection' }
          ]
        },
        {
          text: 'Advanced Topics',
          items: [
            { text: 'Error Monitoring', link: '/en/guide/monitoring' },
            { text: 'Performance Monitoring', link: '/en/guide/performance' },
            { text: 'Service Discovery', link: '/en/guide/service-discovery' },
            { text: 'Testing', link: '/en/guide/testing' },
            { text: 'Deployment Examples', link: '/en/guide/deployment-examples' }
          ]
        },
        {
          text: 'Migration & Maintenance',
          items: [
            { text: 'Migration Guide', link: '/en/guide/migration' },
            { text: 'Troubleshooting & FAQ', link: '/en/guide/troubleshooting' }
          ]
        },
        {
          text: 'Website Usage',
          items: [
            { text: 'Website Usage Guide', link: '/en/guide/website-usage' }
          ]
        }
      ],
      
      '/en/cli/': [
        {
          text: 'CLI Tools (switctl)',
          items: [
            { text: 'Overview', link: '/en/cli/' },
            { text: 'Getting Started', link: '/en/cli/getting-started' },
            { text: 'Commands Reference', link: '/en/cli/commands' },
            { text: 'Template System', link: '/en/cli/templates' }
          ]
        }
      ],
      
      '/en/examples/': [
        {
          text: 'Examples',
          items: [
            { text: 'Overview', link: '/en/examples/' },
            { text: 'Simple HTTP Service', link: '/en/examples/simple-http' },
            { text: 'gRPC Service', link: '/en/examples/grpc-service' },
            { text: 'Full-Featured Service', link: '/en/examples/full-featured' },
            { text: 'Sentry Monitoring', link: '/en/examples/sentry-service' }
          ]
        }
      ],
      
      '/en/api/': [
        {
          text: 'API Documentation',
          items: [
            { text: 'Overview', link: '/en/api/' },
            { text: 'Complete Reference', link: '/en/api/complete' },
            { text: 'SwitServe API', link: '/en/api/switserve' },
            { text: 'SwitAuth API', link: '/en/api/switauth' }
          ]
        }
      ],
      
      '/en/community/': [
        {
          text: 'Community',
          items: [
            { text: 'Overview', link: '/en/community/' },
            { text: 'Contributing', link: '/en/community/contributing' },
            { text: 'Code of Conduct', link: '/en/community/code_of_conduct' },
            { text: 'Security', link: '/en/community/security' }
          ]
        },
        {
          text: 'Development',
          items: [
            { text: 'Development Guide', link: '/en/community/development' },
            { text: 'Website Development', link: '/en/community/website-development' },
            { text: 'User Feedback', link: '/en/community/user-feedback' }
          ]
        }
      ]
    },
    
    search: {
      provider: 'local',
      options: {
        locales: {
          root: {
            translations: {
              button: {
                buttonText: 'Search',
                buttonAriaLabel: 'Search docs'
              },
              modal: {
                noResultsText: 'No results found',
                resetButtonTitle: 'Clear search query',
                footer: {
                  selectText: 'to select',
                  navigateText: 'to navigate',
                  closeText: 'to close'
                }
              }
            }
          }
        }
      }
    }
  }
})