#!/usr/bin/env node

/**
 * Content Validation Utility
 * 
 * Validates markdown content for consistency, quality, and VitePress compatibility
 */

const fs = require('fs').promises;
const path = require('path');

class ContentValidator {
  constructor() {
    this.rules = {
      // Markdown structure rules
      structure: [
        {
          name: 'has_frontmatter',
          description: '检查是否包含 VitePress frontmatter',
          check: (content) => content.startsWith('---'),
          severity: 'error'
        },
        {
          name: 'has_title',
          description: '检查是否包含主标题',
          check: (content) => /^#\s+.+$/m.test(content),
          severity: 'error'
        },
        {
          name: 'heading_hierarchy',
          description: '检查标题层级结构',
          check: (content) => {
            const headings = content.match(/^#{1,6}\s+.+$/gm) || [];
            let lastLevel = 0;
            return headings.every(heading => {
              const level = heading.match(/^#+/)[0].length;
              const valid = level <= lastLevel + 1;
              lastLevel = level;
              return valid;
            });
          },
          severity: 'warning'
        }
      ],
      
      // Link validation rules
      links: [
        {
          name: 'no_broken_internal_links',
          description: '检查内部链接是否有效',
          check: async (content, filePath) => {
            const internalLinks = content.match(/\[([^\]]+)\]\((?!https?:\/\/)([^)]+)\)/g) || [];
            const basePath = path.dirname(filePath);
            
            for (const linkMatch of internalLinks) {
              const linkPath = linkMatch.match(/\]\(([^)]+)\)/)[1];
              const fullPath = path.resolve(basePath, linkPath);
              
              try {
                await fs.access(fullPath);
              } catch (error) {
                return false;
              }
            }
            return true;
          },
          severity: 'warning'
        },
        {
          name: 'valid_external_links',
          description: '检查外部链接格式',
          check: (content) => {
            const externalLinks = content.match(/\[([^\]]+)\]\(https?:\/\/[^)]+\)/g) || [];
            return externalLinks.every(link => {
              const url = link.match(/\]\((https?:\/\/[^)]+)\)/)[1];
              try {
                new URL(url);
                return true;
              } catch {
                return false;
              }
            });
          },
          severity: 'warning'
        }
      ],
      
      // Content quality rules
      quality: [
        {
          name: 'no_empty_sections',
          description: '检查是否有空章节',
          check: (content) => {
            const sections = content.split(/^#{1,6}\s+.+$/m);
            return sections.every(section => section.trim().length > 0);
          },
          severity: 'warning'
        },
        {
          name: 'code_blocks_have_language',
          description: '检查代码块是否标注语言',
          check: (content) => {
            const codeBlocks = content.match(/```[\s\S]*?```/g) || [];
            return codeBlocks.every(block => {
              const firstLine = block.split('\n')[0];
              return firstLine.length > 3; // ```language
            });
          },
          severity: 'info'
        },
        {
          name: 'consistent_list_markers',
          description: '检查列表标记一致性',
          check: (content) => {
            const lines = content.split('\n');
            const listMarkers = new Set();
            
            for (const line of lines) {
              const match = line.match(/^(\s*)([-*+]|\d+\.)\s/);
              if (match) {
                const marker = match[2];
                listMarkers.add(marker.replace(/\d+/, 'N'));
              }
            }
            
            // Should have consistent markers within the same document
            return listMarkers.size <= 2; // Allow both bullet and numbered lists
          },
          severity: 'info'
        }
      ],
      
      // VitePress specific rules
      vitepress: [
        {
          name: 'valid_frontmatter',
          description: '检查 frontmatter 格式',
          check: (content) => {
            if (!content.startsWith('---')) return true; // No frontmatter is OK
            
            const frontmatterEnd = content.indexOf('---', 3);
            if (frontmatterEnd === -1) return false;
            
            const frontmatter = content.substring(3, frontmatterEnd);
            try {
              // Basic YAML validation
              return !frontmatter.includes('\t') && // No tabs
                     frontmatter.split('\n').every(line => 
                       line.trim() === '' || 
                       line.includes(':') || 
                       line.startsWith(' ') ||
                       line.startsWith('-')
                     );
            } catch {
              return false;
            }
          },
          severity: 'error'
        },
        {
          name: 'valid_vue_components',
          description: '检查 Vue 组件语法',
          check: (content) => {
            const vueComponents = content.match(/<[A-Z][a-zA-Z0-9]*[^>]*>/g) || [];
            return vueComponents.every(component => {
              // Check for properly closed tags or self-closing tags
              return component.endsWith('/>') || 
                     content.includes(component.replace('<', '</').replace(/\s.*/, '>'));
            });
          },
          severity: 'warning'
        }
      ]
    };
  }

  /**
   * Validate a single file
   */
  async validateFile(filePath) {
    try {
      const content = await fs.readFile(filePath, 'utf8');
      const results = {
        file: filePath,
        errors: [],
        warnings: [],
        info: [],
        passed: 0,
        total: 0
      };

      // Run all validation rules
      for (const [category, rules] of Object.entries(this.rules)) {
        for (const rule of rules) {
          results.total++;
          
          try {
            const isValid = typeof rule.check === 'function' 
              ? await rule.check(content, filePath)
              : rule.check(content);
              
            if (isValid) {
              results.passed++;
            } else {
              const issue = {
                rule: rule.name,
                description: rule.description,
                category,
                severity: rule.severity
              };
              
              switch (rule.severity) {
                case 'error':
                  results.errors.push(issue);
                  break;
                case 'warning':
                  results.warnings.push(issue);
                  break;
                case 'info':
                  results.info.push(issue);
                  break;
              }
            }
          } catch (error) {
            results.errors.push({
              rule: rule.name,
              description: `验证规则执行失败: ${error.message}`,
              category,
              severity: 'error'
            });
          }
        }
      }

      return results;
    } catch (error) {
      return {
        file: filePath,
        errors: [{
          rule: 'file_access',
          description: `无法读取文件: ${error.message}`,
          category: 'system',
          severity: 'error'
        }],
        warnings: [],
        info: [],
        passed: 0,
        total: 0
      };
    }
  }

  /**
   * Validate multiple files
   */
  async validateDirectory(dirPath, pattern = '**/*.md') {
    const glob = require('glob');
    const files = glob.sync(pattern, { cwd: dirPath });
    const results = [];

    for (const file of files) {
      const fullPath = path.join(dirPath, file);
      const result = await this.validateFile(fullPath);
      results.push(result);
    }

    return results;
  }

  /**
   * Generate validation report
   */
  generateReport(results) {
    const report = {
      timestamp: new Date().toISOString(),
      summary: {
        totalFiles: results.length,
        totalErrors: results.reduce((sum, r) => sum + r.errors.length, 0),
        totalWarnings: results.reduce((sum, r) => sum + r.warnings.length, 0),
        totalInfo: results.reduce((sum, r) => sum + r.info.length, 0),
        totalPassed: results.reduce((sum, r) => sum + r.passed, 0),
        totalChecks: results.reduce((sum, r) => sum + r.total, 0)
      },
      files: results
    };

    report.summary.passRate = report.summary.totalChecks > 0 
      ? Math.round((report.summary.totalPassed / report.summary.totalChecks) * 100)
      : 0;

    return report;
  }

  /**
   * Format report for console output
   */
  formatConsoleReport(report) {
    let output = '\n📋 内容验证报告\n';
    output += '='.repeat(50) + '\n\n';

    // Summary
    output += `📊 汇总统计:\n`;
    output += `   文件总数: ${report.summary.totalFiles}\n`;
    output += `   通过率: ${report.summary.passRate}% (${report.summary.totalPassed}/${report.summary.totalChecks})\n`;
    output += `   错误: ${report.summary.totalErrors}\n`;
    output += `   警告: ${report.summary.totalWarnings}\n`;
    output += `   提示: ${report.summary.totalInfo}\n\n`;

    // File details
    for (const fileResult of report.files) {
      const hasIssues = fileResult.errors.length > 0 || 
                       fileResult.warnings.length > 0 || 
                       fileResult.info.length > 0;

      if (hasIssues) {
        output += `📄 ${path.relative(process.cwd(), fileResult.file)}\n`;
        
        // Errors
        if (fileResult.errors.length > 0) {
          output += `   ❌ 错误 (${fileResult.errors.length}):\n`;
          fileResult.errors.forEach(error => {
            output += `      • ${error.description} [${error.rule}]\n`;
          });
        }
        
        // Warnings
        if (fileResult.warnings.length > 0) {
          output += `   ⚠️  警告 (${fileResult.warnings.length}):\n`;
          fileResult.warnings.forEach(warning => {
            output += `      • ${warning.description} [${warning.rule}]\n`;
          });
        }
        
        // Info
        if (fileResult.info.length > 0) {
          output += `   ℹ️  提示 (${fileResult.info.length}):\n`;
          fileResult.info.forEach(info => {
            output += `      • ${info.description} [${info.rule}]\n`;
          });
        }
        
        output += '\n';
      }
    }

    if (report.summary.totalErrors === 0 && 
        report.summary.totalWarnings === 0 && 
        report.summary.totalInfo === 0) {
      output += '✅ 所有文件验证通过，未发现问题！\n';
    }

    return output;
  }

  /**
   * Fix common issues automatically
   */
  async autoFix(filePath, issues) {
    const content = await fs.readFile(filePath, 'utf8');
    let fixed = content;
    const changes = [];

    for (const issue of issues) {
      switch (issue.rule) {
        case 'consistent_list_markers':
          // Convert all bullet lists to use '-'
          fixed = fixed.replace(/^(\s*)[*+](\s)/gm, '$1-$2');
          changes.push('统一列表标记为 "-"');
          break;
          
        case 'code_blocks_have_language':
          // Add 'text' language to unlabeled code blocks
          fixed = fixed.replace(/```\n/g, '```text\n');
          changes.push('为代码块添加语言标记');
          break;
          
        // Add more auto-fix rules as needed
      }
    }

    if (changes.length > 0) {
      await fs.writeFile(filePath, fixed);
      return changes;
    }

    return [];
  }
}

