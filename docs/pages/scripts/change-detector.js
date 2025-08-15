#!/usr/bin/env node

/**
 * Change Detection and Version Control System
 * 
 * Monitors content changes and manages version control for documentation synchronization
 */

const fs = require('fs').promises;
const path = require('path');
const crypto = require('crypto');

class ChangeDetector {
  constructor() {
    this.projectRoot = path.resolve(__dirname, '../../../');
    this.cacheDir = path.resolve(__dirname, '.cache');
    this.historyDir = path.join(this.cacheDir, 'history');
    this.checksumFile = path.join(this.cacheDir, 'checksums.json');
    this.changelogFile = path.join(this.cacheDir, 'changelog.json');
  }

  /**
   * Initialize the change detector
   */
  async initialize() {
    await fs.mkdir(this.cacheDir, { recursive: true });
    await fs.mkdir(this.historyDir, { recursive: true });
    
    // Initialize checksums file if it doesn't exist
    try {
      await fs.access(this.checksumFile);
    } catch {
      await fs.writeFile(this.checksumFile, JSON.stringify({}, null, 2));
    }
    
    // Initialize changelog if it doesn't exist
    try {
      await fs.access(this.changelogFile);
    } catch {
      await fs.writeFile(this.changelogFile, JSON.stringify([], null, 2));
    }
  }

  /**
   * Calculate file checksum
   */
  async calculateChecksum(filePath) {
    try {
      const content = await fs.readFile(filePath, 'utf8');
      const hash = crypto.createHash('sha256');
      hash.update(content);
      return {
        sha256: hash.digest('hex'),
        size: Buffer.byteLength(content, 'utf8'),
        lines: content.split('\n').length,
        lastModified: (await fs.stat(filePath)).mtime.toISOString()
      };
    } catch (error) {
      return null;
    }
  }

  /**
   * Load stored checksums
   */
  async loadChecksums() {
    try {
      const content = await fs.readFile(this.checksumFile, 'utf8');
      return JSON.parse(content);
    } catch {
      return {};
    }
  }

  /**
   * Save checksums
   */
  async saveChecksums(checksums) {
    await fs.writeFile(this.checksumFile, JSON.stringify(checksums, null, 2));
  }

  /**
   * Detect changes in a file
   */
  async detectFileChanges(filePath) {
    const relativePath = path.relative(this.projectRoot, filePath);
    const checksums = await this.loadChecksums();
    const currentChecksum = await this.calculateChecksum(filePath);
    
    if (!currentChecksum) {
      return {
        path: relativePath,
        status: 'error',
        error: 'Could not calculate checksum'
      };
    }

    const stored = checksums[relativePath];
    
    if (!stored) {
      return {
        path: relativePath,
        status: 'new',
        current: currentChecksum,
        changes: ['File is new']
      };
    }

    const changes = [];
    let hasChanges = false;

    // Check for content changes
    if (stored.sha256 !== currentChecksum.sha256) {
      changes.push('Content modified');
      hasChanges = true;
    }

    // Check for size changes
    if (stored.size !== currentChecksum.size) {
      const sizeDiff = currentChecksum.size - stored.size;
      changes.push(`Size changed by ${sizeDiff > 0 ? '+' : ''}${sizeDiff} bytes`);
    }

    // Check for line count changes
    if (stored.lines !== currentChecksum.lines) {
      const lineDiff = currentChecksum.lines - stored.lines;
      changes.push(`Line count changed by ${lineDiff > 0 ? '+' : ''}${lineDiff} lines`);
    }

    return {
      path: relativePath,
      status: hasChanges ? 'modified' : 'unchanged',
      previous: stored,
      current: currentChecksum,
      changes
    };
  }

  /**
   * Detect changes in multiple files
   */
  async detectChanges(filePaths) {
    const results = [];
    
    for (const filePath of filePaths) {
      const result = await this.detectFileChanges(filePath);
      results.push(result);
    }

    return results;
  }

  /**
   * Create content snapshot
   */
  async createSnapshot(filePath, metadata = {}) {
    const content = await fs.readFile(filePath, 'utf8');
    const timestamp = new Date().toISOString();
    const relativePath = path.relative(this.projectRoot, filePath);
    const checksum = await this.calculateChecksum(filePath);
    
    const snapshot = {
      path: relativePath,
      timestamp,
      checksum,
      metadata,
      content
    };

    // Save snapshot to history
    const snapshotFileName = `${relativePath.replace(/[/\\]/g, '_')}_${timestamp.replace(/[:.]/g, '-')}.json`;
    const snapshotPath = path.join(this.historyDir, snapshotFileName);
    
    await fs.writeFile(snapshotPath, JSON.stringify(snapshot, null, 2));
    
    return {
      snapshotPath,
      snapshot
    };
  }

