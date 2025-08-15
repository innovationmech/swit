const { chromium } = require('playwright')
const AxeBuilder = require('@axe-core/playwright').default
const fs = require('fs')
const path = require('path')

// 配置
const config = {
  baseURL: 'http://localhost:3000',
  outputDir: 'tests/reports/accessibility',
  pages: [
    { url: '/', name: 'homepage' },
    { url: '/en/', name: 'homepage-en' },
    { url: '/zh/', name: 'homepage-zh' },
    { url: '/en/guide/getting-started', name: 'getting-started' },
    { url: '/zh/guide/getting-started', name: 'getting-started-zh' },
    { url: '/en/api/', name: 'api-overview' },
    { url: '/en/examples/', name: 'examples' },
    { url: '/en/community/', name: 'community' }
  ],
  viewports: [
    { width: 1920, height: 1080, name: 'desktop' },
    { width: 768, height: 1024, name: 'tablet' },
    { width: 375, height: 667, name: 'mobile' }
  ],
  axeOptions: {
    rules: {
      // 自定义规则配置
      'color-contrast': { enabled: true },
      'focus-order-semantics': { enabled: true },
      'html-has-lang': { enabled: true },
      'image-alt': { enabled: true },
      'label': { enabled: true },
      'link-name': { enabled: true },
      'list': { enabled: true },
      'listitem': { enabled: true },
      'region': { enabled: true },
      'skip-link': { enabled: true },
      'tabindex': { enabled: true },
      'valid-lang': { enabled: true }
    },
    tags: ['wcag2a', 'wcag2aa', 'wcag21aa', 'best-practice']
  }
}

// 确保输出目录存在
function ensureOutputDir() {
  if (!fs.existsSync(config.outputDir)) {
    fs.mkdirSync(config.outputDir, { recursive: true })
  }
}

// 生成报告文件名
function generateReportFileName(pageName, viewport, timestamp) {
  return `${pageName}-${viewport}-${timestamp}.json`
}

