#!/usr/bin/env node

/**
 * Enhanced API Documentation Generator
 * 
 * This script parses Swagger/OpenAPI specification files and generates
 * comprehensive VitePress-compatible markdown documentation for the Swit framework website.
 * 
 * Features:
 * - Multi-service API documentation generation
 * - Multi-language support (zh/en)
 * - Automatic change detection and updates
 * - Rich code examples and interactive features
 * - Unified API documentation structure
 * 
 * Usage:
 *   node scripts/api-docs-generator.js [command]
 * 
 * Commands:
 *   generate    Generate all API documentation (default)
 *   update      Update existing documentation
 *   watch       Watch for changes and auto-update
 *   merge       Merge multiple service documentations
 */

const fs = require('fs').promises;
const fsSync = require('fs');
const path = require('path');
const crypto = require('crypto');

// Configuration
const CONFIG = {
  // Source directory containing Swagger files
  swaggerDir: path.join(process.cwd(), '../../docs/generated'),
  
  // Output directories for different languages
  outputDirs: {
    zh: path.join(process.cwd(), 'zh/api'),
    en: path.join(process.cwd(), 'en/api')
  },
  
  // Services to process
  services: ['switserve', 'switauth'],
  
  // Language configurations
  languages: {
    zh: {
      title: 'API 文档',
      overviewTitle: 'API 概览',
      endpointsTitle: '接口列表',
      parametersTitle: '参数',
      responsesTitle: '响应',
      examplesTitle: '示例',
      authTitle: '认证',
      errorTitle: '错误代码',
      typeRequired: '必填',
      typeOptional: '可选',
      methodLabel: '方法',
      pathLabel: '路径',
      descriptionLabel: '描述',
      statusLabel: '状态码',
      contentTypeLabel: '内容类型'
    },
    en: {
      title: 'API Documentation',
      overviewTitle: 'API Overview',
      endpointsTitle: 'Endpoints',
      parametersTitle: 'Parameters',
      responsesTitle: 'Responses',
      examplesTitle: 'Examples',
      authTitle: 'Authentication',
      errorTitle: 'Error Codes',
      typeRequired: 'Required',
      typeOptional: 'Optional',
      methodLabel: 'Method',
      pathLabel: 'Path',
      descriptionLabel: 'Description',
      statusLabel: 'Status Code',
      contentTypeLabel: 'Content Type'
    }
  }
};

/**
 * Enhanced API Documentation Generator Class
 */
class APIDocumentationGenerator {
  constructor() {
    this.swaggerSpecs = new Map();
    this.generatedDocs = new Map();
    this.cacheDir = path.join(process.cwd(), 'docs/pages/scripts/.cache');
    this.checksumFile = path.join(this.cacheDir, 'api-checksums.json');
  }

  /**
   * Main execution method
   */
  async generate(force = false) {
    console.log('🚀 Starting API documentation generation...');
    
    try {
      await this.initializeCache();
      
      // Check if regeneration is needed
      if (!force) {
        const hasChanged = await this.hasSpecsChanged();
        if (!hasChanged) {
          console.log('⏭️ No changes detected, skipping generation');
          return { status: 'skipped', reason: 'no_changes' };
        }
      }
      
      await this.loadSwaggerSpecs();
      await this.parseAPIs();
      await this.generateDocumentation();
      await this.createIndexPages();
      await this.generateMergedDocumentation();
      await this.updateChecksums();
      
      console.log('✅ API documentation generation completed successfully!');
      return { status: 'success', generated: this.generatedDocs.size };
    } catch (error) {
      console.error('❌ Error generating API documentation:', error);
      return { status: 'error', error: error.message };
    }
  }

  /**
   * Update existing documentation
   */
  async update() {
    console.log('🔄 Updating API documentation...');
    return await this.generate(false);
  }

