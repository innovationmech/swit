#!/usr/bin/env node

/**
 * Comprehensive Test Suite Runner for Swit Website
 * Runs all available tests and generates a comprehensive report
 */

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

// Test configuration
const testConfig = {
  testStartTime: new Date(),
  results: {
    build: { status: 'pending', errors: [], duration: 0 },
    lighthouse: { status: 'pending', scores: {}, errors: [], duration: 0 },
    accessibility: { status: 'pending', violations: [], errors: [], duration: 0 },
    links: { status: 'pending', broken: [], errors: [], duration: 0 },
    content: { status: 'pending', issues: [], errors: [], duration: 0 },
    security: { status: 'pending', vulnerabilities: [], errors: [], duration: 0 }
  },
  summary: {
    passed: 0,
    failed: 0,
    total: 0,
    duration: 0
  }
};

// Colors for console output
const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  magenta: '\x1b[35m',
  cyan: '\x1b[36m'
};

function log(message, color = 'reset') {
  console.log(`${colors[color]}${message}${colors.reset}`);
}

function logSection(title) {
  log(`\n${'='.repeat(60)}`, 'cyan');
  log(title.toUpperCase(), 'bright');
  log('='.repeat(60), 'cyan');
}

function runCommand(command, options = {}) {
  try {
    const result = execSync(command, {
      encoding: 'utf8',
      stdio: options.silent ? 'pipe' : 'inherit',
      ...options
    });
    return { success: true, output: result };
  } catch (error) {
    return { 
      success: false, 
      error: error.message,
      output: error.stdout || error.stderr || ''
    };
  }
}

// Test 1: Build Test
async function runBuildTest() {
  logSection('Build Test');
  const startTime = Date.now();
  
  log('Testing website build process...', 'yellow');
  
  // Clean previous builds
  log('Cleaning previous builds...', 'blue');
  runCommand('npm run clean', { silent: true });
  
  // Attempt to build
  log('Building website...', 'blue');
  const buildResult = runCommand('npm run build', { silent: true });
  
  testConfig.results.build.duration = Date.now() - startTime;
  
  if (buildResult.success) {
    log('✅ Build test passed', 'green');
    testConfig.results.build.status = 'passed';
    testConfig.summary.passed++;
  } else {
    log('❌ Build test failed', 'red');
    log(`Error: ${buildResult.error}`, 'red');
    testConfig.results.build.status = 'failed';
    testConfig.results.build.errors.push(buildResult.error);
    testConfig.summary.failed++;
  }
  
  testConfig.summary.total++;
}

// Test 2: Content Validation Test
async function runContentValidation() {
  logSection('Content Validation Test');
  const startTime = Date.now();
  
  log('Validating content structure and quality...', 'yellow');
  
  const contentIssues = [];
  
  // Check if required files exist
  const requiredFiles = [
    'en/index.md',
    'zh/index.md',
    'en/guide/getting-started.md',
    'zh/guide/getting-started.md',
    'en/api/index.md',
    'zh/api/index.md',
    'en/community/index.md',
    'zh/community/index.md'
  ];
  
  requiredFiles.forEach(file => {
    const filePath = path.join(__dirname, '..', file);
    if (!fs.existsSync(filePath)) {
      contentIssues.push(`Missing required file: ${file}`);
    } else {
      log(`✅ Found: ${file}`, 'green');
    }
  });
  
  // Check for empty files
  requiredFiles.forEach(file => {
    const filePath = path.join(__dirname, '..', file);
    if (fs.existsSync(filePath)) {
      const content = fs.readFileSync(filePath, 'utf8');
      if (content.trim().length < 100) {
        contentIssues.push(`File appears to be incomplete: ${file}`);
      }
    }
  });
  
  testConfig.results.content.duration = Date.now() - startTime;
  testConfig.results.content.issues = contentIssues;
  
  if (contentIssues.length === 0) {
    log('✅ Content validation passed', 'green');
    testConfig.results.content.status = 'passed';
    testConfig.summary.passed++;
  } else {
    log('❌ Content validation failed', 'red');
    contentIssues.forEach(issue => log(`  - ${issue}`, 'red'));
    testConfig.results.content.status = 'failed';
    testConfig.summary.failed++;
  }
  
  testConfig.summary.total++;
}

