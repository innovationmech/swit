#!/usr/bin/env node

/**
 * Documentation Synchronization Script
 * 
 * This script synchronizes content from the main project README files
 * to the GitHub Pages website, converting and formatting the content
 * for VitePress consumption.
 */

const fs = require('fs').promises;
const path = require('path');
const crypto = require('crypto');

class DocumentSynchronizer {
  constructor() {
    this.projectRoot = path.resolve(__dirname, '../../../');
    this.docsRoot = path.resolve(__dirname, '../');
    this.cacheDir = path.resolve(__dirname, '.cache');
    this.config = {
      sourceFiles: {
        en: path.join(this.projectRoot, 'README.md'),
        zh: path.join(this.projectRoot, 'README-CN.md')
      },
      targetFiles: {
        en: path.join(this.docsRoot, 'en/index.md'),
        zh: path.join(this.docsRoot, 'zh/index.md')
      },
      additionalSources: [
        path.join(this.projectRoot, 'DEVELOPMENT.md'),
        path.join(this.projectRoot, 'DEVELOPMENT-CN.md'),
        path.join(this.projectRoot, 'CODE_OF_CONDUCT.md'),
        path.join(this.projectRoot, 'SECURITY.md')
      ]
    };
  }

  /**
   * Initialize the synchronizer and ensure directories exist
   */
  async initialize() {
    try {
      await fs.mkdir(this.cacheDir, { recursive: true });
      console.log('✓ 缓存目录初始化完成');
    } catch (error) {
      console.warn('⚠ 缓存目录初始化警告:', error.message);
    }
  }

  /**
   * Calculate hash of file content for change detection
   */
  async calculateFileHash(filePath) {
    try {
      const content = await fs.readFile(filePath, 'utf8');
      return crypto.createHash('md5').update(content).digest('hex');
    } catch (error) {
      console.error(`计算文件哈希失败 ${filePath}:`, error.message);
      return null;
    }
  }

  /**
   * Check if file has changed since last sync
   */
  async hasFileChanged(filePath, lang) {
    const cacheFile = path.join(this.cacheDir, `${lang}-hash.txt`);
    const currentHash = await this.calculateFileHash(filePath);
    
    if (!currentHash) return false;

    try {
      const cachedHash = await fs.readFile(cacheFile, 'utf8');
      return currentHash !== cachedHash.trim();
    } catch (error) {
      // Cache file doesn't exist, consider it changed
      return true;
    }
  }

  /**
   * Save file hash to cache
   */
  async saveFileHash(filePath, lang) {
    const cacheFile = path.join(this.cacheDir, `${lang}-hash.txt`);
    const hash = await this.calculateFileHash(filePath);
    
    if (hash) {
      await fs.writeFile(cacheFile, hash);
    }
  }

  /**
   * Convert README content to VitePress format
   */
  convertToVitePress(content, lang) {
    let converted = content;

    // Remove or modify badges for web display
    converted = this.processBadges(converted, lang);

    // Add VitePress frontmatter
    const frontmatter = this.generateFrontmatter(lang);
    converted = frontmatter + '\n\n' + converted;

    // Convert relative links to absolute URLs
    converted = this.processLinks(converted, lang);

    // Process code blocks for better display
    converted = this.processCodeBlocks(converted);

    // Add web-specific sections
    converted = this.addWebSections(converted, lang);

    // Clean up GitHub-specific content
    converted = this.cleanGitHubSpecificContent(converted);

    return converted;
  }

  /**
   * Generate VitePress frontmatter
   */
  generateFrontmatter(lang) {
    const isZh = lang === 'zh';
    return `---
layout: home
title: ${isZh ? 'Swit Go 微服务框架' : 'Swit Go Microservice Framework'}
titleTemplate: ${isZh ? 'Go 微服务开发框架' : 'Go Microservice Development Framework'}

hero:
  name: "Swit"
  text: "${isZh ? 'Go 微服务框架' : 'Go Microservice Framework'}"
  tagline: ${isZh ? '生产就绪的微服务开发基础设施' : 'Production-ready microservice development foundation'}
  actions:
    - theme: brand
      text: ${isZh ? '快速开始' : 'Get Started'}
      link: /${lang}/guide/getting-started
    - theme: alt
      text: ${isZh ? '查看 API' : 'View API'}
      link: /${lang}/api/

features:
  - title: ${isZh ? '统一的服务器框架' : 'Unified Server Framework'}
    details: ${isZh ? '完整的服务器生命周期管理，包括传输协调和健康监控' : 'Complete server lifecycle management with transport coordination and health monitoring'}
    icon: 🚀
  - title: ${isZh ? '多传输层支持' : 'Multi-Transport Support'}
    details: ${isZh ? '无缝的 HTTP 和 gRPC 传输协调，支持可插拔架构' : 'Seamless HTTP and gRPC transport coordination with pluggable architecture'}
    icon: 🔄
  - title: ${isZh ? '依赖注入系统' : 'Dependency Injection'}
    details: ${isZh ? '基于工厂的依赖容器，支持自动生命周期管理' : 'Factory-based dependency container with automatic lifecycle management'}
    icon: 📦
  - title: ${isZh ? '性能监控' : 'Performance Monitoring'}
    details: ${isZh ? '内置指标收集和性能分析，支持阈值监控' : 'Built-in metrics collection and performance profiling with threshold monitoring'}
    icon: 📊
  - title: ${isZh ? '服务发现' : 'Service Discovery'}
    details: ${isZh ? '基于 Consul 的服务注册和健康检查集成' : 'Consul-based service registration with health check integration'}
    icon: 🔍
  - title: ${isZh ? '示例丰富' : 'Rich Examples'}
    details: ${isZh ? '完整的参考实现和最佳实践示例' : 'Complete reference implementations and best practice examples'}
    icon: 📚
---`;
  }