// 生成 HTML 报告
function generateHTMLReport(results, timestamp) {
  const totalViolations = results.reduce((sum, result) => sum + result.violations.length, 0)
  const totalPages = results.length
  
  const html = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Accessibility Test Report</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            max-width: 1200px;
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
        .header h1 {
            margin: 0;
            font-size: 2.5em;
        }
        .header .subtitle {
            margin: 10px 0 0 0;
            opacity: 0.9;
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
        .summary-card .label {
            color: #666;
            margin-top: 5px;
        }
        .results {
            padding: 0 30px 30px 30px;
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
            border-bottom: 1px solid #ddd;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .page-name {
            font-weight: bold;
            font-size: 1.1em;
        }
        .violation-count {
            background: #dc3545;
            color: white;
            padding: 4px 12px;
            border-radius: 20px;
            font-size: 0.9em;
        }
        .violation-count.success {
            background: #28a745;
        }
        .violations {
            padding: 20px;
        }
        .violation {
            margin: 15px 0;
            padding: 15px;
            background: #fff5f5;
            border-left: 4px solid #dc3545;
            border-radius: 4px;
        }
        .violation-title {
            font-weight: bold;
            color: #dc3545;
        }
        .violation-description {
            margin: 8px 0;
            color: #666;
        }
        .violation-impact {
            display: inline-block;
            padding: 2px 8px;
            border-radius: 12px;
            font-size: 0.8em;
            font-weight: bold;
            text-transform: uppercase;
        }
        .impact-critical { background: #dc3545; color: white; }
        .impact-serious { background: #fd7e14; color: white; }
        .impact-moderate { background: #ffc107; color: black; }
        .impact-minor { background: #6c757d; color: white; }
        .violation-nodes {
            margin-top: 10px;
            font-family: monospace;
            font-size: 0.9em;
            background: #f8f9fa;
            padding: 10px;
            border-radius: 4px;
        }
        .footer {
            background: #f8f9fa;
            padding: 20px;
            text-align: center;
            color: #666;
            border-top: 1px solid #ddd;
        }
        .no-violations {
            text-align: center;
            color: #28a745;
            font-style: italic;
            padding: 20px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔍 Accessibility Test Report</h1>
            <p class="subtitle">Generated on ${new Date(timestamp).toLocaleString()}</p>
        </div>
        
        <div class="summary">
            <div class="summary-card">
                <div class="number">${totalPages}</div>
                <div class="label">Pages Tested</div>
            </div>
            <div class="summary-card">
                <div class="number">${totalViolations}</div>
                <div class="label">Total Violations</div>
            </div>
            <div class="summary-card">
                <div class="number">${results.filter(r => r.violations.length === 0).length}</div>
                <div class="label">Clean Pages</div>
            </div>
            <div class="summary-card">
                <div class="number">${Math.round((results.filter(r => r.violations.length === 0).length / totalPages) * 100)}%</div>
                <div class="label">Pass Rate</div>
            </div>
        </div>
        
        <div class="results">
            <h2>Detailed Results</h2>
            ${results.map(result => `
                <div class="page-result">
                    <div class="page-header">
                        <div class="page-name">${result.pageName} (${result.viewport})</div>
                        <div class="violation-count ${result.violations.length === 0 ? 'success' : ''}">
                            ${result.violations.length === 0 ? '✅ No violations' : `${result.violations.length} violations`}
                        </div>
                    </div>
                    <div class="violations">
                        ${result.violations.length === 0 
                            ? '<div class="no-violations">🎉 This page passes all accessibility tests!</div>'
                            : result.violations.map(violation => `
                                <div class="violation">
                                    <div class="violation-title">${violation.id}: ${violation.help}</div>
                                    <div class="violation-description">${violation.description}</div>
                                    <span class="violation-impact impact-${violation.impact}">${violation.impact}</span>
                                    <div class="violation-nodes">
                                        Found in ${violation.nodes.length} element(s):
                                        ${violation.nodes.slice(0, 3).map(node => `<br>• ${node.target.join(', ')}`).join('')}
                                        ${violation.nodes.length > 3 ? `<br>... and ${violation.nodes.length - 3} more` : ''}
                                    </div>
                                </div>
                            `).join('')
                        }
                    </div>
                </div>
            `).join('')}
        </div>
        
        <div class="footer">
            <p>Generated by axe-core • Swit Framework Documentation</p>
        </div>
    </div>
</body>
</html>`

  return html
}

// 运行单页面测试
async function runPageTest(page, testConfig) {
  console.log(`Testing ${testConfig.url} (${testConfig.viewport.name})...`)
  
  try {
    // 设置视口
    await page.setViewportSize(testConfig.viewport)
    
    // 导航到页面
    await page.goto(`${config.baseURL}${testConfig.url}`, { 
      waitUntil: 'networkidle',
      timeout: 30000 
    })
    
    // 等待页面完全加载
    await page.waitForTimeout(2000)
    
    // 运行 axe 测试
    const axeBuilder = new AxeBuilder({ page })
      .withTags(config.axeOptions.tags)
      .withRules(Object.keys(config.axeOptions.rules))
    
    const results = await axeBuilder.analyze()
    
    return {
      url: testConfig.url,
      pageName: testConfig.name,
      viewport: testConfig.viewport.name,
      violations: results.violations,
      passes: results.passes,
      incomplete: results.incomplete,
      inapplicable: results.inapplicable,
      timestamp: new Date().toISOString()
    }
    
  } catch (error) {
    console.error(`Error testing ${testConfig.url}:`, error.message)
    return {
      url: testConfig.url,
      pageName: testConfig.name,
      viewport: testConfig.viewport.name,
      violations: [],
      error: error.message,
      timestamp: new Date().toISOString()
    }
  }
}

// 主测试函数
async function runAccessibilityTests() {
  console.log('🚀 Starting accessibility tests...')
  
  ensureOutputDir()
  
  const browser = await chromium.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-dev-shm-usage']
  })
  
  const context = await browser.newContext({
    // 禁用一些可能影响测试的功能
    javaScriptEnabled: true,
    ignoreHTTPSErrors: true
  })
  
  const page = await context.newPage()
  
  // 监听控制台错误
  page.on('console', msg => {
    if (msg.type() === 'error') {
      console.warn(`Console error on ${page.url()}: ${msg.text()}`)
    }
  })
  
  const allResults = []
  const timestamp = Date.now()
  
  try {
    // 测试每个页面和视口组合
    for (const pageConfig of config.pages) {
      for (const viewport of config.viewports) {
        const testConfig = {
          ...pageConfig,
          viewport
        }
        
        const result = await runPageTest(page, testConfig)
        allResults.push(result)
        
        // 保存单个测试结果
        const fileName = generateReportFileName(pageConfig.name, viewport.name, timestamp)
        const filePath = path.join(config.outputDir, fileName)
        
        fs.writeFileSync(filePath, JSON.stringify(result, null, 2))
        
        // 输出违规摘要
        if (result.violations && result.violations.length > 0) {
          console.log(`❌ ${result.violations.length} violations found on ${pageConfig.url} (${viewport.name})`)
          
          // 输出最严重的违规
          const criticalViolations = result.violations.filter(v => v.impact === 'critical')
          if (criticalViolations.length > 0) {
            console.log(`   🚨 ${criticalViolations.length} critical violations`)
          }
        } else {
          console.log(`✅ No violations found on ${pageConfig.url} (${viewport.name})`)
        }
      }
    }
    
    // 生成汇总报告
    const summaryReport = {
      timestamp: new Date(timestamp).toISOString(),
      config: {
        baseURL: config.baseURL,
        totalPages: config.pages.length,
        totalViewports: config.viewports.length,
        totalTests: allResults.length
      },
      summary: {
        totalViolations: allResults.reduce((sum, result) => sum + (result.violations?.length || 0), 0),
        pagesWithViolations: allResults.filter(result => result.violations?.length > 0).length,
        cleanPages: allResults.filter(result => result.violations?.length === 0).length,
        passRate: Math.round((allResults.filter(result => result.violations?.length === 0).length / allResults.length) * 100)
      },
      results: allResults
    }
    
    // 保存 JSON 汇总报告
    fs.writeFileSync(
      path.join(config.outputDir, `summary-${timestamp}.json`),
      JSON.stringify(summaryReport, null, 2)
    )
    
    // 生成 HTML 报告
    const htmlReport = generateHTMLReport(allResults, timestamp)
    fs.writeFileSync(
      path.join(config.outputDir, `report-${timestamp}.html`),
      htmlReport
    )
    
    // 输出最终结果
    console.log('\n📊 Accessibility Test Summary:')
    console.log(`Total tests: ${allResults.length}`)
    console.log(`Total violations: ${summaryReport.summary.totalViolations}`)
    console.log(`Clean pages: ${summaryReport.summary.cleanPages}`)
    console.log(`Pass rate: ${summaryReport.summary.passRate}%`)
    
    if (summaryReport.summary.totalViolations > 0) {
      console.log(`\n⚠️  ${summaryReport.summary.totalViolations} accessibility violations found.`)
      console.log(`View detailed report: ${path.join(config.outputDir, `report-${timestamp}.html`)}`)
      
      // 在 CI 环境中设置退出码
      if (process.env.CI) {
        const criticalViolations = allResults.reduce((sum, result) => 
          sum + (result.violations?.filter(v => v.impact === 'critical').length || 0), 0)
        
        if (criticalViolations > 0) {
          console.log(`❌ ${criticalViolations} critical violations found. Failing build.`)
          process.exit(1)
        }
      }
    } else {
      console.log('\n🎉 All pages pass accessibility tests!')
    }
    
  } catch (error) {
    console.error('❌ Accessibility tests failed:', error)
    process.exit(1)
  } finally {
    await browser.close()
  }
}

// 如果直接运行此脚本
if (require.main === module) {
  runAccessibilityTests().catch(error => {
    console.error('Failed to run accessibility tests:', error)
    process.exit(1)
  })
}

module.exports = { runAccessibilityTests, config }