  /**
   * Watch for changes and auto-update
   */
  async watch() {
    console.log('👀 Starting API documentation watcher...');
    
    const chokidar = require('chokidar');
    const watchPaths = CONFIG.services.map(service => 
      path.join(CONFIG.swaggerDir, service, 'swagger.json')
    );

    const watcher = chokidar.watch(watchPaths, {
      persistent: true,
      ignoreInitial: true
    });

    watcher.on('change', async (filePath) => {
      console.log(`\n📝 Detected change in: ${path.basename(filePath)}`);
      
      // Wait a bit for file write to complete
      setTimeout(async () => {
        const result = await this.update();
        if (result.status === 'success') {
          console.log('✅ Documentation updated successfully');
        } else if (result.status === 'error') {
          console.error('❌ Documentation update failed:', result.error);
        }
      }, 1000);
    });

    console.log(`✨ Watching ${watchPaths.length} Swagger files for changes...`);
    console.log('Press Ctrl+C to stop watching');

    return watcher;
  }

  /**
   * Initialize cache directory and files
   */
  async initializeCache() {
    try {
      await fs.mkdir(this.cacheDir, { recursive: true });
      
      // Initialize checksums file if it doesn't exist
      if (!fsSync.existsSync(this.checksumFile)) {
        await fs.writeFile(this.checksumFile, JSON.stringify({}, null, 2));
      }
    } catch (error) {
      console.warn('⚠️ Cache initialization warning:', error.message);
    }
  }

  /**
   * Calculate file checksum
   */
  async calculateChecksum(filePath) {
    try {
      const content = await fs.readFile(filePath, 'utf8');
      return crypto.createHash('sha256').update(content).digest('hex');
    } catch {
      return null;
    }
  }

  /**
   * Check if API specs have changed
   */
  async hasSpecsChanged() {
    try {
      const checksums = JSON.parse(await fs.readFile(this.checksumFile, 'utf8'));
      
      for (const service of CONFIG.services) {
        const swaggerPath = path.join(CONFIG.swaggerDir, service, 'swagger.json');
        const currentChecksum = await this.calculateChecksum(swaggerPath);
        
        if (!currentChecksum) continue;
        
        const storedChecksum = checksums[service];
        if (!storedChecksum || storedChecksum !== currentChecksum) {
          return true;
        }
      }
      
      return false;
    } catch {
      return true; // If we can't read checksums, assume changed
    }
  }

  /**
   * Update stored checksums
   */
  async updateChecksums() {
    try {
      const checksums = {};
      
      for (const service of CONFIG.services) {
        const swaggerPath = path.join(CONFIG.swaggerDir, service, 'swagger.json');
        const checksum = await this.calculateChecksum(swaggerPath);
        if (checksum) {
          checksums[service] = checksum;
        }
      }
      
      await fs.writeFile(this.checksumFile, JSON.stringify(checksums, null, 2));
    } catch (error) {
      console.warn('⚠️ Failed to update checksums:', error.message);
    }
  }

  /**
   * Load Swagger/OpenAPI specification files
   */
  async loadSwaggerSpecs() {
    console.log('📖 Loading Swagger specifications...');
    
    for (const service of CONFIG.services) {
      const swaggerPath = path.join(CONFIG.swaggerDir, service, 'swagger.json');
      
      if (fsSync.existsSync(swaggerPath)) {
        try {
          const swaggerContent = await fs.readFile(swaggerPath, 'utf8');
          const swaggerSpec = JSON.parse(swaggerContent);
          this.swaggerSpecs.set(service, swaggerSpec);
          console.log(`  ✓ Loaded ${service} API specification`);
        } catch (error) {
          console.error(`  ❌ Failed to load ${service} specification:`, error.message);
        }
      } else {
        console.warn(`  ⚠️  Swagger file not found for ${service}: ${swaggerPath}`);
      }
    }
  }

  /**
   * Parse API specifications into structured data
   */
  async parseAPIs() {
    console.log('🔍 Parsing API specifications...');
    
    for (const [serviceName, spec] of this.swaggerSpecs) {
      const parsedAPI = this.parseSwaggerSpec(serviceName, spec);
      this.generatedDocs.set(serviceName, parsedAPI);
      console.log(`  ✓ Parsed ${serviceName} API (${parsedAPI.endpoints.length} endpoints)`);
    }
  }

