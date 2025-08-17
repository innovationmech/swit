const { chromium } = require('playwright')
const lighthouse = require('lighthouse')
const fs = require('fs')
const path = require('path')

// 配置
const config = {
  baseURL: 'http://localhost:3000',
  outputDir: 'tests/reports/performance',
  pages: [
    { url: '/', name: 'homepage', critical: true },
    { url: '/en/', name: 'homepage-en', critical: true },
    { url: '/zh/', name: 'homepage-zh', critical: true },
    { url: '/en/guide/getting-started', name: 'getting-started', critical: true },
    { url: '/en/api/', name: 'api-overview', critical: false },
    { url: '/en/examples/', name: 'examples', critical: false },
    { url: '/en/community/', name: 'community', critical: false }
  ],
  thresholds: {
    performance: 90,
    accessibility: 95,
    bestPractices: 90,
    seo: 95,
    // Core Web Vitals
    firstContentfulPaint: 1500,
    largestContentfulPaint: 2500,
    cumulativeLayoutShift: 0.1,
    totalBlockingTime: 300
  },
  lighthouseConfig: {
    extends: 'lighthouse:default',
    settings: {
      maxWaitForFcp: 15 * 1000,
      maxWaitForLoad: 45 * 1000,
      formFactor: 'desktop',
      throttling: {
        rttMs: 40,
        throughputKbps: 10 * 1024,
        cpuSlowdownMultiplier: 1,
        requestLatencyMs: 0,
        downloadThroughputKbps: 0,
        uploadThroughputKbps: 0
      },
      screenEmulation: {
        mobile: false,
        width: 1350,
        height: 940,
        deviceScaleFactor: 1,
        disabled: false
      },
      emulatedUserAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36'
    }
  }
}

// 确保输出目录存在
function ensureOutputDir() {
  if (!fs.existsSync(config.outputDir)) {
    fs.mkdirSync(config.outputDir, { recursive: true })
  }
}

// 运行自定义性能指标测试
async function runCustomPerformanceTest(page, url) {
  console.log(`Running custom performance test for ${url}...`)
  
  try {
    // 收集性能指标
    const metrics = await page.evaluate(() => {
      return new Promise((resolve) => {
        // 等待页面完全加载
        if (document.readyState === 'complete') {
          collectMetrics()
        } else {
          window.addEventListener('load', collectMetrics)
        }
        
        function collectMetrics() {
          const perfData = performance.getEntriesByType('navigation')[0]
          const paintEntries = performance.getEntriesByType('paint')
          
          const metrics = {
            // 导航时间
            domContentLoaded: perfData.domContentLoadedEventEnd - perfData.domContentLoadedEventStart,
            loadComplete: perfData.loadEventEnd - perfData.loadEventStart,
            
            // 网络时间
            dnsLookup: perfData.domainLookupEnd - perfData.domainLookupStart,
            tcpConnection: perfData.connectEnd - perfData.connectStart,
            serverResponse: perfData.responseStart - perfData.requestStart,
            
            // 渲染时间
            domInteractive: perfData.domInteractive - perfData.navigationStart,
            domComplete: perfData.domComplete - perfData.navigationStart,
            
            // 页面大小
            transferSize: perfData.transferSize || 0,
            encodedBodySize: perfData.encodedBodySize || 0,
            decodedBodySize: perfData.decodedBodySize || 0
          }
          
          // Paint 时间
          paintEntries.forEach(entry => {
            if (entry.name === 'first-paint') {
              metrics.firstPaint = entry.startTime
            } else if (entry.name === 'first-contentful-paint') {
              metrics.firstContentfulPaint = entry.startTime
            }
          })
          
          // Layout Shift (如果支持)
          if ('PerformanceObserver' in window) {
            try {
              let cumulativeLayoutShift = 0
              new PerformanceObserver((list) => {
                for (const entry of list.getEntries()) {
                  if (!entry.hadRecentInput) {
                    cumulativeLayoutShift += entry.value
                  }
                }
                metrics.cumulativeLayoutShift = cumulativeLayoutShift
              }).observe({ type: 'layout-shift', buffered: true })
            } catch (e) {
              // Layout Shift API 不支持
            }
          }
          
          // 资源计数
          const resources = performance.getEntriesByType('resource')
          metrics.resourceCount = resources.length
          
          const resourcesByType = {}
          resources.forEach(resource => {
            const type = resource.initiatorType || 'other'
            resourcesByType[type] = (resourcesByType[type] || 0) + 1
          })
          metrics.resourcesByType = resourcesByType
          
          setTimeout(() => resolve(metrics), 1000) // 等待 CLS 收集
        }
      })
    })
    
    return metrics
  } catch (error) {
    console.error(`Error collecting custom metrics for ${url}:`, error)
    return null
  }
}