  /**
   * Compare two snapshots
   */
  compareSnapshots(snapshot1, snapshot2) {
    const differences = [];

    // Basic metadata comparison
    if (snapshot1.checksum.size !== snapshot2.checksum.size) {
      differences.push({
        type: 'size',
        from: snapshot1.checksum.size,
        to: snapshot2.checksum.size,
        change: snapshot2.checksum.size - snapshot1.checksum.size
      });
    }

    if (snapshot1.checksum.lines !== snapshot2.checksum.lines) {
      differences.push({
        type: 'lines',
        from: snapshot1.checksum.lines,
        to: snapshot2.checksum.lines,
        change: snapshot2.checksum.lines - snapshot1.checksum.lines
      });
    }

    // Content comparison (line by line)
    if (snapshot1.checksum.sha256 !== snapshot2.checksum.sha256) {
      const lines1 = snapshot1.content.split('\n');
      const lines2 = snapshot2.content.split('\n');
      const lineDiffs = this.diffLines(lines1, lines2);
      
      differences.push({
        type: 'content',
        lineDiffs
      });
    }

    return differences;
  }

  /**
   * Simple line-by-line diff
   */
  diffLines(lines1, lines2) {
    const diffs = [];
    const maxLength = Math.max(lines1.length, lines2.length);

    for (let i = 0; i < maxLength; i++) {
      const line1 = lines1[i];
      const line2 = lines2[i];

      if (line1 === undefined) {
        diffs.push({ type: 'added', line: i + 1, content: line2 });
      } else if (line2 === undefined) {
        diffs.push({ type: 'removed', line: i + 1, content: line1 });
      } else if (line1 !== line2) {
        diffs.push({ 
          type: 'modified', 
          line: i + 1, 
          from: line1, 
          to: line2 
        });
      }
    }

    return diffs;
  }

  /**
   * Update checksums after successful sync
   */
  async updateChecksums(filePaths) {
    const checksums = await this.loadChecksums();
    
    for (const filePath of filePaths) {
      const relativePath = path.relative(this.projectRoot, filePath);
      const checksum = await this.calculateChecksum(filePath);
      
      if (checksum) {
        checksums[relativePath] = checksum;
      }
    }
    
    await this.saveChecksums(checksums);
  }

  /**
   * Log changes to changelog
   */
  async logChange(changeData) {
    const changelog = JSON.parse(await fs.readFile(this.changelogFile, 'utf8'));
    
    const entry = {
      timestamp: new Date().toISOString(),
      ...changeData
    };
    
    changelog.unshift(entry); // Add to beginning
    
    // Keep only last 1000 entries
    if (changelog.length > 1000) {
      changelog.splice(1000);
    }
    
    await fs.writeFile(this.changelogFile, JSON.stringify(changelog, null, 2));
  }

  /**
   * Get change history for a file
   */
  async getFileHistory(filePath) {
    const relativePath = path.relative(this.projectRoot, filePath);
    const changelog = JSON.parse(await fs.readFile(this.changelogFile, 'utf8'));
    
    return changelog.filter(entry => 
      entry.path === relativePath || 
      entry.files?.includes(relativePath)
    );
  }

  /**
   * Clean old snapshots
   */
  async cleanupSnapshots(maxAge = 30) { // days
    const cutoffDate = new Date();
    cutoffDate.setDate(cutoffDate.getDate() - maxAge);
    
    try {
      const files = await fs.readdir(this.historyDir);
      let cleaned = 0;
      
      for (const file of files) {
        const filePath = path.join(this.historyDir, file);
        const stats = await fs.stat(filePath);
        
        if (stats.mtime < cutoffDate) {
          await fs.unlink(filePath);
          cleaned++;
        }
      }
      
      return { cleaned, total: files.length };
    } catch (error) {
      return { error: error.message };
    }
  }

  /**
   * Generate change summary
   */
  async generateChangeSummary(results) {
    const summary = {
      total: results.length,
      new: results.filter(r => r.status === 'new').length,
      modified: results.filter(r => r.status === 'modified').length,
      unchanged: results.filter(r => r.status === 'unchanged').length,
      errors: results.filter(r => r.status === 'error').length,
      details: results
    };

    // Log the summary
    await this.logChange({
      type: 'batch_check',
      summary: {
        total: summary.total,
        new: summary.new,
        modified: summary.modified,
        unchanged: summary.unchanged,
        errors: summary.errors
      },
      files: results.map(r => r.path)
    });

    return summary;
  }

