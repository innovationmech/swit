import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  
  test: {
    // 测试环境配置
    environment: 'happy-dom',
    
    // 全局设置
    globals: true,
    
    // 测试文件匹配模式
    include: [
      'tests/unit/**/*.{test,spec}.{js,mjs,cjs,ts,mts,cts,jsx,tsx}',
      '.vitepress/theme/**/*.{test,spec}.{js,mjs,cjs,ts,mts,cts,jsx,tsx}'
    ],
    
    // 排除的文件
    exclude: [
      'node_modules',
      'dist',
      '.vitepress/cache',
      'tests/e2e',
      'tests/performance',
      'tests/accessibility'
    ],
    
    // 覆盖率配置
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html', 'lcov'],
      reportsDirectory: './tests/coverage',
      exclude: [
        'node_modules/',
        'tests/',
        'dist/',
        '.vitepress/cache/',
        '**/*.config.*',
        '**/*.d.ts',
        'scripts/',
        'public/'
      ],
      // 覆盖率阈值
      thresholds: {
        global: {
          branches: 70,
          functions: 70,
          lines: 70,
          statements: 70
        }
      }
    },
    
    // 设置文件
    setupFiles: ['./tests/setup.ts'],
    
    // 并发配置
    pool: 'threads',
    poolOptions: {
      threads: {
        singleThread: false,
        minThreads: 1,
        maxThreads: 4
      }
    },
    
    // 测试超时
    testTimeout: 10000,
    hookTimeout: 10000,
    
    // 报告器
    reporters: ['verbose', 'json', 'html'],
    outputFile: {
      json: './tests/reports/results.json',
      html: './tests/reports/index.html'
    },
    
    // 监视模式配置
    watchExclude: [
      'node_modules/**',
      'dist/**',
      '.vitepress/cache/**'
    ],
    
    // 模拟配置
    clearMocks: true,
    restoreMocks: true,
    mockReset: true
  },
  
  // 路径解析
  resolve: {
    alias: {
      '@': resolve(__dirname, '.vitepress/theme'),
      '@components': resolve(__dirname, '.vitepress/theme/components'),
      '@styles': resolve(__dirname, '.vitepress/theme/styles'),
      '@utils': resolve(__dirname, '.vitepress/theme/utils'),
      '@tests': resolve(__dirname, 'tests')
    }
  },
  
  // 定义全局变量
  define: {
    __VUE_OPTIONS_API__: true,
    __VUE_PROD_DEVTOOLS__: false
  }
})