  /**
   * Process badge links for web display
   */
  processBadges(content, lang) {
    // Convert GitHub badges to a more web-friendly format
    let processed = content;

    // Extract badges section
    const badgeRegex = /\[!\[[^\]]+\]\([^)]+\)\]\([^)]+\)/g;
    const badges = content.match(badgeRegex) || [];

    if (badges.length > 0) {
      const isZh = lang === 'zh';
      const badgeSection = `
## ${isZh ? '项目状态' : 'Project Status'}

<div class="project-badges">

${badges.map(badge => `${badge}`).join('\n')}

</div>

<style>
.project-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 1rem 0;
}

.project-badges img {
  height: 20px;
}
</style>
`;

      // Replace the original badges with the formatted section
      processed = processed.replace(badgeRegex, '').trim();
      processed = processed.replace(/^# Swit\s*/, `# Swit\n${badgeSection}\n`);
    }

    return processed;
  }

  /**
   * Process links to make them web-friendly
   */
  processLinks(content, lang) {
    let processed = content;

    // Convert relative documentation links to absolute web links
    processed = processed.replace(
      /\[([^\]]+)\]\((?!https?:\/\/)([^)]+\.md)\)/g,
      `[$1](/${lang}/guide/$2)`
    );

    // Convert internal file links to GitHub links
    processed = processed.replace(
      /\[([^\]]+)\]\((?!https?:\/\/)(?!\/[^/])([^)]+)\)/g,
      '[$1](https://github.com/innovationmech/swit/blob/master/$2)'
    );

    return processed;
  }

  /**
   * Process code blocks for better web display
   */
  processCodeBlocks(content) {
    // Add copy buttons and line numbers to code blocks
    return content.replace(
      /```(\w+)?\n([\s\S]*?)```/g,
      (match, lang, code) => {
        const language = lang || 'text';
        return `\`\`\`${language} {1-10}\n${code}\`\`\``;
      }
    );
  }

  /**
   * Add web-specific sections
   */
  addWebSections(content, lang) {
    const isZh = lang === 'zh';
    
    // Add quick start section
    const quickStartSection = `
## ${isZh ? '快速开始' : 'Quick Start'}

${isZh ? '开始使用 Swit 框架构建您的第一个微服务：' : 'Get started with building your first microservice using Swit framework:'}

\`\`\`bash
# ${isZh ? '克隆项目' : 'Clone the repository'}
git clone https://github.com/innovationmech/swit.git
cd swit

# ${isZh ? '安装依赖' : 'Install dependencies'}
go mod tidy

# ${isZh ? '构建项目' : 'Build the project'}
make build

# ${isZh ? '运行示例服务' : 'Run example service'}
./bin/swit-serve
\`\`\`

::: tip ${isZh ? '提示' : 'Tip'}
${isZh ? '查看我们的' : 'Check out our'} [${isZh ? '详细指南' : 'detailed guide'}](/${lang}/guide/getting-started) ${isZh ? '获取更多信息。' : 'for more information.'}
:::

`;

    // Insert after the main description
    const insertAfter = isZh ? '## 框架特性' : '## Framework Features';
    content = content.replace(insertAfter, quickStartSection + insertAfter);

    return content;
  }

  /**
   * Clean up GitHub-specific content
   */
  cleanGitHubSpecificContent(content) {
    // Remove GitHub-specific sections that don't make sense on the website
    let cleaned = content;

    // Remove or modify contributing section
    cleaned = cleaned.replace(/## Contributing[\s\S]*?(?=##|$)/, '');

    return cleaned;
  }

  /**
   * Validate converted content
   */
  validateContent(content, lang) {
    const issues = [];

    // Check for broken links
    const internalLinks = content.match(/\[([^\]]+)\]\((?!https?:\/\/)([^)]+)\)/g);
    if (internalLinks) {
      issues.push(`发现可能的内部链接: ${internalLinks.length} 个`);
    }

    // Check for missing sections
    const requiredSections = lang === 'zh' 
      ? ['## 框架特性', '## 快速开始']
      : ['## Framework Features', '## Quick Start'];
    
    requiredSections.forEach(section => {
      if (!content.includes(section)) {
        issues.push(`缺少必需章节: ${section}`);
      }
    });

    // Check frontmatter
    if (!content.startsWith('---')) {
      issues.push('缺少 VitePress frontmatter');
    }

    return issues;
  }

  /**
   * Process additional documentation files
   */
  async processAdditionalDocs() {
    const results = [];

    for (const sourcePath of this.config.additionalSources) {
      try {
        const content = await fs.readFile(sourcePath, 'utf8');
        const filename = path.basename(sourcePath, '.md');
        const lang = filename.endsWith('-CN') ? 'zh' : 'en';
        const baseName = filename.replace('-CN', '');
        
        const targetDir = path.join(this.docsRoot, lang, 'community');
        await fs.mkdir(targetDir, { recursive: true });
        
        const targetPath = path.join(targetDir, `${baseName.toLowerCase()}.md`);
        
        // Simple conversion for additional docs
        let processed = content;
        processed = `---
title: ${baseName.replace(/[_-]/g, ' ')}
---

${processed}`;

        await fs.writeFile(targetPath, processed);
        results.push({ source: sourcePath, target: targetPath, status: 'synced' });
      } catch (error) {
        results.push({ source: sourcePath, status: 'error', error: error.message });
      }
    }

    return results;
  }

  /**
   * Main synchronization process
   */
  async sync() {
    console.log('🔄 开始文档同步...');
    
    await this.initialize();
    
    const results = {
      main: {},
      additional: [],
      timestamp: new Date().toISOString()
    };

    // Process main README files
    for (const [lang, sourcePath] of Object.entries(this.config.sourceFiles)) {
      try {
        console.log(`📄 处理 ${lang} 文档: ${sourcePath}`);

        // Check if file has changed
        const hasChanged = await this.hasFileChanged(sourcePath, lang);
        if (!hasChanged) {
          console.log(`⏭️ ${lang} 文档未变更，跳过同步`);
          results.main[lang] = { status: 'skipped', reason: 'no_changes' };
          continue;
        }

        const content = await fs.readFile(sourcePath, 'utf8');
        const converted = this.convertToVitePress(content, lang);
        
        // Validate converted content
        const validationIssues = this.validateContent(converted, lang);
        if (validationIssues.length > 0) {
          console.warn(`⚠️ ${lang} 文档验证问题:`, validationIssues);
        }

        const targetPath = this.config.targetFiles[lang];
        const targetDir = path.dirname(targetPath);
        await fs.mkdir(targetDir, { recursive: true });
        
        await fs.writeFile(targetPath, converted);
        await this.saveFileHash(sourcePath, lang);
        
        results.main[lang] = {
          status: 'synced',
          source: sourcePath,
          target: targetPath,
          validationIssues
        };
        
        console.log(`✅ ${lang} 文档同步完成: ${targetPath}`);
        
      } catch (error) {
        console.error(`❌ ${lang} 文档同步失败:`, error.message);
        results.main[lang] = {
          status: 'error',
          error: error.message
        };
      }
    }

    // Process additional documentation files
    try {
      console.log('📚 处理附加文档...');
      results.additional = await this.processAdditionalDocs();
      console.log(`✅ 附加文档处理完成: ${results.additional.length} 个文件`);
    } catch (error) {
      console.error('❌ 附加文档处理失败:', error.message);
    }

    // Save sync report
    const reportPath = path.join(this.cacheDir, 'sync-report.json');
    await fs.writeFile(reportPath, JSON.stringify(results, null, 2));

    console.log('\n📋 同步报告:');
    console.log(`   主要文档: ${Object.keys(results.main).length} 个语言`);
    console.log(`   附加文档: ${results.additional.length} 个文件`);
    console.log(`   报告保存: ${reportPath}`);
    console.log('🎉 文档同步完成！');

    return results;
  }

  /**
   * Watch mode for continuous synchronization
   */
  async watch() {
    console.log('👀 启动文档监视模式...');
    
    const chokidar = require('chokidar');
    const watchPaths = [
      ...Object.values(this.config.sourceFiles),
      ...this.config.additionalSources
    ];

    const watcher = chokidar.watch(watchPaths, {
      persistent: true,
      ignoreInitial: false
    });

    watcher.on('change', async (path) => {
      console.log(`📝 检测到文件变更: ${path}`);
      await this.sync();
    });

    console.log(`✨ 监视 ${watchPaths.length} 个文件...`);
    console.log('按 Ctrl+C 退出监视模式');
  }
}

// CLI interface
if (require.main === module) {
  const args = process.argv.slice(2);
  const command = args[0] || 'sync';
  
  const synchronizer = new DocumentSynchronizer();
  
  switch (command) {
    case 'sync':
      synchronizer.sync().catch(console.error);
      break;
    case 'watch':
      synchronizer.watch().catch(console.error);
      break;
    case 'help':
      console.log(`
文档同步工具

用法:
  node sync-docs.js [command]

命令:
  sync     同步文档（默认）
  watch    监视文档变更并自动同步
  help     显示此帮助信息

示例:
  node sync-docs.js sync
  node sync-docs.js watch
      `);
      break;
    default:
      console.error(`未知命令: ${command}`);
      console.error('使用 "node sync-docs.js help" 查看可用命令');
      process.exit(1);
  }
}

module.exports = DocumentSynchronizer;