// CLI interface
if (require.main === module) {
  const args = process.argv.slice(2);
  const command = args[0] || 'validate';
  
  const validator = new ContentValidator();
  
  switch (command) {
    case 'validate': {
      const target = args[1] || 'docs/pages';
      validator.validateDirectory(target)
        .then(results => {
          const report = validator.generateReport(results);
          console.log(validator.formatConsoleReport(report));
          
          // Save detailed report
          const reportPath = 'validation-report.json';
          require('fs').writeFileSync(reportPath, JSON.stringify(report, null, 2));
          console.log(`\n📁 详细报告已保存: ${reportPath}`);
          
          // Exit with error code if there are errors
          process.exit(report.summary.totalErrors > 0 ? 1 : 0);
        })
        .catch(console.error);
      break;
    }
      
    case 'autofix': {
      const fixTarget = args[1];
      if (!fixTarget) {
        console.error('请指定要修复的文件路径');
        process.exit(1);
      }
      
      validator.validateFile(fixTarget)
        .then(async (result) => {
          const fixableIssues = [...result.warnings, ...result.info].filter(
            issue => ['consistent_list_markers', 'code_blocks_have_language'].includes(issue.rule)
          );
          
          if (fixableIssues.length > 0) {
            const changes = await validator.autoFix(fixTarget, fixableIssues);
            console.log(`✅ 文件已修复: ${fixTarget}`);
            console.log(`   修复项目: ${changes.join(', ')}`);
          } else {
            console.log('📄 未发现可自动修复的问题');
          }
        })
        .catch(console.error);
      break;
    }
      
    case 'help':
      console.log(`
内容验证工具

用法:
  node content-validator.js [command] [options]

命令:
  validate [path]     验证指定目录下的 markdown 文件（默认: docs/pages）
  autofix <file>      自动修复指定文件中的可修复问题
  help               显示此帮助信息

示例:
  node content-validator.js validate
  node content-validator.js validate docs/pages/zh
  node content-validator.js autofix docs/pages/zh/index.md
      `);
      break;
      
    default:
      console.error(`未知命令: ${command}`);
      console.error('使用 "node content-validator.js help" 查看可用命令');
      process.exit(1);
  }
}

module.exports = ContentValidator;