  /**
   * Parse individual Swagger specification
   */
  parseSwaggerSpec(serviceName, spec) {
    const api = {
      name: serviceName,
      title: spec.info?.title || serviceName,
      description: spec.info?.description || '',
      version: spec.info?.version || '1.0',
      host: spec.host || 'localhost',
      basePath: spec.basePath || '/',
      schemes: spec.schemes || ['http'],
      endpoints: [],
      tags: this.extractTags(spec),
      definitions: spec.definitions || {}
    };

    // Parse endpoints
    for (const [path, methods] of Object.entries(spec.paths || {})) {
      for (const [method, operation] of Object.entries(methods)) {
        if (typeof operation === 'object' && operation !== null) {
          api.endpoints.push({
            id: this.generateEndpointId(method, path),
            method: method.toUpperCase(),
            path: path,
            summary: operation.summary || '',
            description: operation.description || '',
            tags: operation.tags || [],
            parameters: this.parseParameters(operation.parameters || []),
            responses: this.parseResponses(operation.responses || {}),
            security: operation.security || [],
            deprecated: operation.deprecated || false
          });
        }
      }
    }

    return api;
  }

  /**
   * Extract tags from Swagger spec
   */
  extractTags(spec) {
    const tags = new Map();
    
    if (spec.tags) {
      for (const tag of spec.tags) {
        tags.set(tag.name, {
          name: tag.name,
          description: tag.description || ''
        });
      }
    }

    return tags;
  }

  /**
   * Generate unique endpoint ID
   */
  generateEndpointId(method, path) {
    return `${method}-${path.replace(/[^a-zA-Z0-9]/g, '-').toLowerCase()}`;
  }

  /**
   * Parse parameters from Swagger operation
   */
  parseParameters(parameters) {
    return parameters.map(param => ({
      name: param.name,
      in: param.in,
      type: param.type || 'object',
      required: param.required || false,
      description: param.description || '',
      schema: param.schema || null,
      example: param.example || null
    }));
  }

  /**
   * Parse responses from Swagger operation
   */
  parseResponses(responses) {
    const parsedResponses = [];
    
    for (const [statusCode, response] of Object.entries(responses)) {
      parsedResponses.push({
        statusCode: statusCode,
        description: response.description || '',
        schema: response.schema || null,
        headers: response.headers || {},
        examples: response.examples || {}
      });
    }

    return parsedResponses;
  }

  /**
   * Generate documentation files
   */
  async generateDocumentation() {
    console.log('📝 Generating documentation files...');
    
    for (const lang of Object.keys(CONFIG.languages)) {
      const outputDir = CONFIG.outputDirs[lang];
      await this.ensureDirectoryExists(outputDir);
      
      for (const [serviceName, apiData] of this.generatedDocs) {
        const markdown = this.generateServiceMarkdown(apiData, lang);
        const outputPath = path.join(outputDir, `${serviceName}.md`);
        
        await fs.writeFile(outputPath, markdown, 'utf8');
        console.log(`  ✓ Generated ${lang}/${serviceName}.md`);
      }
    }
  }

