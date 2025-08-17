#!/usr/bin/env node

/**
 * Integrated Content Management System
 * 
 * Combines documentation synchronization, change detection, and content validation
 * into a unified content management workflow
 */

const fs = require('fs').promises;
const path = require('path');
const DocumentSynchronizer = require('./sync-docs');
const ContentValidator = require('./content-validator');
const ChangeDetector = require('./change-detector');

class ContentManager {
  constructor() {
    this.synchronizer = new DocumentSynchronizer();
    this.validator = new ContentValidator();
    this.changeDetector = new ChangeDetector();
    
    this.config = {
      sourceFiles: [
        path.resolve(__dirname, '../../../README.md'),
        path.resolve(__dirname, '../../../README-CN.md'),
        path.resolve(__dirname, '../../../DEVELOPMENT.md'),
        path.resolve(__dirname, '../../../DEVELOPMENT-CN.md')
      ],
      targetDir: path.resolve(__dirname, '../'),
      quality: {
        minPassRate: 85, // Minimum pass rate for validation
        allowedErrors: 0,
        allowedWarnings: 5
      }
    };
  }

  /**
   * Initialize all components
   */
  async initialize() {
    console.log('🚀 初始化内容管理系统...');
    
    await Promise.all([
      this.synchronizer.initialize(),
      this.changeDetector.initialize()
    ]);
    
    console.log('✅ 初始化完成');
  }

  /**
   * Full content management workflow
   */
  async manage(options = {}) {
    const {
      force = false,
      validate = true,
      autofix = false,
      createSnapshots = true
    } = options;

    console.log('\n📋 开始内容管理流程...');
    
    const report = {
      timestamp: new Date().toISOString(),
      steps: [],
      overall: 'pending'
    };

    try {
      // Step 1: Change Detection
      report.steps.push(await this.detectChanges(force));
      
      // Step 2: Content Synchronization
      if (force || report.steps[0].hasChanges) {
        if (createSnapshots && !force) {
          report.steps.push(await this.createPreSyncSnapshots());
        }
        
        report.steps.push(await this.synchronizeContent());
        
        // Step 3: Content Validation
        if (validate) {
          report.steps.push(await this.validateContent(autofix));
        }
        
        // Step 4: Post-sync updates
        report.steps.push(await this.updateTrackingData());
      } else {
        console.log('⏭️ 未检测到变更，跳过同步');
        report.steps.push({
          name: 'sync',
          status: 'skipped',
          reason: 'no_changes'
        });
      }

      // Generate final report
      report.overall = this.evaluateOverallStatus(report.steps);
      
    } catch (error) {
      console.error('❌ 内容管理流程失败:', error.message);
      report.overall = 'failed';
      report.error = error.message;
    }

    // Save and display report
    await this.saveReport(report);
    this.displayReport(report);
    
    return report;
  }

  /**
   * Detect changes in source files
   */
  async detectChanges(force = false) {
    console.log('\n🔍 检测文件变更...');
    
    try {
      const results = await this.changeDetector.detectChanges(this.config.sourceFiles);
      const summary = await this.changeDetector.generateChangeSummary(results);
      
      const hasChanges = summary.new > 0 || summary.modified > 0;
      
      console.log(`   新文件: ${summary.new}`);
      console.log(`   已修改: ${summary.modified}`);
      console.log(`   未变更: ${summary.unchanged}`);
      
      if (hasChanges || force) {
        console.log('✅ 检测到变更，需要同步');
      } else {
        console.log('ℹ️ 未检测到变更');
      }
      
      return {
        name: 'change_detection',
        status: 'completed',
        hasChanges: hasChanges || force,
        summary
      };
    } catch (error) {
      console.error('❌ 变更检测失败:', error.message);
      return {
        name: 'change_detection',
        status: 'failed',
        error: error.message
      };
    }
  }

