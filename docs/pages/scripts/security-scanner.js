#!/usr/bin/env node

/**
 * 安全漏洞扫描脚本
 * 扫描项目中的安全问题并生成报告
 */

const fs = require('fs').promises
const path = require('path')
const crypto = require('crypto')

// 扫描配置
const SCAN_CONFIG = {
  // 文件扫描规则
  filePatterns: {
    vue: /\.vue$/,
    js: /\.(js|mjs|ts)$/,
    json: /\.json$/,
    md: /\.md$/,
    html: /\.html$/
  },
  
  // 排除目录
  excludeDirs: [
    'node_modules',
    '.git',
    'dist',
    '.vitepress/cache',
    '.cache'
  ],
  
  // 安全规则
  securityRules: {
    // 敏感信息泄露
    sensitive: [
      {
        pattern: /password\s*[:=]\s*["']([^"']+)["']/gi,
        severity: 'high',
        type: 'credential',
        message: 'Hardcoded password detected'
      },
      {
        pattern: /api[_-]?key\s*[:=]\s*["']([^"']+)["']/gi,
        severity: 'high',
        type: 'credential',
        message: 'API key detected'
      },
      {
        pattern: /secret[_-]?key\s*[:=]\s*["']([^"']+)["']/gi,
        severity: 'high',
        type: 'credential',
        message: 'Secret key detected'
      },
      {
        pattern: /token\s*[:=]\s*["']([^"']+)["']/gi,
        severity: 'medium',
        type: 'credential',
        message: 'Token detected'
      },
      {
        pattern: /\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b/g,
        severity: 'low',
        type: 'pii',
        message: 'Email address detected'
      }
    ],
    
    // XSS 漏洞
    xss: [
      {
        pattern: /innerHTML\s*=\s*[^;]+\+[^;]*$/gm,
        severity: 'high',
        type: 'xss',
        message: 'Potential XSS via innerHTML concatenation'
      },
      {
        pattern: /document\.write\s*\(/gi,
        severity: 'high',
        type: 'xss',
        message: 'Dangerous document.write usage'
      },
      {
        pattern: /eval\s*\(/gi,
        severity: 'critical',
        type: 'xss',
        message: 'Dangerous eval usage'
      },
      {
        pattern: /v-html\s*=\s*["'][^"']*\{\{[^}]*\}\}[^"']*["']/gi,
        severity: 'medium',
        type: 'xss',
        message: 'Potential XSS via v-html directive'
      }
    ],
    
    // 不安全的依赖
    dependencies: [
      {
        pattern: /"lodash"\s*:\s*"[^4]/gi,
        severity: 'medium',
        type: 'dependency',
        message: 'Outdated lodash version (potential prototype pollution)'
      },
      {
        pattern: /"jquery"\s*:\s*"[^3]/gi,
        severity: 'medium',
        type: 'dependency',
        message: 'Outdated jQuery version'
      }
    ],
    
    // 不安全的配置
    config: [
      {
        pattern: /"secure"\s*:\s*false/gi,
        severity: 'medium',
        type: 'config',
        message: 'Insecure configuration detected'
      },
      {
        pattern: /"sameSite"\s*:\s*"none"/gi,
        severity: 'low',
        type: 'config',
        message: 'Potentially insecure SameSite cookie setting'
      }
    ]
  }
}

// 扫描结果
const scanResults = {
  summary: {
    filesScanned: 0,
    issues: 0,
    critical: 0,
    high: 0,
    medium: 0,
    low: 0
  },
  issues: [],
  recommendations: []
}

/**
 * 主扫描函数
 */
async function runSecurityScan() {
  console.log('🔍 Starting security scan...\n')
  
  const startTime = Date.now()
  const projectRoot = path.resolve(__dirname, '..')
  
  try {
    // 扫描文件
    await scanDirectory(projectRoot)
    
    // 分析依赖
    await analyzeDependencies(projectRoot)
    
    // 检查配置文件
    await checkConfiguration(projectRoot)
    
    // 生成报告
    const reportPath = await generateReport(projectRoot)
    
    const endTime = Date.now()
    const duration = ((endTime - startTime) / 1000).toFixed(2)
    
    // 输出结果
    printSummary(duration, reportPath)
    
    // 返回退出码
    return scanResults.summary.critical > 0 ? 2 : 
           scanResults.summary.high > 0 ? 1 : 0
    
  } catch (error) {
    console.error('❌ Security scan failed:', error.message)
    return 3
  }
}

/**
 * 扫描目录
 */
async function scanDirectory(dirPath) {
  const entries = await fs.readdir(dirPath, { withFileTypes: true })
  
  for (const entry of entries) {
    const fullPath = path.join(dirPath, entry.name)
    
    if (entry.isDirectory()) {
      // 跳过排除的目录
      if (SCAN_CONFIG.excludeDirs.includes(entry.name)) {
        continue
      }
      await scanDirectory(fullPath)
    } else if (entry.isFile()) {
      await scanFile(fullPath)
    }
  }
}

/**
 * 扫描单个文件
 */
async function scanFile(filePath) {
  const ext = path.extname(filePath)
  const fileName = path.basename(filePath)
  
  // 检查文件类型
  const shouldScan = Object.values(SCAN_CONFIG.filePatterns)
    .some(pattern => pattern.test(fileName))
  
  if (!shouldScan) return
  
  try {
    const content = await fs.readFile(filePath, 'utf-8')
    scanResults.summary.filesScanned++
    
    // 应用安全规则
    for (const [category, rules] of Object.entries(SCAN_CONFIG.securityRules)) {
      for (const rule of rules) {
        const matches = [...content.matchAll(rule.pattern)]
        
        for (const match of matches) {
          const lineNumber = getLineNumber(content, match.index)
          const issue = {
            file: path.relative(process.cwd(), filePath),
            line: lineNumber,
            severity: rule.severity,
            type: rule.type,
            category,
            message: rule.message,
            match: match[0].substring(0, 100), // 限制显示长度
            context: getContext(content, match.index)
          }
          
          scanResults.issues.push(issue)
          scanResults.summary.issues++
          scanResults.summary[rule.severity]++
        }
      }
    }
    
  } catch (error) {
    console.warn(`⚠️  Failed to scan file ${filePath}: ${error.message}`)
  }
}

/**
 * 分析依赖安全性
 */
async function analyzeDependencies(projectRoot) {
  const packageJsonPath = path.join(projectRoot, 'package.json')
  
  try {
    const packageJson = JSON.parse(await fs.readFile(packageJsonPath, 'utf-8'))
    const allDependencies = {
      ...packageJson.dependencies || {},
      ...packageJson.devDependencies || {}
    }
    
    // 检查已知的有漏洞的包
    const vulnerablePackages = {
      'lodash': {
        versions: ['<4.17.12'],
        issue: 'Prototype pollution vulnerability',
        severity: 'high'
      },
      'jquery': {
        versions: ['<3.5.0'],
        issue: 'XSS vulnerabilities',
        severity: 'medium'
      },
      'node-forge': {
        versions: ['<1.0.0'],
        issue: 'Various cryptographic vulnerabilities',
        severity: 'high'
      }
    }
    
    for (const [pkg, info] of Object.entries(vulnerablePackages)) {
      if (allDependencies[pkg]) {
        const version = allDependencies[pkg]
        scanResults.issues.push({
          file: 'package.json',
          line: 0,
          severity: info.severity,
          type: 'dependency',
          category: 'dependencies',
          message: `Potentially vulnerable package: ${pkg}@${version} - ${info.issue}`,
          match: `"${pkg}": "${version}"`,
          context: 'Package dependencies'
        })
        
        scanResults.summary.issues++
        scanResults.summary[info.severity]++
        
        scanResults.recommendations.push({
          type: 'dependency_update',
          package: pkg,
          currentVersion: version,
          recommendedAction: `Update ${pkg} to latest version`,
          severity: info.severity
        })
      }
    }
    
  } catch (error) {
    console.warn('⚠️  Could not analyze package.json:', error.message)
  }
}

/**
 * 检查配置安全性
 */
async function checkConfiguration(projectRoot) {
  const configFiles = [
    '.vitepress/config.mjs',
    'vite.config.js',
    'tsconfig.json'
  ]
  
  for (const configFile of configFiles) {
    const configPath = path.join(projectRoot, configFile)
    
    try {
      await fs.access(configPath)
      const content = await fs.readFile(configPath, 'utf-8')
      
      // 检查不安全的配置
      const insecureConfigs = [
        {
          pattern: /allowedOrigins.*\*.*\*/gi,
          message: 'Wildcard CORS configuration detected',
          severity: 'medium'
        },
        {
          pattern: /dangerouslySetInnerHTML/gi,
          message: 'Dangerous HTML injection possible',
          severity: 'high'
        }
      ]
      
      for (const config of insecureConfigs) {
        const matches = [...content.matchAll(config.pattern)]
        for (const match of matches) {
          const lineNumber = getLineNumber(content, match.index)
          scanResults.issues.push({
            file: configFile,
            line: lineNumber,
            severity: config.severity,
            type: 'config',
            category: 'config',
            message: config.message,
            match: match[0],
            context: getContext(content, match.index)
          })
          
          scanResults.summary.issues++
          scanResults.summary[config.severity]++
        }
      }
      
    } catch (error) {
      // 文件不存在，跳过
    }
  }
}

/**
 * 获取行号
 */
function getLineNumber(content, index) {
  return content.substring(0, index).split('\n').length
}

/**
 * 获取上下文
 */
function getContext(content, index) {
  const lines = content.split('\n')
  const lineIndex = getLineNumber(content, index) - 1
  const start = Math.max(0, lineIndex - 2)
  const end = Math.min(lines.length, lineIndex + 3)
  
  return lines.slice(start, end).map((line, i) => {
    const currentLine = start + i + 1
    const marker = currentLine === lineIndex + 1 ? '>>>' : '   '
    return `${marker} ${currentLine}: ${line}`
  }).join('\n')
}

/**
 * 生成安全报告
 */
async function generateReport(projectRoot) {
  const timestamp = new Date().toISOString()
  const reportPath = path.join(projectRoot, '.security-report.json')
  
  const report = {
    timestamp,
    summary: scanResults.summary,
    issues: scanResults.issues,
    recommendations: scanResults.recommendations,
    metadata: {
      scannerVersion: '1.0.0',
      rulesVersion: '1.0.0',
      projectRoot: projectRoot
    }
  }
  
  await fs.writeFile(reportPath, JSON.stringify(report, null, 2))
  
  // 生成 HTML 报告
  await generateHTMLReport(report, projectRoot)
  
  return reportPath
}

/**
 * 生成 HTML 报告
 */
async function generateHTMLReport(report, projectRoot) {
  const htmlPath = path.join(projectRoot, '.security-report.html')
  
  const html = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Security Scan Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f5f5f5; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .summary { display: flex; gap: 20px; margin-bottom: 20px; }
        .metric { background: white; padding: 15px; border-radius: 5px; border-left: 4px solid #007acc; }
        .critical { border-left-color: #d73027; }
        .high { border-left-color: #fc8d59; }
        .medium { border-left-color: #fee08b; }
        .low { border-left-color: #91bfdb; }
        .issue { background: #f9f9f9; margin: 10px 0; padding: 15px; border-radius: 5px; }
        .issue-header { display: flex; justify-content: space-between; margin-bottom: 10px; }
        .severity { padding: 3px 8px; border-radius: 3px; color: white; font-size: 12px; }
        .severity.critical { background: #d73027; }
        .severity.high { background: #fc8d59; }
        .severity.medium { background: #fee08b; color: #333; }
        .severity.low { background: #91bfdb; }
        .context { background: #f0f0f0; padding: 10px; font-family: monospace; font-size: 12px; margin-top: 10px; overflow-x: auto; }
        .recommendations { background: #e8f5e8; padding: 15px; border-radius: 5px; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Security Scan Report</h1>
        <p>Generated: ${report.timestamp}</p>
        <p>Scanner Version: ${report.metadata.scannerVersion}</p>
    </div>
    
    <div class="summary">
        <div class="metric">
            <h3>Files Scanned</h3>
            <p>${report.summary.filesScanned}</p>
        </div>
        <div class="metric critical">
            <h3>Critical Issues</h3>
            <p>${report.summary.critical}</p>
        </div>
        <div class="metric high">
            <h3>High Issues</h3>
            <p>${report.summary.high}</p>
        </div>
        <div class="metric medium">
            <h3>Medium Issues</h3>
            <p>${report.summary.medium}</p>
        </div>
        <div class="metric low">
            <h3>Low Issues</h3>
            <p>${report.summary.low}</p>
        </div>
    </div>
    
    <h2>Issues Found</h2>
    ${report.issues.map(issue => `
        <div class="issue">
            <div class="issue-header">
                <strong>${issue.file}:${issue.line}</strong>
                <span class="severity ${issue.severity}">${issue.severity.toUpperCase()}</span>
            </div>
            <p><strong>${issue.message}</strong></p>
            <p>Type: ${issue.type} | Category: ${issue.category}</p>
            <div class="context">${issue.context.replace(/\n/g, '<br>')}</div>
        </div>
    `).join('')}
    
    ${report.recommendations.length > 0 ? `
        <div class="recommendations">
            <h2>Recommendations</h2>
            <ul>
                ${report.recommendations.map(rec => `
                    <li><strong>${rec.recommendedAction}</strong> (${rec.severity})</li>
                `).join('')}
            </ul>
        </div>
    ` : ''}
</body>
</html>
`
  
  await fs.writeFile(htmlPath, html)
}

/**
 * 打印扫描结果摘要
 */
function printSummary(duration, reportPath) {
  const { summary } = scanResults
  
  console.log('\n📊 Security Scan Summary')
  console.log('========================\n')
  
  console.log(`📁 Files scanned: ${summary.filesScanned}`)
  console.log(`⚠️  Total issues: ${summary.issues}`)
  console.log(`🔴 Critical: ${summary.critical}`)
  console.log(`🟠 High: ${summary.high}`)
  console.log(`🟡 Medium: ${summary.medium}`)
  console.log(`🔵 Low: ${summary.low}`)
  console.log(`⏱️  Duration: ${duration}s`)
  
  if (summary.issues > 0) {
    console.log(`\n📄 Detailed report: ${reportPath}`)
    console.log(`🌐 HTML report: ${reportPath.replace('.json', '.html')}`)
    
    // 显示最严重的问题
    const criticalIssues = scanResults.issues.filter(i => i.severity === 'critical')
    const highIssues = scanResults.issues.filter(i => i.severity === 'high')
    
    if (criticalIssues.length > 0) {
      console.log('\n🚨 Critical Issues:')
      criticalIssues.slice(0, 3).forEach(issue => {
        console.log(`   ${issue.file}:${issue.line} - ${issue.message}`)
      })
    }
    
    if (highIssues.length > 0) {
      console.log('\n⚠️  High Priority Issues:')
      highIssues.slice(0, 3).forEach(issue => {
        console.log(`   ${issue.file}:${issue.line} - ${issue.message}`)
      })
    }
  } else {
    console.log('\n✅ No security issues found!')
  }
  
  // 显示建议
  if (scanResults.recommendations.length > 0) {
    console.log('\n💡 Recommendations:')
    scanResults.recommendations.slice(0, 5).forEach(rec => {
      console.log(`   • ${rec.recommendedAction}`)
    })
  }
}

// 运行扫描（如果直接执行此脚本）
if (require.main === module) {
  runSecurityScan()
    .then(exitCode => {
      process.exit(exitCode)
    })
    .catch(error => {
      console.error('Fatal error:', error)
      process.exit(3)
    })
}

module.exports = {
  runSecurityScan,
  SCAN_CONFIG,
  scanResults
}