// Test 3: Link Validation Test
async function runLinkValidation() {
  logSection('Link Validation Test');
  const startTime = Date.now();
  
  log('Checking for broken internal links...', 'yellow');
  
  // This is a simplified link check - in production you'd want a more comprehensive tool
  const brokenLinks = [];
  
  // Check if navigation links have corresponding files
  const navigationLinks = [
    '/en/guide/getting-started',
    '/zh/guide/getting-started',
    '/en/api/',
    '/zh/api/',
    '/en/examples/',
    '/zh/examples/',
    '/en/community/',
    '/zh/community/'
  ];
  
  navigationLinks.forEach(link => {
    // Convert link to file path
    const filePath = link.endsWith('/') 
      ? path.join(__dirname, '..', link, 'index.md')
      : path.join(__dirname, '..', link + '.md');
    
    if (!fs.existsSync(filePath)) {
      brokenLinks.push(`Navigation link has no corresponding file: ${link}`);
    } else {
      log(`✅ Link OK: ${link}`, 'green');
    }
  });
  
  testConfig.results.links.duration = Date.now() - startTime;
  testConfig.results.links.broken = brokenLinks;
  
  if (brokenLinks.length === 0) {
    log('✅ Link validation passed', 'green');
    testConfig.results.links.status = 'passed';
    testConfig.summary.passed++;
  } else {
    log('❌ Link validation failed', 'red');
    brokenLinks.forEach(link => log(`  - ${link}`, 'red'));
    testConfig.results.links.status = 'failed';
    testConfig.summary.failed++;
  }
  
  testConfig.summary.total++;
}

// Test 4: Configuration Validation Test
async function runConfigValidation() {
  logSection('Configuration Validation Test');
  const startTime = Date.now();
  
  log('Validating VitePress configuration...', 'yellow');
  
  const configIssues = [];
  
  // Check if config files exist
  const configFiles = [
    '.vitepress/config/index.mjs',
    '.vitepress/config/shared.mjs',
    '.vitepress/config/en.mjs',
    '.vitepress/config/zh.mjs'
  ];
  
  configFiles.forEach(file => {
    const filePath = path.join(__dirname, '..', file);
    if (!fs.existsSync(filePath)) {
      configIssues.push(`Missing config file: ${file}`);
    } else {
      log(`✅ Found config: ${file}`, 'green');
    }
  });
  
  // Check package.json
  const packagePath = path.join(__dirname, '..', 'package.json');
  if (fs.existsSync(packagePath)) {
    try {
      const packageJson = JSON.parse(fs.readFileSync(packagePath, 'utf8'));
      if (!packageJson.scripts || !packageJson.scripts.build) {
        configIssues.push('Missing build script in package.json');
      }
      if (!packageJson.devDependencies || !packageJson.devDependencies.vitepress) {
        configIssues.push('Missing VitePress dependency');
      }
      log('✅ Package.json configuration OK', 'green');
    } catch (error) {
      configIssues.push('Invalid package.json format');
    }
  } else {
    configIssues.push('Missing package.json');
  }
  
  testConfig.results.security.duration = Date.now() - startTime;
  testConfig.results.security.vulnerabilities = configIssues;
  
  if (configIssues.length === 0) {
    log('✅ Configuration validation passed', 'green');
    testConfig.results.security.status = 'passed';
    testConfig.summary.passed++;
  } else {
    log('❌ Configuration validation failed', 'red');
    configIssues.forEach(issue => log(`  - ${issue}`, 'red'));
    testConfig.results.security.status = 'failed';
    testConfig.summary.failed++;
  }
  
  testConfig.summary.total++;
}