  /**
   * Create snapshots before synchronization
   */
  async createPreSyncSnapshots() {
    console.log('\n📸 创建同步前快照...');
    
    try {
      const snapshots = [];
      
      for (const file of this.config.sourceFiles) {
        try {
          const { snapshotPath } = await this.changeDetector.createSnapshot(file, {
            trigger: 'pre_sync',
            reason: 'Backup before content synchronization'
          });
          snapshots.push(snapshotPath);
        } catch (error) {
          console.warn(`⚠️ 快照创建失败 ${file}: ${error.message}`);
        }
      }
      
      console.log(`✅ 创建了 ${snapshots.length} 个快照`);
      
      return {
        name: 'snapshots',
        status: 'completed',
        snapshots
      };
    } catch (error) {
      console.error('❌ 快照创建失败:', error.message);
      return {
        name: 'snapshots',
        status: 'failed',
        error: error.message
      };
    }
  }

  /**
   * Synchronize content from source to target
   */
  async synchronizeContent() {
    console.log('\n🔄 同步文档内容...');
    
    try {
      const syncResults = await this.synchronizer.sync();
      
      const successCount = Object.values(syncResults.main)
        .filter(r => r.status === 'synced').length;
      const totalCount = Object.keys(syncResults.main).length;
      
      console.log(`✅ 文档同步完成: ${successCount}/${totalCount} 成功`);
      
      return {
        name: 'synchronization',
        status: successCount > 0 ? 'completed' : 'failed',
        results: syncResults
      };
    } catch (error) {
      console.error('❌ 文档同步失败:', error.message);
      return {
        name: 'synchronization',
        status: 'failed',
        error: error.message
      };
    }
  }

  /**
   * Validate synchronized content
   */
  async validateContent(autofix = false) {
    console.log('\n✅ 验证文档内容...');
    
    try {
      const results = await this.validator.validateDirectory(this.config.targetDir);
      const report = this.validator.generateReport(results);
      
      console.log(`   通过率: ${report.summary.passRate}%`);
      console.log(`   错误: ${report.summary.totalErrors}`);
      console.log(`   警告: ${report.summary.totalWarnings}`);
      
      // Check if quality standards are met
      const qualityMet = this.checkQualityStandards(report);
      
      if (!qualityMet.passed && autofix) {
        console.log('\n🔧 尝试自动修复问题...');
        await this.performAutoFix(results);
        
        // Re-validate after fixes
        const retestResults = await this.validator.validateDirectory(this.config.targetDir);
        const retestReport = this.validator.generateReport(retestResults);
        
        console.log(`   修复后通过率: ${retestReport.summary.passRate}%`);
        report.afterFix = retestReport;
      }
      
      return {
        name: 'validation',
        status: qualityMet.passed ? 'completed' : 'completed_with_issues',
        report,
        qualityCheck: qualityMet
      };
    } catch (error) {
      console.error('❌ 内容验证失败:', error.message);
      return {
        name: 'validation',
        status: 'failed',
        error: error.message
      };
    }
  }

  /**
   * Check if content meets quality standards
   */
  checkQualityStandards(report) {
    const { quality } = this.config;
    const issues = [];
    
    if (report.summary.passRate < quality.minPassRate) {
      issues.push(`通过率 ${report.summary.passRate}% 低于要求的 ${quality.minPassRate}%`);
    }
    
    if (report.summary.totalErrors > quality.allowedErrors) {
      issues.push(`错误数量 ${report.summary.totalErrors} 超过允许的 ${quality.allowedErrors}`);
    }
    
    if (report.summary.totalWarnings > quality.allowedWarnings) {
      issues.push(`警告数量 ${report.summary.totalWarnings} 超过允许的 ${quality.allowedWarnings}`);
    }
    
    return {
      passed: issues.length === 0,
      issues
    };
  }