  /**
   * Monitor files for changes
   */
  async watchFiles(filePaths, callback) {
    const chokidar = require('chokidar');
    
    const watcher = chokidar.watch(filePaths, {
      persistent: true,
      ignoreInitial: true
    });

    watcher.on('change', async (filePath) => {
      const result = await this.detectFileChanges(filePath);
      
      if (result.status === 'modified') {
        // Create snapshot before notifying
        await this.createSnapshot(filePath, {
          trigger: 'file_watch',
          changes: result.changes
        });

        // Log the change
        await this.logChange({
          type: 'file_change',
          path: result.path,
          changes: result.changes
        });

        if (callback) {
          callback(result);
        }
      }
    });

    return watcher;
  }
}

// CLI interface
if (require.main === module) {
  const args = process.argv.slice(2);
  const command = args[0] || 'check';
  
  const detector = new ChangeDetector();
  
  async function runCommand() {
    await detector.initialize();
    
    switch (command) {
      case 'check':
        const files = args.slice(1);
        if (files.length === 0) {
          console.error('请指定要检查的文件路径');
          process.exit(1);
        }
        
        const results = await detector.detectChanges(files);
        const summary = await detector.generateChangeSummary(results);
        
        console.log('\n🔍 变更检测结果');
        console.log('='.repeat(50));
        console.log(`总文件数: ${summary.total}`);
        console.log(`新文件: ${summary.new}`);
        console.log(`已修改: ${summary.modified}`);
        console.log(`未变更: ${summary.unchanged}`);
        console.log(`错误: ${summary.errors}`);
        
        if (summary.modified > 0 || summary.new > 0) {
          console.log('\n📝 变更详情:');
          for (const result of results) {
            if (result.status === 'new' || result.status === 'modified') {
              console.log(`\n📄 ${result.path} (${result.status})`);
              result.changes.forEach(change => {
                console.log(`   • ${change}`);
              });
            }
          }
        }
        
        break;
        
      case 'snapshot':
        const file = args[1];
        if (!file) {
          console.error('请指定要创建快照的文件路径');
          process.exit(1);
        }
        
        const { snapshotPath } = await detector.createSnapshot(file, {
          trigger: 'manual',
          reason: args[2] || 'Manual snapshot creation'
        });
        
        console.log(`✅ 快照已创建: ${snapshotPath}`);
        break;
        
      case 'history':
        const targetFile = args[1];
        if (!targetFile) {
          console.error('请指定要查看历史的文件路径');
          process.exit(1);
        }
        
        const history = await detector.getFileHistory(targetFile);
        
        console.log(`\n📚 ${targetFile} 变更历史`);
        console.log('='.repeat(50));
        
        if (history.length === 0) {
          console.log('未找到变更记录');
        } else {
          history.slice(0, 10).forEach(entry => { // Show last 10 entries
            console.log(`\n🕒 ${entry.timestamp}`);
            console.log(`   类型: ${entry.type}`);
            if (entry.changes) {
              console.log(`   变更: ${entry.changes.join(', ')}`);
            }
          });
        }
        
        break;
        
      case 'cleanup':
        const maxAge = parseInt(args[1]) || 30;
        const result = await detector.cleanupSnapshots(maxAge);
        
        if (result.error) {
          console.error(`清理失败: ${result.error}`);
          process.exit(1);
        } else {
          console.log(`✅ 清理完成: 删除了 ${result.cleaned}/${result.total} 个旧快照`);
        }
        
        break;
        
      case 'watch':
        const watchFiles = args.slice(1);
        if (watchFiles.length === 0) {
          console.error('请指定要监视的文件路径');
          process.exit(1);
        }
        
        console.log(`👀 开始监视 ${watchFiles.length} 个文件...`);
        
        await detector.watchFiles(watchFiles, (change) => {
          console.log(`\n📝 检测到变更: ${change.path}`);
          change.changes.forEach(c => console.log(`   • ${c}`));
        });
        
        console.log('按 Ctrl+C 停止监视');
        
        // Keep the process running
        process.on('SIGINT', () => {
          console.log('\n👋 停止监视');
          process.exit(0);
        });
        
        break;
        
      case 'help':
        console.log(`
变更检测工具

用法:
  node change-detector.js [command] [options]

命令:
  check <files...>      检测指定文件的变更
  snapshot <file>       为指定文件创建快照
  history <file>        查看文件的变更历史
  cleanup [days]        清理旧快照（默认30天）
  watch <files...>      监视文件变更
  help                  显示此帮助信息

示例:
  node change-detector.js check README.md README-CN.md
  node change-detector.js snapshot README.md
  node change-detector.js history README.md
  node change-detector.js cleanup 15
  node change-detector.js watch README.md README-CN.md
        `);
        break;
        
      default:
        console.error(`未知命令: ${command}`);
        console.error('使用 "node change-detector.js help" 查看可用命令');
        process.exit(1);
    }
  }
  
  runCommand().catch(console.error);
}

module.exports = ChangeDetector;