// Generate Test Report
function generateTestReport() {
  logSection('Test Report Generation');
  
  const endTime = new Date();
  testConfig.summary.duration = endTime - testConfig.testStartTime;
  
  const reportContent = `# Website Testing Report

**Test Date:** ${endTime.toISOString()}
**Test Duration:** ${Math.round(testConfig.summary.duration / 1000)} seconds
**Environment:** ${process.platform} ${process.arch}

## Summary

- **Total Tests:** ${testConfig.summary.total}
- **Passed:** ${testConfig.summary.passed} ✅
- **Failed:** ${testConfig.summary.failed} ❌
- **Success Rate:** ${testConfig.summary.total > 0 ? Math.round((testConfig.summary.passed / testConfig.summary.total) * 100) : 0}%

## Detailed Results

### Build Test
- **Status:** ${testConfig.results.build.status === 'passed' ? '✅ PASSED' : '❌ FAILED'}
- **Duration:** ${testConfig.results.build.duration}ms
- **Errors:** ${testConfig.results.build.errors.length}

${testConfig.results.build.errors.length > 0 ? '**Error Details:**\n' + testConfig.results.build.errors.map(e => `- ${e}`).join('\n') : ''}

### Content Validation
- **Status:** ${testConfig.results.content.status === 'passed' ? '✅ PASSED' : '❌ FAILED'}  
- **Duration:** ${testConfig.results.content.duration}ms
- **Issues:** ${testConfig.results.content.issues.length}

${testConfig.results.content.issues.length > 0 ? '**Issue Details:**\n' + testConfig.results.content.issues.map(i => `- ${i}`).join('\n') : ''}

### Link Validation
- **Status:** ${testConfig.results.links.status === 'passed' ? '✅ PASSED' : '❌ FAILED'}
- **Duration:** ${testConfig.results.links.duration}ms  
- **Broken Links:** ${testConfig.results.links.broken.length}

${testConfig.results.links.broken.length > 0 ? '**Broken Link Details:**\n' + testConfig.results.links.broken.map(l => `- ${l}`).join('\n') : ''}

### Configuration Validation
- **Status:** ${testConfig.results.security.status === 'passed' ? '✅ PASSED' : '❌ FAILED'}
- **Duration:** ${testConfig.results.security.duration}ms
- **Issues:** ${testConfig.results.security.vulnerabilities.length}

${testConfig.results.security.vulnerabilities.length > 0 ? '**Configuration Issues:**\n' + testConfig.results.security.vulnerabilities.map(v => `- ${v}`).join('\n') : ''}

## Recommendations

${testConfig.summary.failed === 0 ? 
  '✅ **All tests passed!** The website is ready for production deployment.' :
  `❌ **${testConfig.summary.failed} test(s) failed.** Please address the issues above before deploying to production.`
}

### Next Steps
1. Review and fix any failed tests
2. Run manual testing checklist
3. Perform cross-browser testing
4. Conduct accessibility audit
5. Review performance metrics
6. Verify security configuration

---
*Generated by Swit Website Test Suite*
`;

  const reportPath = path.join(__dirname, '..', 'test-report.md');
  fs.writeFileSync(reportPath, reportContent);
  
  log(`Test report generated: ${reportPath}`, 'blue');
  return reportPath;
}

// Main test runner
async function runAllTests() {
  log('🚀 Starting Comprehensive Website Test Suite', 'bright');
  log(`Test started at: ${testConfig.testStartTime.toISOString()}`, 'blue');
  
  try {
    // Run all tests
    await runBuildTest();
    await runContentValidation();
    await runLinkValidation();
    await runConfigValidation();
    
    // Generate report
    const reportPath = generateTestReport();
    
    // Final summary
    logSection('Final Summary');
    
    if (testConfig.summary.failed === 0) {
      log('🎉 All tests passed! Website is ready for production.', 'green');
    } else {
      log(`⚠️  ${testConfig.summary.failed} test(s) failed. Please review the issues.`, 'red');
    }
    
    log(`📊 Test Results: ${testConfig.summary.passed}/${testConfig.summary.total} passed`, 'blue');
    log(`📄 Full report: ${reportPath}`, 'blue');
    log(`⏱️  Total duration: ${Math.round(testConfig.summary.duration / 1000)} seconds`, 'blue');
    
    // Exit with appropriate code
    process.exit(testConfig.summary.failed > 0 ? 1 : 0);
    
  } catch (error) {
    log(`❌ Test suite failed with error: ${error.message}`, 'red');
    process.exit(1);
  }
}

// Run tests if called directly
if (require.main === module) {
  runAllTests();
}

module.exports = { runAllTests, testConfig };