  /**
   * Generate merged documentation that combines all services
   */
  async generateMergedDocumentation() {
    console.log('📚 Generating merged API documentation...');
    
    for (const lang of Object.keys(CONFIG.languages)) {
      const config = CONFIG.languages[lang];
      const isZh = lang === 'zh';
      const outputDir = CONFIG.outputDirs[lang];
      
      let mergedMarkdown = `---
title: ${isZh ? '完整 API 参考' : 'Complete API Reference'}
description: ${isZh ? '所有服务的统一 API 文档' : 'Unified API documentation for all services'}
outline: deep
---

# ${isZh ? '完整 API 参考' : 'Complete API Reference'}

${isZh 
  ? '这里汇总了 Swit 框架中所有服务的 API 接口文档，方便您快速查找和使用。'
  : 'This is a comprehensive reference for all API endpoints in the Swit framework, organized for easy lookup and usage.'
}

## ${isZh ? '服务概览' : 'Services Overview'}

`;

      // Generate service overview table
      mergedMarkdown += `| ${isZh ? '服务名称' : 'Service'} | ${isZh ? '描述' : 'Description'} | ${isZh ? '端点数量' : 'Endpoints'} | ${isZh ? '版本' : 'Version'} |\n`;
      mergedMarkdown += `|------------|-------------|------------|----------|\n`;
      
      for (const [serviceName, apiData] of this.generatedDocs) {
        const endpointCount = apiData.endpoints.length;
        mergedMarkdown += `| [${apiData.title}](#${serviceName}) | ${apiData.description} | ${endpointCount} | ${apiData.version} |\n`;
      }
      
      mergedMarkdown += `\n`;

      // Generate detailed documentation for each service
      for (const [serviceName, apiData] of this.generatedDocs) {
        mergedMarkdown += `\n## ${apiData.title} {#${serviceName}}\n\n`;
        mergedMarkdown += `${apiData.description}\n\n`;
        
        // Service metadata
        mergedMarkdown += `- **${isZh ? '基础 URL' : 'Base URL'}**: \`${apiData.schemes[0]}://${apiData.host}${apiData.basePath}\`\n`;
        mergedMarkdown += `- **${isZh ? '版本' : 'Version'}**: \`${apiData.version}\`\n`;
        mergedMarkdown += `- **${isZh ? '端点数量' : 'Endpoints'}**: ${apiData.endpoints.length}\n\n`;

        // Group and display endpoints
        const endpointsByTag = this.groupEndpointsByTag(apiData.endpoints);
        
        for (const [tagName, endpoints] of endpointsByTag) {
          if (tagName !== 'default') {
            const tagInfo = apiData.tags.get(tagName);
            mergedMarkdown += `### ${tagInfo ? tagInfo.name : tagName}\n\n`;
            if (tagInfo && tagInfo.description) {
              mergedMarkdown += `${tagInfo.description}\n\n`;
            }
          }

          for (const endpoint of endpoints) {
            mergedMarkdown += this.generateEndpointMarkdown(endpoint, config, true);
          }
        }
      }

      const outputPath = path.join(outputDir, 'complete.md');
      await fs.writeFile(outputPath, mergedMarkdown, 'utf8');
      console.log(`  ✓ Generated ${lang}/complete.md`);
    }
  }

  /**
   * Generate markdown content for a service
   */
  generateServiceMarkdown(api, lang) {
    const config = CONFIG.languages[lang];
    const isZh = lang === 'zh';
    
    let markdown = `# ${api.title}\n\n`;
    
    if (api.description) {
      markdown += `${api.description}\n\n`;
    }

    // Service information
    markdown += `## ${config.overviewTitle}\n\n`;
    markdown += `- **${isZh ? '服务名称' : 'Service Name'}**: ${api.name}\n`;
    markdown += `- **${isZh ? '版本' : 'Version'}**: ${api.version}\n`;
    markdown += `- **${isZh ? '基础URL' : 'Base URL'}**: ${api.schemes[0]}://${api.host}${api.basePath}\n\n`;

    // Group endpoints by tags
    const endpointsByTag = this.groupEndpointsByTag(api.endpoints);
    
    markdown += `## ${config.endpointsTitle}\n\n`;

    for (const [tagName, endpoints] of endpointsByTag) {
      if (tagName !== 'default') {
        const tagInfo = api.tags.get(tagName);
        markdown += `### ${tagInfo ? tagInfo.name : tagName}\n\n`;
        if (tagInfo && tagInfo.description) {
          markdown += `${tagInfo.description}\n\n`;
        }
      }

      for (const endpoint of endpoints) {
        markdown += this.generateEndpointMarkdown(endpoint, config);
      }
    }

    return markdown;
  }

