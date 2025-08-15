#!/usr/bin/env node

/**
 * API Documentation Generator
 * 
 * This script parses Swagger/OpenAPI specification files and generates
 * VitePress-compatible markdown documentation for the Swit framework website.
 * 
 * Usage:
 *   node scripts/api-docs-generator.js
 * 
 * The script will:
 * 1. Read Swagger JSON files from docs/generated/
 * 2. Parse API specifications
 * 3. Generate markdown files for each service
 * 4. Create language-specific documentation (zh/en)
 */

const fs = require('fs');
const path = require('path');

// Configuration
const CONFIG = {
  // Source directory containing Swagger files
  swaggerDir: path.join(process.cwd(), 'docs/generated'),
  
  // Output directories for different languages
  outputDirs: {
    zh: path.join(process.cwd(), 'docs/pages/zh/api'),
    en: path.join(process.cwd(), 'docs/pages/en/api')
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
 * API Documentation Generator Class
 */
class APIDocumentationGenerator {
  constructor() {
    this.swaggerSpecs = new Map();
    this.generatedDocs = new Map();
  }

  /**
   * Main execution method
   */
  async generate() {
    console.log('🚀 Starting API documentation generation...');
    
    try {
      await this.loadSwaggerSpecs();
      await this.parseAPIs();
      await this.generateDocumentation();
      await this.createIndexPages();
      
      console.log('✅ API documentation generation completed successfully!');
    } catch (error) {
      console.error('❌ Error generating API documentation:', error);
      process.exit(1);
    }
  }

  /**
   * Load Swagger/OpenAPI specification files
   */
  async loadSwaggerSpecs() {
    console.log('📖 Loading Swagger specifications...');
    
    for (const service of CONFIG.services) {
      const swaggerPath = path.join(CONFIG.swaggerDir, service, 'swagger.json');
      
      if (fs.existsSync(swaggerPath)) {
        try {
          const swaggerContent = fs.readFileSync(swaggerPath, 'utf8');
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
      this.ensureDirectoryExists(outputDir);
      
      for (const [serviceName, apiData] of this.generatedDocs) {
        const markdown = this.generateServiceMarkdown(apiData, lang);
        const outputPath = path.join(outputDir, `${serviceName}.md`);
        
        fs.writeFileSync(outputPath, markdown, 'utf8');
        console.log(`  ✓ Generated ${lang}/${serviceName}.md`);
      }
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
  generateEndpointMarkdown(endpoint, config) {
    let markdown = `#### ${endpoint.method} ${endpoint.path}\n\n`;
    
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
      markdown += `| ${config.descriptionLabel.split(' ')[0]} | ${config.descriptionLabel.split(' ').slice(1).join(' ') || 'Type'} | ${config.typeRequired}/${config.typeOptional} | ${config.descriptionLabel} |\n`;
      markdown += `|------|------|----------|-------------|\n`;
      
      for (const param of endpoint.parameters) {
        const required = param.required ? config.typeRequired : config.typeOptional;
        markdown += `| \`${param.name}\` | ${param.type} | ${required} | ${param.description || '-'} |\n`;
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

    // Example request
    markdown += `**${config.examplesTitle}**\n\n`;
    markdown += '```bash\n';
    markdown += `curl -X ${endpoint.method} "${endpoint.path}"`;
    
    if (endpoint.parameters.some(p => p.in === 'header')) {
      markdown += ' \\\n  -H "Content-Type: application/json"';
    }
    
    if (endpoint.parameters.some(p => p.in === 'body')) {
      markdown += ' \\\n  -d \'{"key": "value"}\'';
    }
    
    markdown += '\n```\n\n';
    
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
      
      let indexMarkdown = `# ${config.title}\n\n`;
      
      if (isZh) {
        indexMarkdown += `欢迎使用 Swit 框架 API 文档。以下是可用的服务和接口说明。\n\n`;
      } else {
        indexMarkdown += `Welcome to the Swit Framework API Documentation. Below are the available services and their API descriptions.\n\n`;
      }

      indexMarkdown += `## ${isZh ? '服务列表' : 'Available Services'}\n\n`;

      for (const [serviceName, apiData] of this.generatedDocs) {
        const serviceTitle = apiData.title;
        const endpointCount = apiData.endpoints.length;
        
        indexMarkdown += `### [${serviceTitle}](./${serviceName}.md)\n\n`;
        indexMarkdown += `${apiData.description}\n\n`;
        indexMarkdown += `- **${isZh ? '接口数量' : 'Endpoints'}**: ${endpointCount}\n`;
        indexMarkdown += `- **${isZh ? '版本' : 'Version'}**: ${apiData.version}\n`;
        indexMarkdown += `- **${isZh ? '基础URL' : 'Base URL'}**: ${apiData.schemes[0]}://${apiData.host}${apiData.basePath}\n\n`;
      }

      // Add common information
      if (isZh) {
        indexMarkdown += `## 通用信息\n\n`;
        indexMarkdown += `### 认证\n\n`;
        indexMarkdown += `所有 API 接口都需要适当的身份验证。请参考各个服务的具体认证要求。\n\n`;
        indexMarkdown += `### 错误处理\n\n`;
        indexMarkdown += `API 使用标准 HTTP 状态码来表示请求的成功或失败状态。\n\n`;
        indexMarkdown += `| 状态码 | 说明 |\n`;
        indexMarkdown += `|--------|------|\n`;
        indexMarkdown += `| 200 | 请求成功 |\n`;
        indexMarkdown += `| 400 | 请求错误 |\n`;
        indexMarkdown += `| 401 | 未授权 |\n`;
        indexMarkdown += `| 403 | 禁止访问 |\n`;
        indexMarkdown += `| 404 | 资源不存在 |\n`;
        indexMarkdown += `| 500 | 服务器内部错误 |\n\n`;
      } else {
        indexMarkdown += `## General Information\n\n`;
        indexMarkdown += `### Authentication\n\n`;
        indexMarkdown += `All API endpoints require appropriate authentication. Please refer to the specific authentication requirements for each service.\n\n`;
        indexMarkdown += `### Error Handling\n\n`;
        indexMarkdown += `The API uses standard HTTP status codes to indicate the success or failure of requests.\n\n`;
        indexMarkdown += `| Status Code | Description |\n`;
        indexMarkdown += `|-------------|-------------|\n`;
        indexMarkdown += `| 200 | Success |\n`;
        indexMarkdown += `| 400 | Bad Request |\n`;
        indexMarkdown += `| 401 | Unauthorized |\n`;
        indexMarkdown += `| 403 | Forbidden |\n`;
        indexMarkdown += `| 404 | Not Found |\n`;
        indexMarkdown += `| 500 | Internal Server Error |\n\n`;
      }

      const indexPath = path.join(outputDir, 'index.md');
      fs.writeFileSync(indexPath, indexMarkdown, 'utf8');
      console.log(`  ✓ Generated ${lang}/index.md`);
    }
  }

  /**
   * Ensure directory exists
   */
  ensureDirectoryExists(dirPath) {
    if (!fs.existsSync(dirPath)) {
      fs.mkdirSync(dirPath, { recursive: true });
    }
  }
}

// Script execution
if (require.main === module) {
  const generator = new APIDocumentationGenerator();
  generator.generate().catch(console.error);
}

module.exports = APIDocumentationGenerator;