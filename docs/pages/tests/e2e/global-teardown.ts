import { FullConfig } from '@playwright/test'
import { execSync } from 'child_process'
import path from 'path'
import { existsSync, writeFileSync } from 'fs'

async function globalTeardown(config: FullConfig) {
  console.log('🧹 Starting global teardown for E2E tests...')

  try {
    // 生成测试报告摘要
    console.log('📊 Generating test summary...')
    
    const reportsDir = path.join(process.cwd(), 'tests/reports')
    const summaryPath = path.join(reportsDir, 'test-summary.json')
    
    const summary = {
      timestamp: new Date().toISOString(),
      environment: {
        nodeEnv: process.env.NODE_ENV,
        ci: !!process.env.CI,
        baseUrl: process.env.E2E_BASE_URL
      },
      teardownCompleted: true
    }

    // 尝试读取 Playwright 测试结果
    const playwrightResultsPath = path.join(reportsDir, 'playwright-results.json')
    if (existsSync(playwrightResultsPath)) {
      try {
        const fs = require('fs')
        const results = JSON.parse(fs.readFileSync(playwrightResultsPath, 'utf8'))
        
        summary.testResults = {
          total: results.suites?.reduce((total: number, suite: any) => 
            total + (suite.specs?.length || 0), 0) || 0,
          passed: results.suites?.reduce((passed: number, suite: any) => 
            passed + (suite.specs?.filter((spec: any) => spec.ok).length || 0), 0) || 0,
          failed: results.suites?.reduce((failed: number, suite: any) => 
            failed + (suite.specs?.filter((spec: any) => !spec.ok).length || 0), 0) || 0,
          duration: results.stats?.duration || 0
        }
        
        console.log(`📈 Test Results: ${summary.testResults.passed}/${summary.testResults.total} passed`)
      } catch (error) {
        console.warn('⚠️  Failed to parse test results:', error.message)
      }
    }

    writeFileSync(summaryPath, JSON.stringify(summary, null, 2))
    console.log('✅ Test summary generated')

    // 清理临时文件
    console.log('🗂️  Cleaning up temporary files...')
    
    const tempDirs = [
      'tests/e2e-results/test-results',
      '.vitepress/cache'
    ]

    tempDirs.forEach(dir => {
      const fullPath = path.join(process.cwd(), dir)
      if (existsSync(fullPath)) {
        try {
          execSync(`rm -rf "${fullPath}"`, { timeout: 30000 })
          console.log(`✅ Cleaned: ${dir}`)
        } catch (error) {
          console.warn(`⚠️  Failed to clean ${dir}:`, error.message)
        }
      }
    })

    // 生成最终报告（如果在 CI 环境中）
    if (process.env.CI) {
      console.log('📋 Generating CI report...')
      
      try {
        // 压缩测试报告
        const reportArchive = path.join(reportsDir, 'test-reports.tar.gz')
        execSync(`tar -czf "${reportArchive}" -C tests/reports .`, { timeout: 60000 })
        console.log(`✅ Test reports archived: ${reportArchive}`)
      } catch (error) {
        console.warn('⚠️  Failed to archive reports:', error.message)
      }

      // 输出测试统计到 GitHub Actions
      if (summary.testResults) {
        const { total, passed, failed, duration } = summary.testResults
        console.log(`::notice title=Test Results::${passed}/${total} tests passed (${failed} failed) in ${Math.round(duration / 1000)}s`)
        
        if (failed > 0) {
          console.log(`::warning title=Test Failures::${failed} tests failed`)
        }
      }
    }

    // 性能监控清理
    console.log('🔍 Finalizing performance monitoring...')
    
    // 如果有性能监控数据，进行清理和汇总
    const perfDataPath = path.join(reportsDir, 'performance-data.json')
    if (existsSync(perfDataPath)) {
      try {
        const fs = require('fs')
        const perfData = JSON.parse(fs.readFileSync(perfDataPath, 'utf8'))
        
        // 计算性能统计
        const perfSummary = {
          timestamp: new Date().toISOString(),
          averageLoadTime: perfData.reduce((sum: number, item: any) => sum + item.loadTime, 0) / perfData.length,
          slowestPage: perfData.reduce((slowest: any, item: any) => 
            item.loadTime > (slowest?.loadTime || 0) ? item : slowest, null),
          totalPages: perfData.length
        }
        
        writeFileSync(
          path.join(reportsDir, 'performance-summary.json'),
          JSON.stringify(perfSummary, null, 2)
        )
        
        console.log(`📊 Performance Summary: Average load time ${Math.round(perfSummary.averageLoadTime)}ms`)
      } catch (error) {
        console.warn('⚠️  Failed to process performance data:', error.message)
      }
    }

    // 可访问性报告清理
    const a11yReportsPath = path.join(reportsDir, 'accessibility')
    if (existsSync(a11yReportsPath)) {
      try {
        const fs = require('fs')
        const files = fs.readdirSync(a11yReportsPath)
        const violations = files.reduce((total: number, file: string) => {
          try {
            const data = JSON.parse(fs.readFileSync(path.join(a11yReportsPath, file), 'utf8'))
            return total + (data.violations?.length || 0)
          } catch {
            return total
          }
        }, 0)
        
        console.log(`♿ Accessibility Summary: ${violations} violations found across ${files.length} pages`)
        
        if (violations > 0) {
          console.log(`::warning title=Accessibility Issues::${violations} accessibility violations found`)
        }
      } catch (error) {
        console.warn('⚠️  Failed to process accessibility reports:', error.message)
      }
    }

    // 清理环境变量
    delete process.env.E2E_SETUP_COMPLETED
    delete process.env.E2E_BASE_URL

    console.log('✅ Global teardown completed successfully')

  } catch (error) {
    console.error('❌ Global teardown failed:', error)
    
    // 即使清理失败，也要尝试输出基本信息
    const errorSummary = {
      timestamp: new Date().toISOString(),
      teardownError: error.message,
      environment: {
        nodeEnv: process.env.NODE_ENV,
        ci: !!process.env.CI
      }
    }

    try {
      const reportsDir = path.join(process.cwd(), 'tests/reports')
      writeFileSync(
        path.join(reportsDir, 'teardown-error.json'),
        JSON.stringify(errorSummary, null, 2)
      )
    } catch (writeError) {
      console.error('❌ Failed to write error summary:', writeError.message)
    }

    // 不抛出错误，避免影响测试结果
    console.log('⚠️  Teardown completed with errors, but continuing...')
  }
}

export default globalTeardown