  /**
   * Group endpoints by tags
   */
  groupEndpointsByTag(endpoints) {
    const groups = new Map();
    
    for (const endpoint of endpoints) {
      const tag = endpoint.tags.length > 0 ? endpoint.tags[0] : 'default';
      
      if (!groups.has(tag)) {
        groups.set(tag, []);
      }
      
      groups.get(tag).push(endpoint);
    }

    return groups;
  }

  /**
   * Generate markdown for individual endpoint
   */
  generateEndpointMarkdown(endpoint, config, isInMerged = false) {
    const headerLevel = isInMerged ? '####' : '####';
    let markdown = `${headerLevel} ${endpoint.method} ${endpoint.path}\n\n`;
    
    if (endpoint.summary) {
      markdown += `**${endpoint.summary}**\n\n`;
    }
    
    if (endpoint.description) {
      markdown += `${endpoint.description}\n\n`;
    }

    // Method and path table
    markdown += `| ${config.methodLabel} | ${config.pathLabel} |\n`;
    markdown += `|---------|--------|\n`;
    markdown += `| \`${endpoint.method}\` | \`${endpoint.path}\` |\n\n`;

    // Parameters
    if (endpoint.parameters.length > 0) {
      markdown += `**${config.parametersTitle}**\n\n`;
      markdown += `| 参数名 | 类型 | ${config.typeRequired}/${config.typeOptional} | ${config.descriptionLabel} |\n`;
      markdown += `|------|------|----------|-------------|\n`;
      
      for (const param of endpoint.parameters) {
        const required = param.required ? config.typeRequired : config.typeOptional;
        markdown += `| \`${param.name}\` | \`${param.type}\` | ${required} | ${param.description || '-'} |\n`;
      }
      markdown += '\n';
    }

    // Responses
    if (endpoint.responses.length > 0) {
      markdown += `**${config.responsesTitle}**\n\n`;
      markdown += `| ${config.statusLabel} | ${config.descriptionLabel} |\n`;
      markdown += `|-------------|-------------|\n`;
      
      for (const response of endpoint.responses) {
        markdown += `| \`${response.statusCode}\` | ${response.description || '-'} |\n`;
      }
      markdown += '\n';
    }

    // Enhanced code examples with tabs
    markdown += `**${config.examplesTitle}**\n\n`;
    markdown += `:::tabs\n\n`;
    
    // cURL example
    markdown += `== cURL\n\n`;
    markdown += '```bash\n';
    markdown += `curl -X ${endpoint.method} \\\n`;
    markdown += `  "http://localhost:8080${endpoint.path}" \\\n`;
    
    if (endpoint.parameters.some(p => p.in === 'header')) {
      markdown += `  -H "Content-Type: application/json" \\\n`;
      markdown += `  -H "Authorization: Bearer <your_token>" \\\n`;
    }
    
    if (endpoint.parameters.some(p => p.in === 'body')) {
      markdown += `  -d '{"key": "value"}'\n`;
    } else {
      markdown = markdown.slice(0, -4) + '\n'; // Remove trailing backslash
    }
    
    markdown += '```\n\n';
    
    // JavaScript example
    markdown += `== JavaScript\n\n`;
    markdown += '```javascript\n';
    
    if (endpoint.method === 'GET') {
      markdown += `const response = await fetch('http://localhost:8080${endpoint.path}', {\n`;
      markdown += `  headers: {\n`;
      markdown += `    'Authorization': 'Bearer <your_token>',\n`;
      markdown += `  },\n`;
      markdown += `});\n`;
    } else {
      markdown += `const response = await fetch('http://localhost:8080${endpoint.path}', {\n`;
      markdown += `  method: '${endpoint.method}',\n`;
      markdown += `  headers: {\n`;
      markdown += `    'Content-Type': 'application/json',\n`;
      markdown += `    'Authorization': 'Bearer <your_token>',\n`;
      markdown += `  },\n`;
      if (endpoint.parameters.some(p => p.in === 'body')) {
        markdown += `  body: JSON.stringify({ key: 'value' }),\n`;
      }
      markdown += `});\n`;
    }
    
    markdown += `const data = await response.json();\n`;
    markdown += `console.log(data);\n`;
    markdown += '```\n\n';
    
    // Python example
    markdown += `== Python\n\n`;
    markdown += '```python\n';
    markdown += `import requests\n\n`;
    markdown += `headers = {\n`;
    markdown += `    'Content-Type': 'application/json',\n`;
    markdown += `    'Authorization': 'Bearer <your_token>',\n`;
    markdown += `}\n\n`;
    
    if (endpoint.method === 'GET') {
      markdown += `response = requests.get(\n`;
      markdown += `    'http://localhost:8080${endpoint.path}',\n`;
      markdown += `    headers=headers\n`;
      markdown += `)\n`;
    } else {
      markdown += `response = requests.${endpoint.method.toLowerCase()}(\n`;
      markdown += `    'http://localhost:8080${endpoint.path}',\n`;
      if (endpoint.parameters.some(p => p.in === 'body')) {
        markdown += `    json={'key': 'value'},\n`;
      }
      markdown += `    headers=headers\n`;
      markdown += `)\n`;
    }
    
    markdown += `data = response.json()\n`;
    markdown += `print(data)\n`;
    markdown += '```\n\n';
    
    markdown += `:::\n\n`;
    
    if (!isInMerged) {
      markdown += `---\n\n`;
    }
    
    return markdown;
  }