  /**
   * Perform automatic fixes on validation issues
   */
  async performAutoFix(results) {
    let fixedCount = 0;
    
    for (const fileResult of results) {
      const fixableIssues = [...fileResult.warnings, ...fileResult.info].filter(
        issue => ['consistent_list_markers', 'code_blocks_have_language'].includes(issue.rule)
      );
      
      if (fixableIssues.length > 0) {
        try {
          const changes = await this.validator.autoFix(fileResult.file, fixableIssues);
          if (changes.length > 0) {
            console.log(`   🔧 ${path.relative(process.cwd(), fileResult.file)}: ${changes.join(', ')}`);
            fixedCount++;
          }
        } catch (error) {
          console.warn(`   ⚠️ 自动修复失败 ${fileResult.file}: ${error.message}`);
        }
      }
    }
    
    console.log(`✅ 自动修复了 ${fixedCount} 个文件`);
  }

  /**
   * Update tracking data after successful sync
   */
  async updateTrackingData() {
    console.log('\n💾 更新跟踪数据...');
    
    try {
      await this.changeDetector.updateChecksums(this.config.sourceFiles);
      
      await this.changeDetector.logChange({
        type: 'content_sync',
        files: this.config.sourceFiles.map(f => path.relative(this.changeDetector.projectRoot, f)),
        trigger: 'content_manager'
      });
      
      console.log('✅ 跟踪数据更新完成');
      
      return {
        name: 'tracking_update',
        status: 'completed'
      };
    } catch (error) {
      console.error('❌ 跟踪数据更新失败:', error.message);
      return {
        name: 'tracking_update',
        status: 'failed',
        error: error.message
      };
    }
  }

  /**
   * Evaluate overall workflow status
   */
  evaluateOverallStatus(steps) {
    if (steps.some(step => step.status === 'failed')) {
      return 'failed';
    }
    
    if (steps.some(step => step.status === 'completed_with_issues')) {
      return 'completed_with_issues';
    }
    
    if (steps.every(step => ['completed', 'skipped'].includes(step.status))) {
      return 'success';
    }
    
    return 'partial';
  }

  /**
   * Save workflow report
   */
  async saveReport(report) {
    const reportPath = path.resolve(__dirname, '.cache', 'workflow-report.json');
    await fs.writeFile(reportPath, JSON.stringify(report, null, 2));
    
    // Also save a timestamped copy
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const timestampedPath = path.resolve(__dirname, '.cache', `workflow-${timestamp}.json`);
    await fs.writeFile(timestampedPath, JSON.stringify(report, null, 2));
  }

  /**
   * Display workflow report
   */
  displayReport(report) {
    console.log('\n📊 内容管理报告');
    console.log('='.repeat(50));
    console.log(`总体状态: ${this.formatStatus(report.overall)}`);
    console.log(`完成时间: ${report.timestamp}`);
    
    console.log('\n📋 步骤详情:');
    report.steps.forEach((step, index) => {
      const icon = this.getStatusIcon(step.status);
      console.log(`${index + 1}. ${icon} ${step.name} (${step.status})`);
      
      if (step.hasChanges !== undefined) {
        console.log(`   变更: ${step.hasChanges ? '是' : '否'}`);
      }
      
      if (step.qualityCheck) {
        console.log(`   质量检查: ${step.qualityCheck.passed ? '通过' : '未通过'}`);
        if (!step.qualityCheck.passed) {
          step.qualityCheck.issues.forEach(issue => {
            console.log(`     • ${issue}`);
          });
        }
      }
      
      if (step.error) {
        console.log(`   错误: ${step.error}`);
      }
    });
    
    // Recommendations
    const recommendations = this.generateRecommendations(report);
    if (recommendations.length > 0) {
      console.log('\n💡 建议:');
      recommendations.forEach(rec => {
        console.log(`   • ${rec}`);
      });
    }
  }

