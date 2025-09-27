import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  // 测试目录
  testDir: './tests/e2e',
  
  // 全局设置
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  
  // 报告器配置
  reporter: [
    ['html', { outputFolder: 'tests/reports/playwright-report' }],
    ['json', { outputFile: 'tests/reports/playwright-results.json' }],
    ['junit', { outputFile: 'tests/reports/playwright-results.xml' }],
    process.env.CI ? ['github'] : ['list']
  ],
  
  // 全局测试配置
  use: {
    // 基础URL
    baseURL: 'http://localhost:3000',
    
    // 浏览器配置
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    
    // 等待策略
    actionTimeout: 30000,
    navigationTimeout: 30000,
    
    // 额外的上下文选项
    locale: 'en-US',
    timezoneId: 'America/New_York',
    colorScheme: 'light',
    
    // 视口设置
    viewport: { width: 1280, height: 720 },
    
    // 忽略HTTPS错误
    ignoreHTTPSErrors: true,
    
    // 用户代理
    userAgent: 'Playwright Test Agent'
  },

  // 项目配置 - 多浏览器测试
  projects: [
    // 桌面浏览器
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },

    // 移动浏览器
    {
      name: 'Mobile Chrome',
      use: { ...devices['Pixel 5'] },
    },
    {
      name: 'Mobile Safari',
      use: { ...devices['iPhone 12'] },
    },

    // 平板设备
    {
      name: 'iPad',
      use: { ...devices['iPad Pro'] },
    },

    // 高DPI显示器
    {
      name: 'High DPI',
      use: {
        ...devices['Desktop Chrome'],
        deviceScaleFactor: 2,
        viewport: { width: 1920, height: 1080 }
      },
    },

    // 深色模式测试
    {
      name: 'Dark Mode',
      use: {
        ...devices['Desktop Chrome'],
        colorScheme: 'dark'
      },
    },

    // 无JavaScript测试
    {
      name: 'No JavaScript',
      use: {
        ...devices['Desktop Chrome'],
        javaScriptEnabled: false
      },
    },

    // 慢速网络测试
    {
      name: 'Slow Network',
      use: {
        ...devices['Desktop Chrome'],
        // 模拟3G网络 - 使用 launchOptions 而不是 contextOptions
        launchOptions: {
          args: ['--force-device-scale-factor=1']
        }
      },
    }
  ],

  // 测试服务器配置
  webServer: {
    command: 'npm run build && npm run preview',
    port: 3000,
    reuseExistingServer: !process.env.CI,
    timeout: 120000,
    env: {
      NODE_ENV: 'test',
      // Force root base during E2E to avoid GitHub Pages base path
      VITEPRESS_BASE: '/'
    }
  },

  // 全局设置和清理
  globalSetup: require.resolve('./tests/e2e/global-setup.ts'),
  globalTeardown: require.resolve('./tests/e2e/global-teardown.ts'),

  // 输出目录
  outputDir: 'tests/e2e-results/',

  // 期望配置
  expect: {
    // 断言超时
    timeout: 10000,
    
    // 屏幕截图配置
    toHaveScreenshot: {
      animations: 'disabled',
      caret: 'hide'
    },
    
    // 可视化比较配置
    toMatchSnapshot: {
      threshold: 0.3,
      maxDiffPixels: 1000
    }
  },

  // 测试超时
  timeout: 60000,

  // 元数据
  metadata: {
    'Test Environment': process.env.NODE_ENV || 'development',
    'Base URL': 'http://localhost:3000',
    'Test Framework': 'Playwright',
    'CI': process.env.CI ? 'true' : 'false'
  }
})