  /**
   * Create index pages for API documentation
   */
  async createIndexPages() {
    console.log('📋 Creating index pages...');
    
    for (const lang of Object.keys(CONFIG.languages)) {
      const config = CONFIG.languages[lang];
      const isZh = lang === 'zh';
      const outputDir = CONFIG.outputDirs[lang];
      
      let indexMarkdown = `---
title: ${config.title}
description: ${isZh ? 'Swit 框架 API 接口文档' : 'Swit Framework API Documentation'}
---

# ${config.title}\n\n`;
      
      if (isZh) {
        indexMarkdown += `欢迎使用 Swit 框架 API 文档。以下是可用的服务和接口说明。\n\n`;
      } else {
        indexMarkdown += `Welcome to the Swit Framework API Documentation. Below are the available services and their API descriptions.\n\n`;
      }

      indexMarkdown += `## ${isZh ? '服务列表' : 'Available Services'}\n\n`;

      // Service overview cards
      indexMarkdown += `<div class="service-grid">\n\n`;
      
      for (const [serviceName, apiData] of this.generatedDocs) {
        const serviceTitle = apiData.title;
        const endpointCount = apiData.endpoints.length;
        
        indexMarkdown += `<div class="service-card">\n\n`;
        indexMarkdown += `### [${serviceTitle}](./${serviceName}.md)\n\n`;
        indexMarkdown += `${apiData.description}\n\n`;
        indexMarkdown += `<div class="service-stats">\n\n`;
        indexMarkdown += `- **${isZh ? '端点数量' : 'Endpoints'}**: ${endpointCount}\n`;
        indexMarkdown += `- **${isZh ? '版本' : 'Version'}**: ${apiData.version}\n`;
        indexMarkdown += `- **${isZh ? '基础URL' : 'Base URL'}**: \`${apiData.schemes[0]}://${apiData.host}${apiData.basePath}\`\n\n`;
        indexMarkdown += `</div>\n\n`;
        indexMarkdown += `[${isZh ? '查看文档' : 'View Documentation'} →](./${serviceName}.md)\n\n`;
        indexMarkdown += `</div>\n\n`;
      }
      
      indexMarkdown += `</div>\n\n`;

      // Quick links
      indexMarkdown += `## ${isZh ? '快速链接' : 'Quick Links'}\n\n`;
      indexMarkdown += `- [${isZh ? '完整 API 参考' : 'Complete API Reference'}](./complete.md)\n`;
      indexMarkdown += `- [${isZh ? '概览' : 'Overview'}](./overview.md)\n`;
      
      if (this.generatedDocs.has('switauth')) {
        indexMarkdown += `- [${isZh ? '认证服务' : 'Authentication Service'}](./switauth.md)\n`;
      }
      
      if (this.generatedDocs.has('switserve')) {
        indexMarkdown += `- [${isZh ? '用户管理服务' : 'User Management Service'}](./switserve.md)\n`;
      }
      
      indexMarkdown += `\n`;

      // Add common information
      if (isZh) {
        indexMarkdown += `## 通用信息\n\n`;
        indexMarkdown += `### 认证\n\n`;
        indexMarkdown += `所有 API 接口都需要适当的身份验证。大多数接口使用 Bearer Token 认证：\n\n`;
        indexMarkdown += '```http\n';
        indexMarkdown += 'Authorization: Bearer <your_access_token>\n';
        indexMarkdown += '```\n\n';
        indexMarkdown += `### 请求格式\n\n`;
        indexMarkdown += `- **Content-Type**: \`application/json\`\n`;
        indexMarkdown += `- **Accept**: \`application/json\`\n`;
        indexMarkdown += `- **编码**: UTF-8\n\n`;
        indexMarkdown += `### 错误处理\n\n`;
        indexMarkdown += `API 使用标准 HTTP 状态码来表示请求的成功或失败状态：\n\n`;
        indexMarkdown += `| 状态码 | 说明 | 示例 |\n`;
        indexMarkdown += `|--------|------|------|\n`;
        indexMarkdown += `| 200 | 请求成功 | 数据获取成功 |\n`;
        indexMarkdown += `| 201 | 创建成功 | 用户创建成功 |\n`;
        indexMarkdown += `| 400 | 请求错误 | 参数格式错误 |\n`;
        indexMarkdown += `| 401 | 未授权 | Token 无效或过期 |\n`;
        indexMarkdown += `| 403 | 禁止访问 | 权限不足 |\n`;
        indexMarkdown += `| 404 | 资源不存在 | 用户不存在 |\n`;
        indexMarkdown += `| 429 | 请求过多 | 触发速率限制 |\n`;
        indexMarkdown += `| 500 | 服务器内部错误 | 系统异常 |\n\n`;
        indexMarkdown += `### 响应格式\n\n`;
        indexMarkdown += `所有响应都采用统一的 JSON 格式：\n\n`;
        indexMarkdown += '```json\n';
        indexMarkdown += '{\n';
        indexMarkdown += '  "status": "success|error",\n';
        indexMarkdown += '  "data": {},\n';
        indexMarkdown += '  "message": "响应消息",\n';
        indexMarkdown += '  "timestamp": "2023-12-01T12:00:00Z"\n';
        indexMarkdown += '}\n';
        indexMarkdown += '```\n\n';
      } else {
        indexMarkdown += `## General Information\n\n`;
        indexMarkdown += `### Authentication\n\n`;
        indexMarkdown += `All API endpoints require appropriate authentication. Most endpoints use Bearer Token authentication:\n\n`;
        indexMarkdown += '```http\n';
        indexMarkdown += 'Authorization: Bearer <your_access_token>\n';
        indexMarkdown += '```\n\n';
        indexMarkdown += `### Request Format\n\n`;
        indexMarkdown += `- **Content-Type**: \`application/json\`\n`;
        indexMarkdown += `- **Accept**: \`application/json\`\n`;
        indexMarkdown += `- **Encoding**: UTF-8\n\n`;
        indexMarkdown += `### Error Handling\n\n`;
        indexMarkdown += `The API uses standard HTTP status codes to indicate the success or failure of requests:\n\n`;
        indexMarkdown += `| Status Code | Description | Example |\n`;
        indexMarkdown += `|-------------|-------------|----------|\n`;
        indexMarkdown += `| 200 | Success | Data retrieved successfully |\n`;
        indexMarkdown += `| 201 | Created | User created successfully |\n`;
        indexMarkdown += `| 400 | Bad Request | Invalid parameter format |\n`;
        indexMarkdown += `| 401 | Unauthorized | Invalid or expired token |\n`;
        indexMarkdown += `| 403 | Forbidden | Insufficient permissions |\n`;
        indexMarkdown += `| 404 | Not Found | User does not exist |\n`;
        indexMarkdown += `| 429 | Too Many Requests | Rate limit exceeded |\n`;
        indexMarkdown += `| 500 | Internal Server Error | System error |\n\n`;
        indexMarkdown += `### Response Format\n\n`;
        indexMarkdown += `All responses follow a unified JSON format:\n\n`;
        indexMarkdown += '```json\n';
        indexMarkdown += '{\n';
        indexMarkdown += '  "status": "success|error",\n';
        indexMarkdown += '  "data": {},\n';
        indexMarkdown += '  "message": "Response message",\n';
        indexMarkdown += '  "timestamp": "2023-12-01T12:00:00Z"\n';
        indexMarkdown += '}\n';
        indexMarkdown += '```\n\n';
      }

      // Add custom CSS for service cards
      indexMarkdown += `<style>
.service-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1rem;
  margin: 1rem 0;
}

.service-card {
  border: 1px solid var(--vp-c-border);
  border-radius: 8px;
  padding: 1.5rem;
  background: var(--vp-c-bg-soft);
}

.service-card h3 {
  margin-top: 0;
  color: var(--vp-c-brand-1);
}

.service-stats {
  margin: 1rem 0;
  font-size: 0.9em;
}

.service-card > p:last-child {
  margin-bottom: 0;
  text-align: right;
  font-weight: 500;
}

.service-card a {
  text-decoration: none;
}

.service-card a:hover {
  text-decoration: underline;
}
</style>

`;

      const indexPath = path.join(outputDir, 'index.md');
      await fs.writeFile(indexPath, indexMarkdown, 'utf8');
      console.log(`  ✓ Generated ${lang}/index.md`);
    }
  }