// 运行 Lighthouse 测试
async function runLighthouseTest(url) {
  console.log(`Running Lighthouse test for ${url}...`)
  
  try {
    const { lhr } = await lighthouse(`${config.baseURL}${url}`, {
      port: 9222, // Chrome 调试端口
      output: 'json',
      logLevel: 'error'
    }, config.lighthouseConfig)
    
    return {
      url,
      scores: {
        performance: Math.round(lhr.categories.performance.score * 100),
        accessibility: Math.round(lhr.categories.accessibility.score * 100),
        bestPractices: Math.round(lhr.categories['best-practices'].score * 100),
        seo: Math.round(lhr.categories.seo.score * 100)
      },
      metrics: {
        firstContentfulPaint: lhr.audits['first-contentful-paint'].numericValue,
        largestContentfulPaint: lhr.audits['largest-contentful-paint'].numericValue,
        totalBlockingTime: lhr.audits['total-blocking-time'].numericValue,
        cumulativeLayoutShift: lhr.audits['cumulative-layout-shift'].numericValue,
        speedIndex: lhr.audits['speed-index'].numericValue
      },
      opportunities: lhr.audits,
      fullReport: lhr
    }
  } catch (error) {
    console.error(`Lighthouse test failed for ${url}:`, error.message)
    return null
  }
}

// 检查性能阈值
function checkThresholds(result) {
  const issues = []
  
  if (result.lighthouse) {
    const scores = result.lighthouse.scores
    const metrics = result.lighthouse.metrics
    
    // 检查 Lighthouse 分数
    if (scores.performance < config.thresholds.performance) {
      issues.push(`Performance score (${scores.performance}) below threshold (${config.thresholds.performance})`)
    }
    
    if (scores.accessibility < config.thresholds.accessibility) {
      issues.push(`Accessibility score (${scores.accessibility}) below threshold (${config.thresholds.accessibility})`)
    }
    
    if (scores.bestPractices < config.thresholds.bestPractices) {
      issues.push(`Best Practices score (${scores.bestPractices}) below threshold (${config.thresholds.bestPractices})`)
    }
    
    if (scores.seo < config.thresholds.seo) {
      issues.push(`SEO score (${scores.seo}) below threshold (${config.thresholds.seo})`)
    }
    
    // 检查 Core Web Vitals
    if (metrics.firstContentfulPaint > config.thresholds.firstContentfulPaint) {
      issues.push(`FCP (${Math.round(metrics.firstContentfulPaint)}ms) above threshold (${config.thresholds.firstContentfulPaint}ms)`)
    }
    
    if (metrics.largestContentfulPaint > config.thresholds.largestContentfulPaint) {
      issues.push(`LCP (${Math.round(metrics.largestContentfulPaint)}ms) above threshold (${config.thresholds.largestContentfulPaint}ms)`)
    }
    
    if (metrics.cumulativeLayoutShift > config.thresholds.cumulativeLayoutShift) {
      issues.push(`CLS (${metrics.cumulativeLayoutShift.toFixed(3)}) above threshold (${config.thresholds.cumulativeLayoutShift})`)
    }
    
    if (metrics.totalBlockingTime > config.thresholds.totalBlockingTime) {
      issues.push(`TBT (${Math.round(metrics.totalBlockingTime)}ms) above threshold (${config.thresholds.totalBlockingTime}ms)`)
    }
  }
  
  return issues
}

