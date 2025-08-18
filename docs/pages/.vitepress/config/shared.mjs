// 共享配置
export const sharedConfig = {
  title: 'Swit Framework',
  description: 'Modern Go Microservice Framework',
  // GitHub Pages 仓库项目路径需要 base，默认 /swit/；允许通过环境变量覆盖（本地 dev 可 unset 使用根路径）。
  base: process.env.VITEPRESS_BASE || '/swit/',
  
  // Build configuration
  outDir: 'dist',
  cacheDir: '.vitepress/cache',
  
  // Head configuration for SEO
  head: [
    // Basic meta tags
    ['meta', { name: 'theme-color', content: '#646cff' }],
    ['meta', { name: 'keywords', content: 'go, golang, microservice, framework, http, grpc, rest, api, swit, cloud native, distributed systems' }],
    ['meta', { name: 'author', content: 'Swit Framework Contributors' }],
    ['meta', { name: 'robots', content: 'index,follow' }],
    ['meta', { name: 'googlebot', content: 'index,follow' }],
    
    // Open Graph tags for social media
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:site_name', content: 'Swit Framework' }],
    ['meta', { property: 'og:image', content: '/images/og-image.png' }],
    ['meta', { property: 'og:image:width', content: '1200' }],
    ['meta', { property: 'og:image:height', content: '630' }],
    ['meta', { property: 'og:image:alt', content: 'Swit Framework - Modern Go Microservice Framework' }],
    ['meta', { property: 'og:locale', content: 'en_US' }],
    ['meta', { property: 'og:locale:alternate', content: 'zh_CN' }],
    
    // Twitter Card tags
    ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
    ['meta', { name: 'twitter:site', content: '@switframework' }],
    ['meta', { name: 'twitter:creator', content: '@switframework' }],
    ['meta', { name: 'twitter:image', content: '/images/twitter-card.png' }],
    ['meta', { name: 'twitter:image:alt', content: 'Swit Framework - Modern Go Microservice Framework' }],
    
    // Structured data for SEO
    ['script', { type: 'application/ld+json' }, JSON.stringify({
      "@context": "https://schema.org",
      "@type": "SoftwareApplication",
      "name": "Swit Framework",
      "applicationCategory": "DeveloperApplication",
      "operatingSystem": "Cross-platform",
      "description": "Modern Go microservice framework for building scalable, production-ready applications with HTTP and gRPC support",
      "url": "https://innovationmech.github.io/swit/",
      "author": {
        "@type": "Organization",
        "name": "Swit Framework Contributors",
        "url": "https://github.com/innovationmech/swit"
      },
      "programmingLanguage": "Go",
      "license": "MIT",
      "codeRepository": "https://github.com/innovationmech/swit",
      "downloadUrl": "https://github.com/innovationmech/swit/releases",
      "installUrl": "https://innovationmech.github.io/swit/guide/getting-started.html",
      "screenshot": "/images/framework-screenshot.png",
      "softwareVersion": "1.0.0",
      "releaseNotes": "https://github.com/innovationmech/swit/releases",
      "supportingData": "https://innovationmech.github.io/swit/api/",
      "maintainer": {
        "@type": "Organization",
        "name": "Innovation Mechanism",
        "url": "https://github.com/innovationmech"
      }
    })],
    
    // Favicons and app icons
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/images/favicon.svg' }],
    ['link', { rel: 'icon', type: 'image/png', sizes: '32x32', href: '/images/favicon-32x32.png' }],
    ['link', { rel: 'icon', type: 'image/png', sizes: '16x16', href: '/images/favicon-16x16.png' }],
    ['link', { rel: 'apple-touch-icon', sizes: '180x180', href: '/images/apple-touch-icon.png' }],
    ['link', { rel: 'manifest', href: '/manifest.json' }],
    
    // Preconnect to external resources
    ['link', { rel: 'preconnect', href: 'https://fonts.googleapis.com' }],
    ['link', { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' }],
    ['link', { rel: 'dns-prefetch', href: 'https://github.com' }],
    
    // Performance optimization
    ['link', { rel: 'prefetch', href: '/images/og-image.png' }],
    ['link', { rel: 'prefetch', href: '/images/twitter-card.png' }]
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

  // Performance and build optimizations
  cleanUrls: true,
  metaChunk: true,
  ignoreDeadLinks: true,
  
  // Simplified Vite configuration
  vite: {
    server: {
      fs: {
        allow: ['..', '../..']
      }
    }
  },

  // PWA and caching configuration
  transformPageData(pageData) {
    // Add performance-related page data
    pageData.frontmatter.head = pageData.frontmatter.head || []
    
    // Add preload directives for critical resources
    if (pageData.relativePath === 'index.md') {
      pageData.frontmatter.head.push(
        ['link', { rel: 'preload', href: '/images/logo.svg', as: 'image' }],
        ['link', { rel: 'preload', href: '/images/favicon.svg', as: 'image' }]
      )
    }
    
    return pageData
  },

  // Theme configuration
  themeConfig: {
    logo: '/images/logo.svg',
    
    socialLinks: [
      { icon: 'github', link: 'https://github.com/innovationmech/swit' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2025 Swit Framework Contributors'
    }
  }
}