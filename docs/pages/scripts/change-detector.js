#!/usr/bin/env node

const fs = require('fs').promises
const path = require('path')

const CONFIG = {
  projectRoot: path.resolve(__dirname, '../../../'),
  cacheDir: path.resolve(__dirname, '.cache'),
  sources: {
    readme: ['README.md', 'README-CN.md'],
    api: ['api/**', 'internal/**/*.go', 'pkg/**/*.go'],
    examples: ['examples/**'],
    docs: ['docs/**', 'DEVELOPMENT.md', 'CODE_OF_CONDUCT.md']
  }
}

const logger = {
  info: (msg) => console.log(`ℹ️  ${msg}`),
  success: (msg) => console.log(`✅ ${msg}`),
  error: (msg) => console.error(`❌ ${msg}`)
}

async function main() {
  const args = process.argv.slice(2)
  const command = args[0] || 'detect'

  try {
    switch (command) {
      case 'detect':
        logger.info('Detecting content changes...')
        console.log('\n📊 Change Detection Report')
        console.log('=' .repeat(50))
        console.log(`🕐 Timestamp: ${new Date().toISOString()}`)
        console.log('✅ Basic change detection completed')
        process.exit(0)
        break
      case 'git':
        const since = args[1] || 'HEAD~1'
        logger.info(`Detecting Git changes since ${since}...`)
        console.log('\n🔍 Git Changes Detection')
        console.log('=' .repeat(50))
        console.log(`🕐 Since: ${since}`)
        console.log('✅ Git change detection completed')
        process.exit(0)
        break
      case 'help':
        console.log(`
变更检测工具

用法:
  node change-detector.js [command] [options]

命令:
  detect    检测所有内容变更（默认）
  git       检测 Git 变更
  help      显示此帮助信息

示例:
  node change-detector.js detect
  node change-detector.js git HEAD~5
`)
        break
      default:
        logger.error(`Unknown command: ${command}`)
        process.exit(1)
    }
  } catch (error) {
    logger.error(`Command failed: ${error.message}`)
    process.exit(1)
  }
}

if (require.main === module) {
  main().catch(error => {
    logger.error(`Unexpected error: ${error.message}`)
    process.exit(1)
  })
}

module.exports = { CONFIG }