  /**
   * Ensure directory exists (async version)
   */
  async ensureDirectoryExists(dirPath) {
    try {
      await fs.mkdir(dirPath, { recursive: true });
    } catch (error) {
      if (error.code !== 'EEXIST') {
        throw error;
      }
    }
  }
}

// Script execution
if (require.main === module) {
  const args = process.argv.slice(2);
  const command = args[0] || 'generate';
  
  const generator = new APIDocumentationGenerator();
  
  // Use IIFE to avoid inner function declaration
  (async function() {
    switch (command) {
      case 'generate': {
        const force = args.includes('--force');
        const result = await generator.generate(force);
        if (result.status === 'error') {
          console.error('Generation failed:', result.error);
          process.exit(1);
        } else if (result.status === 'skipped') {
          console.log('Generation skipped: no changes detected');
        }
        break;
      }
        
      case 'update': {
        const updateResult = await generator.update();
        if (updateResult.status === 'error') {
          console.error('Update failed:', updateResult.error);
          process.exit(1);
        }
        break;
      }
        
      case 'watch': {
        const watcher = await generator.watch();
        
        // Handle graceful shutdown
        process.on('SIGINT', () => {
          console.log('\n👋 Stopping API documentation watcher');
          watcher.close();
          process.exit(0);
        });
        
        // Keep process running
        await new Promise(() => {}); // Wait indefinitely
        break;
      }
        
      case 'help':
        console.log(`
API Documentation Generator

Usage:
  node api-docs-generator.js [command] [options]

Commands:
  generate    Generate API documentation (default)
  update      Update existing documentation
  watch       Watch for changes and auto-update
  help        Show this help message

Options:
  --force     Force regeneration even if no changes detected

Examples:
  node api-docs-generator.js generate
  node api-docs-generator.js generate --force
  node api-docs-generator.js update
  node api-docs-generator.js watch
        `);
        break;
        
      default:
        console.error(`Unknown command: ${command}`);
        console.error('Use "node api-docs-generator.js help" to see available commands');
        process.exit(1);
    }
  })().catch(console.error);
}

module.exports = APIDocumentationGenerator;