  /**
   * Generate recommendations based on report
   */
  generateRecommendations(report) {
    const recommendations = [];
    
    if (report.overall === 'failed') {
      recommendations.push('检查错误日志并修复失败的步骤');
    }
    
    if (report.overall === 'completed_with_issues') {
      recommendations.push('考虑运行自动修复: --autofix');
      recommendations.push('手动检查并修复剩余的验证问题');
    }
    
    const validationStep = report.steps.find(s => s.name === 'validation');
    if (validationStep && validationStep.report) {
      const { totalErrors, totalWarnings } = validationStep.report.summary;
      if (totalErrors > 0) {
        recommendations.push('修复所有错误级别的问题');
      }
      if (totalWarnings > 5) {
        recommendations.push('考虑修复警告级别的问题以提高质量');
      }
    }
    
    return recommendations;
  }

  /**
   * Format status for display
   */
  formatStatus(status) {
    const statusMap = {
      success: '✅ 成功',
      failed: '❌ 失败',
      completed_with_issues: '⚠️ 完成但有问题',
      partial: '🔄 部分完成',
      pending: '⏳ 进行中'
    };
    
    return statusMap[status] || status;
  }

  /**
   * Get status icon
   */
  getStatusIcon(status) {
    const iconMap = {
      completed: '✅',
      failed: '❌',
      skipped: '⏭️',
      completed_with_issues: '⚠️'
    };
    
    return iconMap[status] || '🔄';
  }

  /**
   * Set up continuous monitoring
   */
  async startMonitoring() {
    console.log('👀 启动内容监控模式...');
    
    const watcher = await this.changeDetector.watchFiles(this.config.sourceFiles, async (change) => {
      console.log(`\n📝 检测到变更: ${change.path}`);
      
      // Wait a bit to allow multiple file changes to settle
      setTimeout(async () => {
        console.log('🔄 自动开始内容同步...');
        await this.manage({ validate: true, autofix: true });
      }, 2000);
    });
    
    console.log(`✨ 正在监控 ${this.config.sourceFiles.length} 个文件...`);
    console.log('按 Ctrl+C 停止监控');
    
    return watcher;
  }
}

// CLI interface
if (require.main === module) {
  const args = process.argv.slice(2);
  const command = args[0] || 'manage';
  
  const manager = new ContentManager();
  
  // Use IIFE to avoid inner function declaration
  (async function() {
    await manager.initialize();
    
    // Parse options
    const force = args.includes('--force');
    const noValidate = args.includes('--no-validate');
    const autofix = args.includes('--autofix');
    const noSnapshots = args.includes('--no-snapshots');
    
    switch (command) {
      case 'manage': {
        const result = await manager.manage({
          force,
          validate: !noValidate,
          autofix,
          createSnapshots: !noSnapshots
        });
        
        // Exit with error code if workflow failed
        if (result.overall === 'failed') {
          process.exit(1);
        }
        break;
      }
        
      case 'monitor': {
        const watcher = await manager.startMonitoring();
        
        // Handle graceful shutdown
        process.on('SIGINT', () => {
          console.log('\n👋 停止监控');
          watcher.close();
          process.exit(0);
        });
        
        // Keep process running
        await new Promise(() => {});
        break;
      }
        
      case 'help':
        console.log(`
内容管理系统

用法:
  node content-manager.js [command] [options]

命令:
  manage     执行完整的内容管理流程（默认）
  monitor    启动持续监控模式
  help       显示此帮助信息

选项:
  --force           强制执行所有步骤，忽略变更检测
  --no-validate     跳过内容验证步骤
  --autofix         自动修复可修复的问题
  --no-snapshots    跳过创建快照

示例:
  node content-manager.js manage
  node content-manager.js manage --force --autofix
  node content-manager.js monitor
        `);
        break;
        
      default:
        console.error(`未知命令: ${command}`);
        console.error('使用 "node content-manager.js help" 查看可用命令');
        process.exit(1);
    }
  })().catch(console.error);
}

module.exports = ContentManager;