// 生成 HTML 报告
function generateHTMLReport(results, timestamp) {
  const html = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Performance Test Report</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #646cff, #747bff);
            color: white;
            padding: 30px;
            text-align: center;
        }
        .summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            padding: 30px;
            background: #f8f9fa;
        }
        .summary-card {
            background: white;
            padding: 20px;
            border-radius: 8px;
            text-align: center;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .summary-card .number {
            font-size: 2em;
            font-weight: bold;
            color: #646cff;
        }
        .results {
            padding: 30px;
        }
        .page-result {
            margin: 20px 0;
            border: 1px solid #ddd;
            border-radius: 8px;
            overflow: hidden;
        }
        .page-header {
            background: #f8f9fa;
            padding: 15px 20px;
            font-weight: bold;
        }
        .scores {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
            gap: 15px;
            padding: 20px;
        }
        .score-card {
            text-align: center;
            padding: 15px;
            border-radius: 6px;
        }
        .score-card.excellent { background: #d4edda; color: #155724; }
        .score-card.good { background: #d1ecf1; color: #0c5460; }
        .score-card.average { background: #fff3cd; color: #856404; }
        .score-card.poor { background: #f8d7da; color: #721c24; }
        .metrics {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
            padding: 20px;
            background: #f8f9fa;
        }
        .metric {
            background: white;
            padding: 15px;
            border-radius: 6px;
            text-align: center;
        }
        .metric-value {
            font-size: 1.5em;
            font-weight: bold;
            color: #646cff;
        }
        .metric-label {
            color: #666;
            font-size: 0.9em;
            margin-top: 5px;
        }
        .issues {
            padding: 20px;
            background: #fff5f5;
        }
        .issue {
            color: #dc3545;
            margin: 5px 0;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>⚡ Performance Test Report</h1>
            <p>Generated on ${new Date(timestamp).toLocaleString()}</p>
        </div>
        
        <div class="summary">
            <div class="summary-card">
                <div class="number">${results.length}</div>
                <div class="label">Pages Tested</div>
            </div>
            <div class="summary-card">
                <div class="number">${Math.round(results.reduce((sum, r) => sum + (r.lighthouse?.scores.performance || 0), 0) / results.length)}</div>
                <div class="label">Avg Performance</div>
            </div>
            <div class="summary-card">
                <div class="number">${Math.round(results.reduce((sum, r) => sum + (r.custom?.firstContentfulPaint || 0), 0) / results.length)}ms</div>
                <div class="label">Avg FCP</div>
            </div>
            <div class="summary-card">
                <div class="number">${results.filter(r => r.issues.length === 0).length}</div>
                <div class="label">Passing Pages</div>
            </div>
        </div>
        
        <div class="results">
            ${results.map(result => `
                <div class="page-result">
                    <div class="page-header">${result.pageName} - ${result.url}</div>
                    
                    ${result.lighthouse ? `
                        <div class="scores">
                            ${Object.entries(result.lighthouse.scores).map(([key, value]) => `
                                <div class="score-card ${getScoreClass(value)}">
                                    <div style="font-size: 1.8em; font-weight: bold;">${value}</div>
                                    <div style="font-size: 0.9em; text-transform: capitalize;">${key.replace(/([A-Z])/g, ' $1')}</div>
                                </div>
                            `).join('')}
                        </div>
                        
                        <div class="metrics">
                            <div class="metric">
                                <div class="metric-value">${Math.round(result.lighthouse.metrics.firstContentfulPaint)}ms</div>
                                <div class="metric-label">First Contentful Paint</div>
                            </div>
                            <div class="metric">
                                <div class="metric-value">${Math.round(result.lighthouse.metrics.largestContentfulPaint)}ms</div>
                                <div class="metric-label">Largest Contentful Paint</div>
                            </div>
                            <div class="metric">
                                <div class="metric-value">${result.lighthouse.metrics.cumulativeLayoutShift.toFixed(3)}</div>
                                <div class="metric-label">Cumulative Layout Shift</div>
                            </div>
                            <div class="metric">
                                <div class="metric-value">${Math.round(result.lighthouse.metrics.totalBlockingTime)}ms</div>
                                <div class="metric-label">Total Blocking Time</div>
                            </div>
                        </div>
                    ` : ''}
                    
                    ${result.issues.length > 0 ? `
                        <div class="issues">
                            <h4>⚠️ Performance Issues:</h4>
                            ${result.issues.map(issue => `<div class="issue">• ${issue}</div>`).join('')}
                        </div>
                    ` : '<div style="padding: 20px; text-align: center; color: #28a745;">✅ All performance thresholds met!</div>'}
                </div>
            `).join('')}
        </div>
    </div>
</body>
</html>`

  function getScoreClass(score) {
    if (score >= 90) return 'excellent'
    if (score >= 70) return 'good'
    if (score >= 50) return 'average'
    return 'poor'
  }

  return html
}

// 主测试函数
async function runPerformanceTests() {
  console.log('🚀 Starting performance tests...')
  
  ensureOutputDir()
  
  // 启动 Chrome 用于 Lighthouse
  const browser = await chromium.launch({
    headless: true,
    args: ['--remote-debugging-port=9222', '--no-sandbox', '--disable-dev-shm-usage']
  })
  
  const context = await browser.newContext()
  const page = await context.newPage()
  
  const results = []
  const timestamp = Date.now()
  
  try {
    for (const pageConfig of config.pages) {
      console.log(`\nTesting ${pageConfig.url}...`)
      
      // 导航到页面
      await page.goto(`${config.baseURL}${pageConfig.url}`, { 
        waitUntil: 'networkidle',
        timeout: 30000 
      })
      
      // 运行自定义性能测试
      const customMetrics = await runCustomPerformanceTest(page, pageConfig.url)
      
      // 运行 Lighthouse 测试
      const lighthouseResult = await runLighthouseTest(pageConfig.url)
      
      const result = {
        url: pageConfig.url,
        pageName: pageConfig.name,
        critical: pageConfig.critical,
        timestamp: new Date().toISOString(),
        custom: customMetrics,
        lighthouse: lighthouseResult
      }
      
      // 检查性能阈值
      result.issues = checkThresholds(result, pageConfig.name)
      
      results.push(result)
      
      // 输出结果摘要
      if (lighthouseResult) {
        console.log(`Performance: ${lighthouseResult.scores.performance}/100`)
        console.log(`FCP: ${Math.round(lighthouseResult.metrics.firstContentfulPaint)}ms`)
        console.log(`LCP: ${Math.round(lighthouseResult.metrics.largestContentfulPaint)}ms`)
      }
      
      if (result.issues.length > 0) {
        console.log(`⚠️  ${result.issues.length} performance issues found`)
        result.issues.forEach(issue => console.log(`   • ${issue}`))
      } else {
        console.log('✅ All thresholds met')
      }
      
      // 保存单个结果
      fs.writeFileSync(
        path.join(config.outputDir, `${pageConfig.name}-${timestamp}.json`),
        JSON.stringify(result, null, 2)
      )
    }
    
    // 生成汇总报告
    const summaryReport = {
      timestamp: new Date(timestamp).toISOString(),
      config: {
        baseURL: config.baseURL,
        totalPages: config.pages.length,
        thresholds: config.thresholds
      },
      summary: {
        totalIssues: results.reduce((sum, result) => sum + result.issues.length, 0),
        criticalPagesWithIssues: results.filter(r => r.critical && r.issues.length > 0).length,
        averagePerformanceScore: Math.round(results.reduce((sum, r) => sum + (r.lighthouse?.scores.performance || 0), 0) / results.length),
        averageFCP: Math.round(results.reduce((sum, r) => sum + (r.lighthouse?.metrics.firstContentfulPaint || 0), 0) / results.length)
      },
      results
    }
    
    // 保存 JSON 汇总
    fs.writeFileSync(
      path.join(config.outputDir, `summary-${timestamp}.json`),
      JSON.stringify(summaryReport, null, 2)
    )
    
    // 生成 HTML 报告
    const htmlReport = generateHTMLReport(results, timestamp)
    fs.writeFileSync(
      path.join(config.outputDir, `report-${timestamp}.html`),
      htmlReport
    )
    
    // 输出最终结果
    console.log('\n📊 Performance Test Summary:')
    console.log(`Total pages tested: ${results.length}`)
    console.log(`Total issues: ${summaryReport.summary.totalIssues}`)
    console.log(`Average performance score: ${summaryReport.summary.averagePerformanceScore}/100`)
    console.log(`Average FCP: ${summaryReport.summary.averageFCP}ms`)
    
    // 检查关键页面性能
    const criticalIssues = summaryReport.summary.criticalPagesWithIssues
    if (criticalIssues > 0) {
      console.log(`\n⚠️  ${criticalIssues} critical pages have performance issues.`)
      
      if (process.env.CI) {
        console.log('❌ Performance tests failed for critical pages.')
        process.exit(1)
      }
    } else {
      console.log('\n🎉 All critical pages meet performance thresholds!')
    }
    
  } catch (error) {
    console.error('❌ Performance tests failed:', error)
    process.exit(1)
  } finally {
    await browser.close()
  }
}

// 如果直接运行此脚本
if (require.main === module) {
  runPerformanceTests().catch(error => {
    console.error('Failed to run performance tests:', error)
    process.exit(1)
  })
}

module.exports = { runPerformanceTests, config }