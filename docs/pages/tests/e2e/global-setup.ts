import { chromium, FullConfig } from '@playwright/test'
import { execSync } from 'child_process'
import { existsSync, mkdirSync } from 'fs'
import path from 'path'

async function globalSetup(config: FullConfig) {
  console.log('🚀 Starting global setup for E2E tests...')

  // 确保报告目录存在
  const reportsDir = path.join(process.cwd(), 'tests/reports')
  if (!existsSync(reportsDir)) {
    mkdirSync(reportsDir, { recursive: true })
  }

  // 确保 E2E 结果目录存在
  const e2eResultsDir = path.join(process.cwd(), 'tests/e2e-results')
  if (!existsSync(e2eResultsDir)) {
    mkdirSync(e2eResultsDir, { recursive: true })
  }

  // 确保静态资源准备就绪
  try {
    console.log('📦 Building project for E2E tests...')
    execSync('npm run build', { 
      stdio: 'inherit',
      timeout: 300000 // 5分钟超时
    })
    console.log('✅ Build completed successfully')
  } catch (error) {
    console.error('❌ Build failed:', error)
    process.exit(1)
  }

  // 预热服务器 - 确保第一个测试运行时服务器已就绪
  const browser = await chromium.launch()
  const page = await browser.newPage()
  
  try {
    console.log('🔥 Warming up the server...')
    
    // 等待服务器启动
    let retries = 0
    const maxRetries = 30
    
    while (retries < maxRetries) {
      try {
        await page.goto('http://localhost:3000', { 
          waitUntil: 'networkidle',
          timeout: 10000 
        })
        console.log('✅ Server is ready!')
        break
      } catch (error) {
        retries++
        if (retries === maxRetries) {
          throw new Error(`Server failed to start after ${maxRetries} attempts`)
        }
        console.log(`⏳ Waiting for server... (attempt ${retries}/${maxRetries})`)
        await new Promise(resolve => setTimeout(resolve, 2000))
      }
    }

    // 预加载关键页面以提高测试性能
    const criticalPages = [
      '/',
      '/en/',
      '/zh/',
      '/en/guide/getting-started',
      '/en/api/',
      '/en/examples/'
    ]

    console.log('🔄 Preloading critical pages...')
    for (const pagePath of criticalPages) {
      try {
        await page.goto(`http://localhost:3000${pagePath}`, { 
          waitUntil: 'networkidle',
          timeout: 15000 
        })
        console.log(`✅ Preloaded: ${pagePath}`)
      } catch (error) {
        console.warn(`⚠️  Failed to preload ${pagePath}:`, error.message)
      }
    }

    // 检查关键功能是否可用
    console.log('🔍 Performing health checks...')
    
    // 检查首页是否正常加载
    await page.goto('http://localhost:3000')
    const title = await page.title()
    if (!title.includes('Swit')) {
      throw new Error('Homepage title check failed')
    }
    
    // 检查导航是否工作
    const navLinks = await page.locator('nav a').count()
    if (navLinks === 0) {
      console.warn('⚠️  No navigation links found')
    }
    
    // 检查多语言切换是否可用
    try {
      await page.goto('http://localhost:3000/zh/')
      const chineseTitle = await page.title()
      if (!chineseTitle.includes('Swit') && !chineseTitle.includes('框架')) {
        console.warn('⚠️  Chinese language version may have issues')
      }
    } catch (error) {
      console.warn('⚠️  Chinese language version not accessible:', error.message)
    }

    console.log('✅ Health checks completed')

  } catch (error) {
    console.error('❌ Global setup failed:', error)
    throw error
  } finally {
    await browser.close()
  }

  // 设置测试环境变量
  process.env.E2E_SETUP_COMPLETED = 'true'
  process.env.E2E_BASE_URL = 'http://localhost:3000'
  
  // 记录设置完成时间
  const setupTime = new Date().toISOString()
  console.log(`✅ Global setup completed at ${setupTime}`)
  
  // 可选：保存设置信息到文件
  const setupInfo = {
    timestamp: setupTime,
    baseUrl: 'http://localhost:3000',
    nodeEnv: process.env.NODE_ENV,
    ci: !!process.env.CI
  }
  
  try {
    const fs = require('fs')
    fs.writeFileSync(
      path.join(reportsDir, 'setup-info.json'), 
      JSON.stringify(setupInfo, null, 2)
    )
  } catch (error) {
    console.warn('⚠️  Failed to save setup info:', error.message)
  }
}

export default globalSetup