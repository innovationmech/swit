module.exports = {
  ci: {
    collect: {
      url: [
        'http://localhost:3000',
        'http://localhost:3000/en/',
        'http://localhost:3000/zh/',
        'http://localhost:3000/en/guide/getting-started',
        'http://localhost:3000/en/api/',
        'http://localhost:3000/en/examples/'
      ],
      startServerCommand: 'cd docs/pages && npm run preview',
      startServerReadyPattern: 'Local:',
      startServerReadyTimeout: 30000,
      numberOfRuns: 3
    },
    assert: {
      assertions: {
        'categories:performance': ['warn', { minScore: 0.85 }],
        'categories:accessibility': ['error', { minScore: 0.95 }],
        'categories:best-practices': ['warn', { minScore: 0.90 }],
        'categories:seo': ['warn', { minScore: 0.90 }],
        'categories:pwa': 'off',
        
        // Core Web Vitals
        'first-contentful-paint': ['warn', { maxNumericValue: 2000 }],
        'largest-contentful-paint': ['warn', { maxNumericValue: 2500 }],
        'cumulative-layout-shift': ['error', { maxNumericValue: 0.1 }],
        'total-blocking-time': ['warn', { maxNumericValue: 300 }],
        
        // 其他重要指标
        'speed-index': ['warn', { maxNumericValue: 3000 }],
        'interactive': ['warn', { maxNumericValue: 3500 }],
        
        // 可访问性特定检查
        'color-contrast': 'error',
        'image-alt': 'error',
        'label': 'error',
        'valid-lang': 'error',
        'meta-description': 'warn',
        'document-title': 'error',
        
        // 最佳实践
        'uses-responsive-images': 'warn',
        'efficient-animated-content': 'warn',
        'unused-css-rules': 'off', // VitePress 可能有未使用的 CSS
        'legacy-javascript': 'warn'
      }
    },
    upload: {
      target: 'temporary-public-storage',
      // 如果有 LHCI 服务器可以配置:
      // target: 'lhci',
      // serverBaseUrl: 'https://your-lhci-server.com'
    },
    server: {
      // 如果使用 LHCI 服务器
      // port: 9001,
      // storage: {
      //   storageMethod: 'sql',
      //   sqlDialect: 'sqlite',
      //   sqlDatabasePath: './lhci.db'
      // }
    },
    wizard: {
      // 向导配置（用于初始设置）
    }
  }
}